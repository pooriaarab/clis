// Package search parses MERX public list HTML. There is no JSON API.
// Page size is 25; pageNumber > 40 is clamped to 40. A query can reach
// at most 1,000 results. Call CeilingWarning when Total exceeds that.
package search

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	Host         = "https://www.merx.com"
	PageSize     = 25
	MaxPage      = 40
	MaxReachable = MaxPage * PageSize
)

var lists = map[string]string{
	"open": "/public/solicitations/open", "awarded": "/public/solicitations/awarded",
	"bid-results": "/public/solicitations/bid-results", "closed": "/public/solicitations/closed",
}

var sortByOK = map[string]bool{
	"score": true, "noticeTitle": true, "region": true,
	"publicationDate": true, "closingDate": true,
}

var publishOK = map[string]bool{"HOURS_24": true, "WEEK_1": true}

// Query is one GET against a public list. Status is the path, not a param.
type Query struct {
	Status, Keywords, Category, Location string
	PublishDate, SortBy, SortDirection   string
	Page                                 int
}

// Record is one <tr class="mets-table-row">. Notice rows have no
// public solicitation number, portal prefix, or slug.
type Record struct {
	InternalID    string `json:"internal_id"`
	Solicitation  string `json:"solicitation_number"`
	PortalPrefix  string `json:"portal_prefix"`
	Slug          string `json:"slug"`
	DetailURL     string `json:"detail_url"`
	Title         string `json:"title"`
	Buyer         string `json:"buyer"`
	Location      string `json:"location"`
	Published     string `json:"published"`
	Closing       string `json:"closing"`
	DaysRemaining string `json:"days_remaining"`
	// Description is set for an authenticated row. omitempty leaves public JSON unchanged.
	Description     string   `json:"description,omitempty"`
	ReferenceNumber string   `json:"reference_number,omitempty"`
	ContactName     string   `json:"contact_name,omitempty"`
	ContactPhone    string   `json:"contact_phone,omitempty"`
	ContactEmail    string   `json:"contact_email,omitempty"`
	AgreementTypes  []string `json:"agreement_types,omitempty"`
	// DetailFetched is false when the detail page was not requested.
	// An empty reference number then does not mean the notice has none.
	DetailFetched bool `json:"detail_fetched,omitempty"`
}

// Page is the reported total plus this page's rows.
type Page struct {
	Total   int      `json:"total"`
	Records []Record `json:"records"`
}

func (q Query) URL() (string, error) {
	path, ok := lists[q.Status]
	if !ok {
		return "", fmt.Errorf("unknown --status %q (open, awarded, bid-results, closed)", q.Status)
	}
	if q.Page < 1 || q.Page > MaxPage {
		return "", fmt.Errorf("--page %d is outside 1-%d; the server clamps higher pages to %d and would hide the 1,000-record ceiling", q.Page, MaxPage, MaxPage)
	}
	if q.PublishDate != "" && !publishOK[q.PublishDate] {
		return "", fmt.Errorf("unknown --publish-date %q (HOURS_24, WEEK_1)", q.PublishDate)
	}
	if q.SortBy != "" && !sortByOK[q.SortBy] {
		return "", fmt.Errorf("unknown --sort-by %q (score, noticeTitle, region, publicationDate, closingDate)", q.SortBy)
	}
	dir := strings.ToUpper(q.SortDirection)
	if dir != "" && dir != "ASC" && dir != "DESC" {
		return "", fmt.Errorf("unknown --sort-direction %q (ASC, DESC)", q.SortDirection)
	}
	u, err := url.Parse(Host + path)
	if err != nil {
		return "", err
	}
	v := url.Values{}
	for _, kv := range [][2]string{
		{"keywords", q.Keywords}, {"category", q.Category}, {"location", q.Location},
		{"publishDate", q.PublishDate}, {"sortBy", q.SortBy}, {"sortDirection", dir},
	} {
		if kv[1] != "" {
			v.Set(kv[0], kv[1])
		}
	}
	v.Set("pageNumber", strconv.Itoa(q.Page))
	u.RawQuery = v.Encode()
	return u.String(), nil
}

