package cli

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/config"
	"github.com/pooriaarab/clis/search-console/internal/google"
)

func authCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Log in to Google"}
	cmd.AddCommand(authGoogleCmd(env), authStatusCmd(env))
	return cmd
}

func (e *Env) oauth() *google.OAuth {
	return &google.OAuth{
		HTTP:     e.Client,
		AuthURL:  envOr(e.Getenv, "GOOGLE_OAUTH_AUTH_URL", google.DefaultAuthURL),
		TokenURL: envOr(e.Getenv, "GOOGLE_OAUTH_TOKEN_URL", google.DefaultTokenURL),
	}
}

func envOr(getenv func(string) string, name, def string) string {
	if v := getenv(name); v != "" {
		return v
	}
	return def
}

func authGoogleCmd(env *Env) *cobra.Command {
	var cred google.Credentials
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "google",
		Short: "Log in to Google with the browser loopback flow",
		Long: `Log in to Google with the browser loopback flow and save the refresh token
to ~/.config/search-console/google.json (mode 0600).

Create a Desktop OAuth client first and enable the Site Verification and Search
Console APIs. Set the consent screen to In production, or the refresh token
expires after 7 days.`,
		Args: cobra.NoArgs,
		RunE: run(func([]string) error {
			if env.DryRun {
				return &ExitError{Code: ExitUsage, Err: errors.New("auth google needs a browser and has no dry run")}
			}
			cred.ClientID = firstNonEmpty(cred.ClientID, env.Getenv("GOOGLE_CLIENT_ID"))
			cred.ClientSecret = firstNonEmpty(cred.ClientSecret, env.Getenv("GOOGLE_CLIENT_SECRET"))
			if cred.ClientID == "" || cred.ClientSecret == "" {
				return errors.New("no OAuth client: pass --client-id and --client-secret, or set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
			}
			opts := google.LoginOpts{
				Timeout: timeout,
				Notify:  func(url string) { env.P.Warnf("Open this URL to give consent:\n%s", url) },
			}
			opts.Open = func(url string) error { return openBrowser(env.Getenv("BROWSER"), url) }
			refresh, err := env.oauth().Login(context.Background(), cred, opts)
			if errors.Is(err, google.ErrTimeout) {
				return &ExitError{Code: ExitTimeout, Err: err}
			}
			if err != nil {
				return err
			}
			dir, err := config.Dir(env.Getenv)
			if err != nil {
				return err
			}
			cred.RefreshToken = refresh
			if err := config.Save(dir, "google", cred); err != nil {
				return err
			}
			file := filepath.Join(dir, "google.json")
			return env.Emit(map[string]any{"ok": true, "scopes": google.Scopes, "config_file": file}, func() {
				env.P.Linef("Logged in. Refresh token saved to %s", file)
			})
		}),
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "how long to wait for the consent")
	cmd.Flags().StringVar(&cred.ClientID, "client-id", "", "OAuth client id (or GOOGLE_CLIENT_ID)")
	cmd.Flags().StringVar(&cred.ClientSecret, "client-secret", "", "OAuth client secret (or GOOGLE_CLIENT_SECRET)")
	return cmd
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// openBrowser starts the BROWSER command if set, else the OS opener.
func openBrowser(browser, url string) error {
	if browser == "" {
		browser = "xdg-open"
		if runtime.GOOS == "darwin" {
			browser = "open"
		}
	}
	c := exec.Command(browser, url)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}
