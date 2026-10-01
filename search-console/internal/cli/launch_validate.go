package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
	"github.com/pooriaarab/clis/search-console/internal/sitemap"
)

// launchInputs are the flags that launch checks before any step runs.
type launchInputs struct {
	domain, sitemapURL, keyDir, keyLocation, urlsFile string
	indexnow                                          bool // false when --skip indexnow
}

// validateLaunch checks every input before launch makes a remote change. A dry
// run calls it too, so a dry run refuses the same inputs as a real run. It makes
// read-only requests only: it reads the sitemap that IndexNow will use. A
// sitemap that cannot be read is not an input error. The steps report it.
func (e *Env) validateLaunch(in launchInputs) error {
	usage := func(format string, a ...any) error {
		return &ExitError{Code: ExitUsage, Err: fmt.Errorf(format, a...)}
	}
	if _, err := absoluteURL(in.sitemapURL); err != nil {
		return usage("--sitemap %q: %w", in.sitemapURL, sitemap.ErrNotAbsolute)
	}
	if !in.indexnow {
		return nil
	}
	if info, err := os.Stat(in.keyDir); err != nil || !info.IsDir() {
		return usage("--key-dir %q is not a directory", in.keyDir)
	}
	if in.keyLocation != "" {
		if _, err := absoluteURL(in.keyLocation); err != nil {
			return usage("--key-location %q is not an absolute http(s) URL", in.keyLocation)
		}
	}
	probe := *e // read the list for real, even in a dry run
	probe.Client, probe.DryRun = httpx.New(false), false
	var err error
	if in.urlsFile != "" {
		_, err = indexnowURLs(context.Background(), &probe, in.domain, in.urlsFile, "")
		var ee *ExitError
		if err != nil && !errors.As(err, &ee) {
			return usage("--indexnow-urls: %w", err)
		}
	} else {
		_, err = indexnowURLs(context.Background(), &probe, in.domain, "", in.sitemapURL)
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage {
			err = nil // the sitemap could not be read: the sitemap steps say so
		}
	}
	return err
}

func absoluteURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("not an absolute http(s) URL")
	}
	return u, nil
}
