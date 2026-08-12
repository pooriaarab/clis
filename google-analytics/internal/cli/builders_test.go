package cli

import (
	"encoding/json"
	"testing"
)

func TestReportRequestBuildsTypedGA4Body(t *testing.T) {
	req := reportRequest("sessions,totalRevenue", "date,sessionDefaultChannelGroup", "28daysAgo", "yesterday", 7)
	if len(req.Metrics) != 2 || req.Metrics[1].Name != "totalRevenue" {
		t.Fatalf("metrics not parsed: %#v", req.Metrics)
	}
	if len(req.Dimensions) != 2 || req.Dimensions[0].Name != "date" {
		t.Fatalf("dimensions not parsed: %#v", req.Dimensions)
	}
	if req.Limit != "7" || req.DateRanges[0].StartDate != "28daysAgo" {
		t.Fatalf("bad date/limit: %#v", req)
	}
}
func TestAddOrderAndFilter(t *testing.T) {
	req := reportRequest("sessions", "date", "", "", 0)
	addOrder(&req, "-sessions")
	if len(req.OrderBys) != 1 || !req.OrderBys[0].Desc || req.OrderBys[0].Metric.MetricName != "sessions" {
		t.Fatalf("bad order: %#v", req.OrderBys)
	}
	if err := addRawDimensionFilter(&req, `{"filter":{"fieldName":"country"}}`); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(req)
	if !json.Valid(b) || string(b) == "" {
		t.Fatalf("bad json: %s", b)
	}
}
func TestFunnelRequestBuildsEventSteps(t *testing.T) {
	req := funnelRequest("view_item,add_to_cart", "30daysAgo", "yesterday")
	if len(req.Funnel.Steps) != 2 {
		t.Fatalf("steps=%d", len(req.Funnel.Steps))
	}
	if req.Funnel.Steps[1].FilterExpression.FunnelEventFilter.EventName != "add_to_cart" {
		t.Fatalf("bad step: %#v", req.Funnel.Steps[1])
	}
}
func TestCohortRequestBuildsCohortSpec(t *testing.T) {
	req := cohortRequest("weekly", 6, "2026-06-30")
	if req.CohortSpec == nil {
		t.Fatal("cohortSpec nil")
	}
	if len(req.DateRanges) != 0 {
		t.Fatalf("cohort request must not set dateRanges: %#v", req.DateRanges)
	}
	cs := req.CohortSpec
	if len(cs.Cohorts) != 6 || cs.CohortsRange.Granularity != "WEEKLY" || cs.CohortsRange.EndOffset != 5 {
		t.Fatalf("bad cohortsRange/count: %#v", cs)
	}
	// cohort_0 is the most recent 7-day bucket ending at end.
	if cs.Cohorts[0].DateRange.EndDate != "2026-06-30" || cs.Cohorts[0].DateRange.StartDate != "2026-06-24" {
		t.Fatalf("bad bucket 0: %#v", cs.Cohorts[0].DateRange)
	}
	// Buckets are 7 days apart.
	if cs.Cohorts[1].DateRange.EndDate != "2026-06-23" {
		t.Fatalf("bucket 1 not 7 days back: %#v", cs.Cohorts[1].DateRange)
	}
	if cs.Cohorts[0].Dimension != "firstSessionDate" {
		t.Fatalf("bad cohort dimension: %q", cs.Cohorts[0].Dimension)
	}
	dims := map[string]bool{}
	for _, d := range req.Dimensions {
		dims[d.Name] = true
	}
	if !dims["cohort"] || !dims["cohortNthWeek"] {
		t.Fatalf("missing cohort dims: %#v", req.Dimensions)
	}
	if len(req.Metrics) != 1 || req.Metrics[0].Name != "cohortActiveUsers" {
		t.Fatalf("bad metric: %#v", req.Metrics)
	}
}
func TestCohortRequestClampsPeriods(t *testing.T) {
	req := cohortRequest("daily", 99, "2026-06-30")
	if len(req.CohortSpec.Cohorts) != cohortMaxPeriods || req.CohortSpec.CohortsRange.Granularity != "DAILY" {
		t.Fatalf("clamp/daily failed: %#v", req.CohortSpec)
	}
}
func TestRawURLRouting(t *testing.T) {
	cases := map[string][2]string{
		"admin default": {"admin", "properties/1/customDimensions"},
		"data alpha":    {"data-alpha", "/properties/1:runFunnelReport"},
		"unknown host":  {"bogus", "properties/1"},
	}
	want := map[string]string{
		"admin default": "https://analyticsadmin.googleapis.com/v1beta/properties/1/customDimensions",
		"data alpha":    "https://analyticsdata.googleapis.com/v1alpha/properties/1:runFunnelReport",
		"unknown host":  "https://analyticsadmin.googleapis.com/v1beta/properties/1",
	}
	for name, in := range cases {
		if got := rawURL(in[0], in[1]); got != want[name] {
			t.Fatalf("%s: got %q want %q", name, got, want[name])
		}
	}
	full := "https://example.com/v1/x"
	if got := rawURL("admin", full); got != full {
		t.Fatalf("full URL should pass through: got %q", got)
	}
}
func TestRawMutationDetection(t *testing.T) {
	if rawIsMutating("GET") || rawIsMutating("head") {
		t.Fatal("GET/HEAD must be non-mutating")
	}
	for _, m := range []string{"POST", "patch", "PUT", "DELETE"} {
		if !rawIsMutating(m) {
			t.Fatalf("%s must be mutating", m)
		}
	}
	if !rawNeedsBody("post") || !rawNeedsBody("PATCH") || rawNeedsBody("DELETE") || rawNeedsBody("GET") {
		t.Fatal("rawNeedsBody: want POST/PATCH/PUT only")
	}
}
func TestPropertyResolutionPrefersFlag(t *testing.T) {
	f := &rootFlags{propertyID: "properties/123"}
	if got := configuredProperty(f); got != "123" {
		t.Fatalf("got %q", got)
	}
}
