// Command reqbox is a single-binary API request workspace: save requests,
// run them from a web UI, and republish any of them as a mock endpoint.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	reqbox "reqbox"
	"reqbox/internal/config"
	"reqbox/internal/logging"
	"reqbox/internal/server"
	"reqbox/internal/store"
)

func main() {
	// Before anything is printed: with QuickEdit on, the first drag-select would
	// freeze the window the operator is trying to read.
	allowConsoleCopy()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// The config file is read before the flags are declared so its values can
	// become the flag defaults. That keeps the precedence in one place: a flag
	// beats the file because the flag's default *is* the file's value, and
	// anything the operator types overrides it.
	cfg, cfgPath, cfgErr := config.Load()
	if cfgErr != nil {
		// A broken config must not be silently ignored: the operator would edit
		// a file that appears to have no effect. Say what is wrong and stop.
		return cfgErr
	}

	addrDefault := envOr("REQBOX_ADDR", cfg.Addr(config.DefaultAddr))
	noAuthDefault := !cfg.AuthEnabled()

	var (
		// Loopback by default: this is a single-user tool, and binding every
		// interface exposes the workspace to the whole network by accident.
		addr      = flag.String("addr", addrDefault, "listen address; use :5178 or 0.0.0.0:5178 to accept remote connections")
		dbPath    = flag.String("db", envOr("REQBOX_DB", dbOrDefault(cfg)), "path to the SQLite database file")
		static    = flag.String("static", envOr("REQBOX_STATIC", ""), "serve the web UI from this directory instead of the embedded build")
		token     = flag.String("token", envOr("REQBOX_TOKEN", cfg.Auth.Token), "API token; generated and printed once if empty")
		noAuth    = flag.Bool("no-auth", noAuthDefault, "disable the API token check (anyone who can reach the port gets full access)")
		verbose   = flag.Bool("verbose", cfg.Logging.Verbose, "log every HTTP request")
		noBrowser = flag.Bool("no-browser", false, "do not open a browser at startup")
		showVer   = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("reqbox", server.Version)
		return nil
	}

	log := logging.New(*verbose)

	st, err := store.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()
	// The database holds plaintext credentials. If the OS refused to restrict
	// it, say so: running is better than refusing to start, but the operator
	// should know another local account can read it.
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
	// same thing as owning the workspace.
	auth := server.Auth{}
	if !*noAuth {
		auth, err = server.EnsureStoredToken(st, *token)
		if err != nil {
			return fmt.Errorf("set up auth: %w", err)
		}
	}

	// Say which database is in use. After a rename this is the difference
	// between "it found my data" and "it quietly started empty".
	log.Info("using database", "path", *dbPath)
	// Same reasoning for the config file: "I read it" versus "I did not find it"
	// is the only way to explain why an edit had no effect.
	if cfgErr == nil {
		if _, statErr := os.Stat(cfgPath); statErr == nil {
			log.Info("using config file", "path", cfgPath)
		} else {
			log.Info("no config file; using defaults", "looked in", cfgPath)
		}
	}

	listener, actualAddr, err := listenWithFallback(*addr, cfg.AutoPortEnabled())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}
	if actualAddr != *addr {
		// Never move quietly. The URL below is printed with the real port, but
		// someone reading the config file needs to know it was not the one used.
		log.Warn("requested port is in use; listening on the next free one",
			"requested", *addr, "using", actualAddr)
	}

	handler := server.New(st, log, server.Config{
		Addr:      actualAddr,
		StaticDir: *static,
		Dev:       os.Getenv("REQBOX_DEV") == "1",
		Auth:      auth,
	}, assets)

	// Resolve the URL once, after the listener exists, and print the token with
	// it. Doing this before listening is why `-addr 127.0.0.1:0` used to print a
	// login link pointing at port 0.
	baseURL := displayURL(listener.Addr().String())
	auth.LogToken(log, baseURL)

	srv := &http.Server{
		Addr:              actualAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Say so when the address accepts remote connections. Binding widely is
	// sometimes intended, but it is no longer the default.
	if !listensOnLoopbackOnly(actualAddr) {
		log.Warn("listening on a non-loopback address; anyone who can reach this port can reach the API",
			"addr", actualAddr)
	}

	// Open the browser after the listener is bound, so the page cannot race the
	// server and show a connection error. The sign-in URL always works here
	// because it carries the token; without one (a restart) the bare URL still
	// lands on the unlock screen.
	if cfg.OpenBrowserEnabled() && !*noBrowser {
		target := baseURL + "/"
		if loginURL := auth.LoginURL(baseURL); loginURL != "" {
			target = loginURL
		}
		if err := openBrowser(target); err != nil {
			log.Debug("could not open a browser", "error", err, "url", target)
		} else {
			log.Info("opened the browser", "url", target)
		}
	}

	if err := serve(ctx, srv, listener, log); err != nil {
		return err
	}
	return nil
}

// dbOrDefault resolves the database path, letting the config file override the
// built-in location only when it actually sets one.
func dbOrDefault(cfg config.File) string {
	if cfg.Storage.DB != "" {
		return cfg.Storage.DB
	}
	return defaultDBPath()
}

// listensOnLoopbackOnly reports whether addr binds just the loopback interface.
// An empty host or a wildcard is treated as "everything", which is the case
// worth warning about.
func listensOnLoopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

// serve runs the HTTP server on an already-bound listener until ctx is done.
//
// The listener is bound by the caller so the port is known before anything is
// printed or opened: the sign-in URL and the browser tab must point at the real
// port, including when the configured one was busy and the server moved.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, log *logging.Logger) error {
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
	// The config directory moved too (apimock/ -> reqbox/), so old data lives in
	// a sibling directory rather than beside the new path.
	if existing := legacyConfigDBPath(cfg); existing != "" {
		return existing
	}
	candidate := filepath.Join(cfg, "reqbox", "reqbox.db")
	if existing := legacyDBPath(candidate); existing != "" {
		return existing
	}
	return candidate
}

// legacyDBPath returns the pre-rename database beside candidate, if one exists
// and the new location has nothing yet. Reading the old file in place is safer
// than copying it, which would fork the data if both binaries were then run.
func legacyDBPath(candidate string) string {
	if _, err := os.Stat(candidate); err == nil {
		return ""
	}
	old := filepath.Join(filepath.Dir(candidate), "apimock.db")
	if _, err := os.Stat(old); err == nil {
		return old
	}
	return ""
}

// legacyConfigDBPath covers the directory rename under the config dir.
func legacyConfigDBPath(cfg string) string {
	if _, err := os.Stat(filepath.Join(cfg, "reqbox", "reqbox.db")); err == nil {
		return ""
	}
	old := filepath.Join(cfg, "apimock", "apimock.db")
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
	// Honour the pre-rename variable too, so a deployment whose start script
	// still says APIMOCK_* keeps working. The current name wins when both are set.
	if legacy, ok := legacyEnvAlias[key]; ok {
		if v := os.Getenv(legacy); v != "" {
			return v
		}
	}
	return fallback
}

// legacyEnvAlias maps current variables to their pre-rename names.
var legacyEnvAlias = map[string]string{
	"REQBOX_ADDR":   "APIMOCK_ADDR",
	"REQBOX_DB":     "APIMOCK_DB",
	"REQBOX_STATIC": "APIMOCK_STATIC",
	"REQBOX_TOKEN":  "APIMOCK_TOKEN",
}
