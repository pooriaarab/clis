// Package walmart replays a warm browser session against walmart.ca's internal GraphQL API.
package walmart

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	Base = "https://www.walmart.ca"
	// PurchaseHistoryV2 persisted-query hash; `auth import` captures the current one.
	DefaultHistoryHash = "d15c6dbb75db24eebe6462584af35b618a320e531f9c051913b93fc59e40e94f"
	historyOp          = "PurchaseHistoryV2"
)

// Session is the persisted browser snapshot. Never log cookie values.
type Session struct {
	SavedAt string            `json:"saved_at"`
	Hash    string            `json:"purchase_history_hash"`
	Cookies map[string]string `json:"cookies"`
}

func sessPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".walmart-ca", "session.json")
}

func LoadSession() (*Session, error) {
	data, err := os.ReadFile(sessPath())
	if err != nil {
		return nil, fmt.Errorf("no session: run `auth import` first")
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil || len(s.Cookies) == 0 {
		return nil, fmt.Errorf("bad session file: run `auth import` again")
	}
	return &s, nil
}

func SaveSession(s *Session) error {
	s.SavedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(sessPath()), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessPath(), data, 0o600)
}

var (
	hashPattern   = regexp.MustCompile(`/(?:cph|orders)/graphql/[A-Za-z0-9_]+/([a-f0-9]{64})(?:[/?]|$)`)
	urlPattern    = regexp.MustCompile(`https://[^\s'"]+`)
	cookiePattern = regexp.MustCompile(`-H\s+['"]cookie:\s*([^'"]+)['"]`)
)

// ParseCurl extracts a session from Chrome's "Copy as cURL" of a walmart.ca GraphQL request.
func ParseCurl(raw string) (*Session, error) {
	flat := strings.ReplaceAll(raw, "\\\n", " ")
	u, err := url.Parse(urlPattern.FindString(flat))
	if err != nil || !strings.Contains(u.Host, "walmart.ca") {
		return nil, fmt.Errorf("not a walmart.ca request: copy one from a signed-in tab")
	}
	m := cookiePattern.FindStringSubmatch(flat)
	if m == nil {
		return nil, fmt.Errorf("no cookie header found: copy a fresh request")
	}
	cookies := map[string]string{}
	for _, part := range strings.Split(m[1], ";") {
		if k, v, ok := strings.Cut(strings.TrimSpace(part), "="); ok && k != "" {
			cookies[k] = v
		}
	}
	for _, need := range []string{"CID", "SPID", "auth"} {
		if cookies[need] == "" {
			return nil, fmt.Errorf("capture lacks the %s cookie: copy a fresh request", need)
		}
	}
	hash := DefaultHistoryHash
	if hm := hashPattern.FindStringSubmatch(u.Path); len(hm) == 2 {
		hash = hm[1]
	}
	return &Session{Hash: hash, Cookies: cookies}, nil
}

type OrderGroup struct {
	OrderID         string  `json:"orderId"`
	DisplayID       string  `json:"displayId"`
	ItemCount       int     `json:"itemCount"`
	DeliveryMessage string  `json:"deliveryMessage"`
	DeliveredDate   *string `json:"deliveredDate"`
	Store           *struct {
		Name string `json:"name"`
	} `json:"store"`
	Status *struct {
		StatusType string `json:"statusType"`
	} `json:"status"`
	Items []struct {
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
	} `json:"items"`
	OrderGroupDates []struct {
		Value string `json:"value"`
	} `json:"orderGroupDates"`
}

func (g OrderGroup) Date() string {
	for _, d := range g.OrderGroupDates {
		if d.Value != "" {
			return d.Value
		}
	}
	if g.DeliveredDate != nil {
		return *g.DeliveredDate
	}
	return ""
}

// String is the human one-line summary plus item lines.
func (g OrderGroup) String() string {
	id, status := g.DisplayID, g.DeliveryMessage
	if id == "" {
		id = g.OrderID
	}
	if g.Status != nil && g.Status.StatusType != "" {
		status = g.Status.StatusType
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s  %d items  %s", g.Date(), id, g.ItemCount, status)
	if g.Store != nil && g.Store.Name != "" {
		b.WriteString("  " + g.Store.Name)
	}
	for _, it := range g.Items {
		fmt.Fprintf(&b, "\n    %dx %s", it.Quantity, it.Name)
	}
	return b.String()
}

type historyPage struct {
	PageInfo struct {
		Next string `json:"nextPageCursor"`
	} `json:"pageInfo"`
	Groups []OrderGroup `json:"orderGroups"`
}

// Client replays the session against the GraphQL gateway.
type Client struct {
	session *Session
	http    *http.Client
	lastReq time.Time
}

func NewClient(s *Session) *Client {
	return &Client{session: s, http: &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func (c *Client) setHeaders(req *http.Request) {
	cid := newUUID()
	h := map[string]string{
		"accept": "application/json", "accept-language": "en-CA,en;q=0.9", "referer": Base + "/en/orders",
		"user-agent":              "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"x-apollo-operation-name": historyOp, "x-o-gql-query": "query " + historyOp, "x-o-platform": "rweb",
		"x-o-bu": "WALMART-CA", "x-o-mart": "B2C", "x-o-correlation-id": cid, "wm_qos.correlation_id": cid,
		"sec-fetch-site": "same-origin", "sec-fetch-mode": "cors", "sec-fetch-dest": "empty",
	}
	for k, v := range h {
		req.Header.Set(k, v)
	}
	var parts []string
	for k, v := range c.session.Cookies {
		parts = append(parts, k+"="+v)
	}
	req.Header.Set("Cookie", strings.Join(parts, "; "))
}

func (c *Client) History(ctx context.Context, limit int, cursor, search string) ([]OrderGroup, string, error) {
	if wait := 2*time.Second - time.Since(c.lastReq); wait > 0 {
		time.Sleep(wait)
	}
	c.lastReq = time.Now()

	vars, _ := json.Marshal(map[string]any{"input": map[string]any{"cursor": cursor, "search": search, "filterIds": nil, "limit": limit}, "platform": "WEB"})
	endpoint := fmt.Sprintf("%s/orchestra/cph/graphql/%s/%s?variables=%s",
		Base, historyOp, c.session.Hash, url.QueryEscape(string(vars)))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	for _, sc := range resp.Cookies() {
		if _, ok := c.session.Cookies[sc.Name]; ok {
			c.session.Cookies[sc.Name] = sc.Value
		}
	}
	_ = SaveSession(c.session)

	switch resp.StatusCode {
	case 200:
	case 429:
		return nil, "", fmt.Errorf("rate limited by walmart.ca: wait and retry")
	case 403:
		return nil, "", fmt.Errorf("access denied: session expired, run `auth import` with a fresh capture")
	case 456:
		return nil, "", fmt.Errorf("bot challenge: re-import from a browser that passed the check")
	default:
		return nil, "", fmt.Errorf("HTTP %d from walmart.ca", resp.StatusCode)
	}
	var out struct {
		Data struct {
			OH historyPage `json:"orderHistoryV2"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, "", fmt.Errorf("could not parse response: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, "", fmt.Errorf("walmart.ca error: %s", out.Errors[0].Message)
	}
	return out.Data.OH.Groups, out.Data.OH.PageInfo.Next, nil
}
