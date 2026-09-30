package google

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
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

// BackoffOpts controls how long VerifyWithBackoff waits for DNS.
type BackoffOpts struct {
	Wait     time.Duration // total time before giving up
	Interval time.Duration // first delay; it doubles up to MaxInterval
	// OnRetry is called before each wait. It may be nil.
	OnRetry func(attempt int, delay time.Duration)
}

// MaxInterval caps the delay between two verify attempts.
const MaxInterval = 60 * time.Second

// VerifyWithBackoff calls Verify until it works. Google answers 400 while the
// TXT record is not visible yet, so only 400 is retried. It returns the number
// of attempts.
func (a *API) VerifyWithBackoff(ctx context.Context, domain string, o BackoffOpts) (int, error) {
	deadline := time.Now().Add(o.Wait)
	delay := o.Interval
	for attempt := 1; ; attempt++ {
		err := a.Verify(ctx, domain)
		var api *APIError
		if err == nil || !errors.As(err, &api) || api.Status != http.StatusBadRequest {
			return attempt, err
		}
		if time.Now().Add(delay).After(deadline) {
			return attempt, ErrVerifyTimeout
		}
		if o.OnRetry != nil {
			o.OnRetry(attempt, delay)
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return attempt, ctx.Err()
		}
		delay = min(delay*2, MaxInterval)
	}
}
