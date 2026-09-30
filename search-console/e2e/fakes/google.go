package fakes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Google mimics the Site Verification API.
type Google struct {
	*httptest.Server
	// AccessToken is the only bearer token the fake accepts.
	AccessToken string
	// ReadyAfter is how many verify calls answer 400 before the TXT record "appears".
	// A negative value means never.
	ReadyAfter int
	// VerifyStatus, when set, is the status every verify call answers.
	VerifyStatus int
	// TokenStatus, when set, is the status the token call answers.
	TokenStatus int

	mu    sync.Mutex
	calls map[string]int
}

// NewGoogle starts the fake. It accepts the bearer token "at-test".
func NewGoogle(t *testing.T) *Google {
	t.Helper()
	g := &Google{AccessToken: "at-test", calls: map[string]int{}}
	g.Server = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.Close)
	return g
}

// Calls counts requests to "METHOD /path".
func (g *Google) Calls(key string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls[key]
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]any{"error": map[string]any{"code": status, "message": msg}})
}

func (g *Google) serve(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	key := r.Method + " " + r.URL.Path
	g.calls[key]++
	n := g.calls[key]
	g.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer "+g.AccessToken {
		apiError(w, http.StatusUnauthorized, "Invalid Credentials")
		return
	}
	var body struct {
		Site               struct{ Type, Identifier string }
		VerificationMethod string
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Site.Type != "INET_DOMAIN" || body.Site.Identifier == "" {
		apiError(w, http.StatusBadRequest, "site must be INET_DOMAIN with an identifier")
		return
	}
	switch key {
	case "POST /siteVerification/v1/token":
		if g.TokenStatus != 0 {
			apiError(w, g.TokenStatus, "Site Verification API has not been used in this project")
			return
		}
		if body.VerificationMethod != "DNS_TXT" {
			apiError(w, http.StatusBadRequest, "verificationMethod must be DNS_TXT")
			return
		}
		reply(w, http.StatusOK, map[string]string{"method": "DNS_TXT", "token": "google-site-verification=fake-" + body.Site.Identifier})
	case "POST /siteVerification/v1/webResource":
		if r.URL.Query().Get("verificationMethod") != "DNS_TXT" {
			apiError(w, http.StatusBadRequest, "verificationMethod must be DNS_TXT")
			return
		}
		if g.VerifyStatus != 0 {
			apiError(w, g.VerifyStatus, "verify failed")
			return
		}
		if g.ReadyAfter < 0 || n <= g.ReadyAfter {
			apiError(w, http.StatusBadRequest, "The necessary verification token could not be found on your site.")
			return
		}
		reply(w, http.StatusOK, map[string]any{"id": "id-1", "site": body.Site, "owners": []string{"owner@example.test"}})
	default:
		apiError(w, http.StatusNotFound, fmt.Sprintf("no route %s", key))
	}
}
