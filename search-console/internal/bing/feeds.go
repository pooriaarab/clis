package bing

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Feed is one sitemap of a site, as GetFeeds reports it.
type Feed struct {
	URL         string `json:"Url"`
	Type        string `json:"Type"`
	Status      string `json:"Status"`
	LastCrawled string `json:"LastCrawled"`
	Submitted   string `json:"Submitted"`
	URLCount    int    `json:"UrlCount"`
}

// HasError reports whether Bing gave the feed an error status.
func (f Feed) HasError() bool {
	s := strings.ToLower(f.Status)
	return strings.Contains(s, "error") || strings.Contains(s, "fail")
}

// SubmitFeed asks Bing to read a sitemap. Bing reads it later. It reports true
// when Bing already held the sitemap (error 81058), which is success.
func (a *API) SubmitFeed(ctx context.Context, domain, feedURL string) (bool, error) {
	err := a.call(ctx, http.MethodPost, "SubmitFeed", nil, map[string]string{"siteUrl": SiteURL(domain), "feedUrl": feedURL}, nil)
	return IsAlreadyPresent(err), ignorePresent(err)
}

// Feeds lists the sitemaps Bing holds for the site.
func (a *API) Feeds(ctx context.Context, domain string) ([]Feed, error) {
	var feeds []Feed
	err := a.call(ctx, http.MethodGet, "GetFeeds", url.Values{"siteUrl": {SiteURL(domain)}}, nil, &feeds)
	for i := range feeds {
		feeds[i].LastCrawled = readDate(feeds[i].LastCrawled)
		feeds[i].Submitted = readDate(feeds[i].Submitted)
	}
	return feeds, err
}

// Feed finds one sitemap. It returns nil when Bing does not list it.
func (a *API) Feed(ctx context.Context, domain, feedURL string) (*Feed, error) {
	feeds, err := a.Feeds(ctx, domain)
	for i := range feeds {
		if feeds[i].URL == feedURL {
			return &feeds[i], err
		}
	}
	return nil, err
}

var msDate = regexp.MustCompile(`^/Date\((-?\d+)`)

// readDate turns the .NET date "/Date(1700000000000)/" that Bing sends into
// RFC 3339. Bing sends a date before 1970 (1601-01-01 in live replies) for "never";
// that becomes the empty string. Any other text stays as it is.
func readDate(s string) string {
	m := msDate.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	ms, _ := strconv.ParseInt(m[1], 10, 64)
	if ms < 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
