package fakes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Google mimics the Site Verification and Search Console APIs.
type Google struct {
	*httptest.Server
	// AccessToken is the only bearer token the fake accepts.
	AccessToken string
	// ReadyAfter is how many verify calls answer 400 before the TXT record "appears".
	// A negative value means never.
	ReadyAfter int
	// Ready, when set, replaces ReadyAfter: it reports whether DNS shows the TXT
	// record for the domain.
	Ready func(domain, record string) bool
	// VerifyStatus, when set, is the status every verify call answers.
	VerifyStatus int
	// TokenStatus, when set, is the status the token call answers.
	TokenStatus int
	// AddStatus, when set, is the status the sites.add call answers.
	AddStatus int
	// Permission is the level sites.get reports for an added site. Default "siteOwner".
	Permission string

	mu       sync.Mutex
	calls    map[string]int
	verified map[string]bool
	sites    map[string]bool
}

// NewGoogle starts the fake. It accepts the bearer token "at-test".
func NewGoogle(t *testing.T) *Google {
	t.Helper()
	g := &Google{AccessToken: "at-test", Permission: "siteOwner",
		calls: map[string]int{}, verified: map[string]bool{}, sites: map[string]bool{}}
	g.Server = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.Close)
	return g
}

// Calls counts requests to "METHOD /path". Site paths use the decoded name.
func (g *Google) Calls(key string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls[key]
}

// PreVerify marks a domain as already owned by the account.
func (g *Google) PreVerify(domain string) {
	g.mu.Lock()
	g.verified[domain] = true
	g.mu.Unlock()
}

// HasSite reports whether sites.add ran for the domain property.
func (g *Google) HasSite(domain string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sites["sc-domain:"+domain]
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
	const sitePrefix = "/webmasters/v3/sites/"
	key := r.Method + " " + r.URL.Path
	g.mu.Lock()
	g.calls[key]++
	n := g.calls[key]
	g.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer "+g.AccessToken {
		apiError(w, http.StatusUnauthorized, "Invalid Credentials")
		return
	}
	switch {
	case key == "POST /siteVerification/v1/token" || key == "POST /siteVerification/v1/webResource":
		g.verification(w, r, key, n)
	case key == "GET /siteVerification/v1/webResource":
		items := []map[string]any{}
		g.mu.Lock()
		for d := range g.verified {
			items = append(items, map[string]any{"id": d, "site": map[string]string{"type": "INET_DOMAIN", "identifier": d}})
		}
		g.mu.Unlock()
		reply(w, http.StatusOK, map[string]any{"items": items})
	case strings.HasPrefix(r.URL.Path, sitePrefix):
		g.site(w, r, strings.TrimPrefix(r.URL.Path, sitePrefix))
	default:
		apiError(w, http.StatusNotFound, fmt.Sprintf("no route %s", key))
	}
}

func (g *Google) verification(w http.ResponseWriter, r *http.Request, key string, n int) {
	var body struct {
		Site               struct{ Type, Identifier string }
		VerificationMethod string
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Site.Type != "INET_DOMAIN" || body.Site.Identifier == "" {
		apiError(w, http.StatusBadRequest, "site must be INET_DOMAIN with an identifier")
		return
	}
	if key == "POST /siteVerification/v1/token" {
		if g.TokenStatus != 0 {
			apiError(w, g.TokenStatus, "Site Verification API has not been used in this project")
			return
		}
		if body.VerificationMethod != "DNS_TXT" {
			apiError(w, http.StatusBadRequest, "verificationMethod must be DNS_TXT")
			return
		}
		reply(w, http.StatusOK, map[string]string{"method": "DNS_TXT", "token": "google-site-verification=fake-" + body.Site.Identifier})
		return
	}
	if r.URL.Query().Get("verificationMethod") != "DNS_TXT" {
		apiError(w, http.StatusBadRequest, "verificationMethod must be DNS_TXT")
		return
	}
	if g.VerifyStatus != 0 {
		apiError(w, g.VerifyStatus, "verify failed")
		return
	}
	notReady := g.ReadyAfter < 0 || n <= g.ReadyAfter
	if g.Ready != nil {
		notReady = !g.Ready(body.Site.Identifier, "google-site-verification=fake-"+body.Site.Identifier)
	}
	if notReady {
		apiError(w, http.StatusBadRequest, "The necessary verification token could not be found on your site.")
		return
	}
	g.PreVerify(body.Site.Identifier)
	reply(w, http.StatusOK, map[string]any{"id": "id-1", "site": body.Site, "owners": []string{"owner@example.test"}})
}

// site serves sites.add (PUT) and sites.get (GET). The path must carry the
// colon as %3A, like the real API expects.
func (g *Google) site(w http.ResponseWriter, r *http.Request, name string) {
	if !strings.Contains(r.URL.EscapedPath(), "sc-domain%3A") {
		apiError(w, http.StatusBadRequest, "site URL must be percent-encoded")
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	domain := strings.TrimPrefix(name, "sc-domain:")
	switch r.Method {
	case http.MethodPut:
		if g.AddStatus != 0 {
			apiError(w, g.AddStatus, "User does not have sufficient permission for site")
			return
		}
		if !g.verified[domain] {
			apiError(w, http.StatusForbidden, "The site is not verified")
			return
		}
		g.sites[name] = true
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		if !g.sites[name] {
			apiError(w, http.StatusNotFound, "Site not found")
			return
		}
		reply(w, http.StatusOK, map[string]string{"siteUrl": name, "permissionLevel": g.Permission})
	default:
		apiError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
