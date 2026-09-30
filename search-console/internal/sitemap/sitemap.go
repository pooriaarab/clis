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
	Problems []string `json:"problems"`
}

// Check fetches rawURL and lints it. It returns an error only when the request
// cannot be made. A bad sitemap comes back as problems in the report.
func Check(ctx context.Context, c *httpx.Client, rawURL string) (*Report, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%q: %w", rawURL, ErrNotAbsolute)
	}
	rep := &Report{URL: rawURL, Problems: []string{}}
	resp, err := c.Do(ctx, http.MethodGet, rawURL, map[string]string{"Accept": "application/xml"}, nil)
	if err != nil {
		return nil, err
	}
	if resp.DryRun {
		rep.OK = true
		return rep, nil
	}
	if resp.Status != http.StatusOK {
		rep.problem("%s answered HTTP %d, want 200", rawURL, resp.Status)
		return rep, nil
	}
	if len(resp.Body) > MaxBytes {
		rep.problem("the sitemap is larger than 50 MB")
	}
	root, locs, err := parse(resp.Body)
	rep.Kind, rep.URLs = root, len(locs)
	switch {
	case err != nil:
		rep.problem("the body is not valid XML: %v", err)
	case root != "urlset" && root != "sitemapindex":
		rep.problem("the root element is <%s>, want urlset or sitemapindex", root)
	case len(locs) == 0:
		rep.problem("the sitemap has no URLs")
	case len(locs) >= MaxURLs:
		rep.problem("the sitemap has %d URLs, the limit is under 50,000", len(locs))
	}
	rep.OK = len(rep.Problems) == 0
	return rep, nil
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
