package fakes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// BingSite is one site the fake Bing account holds.
type BingSite struct {
	Verified bool
	// DNSCode is the bare code. The fake reports "<code>.<host>", as the real
	// GetUserSites does.
	DNSCode string
	url     string
	host    string
	feeds   []map[string]any
}

// Bing mimics the Bing Webmaster JSON API. Replies wrap results in {"d": ...}.
type Bing struct {
	*httptest.Server
	// Key is the only API key the fake accepts.
	Key string
	// ErrorsAs200 makes every error arrive as HTTP 200 with an ErrorCode body.
	ErrorsAs200 bool
	// Ready reports whether DNS shows the verification CNAME. Nil means always.
	Ready func(name, target string) bool
	// VerifyError makes VerifySite answer an error body with this message.
	VerifyError string
	// FeedStatus is the status a submitted sitemap gets. The default is "Success",
	// the only status seen on the live API.
	FeedStatus string
	// DuplicateFeedIs81058 makes a resubmit answer error 81058 (already present).
	DuplicateFeedIs81058 bool
	// HideSitesCalls makes the first N GetUserSites calls answer an empty list,
	// as a listing that lags behind AddSite.
	HideSitesCalls int
	// EchoRequest makes every error message repeat the request URL, which holds the key.
	EchoRequest bool
	// LongError pads every error message with this many characters.
	LongError int
	// RawErrors makes every error an HTML page, not a JSON body.
	RawErrors bool
	// HideFeeds makes GetFeeds answer an empty list, as Bing does before it reads a sitemap.
	HideFeeds bool
	// Daily and Monthly are the URL submission quotas.
	Daily, Monthly int

	mu    sync.Mutex
	sites map[string]*BingSite
	calls map[string]int
}

// NewBing starts the fake. It accepts the API key "bing-key".
func NewBing(t *testing.T) *Bing {
	t.Helper()
	b := &Bing{Key: "bing-key", Daily: 10000, Monthly: 300000, sites: map[string]*BingSite{}, calls: map[string]int{}}
	b.Server = httptest.NewServer(http.HandlerFunc(b.serve))
	t.Cleanup(b.Close)
	return b
}

// Configure changes the fake while the server runs. fn runs under the fake's lock, so a
// handler never reads a field half-written.
func (b *Bing) Configure(fn func(*Bing)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fn(b)
}

// AddSite registers a site the account already holds.
func (b *Bing) AddSite(url string, s BingSite) {
	b.mu.Lock()
	s.url, s.host = url, hostOf(url)
	b.sites[url] = &s
	b.mu.Unlock()
}

// Verified reports whether the site is verified in the account.
func (b *Bing) Verified(url string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.find(url)
	return s != nil && s.Verified
}

// find looks a site up the way the real API does: scheme, case and a trailing
// slash do not matter. The caller holds the lock.
func (b *Bing) find(url string) *BingSite {
	for _, s := range b.sites {
		if hostOf(s.url) == hostOf(url) {
			return s
		}
	}
	return nil
}

func hostOf(url string) string {
	url = strings.ToLower(url)
	url = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	return strings.Trim(url, "/")
}

// Feeds lists the sitemap URLs submitted for a site.
func (b *Bing) Feeds(siteURL string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	if s := b.find(siteURL); s != nil {
		for _, f := range s.feeds {
			out = append(out, f["Url"].(string))
		}
	}
	return out
}

// Calls counts requests to a method name such as "GetUserSites".
func (b *Bing) Calls(name string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls[name]
}

// fail answers an error. b.mu is held.
func (b *Bing) fail(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if b.EchoRequest {
		msg += " (request: " + r.URL.String() + ")"
	}
	msg += strings.Repeat("x", b.LongError)
	if b.RawErrors {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>" + msg + "</html>"))
		return
	}
	status := http.StatusBadRequest
	if b.ErrorsAs200 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"ErrorCode": code, "Message": msg})
}

func (b *Bing) ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"d": v})
}

