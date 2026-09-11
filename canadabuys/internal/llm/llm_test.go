package llm

import (
	"canadabuys-cli/internal/score"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
}

func TestClampShapeLeavesProductAlone(t *testing.T) {
	e := Enrichment{Shape: "Product", ProductFit: 90}
	clampShape(&e)
	if e.Shape != "product" || e.ProductFit != 90 {
		t.Fatalf("%+v", e)
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
