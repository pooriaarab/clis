package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/html"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
	"merx-cli/internal/session"
)

const privateSearchPath = "/private/supplier/solicitations/search"

const maxSearchRedirects = 10

var privateStatus = map[string]string{
	"open": "OPEN", "closed": "CLOSED", "awarded": "AWARD", "bid-results": "AWARD",
}

type harvestDeps struct {
	dir, status, since, until string
	slices                    []Slice
	count                     func(Slice) (int, error)
	page                      func(Slice, int) (search.Page, error)
	detail                    func(string) error
	resume                    bool
}

func executePlan(d harvestDeps) (Summary, error) {
	sum := Summary{Status: d.status}
	store, manifest := harvestPaths(d.dir, d.status)
	if !d.resume {
		os.Remove(store)
		os.Remove(manifest)
	}
	known, err := loadIDs(store)
	if err != nil {
		return sum, err
	}
	man, err := loadManifest(manifest)
	if err != nil {
		return sum, err
	}
	if man.Since == "" {
		man.Since = d.since
	}
	if man.Until == "" {
		man.Until = d.until
	}
	done := map[string]bool{}
	for _, e := range man.Slices {
		done[e.Start+"\x00"+e.End] = true
	}
	leaves, totals, err := plan(d.slices, d.count, done)
	if err != nil {
		return sum, err
	}
	if err := os.MkdirAll(filepath.Dir(store), 0o755); err != nil {
		return sum, err
	}
	f, err := os.OpenFile(store, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return sum, err
	}
	defer f.Close()
	// Append the record, then checkpoint. Reverse order loses rows on crash.
	for _, leaf := range leaves {
		rep := totals[key(leaf)]
		pages := (rep + search.PageSize - 1) / search.PageSize
		if pages > search.MaxPage {
			pages = search.MaxPage
		}
		got := 0
		failed := false
		for p := 1; p <= pages; p++ {
			pg, err := d.page(leaf, p)
			if err != nil {
				return sum, err
			}
			for _, r := range pg.Records {
				if known[r.InternalID] {
					continue
				}
				if err := d.detail(r.DetailURL); err != nil {
					fmt.Fprintf(os.Stderr, "detail %s: %v\n", r.InternalID, err)
					failed = true
					continue
				}
				if err := appendRecord(f, r); err != nil {
					return sum, err
				}
				known[r.InternalID] = true
				got++
			}
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s page %d/%d\n", d.status, leaf.Start, leaf.End, p, pages)
		}
		e := Entry{Status: d.status, Start: leaf.Start, End: leaf.End, Reported: rep, Captured: got,
			CompletedAt: time.Now().UTC().Format(time.RFC3339), Incomplete: incomplete(leaf, rep)}
		sum.Slices++
		sum.Reported += rep
		sum.Captured += got
		if e.Incomplete {
			sum.Incomplete++
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s incomplete: reported %d, captured %d\n", d.status, leaf.Start, leaf.End, rep, got)
		}
		if failed {
			// A detail fetch failed for at least one record: leave the leaf
			// out of the manifest so a later --resume retries the records
			// that were not stored, instead of marking the slice done with
			// a permanent gap.
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s not recorded: a detail fetch failed, rerun with --resume to retry\n", d.status, leaf.Start, leaf.End)
			continue
		}
		man.Slices = append(man.Slices, e)
		if err := saveManifest(manifest, man); err != nil {
			return sum, err
		}
	}
	return sum, nil
}

func loadSearchForm(c *httpx.Client, portal string) (url.Values, string, error) {
	req, err := httpx.NewRequest(http.MethodGet, strings.TrimRight(portal, "/")+privateSearchPath)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("search page: got HTTP %d", resp.StatusCode)
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, "", err
	}
	// Four forms share the search action. The one that holds the search
	// fields has no _csrf. Match the fields, then take the token from the page.
	fields, ok := searchFormFields(doc)
	if !ok {
		return nil, "", fmt.Errorf("search page has no form with fields status and publishedDate.dateType")
	}
	token, header, err := pageCSRF(doc)
	if err != nil {
		return nil, "", err
	}
	v := make(url.Values, len(fields)+1)
	for k, vs := range fields {
		v[k] = append([]string(nil), vs...)
	}
	v.Set("_csrf", token)
	return v, header, nil
}

