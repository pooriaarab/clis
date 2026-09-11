package cli

import (
	"canadabuys-cli/internal/llm"
	"canadabuys-cli/internal/model"
	"strings"
	"testing"
)

func TestKeepLatestAmendmentBySolicitation(t *testing.T) {
	kept, n := keepLatestAmendments([]model.Tender{
		{Reference: "old", Solicitation: "BPM014975/17913", Amendment: "000", Title: "Exceed TurboX Premium Licences and Maintenance for SSC", Publication: "2023-01-02"},
		{Reference: "new", Solicitation: "BPM014975/17913", Amendment: "003", AmendmentDate: "2026-03-31", Title: "Exceed TurboX Premium Licences and Maintenance for SSC", Publication: "2022-12-19"},
	})
	if n != 1 || len(kept) != 1 || kept[0].Reference != "new" {
		t.Fatalf("kept=%+v collapsed=%d", kept, n)
	}
}

func TestBlankSolicitationFallsBackToReference(t *testing.T) {
	in := []model.Tender{
		{Reference: "r-a", Title: "Online ArcGIS training", Amendment: "000"},
		{Reference: "r-b", Title: "Online ArcGIS training", Amendment: "001"},
	}
	kept, n := keepLatestAmendments(in)
	if n != 0 || len(kept) != 2 {
		t.Fatalf("blank solicitation must not merge different references: kept=%d collapsed=%d", len(kept), n)
	}
	same := []model.Tender{
		{Reference: "r-only", Title: "Online ArcGIS training", Amendment: "000", Publication: "2020-01-01"},
		{Reference: "r-only", Title: "Online ArcGIS training", Amendment: "002", Publication: "2021-01-01"},
	}
	kept, n = keepLatestAmendments(same)
	if n != 1 || len(kept) != 1 || kept[0].Amendment != "002" {
		t.Fatalf("same blank-solicitation reference: kept=%+v collapsed=%d", kept, n)
	}
}

func TestCollapsedSetHasUniqueSolicitation(t *testing.T) {
	kept, _ := keepLatestAmendments([]model.Tender{
		{Reference: "a1", Solicitation: "S1", Amendment: "000"},
		{Reference: "a2", Solicitation: "S1", Amendment: "002"},
		{Reference: "b1", Solicitation: "S2", Amendment: "000"},
		{Reference: "c1", Solicitation: "", Amendment: "000"},
		{Reference: "c2", Solicitation: "", Amendment: "001"},
	})
	seen := map[string]bool{}
	for _, tnd := range kept {
		k := tnd.Solicitation
		if k == "" {
			k = tnd.Reference
		}
		if seen[k] {
			t.Fatalf("duplicate key %q", k)
		}
		seen[k] = true
	}
	if len(kept) != 4 {
		t.Fatalf("kept=%d", len(kept))
	}
}

func TestSameTitleDifferentSolicitationNotMerged(t *testing.T) {
	// "Dental Services" is many distinct prison contracts. Title-only
	// grouping discarded 15,510 real solicitations; this must not.
	in := []model.Tender{
		{Reference: "d1", Solicitation: "S-2014", Title: "Dental Services", Org: "Correctional Service of Canada", Amendment: "000"},
		{Reference: "d2", Solicitation: "S-2024", Title: "Dental Services", Org: "Correctional Service of Canada", Amendment: "000"},
		{Reference: "d3", Solicitation: "S-OTHER", Title: "Dental Services", Org: "Shared Services Canada", Amendment: "000"},
	}
	kept, n := keepLatestAmendments(in)
	if n != 0 || len(kept) != 3 {
		t.Fatalf("title must not merge solicitations: kept=%d collapsed=%d", len(kept), n)
	}
}

func TestGroupSimilarIsBuyerPlusTitle(t *testing.T) {
	in := []model.Tender{
		{Reference: "a1", Solicitation: "S1", Title: "Request for Qualifications - Web Development Consultants", Org: "Dept A", Amendment: "000"},
		{Reference: "a2", Solicitation: "S2", Title: "Request for Qualifications - Web Development Consultants", Org: "Dept A", Amendment: "001"},
		{Reference: "b1", Solicitation: "S3", Title: "Request for Qualifications - Web Development Consultants", Org: "Dept B", Amendment: "000"},
	}
	kept, n := groupSimilarNotices(in)
	if n != 1 || len(kept) != 2 {
		t.Fatalf("buyer+title: kept=%d collapsed=%d", len(kept), n)
	}
	buyers := map[string]bool{}
	for _, tnd := range kept {
		buyers[tnd.Org] = true
	}
	if !buyers["Dept A"] || !buyers["Dept B"] {
		t.Fatalf("different buyers must stay separate: %+v", kept)
	}
}

func TestOppSummaryMentionsGroupingOnlyWhenAsked(t *testing.T) {
	plain := oppSummary(4102, 13181, 17730, 0, false, false)
	if !strings.Contains(plain, "4102 amendment rows collapsed") {
		t.Fatalf("missing amendment count: %s", plain)
	}
	if strings.Contains(plain, "similar") || strings.Contains(plain, "grouped") {
		t.Fatalf("grouping leaked into the default summary: %s", plain)
	}
	opt := oppSummary(4102, 13181, 17730, 0, true, false)
	if !strings.Contains(opt, "13181 similar notices grouped") {
		t.Fatalf("opt-in summary: %s", opt)
	}
}

func TestRankOnProductFit(t *testing.T) {
	rows := []oppOut{
		{Enrichment: &llm.Enrichment{Reference: "resale", ProductFit: 8, Shape: "resale"}},
		{Enrichment: &llm.Enrichment{Reference: "product", ProductFit: 90, Shape: "product"}},
		{Enrichment: &llm.Enrichment{Reference: "staff", ProductFit: 12, Shape: "staffing"}},
		{Enrichment: &llm.Enrichment{Reference: "train", ProductFit: 10, Shape: "training"}},
	}
	// moreProductFit is the sort callback; apply it the same way the command does.
	less := moreProductFit(rows)
	if !less(1, 0) || less(0, 1) {
		t.Fatal("product must rank above resale")
	}
	if productFitOf(rows[0]) >= productFitOf(rows[1]) {
		t.Fatal("resale outranked product")
	}
}
