package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"reqbox/internal/store"
)

func (a *api) handleListHistory(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListHistory(r.URL.Query().Get("q"), queryInt(r, "limit", 100))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.GetHistory(chi.URLParam(r, "id"))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *api) handleDeleteHistory(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteHistory(chi.URLParam(r, "id")); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	n, err := a.store.ClearHistory()
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

func (a *api) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListEnvironments()
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"active": a.store.GetSetting("activeEnvironmentId", ""),
	})
}

func (a *api) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string         `json:"name"`
		Variables []store.EnvVar `json:"variables"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		body.Name = "New environment"
	}
	item, err := a.store.CreateEnvironment(body.Name, body.Variables)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *api) handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string         `json:"name"`
		Variables []store.EnvVar `json:"variables"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := a.store.UpdateEnvironment(chi.URLParam(r, "id"), body.Name, body.Variables)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *api) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeleteEnvironment(id); err != nil {
		a.fail(w, err)
		return
	}
	if a.store.GetSetting("activeEnvironmentId", "") == id {
		_ = a.store.SetSetting("activeEnvironmentId", "")
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleActivateEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ID != "" {
		if _, err := a.store.GetEnvironment(body.ID); err != nil {
			a.fail(w, err)
			return
		}
	}
	if err := a.store.SetSetting("activeEnvironmentId", body.ID); err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"active": body.ID})
}

func (a *api) handleExport(w http.ResponseWriter, r *http.Request) {
	snap, err := a.store.Snap()
	if err != nil {
		a.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="reqbox-export.json"`)
	_ = json.NewEncoder(w).Encode(snap)
}

func (a *api) handleImport(w http.ResponseWriter, r *http.Request) {
	var snap store.Export
	if err := decodeBody(r, &snap); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if snap.Version == "" {
		writeErr(w, http.StatusBadRequest, "not an Reqbox export file")
		return
	}
	if err := a.store.Restore(snap); err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"collections":  len(snap.Collections),
		"requests":     len(snap.Requests),
		"environments": len(snap.Environments),
	})
}

// handleReplayHistory re-runs a stored execution with its original variables.
func (a *api) handleReplayHistory(w http.ResponseWriter, r *http.Request) {
	entry, err := a.store.GetHistory(chi.URLParam(r, "id"))
	if err != nil {
		a.fail(w, err)
		return
	}
	exec, ok := store.ParseSnapshot(entry.RequestSnapshot)
	if !ok {
		writeErr(w, http.StatusBadRequest, "this history entry has no replayable request")
		return
	}
	body := execRequest{Request: exec.Request, Variables: exec.Vars}
	payload, _ := json.Marshal(body)
	r2 := r.Clone(r.Context())
	r2.Body = io.NopCloser(bytes.NewReader(payload))
	r2.ContentLength = int64(len(payload))
	a.handleExecute(w, r2)
}

// handleDownloadBody returns the raw (possibly binary) response body.
func (a *api) handleDownloadBody(w http.ResponseWriter, r *http.Request) {
	entry, err := a.store.GetHistory(chi.URLParam(r, "id"))
	if err != nil {
		a.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="response.bin"`)
	if entry.BodyEncoding == "base64" {
		raw, err := base64.StdEncoding.DecodeString(entry.ResponseBody)
		if err == nil {
			_, _ = w.Write(raw)
			return
		}
	}
	_, _ = w.Write([]byte(entry.ResponseBody))
}