func postSlice(c *httpx.Client, portal string, form url.Values, csrfHeader, priv string, s Slice, page int) (search.Page, error) {
	v := make(url.Values, len(form))
	for k, vs := range form {
		v[k] = append([]string(nil), vs...)
	}
	v.Set("status", priv)
	v.Set("publishedDate.dateType", "RANGE")
	v.Set("publishedDate.localRangeStart", s.Start)
	v.Set("publishedDate.localRangeEnd", s.End)
	v.Set("pageNumber", strconv.Itoa(page))
	v.Set("pageSize", strconv.Itoa(search.PageSize))
	enc := v.Encode()
	req, err := httpx.NewRequest(http.MethodPost, strings.TrimRight(portal, "/")+privateSearchPath)
	if err != nil {
		return search.Page{}, err
	}
	req.ContentLength = int64(len(enc))
	req.Body = io.NopCloser(strings.NewReader(enc))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(enc)), nil }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if v.Get("_csrf") == "" {
		return search.Page{}, fmt.Errorf("refusing to post search: _csrf token is empty")
	}
	if csrfHeader != "" {
		req.Header.Set(csrfHeader, v.Get("_csrf"))
	}
	body, err := followSearch(c, req)
	if err != nil {
		return search.Page{}, fmt.Errorf("slice %s..%s page %d: %w", s.Start, s.End, page, err)
	}
	return search.Parse(strings.NewReader(string(body)))
}

// followSearch walks Post/Redirect/Get. 301, 302 and 303 become GET. 307 and 308 keep the method. A login URL is a session error.
func followSearch(c *httpx.Client, req *http.Request) ([]byte, error) {
	cur := req.URL.String()
	for sent := 1; ; sent++ {
		resp, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.Request != nil && resp.Request.URL != nil {
			cur = resp.Request.URL.String()
		}
		if loginLanding(cur) {
			return nil, fmt.Errorf("session expired: landed on login page %s", cur)
		}
		code := resp.StatusCode
		if !searchRedirect(code) {
			if code != http.StatusOK {
				return nil, fmt.Errorf("got HTTP %d", code)
			}
			return body, nil
		}
		if sent >= maxSearchRedirects {
			return nil, fmt.Errorf("too many redirects (max %d)", maxSearchRedirects)
		}
		loc := resp.Header.Get("Location")
		if loc == "" {
			return nil, fmt.Errorf("redirect with empty Location (HTTP %d)", code)
		}
		next, err := resolveRef(cur, loc)
		if err != nil {
			return nil, err
		}
		method := http.MethodGet
		if code == http.StatusTemporaryRedirect || code == http.StatusPermanentRedirect {
			method = req.Method
		}
		prev := req
		req, err = httpx.NewRequest(method, next)
		if err != nil {
			return nil, err
		}
		if method != http.MethodGet && method != http.MethodHead {
			if prev.GetBody == nil {
				return nil, fmt.Errorf("HTTP %d redirect cannot replay the body", code)
			}
			b, berr := prev.GetBody()
			if berr != nil {
				return nil, berr
			}
			req.Body, req.GetBody, req.ContentLength = b, prev.GetBody, prev.ContentLength
			req.Header = prev.Header.Clone()
			req.Header.Del("Content-Length")
			req.Header.Del("Cookie")
		}
		cur = next
	}
}

func searchRedirect(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func resolveRef(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(r).String(), nil
}

func loginLanding(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.Contains(u.Path, "/authentication/login") || strings.Contains(u.Path, "/saml/login")
}

func fetchDetail(c *httpx.Client, detailURL string) error {
	if detailURL == "" {
		return fmt.Errorf("empty detail URL")
	}
	req, err := httpx.NewRequest(http.MethodGet, detailURL)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, copyErr := io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", detailURL, resp.StatusCode)
	}
	return copyErr
}

