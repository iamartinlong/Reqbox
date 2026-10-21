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
