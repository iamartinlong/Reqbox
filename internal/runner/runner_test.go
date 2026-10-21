package runner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reqbox/internal/store"
)

func TestResolve(t *testing.T) {
	vars := map[string]string{"baseUrl": "https://api.test", "token": "abc"}
	cases := map[string]string{
		"{{baseUrl}}/users":         "https://api.test/users",
		"{{ baseUrl }}/users":       "https://api.test/users",
		"{{missing}}":               "{{missing}}",
		"no tokens here":            "no tokens here",
		"{{token}}-{{token}}":       "abc-abc",
		"prefix {{baseUrl}} suffix": "prefix https://api.test suffix",
		"unclosed {{baseUrl":        "unclosed {{baseUrl",
	}
	for in, want := range cases {
		if got := Resolve(in, vars); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveBuiltins(t *testing.T) {
	got := Resolve("{{$timestamp}}", nil)
	if len(got) != 10 {
		t.Errorf("$timestamp should be 10 digits, got %q", got)
	}
	uuid := Resolve("{{$uuid}}", nil)
	if len(uuid) != 36 {
		t.Errorf("$uuid should be 36 chars, got %q", uuid)
	}
}

func TestBuildURL(t *testing.T) {
	params := []store.KV{
		{Key: "limit", Value: "10", Enabled: true},
		{Key: "skip", Value: "5", Enabled: false},
		{Key: "q", Value: "{{term}}", Enabled: true},
		{Key: "", Value: "ignored", Enabled: true},
	}
	got, err := BuildURL("example.test/search", params, map[string]string{"term": "a b"})
	if err != nil {
		t.Fatal(err)
	}
	want := "http://example.test/search?limit=10&q=a+b"
	if got != want {
		t.Errorf("BuildURL = %q, want %q", got, want)
	}
}

func TestBuildURLEmpty(t *testing.T) {
	if _, err := BuildURL("", nil, nil); err == nil {
		t.Fatal("expected an error for an empty URL")
	}
}

// A param overrides a query parameter written into the URL, since the URL field
// is a draft and the Params panel is the authoritative configuration.
func TestBuildURLParamOverridesURLQuery(t *testing.T) {
	got, err := BuildURL("http://x.test/api?q=1&keep=yes", []store.KV{
		{Key: "q", Value: "2", Enabled: true},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://x.test/api?keep=yes&q=2"
	if got != want {
		t.Errorf("BuildURL = %q, want %q", got, want)
	}
}

// Repeated keys among params are a deliberate choice and must survive.
func TestBuildURLKeepsRepeatedParams(t *testing.T) {
	got, err := BuildURL("http://x.test/api", []store.KV{
		{Key: "tag", Value: "a", Enabled: true},
		{Key: "tag", Value: "b", Enabled: true},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://x.test/api?tag=a&tag=b"
	if got != want {
		t.Errorf("BuildURL = %q, want %q", got, want)
	}
}

// Disabled params must not override, and must not remove the URL's value either.
func TestBuildURLDisabledParamLeavesURLQuery(t *testing.T) {
	got, err := BuildURL("http://x.test/api?q=1", []store.KV{
		{Key: "q", Value: "2", Enabled: false},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://x.test/api?q=1"
	if got != want {
		t.Errorf("BuildURL = %q, want %q", got, want)
	}
}

func TestBuildBody(t *testing.T) {
	body, ct, err := BuildBody(store.Request{
		BodyType: "json",
		Body:     `{"id": "{{id}}"}`,
	}, map[string]string{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if ct != "application/json" {
		t.Errorf("content type = %q", ct)
	}
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, body); err != nil {
		t.Fatal(err)
	}
	if buf.String() != `{"id": "42"}` {
		t.Errorf("body = %q", buf.String())
	}
}

// The tool adds headers the user never typed (Content-Type, Accept,
// User-Agent, Authorization). They must be visible afterwards, since "what did
// I actually send" is the most common question when a call misbehaves.
func TestRunCapturesRequestHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method:   "POST",
		URL:      srv.URL + "/echo",
		BodyType: "json",
		Body:     `{"a":1}`,
		Headers: []store.KV{
			{Key: "X-Custom", Value: "mine", Enabled: true},
			{Key: "X-Off", Value: "nope", Enabled: false},
		},
		AuthType: "bearer",
		Auth:     store.Auth{Token: "tok"},
	}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "*/*",
		"User-Agent":    "Reqbox/1.0",
		"Authorization": "Bearer tok",
		"X-Custom":      "mine",
	}
	for k, v := range want {
		got := res.RequestHeaders[k]
		if len(got) == 0 || got[0] != v {
			t.Errorf("request header %s = %v, want %q", k, got, v)
		}
	}
	if _, ok := res.RequestHeaders["X-Off"]; ok {
		t.Error("disabled header was reported as sent")
	}
	if res.RequestURL == "" {
		t.Error("RequestURL should be populated")
	}
	if res.HTTPVersion == "" {
		t.Error("HTTPVersion should be populated")
	}
	if res.RemoteAddr == "" {
		t.Error("RemoteAddr should be populated for a completed call")
	}
}

// A failed call still has a fully formed request. Reporting nothing leaves the
// user unable to see which headers were sent or where the request pointed.
func TestRunCapturesRequestDetailsOnFailure(t *testing.T) {
	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method:   "GET",
		URL:      "http://127.0.0.1:1/nope",
		Headers:  []store.KV{{Key: "X-Probe", Value: "1", Enabled: true}},
		AuthType: "bearer",
		Auth:     store.Auth{Token: "tok"},
	}, nil, DefaultOpts())
	if err == nil {
		t.Fatal("expected the connection to fail")
	}
	if len(res.RequestHeaders) == 0 {
		t.Fatal("request headers should be captured even when the call fails")
	}
	if got := res.RequestHeaders["X-Probe"]; len(got) == 0 || got[0] != "1" {
		t.Errorf("X-Probe = %v", got)
	}
	if res.RequestURL == "" {
		t.Error("RequestURL should be populated even on failure")
	}
}

// The request body that was actually sent differs from the editor's copy once
// variables are substituted. It must be reported so the user can confirm what
// left the process.
func TestRunCapturesSentRequestBody(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		received = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method:   "POST",
		URL:      srv.URL + "/echo",
		BodyType: "json",
		Body:     `{"id":"{{id}}","n":1}`,
	}, map[string]string{"id": "42"}, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}

	want := `{"id":"42","n":1}`
	if res.RequestBody != want {
		t.Errorf("RequestBody = %q, want %q", res.RequestBody, want)
	}
	if res.RequestType != "json" {
		t.Errorf("RequestType = %q", res.RequestType)
	}
	// The reported body must match what the server actually received.
	if received != want {
		t.Errorf("server received %q, differering from the reported body", received)
	}
}

func TestRunReportsNoBodyForGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method: "GET", URL: srv.URL, BodyType: "none",
	}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.RequestBody != "" {
		t.Errorf("RequestBody = %q, want empty", res.RequestBody)
	}
	if res.RequestType != "none" {
		t.Errorf("RequestType = %q, want none", res.RequestType)
	}
}