func (b *Bing) serve(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	b.mu.Lock() // held for the whole request: fail reads ErrorsAs200 and the cases read the settings
	defer b.mu.Unlock()
	b.calls[name]++
	if r.URL.Query().Get("apikey") != b.Key {
		b.fail(w, r, 3, "ERROR!!! InvalidApiKey")
		return
	}
	switch {
	case r.Method == http.MethodGet && name == "GetUserSites":
		out := []map[string]any{}
		if b.calls[name] > b.HideSitesCalls {
			for url, s := range b.sites {
				// The real reply: DnsVerificationCode is the whole record name, "<code>.<host>".
				out = append(out, map[string]any{"__type": "Site:#Microsoft.Bing.Webmaster.Api", "AuthenticationCode": "619CF6025D0F59C03351F8E3980934CD",
					"DnsVerificationCode": s.DNSCode + "." + s.host, "IsVerified": s.Verified, "Url": url})
			}
		}
		b.ok(w, out)
	case r.Method == http.MethodGet && name == "GetUrlSubmissionQuota":
		if b.find(r.URL.Query().Get("siteUrl")) == nil {
			b.fail(w, r, 14, "ERROR!!! NotAuthorized") // the live answer for a site that is not in the account
			return
		}
		b.ok(w, map[string]any{"__type": "UrlSubmissionQuota:#Microsoft.Bing.Webmaster.Api", "DailyQuota": b.Daily, "MonthlyQuota": b.Monthly})
	case r.Method == http.MethodPost && name == "AddSite":
		var body struct{ SiteURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasPrefix(body.SiteURL, "https://") {
			b.fail(w, r, 2, "ERROR!!! InvalidParameter: bad siteUrl")
			return
		}
		if b.find(body.SiteURL) != nil {
			// Code 81058 is from the code review of the CLI. The message text was never
			// seen live, because AddSite is not safe to call against a real account.
			b.fail(w, r, 81058, "ERROR!!! the site is already present")
			return
		}
		b.sites[body.SiteURL] = &BingSite{DNSCode: "0123456789abcdef0123456789abcdef", url: body.SiteURL, host: hostOf(body.SiteURL)}
		b.ok(w, nil)
	case r.Method == http.MethodPost && name == "VerifySite":
		var body struct{ SiteURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		s := b.find(body.SiteURL)
		switch {
		case s == nil:
			b.fail(w, r, 3, "ERROR!!! InvalidParameter: site not found")
		case b.VerifyError != "":
			b.fail(w, r, 9, b.VerifyError)
		case b.Ready == nil || b.Ready(s.DNSCode+"."+s.host, "verify.bing.com"):
			s.Verified = true
			b.ok(w, true)
		default:
			b.ok(w, false)
		}
	case r.Method == http.MethodPost && name == "SubmitFeed":
		var body struct{ SiteURL, FeedURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		s := b.find(body.SiteURL)
		switch {
		case s == nil || !s.Verified:
			b.fail(w, r, 3, "ERROR!!! InvalidParameter: site is not verified")
		case !strings.HasPrefix(body.FeedURL, "http"):
			b.fail(w, r, 2, "ERROR!!! InvalidParameter: bad feedUrl")
		default:
			status := b.FeedStatus
			if status == "" {
				status = "Success"
			}
			// The live feed shape. Bing sends the date 1601-01-01 for "never": it was seen on Submitted.
			feed := map[string]any{"__type": "Feed:#Microsoft.Bing.Webmaster.Api", "Compressed": false, "FileSize": 0, "Url": body.FeedURL,
				"Type": "Sitemap", "Status": status, "Submitted": "/Date(1700000000000)/", "LastCrawled": "/Date(-11644473600000)/", "UrlCount": 0}
			for i, f := range s.feeds {
				if f["Url"] == body.FeedURL { // a resubmit replaces the feed, it does not add one
					if b.DuplicateFeedIs81058 {
						b.fail(w, r, 81058, "ERROR!!! the feed is already present")
						return
					}
					s.feeds[i] = feed
					b.ok(w, nil)
					return
				}
			}
			s.feeds = append(s.feeds, feed)
			b.ok(w, nil)
		}
	case r.Method == http.MethodGet && name == "GetFeeds":
		s := b.find(r.URL.Query().Get("siteUrl"))
		if s == nil {
			b.fail(w, r, 14, "ERROR!!! NotAuthorized")
			return
		}
		out := []map[string]any{}
		if !b.HideFeeds {
			out = append(out, s.feeds...)
		}
		b.ok(w, out)
	default:
		b.fail(w, r, 1, "unknown method "+name)
	}
}
