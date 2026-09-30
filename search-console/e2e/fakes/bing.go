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
	DNSCode  string
	host     string
	feeds    []map[string]any
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
	// FeedStatus is the status a submitted sitemap gets. The default is "Pending".
	FeedStatus string
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

// AddSite registers a site the account already holds.
func (b *Bing) AddSite(url string, s BingSite) {
	b.mu.Lock()
	s.host = strings.Trim(strings.TrimPrefix(url, "https://"), "/")
	b.sites[url] = &s
	b.mu.Unlock()
}

// Verified reports whether the site is verified in the account.
func (b *Bing) Verified(url string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.sites[url]
	return ok && s.Verified
}

// Feeds lists the sitemap URLs submitted for a site.
func (b *Bing) Feeds(siteURL string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	if s := b.sites[siteURL]; s != nil {
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

func (b *Bing) fail(w http.ResponseWriter, code int, msg string) {
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
	b.mu.Lock()
	b.calls[name]++
	b.mu.Unlock()
	if r.URL.Query().Get("apikey") != b.Key {
		b.fail(w, 14, "NotAuthorized")
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && name == "GetUserSites":
		out := []map[string]any{}
		for url, s := range b.sites {
			out = append(out, map[string]any{"Url": url, "IsVerified": s.Verified, "DnsVerificationCode": s.DNSCode})
		}
		b.ok(w, out)
	case r.Method == http.MethodGet && name == "GetUrlSubmissionQuota":
		if _, ok := b.sites[r.URL.Query().Get("siteUrl")]; !ok {
			b.fail(w, 3, "ERROR!!! InvalidParameter: siteUrl is not a site of this account")
			return
		}
		b.ok(w, map[string]int{"DailyQuota": b.Daily, "MonthlyQuota": b.Monthly})
	case r.Method == http.MethodPost && name == "AddSite":
		var body struct{ SiteURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, dup := b.sites[body.SiteURL]; dup || !strings.HasPrefix(body.SiteURL, "https://") {
			b.fail(w, 2, "ERROR!!! InvalidParameter: site already added or bad siteUrl")
			return
		}
		host := strings.Trim(strings.TrimPrefix(body.SiteURL, "https://"), "/")
		b.sites[body.SiteURL] = &BingSite{DNSCode: "0123456789abcdef0123456789abcdef", host: host}
		b.ok(w, nil)
	case r.Method == http.MethodPost && name == "VerifySite":
		var body struct{ SiteURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		s, found := b.sites[body.SiteURL]
		switch {
		case !found:
			b.fail(w, 3, "ERROR!!! InvalidParameter: site not found")
		case b.VerifyError != "":
			b.fail(w, 9, b.VerifyError)
		case b.Ready == nil || b.Ready(s.DNSCode+"."+s.host, "verify.bing.com"):
			s.Verified = true
			b.ok(w, true)
		default:
			b.ok(w, false)
		}
	case r.Method == http.MethodPost && name == "SubmitFeed":
		var body struct{ SiteURL, FeedURL string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		s, found := b.sites[body.SiteURL]
		switch {
		case !found || !s.Verified:
			b.fail(w, 3, "ERROR!!! InvalidParameter: site is not verified")
		case !strings.HasPrefix(body.FeedURL, "http"):
			b.fail(w, 2, "ERROR!!! InvalidParameter: bad feedUrl")
		default:
			status := b.FeedStatus
			if status == "" {
				status = "Pending"
			}
			feed := map[string]any{"Url": body.FeedURL, "Type": "Sitemap", "Status": status,
				"Submitted": "/Date(1700000000000)/", "LastCrawled": "/Date(-62135596800000)/", "UrlCount": 0}
			for i, f := range s.feeds {
				if f["Url"] == body.FeedURL { // a resubmit replaces the feed, it does not add one
					s.feeds[i] = feed
					b.ok(w, nil)
					return
				}
			}
			s.feeds = append(s.feeds, feed)
			b.ok(w, nil)
		}
	case r.Method == http.MethodGet && name == "GetFeeds":
		s, found := b.sites[r.URL.Query().Get("siteUrl")]
		if !found {
			b.fail(w, 3, "ERROR!!! InvalidParameter: siteUrl is not a site of this account")
			return
		}
		out := []map[string]any{}
		if !b.HideFeeds {
			out = append(out, s.feeds...)
		}
		b.ok(w, out)
	default:
		b.fail(w, 1, "unknown method "+name)
	}
}
