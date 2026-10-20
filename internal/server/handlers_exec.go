package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"apimock/internal/runner"
	"apimock/internal/store"
)

type execOptions struct {
	TimeoutSeconds  int  `json:"timeoutSeconds"`
	FollowRedirects bool `json:"followRedirects"`
	VerifyTLS       bool `json:"verifyTls"`
}

type execRequest struct {
	Request       store.Request     `json:"request"`
	EnvironmentID string            `json:"environmentId"`
	Variables     map[string]string `json:"variables"`
	Options       *execOptions      `json:"options"`
}

func (a *api) resolveVars(environmentID string, extra map[string]string) (map[string]string, string) {
	vars := map[string]string{}
	envName := ""

	if environmentID == "" {
		environmentID = a.store.GetSetting("activeEnvironmentId", "")
	}
	if environmentID != "" {
		if env, err := a.store.GetEnvironment(environmentID); err == nil {
			envName = env.Name
			for _, v := range env.Variables {
				if v.Enabled && v.Key != "" {
					vars[v.Key] = v.Value
				}
			}
		}
	}
	for k, v := range extra {
		vars[k] = v
	}
	return vars, envName
}

// handleExecute runs a request definition (saved or draft) and records history.
func (a *api) handleExecute(w http.ResponseWriter, r *http.Request) {
	var body execRequest
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Request.URL) == "" {
		writeErr(w, http.StatusBadRequest, "request URL is empty")
		return
	}

	opts := runner.DefaultOpts()
	if body.Options != nil {
		if body.Options.TimeoutSeconds > 0 && body.Options.TimeoutSeconds <= 600 {
			opts.Timeout = time.Duration(body.Options.TimeoutSeconds) * time.Second
		}
		opts.FollowRedirects = body.Options.FollowRedirects
		opts.VerifyTLS = body.Options.VerifyTLS
	}

	vars, envName := a.resolveVars(body.EnvironmentID, body.Variables)

	started := time.Now()
	res, runErr := a.runner.Run(r.Context(), body.Request, vars, opts)
	elapsed := time.Since(started)

	entry := store.HistoryEntry{
		RequestID:       body.Request.ID,
		Name:            body.Request.Name,
		Method:          strings.ToUpper(body.Request.Method),
		URL:             body.Request.URL,
		DurationMs:      elapsed.Milliseconds(),
		OK:              runErr == nil,
		RequestSnapshot: store.Execution{Request: body.Request, Vars: vars, EnvName: envName}.MarshalSnapshot(),
	}
	if entry.Method == "" {
		entry.Method = "GET"
	}

	type execResponse struct {
		runner.Response
		HistoryID   string `json:"historyId"`
		EnvName     string `json:"envName"`
		Error       string `json:"error,omitempty"`
		ResolvedURL string `json:"resolvedUrl"`
	}

	if runErr != nil {
		entry.Error = runErr.Error()
		entry.Status = 0
		entry.BodyEncoding = "text"
		saved, err := a.store.InsertHistory(entry)
		if err != nil {
			a.log.Warn("could not record history", "error", err)
		}
		writeJSON(w, http.StatusOK, execResponse{
			Response:    runner.Response{ReceivedAt: time.Now()},
			HistoryID:   saved.ID,
			EnvName:     envName,
			Error:       runErr.Error(),
			ResolvedURL: body.Request.URL,
		})
		return
	}

	resolvedURL, _ := runner.BuildURL(body.Request.URL, body.Request.Params, vars)
	entry.Status = res.Status
	entry.URL = resolvedURL
	entry.SizeBytes = res.SizeBytes
	entry.RequestSnapshot = store.Execution{Request: body.Request, Vars: vars, EnvName: envName}.MarshalSnapshot()
	headers, _ := json.Marshal(res.Headers)
	entry.ResponseHeaders = string(headers)
	entry.ResponseBody = res.BodyText
	entry.BodyEncoding = res.Encoding

	saved, err := a.store.InsertHistory(entry)
	if err != nil {
		a.log.Warn("could not record history", "error", err)
	}

	_ = a.store.PruneHistory(2000)

	writeJSON(w, http.StatusOK, execResponse{
		Response:    res,
		HistoryID:   saved.ID,
		EnvName:     envName,
		ResolvedURL: resolvedURL,
	})
}

// handleCurl previews the equivalent curl command.
func (a *api) handleCurl(w http.ResponseWriter, r *http.Request) {
	var body execRequest
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	vars, _ := a.resolveVars(body.EnvironmentID, body.Variables)
	writeJSON(w, http.StatusOK, map[string]string{"curl": runner.EncodeCurl(body.Request, vars)})
}

// handleMock serves an exposed request at /mock/{slug}.
func (a *api) handleMock(w http.ResponseWriter, r *http.Request) {
	slug := trimSlug(chi.URLParam(r, "slug"))
	if slug == "" {
		writeErr(w, http.StatusNotFound, "unknown mock")
		return
	}
	saved, err := a.store.GetRequestBySlug(slug)
	if err != nil {
		writeErr(w, http.StatusNotFound, "unknown mock")
		return
	}

	opts := runner.DefaultOpts()
	opts.Timeout = 60 * time.Second

	vars, _ := a.resolveVars("", map[string]string{})
	// Query strings on the mock URL are treated as variable overrides.
	for key := range r.URL.Query() {
		vars[key] = r.URL.Query().Get(key)
	}

	// The caller's method wins when it is not a plain GET, so a saved POST can
	// be exercised from a browser address bar too.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		saved.Method = r.Method
	}

	res, runErr := a.runner.Run(r.Context(), saved, vars, opts)
	if runErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": runErr.Error()})
		return
	}

	contentType := res.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-ApiMock-History", "off")
	w.WriteHeader(res.Status)
	_, _ = w.Write(res.Body)
}
