// Package session is the MERX SAML cookie jar and form helpers.
package session

import (
	"encoding/json"
	"fmt"
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
	JarFile = "cookies.json" // cookie file under the cache dir
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
		if c.MaxAge < 0 || (!c.Expires.IsZero() && !now.Before(c.Expires)) {
			if keep >= 0 {
				j.items = append(j.items[:keep], j.items[keep+1:]...)
			}
			continue
		}
		expires := c.Expires
		if expires.IsZero() && c.MaxAge > 0 {
			expires = now.Add(time.Duration(c.MaxAge) * time.Second)
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
