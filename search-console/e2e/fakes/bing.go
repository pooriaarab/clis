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
}

// Bing mimics the Bing Webmaster JSON API. Replies wrap results in {"d": ...}.
type Bing struct {
	*httptest.Server
	// Key is the only API key the fake accepts.
	Key string
	// ErrorsAs200 makes every error arrive as HTTP 200 with an ErrorCode body.
	ErrorsAs200 bool
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
	b.sites[url] = &s
	b.mu.Unlock()
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
	default:
		b.fail(w, 1, "unknown method "+name)
	}
}
