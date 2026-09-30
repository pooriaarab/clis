package google

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// count is a number that Google sends as a JSON string, like "3".
type count int

func (c *count) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		s = string(b)
	}
	n, err := strconv.Atoi(s)
	*c = count(n)
	return err
}

// SitemapStatus is what Search Console knows about one submitted sitemap.
type SitemapStatus struct {
	Path           string `json:"path"`
	IsPending      bool   `json:"isPending"`
	IsIndex        bool   `json:"isSitemapsIndex"`
	LastSubmitted  string `json:"lastSubmitted,omitempty"`
	LastDownloaded string `json:"lastDownloaded,omitempty"`
	Errors         count  `json:"errors"`
	Warnings       count  `json:"warnings"`
}

// SubmitSitemap tells Search Console to fetch the sitemap.
func (a *API) SubmitSitemap(ctx context.Context, domain, sitemapURL string) error {
	return a.call(ctx, http.MethodPut, feedPath(domain, sitemapURL), nil, nil, nil)
}

// Sitemap reads the status of a submitted sitemap.
func (a *API) Sitemap(ctx context.Context, domain, sitemapURL string) (*SitemapStatus, error) {
	var st SitemapStatus
	if err := a.call(ctx, http.MethodGet, feedPath(domain, sitemapURL), nil, nil, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// WaitSitemap reads the status until Google is done with it or wait runs out.
// A sitemap that is still pending when wait ends is returned as is.
func (a *API) WaitSitemap(ctx context.Context, domain, sitemapURL string, wait, interval time.Duration) (*SitemapStatus, error) {
	deadline := time.Now().Add(wait)
	for {
		st, err := a.Sitemap(ctx, domain, sitemapURL)
		if err != nil || !st.IsPending || time.Now().Add(interval).After(deadline) {
			return st, err
		}
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return st, ctx.Err()
		}
	}
}
