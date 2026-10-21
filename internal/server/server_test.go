package server

import (
	"reqbox/internal/logging"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"reqbox/internal/store"
)

func newTestServer(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	log := logging.Discard()
	return New(st, log, Config{}, nil), st
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestHealth(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "GET", "/api/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode[map[string]string](t, rec)
	if body["status"] != "ok" {
		t.Errorf("health = %+v", body)
	}
}

func TestCollectionAndRequestCRUD(t *testing.T) {
	h, _ := newTestServer(t)

	rec := do(t, h, "POST", "/api/collections", map[string]string{"name": "Users"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create collection: %d %s", rec.Code, rec.Body)
	}
	col := decode[store.Collection](t, rec)

	rec = do(t, h, "POST", "/api/requests", map[string]any{
		"name": "List users", "method": "GET", "url": "https://api.test/users",
		"collectionId": col.ID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create request: %d %s", rec.Code, rec.Body)
	}
	created := decode[store.Request](t, rec)
	if created.CollectionID == nil || *created.CollectionID != col.ID {
		t.Errorf("collection link = %v", created.CollectionID)
	}

	rec = do(t, h, "PUT", "/api/requests/"+created.ID, map[string]any{
		"id": created.ID, "name": "List all users", "method": "GET", "url": "https://api.test/users",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := decode[store.Request](t, rec); got.Name != "List all users" {
		t.Errorf("name = %q", got.Name)
	}

	rec = do(t, h, "POST", "/api/requests/"+created.ID+"/duplicate", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "DELETE", "/api/requests/"+created.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}

	rec = do(t, h, "DELETE", "/api/requests/"+created.ID, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", rec.Code)
	}
}

func TestExecuteAndHistory(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "7" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"name":"Ada"}`))
	}))
	defer upstream.Close()

	h, _ := newTestServer(t)

	rec := do(t, h, "POST", "/api/execute", map[string]any{
		"request": map[string]any{
			"name": "Get user", "method": "GET", "url": upstream.URL + "/users/{{id}}",
			"params": []map[string]any{{"id": "p1", "key": "id", "value": "7", "enabled": true}},
		},
		"variables": map[string]string{"id": "7"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("execute: %d %s", rec.Code, rec.Body)
	}
	res := decode[map[string]any](t, rec)
	if res["status"].(float64) != 200 {
		t.Errorf("status = %v", res["status"])
	}
	if res["body"] != `{"id":7,"name":"Ada"}` {
		t.Errorf("body = %v", res["body"])
	}
	if res["historyId"] == "" {
		t.Error("expected a historyId")
	}

	rec = do(t, h, "GET", "/api/history", nil)
	entries := decode[[]store.HistoryEntry](t, rec)
	if len(entries) != 1 {
		t.Fatalf("history = %d entries", len(entries))
	}
	if entries[0].Status != 200 || entries[0].Method != "GET" {
		t.Errorf("entry = %+v", entries[0])
	}

	// Replay rebuilds the same call from the stored snapshot.
	rec = do(t, h, "POST", "/api/history/"+entries[0].ID+"/replay", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("replay: %d %s", rec.Code, rec.Body)
	}
}

func TestExecuteRejectsEmptyURL(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/execute", map[string]any{"request": map[string]any{"url": ""}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// Omitting the security flags must not turn protection off. A client that sends
// only a timeout previously got TLS verification disabled, because a missing
// bool decoded to false. That decoding lives in the server, so this asserts the
// end-to-end behaviour against a self-signed endpoint.
func TestExecuteOmittedFlagsKeepTLSVerification(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secret"))
	}))
	defer upstream.Close()

	h, _ := newTestServer(t)

	// Only a timeout: verifyTls is absent and must default to true, so the
	// self-signed certificate has to be rejected.
	rec := do(t, h, "POST", "/api/execute", map[string]any{
		"request": map[string]any{"url": upstream.URL, "method": "GET"},
		"options": map[string]any{"timeoutSeconds": 5},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode[map[string]any](t, rec)
	errText, _ := body["error"].(string)
	if errText == "" {
		t.Fatal("expected TLS verification to fail with omitted verifyTls")
	}
	if !strings.Contains(errText, "certificate") {
		t.Errorf("expected a certificate error, got %q", errText)
	}

	// The same call with verification explicitly off must succeed.
	rec = do(t, h, "POST", "/api/execute", map[string]any{
		"request": map[string]any{"url": upstream.URL, "method": "GET"},
		"options": map[string]any{"timeoutSeconds": 5, "verifyTls": false},
	})
	body = decode[map[string]any](t, rec)
	if body["error"] != nil {
		t.Errorf("explicit verifyTls=false should succeed, got %v", body["error"])
	}
	if body["body"] != "secret" {
		t.Errorf("body = %v", body["body"])
	}
}

// A brand-new draft from the UI has empty timestamps and empty ids. Those must
// not be rejected by timestamp parsing.
func TestExecuteAcceptsBlankDraft(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/execute", map[string]any{
		"request": map[string]any{
			"id": "", "collectionId": nil, "name": "Draft", "method": "POST",
			"url": upstream.URL, "params": []any{}, "headers": []any{},
			"bodyType": "json", "body": `{"a":1}`, "form": []any{},
			"authType": "none", "auth": map[string]any{}, "sortIndex": 0,
			"exposeSlug": "", "exposed": false, "createdAt": "", "updatedAt": "",
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("blank draft was rejected: %d %s", rec.Code, rec.Body)
	}
	if got := decode[map[string]any](t, rec); got["status"].(float64) != 200 {
		t.Errorf("status = %v", got["status"])
	}
}

// Request create/update must also tolerate empty timestamps from a draft.
func TestCreateRequestAcceptsEmptyTimestamps(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/requests", map[string]any{
		"name": "Draft", "method": "GET", "url": "https://x.test",
		"createdAt": "", "updatedAt": "",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with blank timestamps = %d %s", rec.Code, rec.Body)
	}
	created := decode[store.Request](t, rec)
	if created.CreatedAt.IsZero() {
		t.Error("server should stamp CreatedAt itself")
	}
}

func TestExecuteFailureIsRecordedNotFatal(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/execute", map[string]any{
		"request": map[string]any{"name": "bad", "method": "GET", "url": "http://127.0.0.1:1/nope"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with an error field", rec.Code)
	}
	body := decode[map[string]any](t, rec)
	if body["error"] == "" || body["error"] == nil {
		t.Errorf("expected an error message, got %+v", body)
	}

	rec = do(t, h, "GET", "/api/history", nil)
	entries := decode[[]store.HistoryEntry](t, rec)
	if len(entries) != 1 || entries[0].OK {
		t.Errorf("failed run should be stored as not-ok: %+v", entries)
	}
}

func TestMockEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("from upstream: " + r.URL.Query().Get("who")))
	}))
	defer upstream.Close()

	h, st := newTestServer(t)
	created, err := st.CreateRequest(store.Request{
		Name: "Mocked", Method: "GET", URL: upstream.URL + "/x?who={{who}}",
		ExposeSlug: "hello", Exposed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ExposeSlug != "hello" {
		t.Fatalf("slug = %q", created.ExposeSlug)
	}

	rec := do(t, h, "GET", "/mock/hello?who=amy", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("mock: %d %s", rec.Code, rec.Body)
	}
	if rec.Body.String() != "from upstream: amy" {
		t.Errorf("body = %q", rec.Body.String())
	}

	rec = do(t, h, "GET", "/mock/unknown", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown mock = %d, want 404", rec.Code)
	}
}

func TestMockIsHiddenWhenUnpublished(t *testing.T) {
	h, st := newTestServer(t)
	_, _ = st.CreateRequest(store.Request{
		Name: "Draft", Method: "GET", URL: "http://127.0.0.1:1/", ExposeSlug: "draft", Exposed: false,
	})
	rec := do(t, h, "GET", "/mock/draft", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unpublished mock = %d, want 404", rec.Code)
	}
}

// A caller must not be able to redirect a published mock to another host, or
// they would receive the owner's stored credentials.
func TestMockDropsCredentialsWhenHostChanges(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	h, st := newTestServer(t)
	_, err := st.CreateRequest(store.Request{
		Name: "Leaky", Method: "GET", URL: "{{baseUrl}}/thing",
		AuthType: "bearer", Auth: store.Auth{Token: "SECRET"},
		ExposeSlug: "leaky", Exposed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The caller points baseUrl somewhere the owner never intended.
	rec := do(t, h, "GET", "/mock/leaky?baseUrl="+upstream.URL, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("mock = %d %s", rec.Code, rec.Body)
	}
	if gotAuth != "" {
		t.Errorf("credentials leaked to a caller-chosen host: %q", gotAuth)
	}
}

func TestSlugCollision(t *testing.T) {
	h, _ := newTestServer(t)
	body := map[string]any{
		"name": "A", "method": "GET", "url": "https://x.test",
		"exposeSlug": "same", "exposed": true,
	}
	if rec := do(t, h, "POST", "/api/requests", body); rec.Code != http.StatusCreated {
		t.Fatalf("first create: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/api/requests", body); rec.Code != http.StatusConflict {
		t.Errorf("duplicate explicit slug = %d, want 409", rec.Code)
	}
}

// Publishing without a slug must derive one, otherwise the mock is unreachable.
func TestPublishDerivesSlug(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/requests", map[string]any{
		"name": "Daily exchange rates", "method": "GET",
		"url": "https://x.test/rates", "exposed": true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	created := decode[store.Request](t, rec)
	if created.ExposeSlug != "daily-exchange-rates" {
		t.Errorf("derived slug = %q", created.ExposeSlug)
	}

	// A second request with the same name gets a distinct, still-working slug.
	rec = do(t, h, "POST", "/api/requests", map[string]any{
		"name": "Daily exchange rates", "method": "GET",
		"url": "https://x.test/rates2", "exposed": true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("second create: %d %s", rec.Code, rec.Body)
	}
	second := decode[store.Request](t, rec)
	if second.ExposeSlug != "daily-exchange-rates-2" {
		t.Errorf("second derived slug = %q", second.ExposeSlug)
	}
}

func TestUnpublishedRequestCanKeepSlug(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/requests", map[string]any{
		"name": "Ping", "method": "GET", "url": "https://x.test",
		"exposeSlug": "ping", "exposed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if got := decode[store.Request](t, rec); got.ExposeSlug != "ping" || got.Exposed {
		t.Errorf("unpublished request should keep its slug: %+v", got)
	}
}

func TestEnvironmentActivation(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/environments", map[string]any{
		"name": "Local",
		"variables": []map[string]any{
			{"key": "baseUrl", "value": "http://localhost:9", "enabled": true},
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create env: %d %s", rec.Code, rec.Body)
	}
	env := decode[store.Environment](t, rec)

	rec = do(t, h, "PUT", "/api/environments/active", map[string]string{"id": env.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "GET", "/api/environments", nil)
	list := decode[struct {
		Items  []store.Environment `json:"items"`
		Active string              `json:"active"`
	}](t, rec)
	if list.Active != env.ID {
		t.Errorf("active = %q, want %q", list.Active, env.ID)
	}

	// Deleting the active environment clears the setting.
	if rec := do(t, h, "DELETE", "/api/environments/"+env.ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	rec = do(t, h, "GET", "/api/environments", nil)
	list = decode[struct {
		Items  []store.Environment `json:"items"`
		Active string              `json:"active"`
	}](t, rec)
	if list.Active != "" {
		t.Errorf("active should be cleared, got %q", list.Active)
	}
}

func TestExportImport(t *testing.T) {
	h, st := newTestServer(t)
	col, _ := st.CreateCollection("Transfer")
	_, _ = st.CreateRequest(store.Request{CollectionID: &col.ID, Name: "Ping", Method: "GET", URL: "https://x.test"})

	rec := do(t, h, "GET", "/api/export", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("export: %d", rec.Code)
	}
	var snap store.Export
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}

	// Import into a second server and check the data survived.
	other, _ := newTestServer(t)
	rec = do(t, other, "POST", "/api/import", snap)
	if rec.Code != http.StatusOK {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	rec = do(t, other, "GET", "/api/requests", nil)
	reqs := decode[[]store.Request](t, rec)
	if len(reqs) != 1 || reqs[0].Name != "Ping" {
		t.Errorf("imported = %+v", reqs)
	}
}

func TestCurlEndpoint(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/curl", map[string]any{
		"request": map[string]any{
			"method": "POST", "url": "https://api.test/v1",
			"bodyType": "json", "body": `{"a":1}`,
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("curl: %d %s", rec.Code, rec.Body)
	}
	out := decode[map[string]string](t, rec)
	if out["curl"] == "" {
		t.Error("expected a curl command")
	}
}

func TestCrossOriginWritesAreRejected(t *testing.T) {
	h, _ := newTestServer(t)

	req := httptest.NewRequest("POST", "/api/requests", strings.NewReader(`{"name":"x","url":"http://x.test"}`))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-origin write = %d, want 403", rec.Code)
	}

	// A same-origin write still works.
	req = httptest.NewRequest("POST", "/api/requests", strings.NewReader(`{"name":"ok","url":"http://x.test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://"+req.Host)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("same-origin write = %d, want 201: %s", rec.Code, rec.Body)
	}

	// No Origin at all (curl, tests) is allowed: this is a local tool.
	req = httptest.NewRequest("POST", "/api/requests", strings.NewReader(`{"name":"cli","url":"http://x.test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("origin-less write = %d, want 201", rec.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, "GET", "/requests/anything", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("SPA fallback = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content type = %q", ct)
	}
}
