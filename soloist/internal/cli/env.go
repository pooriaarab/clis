// Hand-authored (not generated). Environment selection (prod vs nonprod).
//
// soloist.ai is the prod Firebase project (moz-ocho-solo-prod); main.soloist.ai
// is nonprod (moz-ocho-solo-nonprod). The generated command paths hardcode the
// prod project, so for nonprod we rewrite the project segment in every outgoing
// Firestore request via a RoundTripper installed on the client. Web-key calls
// (publish/login/refresh) read the web origin + Firebase key from env too.
//
// Enable nonprod with:
//   SOLOIST_ENV=nonprod
//   SOLOIST_WEB_BASE_URL=https://main.soloist.ai
//   SOLOIST_FIREBASE_API_KEY=<nonprod web key>
// or override the project directly with SOLOIST_PROJECT=<project-id>.

package cli

import (
	"net/http"
	"os"
	"strings"

	"soloist-pp-cli/internal/client"
)

func getenv(k string) string { return os.Getenv(k) }

const (
	prodFirestoreProject    = "moz-ocho-solo-prod"
	nonprodFirestoreProject = "moz-ocho-solo-nonprod"
	prodWebBaseURL          = "https://soloist.ai"
)

func init() {
	registerClientHook(func(c *client.Client) error {
		project := targetFirestoreProject()
		if project == prodFirestoreProject || c == nil || c.HTTPClient == nil {
			return nil
		}
		base := c.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		c.HTTPClient.Transport = &projectRewriteTransport{base: base, project: project}
		return nil
	})
}

// targetFirestoreProject returns the Firestore project to talk to. Prod unless
// SOLOIST_PROJECT is set explicitly or SOLOIST_ENV selects nonprod.
func targetFirestoreProject() string {
	if p := strings.TrimSpace(getenv("SOLOIST_PROJECT")); p != "" {
		return p
	}
	if strings.EqualFold(strings.TrimSpace(getenv("SOLOIST_ENV")), "nonprod") {
		return nonprodFirestoreProject
	}
	return prodFirestoreProject
}

// soloistWebBase returns the web origin used for the publish/login/refresh
// server + Firebase-key calls (Referer + publish POST base).
func soloistWebBase() string {
	if u := strings.TrimSpace(getenv("SOLOIST_WEB_BASE_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return prodWebBaseURL
}

// projectRewriteTransport swaps the prod Firestore project id for the target
// project in each request URL, so nonprod hits the right database without
// regenerating the hardcoded command paths.
type projectRewriteTransport struct {
	base    http.RoundTripper
	project string
}

func (t *projectRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, prodFirestoreProject) {
		clone := req.Clone(req.Context())
		clone.URL.Path = strings.ReplaceAll(clone.URL.Path, prodFirestoreProject, t.project)
		req = clone
	}
	return t.base.RoundTrip(req)
}