// Timing must reflect where the time actually went, not just the total.
func TestRunCapturesTiming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method: "GET", URL: srv.URL,
	}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.Timing == nil {
		t.Fatal("Timing should be populated for a completed call")
	}
	// The server delays 120ms before the first byte, so TTFB must cover it.
	if res.Timing.TTFBMs < 100 {
		t.Errorf("TTFB = %dms, expected >=100 given the handler delay", res.Timing.TTFBMs)
	}
	// TTFB cannot exceed the total.
	if res.Timing.TTFBMs > res.DurationMs+5 {
		t.Errorf("TTFB %dms exceeds total %dms", res.Timing.TTFBMs, res.DurationMs)
	}
	if res.Timing.DownloadMs < 0 {
		t.Errorf("DownloadMs = %d", res.Timing.DownloadMs)
	}
}

// Download time is what the body cost after the first byte, so a slow body with
// fast headers must show up there rather than in TTFB.
func TestRunSeparatesDownloadFromTTFB(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 5; i++ {
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte("chunk\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method: "GET", URL: srv.URL,
	}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.Timing == nil {
		t.Fatal("Timing should be populated")
	}
	if res.Timing.DownloadMs < 100 {
		t.Errorf("DownloadMs = %d, expected the slow body to dominate", res.Timing.DownloadMs)
	}
}

// A failed call should still report what timing it managed to collect, since a
// timeout is fundamentally a timing question.
func TestRunReportsTimingOnFailure(t *testing.T) {
	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method: "GET", URL: "http://127.0.0.1:1/nope",
	}, nil, DefaultOpts())
	if err == nil {
		t.Fatal("expected the connection to fail")
	}
	if res.Timing == nil {
		t.Error("Timing should be non-nil even when the call fails")
	}
}

func TestRunRecordsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Trace") != "yes" {
			t.Errorf("missing resolved header, got %q", r.Header.Get("X-Trace"))
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("missing query param, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{
		Method: "POST",
		URL:    srv.URL + "/things",
		Params: []store.KV{{Key: "page", Value: "2", Enabled: true}},
		Headers: []store.KV{
			{Key: "X-Trace", Value: "{{flag}}", Enabled: true},
			{Key: "X-Off", Value: "nope", Enabled: false},
		},
		BodyType: "json",
		Body:     `{"hello":"world"}`,
	}, map[string]string{"flag": "yes"}, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 201 {
		t.Errorf("status = %d", res.Status)
	}
	if res.Encoding != "text" || !strings.Contains(res.BodyText, `"ok"`) {
		t.Errorf("unexpected body: %q (%s)", res.BodyText, res.Encoding)
	}
	if res.DurationMs < 0 {
		t.Error("duration should be non-negative")
	}
}

func TestRunTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	opts := DefaultOpts()
	opts.Timeout = 50 * time.Millisecond

	_, err := ex.Run(context.Background(), store.Request{Method: "GET", URL: srv.URL}, nil, opts)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q", err)
	}
}

func TestRunDoesNotFollowRedirectsWhenDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/from" {
			http.Redirect(w, r, "/to", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("arrived"))
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	opts := DefaultOpts()
	opts.FollowRedirects = false

	res, err := ex.Run(context.Background(), store.Request{Method: "GET", URL: srv.URL + "/from"}, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusFound {
		t.Errorf("status = %d, want 302", res.Status)
	}
}

func TestBinaryBodyIsBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x00, 0x01, 0x02})
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()

	res, err := ex.Run(context.Background(), store.Request{Method: "GET", URL: srv.URL}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.Encoding != "base64" {
		t.Errorf("encoding = %q", res.Encoding)
	}
	if res.BodyText != "AAEC" {
		t.Errorf("body = %q", res.BodyText)
	}
}

func TestApplyBearerAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok123" {
			t.Errorf("auth header = %q", got)
		}
	}))
	defer srv.Close()

	ex := New()
	defer ex.Close()
	_, err := ex.Run(context.Background(), store.Request{
		Method:   "GET",
		URL:      srv.URL,
		AuthType: "bearer",
		Auth:     store.Auth{Token: "tok123"},
	}, nil, DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
}

func TestEncodeCurl(t *testing.T) {
	out := EncodeCurl(store.Request{
		Method:   "POST",
		URL:      "https://api.test/v1",
		Params:   []store.KV{{Key: "a", Value: "1", Enabled: true}},
		Headers:  []store.KV{{Key: "X-Key", Value: "v", Enabled: true}},
		BodyType: "json",
		Body:     `{"x":1}`,
	}, nil)

	for _, want := range []string{"curl -X POST", "https://api.test/v1?a=1", "-H 'X-Key: v'", "--data-raw"} {
		if !strings.Contains(out, want) {
			t.Errorf("curl output missing %q:\n%s", want, out)
		}
	}
}
