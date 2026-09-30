// Package httpx sends HTTP requests and records them in dry-run mode.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Call is one request that dry-run mode printed instead of sending.
type Call struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   string `json:"body,omitempty"`
}

// Client sends requests. In dry-run mode it records them and sends nothing.
type Client struct {
	HTTP   *http.Client
	DryRun bool
	Calls  []Call
}

// New returns a client with a 30 second timeout per request.
func New(dryRun bool) *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Second}, DryRun: dryRun}
}

// Response is a reply. A dry-run reply is empty and counts as OK.
type Response struct {
	Status int
	Body   []byte
	DryRun bool
}

// OK reports a 2xx reply, or a dry-run.
func (r *Response) OK() bool { return r.DryRun || (r.Status >= 200 && r.Status < 300) }

// Decode reads the JSON body into v. A dry-run or empty body leaves v unchanged.
func (r *Response) Decode(v any) error {
	if r.DryRun || len(bytes.TrimSpace(r.Body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("decode reply: %w", err)
	}
	return nil
}

// Do sends one request. body is nil, raw JSON ([]byte), a form (url.Values) or any value to encode
// as JSON. Any HTTP status comes back as a Response.
// Only a transport failure returns an error.
func (c *Client) Do(ctx context.Context, method, rawURL string, header map[string]string, body any) (*Response, error) {
	var payload []byte
	contentType := ""
	switch b := body.(type) {
	case nil:
	case []byte:
		payload, contentType = b, "application/json"
	case url.Values:
		payload, contentType = []byte(b.Encode()), "application/x-www-form-urlencoded"
	default:
		enc, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		payload, contentType = enc, "application/json"
	}
	if c.DryRun {
		c.Calls = append(c.Calls, Call{Method: method, URL: rawURL, Body: redactBody(contentType, payload)})
		return &Response{Status: http.StatusOK, DryRun: true}, nil
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	return &Response{Status: resp.StatusCode, Body: data}, nil
}

// redactBody hides form values, which carry secrets, except the safe ones.
func redactBody(contentType string, payload []byte) string {
	if contentType != "application/x-www-form-urlencoded" {
		return string(payload)
	}
	form, err := url.ParseQuery(string(payload))
	if err != nil {
		return "REDACTED"
	}
	for k := range form {
		if k != "grant_type" && k != "redirect_uri" && k != "client_id" {
			form.Set(k, "REDACTED")
		}
	}
	return form.Encode()
}
