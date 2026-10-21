package server

import (
	"net/http"
	"strings"
)

// handleSession signs the browser in from a posted token.
//
// This exists in addition to the ?token= link because that link is only good
// once: after the redirect the token is gone from the address bar, so a user
// who clears cookies (or opens a second browser) needs a place to paste it.
func (a *api) handleSession(w http.ResponseWriter, r *http.Request) {
	if !a.auth.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": false})
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	token := strings.TrimSpace(body.Token)
	if !a.auth.matches(token) {
		// Deliberately vague: distinguishing "no token" from "wrong token" would
		// tell a prober that a token exists and is merely wrong.
		writeErr(w, http.StatusUnauthorized, "invalid token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    a.auth.sessionValue(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 30,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSessionStatus tells the SPA whether it needs to show the unlock screen.
//
// It answers even without a token, because the SPA calls it before it knows
// anything: a 200 means "you are in", a 401 means "show the unlock screen".
// Making it public and returning the status explicitly avoids the frontend
// having to treat a logged-out state as an error.
func (a *api) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	if !a.auth.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": false})
		return
	}
	if !a.auth.authorize(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "auth": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": true})
}

// handleLogout clears the session cookie, which is what "sign out" means when
// the browser is trusted by cookie alone.
func (a *api) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
