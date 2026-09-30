package google

import (
	"context"
	"errors"

	"github.com/pooriaarab/clis/search-console/internal/config"
)

// Resolved is the credential set after env overrides, with its source.
type Resolved struct {
	Credentials
	AccessToken string
	// Source is "env:<NAME>", "file" or "" when nothing is configured.
	Source string
}

// Resolve merges google.json with the env. Env values win one by one:
// GOOGLE_ACCESS_TOKEN, GOOGLE_REFRESH_TOKEN, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET.
func Resolve(getenv func(string) string) (Resolved, error) {
	var r Resolved
	dir, err := config.Dir(getenv)
	if err != nil {
		return r, err
	}
	found, err := config.Load(dir, "google", &r.Credentials)
	if err != nil {
		return r, err
	}
	if found {
		r.Source = "file"
	}
	if v := getenv("GOOGLE_CLIENT_ID"); v != "" {
		r.ClientID = v
	}
	if v := getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		r.ClientSecret = v
	}
	if v := getenv("GOOGLE_REFRESH_TOKEN"); v != "" {
		r.RefreshToken, r.Source = v, "env:GOOGLE_REFRESH_TOKEN"
	}
	if v := getenv("GOOGLE_ACCESS_TOKEN"); v != "" {
		r.AccessToken, r.Source = v, "env:GOOGLE_ACCESS_TOKEN"
	}
	return r, nil
}

// Ready reports whether the credentials can produce a bearer token, without
// calling Google.
func (r Resolved) Ready() error {
	if r.AccessToken != "" {
		return nil
	}
	if r.RefreshToken == "" || r.ClientID == "" || r.ClientSecret == "" {
		return errors.New("not logged in to Google: run `search-console auth google`, or set GOOGLE_ACCESS_TOKEN, or GOOGLE_REFRESH_TOKEN with GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
	return nil
}

// AccessToken returns a bearer token. An access token in the env is used as is.
func (o *OAuth) AccessToken(ctx context.Context, r Resolved) (string, error) {
	if err := r.Ready(); err != nil {
		return "", err
	}
	if r.AccessToken != "" {
		return r.AccessToken, nil
	}
	return o.Refresh(ctx, r.Credentials)
}