var exitFunc = os.Exit

// logoutOnce runs Logout at most once so the signal handler and runHarvest's
// deferred cleanup cannot both fire it for the same session.
func logoutOnce(once *sync.Once, c *httpx.Client, portal string) {
	once.Do(func() {
		if err := session.Logout(c, portal); err != nil {
			fmt.Fprintln(os.Stderr, "logout:", err)
		}
	})
}

// installLogout logs out on SIGINT/SIGTERM. Caller must stop() on success or a
// later signal still overrides the exit code. #96 is unverified.
func installLogout(once *sync.Once, c *httpx.Client, portal string) func() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		fmt.Fprintln(os.Stderr, "interrupted: logging out")
		logoutOnce(once, c, portal)
		exitFunc(1)
	}()
	return func() { signal.Stop(ch) }
}

func runHarvest(status, since, until string, resume bool, seeds []Slice) error {
	priv := privateStatus[status]
	j, c, jarPath, err := openSession()
	if err != nil {
		return err
	}
	valid := false
	if _, err := os.Stat(jarPath); err == nil {
		valid, _ = session.Valid(c, session.Production.Portal)
	}
	if !valid {
		user, pass := os.Getenv("MERX_USERNAME"), os.Getenv("MERX_PASSWORD")
		if user == "" || pass == "" {
			return fmt.Errorf("missing credentials: export MERX_USERNAME and MERX_PASSWORD")
		}
		if err := session.Login(c, session.Production, user, pass); err != nil {
			return err
		}
	}
	// A lost jar cannot be logged out. Write it before the first fetch.
	if err := j.Save(); err != nil {
		return err
	}
	portal := session.Production.Portal
	var once sync.Once
	stop := installLogout(&once, c, portal)
	defer func() {
		stop()
		logoutOnce(&once, c, portal)
		os.Remove(jarPath)
	}()
	form, csrfHeader, err := loadSearchForm(c, portal)
	if err != nil {
		return err
	}
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	sum, err := executePlan(harvestDeps{dir: dir, status: status, since: since, until: until, slices: seeds, resume: resume,
		count: func(s Slice) (int, error) {
			pg, err := postSlice(c, portal, form, csrfHeader, priv, s, 1)
			return pg.Total, err
		},
		page:   func(s Slice, p int) (search.Page, error) { return postSlice(c, portal, form, csrfHeader, priv, s, p) },
		detail: func(u string) error { return fetchDetail(c, u) }})
	if err != nil {
		return err
	}
	if flagJSON {
		return emit(sum)
	}
	fmt.Printf("%s: %d slices, reported %d, captured %d, incomplete %d\n", sum.Status, sum.Slices, sum.Reported, sum.Captured, sum.Incomplete)
	return nil
}

// pageCSRF reads the session token from <meta name="_csrf">, then from any
// _csrf input. The header name is <meta name="_csrf_header">.
func pageCSRF(doc *html.Node) (string, string, error) {
	token := tagAttr(doc, "meta", "name", "_csrf", "content")
	if token == "" {
		token = tagAttr(doc, "input", "name", "_csrf", "value")
	}
	if token == "" {
		return "", "", fmt.Errorf("search page has no CSRF token: missing <meta name=\"_csrf\"> and no _csrf input")
	}
	return token, tagAttr(doc, "meta", "name", "_csrf_header", "content"), nil
}

// searchFormFields returns the successful controls of the form that contains
// status and publishedDate.dateType. An unchecked box still identifies the
// form. A browser omits that box from the body. The form id is not used.
func searchFormFields(doc *html.Node) (url.Values, bool) {
	var fields url.Values
	found := walkHTML(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "form" {
			return false
		}
		if !controlNamed(n, "status") || !controlNamed(n, "publishedDate.dateType") {
			return false
		}
		fields = formControls(n)
		return true
	})
	return fields, found
}

// formControls serializes one form the way a browser builds the POST body.
func formControls(form *html.Node) url.Values {
	got := make(url.Values)
	walkForm(form, func(n *html.Node) {
		name, vals, ok := successfulControl(n)
		if !ok {
			return
		}
		for _, val := range vals {
			got.Add(name, val)
		}
	})
	return got
}

