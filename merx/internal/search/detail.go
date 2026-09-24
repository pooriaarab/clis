// Detail pages: key values by .mets-field-label text, never by generated #g_NN ids.
package search

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Category struct {
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type Notice struct {
	InternalID      string     `json:"internal_id,omitempty"`
	DetailURL       string     `json:"detail_url,omitempty"`
	ReferenceNumber string     `json:"reference_number,omitempty"`
	Buyer           string     `json:"issuing_organization,omitempty"`
	ProjectType     string     `json:"project_type,omitempty"`
	ProjectNumber   string     `json:"project_number,omitempty"`
	Title           string     `json:"title,omitempty"`
	SourceID        string     `json:"source_id,omitempty"`
	AgreementTypes  []string   `json:"agreement_types,omitempty"`
	Location        string     `json:"location,omitempty"`
	JobLocation     string     `json:"job_location,omitempty"`
	PurchaseType    string     `json:"purchase_type,omitempty"`
	Description     string     `json:"description,omitempty"`
	Published       string     `json:"publication_date,omitempty"`
	Closing         string     `json:"closing_date,omitempty"`
	BidIntent       string     `json:"bid_intent,omitempty"`
	ContactName     string     `json:"contact_name,omitempty"`
	ContactPhone    string     `json:"contact_phone,omitempty"`
	ContactEmail    string     `json:"contact_email,omitempty"`
	BidSubmission   string     `json:"bid_submission_type,omitempty"`
	Pricing         string     `json:"pricing,omitempty"`
	MERX            []Category `json:"categories_merx,omitempty"`
	GSIN            []Category `json:"categories_gsin,omitempty"`
	UNSPSC          []Category `json:"categories_unspsc,omitempty"`
	CategoriesPath  string     `json:"-"`
}

func DetailCandidates(id string) []string {
	id = strings.TrimSpace(id)
	switch {
	case id == "":
		return nil
	case strings.Contains(id, "://"):
		return []string{id}
	case strings.Contains(id, "/"):
		return []string{Host + "/" + strings.TrimPrefix(id, "/")}
	case len(id) >= 12:
		return []string{Host + "/public/supplier/interception/view-notice/" + id}
	case strings.HasPrefix(id, "0"):
		return nil
	default:
		return []string{Host + "/public/solicitations/" + id + "/abstract", Host + "/public/supplier/interception/view-notice/" + id}
	}
}

func PickSearchURL(page Page, id string) (string, error) {
	for _, r := range page.Records {
		if r.Solicitation == id || r.InternalID == id {
			return r.DetailURL, nil
		}
	}
	return "", fmt.Errorf("no public notice matches %q", id)
}

func ParseDetail(r io.Reader) (Notice, error) {
	root, err := html.Parse(r)
	if err != nil {
		return Notice{}, err
	}
	var n Notice
	var unlabeled []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if hasClass(node, "mets-field") {
			lab := strings.ToLower(strings.TrimSpace(text(findClass(node, "mets-field-label"))))
			if lab == "" {
				if v := text(findClass(node, "mets-field-body")); v != "" {
					unlabeled = append(unlabeled, v)
				}
				return
			}
			apply(&n, lab, node)
			return
		}
		chrome(node, &n)
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	for _, v := range unlabeled {
		switch {
		case strings.Contains(v, "@"):
			set(&n.ContactEmail, v)
		case phone(v):
			set(&n.ContactPhone, v)
		default:
			set(&n.ContactName, v)
		}
	}
	if n.ReferenceNumber == "" && n.Title == "" {
		return n, fmt.Errorf("no notice fields (expected .mets-field-label)")
	}
	return n, nil
}

func ParseCategories(r io.Reader) (merx, gsin, unspsc []Category, err error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, nil, err
	}
	root, err := html.Parse(strings.NewReader(Unwrap(string(raw))))
	if err != nil {
		return nil, nil, nil, err
	}
	return cats(root, "selectedCategoryContainerMERX"), cats(root, "selectedCategoryContainerGSIN"), cats(root, "selectedCategoryContainerUNSPSC"), nil
}

func apply(n *Notice, lab string, field *html.Node) {
	body := text(findClass(field, "mets-field-body"))
	if lab == "description" {
		if d := find(field, func(x *html.Node) bool { return attr(x, "id") == "descriptionText" }); d != nil {
			body = text(d)
		}
	}
	dest := map[string]*string{
		"reference number": &n.ReferenceNumber, "issuing organization": &n.Buyer,
		"project type": &n.ProjectType, "solicitation type": &n.ProjectType,
		"project number": &n.ProjectNumber, "solicitation number": &n.ProjectNumber,
		"title": &n.Title, "source id": &n.SourceID, "location": &n.Location,
		"job location": &n.JobLocation, "purchase type": &n.PurchaseType,
		"description": &n.Description, "publication": &n.Published, "closing date": &n.Closing,
		"bid intent": &n.BidIntent, "bid submission type": &n.BidSubmission, "pricing": &n.Pricing,
	}
	if lab == "agreement types" {
		each(field, func(x *html.Node) bool { return hasClass(x, "agreementLine") }, func(x *html.Node) {
			if v := text(x); v != "" {
				n.AgreementTypes = append(n.AgreementTypes, v)
			}
		})
		return
	}
	if p := dest[lab]; p != nil {
		set(p, body)
	}
}

