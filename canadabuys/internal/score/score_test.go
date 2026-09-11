package score

import (
	"canadabuys-cli/internal/model"
	"strings"
	"testing"
)

// $150k open software must outscore $40M sole-sourced construction;
// a missing award join is MISSING, never a penalty.
func TestSoftwareBeatsConstruction(t *testing.T) {
	ctx := Context{AwardCents: map[string]int64{"S1": 150000 * 100}, RecurYears: map[string]int{}, CatShare: map[string]float64{"43230000": 0.1}, CatTop: map[string]string{"43230000": "Someone Inc"}}
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

func TestSoftwareSignalsStaySoftwareShaped(t *testing.T) {
	for _, tc := range []struct {
		title, code string
		want        float64
	}{
		{"Analytics platform", "43230000", 1},
		{"Laser scanner", "43211711", -1},
		{"Design-Build Services For Harrow Research", "43233600", 0},
		{"Isothermal Calorimeter", "43232605", 0.45},
		{"Artifacts Management Software", "43232605", 1},
	} {
		g := categoryFit(model.Tender{Title: tc.title, UNSPSC: []string{tc.code}}, nil)
		if tc.want < 0 && g.Score >= 1 || tc.want >= 0 && g.Score != tc.want {
			t.Fatalf("%q: got %.2f want %.2f", tc.title, g.Score, tc.want)
		}
	}
	if g := keyword(model.Tender{Title: "Elder Services", Description: "the system shall provide software data"}); g.Score >= 0.4 {
		t.Fatalf("boilerplate description must not lift a service title, got %+v", g)
	}
	if _, err := ScoreLens(model.Tender{}, Context{}, "wedge"); err == nil ||
		!strings.Contains(err.Error(), "horizontal") || !strings.Contains(err.Error(), "displace") || !strings.Contains(err.Error(), "bootstrap") {
		t.Fatal("unknown lens must name available lenses")
	}
	sw, fk := model.Tender{Title: "Cloud analytics platform SaaS", UNSPSC: []string{"43230000"}}, model.Tender{Title: "Telehandler, Variable-Reach Forklift", UNSPSC: []string{"24101600"}}
	for _, l := range ListLenses() {
		a, _ := ScoreLens(sw, Context{}, l.Name)
		b, _ := ScoreLens(fk, Context{}, l.Name)
		if a.Score <= b.Score {
			t.Fatalf("%s: software %.3f vs forklift %.3f", l.Name, a.Score, b.Score)
		}
		var fit, kw bool
		for _, s := range b.Signals {
			fit = fit || s.Name == "category-fit"
			kw = kw || s.Name == "keyword"
		}
		if !fit || !kw {
			t.Fatalf("%s dropped the software gate", l.Name)
		}
	}
}

func TestIncumbentUsesUnspscNotGoodsBucket(t *testing.T) {
	davie := map[string]string{"*GD": "Chantier Davie", "46171610": "Simex Defence Inc.", "461716": "Other"}
	for _, tc := range []struct {
		code  string
		share map[string]float64
		top   map[string]string
		want  string
	}{
		{"46171610", map[string]float64{"*GD": 0.505, "46171610": 0.8, "461716": 0.4}, davie, "Simex Defence Inc. (80%)"},
		{"40141600", map[string]float64{"*GD": 0.505}, davie, "no incumbent identified"},
		{"*46171610", map[string]float64{"*GD": 0.505, "461716": 0.62}, map[string]string{"*GD": "Chantier Davie", "461716": "Simex Defence Inc."}, "Simex Defence Inc. (62%)"},
	} {
		o, err := ScoreLens(model.Tender{UNSPSC: []string{tc.code}, Categories: []string{"*GD"}}, Context{CatShare: tc.share, CatTop: tc.top}, "displace")
		if err != nil || o.Incumbent != tc.want {
			t.Fatalf("want %q, got %q %v", tc.want, o.Incumbent, err)
		}
	}
}