// CeilingWarning names both the reported total and the 1,000-record reach limit.
func CeilingWarning(total int) string {
	if total <= MaxReachable {
		return ""
	}
	return fmt.Sprintf("warning: query reports %d results; MERX only serves the first %d (pages 1-%d, %d per page)", total, MaxReachable, MaxPage, PageSize)
}

// Parse reads list HTML. Anchors are class names, not generated ids like #g_10.
func Parse(r io.Reader) (Page, error) {
	root, err := html.Parse(r)
	if err != nil {
		return Page{}, err
	}
	var out Page
	var foundTotal bool
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if hasClass(n, "simpleSolResultsNumResults") {
			n, err := digits(text(n))
			if err != nil {
				return err
			}
			out.Total, foundTotal = n, true
		}
		if n.Data == "tr" && hasClass(n, "mets-table-row") {
			rec, err := parseRow(n)
			if err != nil {
				return err
			}
			out.Records = append(out.Records, rec)
			return nil
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return Page{}, err
	}
	if !foundTotal {
		return Page{}, fmt.Errorf("missing span.simpleSolResultsNumResults")
	}
	return out, nil
}

func parseRow(n *html.Node) (Record, error) {
	a := find(n, func(x *html.Node) bool { return x.Data == "a" && hasClass(x, "solicitation-link") })
	if a == nil {
		return Record{}, fmt.Errorf("mets-table-row without a.solicitation-link")
	}
	href := attr(a, "href")
	if href == "" {
		return Record{}, fmt.Errorf("solicitation-link missing href")
	}
	id, err := internalID(attr(a, "id"))
	if err != nil {
		return Record{}, err
	}
	portal, slug, number, detail := parseHref(href)
	return Record{
		InternalID: id, Solicitation: number, PortalPrefix: portal, Slug: slug, DetailURL: detail,
		Title: text(findClass(n, "rowTitle")), Buyer: text(findClass(n, "buyer-name")),
		Location: text(findClass(n, "location")), Published: dateValue(n, "publicationDate"),
		Closing: dateValue(n, "closingDate"), DaysRemaining: text(findClass(n, "timeRemaining")),
	}, nil
}

func internalID(id string) (string, error) {
	for _, p := range []string{"searchResultSol_solicitation_", "searchResultSol_notice_"} {
		if s := strings.TrimPrefix(id, p); s != id && s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("unrecognized row id %q", id)
}

// parseHref reads portal/slug/number only from /solicitations/open-bids/<slug>/<id>.
func parseHref(href string) (portal, slug, number, detail string) {
	u, err := url.Parse(href)
	if err != nil {
		return "", "", "", Host + href
	}
	detail = Host + u.Path
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts); i++ {
		if parts[i] != "solicitations" || i+3 >= len(parts) || parts[i+1] != "open-bids" {
			continue
		}
		if i > 0 && parts[i-1] != "public" {
			portal = parts[i-1]
		}
		return portal, parts[i+2], parts[i+3], detail
	}
	return "", "", "", detail
}

func dateValue(n *html.Node, wrap string) string {
	if box := findClass(n, wrap); box != nil {
		return text(findClass(box, "dateValue"))
	}
	return ""
}

func digits(s string) (int, error) {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return 0, fmt.Errorf("no digits in results count %q", s)
	}
	return strconv.Atoi(b.String())
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, name string) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == name {
			return true
		}
	}
	return false
}

func findClass(n *html.Node, name string) *html.Node {
	return find(n, func(x *html.Node) bool { return hasClass(x, name) })
}

func find(n *html.Node, ok func(*html.Node) bool) *html.Node {
	if n == nil {
		return nil
	}
	if ok(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := find(c, ok); found != nil {
			return found
		}
	}
	return nil
}

func text(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(stripControl(b.String())), " ")
}

// stripControl replaces C0 control bytes (e.g. ANSI/OSC escape sequences a
// buyer could embed in a title) with spaces before text reaches a terminal.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}
