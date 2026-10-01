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
	// MaxChildren caps the sitemaps that one index may list. The protocol allows
	// 50,000, but a CLI that fetches that many children is easy to point at a target.
	MaxChildren = 1000
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
	if _, err := rep.crawl(ctx, c, rawURL, false); err != nil {
		return nil, err
	}
	rep.OK = len(rep.Problems) == 0
	return rep, nil
}

// Locs returns every page URL of a sitemap, and follows an index one level. It
// does not lint the page hosts. It fails when a sitemap cannot be read.
func Locs(ctx context.Context, c *httpx.Client, rawURL string) ([]string, error) {
	if _, err := absolute(rawURL); err != nil {
		return nil, fmt.Errorf("%q: %w", rawURL, ErrNotAbsolute)
	}
	rep := &Report{}
	pages, err := rep.crawl(ctx, c, rawURL, true)
	if err != nil {
		return nil, err
	}
	if len(rep.Problems) > 0 {
		return nil, fmt.Errorf("cannot read the sitemap: %s", strings.Join(rep.Problems, "; "))
	}
	return pages, nil
}

// crawl fetches rawURL and fills Kind, URLs and Sitemaps. Every child of an
// index must be on the host of the index, there are at most MaxChildren, and no
// child may be an index. A child that breaks a rule is not fetched, or its URLs
// are not counted. When collect is true, crawl returns the page URLs. When it is
// false, crawl lints the page URLs and returns none, so a big index stays small
// in memory.
func (r *Report) crawl(ctx context.Context, c *httpx.Client, rawURL string, collect bool) ([]string, error) {
	root, locs, err := r.fetch(ctx, c, rawURL, "")
	if err != nil {
		return nil, err
	}
	r.Kind, r.URLs = root, len(locs)
	if root != "sitemapindex" {
		if !collect {
			r.checkLocs("", rawURL, locs)
		}
		return locs, nil
	}
	r.URLs = 0
	r.checkLocs("", rawURL, locs) // children must be absolute and on the index host
	if len(locs) > MaxChildren {
		r.problem("the index lists %d sitemaps, the limit is %d", len(locs), MaxChildren)
		return nil, nil
	}
	base, _ := url.Parse(rawURL)
	var pages []string
	for _, child := range locs {
		u, err := absolute(child)
		if err != nil || !strings.EqualFold(u.Host, base.Host) {
			continue // checkLocs reported it, and the CLI must not connect there
		}
		prefix := "child " + child + ": "
		croot, clocs, err := r.fetch(ctx, c, child, prefix)
		if err != nil {
			return nil, err
		}
		r.Sitemaps++
		if croot == "sitemapindex" {
			r.problem("child %s: an index inside an index is not allowed", child)
			continue
		}
		if collect {
			pages = append(pages, clocs...)
		} else {
			r.checkLocs(prefix, child, clocs)
		}
		r.URLs += len(clocs)
	}
	return pages, nil
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
