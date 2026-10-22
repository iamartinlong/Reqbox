package server

import (
	"reqbox/internal/logging"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"reqbox/internal/store"
)

// SettingAuthToken is the settings key holding the SHA-256 of the access token.
//
// Only the hash is stored. The point of the token is to keep someone who can
// reach the port from reading the credentials inside the database, so writing
// the token itself into that same database would undercut the whole exercise.
const SettingAuthToken = "authTokenHash"

// SessionCookie is the cookie the browser carries after presenting the token
// once. It holds the same hash rather than a separate secret: there is nothing
// to gain from a second credential, and one value is one thing to rotate.
const SessionCookie = "reqbox_session"

// AuthMode describes how the server authenticates API callers.
type AuthMode int

const (
	// AuthOff disables the check entirely. Only reachable via -no-auth, which
	// logs a warning because it exposes stored credentials to the network.
	AuthOff AuthMode = iota
	// AuthToken requires a bearer token.
	AuthToken
)

// Auth holds the token check. A zero value is "off", so a Config built without
// auth (tests, `-no-auth`) behaves the way it always did.
type Auth struct {
	Mode AuthMode
	// hash is the SHA-256 of the accepted token, hex encoded.
	hash string
	// token is the plaintext, kept only long enough to print it at startup.
	token string
	// generated distinguishes a token this process created from one the
	// operator supplied, purely so the startup log can say which happened.
	generated bool
}

// Enabled reports whether requests need a token.
func (a Auth) Enabled() bool { return a.Mode == AuthToken }

// Token returns the plaintext token, for the startup banner. It is empty when
// the token was supplied by the operator (-token) rather than generated here,
// since reprinting a configured secret adds nothing.
func (a Auth) Token() string { return a.token }

// NewAuthToken builds an Auth that accepts exactly the given token. An empty
// token generates a fresh one, which the caller should print once.
//
// The generated token is 32 bytes of crypto/rand, base64url without padding: a
// value the operator copies rather than types, so length costs nothing and
// removes any question of guessability.
func NewAuthToken(token string) (Auth, error) {
	if strings.TrimSpace(token) == "" {
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return Auth{}, fmt.Errorf("generate token: %w", err)
		}
		token = base64.RawURLEncoding.EncodeToString(raw[:])
	}
	token = strings.TrimSpace(token)
	if len(token) < 8 {
		return Auth{}, errors.New("token must be at least 8 characters")
	}
	return Auth{Mode: AuthToken, hash: hashToken(token), token: token, generated: true}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// matches compares a candidate token against the stored hash in constant time,
// so a caller cannot learn the token by measuring how long comparisons take.
func (a Auth) matches(candidate string) bool {
	if !a.Enabled() || candidate == "" {
		return false
	}
	want := []byte(a.hash)
	got := []byte(hashToken(candidate))
	return subtle.ConstantTimeCompare(want, got) == 1
}

// sessionValue is what the cookie carries. It is the same hash used for the
// bearer check, so a stolen cookie is not more useful than a stolen token.
func (a Auth) sessionValue() string { return a.hash }

// EnsureStoredToken resolves the token to use, preferring an explicit one from
// the operator so a container restart keeps working without re-reading logs.
//
// When no token is configured, the stored hash is reused, so restarting the app
// does not invalidate the browser session. If nothing is stored, a new token is
// generated and its hash persisted; the plaintext then exists only in memory
// and in the startup banner.
func EnsureStoredToken(st *store.Store, configured string) (Auth, error) {
	if strings.TrimSpace(configured) != "" {
		auth, err := NewAuthToken(configured)
		if err != nil {
			return Auth{}, err
		}
		// An operator-supplied token is not "generated": the log should not
		// claim to have invented a value the operator chose.
		auth.generated = false
		// A configured token supersedes anything stored, so a rotation takes
		// effect on the next start rather than being shadowed by the old hash.
		if err := st.SetSetting(SettingAuthToken, auth.hash); err != nil {
			return Auth{}, err
		}
		return auth, nil
	}

	if stored := st.GetSetting(SettingAuthToken, ""); stored != "" {
		// The plaintext is gone, so there is nothing to print; the session
		// survives, which is the point of persisting the hash.
		return Auth{Mode: AuthToken, hash: stored}, nil
	}

	auth, err := NewAuthToken("")
	if err != nil {
		return Auth{}, err
	}
	if err := st.SetSetting(SettingAuthToken, auth.hash); err != nil {
		return Auth{}, err
	}
	return auth, nil
}

// bearerFrom extracts the token from an Authorization header, accepting both
// "Bearer x" and a bare token so `curl -H "Authorization: x"` works too.
func bearerFrom(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return ""
	}
	if len(raw) > 7 && strings.EqualFold(raw[:7], "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	return raw
}

