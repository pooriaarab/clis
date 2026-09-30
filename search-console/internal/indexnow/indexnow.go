// Package indexnow sends changed URLs to the IndexNow endpoint.
package indexnow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// DefaultBase is the shared IndexNow endpoint. INDEXNOW_API_BASE overrides it.
const DefaultBase = "https://api.indexnow.org/indexnow"

// API posts URL lists to IndexNow.
type API struct {
	HTTP *httpx.Client
	Base string
}

// NewKey makes a 32 character key. IndexNow allows 8 to 128 letters, digits and dashes.
func NewKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CheckKeyFile fetches the key file the way IndexNow will. It fails unless the
// file answers 200 and holds exactly the key.
func (a *API) CheckKeyFile(ctx context.Context, location, key string) error {
	resp, err := a.HTTP.Do(ctx, http.MethodGet, location, nil, nil)
	if err != nil {
		return err
	}
	switch {
	case resp.DryRun || (resp.Status == http.StatusOK && strings.TrimSpace(string(resp.Body)) == key):
		return nil
	case resp.Status != http.StatusOK:
		return fmt.Errorf("%s answered HTTP %d, want 200", location, resp.Status)
	default:
		return fmt.Errorf("%s does not hold the key (a soft 404 page, or an old key file)", location)
	}
}

// Submit posts one batch. IndexNow answers 200 or 202 when it accepts the batch.
func (a *API) Submit(ctx context.Context, host, key, location string, urls []string) (int, error) {
	body := map[string]any{"host": host, "key": key, "keyLocation": location, "urlList": urls}
	resp, err := a.HTTP.Do(ctx, http.MethodPost, a.Base, map[string]string{"Content-Type": "application/json; charset=utf-8"}, body)
	if err != nil {
		return 0, err
	}
	if resp.DryRun || resp.Status == http.StatusOK || resp.Status == http.StatusAccepted {
		return resp.Status, nil
	}
	return resp.Status, fmt.Errorf("indexnow: HTTP %d", resp.Status)
}
