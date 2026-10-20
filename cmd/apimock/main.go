// Command apimock is a single-binary API request workspace: save requests,
// run them from a web UI, and republish any of them as a mock endpoint.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	apimock "apimock"
	"apimock/internal/server"
	"apimock/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr    = flag.String("addr", envOr("APIMOCK_ADDR", ":5178"), "listen address, e.g. :5178 or 127.0.0.1:8080")
		dbPath  = flag.String("db", envOr("APIMOCK_DB", defaultDBPath()), "path to the SQLite database file")
		static  = flag.String("static", envOr("APIMOCK_STATIC", ""), "serve the web UI from this directory instead of the embedded build")
		verbose = flag.Bool("verbose", false, "log every HTTP request")
		showVer = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("apimock", server.Version)
		return nil
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	st, err := store.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	var assets fs.FS
	if sub, err := fs.Sub(apimock.WebAssets, "web/dist"); err == nil {
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			assets = apimock.WebAssets
		}
	}

	handler := server.New(st, log, server.Config{
		Addr:      *addr,
		StaticDir: *static,
		Dev:       os.Getenv("APIMOCK_DEV") == "1",
	}, assets)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := listenAndServe(ctx, srv, log); err != nil {
		return err
	}
	return nil
}

func listenAndServe(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}

	url := displayURL(ln.Addr().String())
	log.Info("ApiMock started", "url", url, "version", server.Version)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func displayURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// defaultDBPath keeps user data next to the binary when that folder is
// writable, otherwise under the OS config directory. Binaries launched from a
// temporary location (go run, a downloaded zip) skip the first option so data
// is not scattered into throwaway folders.
func defaultDBPath() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if !isTempDir(dir) {
			candidate := filepath.Join(dir, "data", "apimock.db")
			if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err == nil {
				if file, err := os.OpenFile(candidate, os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
					file.Close()
					return candidate
				}
			}
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "apimock.db"
	}
	return filepath.Join(cfg, "apimock", "apimock.db")
}

// isTempDir reports whether dir sits inside the OS temp directory, including
// the go-build folder used by `go run`.
func isTempDir(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for _, scratch := range []string{os.TempDir()} {
		if scratch == "" {
			continue
		}
		absScratch, err := filepath.Abs(scratch)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absScratch, absDir)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
