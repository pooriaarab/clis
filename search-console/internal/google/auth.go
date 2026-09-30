// Package google talks to Google OAuth, Site Verification and Search Console.
package google

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// Endpoints. GOOGLE_OAUTH_AUTH_URL and GOOGLE_OAUTH_TOKEN_URL override them.
const (
	DefaultAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	DefaultTokenURL = "https://oauth2.googleapis.com/token"
)

// Scopes the CLI asks for.
var Scopes = []string{
	"https://www.googleapis.com/auth/siteverification",
	"https://www.googleapis.com/auth/webmasters",
}

// ErrTimeout means nobody finished the consent before the deadline.
var ErrTimeout = errors.New("timed out waiting for the browser consent")

// Credentials is the OAuth client and the login it produced. It is the
// content of google.json.
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuth runs the loopback login.
type OAuth struct {
	HTTP     *httpx.Client
	AuthURL  string
	TokenURL string
}

// LoginOpts controls one login.
type LoginOpts struct {
	Timeout time.Duration
	// Notify receives the consent URL so the user can open it by hand.
	Notify func(url string)
	// Open tries to open the consent URL in a browser. A nil Open skips it.
	Open func(url string) error
}

// Login runs the Desktop-app loopback flow with PKCE and returns the new
// refresh token.
func (o *OAuth) Login(ctx context.Context, cred Credentials, opts LoginOpts) (string, error) {
	verifier, err := randomString(48)
	if err != nil {
		return "", err
	}
	state, err := randomString(24)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(verifier))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	redirect := fmt.Sprintf("http://%s/callback", ln.Addr())
	codes := make(chan callback, 1)
	srv := &http.Server{Handler: callbackHandler(state, codes), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	defer srv.Close()

	consent := o.consentURL(cred, redirect, state, sum)
	opts.Notify(consent)
	if opts.Open != nil {
		_ = opts.Open(consent)
	}

	var cb callback
	select {
	case cb = <-codes:
	case <-time.After(opts.Timeout):
		return "", ErrTimeout
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if cb.err != nil {
		return "", cb.err
	}
	if cb.code == "" {
		return "", errors.New("consent failed: the reply has no code")
	}
	tok, err := o.exchange(ctx, url.Values{
		"grant_type": {"authorization_code"}, "code": {cb.code}, "code_verifier": {verifier},
		"redirect_uri": {redirect}, "client_id": {cred.ClientID}, "client_secret": {cred.ClientSecret},
	})
	if err != nil {
		return "", err
	}
	if tok.RefreshToken == "" {
		return "", errors.New("Google returned no refresh token; remove this app at myaccount.google.com/permissions and run auth google again")
	}
	return tok.RefreshToken, nil
}

type callback struct {
	code string
	err  error
}

// callbackHandler receives the browser redirect. It checks the state value.
func callbackHandler(state string, out chan<- callback) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var cb callback
		switch {
		case q.Get("state") != state:
			cb.err = errors.New("the consent reply has a different state value; start again")
		case q.Get("error") != "":
			cb.err = fmt.Errorf("consent failed: %s", q.Get("error"))
		default:
			cb.code = q.Get("code")
		}
		fmt.Fprintln(w, "You can close this tab and return to the terminal.")
		select {
		case out <- cb:
		default:
		}
	})
	return mux
}

func (o *OAuth) consentURL(cred Credentials, redirect, state string, challenge [32]byte) string {
	q := url.Values{
		"client_id":             {cred.ClientID},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {strings.Join(Scopes, " ")},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return o.AuthURL + "?" + q.Encode()
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}

type tokenReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

// TokenError is a refusal from the token endpoint.
type TokenError struct {
	Status      int
	Code        string
	Description string
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("google oauth: HTTP %d: %s %s", e.Status, e.Code, e.Description)
}

func (o *OAuth) exchange(ctx context.Context, form url.Values) (tokenReply, error) {
	var tok tokenReply
	resp, err := o.HTTP.Do(ctx, http.MethodPost, o.TokenURL, nil, form)
	if err != nil {
		return tok, err
	}
	if !resp.OK() {
		_ = resp.Decode(&tok)
		return tok, &TokenError{Status: resp.Status, Code: tok.Error, Description: tok.Description}
	}
	return tok, resp.Decode(&tok)
}

// Refresh trades a refresh token for an access token.
func (o *OAuth) Refresh(ctx context.Context, cred Credentials) (string, error) {
	tok, err := o.exchange(ctx, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {cred.RefreshToken},
		"client_id": {cred.ClientID}, "client_secret": {cred.ClientSecret},
	})
	var te *TokenError
	if errors.As(err, &te) && te.Code == "invalid_grant" {
		return "", fmt.Errorf("%w; the refresh token is expired or revoked. "+
			"Set the OAuth consent screen to In production (Testing tokens last 7 days), then run auth google again", err)
	}
	return tok.AccessToken, err
}
