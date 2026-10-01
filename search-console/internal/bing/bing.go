// Package bing talks to the Bing Webmaster API.
package bing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// DefaultBase is the Bing Webmaster JSON API. BING_API_BASE overrides it.
const DefaultBase = "https://ssl.bing.com/webmaster/api.svc/json"

// API calls Bing with an API key from BING_WEBMASTER_API_KEY.
type API struct {
	HTTP *httpx.Client
	Base string
	Key  string
}

// CodeAlreadyPresent is the error code Bing sends when the site or the sitemap
// is already there. It means the work is done.
const CodeAlreadyPresent = 81058

// maxMessage is the most text of a Bing error the CLI keeps.
const maxMessage = 200

// APIError is an error reply from Bing. Bing sends it as HTTP 400 or as a
// 200 reply whose body holds an ErrorCode. Message never holds the API key.
type APIError struct {
	Status  int
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bing api: HTTP %d: %s (code %d)", e.Status, e.Message, e.Code)
}

// IsAlreadyPresent reports whether err is Bing saying the site or sitemap already exists.
func IsAlreadyPresent(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == CodeAlreadyPresent
}

var apikeyParam = regexp.MustCompile(`(?i)apikey=[^&\s)"'<]*`)

// safeMessage removes the API key from Bing's error text and cuts it short.
// Bing can repeat the request URL, and the key is in its query. The reply can also be a page of HTML.
func safeMessage(msg, key string) string {
	for _, k := range []string{key, url.QueryEscape(key), url.PathEscape(key)} {
		if k != "" {
			msg = strings.ReplaceAll(msg, k, "REDACTED")
		}
	}
	msg = apikeyParam.ReplaceAllString(msg, "apikey=REDACTED")
	msg = strings.Join(strings.Fields(msg), " ")
	if r := []rune(msg); len(r) > maxMessage {
		msg = string(r[:maxMessage]) + "..."
	}
	return msg
}

// SiteURL is how the CLI names a domain in Bing.
func SiteURL(domain string) string { return "https://" + domain + "/" }

// call sends one request. Replies wrap the result in {"d": ...}. out may be nil.
func (a *API) call(ctx context.Context, method, name string, query url.Values, body, out any) error {
	q := url.Values{"apikey": {a.Key}}
	for k, v := range query {
		q[k] = v
	}
	resp, err := a.HTTP.Do(ctx, method, a.Base+"/"+name+"?"+q.Encode(), nil, body)
	if err != nil {
		return err
	}
	if resp.DryRun {
		return nil
	}
	var env struct {
		D         json.RawMessage `json:"d"`
		ErrorCode *int            `json:"ErrorCode"`
		Message   string          `json:"Message"`
	}
	_ = json.Unmarshal(resp.Body, &env)
	if !resp.OK() || env.ErrorCode != nil {
		e := &APIError{Status: resp.Status, Message: env.Message}
		if env.ErrorCode != nil {
			e.Code = *env.ErrorCode
		}
		if e.Message == "" {
			e.Message = string(resp.Body)
		}
		e.Message = safeMessage(e.Message, a.Key)
		return e
	}
	if out == nil || len(env.D) == 0 {
		return nil
	}
	return json.Unmarshal(env.D, out)
}

// Site is one site of the account, as GetUserSites reports it.
type Site struct {
	URL        string `json:"Url"`
	IsVerified bool   `json:"IsVerified"`
	// DNSRecord is the whole name of the verification CNAME, "<code>.<domain>".
	// GetUserSites sends it in the field DnsVerificationCode.
	DNSRecord string `json:"DnsVerificationCode"`
}

// Sites lists the sites of the account.
func (a *API) Sites(ctx context.Context) ([]Site, error) {
	var sites []Site
	err := a.call(ctx, "GET", "GetUserSites", nil, nil, &sites)
	return sites, err
}

// Quota is the URL submission quota left for a site.
type Quota struct {
	Daily   int `json:"DailyQuota"`
	Monthly int `json:"MonthlyQuota"`
}

// Quota reads how many URLs the site can still submit.
func (a *API) Quota(ctx context.Context, domain string) (Quota, error) {
	var q Quota
	err := a.call(ctx, "GET", "GetUrlSubmissionQuota", url.Values{"siteUrl": {SiteURL(domain)}}, nil, &q)
	return q, err
}
