package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/pooriaarab/clis/google-analytics/internal/ga4"
	"github.com/spf13/cobra"
)

const cohortMaxPeriods = 12

// cohortRequest builds a real GA4 cohortSpec runReport that mirrors the UI
// Cohort exploration: `periods` acquisition cohorts by firstSessionDate (cohort_0
// is the most recent bucket ending at `end`), each tracked over the same
// retention axis. Granularity is daily|weekly|monthly; `end` is YYYY-MM-DD and
// falls back to today (UTC) when empty/unparseable. Buckets align to the
// granularity so cohorts[].dateRange stays valid against cohortsRange.
func cohortRequest(granularity string, periods int, end string) ga4.RunReportRequest {
	if periods <= 0 {
		periods = 6
	}
	if periods > cohortMaxPeriods {
		periods = cohortMaxPeriods
	}
	endT, err := time.Parse("2006-01-02", strings.TrimSpace(end))
	if err != nil {
		endT = time.Now().UTC()
	}
	var apiGran, nthDim string
	cohorts := make([]ga4.Cohort, 0, periods)
	switch strings.ToLower(strings.TrimSpace(granularity)) {
	case "daily", "day":
		apiGran, nthDim = "DAILY", "cohortNthDay"
		for i := 0; i < periods; i++ {
			d := endT.AddDate(0, 0, -i).Format("2006-01-02")
			cohorts = append(cohorts, cohortAt(i, d, d))
		}
	case "monthly", "month":
		apiGran, nthDim = "MONTHLY", "cohortNthMonth"
		for i := 0; i < periods; i++ {
			m := endT.AddDate(0, -i, 0)
			first := time.Date(m.Year(), m.Month(), 1, 0, 0, 0, 0, time.UTC)
			last := first.AddDate(0, 1, -1)
			cohorts = append(cohorts, cohortAt(i, first.Format("2006-01-02"), last.Format("2006-01-02")))
		}
	default: // weekly
		apiGran, nthDim = "WEEKLY", "cohortNthWeek"
		for i := 0; i < periods; i++ {
			endW := endT.AddDate(0, 0, -i*7)
			startW := endW.AddDate(0, 0, -6)
			cohorts = append(cohorts, cohortAt(i, startW.Format("2006-01-02"), endW.Format("2006-01-02")))
		}
	}
	return ga4.RunReportRequest{
		Dimensions: []ga4.Dimension{{Name: "cohort"}, {Name: nthDim}},
		Metrics:    []ga4.Metric{{Name: "cohortActiveUsers"}},
		CohortSpec: &ga4.CohortSpec{
			Cohorts:      cohorts,
			CohortsRange: ga4.CohortsRange{Granularity: apiGran, StartOffset: 0, EndOffset: periods - 1},
		},
	}
}
func cohortAt(i int, start, end string) ga4.Cohort {
	// GA4 rejects names beginning with "cohort_"; the acquisition start date is
	// unique per bucket and self-describing when it surfaces as the cohort dim.
	return ga4.Cohort{Name: start, Dimension: "firstSessionDate", DateRange: ga4.DateRange{StartDate: start, EndDate: end}}
}

func reportFlags(c *cobra.Command, metrics, dims, start, end *string, limit *int) {
	c.Flags().StringVar(metrics, "metrics", "sessions,totalUsers,conversions,totalRevenue", "Comma-separated metrics")
	c.Flags().StringVar(dims, "dimensions", "date", "Comma-separated dimensions")
	dateLimitFlags(c, start, end, limit)
}
func dateLimitFlags(c *cobra.Command, start, end *string, limit *int) {
	c.Flags().StringVar(start, "start", "30daysAgo", "Start date (YYYY-MM-DD or NdaysAgo)")
	c.Flags().StringVar(end, "end", "yesterday", "End date")
	c.Flags().IntVar(limit, "limit", 25, "Max rows")
}
func splitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func splitDefault(s, d string) []string {
	if strings.TrimSpace(s) == "" {
		s = d
	}
	return splitCSV(s)
}
func metricNames(xs []string) []ga4.Metric {
	out := []ga4.Metric{}
	for _, x := range xs {
		out = append(out, ga4.Metric{Name: x})
	}
	return out
}
func dimensionNames(xs []string) []ga4.Dimension {
	out := []ga4.Dimension{}
	for _, x := range xs {
		out = append(out, ga4.Dimension{Name: x})
	}
	return out
}
func reportRequest(metrics, dims, start, end string, limit int) ga4.RunReportRequest {
	if start == "" {
		start = "30daysAgo"
	}
	if end == "" {
		end = "yesterday"
	}
	if limit <= 0 {
		limit = 25
	}
	return ga4.RunReportRequest{DateRanges: []ga4.DateRange{{StartDate: start, EndDate: end}}, Metrics: metricNames(splitCSV(metrics)), Dimensions: dimensionNames(splitCSV(dims)), Limit: strconv.Itoa(limit)}
}
func realtimeRequest(metrics, dims string, limit int) ga4.RunRealtimeReportRequest {
	if limit <= 0 {
		limit = 10
	}
	return ga4.RunRealtimeReportRequest{Metrics: metricNames(splitDefault(metrics, "activeUsers")), Dimensions: dimensionNames(splitCSV(dims)), Limit: strconv.Itoa(limit)}
}
func compatibilityRequest(metrics, dims string) ga4.CheckCompatibilityRequest {
	return ga4.CheckCompatibilityRequest{Metrics: metricNames(splitCSV(metrics)), Dimensions: dimensionNames(splitCSV(dims)), CompatibilityFilter: "COMPATIBLE"}
}
func addRawDimensionFilter(req *ga4.RunReportRequest, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(raw), &js); err != nil {
		return err
	}
	req.DimensionFilter = &ga4.FilterExpression{Raw: js}
	return nil
}
func addOrder(req *ga4.RunReportRequest, order string) {
	if order == "" {
		return
	}
	desc := strings.HasPrefix(order, "-")
	name := strings.TrimPrefix(order, "-")
	for _, dim := range req.Dimensions {
		if dim.Name == name {
			req.OrderBys = []ga4.OrderBy{{Desc: desc, Dimension: &ga4.DimensionOrderBy{DimensionName: name}}}
			return
		}
	}
	req.OrderBys = []ga4.OrderBy{{Desc: desc, Metric: &ga4.MetricOrderBy{MetricName: name}}}
}
func funnelRequest(steps, start, end string) ga4.RunFunnelReportRequest {
	fs := []ga4.FunnelStep{}
	for _, s := range splitCSV(steps) {
		fs = append(fs, ga4.FunnelStep{Name: s, FilterExpression: &ga4.FunnelFilterExpression{FunnelEventFilter: &ga4.FunnelEventFilter{EventName: s}}})
	}
	return ga4.RunFunnelReportRequest{DateRanges: []ga4.DateRange{{StartDate: start, EndDate: end}}, Funnel: ga4.Funnel{Steps: fs}}
}