// authorize reports whether this request may proceed, checking the header
// first and then the session cookie.
func (a Auth) authorize(r *http.Request) bool {
	if !a.Enabled() {
		return true
	}
	if a.matches(bearerFrom(r)) {
		return true
	}
	if c, err := r.Cookie(SessionCookie); err == nil {
		// Compare the cookie against the stored hash directly: it already is
		// the hash, so hashing it again would never match.
		want := []byte(a.hash)
		got := []byte(c.Value)
		if len(want) == len(got) && subtle.ConstantTimeCompare(want, got) == 1 {
			return true
		}
	}
	return false
}

// publicPath reports whether a route is reachable without a token.
//
// Exclusions, all deliberate:
//
//   - /api/health backs the online indicator in the header. Requiring a token
//     would show "offline" to anyone not yet signed in, which is exactly the
//     state the unlock screen is trying to explain.
//   - /api/session is the sign-in endpoint. Guarding it would make signing in
//     impossible; it is the one place where presenting a token is the request.
//   - /api/logout must work even with a stale cookie.
//   - /mock/* is the published-mock gateway. Its whole purpose is to be called
//     by someone else; requiring our token would defeat that. Those requests
//     run with the owner's saved credentials, which is the published
//     capability they agreed to.
func publicPath(path string) bool {
	switch path {
	case "/api/health", "/api/session", "/api/session/status", "/api/logout":
		return true
	}
	return path == "/mock" || strings.HasPrefix(path, "/mock/")
}

// authGuard rejects API calls that present no valid token.
//
// It only guards /api and /mock. Static assets must stay open or the unlock
// screen could not load, and the root page is what turns ?token= into a cookie.
func (a Auth) guard(next http.Handler) http.Handler {
	if !a.Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		// Only the JSON API is protected. The SPA shell has to be loadable so
		// the unlock screen can render.
		if !strings.HasPrefix(p, "/api/") && p != "/api" {
			next.ServeHTTP(w, r)
			return
		}
		if publicPath(p) || a.authorize(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer realm="reqbox"`)
		writeErr(w, http.StatusUnauthorized, "missing or invalid API token")
	})
}

// exchangeToken turns a ?token= query parameter into a session cookie.
//
// The token is accepted in the URL because that is the one thing a human can
// paste into a browser. It is immediately removed from the address bar with a
// redirect, so it does not linger in history or get copied when someone shares
// the URL.
func (a Auth) exchangeToken(w http.ResponseWriter, r *http.Request) bool {
	if !a.Enabled() {
		return false
	}
	token := r.URL.Query().Get("token")
	if token == "" || !a.matches(token) {
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    a.sessionValue(),
		Path:     "/",
		HttpOnly: true,
		// Strict rather than Lax: this cookie grants access to every stored
		// credential, and nothing here needs to work from a cross-site link.
		SameSite: http.SameSiteStrictMode,
		// Secure is deliberately not set: the tool is usually served over
		// plain HTTP on localhost, where a Secure cookie would never be sent.
		MaxAge: 60 * 60 * 24 * 30,
	})

	// Strip the token from the URL. Keeping other query parameters is not
	// needed (the SPA reads none), and a bare "/" avoids echoing anything.
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return true
}

// LogToken prints the token once at startup, with a ready-to-click URL.
//
// Printing the token is the intended way to hand it to the user: it never lands
// in a file, and it is the only copy unless they configured one.
// LoginURL returns the one-time sign-in link, or "" when there is no token to
// put in it.
//
// The caller passes the base URL that the listener actually bound to, so this
// works for `-addr :0` and for a port the server had to fall back to. Building
// the URL from the configured address instead would print a link to a port
// nothing is listening on.
func (a Auth) LoginURL(baseURL string) string {
	if !a.Enabled() || a.token == "" {
		return ""
	}
	return baseURL + "/?token=" + a.token
}

// LogToken prints the token once at startup, with a ready-to-click URL.
//
// Printing the token is the intended way to hand it to the user: it never lands
// in a file, and it is the only copy unless they configured one.
func (a Auth) LogToken(log *logging.Logger, baseURL string) {
	if !a.Enabled() {
		log.Warn("authentication is DISABLED: anyone who can reach this port can read every stored credential")
		return
	}
	if a.token == "" {
		log.Info("using the previously generated API token (open the app to sign in)")
		return
	}
	url := a.LoginURL(baseURL)
	if a.generated {
		log.Info("API token generated; open this URL once to sign in",
			"token", a.token, "url", url)
		return
	}
	log.Info("using the configured API token", "url", url)
}

// displayAddr turns a listen address into something clickable.
func displayAddr(addr string) string {
	if addr == "" {
		return "127.0.0.1:5178"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return host + ":" + port
}
