package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
)

// Public MERX category ids are 10004-10055 (52 codes). Locations are
// the 13 provinces/territories plus All of Canada (38000).
var merxLocations = []string{
	"325", "331", "337", "343", "349", "355", "361",
	"367", "373", "379", "385", "391", "397", "38000",
}

func merxCategories() []string {
	out := make([]string, 52)
	for i := range out {
		out[i] = strconv.Itoa(10004 + i)
	}
	return out
}

// facetSlices is the 52×14 public search grid. Date RANGE cannot
// match notices with no publishedDate; these facets still can.
func facetSlices(status string) []Slice {
	cats := merxCategories()
	out := make([]Slice, 0, len(cats)*len(merxLocations))
	for _, c := range cats {
		for _, loc := range merxLocations {
			out = append(out, Slice{Status: status, Category: c, Location: loc})
		}
	}
	return out
}

func defaultUndated() *Undated { return &Undated{} }

func mergeUndated(old, upd *Undated) *Undated {
	if old == nil {
		return upd
	}
	if upd == nil {
		return old
	}
	out := *old
	out.CoveredByDates = false
	if upd.Looked {
		out.Looked = true
		out.Exists = upd.Exists
		out.Reported = upd.Reported
	}
	if upd.FacetPassRan {
		out.FacetPassRan = true
	}
	return &out
}

func sliceLabel(s Slice) string {
	if s.Category != "" || s.Location != "" {
		return s.Category + "/" + s.Location
	}
	return s.Start + ".." + s.End
}

func noDate(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || strings.EqualFold(s, "Not Available") || strings.EqualFold(s, "N/A")
}

// publicPage GETs one public list page through search.Query / search.Parse.
func publicPage(c *httpx.Client, s Slice, page int) (search.Page, error) {
	q := search.Query{Status: s.Status, Category: s.Category, Location: s.Location, Page: page}
	u, err := q.URL()
	if err != nil {
		return search.Page{}, err
	}
	body, _, err := getOK(c, u)
	if err != nil {
		return search.Page{}, err
	}
	return search.Parse(bytes.NewReader(body))
}

// probeUndated asks the public list, oldest publicationDate first.
// The portal has no undated-only filter, so Reported is set only when
// the exact count is visible on page 1.
func probeUndated(c *httpx.Client, status string) (bool, *int, error) {
	q := search.Query{Status: status, SortBy: "publicationDate", SortDirection: "ASC", Page: 1}
	u, err := q.URL()
	if err != nil {
		return false, nil, err
	}
	body, _, err := getOK(c, u)
	if err != nil {
		return false, nil, err
	}
	pg, err := search.Parse(bytes.NewReader(body))
	if err != nil {
		return false, nil, err
	}
	exists, reported := undatedFromPage(pg)
	return exists, reported, nil
}

func undatedFromPage(pg search.Page) (bool, *int) {
	n := 0
	for _, r := range pg.Records {
		if noDate(r.Published) && noDate(r.Closing) {
			n++
		}
	}
	if n == 0 {
		z := 0
		return false, &z
	}
	if n == len(pg.Records) && pg.Total <= search.PageSize {
		t := pg.Total
		return true, &t
	}
	return true, nil
}

// runUndated walks the facet grid on the public lists. It never wipes
// the date-pass store: a notice already stored is skipped by id.
func runUndated(status string) error {
	c, err := httpx.New()
	if err != nil {
		return err
	}
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	exists, reported, err := probeUndated(c, status)
	if err != nil {
		return err
	}
	gap := &Undated{Looked: true, Exists: &exists, Reported: reported, FacetPassRan: true}
	sum, err := executePlan(harvestDeps{dir: dir, status: status, resume: true, keepStore: true,
		slices: facetSlices(status), undated: gap,
		count: func(s Slice) (int, error) {
			pg, err := publicPage(c, s, 1)
			return pg.Total, err
		},
		page:   func(s Slice, p int) (search.Page, error) { return publicPage(c, s, p) },
		detail: func(u string) error { return fetchDetail(c, u) }})
	if err != nil {
		return err
	}
	if flagJSON {
		return emit(sum)
	}
	fmt.Printf("%s undated: %d slices, reported %d, captured %d, incomplete %d\n", sum.Status, sum.Slices, sum.Reported, sum.Captured, sum.Incomplete)
	return nil
}