func controlNamed(form *html.Node, name string) bool {
	found := false
	walkForm(form, func(n *html.Node) {
		if found || n.Type != html.ElementNode {
			return
		}
		if n.Data != "input" && n.Data != "select" && n.Data != "textarea" {
			return
		}
		if nodeAttr(n, "name") == name {
			found = true
		}
	})
	return found
}

func walkForm(form *html.Node, visit func(*html.Node)) {
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, root bool) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "form" && !root {
			return
		}
		visit(n)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, false)
		}
	}
	walk(form, true)
}

// successfulControl reports one control a browser would submit.
// Unchecked radios and checkboxes are omitted. A submit button is omitted
// because this POST is not a click on that button. Repeated names stay
// repeated: the caller adds every returned value.
func successfulControl(n *html.Node) (string, []string, bool) {
	if n.Type != html.ElementNode {
		return "", nil, false
	}
	switch n.Data {
	case "input", "select", "textarea":
	default:
		return "", nil, false
	}
	if _, off := attrPresent(n, "disabled"); off {
		return "", nil, false
	}
	name := nodeAttr(n, "name")
	if name == "" {
		return "", nil, false
	}
	switch n.Data {
	case "textarea":
		return name, []string{textareaValue(n)}, true
	case "select":
		vals, ok := selectValues(n)
		if !ok {
			return "", nil, false
		}
		return name, vals, true
	default:
		switch strings.ToLower(nodeAttr(n, "type")) {
		case "radio", "checkbox":
			if _, on := attrPresent(n, "checked"); !on {
				return "", nil, false
			}
		case "button", "submit", "reset", "image":
			return "", nil, false
		}
		return name, []string{inputValue(n)}, true
	}
}

func inputValue(n *html.Node) string {
	if v, ok := attrPresent(n, "value"); ok {
		return v
	}
	switch strings.ToLower(nodeAttr(n, "type")) {
	case "checkbox", "radio":
		return "on"
	default:
		return ""
	}
}

func selectValues(n *html.Node) ([]string, bool) {
	_, multiple := attrPresent(n, "multiple")
	var first string
	var chosen []string
	seen := false
	var walk func(*html.Node)
	walk = func(parent *html.Node) {
		for c := parent.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			switch c.Data {
			case "optgroup":
				walk(c)
			case "option":
				if _, off := attrPresent(c, "disabled"); off {
					continue
				}
				val := optionValue(c)
				if !seen {
					first, seen = val, true
				}
				if _, on := attrPresent(c, "selected"); on {
					chosen = append(chosen, val)
				}
			}
		}
	}
	walk(n)
	if !seen {
		return nil, false
	}
	if len(chosen) == 0 {
		return []string{first}, true
	}
	if !multiple && len(chosen) > 1 {
		return chosen[len(chosen)-1:], true
	}
	return chosen, true
}

func optionValue(o *html.Node) string {
	if v, ok := attrPresent(o, "value"); ok {
		return v
	}
	return directText(o)
}

func textareaValue(n *html.Node) string {
	return strings.TrimPrefix(rawText(n), "\n")
}

func directText(n *html.Node) string {
	return strings.TrimSpace(rawText(n))
}

func rawText(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

func tagAttr(root *html.Node, tag, key, want, out string) string {
	var v string
	walkHTML(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != tag || nodeAttr(n, key) != want {
			return false
		}
		got := strings.TrimSpace(nodeAttr(n, out))
		if got == "" {
			return false
		}
		v = got
		return true
	})
	return v
}

func nodeAttr(n *html.Node, key string) string {
	v, _ := attrPresent(n, key)
	return v
}

func attrPresent(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// walkHTML visits n, then its children. It stops when visit returns true.
func walkHTML(n *html.Node, visit func(*html.Node) bool) bool {
	if n == nil {
		return false
	}
	if visit(n) {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if walkHTML(c, visit) {
			return true
		}
	}
	return false
}
