package fakes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Page is one response the fake site serves.
type Page struct {
	Status      int
	ContentType string
	Body        string
}

// Site mimics the website that serves sitemap.xml and the IndexNow key file.
type Site struct {
	*httptest.Server
	mu    sync.Mutex
	pages map[string]Page
	hits  map[string]int
}

// NewSite starts a site with no pages. Unknown paths answer 404.
func NewSite(t *testing.T) *Site {
	t.Helper()
	s := &Site{pages: map[string]Page{}, hits: map[string]int{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		p, ok := s.pages[r.URL.Path]
		s.hits[r.URL.Path]++
		s.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", p.ContentType)
		w.WriteHeader(p.Status)
		fmt.Fprint(w, p.Body)
	}))
	t.Cleanup(s.Close)
	return s
}

// Serve sets the response for a path.
func (s *Site) Serve(path string, status int, contentType, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pages[path] = Page{status, contentType, body}
}

// Hits counts requests for a path.
func (s *Site) Hits(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

// URLSet builds a sitemap body for the given URLs.
func URLSet(urls ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		fmt.Fprintf(&b, "<url><loc>%s</loc></url>", u)
	}
	b.WriteString("</urlset>")
	return b.String()
}
