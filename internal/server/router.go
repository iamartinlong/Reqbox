package server

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"apimock/internal/runner"
	"apimock/internal/store"
)

// Version is stamped at build time via -ldflags.
var Version = "dev"

// Config controls how the HTTP server behaves.
type Config struct {
	Addr      string
	StaticDir string // optional: serve a directory instead of the embedded build
	Dev       bool   // when true, serve a placeholder page and skip static assets
}

// New builds the router, wiring JSON API routes, the mock gateway and the SPA.
func New(store *store.Store, log *slog.Logger, cfg Config, assets fs.FS) http.Handler {
	a := &api{store: store, runner: runner.New(), log: log}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/json", "text/html", "text/css", "application/javascript", "image/svg+xml"))
	r.Use(noCacheAPI)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", a.handleHealth)
		r.Get("/stats", a.handleStats)

		r.Get("/collections", a.handleListCollections)
		r.Post("/collections", a.handleCreateCollection)
		r.Put("/collections/{id}", a.handleUpdateCollection)
		r.Delete("/collections/{id}", a.handleDeleteCollection)

		r.Get("/requests", a.handleListRequests)
		r.Post("/requests", a.handleCreateRequest)
		r.Put("/requests/reorder", a.handleReorderRequests)
		r.Put("/requests/{id}", a.handleUpdateRequest)
		r.Delete("/requests/{id}", a.handleDeleteRequest)
		r.Post("/requests/{id}/duplicate", a.handleDuplicateRequest)

		r.Post("/execute", a.handleExecute)
		r.Post("/curl", a.handleCurl)

		r.Get("/history", a.handleListHistory)
		r.Delete("/history", a.handleClearHistory)
		r.Get("/history/{id}", a.handleGetHistory)
		r.Delete("/history/{id}", a.handleDeleteHistory)
		r.Post("/history/{id}/replay", a.handleReplayHistory)
		r.Get("/history/{id}/body", a.handleDownloadBody)

		r.Get("/environments", a.handleListEnvironments)
		r.Post("/environments", a.handleCreateEnvironment)
		r.Put("/environments/active", a.handleActivateEnvironment)
		r.Put("/environments/{id}", a.handleUpdateEnvironment)
		r.Delete("/environments/{id}", a.handleDeleteEnvironment)

		r.Get("/export", a.handleExport)
		r.Post("/import", a.handleImport)
	})

	// Public mock gateway: a saved request can be replayed by slug.
	r.HandleFunc("/mock/{slug}", a.handleMock)
	r.HandleFunc("/mock/{slug}/*", a.handleMock)

	// Static SPA. Embedded assets win unless an explicit directory is set.
	var staticFS fs.FS
	if cfg.StaticDir != "" {
		staticFS = os.DirFS(cfg.StaticDir)
	} else if assets != nil {
		staticFS, _ = fs.Sub(assets, "web/dist")
	}

	if staticFS != nil {
		r.Handle("/*", spaHandler(staticFS, cfg.Dev))
	} else {
		r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(devPlaceholder))
		})
	}

	return r
}

// noCacheAPI keeps the JSON API honest during development.
func noCacheAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves static files and falls back to index.html for client routes.
func spaHandler(staticFS fs.FS, dev bool) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "" || clean == "." {
			clean = "index.html"
		}
		if f, err := staticFS.Open(clean); err == nil {
			defer f.Close()
			if info, statErr := f.Stat(); statErr == nil && !info.IsDir() {
				if strings.HasPrefix(clean, "assets/") {
					// Vite emits hashed filenames, safe to cache hard.
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else if !dev {
					w.Header().Set("Cache-Control", "no-cache")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			http.Error(w, "web build not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

const devPlaceholder = `<!doctype html>
<html><head><meta charset="utf-8"><title>ApiMock</title></head>
<body style="font:14px/1.6 ui-monospace,monospace;padding:3rem;background:#0b0c0e;color:#e6e6e6">
<h1>ApiMock API is running</h1>
<p>The web build is not embedded in this binary.</p>
<p>Run <code>pnpm dev</code> inside <code>web/</code>, or rebuild with <code>go build -tags web</code>.</p>
</body></html>`
