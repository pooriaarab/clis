// Package sitemap fetches a sitemap and lints it the way Google and Bing would.
package sitemap

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// Limits from the sitemap protocol.
const (
	MaxURLs  = 50000
	MaxBytes = 50 << 20
)

// ErrNotAbsolute means the sitemap URL is not an absolute http(s) URL.
var ErrNotAbsolute = errors.New("sitemap URL is not absolute: use https://host/path")

// Report is the result of one check. An empty Problems list means the sitemap is fine.
type Report struct {
	OK       bool     `json:"ok"`
	URL      string   `json:"url"`
	Kind     string   `json:"kind,omitempty"`
	URLs     int      `json:"urls"`
	Sitemaps int      `json:"sitemaps,omitempty"`
	Problems []string `json:"problems"`
}

// Check fetches rawURL and lints it. A sitemap index is followed one level. It
// returns an error only when a request cannot be made. A bad sitemap comes back
// as problems in the report.
func Check(ctx context.Context, c *httpx.Client, rawURL string) (*Report, error) {
	if _, err := absolute(rawURL); err != nil {
		return nil, fmt.Errorf("%q: %w", rawURL, ErrNotAbsolute)
	}
	rep := &Report{URL: rawURL, Problems: []string{}}
	root, locs, err := rep.fetch(ctx, c, rawURL, "")
	if err != nil {
		return nil, err
	}
	rep.checkLocs("", rawURL, locs)
	rep.Kind, rep.URLs = root, len(locs)
	if root == "sitemapindex" {
		rep.URLs = 0
		for _, child := range locs {
			if _, err := absolute(child); err != nil {
				continue // already reported as a bad <loc>
			}
			croot, clocs, err := rep.fetch(ctx, c, child, "child "+child+": ")
			if err != nil {
				return nil, err
			}
			rep.checkLocs("child "+child+": ", child, clocs)
			if croot == "sitemapindex" {
				rep.problem("child %s: an index inside an index is not allowed", child)
			}
			rep.Sitemaps++
			rep.URLs += len(clocs)
		}
	}
	rep.OK = len(rep.Problems) == 0 || c.DryRun
	return rep, nil
}

// Locs returns every page URL of a sitemap, and follows an index one level. It
// does not lint the hosts. It fails when a sitemap cannot be read.
func Locs(ctx context.Context, c *httpx.Client, rawURL string) ([]string, error) {
	if _, err := absolute(rawURL); err != nil {
		return nil, fmt.Errorf("%q: %w", rawURL, ErrNotAbsolute)
	}
	rep := &Report{}
	root, locs, err := rep.fetch(ctx, c, rawURL, "")
	if err == nil && root == "sitemapindex" {
		var pages []string
		for _, child := range locs {
			_, clocs, cerr := rep.fetch(ctx, c, child, "child "+child+": ")
			if cerr != nil {
				return nil, cerr
			}
			pages = append(pages, clocs...)
		}
		locs = pages
	}
	if err != nil {
		return nil, err
	}
	if len(rep.Problems) > 0 {
		return nil, fmt.Errorf("cannot read the sitemap: %s", strings.Join(rep.Problems, "; "))
	}
	return locs, nil
}

// absolute parses raw and requires an http(s) URL with a host.
func absolute(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrNotAbsolute
	}
	return u, nil
}

// fetch downloads one sitemap and records its problems. prefix names a child.
func (r *Report) fetch(ctx context.Context, c *httpx.Client, rawURL, prefix string) (string, []string, error) {
	resp, err := c.Do(ctx, http.MethodGet, rawURL, map[string]string{"Accept": "application/xml"}, nil)
	if err != nil {
		return "", nil, err
	}
	if resp.DryRun {
		return "", nil, nil
	}
	if resp.Status != http.StatusOK {
		r.problem("%s%s answered HTTP %d, want 200", prefix, rawURL, resp.Status)
		return "", nil, nil
	}
	if len(resp.Body) > MaxBytes {
		r.problem("%sthe sitemap is larger than 50 MB", prefix)
	}
	root, locs, err := parse(resp.Body)
	switch {
	case err != nil:
		r.problem("%sthe body is not valid XML: %v", prefix, err)
	case root != "urlset" && root != "sitemapindex":
		r.problem("%sthe root element is <%s>, want urlset or sitemapindex", prefix, root)
	case len(locs) == 0:
		r.problem("%sthe sitemap has no URLs", prefix)
	case len(locs) >= MaxURLs:
		r.problem("%sthe sitemap has %d URLs, the limit is under 50,000", prefix, len(locs))
	}
	return root, locs, nil
}

// checkLocs flags URLs that are relative or not on the sitemap host. It names
// up to three examples so the report stays short.
func (r *Report) checkLocs(prefix, sitemapURL string, locs []string) {
	base, _ := url.Parse(sitemapURL)
	var relative, foreign []string
	for _, loc := range locs {
		u, err := absolute(loc)
		switch {
		case err != nil:
			relative = append(relative, loc)
		case !strings.EqualFold(u.Host, base.Host):
			foreign = append(foreign, loc)
		}
	}
	for _, g := range []struct {
		bad  []string
		what string
	}{
		{relative, "not an absolute http(s) URL"},
		{foreign, "not on the sitemap host " + base.Host},
	} {
		if len(g.bad) == 0 {
			continue
		}
		shown := g.bad[:min(3, len(g.bad))]
		r.problem("%s%d <loc> values are %s, for example %s", prefix, len(g.bad), g.what, strings.Join(shown, ", "))
	}
}

func (r *Report) problem(format string, a ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, a...))
}

// parse returns the root element name and every <loc> value.
func parse(body []byte) (root string, locs []string, err error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return root, locs, nil
		}
		if err != nil {
			return root, locs, err
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if root == "" {
			root = el.Name.Local
		}
		if el.Name.Local == "loc" {
			var s string
			if err := dec.DecodeElement(&s, &el); err != nil {
				return root, locs, err
			}
			locs = append(locs, strings.TrimSpace(s))
		}
	}
}
