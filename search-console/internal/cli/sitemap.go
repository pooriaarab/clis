package cli

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
	"github.com/pooriaarab/clis/search-console/internal/sitemap"
)

// sitemapFetchBaseEnv points every sitemap request at one server. The E2E tests
// use it, because the sitemap URL must be a real https URL on the property.
const sitemapFetchBaseEnv = "SITEMAP_FETCH_BASE"

// rebase sends each request to another scheme and host.
type rebase struct {
	to   *url.URL
	next http.RoundTripper
}

func (r rebase) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	c.URL.Scheme, c.URL.Host = r.to.Scheme, r.to.Host
	return r.next.RoundTrip(c)
}

// sitemapClient returns the client that fetches sitemaps. A fetch is a read, so
// it goes out in a dry run too. That way a dry run judges the sitemap like the real run.
func (e *Env) sitemapClient() (*httpx.Client, error) {
	base := e.Getenv(sitemapFetchBaseEnv)
	if base == "" {
		return httpx.New(false), nil
	}
	to, err := url.Parse(base)
	if err != nil || to.Host == "" {
		return nil, &ExitError{Code: ExitUsage, Err: errors.New(sitemapFetchBaseEnv + " is not an absolute URL")}
	}
	hc := &http.Client{Timeout: 30 * time.Second, Transport: rebase{to: to, next: http.DefaultTransport}}
	return &httpx.Client{HTTP: hc}, nil
}

func sitemapCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "sitemap", Short: "Check a sitemap before you submit it"}
	cmd.AddCommand(&cobra.Command{
		Use:   "check <sitemap-url>",
		Short: "Fetch a sitemap and check it",
		Long: `Fetch a sitemap and check it for HTTP 200, valid XML and fewer than 50,000 URLs.
An index must list at most 1,000 sitemaps, all on its own host, and none of them an index.
--dry-run still fetches the sitemap, because a fetch changes nothing. Exit code 3 means the sitemap has problems.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			client, err := env.sitemapClient()
			if err != nil {
				return err
			}
			rep, err := sitemap.Check(context.Background(), client, args[0])
			if errors.Is(err, sitemap.ErrNotAbsolute) {
				return &ExitError{Code: ExitUsage, Err: err}
			}
			if err != nil {
				return err
			}
			if err := env.Emit(rep, func() {
				if rep.OK {
					env.P.Linef("ok: %s has %d URLs (%s)", rep.URL, rep.URLs, rep.Kind)
					return
				}
				env.P.Linef("problems in %s:\n  %s", rep.URL, strings.Join(rep.Problems, "\n  "))
			}); err != nil {
				return err
			}
			if !rep.OK {
				return &ExitError{Code: ExitSitemapError, Err: errors.New("the sitemap has problems"), Printed: true}
			}
			return nil
		}),
	})
	return cmd
}
