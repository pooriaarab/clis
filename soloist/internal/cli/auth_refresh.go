// Hand-authored (not generated). Transparent Firebase ID-token refresh: before
// each authenticated request, if the stored token is expired (or about to be)
// and a refresh token is present, mint a fresh ID token via the Firebase
// securetoken service and persist it. Wired from rootFlags.newClient (see the
// patch note).

package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"soloist-pp-cli/internal/config"
)

const secureTokenURL = "https://securetoken.googleapis.com/v1/token"

// tokenSkew refreshes slightly before actual expiry so an in-flight request
// doesn't race the boundary.
const tokenSkew = 60 * time.Second

// maybeRefreshToken refreshes cfg's stored ID token in place (and on disk) when
// it is expired/near-expiry and a refresh token is available. Best-effort: on
// any failure it leaves cfg untouched so the existing token is still tried (and
// a genuine 401 surfaces normally). Returns true if a refresh happened.
func maybeRefreshToken(cfg *config.Config, timeout time.Duration) bool {
	if cfg == nil || cfg.RefreshToken == "" {
		return false
	}
	// Env-var token overrides always win and are managed outside the CLI.
	if os.Getenv("SOLOIST_ID_TOKEN") != "" {
		return false
	}
	// Still valid with margin -> nothing to do.
	if !cfg.TokenExpiry.IsZero() && time.Now().Add(tokenSkew).Before(cfg.TokenExpiry) {
		return false
	}
	// Stored expiry unknown (e.g. token set via `auth set-token`): derive the
	// real expiry from the ID token's own `exp` claim so an expired token still
	// gets refreshed. Previously this returned false whenever an access token
	// was present, so an expired-but-unknown-expiry token never refreshed and
	// every request 401'd until a manual re-login.
	if cfg.TokenExpiry.IsZero() && cfg.AccessToken != "" {
		exp, ok := jwtExp(cfg.AccessToken)
		if !ok {
			return false // can't tell; leave the existing token to be tried
		}
		if time.Now().Add(tokenSkew).Before(exp) {
			return false // still valid per the token's own exp claim
		}
		// else: expired -> fall through and refresh
	}

	key := os.Getenv(soloistFirebaseKeyEnv)
	if key == "" {
		key = defaultFirebaseWebAPIKey()
	}
	idToken, refreshToken, expiresIn, err := secureTokenRefresh(context.Background(), cfg.RefreshToken, key, timeout)
	if err != nil || idToken == "" {
		return false
	}
	if refreshToken == "" {
		refreshToken = cfg.RefreshToken
	}
	expiry := time.Now().Add(time.Duration(expiresIn) * time.Second)
	if expiresIn <= 0 {
		expiry = time.Now().Add(time.Hour)
	}
	// Persist (also mutates cfg in place, so the client built next reads it).
	_ = cfg.SaveTokens("", "", idToken, refreshToken, expiry)
	return true
}

// secureTokenRefresh exchanges a refresh token for a fresh ID token. The
// securetoken service expects form-encoded input and returns snake_case JSON.
func secureTokenRefresh(ctx context.Context, refreshToken, apiKey string, timeout time.Duration) (idToken, newRefresh string, expiresIn int, err error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, secureTokenURL+"?key="+url.QueryEscape(apiKey), strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// The Firebase web API key is HTTP-referrer-restricted to the app origin;
	// a CLI request has no referer and is blocked (API_KEY_HTTP_REFERRER_BLOCKED)
	// without this header.
	req.Header.Set("Referer", soloistWebBase()+"/")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, &refreshError{status: resp.StatusCode, body: string(body)}
	}
	var parsed struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", 0, err
	}
	secs, _ := strconv.Atoi(strings.TrimSpace(parsed.ExpiresIn))
	return parsed.IDToken, parsed.RefreshToken, secs, nil
}

type refreshError struct {
	status int
	body   string
}

func (e *refreshError) Error() string {
	return "securetoken refresh failed: HTTP " + strconv.Itoa(e.status) + ": " + e.body
}

// jwtExp reads the `exp` (seconds since epoch) claim from a JWT's payload
// without verifying the signature. Used to recover a token's real expiry when
// none was persisted alongside it. Returns ok=false if the token isn't a
// parseable JWT or has no exp.
func jwtExp(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		if payload, err = base64.URLEncoding.DecodeString(parts[1]); err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
