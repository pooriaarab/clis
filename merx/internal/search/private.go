package search

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// ParsePrivate reads an authenticated solicitation results page.
// It rejects the public list shape. A row without a.solicitationsTitleLink
// is an error, so a parser break cannot become an empty record.
func ParsePrivate(r io.Reader) (Page, error) {
	root, err := html.Parse(r)
	if err != nil {
		return Page{}, err
	}
	var out Page
	var foundTotal bool
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if hasClass(n, "mets-total-elements-display") {
			total, err := totalAfterOf(text(n))
			if err != nil {
				return err
			}
			out.Total, foundTotal = total, true
		}
		if n.Data == "tr" && hasClass(n, "mets-table-row") {
			rec, err := parsePrivateRow(n)
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
		return Page{}, fmt.Errorf("missing span.mets-total-elements-display")
	}
	return out, nil
}

func parsePrivateRow(n *html.Node) (Record, error) {
	a := find(n, func(x *html.Node) bool {
		return x.Data == "a" && hasClass(x, "solicitationsTitleLink")
	})
	if a == nil {
		return Record{}, fmt.Errorf("mets-table-row without a.solicitationsTitleLink")
	}
	id, detail, err := privateIDAndURL(attr(a, "href"))
	if err != nil {
		return Record{}, err
	}
	return Record{
		InternalID:  id,
		DetailURL:   detail,
		Title:       visibleText(a),
		Buyer:       visibleText(findClass(n, "buyerIdentification")),
		Location:    visibleText(findClass(n, "regionValue")),
		Published:   firstSlashDate(visibleText(findClass(n, "publicationDate"))),
		Closing:     visibleText(findClass(n, "dateValue")),
		Description: visibleText(findClass(n, "solicitationDescription")),
	}, nil
}

// privateIDAndURL takes the internal id from the href path, not the anchor id.
// The authenticated href is /private/supplier/interception/{open-solicitation|view-notice}/{id}.
func privateIDAndURL(href string) (string, string, error) {
	if strings.TrimSpace(href) == "" {
		return "", "", fmt.Errorf("solicitationsTitleLink missing href")
	}
	u, err := url.Parse(href)
	if err != nil {
		return "", "", fmt.Errorf("solicitationsTitleLink bad href: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	id := parts[len(parts)-1]
	if !allDigits(id) {
		return "", "", fmt.Errorf("solicitationsTitleLink href has no internal id")
	}
	path := u.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return id, Host + path, nil
}

func totalAfterOf(s string) (int, error) {
	fields := strings.Fields(s)
	for i, f := range fields {
		if !strings.EqualFold(f, "of") || i+1 >= len(fields) {
			continue
		}
		n, err := strconv.Atoi(strings.ReplaceAll(fields[i+1], ",", ""))
		if err != nil {
			break
		}
		return n, nil
	}
	return 0, fmt.Errorf("no total in results count %q", strings.TrimSpace(s))
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func firstSlashDate(s string) string {
	for _, f := range strings.Fields(s) {
		if isSlashDate(f) {
			return f
		}
	}
	return ""
}

func isSlashDate(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !allDigits(p) {
			return false
		}
	}
	return true
}

// visibleText skips script, style, and accessibility-hidden nodes.
// The title link appends "(opens in a new window)" in that hidden span.
func visibleText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || hasClass(n, "accessibility-hidden")) {
			return
		}
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
