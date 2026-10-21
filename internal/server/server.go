package server

import (
	"reqbox/internal/logging"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"reqbox/internal/runner"
	"reqbox/internal/store"
)

// api bundles the dependencies shared by all handlers.
type api struct {
	store  *store.Store
	runner *runner.Executor
	log    *logging.Logger
	auth   Auth
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// fail maps storage errors onto HTTP status codes.
func (a *api) fail(w http.ResponseWriter, err error) {
	var slugErr *slugError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	case errors.As(err, &slugErr):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		a.log.Error("request failed", "error", err)
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeBody(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 8<<20))
	return dec.Decode(dst)
}

func (a *api) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": Version,
	})
}

func (a *api) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.store.Stats()
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func trimSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteRune(c)
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
