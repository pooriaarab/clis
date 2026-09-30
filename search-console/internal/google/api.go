package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// DefaultAPIBase is the host of the Site Verification and Search Console APIs.
// GOOGLE_API_BASE overrides it.
const DefaultAPIBase = "https://www.googleapis.com"

// API calls Google REST APIs with a bearer token.
type API struct {
	HTTP  *httpx.Client
	Base  string
	Token string
}

// APIError is a non-2xx reply from Google.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("google api: HTTP %d: %s", e.Status, e.Message) }

// call sends one request. A non-2xx reply becomes an *APIError. out may be nil.
func (a *API) call(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := a.Base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	resp, err := a.HTTP.Do(ctx, method, u, map[string]string{"Authorization": "Bearer " + a.Token}, body)
	if err != nil {
		return err
	}
	if !resp.OK() {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(resp.Body, &e)
		msg := e.Error.Message
		if msg == "" {
			msg = strings.TrimSpace(string(resp.Body))
		}
		return &APIError{Status: resp.Status, Message: msg}
	}
	if out == nil {
		return nil
	}
	return resp.Decode(out)
}
