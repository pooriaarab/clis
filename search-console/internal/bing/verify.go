package bing

import (
	"context"
	"errors"
	"net/http"

	"github.com/pooriaarab/clis/search-console/internal/backoff"
)

// ErrVerifyTimeout means Bing never found the CNAME record in time.
var ErrVerifyTimeout = errors.New("Bing did not find the CNAME record in time")

// VerifyTarget is where the verification CNAME must point.
const VerifyTarget = "verify.bing.com"

// FindSite returns the site of the account for domain, or nil when it is not added.
func (a *API) FindSite(ctx context.Context, domain string) (*Site, error) {
	sites, err := a.Sites(ctx)
	if err != nil {
		return nil, err
	}
	for i := range sites {
		if sites[i].URL == SiteURL(domain) {
			return &sites[i], nil
		}
	}
	return nil, nil
}

// AddSite adds the domain to the account. Bing refuses a site it already holds.
func (a *API) AddSite(ctx context.Context, domain string) error {
	return a.call(ctx, http.MethodPost, "AddSite", nil, map[string]string{"siteUrl": SiteURL(domain)}, nil)
}

// VerifyOnce asks Bing to check the CNAME record. It reports whether Bing
// found it.
func (a *API) VerifyOnce(ctx context.Context, domain string) (bool, error) {
	var ok bool
	err := a.call(ctx, http.MethodPost, "VerifySite", nil, map[string]string{"siteUrl": SiteURL(domain)}, &ok)
	return ok || a.HTTP.DryRun, err
}

// VerifyWithBackoff checks until Bing finds the record. It returns the number of tries.
func (a *API) VerifyWithBackoff(ctx context.Context, domain string, o backoff.Opts) (int, error) {
	n, err := backoff.Until(ctx, o, func() (bool, error) { return a.VerifyOnce(ctx, domain) })
	if errors.Is(err, backoff.ErrTimeout) {
		err = ErrVerifyTimeout
	}
	return n, err
}
