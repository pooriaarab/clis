package google

import (
	"context"
	"net/http"
	"strings"
)

// SiteURL is the Search Console name of a domain property.
func SiteURL(domain string) string { return "sc-domain:" + domain }

// sitePath is the API path of a property. Google wants the colon encoded.
func sitePath(domain string) string {
	return "/webmasters/v3/sites/" + strings.ReplaceAll(SiteURL(domain), ":", "%3A")
}

// AddSite adds the domain property to Search Console. Adding it twice is fine.
func (a *API) AddSite(ctx context.Context, domain string) error {
	return a.call(ctx, http.MethodPut, sitePath(domain), nil, nil, nil)
}

// Permission returns the account's permission level on the domain property.
func (a *API) Permission(ctx context.Context, domain string) (string, error) {
	var out struct {
		PermissionLevel string `json:"permissionLevel"`
	}
	err := a.call(ctx, http.MethodGet, sitePath(domain), nil, nil, &out)
	return out.PermissionLevel, err
}
