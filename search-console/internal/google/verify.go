package google

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/pooriaarab/clis/search-console/internal/backoff"
)

// ErrVerifyTimeout means Google never found the TXT record in time.
var ErrVerifyTimeout = errors.New("Google did not find the TXT record in time")

func domainSite(domain string) map[string]string {
	return map[string]string{"type": "INET_DOMAIN", "identifier": domain}
}

// VerificationToken asks for the TXT record value that proves ownership of domain.
func (a *API) VerificationToken(ctx context.Context, domain string) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	err := a.call(ctx, http.MethodPost, "/siteVerification/v1/token", nil,
		map[string]any{"site": domainSite(domain), "verificationMethod": "DNS_TXT"}, &out)
	return out.Token, err
}

// Verify asks Google to check the TXT record once.
func (a *API) Verify(ctx context.Context, domain string) error {
	return a.call(ctx, http.MethodPost, "/siteVerification/v1/webResource",
		url.Values{"verificationMethod": {"DNS_TXT"}}, map[string]any{"site": domainSite(domain)}, nil)
}

// VerifyWithBackoff calls Verify until it works. Google answers 400 while the
// TXT record is not visible yet, so only 400 is retried. It returns the number
// of attempts.
func (a *API) VerifyWithBackoff(ctx context.Context, domain string, o backoff.Opts) (int, error) {
	n, err := backoff.Until(ctx, o, func() (bool, error) {
		err := a.Verify(ctx, domain)
		var api *APIError
		if errors.As(err, &api) && api.Status == http.StatusBadRequest {
			return false, nil
		}
		return err == nil, err
	})
	if errors.Is(err, backoff.ErrTimeout) {
		err = ErrVerifyTimeout
	}
	return n, err
}

// IsVerified reports whether the account already owns domain.
func (a *API) IsVerified(ctx context.Context, domain string) (bool, error) {
	var out struct {
		Items []struct {
			Site struct{ Type, Identifier string } `json:"site"`
		} `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/siteVerification/v1/webResource", nil, nil, &out); err != nil {
		return false, err
	}
	for _, it := range out.Items {
		if it.Site.Type == "INET_DOMAIN" && it.Site.Identifier == domain {
			return true, nil
		}
	}
	return false, nil
}
