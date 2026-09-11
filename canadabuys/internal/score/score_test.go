package score

import (
	"canadabuys-cli/internal/model"
	"testing"
)

// $150k open software must outscore $40M sole-sourced construction;
// a missing award join is MISSING, never a penalty.
func TestSoftwareBeatsConstruction(t *testing.T) {
	ctx := Context{AwardCents: map[string]int64{"S1": 150000 * 100}, RecurYears: map[string]int{}, CatShare: map[string]float64{"*GD": 0.1}, CatTop: map[string]string{"*GD": "Someone Inc"}}
	sw := model.Tender{Reference: "R1", Solicitation: "S1", Title: "Cloud analytics platform SaaS",
		Description: "Data dashboard with API integration", UNSPSC: []string{"43230000"},
		Method: "Competitive - Open bidding", Org: "Dept X", Categories: []string{"*GD"}}
	con := model.Tender{Reference: "R2", Solicitation: "S9", Title: "Ammunition vehicle construction",
		Description: "Paving and roofing works", UNSPSC: []string{"72100000"},
		Method: "Non-competitive", LimitedReason: "Exclusive Rights", Org: "Dept Y"}
	a, b := Score(sw, ctx), Score(con, ctx)
	if a.Score <= b.Score {
		t.Fatalf("software %.3f did not beat construction %.3f", a.Score, b.Score)
	}
	plain := sw
	plain.Solicitation = "MISSING"
	c := Score(plain, ctx)
	for _, s := range c.Signals {
		if s.Name == "award-band" && (s.HasData || s.Raw != "no award data") {
			t.Fatalf("missing award must be explicit, got %+v", s)
		}
	}
	if c.Score <= b.Score {
		t.Fatalf("software without award %.3f did not beat construction %.3f", c.Score, b.Score)
	}
	if ok, _ := IsStaffing(model.Tender{Title: "TBIPS Programmer, Level 2", NoticeType: "RFP against Supply Arrangement"}); !ok {
		t.Fatal("TBIPS role not detected as staffing")
	}
	if ok, _ := IsStaffing(model.Tender{Title: "Cloud analytics platform"}); ok {
		t.Fatal("genuine product flagged as staffing")
	}
}
