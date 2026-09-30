package fakes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// IndexNowPost is one batch the fake received.
type IndexNowPost struct {
	Host, Key, KeyLocation string
	URLs                   []string
}

// IndexNow mimics the IndexNow endpoint. Like the real one, it fetches the
// key file before it accepts a batch.
type IndexNow struct {
	*httptest.Server
	// Answer maps a batch number (from 0) to a status. Other batches get the default 200.
	Answer map[int]int

	mu    sync.Mutex
	posts []IndexNowPost
}

var keyRE = regexp.MustCompile(`^[a-zA-Z0-9-]{8,128}$`)

// NewIndexNow starts the fake at /indexnow.
func NewIndexNow(t *testing.T) *IndexNow {
	t.Helper()
	n := &IndexNow{Answer: map[int]int{}}
	n.Server = httptest.NewServer(http.HandlerFunc(n.serve))
	t.Cleanup(n.Close)
	return n
}

// Posts returns every batch received, including the refused ones.
func (n *IndexNow) Posts() []IndexNowPost {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]IndexNowPost(nil), n.posts...)
}

func (n *IndexNow) serve(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Host        string   `json:"host"`
		Key         string   `json:"key"`
		KeyLocation string   `json:"keyLocation"`
		URLList     []string `json:"urlList"`
	}
	if r.Method != http.MethodPost || r.URL.Path != "/indexnow" || json.NewDecoder(r.Body).Decode(&p) != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	n.mu.Lock()
	idx := len(n.posts)
	n.posts = append(n.posts, IndexNowPost{p.Host, p.Key, p.KeyLocation, p.URLList})
	forced := n.Answer[idx]
	n.mu.Unlock()
	switch {
	case forced != 0:
		http.Error(w, "forced", forced)
	case p.Host == "" || !keyRE.MatchString(p.Key) || len(p.URLList) == 0 || len(p.URLList) > 10000:
		http.Error(w, "bad request", http.StatusBadRequest)
	case !n.keyFileHolds(p.KeyLocation, p.Key):
		http.Error(w, "key file", http.StatusForbidden)
	case !allOn(p.Host, p.URLList):
		http.Error(w, "url host", http.StatusUnprocessableEntity)
	default:
		fmt.Fprint(w, "")
	}
}

func (n *IndexNow) keyFileHolds(location, key string) bool {
	resp, err := http.Get(location)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode == 200 && strings.TrimSpace(string(body)) == key
}

func allOn(host string, urls []string) bool {
	for _, s := range urls {
		if u, err := url.Parse(s); err != nil || u.Host != host {
			return false
		}
	}
	return true
}
