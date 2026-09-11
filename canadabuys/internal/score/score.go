// Package score ranks tender notices for a small software team.
package score

import (
	"canadabuys-cli/internal/model"
	"fmt"
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile("[^a-z0-9]+")

// WeightTable is the only home of signal weights. Missing signals are
// excluded and the rest renormalized; a missing award is MISSING.
var WeightTable = []struct {
	Name   string
	Weight float64
	Why    string
}{
	{"category-fit", 3, "UNSPSC 43 IT or 81 research; GSIN maps to UNSPSC where blank"},
	{"keyword", 3, "software/product terms up, construction and goods terms down"},
	{"award-band", 2, "$50k-$2M is small-team-winnable; banded, never raw"},
	{"recurrence", 2, "same buyer and title across years is a product"},
	{"competition", 2, "open process up; sole-source and limited tendering down"},
	{"incumbency", 1, "fragmented categories are open; concentrated ones closed"},
}

type Signal struct {
	Name         string  `json:"name"`
	Raw          string  `json:"raw"`
	Weight       float64 `json:"weight"`
	Score        float64 `json:"score"`
	Contribution float64 `json:"contribution"`
	HasData      bool    `json:"hasData"`
}
type Opportunity struct {
	Reference   string   `json:"reference"`
	Title       string   `json:"title"`
	Buyer       string   `json:"buyer"`
	Category    string   `json:"category"`
	Score       float64  `json:"score"`
	Signals     []Signal `json:"signals"`
	AwardBand   string   `json:"awardBand"`
	Closing     string   `json:"closing"`
	Incumbent   string   `json:"incumbent,omitempty"`
	Description string   `json:"-"`
	Staffing    bool     `json:"staffingVehicle"`
}
type Context struct {
	GsinUNSPSC map[string]string
	AwardCents map[string]int64
	RecurYears map[string]int
	CatShare   map[string]float64
	CatTop     map[string]string
}

func Score(t model.Tender, c Context) Opportunity {
	ab, band := awardBand(t.Solicitation, c.AwardCents)
	is, inc := incumbency(t, c)
	s := []Signal{categoryFit(t, c.GsinUNSPSC), keyword(t), ab, recurrence(t, c), competition(t), is}
	var num, den float64
	for i := range s {
		if s[i].HasData {
			num += s[i].Weight * s[i].Score
			den += s[i].Weight
			s[i].Contribution = s[i].Weight * s[i].Score
		}
	}
	total := num / den
	for i := range s {
		s[i].Contribution /= den
	}
	cat := strings.Join(t.Categories, ";")
	if cat == "" {
		cat = "unknown"
	}
	o := Opportunity{Reference: t.Reference, Title: t.Title, Buyer: t.Org, Category: cat, Score: total, Signals: s, AwardBand: band, Incumbent: inc, Closing: t.Closing, Description: t.Description}
	o.Staffing, _ = IsStaffing(t)
	return o
}
func categoryFit(t model.Tender, gsin map[string]string) Signal {
	g := Signal{Name: "category-fit", Weight: WeightTable[0].Weight, HasData: true, Score: 0.5}
	var codes []string
	for _, c := range t.UNSPSC {
		if c != "" {
			codes = append(codes, c)
		}
	}
	src := "unspsc"
	if len(codes) == 0 {
		src = "gsin"
		for _, c := range t.GSIN {
			if u := gsin[strings.TrimSpace(c)]; u != "" {
				codes = append(codes, u)
			}
		}
	}
	g.Raw = src + " " + strings.Join(codes, ",")
	for _, code := range codes {
		switch segment(code) {
		case "43":
			g.Score, g.Raw = 1, src+" "+code
		case "81":
			if g.Score < 1 {
				g.Score, g.Raw = 0.7, src+" "+code
			}
		}
	}
	if g.Score == 0.5 {
		hay := strings.ToLower(t.UNSPSCDesc + "\n" + t.GSINDesc)
		if strings.Contains(hay, "software") || strings.Contains(hay, "computer") || strings.Contains(hay, "information") || strings.Contains(hay, "data") {
			g.Score = 0.6
		}
		if len(codes) == 0 && hay == "\n" {
			g.Score, g.Raw = 0, "no category codes"
		}
	}
	return g
}
func segment(code string) string {
	f := strings.FieldsFunc(code, func(r rune) bool { return r < '0' || r > '9' })
	if len(f) == 0 || len(f[0]) < 2 {
		return ""
	}
	return f[0][:2]
}
func keyword(t model.Tender) Signal {
	g := Signal{Name: "keyword", Weight: WeightTable[1].Weight, HasData: true}
	hay := strings.ToLower(t.Title + "\n" + t.Description)
	np, nn := 0, 0
	for _, k := range []string{"software", "platform", "saas", "cloud", "data", "analytics", "machine learning", "portal", "dashboard", "api", "integration", "automation", "licence", "license", "application", "system"} {
		if strings.Contains(hay, k) {
			np++
		}
	}
	for _, k := range []string{"construction", "janitorial", "furniture", "vehicle", "ammunition", "snow", "roof", "paving", "uniform", "catering", "pest", "landscaping", "plumbing", "electrical"} {
		if strings.Contains(hay, k) {
			nn++
		}
	}
	g.Raw = fmt.Sprintf("+%d positive, -%d negative", np, nn)
	g.Score = max(0, min(1, 0.4+0.12*float64(np)-0.3*float64(nn)))
	return g
}
func awardBand(solic string, cents map[string]int64) (Signal, string) {
	g := Signal{Name: "award-band", Weight: WeightTable[2].Weight}
	v, ok := cents[solic]
	if !ok || solic == "" {
		g.Raw = "no award data"
		return g, g.Raw
	}
	g.HasData, g.Raw = true, fmt.Sprintf("$%d", v/100)
	switch k := v / 100000; {
	case k < 50:
		g.Score = 0.4
	case k < 250:
		g.Score = 1
	case k < 1000:
		g.Score = 0.8
	case k < 2000:
		g.Score = 0.6
	case k < 5000:
		g.Score = 0.3
	default:
		g.Score = 0.1
	}
	return g, g.Raw
}
func recurrence(t model.Tender, c Context) Signal {
	g := Signal{Name: "recurrence", Weight: WeightTable[3].Weight, HasData: true, Score: 0.2}
	n := max(1, c.RecurYears[RecurKey(t)])
	g.Raw = fmt.Sprintf("%d distinct year(s)", n)
	if n == 2 {
		g.Score = 0.6
	} else if n >= 3 {
		g.Score = 1
	}
	return g
}
func competition(t model.Tender) Signal {
	g := Signal{Name: "competition", Weight: WeightTable[4].Weight, HasData: true, Raw: t.Method, Score: 0.5}
	m := strings.ToLower(t.Method)
	switch {
	case strings.Contains(m, "open bidding"):
		g.Score = 1
	case strings.Contains(m, "selective") || strings.Contains(m, "traditional"):
		g.Score = 0.7
	case strings.Contains(m, "limited") || strings.Contains(m, "non-competitive") || strings.Contains(t.NoticeType, "Advance Contract") || strings.Contains(t.NoticeType, "Directed"):
		g.Score = 0.2
	}
	if r := strings.ToLower(t.LimitedReason); r != "" && r != "none" {
		g.Score = min(g.Score, 0.3)
		g.Raw += " / " + t.LimitedReason
	}
	return g
}
func incumbency(t model.Tender, c Context) (Signal, string) {
	g := Signal{Name: "incumbency", Weight: WeightTable[5].Weight, Raw: "no supplier data"}
	for _, k := range t.Categories {
		if sh, ok := c.CatShare[k]; ok {
			g.HasData, g.Raw, g.Score = true, fmt.Sprintf("top share %.0f%%", sh*100), 1-sh
			return g, fmt.Sprintf("%s (%.0f%%)", c.CatTop[k], sh*100)
		}
	}
	return g, ""
}
func RecurKey(t model.Tender) string { return t.Org + "\x00" + normTitle(t.Title) }
func normTitle(s string) string {
	return strings.Join(strings.Fields(nonAlnum.ReplaceAllString(strings.ToLower(s), " ")), " ")
}
func IsStaffing(t model.Tender) (bool, string) {
	hay := strings.ToLower(t.Title + "\n" + t.Description + "\n" + t.NoticeType)
	for _, k := range []string{"tbips", "sbips", "proservices", "task-based", "supply arrangement", "professional services", "level 1", "level 2", "level 3", "level 4"} {
		if strings.Contains(hay, k) {
			return true, "matched " + k
		}
	}
	return false, "not a staffing vehicle"
}
