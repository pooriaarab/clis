package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
)

// daysPer records 100 notices per day in the slice.
func daysPer(s Slice) (int, error) {
	a, err := time.Parse(harvestDateLayout, s.Start)
	if err != nil {
		return 0, err
	}
	b, err := time.Parse(harvestDateLayout, s.End)
	if err != nil {
		return 0, err
	}
	return 100 * (int(b.Sub(a).Hours()/24) + 1), nil
}

func TestNarrowSplitsOverflow(t *testing.T) {
	s := Slice{Status: "open", Start: "2020-01-01", End: "2020-01-31"}
	leaves, totals, err := plan([]Slice{s}, daysPer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaves) < 2 {
		t.Fatalf("3100-record month was not narrowed: %v", leaves)
	}
	cover := 0
	prev := ""
	for i, l := range leaves {
		n := totals[key(l)]
		if n > search.MaxReachable {
			t.Fatalf("leaf %v still reports %d", l, n)
		}
		a, _ := time.Parse(harvestDateLayout, l.Start)
		b, _ := time.Parse(harvestDateLayout, l.End)
		cover += int(b.Sub(a).Hours()/24) + 1
		if i > 0 && l.Start <= prev {
			t.Fatalf("leaves overlap or gap: %v", leaves)
		}
		prev = l.End
	}
	if leaves[0].Start != "2020-01-01" || prev != "2020-01-31" || cover != 31 {
		t.Fatalf("narrowing truncated the range: %v", leaves)
	}
}

func TestUnnarrowableSliceIncomplete(t *testing.T) {
	dir := t.TempDir()
	const rep = 5000
	count := func(Slice) (int, error) { return rep, nil }
	s := Slice{Status: "open", Start: "2020-01-01", End: "2020-01-01"}
	leaves, totals, err := plan([]Slice{s}, count, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaves) != 1 {
		t.Fatalf("single-day slice must stay one leaf: %v", leaves)
	}
	got := totals[key(leaves[0])]
	if !incomplete(leaves[0], got) {
		t.Fatalf("single-day %d-record slice must be incomplete", got)
	}
	e := Entry{Status: "open", Start: s.Start, End: s.End,
		Reported: got, Captured: capturedCount(got), Incomplete: true}
	_, manifest := harvestPaths(dir, "open")
	if err := saveManifest(manifest, Manifest{Slices: []Entry{e}}); err != nil {
		t.Fatal(err)
	}
	stored, err := loadManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Slices) != 1 {
		t.Fatalf("entries %v", stored.Slices)
	}
	if !stored.Slices[0].Incomplete || stored.Slices[0].Reported != rep || stored.Slices[0].Captured != search.MaxReachable {
		t.Fatalf("entry must carry both numbers: %+v", stored.Slices[0])
	}
}

