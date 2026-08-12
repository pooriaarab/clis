// Copyright 2026 Cathryn Lavery and contributors. Licensed under Apache-2.0. See LICENSE.

package ga4

import "encoding/json"

const AnalyticsReadonlyScope = "https://www.googleapis.com/auth/analytics.readonly"

// AnalyticsEditScope permits GA4 Admin config writes. RequestedScopes is the
// space-delimited set minted into every token: readonly satisfies the Data API,
// edit satisfies Admin writes. Actual write permission still requires an Editor+
// role on the property; the scope only authorizes the token to ask.
const AnalyticsEditScope = "https://www.googleapis.com/auth/analytics.edit"

const RequestedScopes = AnalyticsReadonlyScope + " " + AnalyticsEditScope

type ServiceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
	ProjectID   string `json:"project_id"`
}

type DateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type Named struct {
	Name string `json:"name"`
}

type Metric = Named
type Dimension = Named

type OrderBy struct {
	Desc      bool              `json:"desc,omitempty"`
	Metric    *MetricOrderBy    `json:"metric,omitempty"`
	Dimension *DimensionOrderBy `json:"dimension,omitempty"`
}

type MetricOrderBy struct {
	MetricName string `json:"metricName"`
}
type DimensionOrderBy struct {
	DimensionName string `json:"dimensionName"`
}

type StringFilter struct {
	MatchType string `json:"matchType,omitempty"`
	Value     string `json:"value"`
}

type Filter struct {
	FieldName    string        `json:"fieldName"`
	StringFilter *StringFilter `json:"stringFilter,omitempty"`
}

type FilterExpression struct {
	Filter *Filter         `json:"filter,omitempty"`
	Raw    json.RawMessage `json:"-"`
}

func (f FilterExpression) MarshalJSON() ([]byte, error) {
	if len(f.Raw) > 0 {
		return f.Raw, nil
	}
	type alias FilterExpression
	return json.Marshal(alias(f))
}

type RunReportRequest struct {
	DateRanges      []DateRange       `json:"dateRanges,omitempty"`
	Metrics         []Metric          `json:"metrics,omitempty"`
	Dimensions      []Dimension       `json:"dimensions,omitempty"`
	Limit           string            `json:"limit,omitempty"`
	DimensionFilter *FilterExpression `json:"dimensionFilter,omitempty"`
	OrderBys        []OrderBy         `json:"orderBys,omitempty"`
	CohortSpec      *CohortSpec       `json:"cohortSpec,omitempty"`
}

// CohortSpec drives GA4's real Cohort exploration (runReport cohortSpec).
// Each Cohort selects an acquisition group by firstSessionDate; CohortsRange
// defines the retention axis (granularity + offsets) tracked for every cohort.
type CohortSpec struct {
	Cohorts      []Cohort     `json:"cohorts"`
	CohortsRange CohortsRange `json:"cohortsRange"`
}

type Cohort struct {
	Name      string    `json:"name"`
	Dimension string    `json:"dimension"`
	DateRange DateRange `json:"dateRange"`
}

type CohortsRange struct {
	Granularity string `json:"granularity"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
}

type Pivot struct {
	FieldNames []string `json:"fieldNames"`
	Limit      string   `json:"limit,omitempty"`
}

type RunPivotReportRequest struct {
	RunReportRequest
	Pivots []Pivot `json:"pivots,omitempty"`
}

type BatchRunReportsRequest struct {
	Requests []RunReportRequest `json:"requests"`
}

type RunRealtimeReportRequest struct {
	Metrics    []Metric    `json:"metrics,omitempty"`
	Dimensions []Dimension `json:"dimensions,omitempty"`
	Limit      string      `json:"limit,omitempty"`
}

type CheckCompatibilityRequest struct {
	Metrics             []Metric    `json:"metrics,omitempty"`
	Dimensions          []Dimension `json:"dimensions,omitempty"`
	CompatibilityFilter string      `json:"compatibilityFilter,omitempty"`
}

type FunnelEventFilter struct {
	EventName string `json:"eventName"`
}
type FunnelFilterExpression struct {
	FunnelEventFilter *FunnelEventFilter `json:"funnelEventFilter"`
}
type FunnelStep struct {
	Name             string                  `json:"name"`
	FilterExpression *FunnelFilterExpression `json:"filterExpression"`
}
type Funnel struct {
	Steps []FunnelStep `json:"steps"`
}
type RunFunnelReportRequest struct {
	DateRanges []DateRange `json:"dateRanges,omitempty"`
	Funnel     Funnel      `json:"funnel"`
}

type Header struct {
	Name string `json:"name"`
}
type Value struct {
	Value string `json:"value"`
}
type Row struct {
	DimensionValues []Value `json:"dimensionValues,omitempty"`
	MetricValues    []Value `json:"metricValues,omitempty"`
}
type ReportResponse struct {
	DimensionHeaders []Header       `json:"dimensionHeaders,omitempty"`
	MetricHeaders    []Header       `json:"metricHeaders,omitempty"`
	Rows             []Row          `json:"rows,omitempty"`
	Totals           []Row          `json:"totals,omitempty"`
	Raw              map[string]any `json:"-"`
}

type AccountSummariesResponse struct {
	AccountSummaries []AccountSummary `json:"accountSummaries,omitempty"`
	NextPageToken    string           `json:"nextPageToken,omitempty"`
}
type AccountSummary struct {
	Name              string            `json:"name,omitempty"`
	Account           string            `json:"account,omitempty"`
	DisplayName       string            `json:"displayName,omitempty"`
	PropertySummaries []PropertySummary `json:"propertySummaries,omitempty"`
}
type PropertySummary struct {
	Property     string `json:"property,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	PropertyType string `json:"propertyType,omitempty"`
}
type Property struct {
	Name         string `json:"name,omitempty"`
	Parent       string `json:"parent,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	TimeZone     string `json:"timeZone,omitempty"`
	CurrencyCode string `json:"currencyCode,omitempty"`
}
type DataStreamsResponse struct {
	DataStreams []DataStream `json:"dataStreams,omitempty"`
}
type DataStream struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}
