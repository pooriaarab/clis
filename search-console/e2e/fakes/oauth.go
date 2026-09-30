// Package fakes holds local HTTP servers that mimic Google, Cloudflare, Bing,
// IndexNow and a site. The E2E tests point the real binary at them.
package fakes

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// OAuth mimics the Google consent page and token endpoint.
type OAuth struct {
	*httptest.Server
	// Mode changes the consent result: "deny", "badstate" or "norefresh".
	Mode string

	mu        sync.Mutex
	challenge string
	redirect  string
	tokenHits int
}

// NewOAuth starts the fake. URLs are OAuth.URL+"/auth" and OAuth.URL+"/token".
func NewOAuth(t *testing.T) *OAuth {
	t.Helper()
	o := &OAuth{}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", o.auth)
	mux.HandleFunc("/token", o.token)
	o.Server = httptest.NewServer(mux)
	t.Cleanup(o.Close)
	return o
}

// TokenHits counts calls to the token endpoint.
func (o *OAuth) TokenHits() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.tokenHits
}

func (o *OAuth) auth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scope := q.Get("scope")
	if q.Get("client_id") == "" || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" ||
		q.Get("access_type") != "offline" || !strings.Contains(scope, "/auth/siteverification") || !strings.Contains(scope, "/auth/webmasters") {
		http.Error(w, "bad authorization request", http.StatusBadRequest)
		return
	}
	o.mu.Lock()
	o.challenge, o.redirect = q.Get("code_challenge"), q.Get("redirect_uri")
	o.mu.Unlock()
	state, result := q.Get("state"), "code=code-1"
	switch o.Mode {
	case "deny":
		result = "error=access_denied"
	case "badstate":
		state = "wrong"
	}
	http.Redirect(w, r, q.Get("redirect_uri")+"?"+result+"&state="+state, http.StatusFound)
}

func (o *OAuth) token(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	o.tokenHits++
	challenge, redirect := o.challenge, o.redirect
	o.mu.Unlock()
	_ = r.ParseForm()
	fail := func(code string) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": "fake " + code})
	}
	f := r.PostForm
	if f.Get("grant_type") == "refresh_token" {
		if rt := f.Get("refresh_token"); rt != "rt-valid" && rt != "rt-env" {
			fail("invalid_grant")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at-refreshed", "expires_in": 3600, "token_type": "Bearer"})
		return
	}
	sum := sha256.Sum256([]byte(f.Get("code_verifier")))
	if f.Get("grant_type") != "authorization_code" || f.Get("client_secret") != "csecret" || f.Get("code") != "code-1" ||
		f.Get("redirect_uri") != redirect || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
		fail("invalid_grant")
		return
	}
	resp := map[string]any{"access_token": "at-login", "expires_in": 3600, "token_type": "Bearer"}
	if o.Mode != "norefresh" {
		resp["refresh_token"] = "rt-valid"
	}
	_ = json.NewEncoder(w).Encode(resp)
}
