package llm

import (
	"canadabuys-cli/internal/score"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNoKeyNamesBothVars(t *testing.T) {
	t.Setenv("CANADABUYS_LLM_API_KEY", "")
	t.Setenv("CEREBRAS_API_KEY", "")
	_, _, err := (&Client{CacheDir: t.TempDir()}).Enrich(nil, "m")
	if err == nil || !strings.Contains(err.Error(), "CANADABUYS_LLM_API_KEY") || !strings.Contains(err.Error(), "CEREBRAS_API_KEY") {
		t.Fatalf("empty key: %v", err)
	}
}

func TestPrefersGeneralKeyEnv(t *testing.T) {
	t.Setenv("CANADABUYS_LLM_API_KEY", "general")
	t.Setenv("CEREBRAS_API_KEY", "cerebras")
	c := &Client{CacheDir: t.TempDir()}
	if _, _, err := c.Enrich(nil, "m"); err != nil {
		t.Fatal(err)
	}
	if c.Key != "general" {
		t.Fatalf("key=%q", c.Key)
	}
}

func TestKeyRejectsPathTraversalReference(t *testing.T) {
	dir := t.TempDir()
	c := &Client{CacheDir: dir}
	o := score.Opportunity{Reference: "../../../../tmp/evil", Title: "t", Buyer: "b", Description: "d"}
	name := c.key(o, "m")
	if strings.ContainsAny(name, "/\\") {
		t.Fatalf("cache filename escapes CacheDir: %q", name)
	}
	if filepath.Dir(filepath.Join(dir, name)) != dir {
		t.Fatalf("joined path leaves CacheDir: %q", filepath.Join(dir, name))
	}
}

func TestCacheHitIssuesNoRequest(t *testing.T) {
	dir := t.TempDir()
	o := score.Opportunity{Reference: "r1", Title: "t", Buyer: "b", Description: "d"}
	c := &Client{Key: "dummy", CacheDir: dir}
	want := Enrichment{Reference: "r1", Deliverable: 80, ProductFit: 60, Shape: "product", Thesis: "x", Category: "y"}
	raw, _ := json.Marshal(want)
	if err := os.WriteFile(filepath.Join(dir, c.key(o, "m")), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	got, skip, err := c.Enrich([]score.Opportunity{o}, "m")
	if err != nil || skip != 0 || len(got) != 1 || got[0] == nil || *got[0] != want {
		t.Fatalf("got=%v skip=%d err=%v", got, skip, err)
	}
}

func TestInstructionAsksForProductFit(t *testing.T) {
	low := strings.ToLower(instruction)
	for _, w := range []string{"product fit", "resale", "staffing", "training", "deliverable", "productfit"} {
		if !strings.Contains(low, w) {
			t.Fatalf("instruction missing %q", w)
		}
	}
	if strings.Contains(instruction, "Rate each procurement notice for a small software team") {
		t.Fatal("old fulfilment prompt is still in use")
	}
	if strings.Contains(low, "buildability") {
		t.Fatal("buildability must not remain in the prompt")
	}
}

func TestCacheKeyIncludesInstructionVersion(t *testing.T) {
	o := score.Opportunity{Reference: "r1", Title: "t", Buyer: "b", Description: "d"}
	got := (&Client{}).key(o, "m")
	sum := sha256.Sum256([]byte(instructionVersion + "\x00" + "m" + "\x00" + instruction + "\x00" + promptFor(o)))
	want := "llm-r1-" + hex.EncodeToString(sum[:])[:16] + ".json"
	if got != want {
		t.Fatalf("key=%q want %q", got, want)
	}
	old := sha256.Sum256([]byte("m" + "\x00" + instruction + "\x00" + promptFor(o)))
	if strings.Contains(got, hex.EncodeToString(old[:])[:16]) {
		t.Fatal("cache key omitted instructionVersion")
	}
}

func TestEnrichmentJSONSplitsScores(t *testing.T) {
	raw, err := json.Marshal(Enrichment{Reference: "r", Deliverable: 100, ProductFit: 10, Shape: "resale", Thesis: "licences", Category: "software"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["buildability"]; ok {
		t.Fatal("buildability must not be serialized")
	}
	for _, k := range []string{"deliverable", "productFit", "shape"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s in %s", k, raw)
		}
	}
}

func TestOldBuildabilityCacheIsIgnored(t *testing.T) {
	dir := t.TempDir()
	o := score.Opportunity{Reference: "r1", Title: "t", Buyer: "b", Description: "d"}
	c := &Client{Key: "dummy", CacheDir: dir}
	old := []byte(`{"reference":"r1","buildability":100,"thesis":"licences","category":"software"}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, c.key(o, "m")), old, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.cached(o, "m"); ok {
		t.Fatal("cache hit on old buildability-only entry")
	}
}

var pilotFixtures = []struct {
	title string
	shape string
}{
	{"Exceed TurboX Premium Licences and Maintenance for SSC", "resale"},
	{"Request for Qualifications - Web Development Consultants", "staffing"},
	{"Online ArcGIS training", "training"},
}

func TestPilotFixturesAreNonProductAndNotTop(t *testing.T) {
	product := Enrichment{Reference: "product", ProductFit: 90, Shape: "product", Thesis: "case management"}
	for _, f := range pilotFixtures {
		e := Enrichment{Reference: f.shape, Deliverable: 100, ProductFit: 100, Shape: f.shape, Thesis: f.title}
		clampShape(&e)
		if e.Shape != f.shape {
			t.Fatalf("%q shape=%q", f.title, e.Shape)
		}
		if e.ProductFit > 25 {
			t.Fatalf("%q productFit %.0f after clamp", f.title, e.ProductFit)
		}
		if e.ProductFit >= product.ProductFit {
			t.Fatalf("%q ranked over a product", f.title)
		}
	}
	if !strings.Contains(instruction, pilotFixtures[0].title) || !strings.Contains(instruction, pilotFixtures[1].title) || !strings.Contains(instruction, pilotFixtures[2].title) {
		t.Fatal("instruction dropped the pilot fixtures")
	}
}

func TestMalformedRatingIsSkipped(t *testing.T) {
	if validReply(Enrichment{Shape: "consulting", ProductFit: 1000, Deliverable: 80}) {
		t.Fatal(`{"shape":"consulting","productFit":1000} must be skipped`)
	}
	// A model that answers 1000 did not understand the question: skip,
	// do not clamp, and do not cache the thesis. This must hold on the
	// concurrent path as well as the unit check above.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := json.Marshal(map[string]any{"results": []any{
			map[string]any{"reference": "r00", "deliverable": 80, "productFit": 1000, "shape": "consulting", "thesis": "bad", "category": "x"},
		}})
		env, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
		w.Header().Set("Content-Type", "application/json")
		w.Write(env)
	}))
	defer srv.Close()
	dir := t.TempDir()
	c := &Client{Key: "dummy", CacheDir: dir, BaseURL: srv.URL, Concurrency: 8}
	got, skip, err := c.Enrich(testNotices(1), "m")
	if err != nil {
		t.Fatal(err)
	}
	if skip != 1 || got[0] != nil {
		t.Fatalf("got=%v skip=%d (want skipped, not clamped)", got, skip)
	}
	if _, ok := c.cached(testNotices(1)[0], "m"); ok {
		t.Fatal("malformed reply must not be cached")
	}
}

func TestClampShapeLeavesProductAlone(t *testing.T) {
	e := Enrichment{Shape: "Product", ProductFit: 90}
	clampShape(&e)
	if e.Shape != "product" || e.ProductFit != 90 {
		t.Fatalf("%+v", e)
	}
}

// fakeProvider serves one result per "reference: ..." line in the prompt,
// the way the real provider batches ten notices per request. refsIn parses
// the references back out of the request body.
func refsIn(t *testing.T, r *http.Request) []string {
	t.Helper()
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	var refs []string
	for _, line := range strings.Split(req.Messages[len(req.Messages)-1].Content, "\n") {
		if ref, ok := strings.CutPrefix(line, "reference: "); ok {
			refs = append(refs, strings.TrimSpace(ref))
		}
	}
	return refs
}

func replyWith(t *testing.T, w http.ResponseWriter, refs []string) {
	t.Helper()
	results := make([]Enrichment, 0, len(refs))
	for _, ref := range refs {
		results = append(results, Enrichment{Reference: ref, Deliverable: 80, ProductFit: 60, Shape: "product", Thesis: "thesis for " + ref, Category: "software"})
	}
	content, _ := json.Marshal(map[string]any{"results": results})
	env, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
	w.Header().Set("Content-Type", "application/json")
	w.Write(env)
}

func testNotices(n int) []score.Opportunity {
	out := make([]score.Opportunity, n)
	for i := range out {
		out[i] = score.Opportunity{Reference: fmt.Sprintf("r%02d", i), Title: "t", Buyer: "b", Description: "d"}
	}
	return out
}

// Serial and concurrent runs must fill the same pre-sized slots: order and
// reference set identical however the batches complete.
func TestConcurrentMatchesSerial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyWith(t, w, refsIn(t, r))
	}))
	defer srv.Close()
	var serial, parallel []*Enrichment
	var serialSkip, parallelSkip int
	for _, tc := range []struct {
		name        string
		concurrency int
		dst         *[]*Enrichment
		skip        *int
	}{{"serial", 1, &serial, &serialSkip}, {"parallel", 8, &parallel, &parallelSkip}} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{Key: "dummy", CacheDir: t.TempDir(), BaseURL: srv.URL, Concurrency: tc.concurrency}
			got, skip, err := c.Enrich(testNotices(25), "m")
			if err != nil {
				t.Fatal(err)
			}
			*tc.dst, *tc.skip = got, skip
		})
	}
	if serialSkip != 0 || parallelSkip != 0 {
		t.Fatalf("skip serial=%d parallel=%d", serialSkip, parallelSkip)
	}
	if len(serial) != len(parallel) {
		t.Fatalf("len serial=%d parallel=%d", len(serial), len(parallel))
	}
	for i := range serial {
		if (serial[i] == nil) != (parallel[i] == nil) {
			t.Fatalf("slot %d nil mismatch", i)
		}
		if serial[i] != nil && serial[i].Reference != parallel[i].Reference {
			t.Fatalf("slot %d ref %q vs %q", i, serial[i].Reference, parallel[i].Reference)
		}
	}
}

// A 429 must be retried, not counted as a skip: the notices still land.
func TestRateLimitIsRetriedNotSkipped(t *testing.T) {
	old := retryBaseDelay
	retryBaseDelay = time.Millisecond
	defer func() { retryBaseDelay = old }()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		replyWith(t, w, refsIn(t, r))
	}))
	defer srv.Close()
	c := &Client{Key: "dummy", CacheDir: t.TempDir(), BaseURL: srv.URL, Concurrency: 4}
	got, skip, err := c.Enrich(testNotices(10), "m")
	if err != nil {
		t.Fatal(err)
	}
	if skip != 0 {
		t.Fatalf("retried batch counted as skip: %d", skip)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d want 3", calls.Load())
	}
	for i, e := range got {
		if e == nil || e.Reference != fmt.Sprintf("r%02d", i) {
			t.Fatalf("slot %d: %+v", i, e)
		}
	}
}

// A 400 is a real rejection: it fails immediately without burning retries.
func TestBadRequestFailsFast(t *testing.T) {
	old := retryBaseDelay
	retryBaseDelay = time.Millisecond
	defer func() { retryBaseDelay = old }()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "bad model", http.StatusBadRequest)
	}))
	defer srv.Close()
	c := &Client{Key: "dummy", CacheDir: t.TempDir(), BaseURL: srv.URL, Concurrency: 4}
	if _, _, err := c.Enrich(testNotices(10), "m"); err == nil {
		t.Fatal("want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d want 1", calls.Load())
	}
}

// A permanently failing batch must not abandon its siblings: the good
// batches still fill their slots and the cache, and the error is reported.
func TestFailingBatchDoesNotAbandonOthers(t *testing.T) {
	old := retryBaseDelay
	retryBaseDelay = time.Millisecond
	defer func() { retryBaseDelay = old }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refs := refsIn(t, r)
		for _, ref := range refs {
			if ref == "r00" {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
		}
		replyWith(t, w, refs)
	}))
	defer srv.Close()
	dir := t.TempDir()
	c := &Client{Key: "dummy", CacheDir: dir, BaseURL: srv.URL, Concurrency: 8}
	got, _, err := c.Enrich(testNotices(20), "m")
	if err == nil {
		t.Fatal("want the failing batch reported")
	}
	// Batch one (r00-r09) failed; batch two (r10-r19) must be intact.
	for i := 10; i < 20; i++ {
		if got[i] == nil || got[i].Reference != fmt.Sprintf("r%02d", i) {
			t.Fatalf("slot %d abandoned: %+v", i, got[i])
		}
	}
	ents, _ := os.ReadDir(dir)
	var cached int
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), ".") {
			cached++
		}
	}
	if cached != 10 {
		t.Fatalf("cached=%d want 10", cached)
	}
}

// A malformed reply costs the whole batch as skips, and the skip count
// stays exact when batches fail concurrently.
func TestMalformedBatchCountsSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()
	c := &Client{Key: "dummy", CacheDir: t.TempDir(), BaseURL: srv.URL, Concurrency: 8}
	got, skip, err := c.Enrich(testNotices(25), "m")
	if err != nil {
		t.Fatal(err)
	}
	if skip != 25 {
		t.Fatalf("skip=%d want 25", skip)
	}
	for i, e := range got {
		if e != nil {
			t.Fatalf("slot %d: %+v", i, e)
		}
	}
}

// Concurrent cache writes must each leave a complete file behind.
func TestConcurrentCacheWritesAreComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyWith(t, w, refsIn(t, r))
	}))
	defer srv.Close()
	notices := testNotices(30)
	c := &Client{Key: "dummy", CacheDir: t.TempDir(), BaseURL: srv.URL, Concurrency: 16}
	if _, _, err := c.Enrich(notices, "m"); err != nil {
		t.Fatal(err)
	}
	sort.Slice(notices, func(i, j int) bool { return notices[i].Reference < notices[j].Reference })
	for _, n := range notices {
		e, ok := c.cached(n, "m")
		if !ok || e.Reference != n.Reference || e.Thesis == "" {
			t.Fatalf("ref %s: ok=%t %+v", n.Reference, ok, e)
		}
	}
}

func TestPilotResponseRoundTrip(t *testing.T) {
	doc := struct {
		Results []Enrichment `json:"results"`
	}{Results: []Enrichment{
		{Reference: "ssc-turbox", Deliverable: 95, ProductFit: 8, Shape: "resale", Thesis: "third-party licences", Category: "software"},
		{Reference: "rfq-webdev", Deliverable: 90, ProductFit: 12, Shape: "staffing", Thesis: "staff augmentation", Category: "consulting"},
		{Reference: "arcgis-train", Deliverable: 88, ProductFit: 10, Shape: "training", Thesis: "course delivery", Category: "training"},
	}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Results []Enrichment `json:"results"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 3 {
		t.Fatalf("results=%d", len(got.Results))
	}
	want := []string{"resale", "staffing", "training"}
	for i, e := range got.Results {
		if e.Shape != want[i] || e.ProductFit >= 50 {
			t.Fatalf("%+v", e)
		}
		if _, ok := map[string]bool{"resale": true, "staffing": true, "training": true}[e.Shape]; !ok {
			t.Fatalf("shape %q", e.Shape)
		}
	}
}
