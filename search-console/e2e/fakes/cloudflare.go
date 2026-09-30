package fakes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// DNSRecord is one record the fake Cloudflare holds.
type DNSRecord struct {
	Type, Name, Content string
}

// Cloudflare mimics the zones and DNS records endpoints.
type Cloudflare struct {
	*httptest.Server
	// APIToken is the only bearer token the fake accepts.
	APIToken string
	// CreateStatus, when set, is the status every record create answers.
	CreateStatus int

	mu      sync.Mutex
	zones   map[string]string // zone name -> id
	records map[string][]DNSRecord
	calls   map[string]int
}

// NewCloudflare starts the fake. It accepts the bearer token "cf-test".
func NewCloudflare(t *testing.T) *Cloudflare {
	t.Helper()
	c := &Cloudflare{APIToken: "cf-test", zones: map[string]string{}, records: map[string][]DNSRecord{}, calls: map[string]int{}}
	c.Server = httptest.NewServer(http.HandlerFunc(c.serve))
	t.Cleanup(c.Close)
	return c
}

// AddZone registers a zone name and its id.
func (c *Cloudflare) AddZone(name, id string) {
	c.mu.Lock()
	c.zones[name] = id
	c.mu.Unlock()
}

// SeedRecord puts a record in a zone before the test runs.
func (c *Cloudflare) SeedRecord(zoneID, typ, name, content string) {
	c.mu.Lock()
	c.records[zoneID] = append(c.records[zoneID], DNSRecord{typ, name, content})
	c.mu.Unlock()
}

// Calls counts requests to "METHOD /path".
func (c *Cloudflare) Calls(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[key]
}

// Records returns the records held for a zone id.
func (c *Cloudflare) Records(zoneID string) []DNSRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]DNSRecord(nil), c.records[zoneID]...)
}

// HasTXT reports whether any zone holds a TXT record with this name and content.
func (c *Cloudflare) HasTXT(name, content string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, recs := range c.records {
		for _, r := range recs {
			if r.Type == "TXT" && r.Name == name && r.Content == content {
				return true
			}
		}
	}
	return false
}

func cfReply(w http.ResponseWriter, status int, result any, errMsg string) {
	body := map[string]any{"success": errMsg == "", "errors": []map[string]any{}, "result": result}
	if errMsg != "" {
		body["errors"] = []map[string]any{{"code": 10000, "message": errMsg}}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (c *Cloudflare) serve(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	c.calls[r.Method+" "+r.URL.Path]++
	c.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+c.APIToken {
		cfReply(w, http.StatusForbidden, nil, "Authentication error")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	q := r.URL.Query()
	if r.Method == http.MethodGet && r.URL.Path == "/zones" {
		out := []map[string]string{}
		if id, ok := c.zones[q.Get("name")]; ok {
			out = append(out, map[string]string{"id": id, "name": q.Get("name")})
		}
		cfReply(w, http.StatusOK, out, "")
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/zones/"), "/dns_records")
	known := false
	for _, z := range c.zones {
		known = known || z == id
	}
	if !strings.HasPrefix(r.URL.Path, "/zones/") || !strings.HasSuffix(r.URL.Path, "/dns_records") || !known {
		cfReply(w, http.StatusNotFound, nil, "Zone not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		out := []map[string]string{}
		for _, rec := range c.records[id] {
			if rec.Type == q.Get("type") && rec.Name == q.Get("name") && rec.Content == q.Get("content") {
				out = append(out, map[string]string{"type": rec.Type, "name": rec.Name, "content": rec.Content})
			}
		}
		cfReply(w, http.StatusOK, out, "")
	case http.MethodPost:
		var rec DNSRecord
		var body struct{ Type, Name, Content string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		rec = DNSRecord(body)
		if c.CreateStatus != 0 {
			cfReply(w, c.CreateStatus, nil, "Request is not allowed for this zone")
			return
		}
		for _, old := range c.records[id] {
			if old == rec {
				cfReply(w, http.StatusBadRequest, nil, "An identical record already exists.")
				return
			}
		}
		c.records[id] = append(c.records[id], rec)
		cfReply(w, http.StatusOK, map[string]string{"id": "rec-1"}, "")
	default:
		cfReply(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}