func chrome(node *html.Node, n *Notice) {
	if node.Type != html.ElementNode {
		return
	}
	for _, a := range node.Attr {
		set(&n.InternalID, idFromPath(a.Val))
		if a.Key == "data-ajax-url" && strings.HasPrefix(a.Val, "/") && !strings.HasPrefix(a.Val, "//") && strings.Contains(a.Val, "/abstract/categories") {
			set(&n.CategoriesPath, a.Val)
		}
		if a.Key == "rel" && a.Val == "canonical" {
			set(&n.DetailURL, attr(node, "href"))
		}
	}
}

func LoadCategories(get func(string) ([]byte, error), n *Notice) {
	if !strings.HasPrefix(n.CategoriesPath, "/") || strings.HasPrefix(n.CategoriesPath, "//") {
		return
	}
	if raw, err := get(Host + n.CategoriesPath); err == nil {
		n.MERX, n.GSIN, n.UNSPSC, _ = ParseCategories(strings.NewReader(string(raw)))
	}
}

func idFromPath(s string) string {
	for _, p := range []string{"/public/supplier/solicitations/notice/", "/public/solicitations/", "/view-notice/"} {
		if i := strings.Index(s, p); i >= 0 {
			rest := s[i+len(p):]
			j := 0
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if j > 0 {
				return rest[:j]
			}
		}
	}
	return ""
}

// Unwrap pulls the HTML fragment out of a MERX tab's text/javascript
// response. Tabs inject markup with $("#innerTabContent").html('...').
func Unwrap(s string) string {
	const p = `$("#innerTabContent").html('`
	i := strings.Index(s, p)
	if i < 0 {
		return s
	}
	s = s[i+len(p):]
	if j := strings.Index(s, "');"); j >= 0 {
		s = s[:j]
	}
	s = strings.NewReplacer(`\/`, `/`, `\'`, `'`).Replace(s)
	if q, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return q
	}
	return s
}

func cats(root *html.Node, id string) []Category {
	table := find(root, func(n *html.Node) bool { return n.Data == "table" && attr(n, "id") == id })
	var out []Category
	each(table, func(n *html.Node) bool {
		return n.Data == "tr" && hasClass(n, "mets-table-row") && !hasClass(n, "mets-table-row-empty") && !hasClass(n, "loading-children")
	}, func(n *html.Node) {
		code := text(find(n, func(x *html.Node) bool { return x.Data == "span" && strings.HasSuffix(attr(x, "id"), "-code") }))
		name := text(find(n, func(x *html.Node) bool { return x.Data == "span" && hasClass(x, "categoryName") }))
		if code != "" && name != "" {
			out = append(out, Category{Code: code, Name: name})
		}
	})
	return out
}

func each(n *html.Node, pred func(*html.Node) bool, visit func(*html.Node)) {
	if n == nil {
		return
	}
	if pred(n) {
		visit(n)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		each(c, pred, visit)
	}
}

func set(dst *string, v string) {
	if v != "" && *dst == "" {
		*dst = v
	}
}

func phone(s string) bool {
	d := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			d++
		} else if !strings.ContainsRune(" ()-+.", r) {
			return false
		}
	}
	return d >= 7
}

// Doc is one attachment-preview-* link from a docs-items tab.
type Doc struct{ ID, Filename, URL string }

// ParseDocList reads attachment-preview-* anchors from a docs-items tab
// (JS-wrapped or already unwrapped HTML). Other links are ignored.
func ParseDocList(r io.Reader) ([]Doc, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(strings.NewReader(Unwrap(string(raw))))
	if err != nil {
		return nil, err
	}
	var out []Doc
	each(root, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "a" }, func(n *html.Node) {
		href := attr(n, "href")
		i := strings.Index(href, "/docs-items/")
		if i < 0 || !strings.Contains(href, "attachment-preview-") {
			return
		}
		id := href[i+len("/docs-items/"):]
		if j := strings.IndexAny(id, "/?#"); j > 0 {
			id = id[:j]
		}
		name := text(n)
		if name == "" {
			name = id
		}
		out = append(out, Doc{id, filepath.Base(name), href})
	})
	return out, nil
}
