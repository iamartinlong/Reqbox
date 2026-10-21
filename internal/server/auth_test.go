package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"reqbox/internal/logging"
	"reqbox/internal/store"
)

// newAuthServer builds a handler with auth enabled and returns the token.
func newAuthServer(t *testing.T) (http.Handler, *store.Store, string) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	log := logging.Discard()
	auth, err := EnsureStoredToken(st, "")
	if err != nil {
		t.Fatal(err)
	}
	return New(st, log, Config{Auth: auth}, nil), st, auth.Token()
}

// withToken sends a request carrying a bearer token.
func withToken(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// The endpoint that used to leak every stored credential. If this ever returns
// 200 without a token, the whole feature is pointless.
func TestAuthRejectsRequestListWithoutToken(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := do(t, h, "GET", "/api/requests", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %q)", rec.Code, rec.Body.String())
	}
	// The body must not hint at the data, only at the missing credential.
	if strings.Contains(rec.Body.String(), "token") && strings.Contains(rec.Body.String(), "SECRET") {
		t.Errorf("unauthorised body leaked data: %q", rec.Body.String())
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected a WWW-Authenticate header")
	}
}

func TestAuthAcceptsBearerToken(t *testing.T) {
	h, _, token := newAuthServer(t)

	rec := withToken(t, h, "GET", "/api/requests", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestAuthAcceptsBareToken(t *testing.T) {
	h, _, token := newAuthServer(t)

	// Some scripts send the raw value; accepting both avoids a confusing 401
	// that looks like a wrong token.
	req := httptest.NewRequest("GET", "/api/requests", nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAuthRejectsWrongToken(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := withToken(t, h, "GET", "/api/requests", "definitely-not-the-token", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// Health backs the header's online indicator. It must answer before sign-in,
// otherwise the unlock screen reports the server as offline.
func TestAuthLeavesHealthPublic(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := do(t, h, "GET", "/api/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// Published mocks are called by other people; requiring our token would defeat
// the feature. The mock does not exist here, so 404 proves it got past the
// guard rather than being rejected with 401.
func TestAuthLeavesMockGatewayPublic(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := do(t, h, "GET", "/mock/does-not-exist", nil)
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("mock gateway must not require a token")
	}
}

// Every other API route must be covered. A route added later without thinking
// about auth would otherwise be silently public.
func TestAuthGuardsEveryAPIRoute(t *testing.T) {
	h, _, _ := newAuthServer(t)

	routes := []struct{ method, path string }{
		{"GET", "/api/stats"},
		{"GET", "/api/requests"},
		{"POST", "/api/requests"},
		{"GET", "/api/collections"},
		{"POST", "/api/collections"},
		{"PUT", "/api/collections/x"},
		{"DELETE", "/api/collections/x"},
		{"POST", "/api/execute"},
		{"POST", "/api/curl"},
		{"GET", "/api/history"},
		{"DELETE", "/api/history"},
		{"GET", "/api/history/x"},
		{"GET", "/api/history/x/body"},
		{"GET", "/api/environments"},
		{"POST", "/api/environments"},
		{"PUT", "/api/environments/active"},
		{"DELETE", "/api/environments/x"},
		// The export endpoint dumps every credential in the workspace, so it is
		// the most important one to have covered.
		{"GET", "/api/export"},
		{"POST", "/api/import"},
	}
	for _, rt := range routes {
		rec := do(t, h, rt.method, rt.path, map[string]any{})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", rt.method, rt.path, rec.Code)
		}
	}
}

func TestAuthSessionExchangesTokenForCookie(t *testing.T) {
	h, _, token := newAuthServer(t)

	rec := withToken(t, h, "POST", "/api/session", "", map[string]string{"token": token})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookie {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session cookie was set")
	}
	if !session.HttpOnly {
		t.Error("session cookie must be HttpOnly so scripts cannot read it")
	}
	if session.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", session.SameSite)
	}
	// The cookie must not be the token itself, or stealing it would be
	// equivalent to stealing the plaintext.
	if session.Value == token {
		t.Error("cookie value must not equal the plaintext token")
	}

	// And the cookie must actually authorise requests.
	req := httptest.NewRequest("GET", "/api/requests", nil)
	req.AddCookie(session)
	inner := httptest.NewRecorder()
	h.ServeHTTP(inner, req)
	if inner.Code != http.StatusOK {
		t.Fatalf("cookie did not authorise: %d", inner.Code)
	}
}

func TestAuthSessionRejectsWrongToken(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := withToken(t, h, "POST", "/api/session", "", map[string]string{"token": "nope"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// Signing in must be possible without a token, or no one could ever get in.
func TestAuthSessionEndpointIsReachable(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := withToken(t, h, "POST", "/api/session", "", map[string]string{"token": ""})
	if rec.Code == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("the sign-in endpoint itself must not be behind the guard")
	}
}

// Pasting ?token= into the address bar is how a human signs in. The token must
// be traded for a cookie and then removed from the URL.
func TestAuthTokenQueryRedirectsAndStripsToken(t *testing.T) {
	h, _, token := newAuthServer(t)

	rec := do(t, h, "GET", "/?token="+token, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want / so the token leaves the address bar", loc)
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected a session cookie after the exchange")
	}
}

func TestAuthTokenQueryWithWrongTokenDoesNotSignIn(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := do(t, h, "GET", "/?token=wrong", nil)
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie && c.Value != "" {
			t.Fatal("a wrong token must not produce a session cookie")
		}
	}
}

func TestAuthLogoutClearsCookie(t *testing.T) {
	h, _, _ := newAuthServer(t)

	rec := do(t, h, "POST", "/api/logout", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie && c.MaxAge >= 0 {
			t.Errorf("logout should expire the cookie, got MaxAge=%d", c.MaxAge)
		}
	}
}

func TestAuthSessionStatusReflectsSignIn(t *testing.T) {
	h, _, token := newAuthServer(t)

	// Signed out: the SPA needs a distinguishable answer, not a plain 401 that
	// is indistinguishable from "the request failed".
	rec := do(t, h, "GET", "/api/session/status", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when signed out", rec.Code)
	}

	rec = withToken(t, h, "GET", "/api/session/status", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when signed in", rec.Code)
	}
}

// Restarting the app must not sign everyone out, or the token becomes a chore
// people work around by disabling auth.
func TestAuthTokenSurvivesRestart(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	first, err := EnsureStoredToken(st, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Token() == "" {
		t.Fatal("expected a generated token")
	}
	hash := st.GetSetting(SettingAuthToken, "")
	if hash == "" {
		t.Fatal("the hash should be persisted")
	}
	if hash == first.Token() {
		t.Error("the plaintext token must not be stored")
	}

	second, err := EnsureStoredToken(st, "")
	if err != nil {
		t.Fatal(err)
	}
	// The same hash is reused, so the old cookie keeps working.
	if second.hash != first.hash {
		t.Error("restart should reuse the stored token")
	}
	// The plaintext is not recoverable, which is the point of storing a hash.
	if second.Token() != "" {
		t.Error("a stored token should not be reprinted in plaintext")
	}
}

// An explicitly configured token must win, so operators can rotate it.
func TestConfiguredTokenOverridesStored(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	first, err := EnsureStoredToken(st, "")
	if err != nil {
		t.Fatal(err)
	}

	second, err := EnsureStoredToken(st, "configured-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if second.hash == first.hash {
		t.Fatal("the configured token should replace the stored hash")
	}
	if !second.matches("configured-token-value") {
		t.Error("configured token should be accepted")
	}
	if second.matches(first.Token()) {
		t.Error("the old generated token should no longer work")
	}
}

func TestAuthTokenTooShortIsRejected(t *testing.T) {
	if _, err := NewAuthToken("short"); err == nil {
		t.Error("a trivially short token should be refused")
	}
}

// With auth off the API behaves exactly as before, which is what -no-auth and
// the test helpers rely on.
func TestAuthOffLeavesEverythingOpen(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	log := logging.Discard()
	h := New(st, log, Config{}, nil)

	rec := do(t, h, "GET", "/api/requests", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when auth is off", rec.Code)
	}
}
