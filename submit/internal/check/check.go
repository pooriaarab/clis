// Package check verifies a directory listing: it fetches the listing URL and
// reports reachability, redirects, and whether the page links to our site
// (follow vs nofollow).
package check

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result is one listing verification.
type Result struct {
	Site         string   `json:"site"`
	SiteURL      string   `json:"site_url"`
	Directory    string   `json:"directory"`
	ListingURL   string   `json:"listing_url"`
	StatusCode   int      `json:"status_code"`
	FinalURL     string   `json:"final_url"`
	Redirects    []string `json:"redirects"`
	LinkFound    bool     `json:"link_found"`
	LinkHref     string   `json:"link_href,omitempty"`
	LinkRel      string   `json:"link_rel"` // follow, nofollow, not-found
	PageNoFollow bool     `json:"page_nofollow"`
	IndexedHint  string   `json:"indexed_hint"`
}

var (
	anchorRe = regexp.MustCompile(`(?i)<a\b[^>]*\bhref\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)[^>]*>`)
	relRe    = regexp.MustCompile(`(?i)\brel\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	robotsRe = regexp.MustCompile(`(?i)<meta\b[^>]*\bname\s*=\s*["']?robots["']?[^>]*>`)
)

// Run fetches listingURL and checks it for a link to siteURL.
func Run(site, siteURL, directory, listingURL string) (Result, error) {
	r := Result{Site: site, SiteURL: siteURL, Directory: directory, ListingURL: listingURL, LinkRel: "not-found"}
	wantHost, err := hostOf(siteURL)
	if err != nil {
		return r, fmt.Errorf("site URL %q: %w", siteURL, err)
	}
	listU, err := url.Parse(listingURL)
	if err != nil || listU.Scheme == "" || listU.Host == "" {
		return r, fmt.Errorf("listing URL %q is not an absolute URL", listingURL)
	}
	client := &http.Client{
		Timeout: 25 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			r.Redirects = append(r.Redirects, req.URL.String())
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", listingURL, nil)
	if err != nil {
		return r, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; submit-cli/1.0; listing-check)")
	resp, err := client.Do(req)
	if err != nil {
		return r, fmt.Errorf("fetch listing %s: %w", listingURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return r, err
	}
	r.StatusCode = resp.StatusCode
	r.FinalURL = resp.Request.URL.String()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		r.IndexedHint = fmt.Sprintf("listing page returns %d; fix the URL before any index check", resp.StatusCode)
		return r, nil
	}
	page := string(body)
	if m := robotsRe.FindString(page); m != "" && strings.Contains(strings.ToLower(m), "nofollow") {
		r.PageNoFollow = true
	}
	for _, a := range anchorRe.FindAllString(page, -1) {
		href := unquote(hrefOf(a))
		if href == "" {
			continue
		}
		h, err := hostOf(resolve(listU, href))
		if err != nil || h != wantHost {
			continue
		}
		r.LinkFound = true
		r.LinkHref = href
		rel := ""
		if m := relRe.FindStringSubmatch(a); m != nil {
			rel = strings.ToLower(unquote(m[1]))
		}
		if strings.Contains(rel, "nofollow") || r.PageNoFollow {
			r.LinkRel = "nofollow"
		} else {
			r.LinkRel = "follow"
		}
		break
	}
	listHost, _ := hostOf(listingURL)
	if r.LinkFound {
		r.IndexedHint = fmt.Sprintf("page is reachable and links (%s) to the site; confirm with: site:%s %s", r.LinkRel, listHost, site)
	} else {
		r.IndexedHint = fmt.Sprintf("page is reachable but no link to %s was found; confirm with: site:%s %s", wantHost, listHost, site)
	}
	return r, nil
}

func hrefOf(anchor string) string {
	m := anchorRe.FindStringSubmatch(anchor)
	if m == nil {
		return ""
	}
	return m[1]
}

func unquote(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

func resolve(base *url.URL, href string) string {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return href
	}
	return base.ResolveReference(u).String()
}

func hostOf(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("not an absolute URL")
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), nil
}