func writeStore(t *testing.T, dir, status string, ids ...string) {
	t.Helper()
	store, _ := harvestPaths(dir, status)
	if err := os.MkdirAll(filepath.Dir(store), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(store, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, id := range ids {
		if err := appendRecord(f, search.Record{InternalID: id, DetailURL: "https://www.merx.com/d/" + id}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResumeSkipsStoredIDs(t *testing.T) {
	dir := t.TempDir()
	writeStore(t, dir, "open", "old-1", "old-2")
	store, manifest := harvestPaths(dir, "open")
	known, err := loadIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 2 || !known["old-1"] || !known["old-2"] {
		t.Fatalf("store ids %v", known)
	}
	done := []Entry{{Status: "open", Start: "2020-01-01", End: "2020-01-31",
		Reported: 50, Captured: 50, CompletedAt: time.Now().UTC().Format(time.RFC3339)}}
	if err := saveManifest(manifest, Manifest{Slices: done}); err != nil {
		t.Fatal(err)
	}
	skip := map[string]bool{}
	for _, e := range done {
		skip[e.Start+"\x00"+e.End] = true
	}
	seeds := []Slice{
		{Status: "open", Start: "2020-01-01", End: "2020-01-31"},
		{Status: "open", Start: "2020-02-01", End: "2020-02-29"},
	}
	leaves, _, err := plan(seeds, func(Slice) (int, error) { return 3, nil }, skip)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range leaves {
		if l.Start == "2020-01-01" {
			t.Fatal("ids already in the store were planned again")
		}
	}
	if len(leaves) != 1 || leaves[0].Start != "2020-02-01" {
		t.Fatalf("planned %v, want only February", leaves)
	}
	again, err := loadIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 || !again["old-1"] || !again["old-2"] {
		t.Fatalf("store ids changed: %v", again)
	}
}

func TestManifestRecordsCoverageWindow(t *testing.T) {
	dir := t.TempDir()
	_, manifest := harvestPaths(dir, "open")
	want := Manifest{Since: harvestEpoch, Until: "2020-06-30", Slices: []Entry{{
		Status: "open", Start: "2020-01-01", End: "2020-01-01",
		Reported: 5000, Captured: 1000, CompletedAt: "2020-01-02T00:00:00Z", Incomplete: true,
	}}}
	if err := saveManifest(manifest, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Since != want.Since || got.Until != want.Until || len(got.Slices) != 1 || got.Slices[0] != want.Slices[0] {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestHarvestFlagErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--status", "won"}, "unknown --status"},
		{[]string{"--since", "nope", "--dry-run"}, "invalid --since"},
		{[]string{"--since", "2021-01-01", "--until", "2020-01-01", "--dry-run"}, "after --until"},
	}
	for _, tc := range cases {
		cmd := harvestCmd()
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("args %v: error %v, want %s", tc.args, err, tc.want)
		}
	}
}

func TestHarvestSincePlansFrom2015(t *testing.T) {
	cmd := harvestCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--status", "closed", "--since", "2015-01-01", "--until", "2015-02-28", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "closed 2015-01-01") || strings.Contains(got, "2020-01") {
		t.Fatalf("got %q", got)
	}
}

func recs(ids ...string) []search.Record {
	out := make([]search.Record, len(ids))
	for i, id := range ids {
		out[i] = search.Record{InternalID: id, DetailURL: "https://www.merx.com/d/" + id}
	}
	return out
}

func TestSlicePagedToExhaustion(t *testing.T) {
	dir, pages, total := t.TempDir(), []int{}, 51
	sum, err := executePlan(harvestDeps{dir: dir, status: "open",
		slices: []Slice{{Status: "open", Start: "2020-01-01", End: "2020-01-31"}},
		count:  func(Slice) (int, error) { return total, nil },
		page: func(_ Slice, p int) (search.Page, error) {
			pages = append(pages, p)
			ids := make([]string, 0, search.PageSize)
			for i := (p-1)*search.PageSize + 1; i <= total && i <= p*search.PageSize; i++ {
				ids = append(ids, fmt.Sprintf("n-%d", i))
			}
			return search.Page{Total: total, Records: recs(ids...)}, nil
		},
		detail: func(string) error { return nil }})
	store, _ := harvestPaths(dir, "open")
	known, e2 := loadIDs(store)
	if err != nil || e2 != nil || len(pages) != 3 || pages[2] != 3 || sum.Captured != total || len(known) != total || !known["n-1"] || !known["n-51"] {
		t.Fatalf("pages %v cap %d store %d %v %v", pages, sum.Captured, len(known), err, e2)
	}
}

func TestResumeMidSliceLosesNothing(t *testing.T) {
	dir, fail, detailed := t.TempDir(), true, []string{}
	page := func(_ Slice, p int) (search.Page, error) {
		if p == 1 {
			return search.Page{Total: 26, Records: recs("a", "b")}, nil
		}
		if fail {
			return search.Page{}, fmt.Errorf("killed")
		}
		return search.Page{Total: 26, Records: recs("c", "d")}, nil
	}
	deps := harvestDeps{dir: dir, status: "open",
		slices: []Slice{{Status: "open", Start: "2020-01-01", End: "2020-01-31"}},
		count:  func(Slice) (int, error) { return 26, nil }, page: page,
		detail: func(u string) error { detailed = append(detailed, u); return nil }}
	if _, err := executePlan(deps); err == nil {
		t.Fatal("first run should stop mid-slice")
	}
	store, manifest := harvestPaths(dir, "open")
	known, _ := loadIDs(store)
	man, _ := loadManifest(manifest)
	if len(known) != 2 || !known["a"] || len(man.Slices) != 0 {
		t.Fatalf("after kill store %v manifest %+v", known, man.Slices)
	}
	fail, detailed, deps.resume = false, nil, true
	sum, err := executePlan(deps)
	known, _ = loadIDs(store)
	if err != nil || len(detailed) != 2 || detailed[0] != "https://www.merx.com/d/c" || sum.Captured != 2 || len(known) != 4 || !known["c"] || !known["d"] {
		t.Fatalf("resume details %v cap %d store %v %v", detailed, sum.Captured, known, err)
	}
}

func TestDetailFailureIsRetriedOnResume(t *testing.T) {
	dir, failB := t.TempDir(), true
	page := func(_ Slice, _ int) (search.Page, error) {
		return search.Page{Total: 2, Records: recs("a", "b")}, nil
	}
	detail := func(u string) error {
		if failB && strings.HasSuffix(u, "/b") {
			return fmt.Errorf("boom")
		}
		return nil
	}
	deps := harvestDeps{dir: dir, status: "open",
		slices: []Slice{{Status: "open", Start: "2020-01-01", End: "2020-01-31"}},
		count:  func(Slice) (int, error) { return 2, nil }, page: page, detail: detail}
	sum, err := executePlan(deps)
	if err != nil {
		t.Fatal(err)
	}
	store, manifest := harvestPaths(dir, "open")
	known, _ := loadIDs(store)
	man, _ := loadManifest(manifest)
	if sum.Captured != 1 || len(known) != 1 || !known["a"] || len(man.Slices) != 0 {
		t.Fatalf("first run captured %d store %v manifest %+v (a failed detail must not mark the slice done)", sum.Captured, known, man.Slices)
	}
	failB, deps.resume = false, true
	sum, err = executePlan(deps)
	known, _ = loadIDs(store)
	man, _ = loadManifest(manifest)
	if err != nil || sum.Captured != 1 || len(known) != 2 || !known["b"] || len(man.Slices) != 1 {
		t.Fatalf("resume captured %d store %v manifest %+v err %v (the failed record must be retried)", sum.Captured, known, man.Slices, err)
	}
}

func TestCSRFTokenFromPage(t *testing.T) {
	raw, err := os.ReadFile("testdata/search_form.golden")
	if err != nil {
		t.Fatal(err)
	}
	const results = `<span class="simpleSolResultsNumResults">1</span>` +
		`<table><tr class="mets-table-row"><td><a id="searchResultSol_notice_7" href="/d/7" class="solicitation-link"><span class="rowTitle">T</span></a></td></tr></table>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write(raw)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("_csrf") != "SEARCH-TOKEN-9f3" {
			t.Errorf("csrf field %q", r.Form.Get("_csrf"))
		}
		if r.Header.Get("X-CSRF-TOKEN") != "SEARCH-TOKEN-9f3" {
			t.Errorf("csrf header %q", r.Header.Get("X-CSRF-TOKEN"))
		}
		// Witness that the body came from the form that holds the date fields.
		// That form has no _csrf. The decoy forms share its action.
		if r.Form.Get("publishedDate.timeZoneOffset") != "-14400000" || r.Form.Get("decoyFrm") != "" || r.Form.Get("decoyCommand") != "" || r.Form.Get("decoyCriteria") != "" {
			t.Errorf("posted fields %v", r.Form)
		}
		w.Write([]byte(results))
	}))
	t.Cleanup(srv.Close)
	c := &httpx.Client{HTTP: srv.Client()}
	form, header, err := loadSearchForm(c, srv.URL)
	if err != nil || form.Get("_csrf") != "SEARCH-TOKEN-9f3" || header != "X-CSRF-TOKEN" {
		t.Fatalf("csrf %q header %q err %v (token must come from the page)", form.Get("_csrf"), header, err)
	}
	if form.Get("publishedDate.timeZoneOffset") != "-14400000" || form.Get("decoyFrm") != "" {
		t.Fatalf("form fields %v", form)
	}
	pg, err := postSlice(c, srv.URL, form, header, "OPEN", Slice{Start: "2020-01-01", End: "2020-01-15"}, 2)
	if err != nil || pg.Total != 1 || len(pg.Records) != 1 || pg.Records[0].InternalID != "7" {
		t.Fatalf("page %+v err %v", pg, err)
	}
}

func TestSearchFormSerializesLikeBrowser(t *testing.T) {
	raw, err := os.ReadFile("testdata/search_form.golden")
	if err != nil {
		t.Fatal(err)
	}
	const results = `<span class="simpleSolResultsNumResults">1</span>` +
		`<table><tr class="mets-table-row"><td><a id="searchResultSol_notice_7" href="/d/7" class="solicitation-link"><span class="rowTitle">T</span></a></td></tr></table>`
	var posted url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write(raw)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			return
		}
		posted = r.PostForm
		w.Write([]byte(results))
	}))
	t.Cleanup(srv.Close)
	c := &httpx.Client{HTTP: srv.Client()}
	form, header, err := loadSearchForm(c, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(form["status"], []string{"OPEN"}) {
		t.Fatalf("status = %q, want the checked OPEN radio", form["status"])
	}
	if _, ok := form["publishedDate.dateType"]; ok {
		t.Fatalf("unchecked date type was submitted: %q", form["publishedDate.dateType"])
	}
	if _, ok := form["g_617"]; ok {
		t.Fatalf("unchecked checkbox was submitted: %q", form["g_617"])
	}
	if _, ok := form["publishedDate.within"]; ok {
		t.Fatalf("disabled select was submitted: %q", form["publishedDate.within"])
	}
	if form.Get("dynamicSort[0].fieldName") != "publicationDate" {
		t.Fatalf("sort field = %q, want the selected option", form.Get("dynamicSort[0].fieldName"))
	}
	if !slices.Equal(form["dynamicSort[1].fieldName"], []string{""}) {
		t.Fatalf("unset sort = %q, want the first option", form["dynamicSort[1].fieldName"])
	}
	if !slices.Equal(form["selectedSavedSearch"], []string{""}) {
		t.Fatalf("saved search = %q, want the empty option value", form["selectedSavedSearch"])
	}
	if !slices.Equal(form["_publishedDate.dateType"], []string{"on", "on", "on", "on"}) {
		t.Fatalf("spring marker = %q, want four values", form["_publishedDate.dateType"])
	}
	if !slices.Equal(form["keywords"], []string{"", " bridge"}) {
		t.Fatalf("keywords = %q", form["keywords"])
	}
	if form.Get("regionId") != "[null-null]{}" {
		t.Fatalf("regionId = %q", form.Get("regionId"))
	}
	if _, err := postSlice(c, srv.URL, form, header, "CLOSED", Slice{Start: "2026-09-20", End: "2026-09-22"}, 3); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(posted["status"], []string{"CLOSED"}) {
		t.Fatalf("posted status = %q, want the overlay to replace OPEN", posted["status"])
	}
	if !slices.Equal(posted["publishedDate.dateType"], []string{"RANGE"}) {
		t.Fatalf("posted date type = %q, want RANGE once", posted["publishedDate.dateType"])
	}
	if !slices.Equal(posted["publishedDate.localRangeStart"], []string{"2026-09-20"}) || !slices.Equal(posted["publishedDate.localRangeEnd"], []string{"2026-09-22"}) {
		t.Fatalf("posted range start %q end %q", posted["publishedDate.localRangeStart"], posted["publishedDate.localRangeEnd"])
	}
	if !slices.Equal(posted["pageNumber"], []string{"3"}) {
		t.Fatalf("posted page = %q", posted["pageNumber"])
	}
	if !slices.Equal(posted["_publishedDate.dateType"], []string{"on", "on", "on", "on"}) {
		t.Fatalf("posted spring marker = %q", posted["_publishedDate.dateType"])
	}
	if _, ok := form["publishedDate.dateType"]; ok || !slices.Equal(form["status"], []string{"OPEN"}) {
		t.Fatalf("overlay mutated the cached form: %v", form)
	}
}

func TestCSRFTokenPrefersMeta(t *testing.T) {
	const page = `<html><head>` +
		`<meta name="_csrf" content="META-TOKEN">` +
		`<meta name="_csrf_header" content="X-Page-CSRF">` +
		`</head><body>` +
		`<form action="/private/supplier/solicitations/search">` +
		`<input type="hidden" name="_csrf" value="INPUT-TOKEN">` +
		`<input type="hidden" name="decoy" value="1">` +
		`</form>` +
		`<form action="/private/supplier/solicitations/search">` +
		`<input type="radio" name="status" value="OPEN">` +
		`<input name="publishedDate.dateType" value="ANYTIME">` +
		`<input name="publishedDate.timeZoneOffset" value="0">` +
		`</form></body></html>`
	c, portal := csrfPage(t, page)
	form, header, err := loadSearchForm(c, portal)
	if err != nil || form.Get("_csrf") != "META-TOKEN" || header != "X-Page-CSRF" || form.Get("decoy") != "" {
		t.Fatalf("csrf %q header %q fields %v err %v", form.Get("_csrf"), header, form, err)
	}
}

func TestCSRFTokenFallsBackToInput(t *testing.T) {
	const page = `<html><head><meta name="_csrf_header" content="X-CSRF-TOKEN"></head><body>` +
		`<form action="/private/supplier/solicitations/search">` +
		`<input type="hidden" name="_csrf" value="INPUT-TOKEN">` +
		`</form>` +
		`<form action="/private/supplier/solicitations/search">` +
		`<input type="radio" name="status" value="OPEN">` +
		`<input name="publishedDate.dateType" value="ANYTIME">` +
		`</form></body></html>`
	c, portal := csrfPage(t, page)
	form, header, err := loadSearchForm(c, portal)
	if err != nil || form.Get("_csrf") != "INPUT-TOKEN" || header != "X-CSRF-TOKEN" {
		t.Fatalf("csrf %q header %q err %v", form.Get("_csrf"), header, err)
	}
}

func TestCSRFTokenMissing(t *testing.T) {
	const page = `<html><body><form action="/private/supplier/solicitations/search">` +
		`<input type="radio" name="status" value="OPEN">` +
		`<input name="publishedDate.dateType" value="ANYTIME">` +
		`</form></body></html>`
	c, portal := csrfPage(t, page)
	_, _, err := loadSearchForm(c, portal)
	if err == nil || !strings.Contains(err.Error(), `meta name="_csrf"`) || !strings.Contains(err.Error(), "no _csrf input") {
		t.Fatalf("err %v", err)
	}
}

func csrfPage(t *testing.T, page string) (*httpx.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("posted without a token: %s", r.Method)
		}
		w.Write([]byte(page))
	}))
	t.Cleanup(srv.Close)
	return &httpx.Client{HTTP: srv.Client()}, srv.URL
}

func noFollow(t *testing.T, h http.HandlerFunc) (*httpx.Client, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	hc := srv.Client()
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &httpx.Client{HTTP: hc}, srv.URL
}

func TestPostSliceFollowsRedirect(t *testing.T) {
	const page = `<span class="simpleSolResultsNumResults">1</span>` +
		`<table><tr class="mets-table-row"><td><a id="searchResultSol_notice_7" href="/d/7" class="solicitation-link"><span class="rowTitle">T</span></a></td></tr></table>`
	for _, code := range []int{http.StatusFound, http.StatusSeeOther} {
		var method, host, uri string
		c, portal := noFollow(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == privateSearchPath {
				w.Header().Set("Location", "results?page=1")
				w.WriteHeader(code)
				return
			}
			method, host, uri = r.Method, r.Host, r.URL.RequestURI()
			w.Write([]byte(page))
		})
		pg, err := postSlice(c, portal, url.Values{"_csrf": {"tok"}}, "", "OPEN", Slice{Start: "2026-09-20", End: "2026-09-22"}, 1)
		u, _ := url.Parse(portal)
		if err != nil || pg.Total != 1 || len(pg.Records) != 1 || pg.Records[0].InternalID != "7" || method != http.MethodGet || host != u.Host || uri != "/private/supplier/solicitations/results?page=1" {
			t.Fatalf("HTTP %d page %+v err %v %s %s %s", code, pg, err, method, host, uri)
		}
	}
}

func TestPostSliceLoginIsSessionError(t *testing.T) {
	c, portal := noFollow(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == privateSearchPath {
			w.Header().Set("Location", "/public/authentication/login")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.Write([]byte(`<span class="simpleSolResultsNumResults">0</span>`))
	})
	pg, err := postSlice(c, portal, url.Values{"_csrf": {"tok"}}, "", "OPEN", Slice{Start: "2026-09-20", End: "2026-09-22"}, 1)
	if err == nil || !strings.Contains(err.Error(), "session") || pg.Total != 0 || len(pg.Records) != 0 {
		t.Fatalf("page %+v err %v", pg, err)
	}
}

func TestPostSliceTooManyRedirects(t *testing.T) {
	if maxSearchRedirects != 10 {
		t.Fatalf("cap %d, want 10", maxSearchRedirects)
	}
	seen := 0
	c, portal := noFollow(t, func(w http.ResponseWriter, r *http.Request) {
		seen++
		if seen > maxSearchRedirects {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Location", "/hop")
		w.WriteHeader(http.StatusFound)
	})
	pg, err := postSlice(c, portal, url.Values{"_csrf": {"tok"}}, "", "OPEN", Slice{Start: "2026-09-20", End: "2026-09-22"}, 1)
	if err == nil || seen != maxSearchRedirects || !strings.Contains(err.Error(), "too many redirects") {
		t.Fatalf("page %+v err %v requests %d", pg, err, seen)
	}
}

func TestSignalHandlerLogsOut(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/logout") {
			hits++
		}
	}))
	old, exited := exitFunc, make(chan struct{})
	exitFunc = func(int) { close(exited) }
	t.Cleanup(func() { srv.Close(); exitFunc = old; signal.Reset(os.Interrupt, syscall.SIGTERM) })
	installLogout(&sync.Once{}, &httpx.Client{HTTP: srv.Client()}, srv.URL)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("signal handler did not exit")
	}
	if hits != 1 {
		t.Fatalf("logout requests %d, want 1", hits)
	}
}
