// Command reqbox is a single-binary API request workspace: save requests,
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

	reqbox "reqbox"
	"reqbox/internal/server"
	"reqbox/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		// Loopback by default: this is a single-user tool, and binding every
		// interface exposes the workspace to the whole network by accident.
		// Reaching it from another machine is a deliberate choice, so it needs
		// an explicit -addr.
		addr    = flag.String("addr", envOr("REQBOX_ADDR", "127.0.0.1:5178"), "listen address; use :5178 or 0.0.0.0:5178 to accept remote connections")
		dbPath  = flag.String("db", envOr("REQBOX_DB", defaultDBPath()), "path to the SQLite database file")
		static  = flag.String("static", envOr("REQBOX_STATIC", ""), "serve the web UI from this directory instead of the embedded build")
		token   = flag.String("token", envOr("REQBOX_TOKEN", ""), "API token; generated and printed once if empty")
		noAuth  = flag.Bool("no-auth", false, "disable the API token check (anyone who can reach the port gets full access)")
		verbose = flag.Bool("verbose", false, "log every HTTP request")
		showVer = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("reqbox", server.Version)
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
	// The database holds plaintext credentials. If the operating system refused
	// to restrict it, say so: running is better than refusing to start, but the
	// operator should know another local account can read it.
	if st.PermissionWarning != nil {
		log.Warn("could not restrict database permissions; other local accounts may read stored credentials",
			"error", st.PermissionWarning, "path", *dbPath)
	}

	var assets fs.FS
	if sub, err := fs.Sub(reqbox.WebAssets, "web/dist"); err == nil {
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			assets = reqbox.WebAssets
		}
	}

	// Auth: a token guards the JSON API so that reaching the port is not the
	// same thing as owning the workspace. -no-auth exists for throwaway local
	// use, and says loudly what it costs.
	auth := server.Auth{}
	if !*noAuth {
		auth, err = server.EnsureStoredToken(st, *token)
		if err != nil {
			return fmt.Errorf("set up auth: %w", err)
		}
	}

	// Say which database is in use. After a rename this is the difference
	// between "it found my data" and "it quietly started empty", and there is
	// otherwise no way to tell from the outside.
	log.Info("using database", "path", *dbPath)

	handler := server.New(st, log, server.Config{
		Addr:      *addr,
		StaticDir: *static,
		Dev:       envOr("REQBOX_DEV", "") == "1",
		Auth:      auth,
	}, assets)

	// Print the token before serving, so the operator has it the moment the
	// banner appears rather than having to find it in later output.
	auth.LogToken(log, *addr)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Say so when the address accepts remote connections. Binding widely is
	// sometimes intended (a shared dev box), but it is worth one line in the
	// log because it is no longer the default.
	if !listensOnLoopbackOnly(*addr) {
		log.Warn("listening on a non-loopback address; anyone who can reach this port can reach the API",
			"addr", *addr)
	}

	if err := listenAndServe(ctx, srv, log); err != nil {
		return err
	}
	return nil
}

// listensOnLoopbackOnly reports whether addr binds just the loopback interface.
// An empty host or a wildcard is treated as "everything", which is the case
// worth warning about.
func listensOnLoopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// A bare host or port with no colon: treat as unknown, so warn.
		return false
	}
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	// "localhost" is loopback by definition; anything else is a hostname that
	// may resolve beyond it.
	return strings.EqualFold(host, "localhost")
}

func listenAndServe(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}

	url := displayURL(ln.Addr().String())
	log.Info("Reqbox started", "url", url, "version", server.Version)

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
			candidate := filepath.Join(dir, "data", "reqbox.db")
			if existing := legacyDBPath(candidate); existing != "" {
				return existing
			}
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
		return "reqbox.db"
	}
	// The config directory moved too (apimock/ -> reqbox/), so the old database
	// lives in a sibling directory rather than beside the new path.
	if existing := legacyConfigDBPath(cfg); existing != "" {
		return existing
	}
	candidate := filepath.Join(cfg, "reqbox", "reqbox.db")
	if existing := legacyDBPath(candidate); existing != "" {
		return existing
	}
	return candidate
}

// legacyConfigDBPath returns the pre-rename database under the config directory.
//
// This is a separate case from legacyDBPath because the rename changed the
// directory name as well as the file name: the old data sits in <cfg>/apimock/
// while the new code looks in <cfg>/reqbox/. Checking only the sibling file
// would miss it and an upgrading user would start on an empty workspace.
func legacyConfigDBPath(cfg string) string {
	newDir := filepath.Join(cfg, "reqbox", "reqbox.db")
	if _, err := os.Stat(newDir); err == nil {
		return "" // already migrated or freshly created; it wins
	}
	old := filepath.Join(cfg, "apimock", "apimock.db")
	if _, err := os.Stat(old); err == nil {
		return old
	}
	return ""
}

// legacyDBPath returns the pre-rename database beside candidate, if one exists
// and the new location has nothing yet.
//
// The rename changed the default file name from apimock.db to reqbox.db. Without
// this, anyone upgrading would start on an empty database and reasonably
// conclude their workspace was lost. Reading the old file in place is safer
// than silently copying it, because a copy would fork the data if the user then
// ran both binaries.
func legacyDBPath(candidate string) string {
	if _, err := os.Stat(candidate); err == nil {
		return "" // the new file already exists; it wins
	}
	old := filepath.Join(filepath.Dir(candidate), "apimock.db")
	if _, err := os.Stat(old); err == nil {
		return old
	}
	return ""
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
	// Read the pre-rename variable too, so an existing deployment whose start
	// script still says APIMOCK_* keeps working. The new name wins when both
	// are set, which is what anyone migrating one variable at a time expects.
	if legacy, ok := legacyEnvAlias[key]; ok {
		if v := os.Getenv(legacy); v != "" {
			return v
		}
	}
	return fallback
}

// legacyEnvAlias maps current variables to the names used before the rename.
// Remove this once no deployment can plausibly still be using the old ones.
var legacyEnvAlias = map[string]string{
	"REQBOX_ADDR":   "APIMOCK_ADDR",
	"REQBOX_DB":     "APIMOCK_DB",
	"REQBOX_STATIC": "APIMOCK_STATIC",
	"REQBOX_TOKEN":  "APIMOCK_TOKEN",
}
