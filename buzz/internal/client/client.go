package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	RelayURL string
	Keys     *nostr.KeyPair
	AuthTag  nostr.Tag
	HTTP     *http.Client
}

type Filter map[string]any

type PublishResponse struct {
	OK      bool            `json:"ok"`
	Message string          `json:"message,omitempty"`
	EventID string          `json:"event_id,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

func New(relayURL string, keys *nostr.KeyPair, authTag nostr.Tag) *Client {
	return &Client{
		RelayURL: strings.TrimSpace(relayURL),
		Keys:     keys,
		AuthTag:  authTag,
		HTTP:     http.DefaultClient,
	}
}

func (c *Client) PostEvent(ctx context.Context, event nostr.Event) (json.RawMessage, error) {
	var body any = event
	return c.postJSON(ctx, "/events", body)
}

func (c *Client) Query(ctx context.Context, filters []Filter) (json.RawMessage, error) {
	return c.postJSON(ctx, "/query", filters)
}

func (c *Client) Count(ctx context.Context, filters []Filter) (json.RawMessage, error) {
	return c.postJSON(ctx, "/count", filters)
}

// Post issues a NIP-98-signed (when Keys is set) POST to an arbitrary relay path.
func (c *Client) Post(ctx context.Context, path string, payload any) (json.RawMessage, error) {
	return c.postJSON(ctx, path, payload)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) (json.RawMessage, error) {
	if strings.TrimSpace(c.RelayURL) == "" {
		return nil, errors.New("relay URL is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.httpEndpoint(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	if len(c.AuthTag) > 0 {
		authTag, err := nostr.AuthTagJSON(c.AuthTag)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-auth-tag", authTag)
	}
	if c.Keys != nil {
		auth, err := c.httpAuthHeader(endpoint, http.MethodPost, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("authorization", auth)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("relay returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return json.RawMessage(respBody), nil
}

// Get issues a NIP-98-signed GET to an arbitrary relay path (incl. query
// string), returning the raw JSON body. Used by moderation reads, which
// serve structured queue/audit rows rather than stored events.
func (c *Client) Get(ctx context.Context, path string) (json.RawMessage, error) {
	if strings.TrimSpace(c.RelayURL) == "" {
		return nil, errors.New("relay URL is required")
	}
	endpoint, err := c.httpEndpoint(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if len(c.AuthTag) > 0 {
		authTag, err := nostr.AuthTagJSON(c.AuthTag)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-auth-tag", authTag)
	}
	if c.Keys != nil {
		auth, err := c.httpAuthHeader(endpoint, http.MethodGet, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("authorization", auth)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("relay returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return json.RawMessage(respBody), nil
}

// PutRaw issues a PUT to a relay-relative path with an explicit header set
// and a raw (non-JSON-marshaled) body, returning the raw response body.
// Used for Blossom media upload (BUD-02), which carries its own
// kind:24242 auth header rather than NIP-98.
func (c *Client) PutRaw(ctx context.Context, path string, body []byte, headers map[string]string, timeout time.Duration) (json.RawMessage, int, error) {
	endpoint, err := c.httpEndpoint(path)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if len(c.AuthTag) > 0 {
		authTag, err := nostr.AuthTagJSON(c.AuthTag)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("x-auth-tag", authTag)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	if timeout > 0 {
		client := *hc
		client.Timeout = timeout
		hc = &client
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 512<<20))
	if err != nil {
		return nil, 0, err
	}
	return json.RawMessage(respBody), resp.StatusCode, nil
}

// GetBytes issues a GET to an absolute URL (not relay-relative — callers
// validate same-origin themselves) with explicit headers, returning raw
// response bytes without following redirects. Used for Blossom media
// download (BUD-01), whose Authorization/x-auth-tag headers must never be
// forwarded to a redirect target.
func (c *Client) GetBytes(ctx context.Context, absoluteURL string, headers map[string]string, timeout time.Duration) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, absoluteURL, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if len(c.AuthTag) > 0 {
		authTag, err := nostr.AuthTagJSON(c.AuthTag)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("x-auth-tag", authTag)
	}
	hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if timeout > 0 {
		hc.Timeout = timeout
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<20))
	if err != nil {
		return nil, 0, err
	}
	return body, resp.StatusCode, nil
}

// GetRelayInfo fetches the NIP-11 relay information document from "/" without auth.
func (c *Client) GetRelayInfo(ctx context.Context) (json.RawMessage, error) {
	endpoint, err := c.httpEndpoint("/")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/nostr+json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("relay returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.RawMessage(body), nil
}

// httpAuthHeader signs a NIP-98 (kind:27235) HTTP auth event for endpoint.
// payload is nil for GET requests without a body — the "payload" tag is
// only included when there is a body to hash, matching the bundled CLI's
// sign_nip98 (buzz-cli/src/client.rs).
func (c *Client) httpAuthHeader(endpoint, method string, payload []byte) (string, error) {
	if c.Keys == nil {
		return "", errors.New("private key is required")
	}
	tags := nostr.Tags{
		{"u", endpoint},
		{"method", strings.ToUpper(method)},
		{"nonce", uuid.NewString()},
	}
	if payload != nil {
		payloadHash := sha256.Sum256(payload)
		tags = append(tags, nostr.Tag{"payload", hex.EncodeToString(payloadHash[:])})
	}
	if len(c.AuthTag) > 0 {
		tags = append(tags, c.AuthTag)
	}
	event := nostr.NewUnsignedEvent(nostr.KindHTTPAuth, "", "", tags, time.Now().Unix())
	if err := event.Sign(c.Keys); err != nil {
		return "", err
	}
	return "Nostr " + base64.StdEncoding.EncodeToString(event.MustJSON()), nil
}

func (c *Client) httpEndpoint(path string) (string, error) {
	u, err := parseRelayURL(c.RelayURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

type WSClient struct {
	conn     *websocket.Conn
	relayURL string
	keys     *nostr.KeyPair
	authTag  nostr.Tag
}

type WSMessage struct {
	Type    string
	SubID   string
	Event   *nostr.Event
	Notice  string
	OK      bool
	Message string
}

func DialWS(ctx context.Context, relayURL string, keys *nostr.KeyPair, authTag nostr.Tag) (*WSClient, error) {
	wsURL, err := websocketEndpoint(relayURL)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	client := &WSClient{conn: conn, relayURL: relayURL, keys: keys, authTag: authTag}
	if err := client.handleInitialAuth(ctx); err != nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "auth failed")
		return nil, err
	}
	return client, nil
}

func (c *WSClient) Close(status websocket.StatusCode, reason string) error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(status, reason)
}

func (c *WSClient) Publish(ctx context.Context, event nostr.Event) error {
	return c.writeJSON(ctx, []any{"EVENT", event})
}

func (c *WSClient) Req(ctx context.Context, subID string, filters ...Filter) error {
	msg := []any{"REQ", subID}
	for _, filter := range filters {
		msg = append(msg, filter)
	}
	return c.writeJSON(ctx, msg)
}

func (c *WSClient) CloseSubscription(ctx context.Context, subID string) error {
	return c.writeJSON(ctx, []any{"CLOSE", subID})
}

func (c *WSClient) Read(ctx context.Context) (WSMessage, error) {
	var parts []json.RawMessage
	if err := wsjsonRead(ctx, c.conn, &parts); err != nil {
		return WSMessage{}, err
	}
	return decodeWS(parts)
}

func (c *WSClient) handleInitialAuth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	var parts []json.RawMessage
	err := wsjsonRead(ctx, c.conn, &parts)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}
	if len(parts) == 0 {
		return nil
	}
	var typ string
	if err := json.Unmarshal(parts[0], &typ); err != nil {
		return err
	}
	if typ != "AUTH" {
		return nil
	}
	if c.keys == nil {
		return errors.New("relay requested AUTH but no private key was configured")
	}
	var challenge string
	if len(parts) < 2 || json.Unmarshal(parts[1], &challenge) != nil {
		return errors.New("relay AUTH challenge was malformed")
	}
	authEvent, err := nostr.BuildAuthEvent(challenge, c.relayURL, c.keys, c.authTag)
	if err != nil {
		return err
	}
	return c.writeJSON(context.Background(), []any{"AUTH", authEvent})
}

func (c *WSClient) writeJSON(ctx context.Context, value any) error {
	return wsjson.Write(ctx, c.conn, value)
}

func decodeWS(parts []json.RawMessage) (WSMessage, error) {
	if len(parts) == 0 {
		return WSMessage{}, errors.New("empty relay message")
	}
	var typ string
	if err := json.Unmarshal(parts[0], &typ); err != nil {
		return WSMessage{}, err
	}
	msg := WSMessage{Type: typ}
	switch typ {
	case "EVENT":
		if len(parts) >= 2 {
			_ = json.Unmarshal(parts[1], &msg.SubID)
		}
		if len(parts) >= 3 {
			var event nostr.Event
			if err := json.Unmarshal(parts[2], &event); err != nil {
				return WSMessage{}, err
			}
			msg.Event = &event
		}
	case "EOSE", "CLOSED":
		if len(parts) >= 2 {
			_ = json.Unmarshal(parts[1], &msg.SubID)
		}
	case "NOTICE":
		if len(parts) >= 2 {
			_ = json.Unmarshal(parts[1], &msg.Notice)
		}
	case "OK":
		if len(parts) >= 2 {
			var id string
			_ = json.Unmarshal(parts[1], &id)
			msg.SubID = id
		}
		if len(parts) >= 3 {
			_ = json.Unmarshal(parts[2], &msg.OK)
		}
		if len(parts) >= 4 {
			_ = json.Unmarshal(parts[3], &msg.Message)
		}
	case "AUTH":
		if len(parts) >= 2 {
			_ = json.Unmarshal(parts[1], &msg.Message)
		}
	}
	return msg, nil
}

func parseRelayURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("relay URL must include scheme and host")
	}
	return u, nil
}

func websocketEndpoint(relayURL string) (string, error) {
	u, err := parseRelayURL(relayURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported relay scheme %q", u.Scheme)
	}
	return u.String(), nil
}

func wsjsonRead(ctx context.Context, conn *websocket.Conn, value any) error {
	return wsjson.Read(ctx, conn, value)
}
