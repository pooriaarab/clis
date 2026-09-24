package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"merx-cli/internal/httpx"
)

// rewriteTransport redirects every merx.com request to the httptest server
// so the fallback test never touches the live site.
type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tu, err := url.Parse(t.target)
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	u := *clone.URL
	u.Scheme = tu.Scheme
	u.Host = tu.Host
	clone.URL = &u
	return http.DefaultTransport.RoundTrip(clone)
}

func emptyListHTML() string {
	return `<span class="simpleSolResultsNumResults">0 results</span>`
}

func matchListHTML(ref string) string {
	return `<span class="simpleSolResultsNumResults">1 results</span>` +
		`<table><tbody><tr class="mets-table-row"><td>` +
		`<a id="searchResultSol_solicitation_4071906311" href="/solicitations/open-bids/Test-Slug/` + ref + `?origin=0" class="solicitation-link">` +
		`<span class="rowTitle">Test title for ` + ref + `</span>` +
		`<span class="buyer-name">Test Buyer</span>` +
		`</a></td></tr></tbody></table>`
}

func detailHTML(ref string) string {
	return `<div class="mets-field"><span class="mets-field-label">Reference Number</span><div class="mets-field-body">` + ref + `</div></div>` +
		`<div class="mets-field"><span class="mets-field-label">Title</span><div class="mets-field-body">Test title for ` + ref + `</div></div>`
}

// TestLoadNoticeMultiStatusFallback proves a reference number that exists
// only under a non-open status still resolves, and that the search stops
// at the first matching status instead of querying the rest.
func TestLoadNoticeMultiStatusFallback(t *testing.T) {
	statuses := []string{"open", "awarded", "bid-results", "closed"}
	// Starts with 0 and is short, so DetailCandidates returns nil and the
	// test exercises only the status fallback, with no direct-URL noise.
	const ref = "0000999999"

	cases := []struct {
		name  string
		match string // status serving the matching row; "" serves none
	}{
		{"found under open", "open"},
		{"found under awarded", "awarded"},
		{"found under bid-results", "bid-results"},
		{"found under closed", "closed"},
		{"found nowhere", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits := map[string]int{}
			var detailHits int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for _, s := range statuses {
					if r.URL.Path == "/public/solicitations/"+s {
						hits[s]++
						if s == tc.match {
							fmt.Fprint(w, matchListHTML(ref))
						} else {
							fmt.Fprint(w, emptyListHTML())
						}
						return
					}
				}
				if strings.HasPrefix(r.URL.Path, "/solicitations/open-bids/") {
					detailHits++
					fmt.Fprint(w, detailHTML(ref))
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			c := &httpx.Client{HTTP: &http.Client{Transport: &rewriteTransport{target: srv.URL}}}
			n, err := loadNotice(c, ref)

			if tc.match == "" {
				if err == nil || !strings.Contains(err.Error(), "no public notice matches") {
					t.Fatalf("got notice %+v err %v, want not-found error", n, err)
				}
				for _, s := range statuses {
					if hits[s] != 1 {
						t.Fatalf("status %q queried %d times, want 1: %v", s, hits[s], hits)
					}
				}
				if detailHits != 0 {
					t.Fatalf("detail fetched %d times, want 0", detailHits)
				}
				return
			}

			if err != nil {
				t.Fatalf("loadNotice: %v", err)
			}
			if n.ReferenceNumber != ref {
				t.Fatalf("ReferenceNumber %q, want %q", n.ReferenceNumber, ref)
			}
			if detailHits != 1 {
				t.Fatalf("detail fetched %d times, want 1", detailHits)
			}
			for _, s := range statuses {
				want := 0
				if indexOf(statuses, s) <= indexOf(statuses, tc.match) {
					want = 1
				}
				if hits[s] != want {
					t.Fatalf("status %q queried %d times, want %d: %v", s, hits[s], want, hits)
				}
			}
		})
	}
}

func indexOf(ss []string, v string) int {
	for i, s := range ss {
		if s == v {
			return i
		}
	}
	return -1
}
