package fakes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	// PendingPolls is how many sitemap status reads answer isPending before Google "reads" it.
	PendingPolls int
	// SitemapErrors and SitemapWarnings are the counts Google reports once it read the sitemap.
	SitemapErrors, SitemapWarnings int
	// SitemapStatus, when set, is the status the sitemap submit answers.
	SitemapStatus int
	// Permission is the level sites.get reports for an added site. Default "siteOwner".
	Permission string

	mu       sync.Mutex
	calls    map[string]int
	verified map[string]bool
	sites    map[string]bool
	feeds    map[string]int // submitted sitemap URL -> status reads so far
}

// NewGoogle starts the fake. It accepts the bearer token "at-test".
func NewGoogle(t *testing.T) *Google {
	t.Helper()
	g := &Google{AccessToken: "at-test", Permission: "siteOwner",
		calls: map[string]int{}, verified: map[string]bool{}, sites: map[string]bool{}, feeds: map[string]int{}}
	g.Server = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.Close)
	return g
}

// Configure changes the fake while the server runs. fn runs under the fake's lock, so a
// handler never reads a field half-written.
func (g *Google) Configure(fn func(*Google)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	fn(g)
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

// PreAdd marks a domain as verified and added to Search Console.
func (g *Google) PreAdd(domain string) {
	g.PreVerify(domain)
	g.mu.Lock()
	g.sites["sc-domain:"+domain] = true
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
	token := g.AccessToken
	g.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer "+token {
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
	case strings.HasPrefix(r.URL.EscapedPath(), sitePrefix):
		g.site(w, r, strings.TrimPrefix(r.URL.EscapedPath(), sitePrefix))
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
	g.mu.Lock()
	cfg := struct {
		tokenStatus, verifyStatus, readyAfter int
		ready                                 func(domain, record string) bool
	}{g.TokenStatus, g.VerifyStatus, g.ReadyAfter, g.Ready}
	g.mu.Unlock()
	if body.Site.Type != "INET_DOMAIN" || body.Site.Identifier == "" {
		apiError(w, http.StatusBadRequest, "site must be INET_DOMAIN with an identifier")
		return
	}
	if key == "POST /siteVerification/v1/token" {
		if cfg.tokenStatus != 0 {
			apiError(w, cfg.tokenStatus, "Site Verification API has not been used in this project")
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
	if cfg.verifyStatus != 0 {
		apiError(w, cfg.verifyStatus, "verify failed")
		return
	}
	notReady := cfg.readyAfter < 0 || n <= cfg.readyAfter
	if cfg.ready != nil {
		notReady = !cfg.ready(body.Site.Identifier, "google-site-verification=fake-"+body.Site.Identifier)
	}
	if notReady {
		apiError(w, http.StatusBadRequest, "The necessary verification token could not be found on your site.")
		return
	}
	g.PreVerify(body.Site.Identifier)
	reply(w, http.StatusOK, map[string]any{"id": "id-1", "site": body.Site, "owners": []string{"owner@example.test"}})
}

// Sitemaps returns the sitemap URLs that were submitted.
func (g *Google) Sitemaps() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := []string{}
	for f := range g.feeds {
		out = append(out, f)
	}
	return out
}

// site serves sites.add (PUT), sites.get (GET) and the sitemaps calls. The
// paths must carry ":" and "/" percent-encoded, like the real API expects.
func (g *Google) site(w http.ResponseWriter, r *http.Request, rest string) {
	if !strings.HasPrefix(rest, "sc-domain%3A") {
		apiError(w, http.StatusBadRequest, "site URL must be percent-encoded")
		return
	}
	siteEsc, feedEsc, isFeed := strings.Cut(rest, "/sitemaps/")
	name, _ := url.PathUnescape(siteEsc)
	domain := strings.TrimPrefix(name, "sc-domain:")
	g.mu.Lock()
	defer g.mu.Unlock()
	if isFeed {
		g.feed(w, r, name, feedEsc)
		return
	}
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

// feed serves sitemaps.submit (PUT) and sitemaps.get (GET). g.mu is held.
func (g *Google) feed(w http.ResponseWriter, r *http.Request, site, feedEsc string) {
	if strings.Contains(feedEsc, "/") {
		apiError(w, http.StatusBadRequest, "feedpath must be percent-encoded")
		return
	}
	feed, _ := url.PathUnescape(feedEsc)
	if !g.sites[site] {
		apiError(w, http.StatusNotFound, "Site not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		if g.SitemapStatus != 0 {
			apiError(w, g.SitemapStatus, "sitemap refused")
			return
		}
		g.feeds[feed] = 0
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		reads, ok := g.feeds[feed]
		if !ok {
			apiError(w, http.StatusNotFound, "Sitemap not found")
			return
		}
		g.feeds[feed] = reads + 1
		st := map[string]any{"path": feed, "lastSubmitted": "2026-01-02T03:04:05.000Z", "isPending": reads < g.PendingPolls,
			"isSitemapsIndex": false, "errors": "0", "warnings": "0"}
		if reads >= g.PendingPolls {
			st["lastDownloaded"] = "2026-01-02T03:05:00.000Z"
			st["errors"], st["warnings"] = fmt.Sprint(g.SitemapErrors), fmt.Sprint(g.SitemapWarnings)
		}
		reply(w, http.StatusOK, st)
	default:
		apiError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
