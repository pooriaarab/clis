// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored Google auth. Not generated; safe across regen.
//
// GA4 needs a dynamically-minted Google OAuth2 access token, not a static
// bearer. This file unifies the three identity modes the user asked for:
//   1. Service account   — GOOGLE_APPLICATION_CREDENTIALS points at a JSON key.
//   2. gcloud user login  — `gcloud auth application-default login` writes ADC.
//   3. Workload identity   — GCE/Cloud Run metadata server.
// All three resolve through google.FindDefaultCredentials. A stored OAuth
// refresh token (client_id + refresh_token in config) is a fallback for the
// desktop-OAuth case without gcloud.

package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GAScopes are the Analytics scopes the CLI requests. It covers full read +
// write: analytics.edit (Admin config writes), analytics.manage.users (access
// bindings), and analytics.readonly (Data API + Admin reads). Write operations
// are still gated at the command layer behind --confirm; the scope only makes
// them possible when the caller opts in. Set GOOGLE_ANALYTICS_TOKEN to override
// with a narrower token if you want a read-only deployment.
var GAScopes = []string{
	"https://www.googleapis.com/auth/analytics.edit",
	"https://www.googleapis.com/auth/analytics.manage.users",
	"https://www.googleapis.com/auth/analytics.readonly",
}

// EnsureGoogleToken mints a Google access token into c.AccessToken when one is
// needed. It is a no-op when an explicit token override is set or the current
// minted token is still valid. Callers treat a returned error as "no creds";
// the API 401 path already prints an actionable message, so the client hook
// swallows the error and lets the request surface it.
func (c *Config) EnsureGoogleToken(ctx context.Context) error {
	// Explicit manual overrides win: a raw token env var, or a fully-formed
	// Authorization header already parsed into config.
	if os.Getenv("GOOGLE_ANALYTICS_TOKEN") != "" || c.AuthHeaderVal != "" {
		return nil
	}
	// Still-valid minted token (60s skew guard).
	if c.AccessToken != "" && !c.TokenExpiry.IsZero() && time.Now().Before(c.TokenExpiry.Add(-60*time.Second)) {
		return nil
	}

	var ts oauth2.TokenSource
	if c.RefreshToken != "" && c.ClientID != "" {
		oc := &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       GAScopes,
		}
		ts = oc.TokenSource(ctx, &oauth2.Token{RefreshToken: c.RefreshToken})
	} else {
		creds, err := google.FindDefaultCredentials(ctx, GAScopes...)
		if err != nil {
			return fmt.Errorf("no Google credentials found: run 'gcloud auth application-default login', "+
				"or set GOOGLE_APPLICATION_CREDENTIALS to a service-account key with GA4 Viewer, "+
				"or set GOOGLE_ANALYTICS_TOKEN to a raw access token: %w", err)
		}
		ts = creds.TokenSource
	}

	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("obtaining Google access token: %w", err)
	}
	c.AccessToken = tok.AccessToken
	c.TokenExpiry = tok.Expiry
	return nil
}
