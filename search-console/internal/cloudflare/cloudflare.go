// Package cloudflare creates DNS records through the Cloudflare API.
package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// DefaultBase is the Cloudflare API. CLOUDFLARE_API_BASE overrides it.
const DefaultBase = "https://api.cloudflare.com/client/v4"

// ErrNoZone means the account has no zone for the domain.
var ErrNoZone = errors.New("no Cloudflare zone found for the domain")

// API calls Cloudflare with an API token.
type API struct {
	HTTP  *httpx.Client
	Base  string
	Token string
}

// Record is a DNS record.
type Record struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// call sends one request and unwraps the {success, errors, result} envelope.
func (a *API) call(ctx context.Context, method, path string, query url.Values, body, result any) error {
	u := a.Base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	resp, err := a.HTTP.Do(ctx, method, u, map[string]string{"Authorization": "Bearer " + a.Token}, body)
	if err != nil {
		return err
	}
	if resp.DryRun {
		return nil
	}
	var env struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result json.RawMessage `json:"result"`
	}
	_ = json.Unmarshal(resp.Body, &env)
	if !resp.OK() || !env.Success {
		msgs := []string{}
		for _, e := range env.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("cloudflare: HTTP %d: %s", resp.Status, strings.Join(msgs, "; "))
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(env.Result, result)
}

// ZoneID finds the zone that holds domain. It tries domain, then each parent.
func (a *API) ZoneID(ctx context.Context, domain string) (string, error) {
	labels := strings.Split(domain, ".")
	for i := 0; i < len(labels)-1; i++ {
		var zones []struct {
			ID string `json:"id"`
		}
		name := strings.Join(labels[i:], ".")
		if err := a.call(ctx, http.MethodGet, "/zones", url.Values{"name": {name}}, nil, &zones); err != nil {
			return "", err
		}
		if a.HTTP.DryRun {
			return "ZONE_ID", nil
		}
		if len(zones) > 0 {
			return zones[0].ID, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrNoZone, domain)
}

// EnsureRecord creates r unless a record with the same type, name and content
// exists. It reports whether it created one.
func (a *API) EnsureRecord(ctx context.Context, zoneID string, r Record) (bool, error) {
	var found []Record
	q := url.Values{"type": {r.Type}, "name": {r.Name}, "content": {r.Content}}
	if err := a.call(ctx, http.MethodGet, "/zones/"+zoneID+"/dns_records", q, nil, &found); err != nil {
		return false, err
	}
	if len(found) > 0 {
		return false, nil
	}
	return true, a.call(ctx, http.MethodPost, "/zones/"+zoneID+"/dns_records", nil, r, nil)
}
