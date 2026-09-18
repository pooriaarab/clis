// Package session is the MERX SAML cookie jar and login flow.
// Credentials are never logged, echoed, or placed in errors or URLs.
// Login posts the password exactly once.
package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"merx-cli/internal/httpx"
)

// Endpoints is one portal + IdP pair. Tests point both at httptest.
type Endpoints struct{ Portal, IDP string }

// Production is the live MERX pair.
var Production = Endpoints{Portal: "https://www.merx.com", IDP: "https://idp.merx.com"}

const (
	JarFile    = "cookies.json" // cookie file under the cache dir
	anonMarker = `"memberType":"Anonymous"`
	badCreds   = "The username or password you entered is incorrect."
	inUse      = "The account provided is currently in use. Only one session is permitted per account."
	maxHops    = 8
)

// Jar persists cookies for www.merx.com and idp.merx.com.
type Jar struct {
	Path  string
	mu    sync.Mutex
	items []storedCookie
}

type storedCookie struct {
	Name, Value, Path, Domain string
	Expires                   time.Time
	Secure, HttpOnly          bool
}

// Open reads an existing jar, or returns an empty one.
func Open(path string) (*Jar, error) {
	j := &Jar{Path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return j, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return j, nil
	}
	if err := json.Unmarshal(raw, &j.items); err != nil {
		return nil, fmt.Errorf("stored session is corrupt: %w", err)
	}
	return j, nil
}

// domainMatch reports whether host is within domain per RFC 6265 §5.1.3:
// an exact match, or a proper subdomain of it.
func domainMatch(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func (j *Jar) SetCookies(u *url.URL, cs []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	reqHost := strings.ToLower(u.Hostname())
	for _, c := range cs {
		domain := strings.ToLower(strings.TrimPrefix(c.Domain, "."))
		if domain == "" {
			domain = reqHost
		} else if !domainMatch(reqHost, domain) {
			// The response can only set cookies for its own host or a
			// parent of it, never an unrelated domain.
			continue
		}
		path := c.Path
		if path == "" {
			path = "/"
		}
		keep := -1
		for i, sc := range j.items {
			if sc.Name == c.Name && sc.Domain == domain && sc.Path == path {
				keep = i
				break
			}
		}
		// Max-Age takes precedence over Expires per RFC 6265 §5.3: when both
		// are present, a stale Expires must not override a live Max-Age.
		var expires time.Time
		expired := false
		switch {
		case c.MaxAge < 0:
			expired = true
		case c.MaxAge > 0:
			expires = now.Add(time.Duration(c.MaxAge) * time.Second)
		default:
			expires = c.Expires
			expired = !expires.IsZero() && !now.Before(expires)
		}
		if expired {
			if keep >= 0 {
				j.items = append(j.items[:keep], j.items[keep+1:]...)
			}
			continue
		}
		sc := storedCookie{c.Name, c.Value, path, domain, expires, c.Secure, c.HttpOnly}
		if keep >= 0 {
			j.items[keep] = sc
		} else {
			j.items = append(j.items, sc)
		}
	}
}

func (j *Jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	host, now := strings.ToLower(u.Hostname()), time.Now()
	var out []*http.Cookie
	for _, sc := range j.items {
		if !sc.Expires.IsZero() && !now.Before(sc.Expires) {
			continue
		}
		if host != sc.Domain && !strings.HasSuffix(host, "."+sc.Domain) {
			continue
		}
		if !strings.HasPrefix(u.Path, sc.Path) || (sc.Secure && u.Scheme != "https") {
			continue
		}
		out = append(out, &http.Cookie{Name: sc.Name, Value: sc.Value, Path: sc.Path, Domain: sc.Domain, Expires: sc.Expires, Secure: sc.Secure, HttpOnly: sc.HttpOnly})
	}
	return out
}

// Save writes the jar at 0600. Chmod covers a pre-existing file.
// Expired cookies are dropped so they don't accumulate in the file forever.
func (j *Jar) Save() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o755); err != nil {
		return err
	}
	now := time.Now()
	live := j.items[:0:0]
	for _, sc := range j.items {
		if !sc.Expires.IsZero() && !now.Before(sc.Expires) {
			continue
		}
		live = append(live, sc)
	}
	j.items = live
	raw, err := json.Marshal(j.items)
	if err != nil {
		return err
	}
	if err := os.WriteFile(j.Path, raw, 0o600); err != nil {
		return err
	}
	return os.Chmod(j.Path, 0o600)
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func walk(n *html.Node, visit func(*html.Node) bool) {
	if visit(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}

// FindForm returns the action and inputs of the first form that has every wanted field.
func FindForm(doc *html.Node, want ...string) (string, map[string]string, bool) {
	var action string
	var fields map[string]string
	found := false
	walk(doc, func(n *html.Node) bool {
		if found || n.Type != html.ElementNode || n.Data != "form" {
			return found
		}
		got := map[string]string{}
		walk(n, func(m *html.Node) bool {
			if m.Type == html.ElementNode && m.Data == "input" {
				if name := attr(m, "name"); name != "" {
					got[name] = attr(m, "value")
				}
			}
			return false
		})
		for _, w := range want {
			if _, ok := got[w]; !ok {
				return false
			}
		}
		action, fields, found = attr(n, "action"), got, true
		return true
	})
	return action, fields, found
}

// ExecutionToken parses the per-attempt token out of the IdP form action.
func ExecutionToken(action string) (string, error) {
	u, err := url.Parse(action)
	if err != nil {
		return "", fmt.Errorf("IdP form action is not a URL: %w", err)
	}
	if t := u.Query().Get("execution"); t != "" {
		return t, nil
	}
	return "", fmt.Errorf("IdP login form has no execution token")
}

// Bind attaches the jar and disables auto-follow so each hop goes through httpx.
func Bind(j *Jar) (*httpx.Client, error) {
	c, err := httpx.New()
	if err != nil {
		return nil, err
	}
	c.HTTP.Jar = j
	c.HTTP.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c, nil
}

func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

func readResp(c *httpx.Client, req *http.Request) (int, string, []byte, error) {
	resp, err := c.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location"), raw, err
}

func do(c *httpx.Client, method, rawurl string, vals url.Values) (int, []byte, error) {
	req, err := httpx.NewRequest(method, rawurl)
	if err != nil {
		return 0, nil, err
	}
	if vals != nil {
		enc := vals.Encode()
		req.ContentLength = int64(len(enc))
		req.Body = io.NopCloser(strings.NewReader(enc))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	st, loc, body, err := readResp(c, req)
	if err != nil {
		return 0, nil, err
	}
	return follow(c, rawurl, loc, st, body)
}

// follow walks 3xx hops. The credential POST commonly answers 302 first;
// stopping there hid every real failure as "got HTTP 302".
func follow(c *httpx.Client, current, loc string, status int, body []byte) (int, []byte, error) {
	for i := 0; status >= 300 && status <= 399 && i < maxHops; i++ {
		if loc == "" {
			return status, body, fmt.Errorf("redirect with empty Location (HTTP %d)", status)
		}
		current = resolveURL(current, loc)
		req, err := httpx.NewRequest(http.MethodGet, current)
		if err != nil {
			return 0, nil, err
		}
		status, loc, body, err = readResp(c, req)
		if err != nil {
			return 0, nil, err
		}
	}
	if status >= 300 && status <= 399 {
		return status, body, fmt.Errorf("too many redirects (HTTP %d)", status)
	}
	return status, body, nil
}

func toValues(m map[string]string) url.Values {
	v := make(url.Values, len(m))
	for k, val := range m {
		v.Set(k, val)
	}
	return v
}

func parseForm(body []byte, want ...string) (string, map[string]string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("parsing HTML: %w", err)
	}
	action, fields, ok := FindForm(doc, want...)
	if !ok || action == "" {
		return "", nil, fmt.Errorf("form not found")
	}
	return action, fields, nil
}

func step(c *httpx.Client, method, rawurl string, vals url.Values, label string) ([]byte, error) {
	st, body, err := do(c, method, rawurl, vals)
	if err != nil {
		return nil, err
	}
	if st < 200 || st >= 300 {
		return body, fmt.Errorf("%s: got HTTP %d", label, st)
	}
	return body, nil
}

// Valid reports whether the homepage has dropped the anonymous-visitor marker.
func Valid(c *httpx.Client, portal string) (bool, error) {
	body, err := step(c, http.MethodGet, strings.TrimRight(portal, "/")+"/", nil, "session check")
	if err != nil {
		return false, err
	}
	return !strings.Contains(string(body), anonMarker), nil
}

func outcomeErr(body []byte) error {
	switch {
	case strings.Contains(string(body), badCreds):
		return fmt.Errorf("login failed: bad credentials")
	case strings.Contains(string(body), inUse):
		return fmt.Errorf("login failed: the account is currently in use (only one session is permitted). Run merx logout or wait for the existing session to time out")
	}
	return nil
}

// Login runs the SAML flow with exactly one credential attempt.
func Login(c *httpx.Client, ep Endpoints, user, pass string) error {
	body, err := step(c, http.MethodGet, ep.Portal+"/public/authentication/login", nil, "SAML login page")
	if err != nil {
		return err
	}
	idpAction, samlFields, err := parseForm(body, "SAMLRequest")
	if err != nil {
		return fmt.Errorf("SAML auto-post form not found")
	}
	body, err = step(c, http.MethodPost, resolveURL(ep.Portal, idpAction), toValues(samlFields), "identity provider")
	if err != nil {
		return err
	}
	loginAction, loginFields, err := parseForm(body, "j_username", "j_password")
	if err != nil {
		return fmt.Errorf("IdP username/password form not found")
	}
	if _, err := ExecutionToken(loginAction); err != nil {
		return err
	}
	vals := toValues(loginFields)
	vals.Set("j_username", user)
	vals.Set("j_password", pass)
	// Follow the 302, then classify from the landed page text.
	st, body, err := do(c, http.MethodPost, resolveURL(ep.IDP, loginAction), vals)
	if err != nil {
		return err
	}
	if err := outcomeErr(body); err != nil {
		return err
	}
	if st != http.StatusOK {
		return fmt.Errorf("identity provider: got HTTP %d", st)
	}
	acsAction, acsFields, err := parseForm(body, "SAMLResponse")
	if err != nil || !strings.Contains(acsAction, "/saml/SSO") {
		return fmt.Errorf("login failed: no SAML response from the identity provider")
	}
	if _, err = step(c, http.MethodPost, resolveURL(ep.Portal, acsAction), toValues(acsFields), "SAML ACS"); err != nil {
		return err
	}
	ok, err := Valid(c, ep.Portal)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("login failed: portal still shows an anonymous session")
	}
	return nil
}

// Logout ends the server session. MERX allows one session per account,
// so skipping this blocks the next login.
func Logout(c *httpx.Client, portal string) error {
	_, err := step(c, http.MethodGet, strings.TrimRight(portal, "/")+"/logout", nil, "portal logout")
	return err
}
