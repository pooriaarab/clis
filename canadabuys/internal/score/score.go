// Package score ranks tender notices for a small software team.
package score

import (
	"canadabuys-cli/internal/model"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var nonAlnum = regexp.MustCompile("[^a-z0-9]+")

// WeightTable is the only home of signal weights. Missing signals are
// excluded and the rest renormalized; a missing award is MISSING.
var WeightTable = []struct {
	Name   string
	Weight float64
	Why    string
}{
	{"category-fit", 3, "UNSPSC 4323 software or 8111 computer services; other 43/81 is weak; GSIN maps where blank"},
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
	Reference    string   `json:"reference"`
	Solicitation string   `json:"solicitationNumber,omitempty"`
	Title        string   `json:"title"`
	Buyer        string   `json:"buyer"`
	Category     string   `json:"category"`
	Score        float64  `json:"score"`
	Signals      []Signal `json:"signals"`
	AwardBand    string   `json:"awardBand"`
	Closing      string   `json:"closing"`
	Incumbent    string   `json:"incumbent,omitempty"`
	Description  string   `json:"-"`
	Staffing     bool     `json:"staffingVehicle"`
}
type Context struct {
	GsinUNSPSC map[string]string
	AwardCents map[string]int64
	RecurYears map[string]int
	CatShare   map[string]float64
	CatTop     map[string]string
	TitleDepts map[string]int
	ProductFit map[string]float64
	FitShape   map[string]string
	BuyerCat   map[string]int
	CatBuyers  map[string]int
}

// Now is the clock urgency and FutureClosing read; tests override it.
var Now = time.Now

type Lens struct {
	Name, Why string
	Weights   map[string]float64
}

func ListLenses() []Lens {
	def := map[string]float64{}
	for _, w := range WeightTable {
		def[w.Name] = w.Weight
	}
	gate := func(extra map[string]float64) map[string]float64 {
		w := map[string]float64{"category-fit": 3, "keyword": 3}
		for k, v := range extra {
			w[k] = v
		}
		return w
	}
	return []Lens{
		{"default", "the blended small-team score", def},
		{"horizontal", "market breadth across departments", gate(map[string]float64{"category-fit": 5, "keyword": 4, "breadth": 5, "recurrence": 2, "product-fit": 2})},
		{"displace", "incumbent vulnerability", gate(map[string]float64{"concentration": 5, "recurrence": 2, "award-band": 1, "competition": 1})},
		{"bootstrap", "winnable without capital", gate(map[string]float64{"band-fit": 4, "competition": 3, "incumbency": 2})},
		{"wedge", "beachhead depth times category breadth", gate(map[string]float64{"category-fit": 5, "keyword": 4, "wedge": 5, "recurrence": 2, "product-fit": 2})},
		{"recurring", "subscription-shaped purchase cadence", gate(map[string]float64{"category-fit": 5, "keyword": 4, "recurrence": 8})},
		{"biddable", "open now, ranked by fit and closing date", gate(map[string]float64{"category-fit": 6, "keyword": 5, "product-fit": 5, "urgency": 3, "competition": 1})},
	}
}

func LensByName(name string) (Lens, error) {
	if name == "" {
		name = "default"
	}
	var names []string
	for _, l := range ListLenses() {
		if l.Name == name {
			return l, nil
		}
		names = append(names, l.Name)
	}
	return Lens{}, fmt.Errorf("unknown lens %q (one of %s)", name, strings.Join(names, ", "))
}

var signalOrder = []string{"category-fit", "keyword", "award-band", "recurrence", "competition", "incumbency", "breadth", "concentration", "band-fit", "product-fit", "wedge", "urgency"}

func Score(t model.Tender, c Context) Opportunity {
	o, _ := ScoreLens(t, c, "default")
	return o
}

func ScoreLens(t model.Tender, c Context, lens string) (Opportunity, error) {
	l, err := LensByName(lens)
	if err != nil {
		return Opportunity{}, err
	}
	ab, band := awardBand(t.Solicitation, c.AwardCents)
	is, inc := incumbency(t, c)
	byName := map[string]Signal{
		"category-fit": categoryFit(t, c.GsinUNSPSC), "keyword": keyword(t),
		"award-band": ab, "recurrence": recurrence(t, c),
		"competition": competition(t), "incumbency": is,
		"breadth": breadth(t, c), "concentration": concentration(t, c),
		"band-fit": bandFit(t.Solicitation, c.AwardCents), "product-fit": productFit(c, t.Reference),
		"wedge": wedge(t, c), "urgency": urgency(t),
	}
	s := []Signal{}
	var num, den float64
	for _, name := range signalOrder {
		w, ok := l.Weights[name]
		if !ok || w <= 0 {
			continue
		}
		g := byName[name]
		g.Weight = w
		if g.HasData {
			num += w * g.Score
			den += w
			g.Contribution = w * g.Score
		}
		s = append(s, g)
	}
	total := num / den
	for i := range s {
		s[i].Contribution /= den
	}
	cat := strings.Join(t.Categories, ";")
	if cat == "" {
		cat = "unknown"
	}
	o := Opportunity{Reference: t.Reference, Solicitation: t.Solicitation, Title: t.Title, Buyer: t.Org, Category: cat, Score: total, Signals: s, AwardBand: band, Incumbent: inc, Closing: t.Closing, Description: t.Description}
	o.Staffing, _ = IsStaffing(t)
	return o, nil
}
func categoryFit(t model.Tender, gsin map[string]string) Signal {
	g := Signal{Name: "category-fit", Weight: WeightTable[0].Weight, HasData: true}
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
		d := unspscDigits(code)
		switch {
		case len(d) >= 4 && (d[:4] == "4323" || d[:4] == "8111"):
			g.Score, g.Raw = 1, src+" "+code
		case segment(code) == "43" && g.Score < 0.25:
			g.Score, g.Raw = 0.25, src+" "+code+" hardware"
		case segment(code) == "81" && g.Score < 0.25:
			g.Score, g.Raw = 0.25, src+" "+code+" research"
		}
	}
	if g.Score == 0 {
		hay := strings.ToLower(t.UNSPSCDesc)
		if strings.Contains(hay, "software") || strings.Contains(hay, "computer") {
			g.Score = 0.6
		}
		if len(codes) == 0 && hay == "" {
			g.Raw = "no category codes"
		}
	}
	if countTerms(t.Title, goodsTitleTerms) > 0 {
		g.Score, g.Raw = 0, g.Raw+" title not software"
	} else if countTerms(t.Title, softwareTitleTerms) == 0 && g.Score >= 1 {
		g.Score, g.Raw = 0.45, g.Raw+" title silent"
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
	np, nn := countTerms(t.Title, softwareTitleTerms), countTerms(t.Title, goodsTitleTerms) // title only: descriptions are boilerplate
	g.Raw = fmt.Sprintf("+%d positive, -%d negative", np, nn)
	g.Score = max(0, min(1, 0.4+0.12*float64(np)-0.3*float64(nn)))
	if np == 0 {
		g.Score = max(0, 0.1-0.3*float64(nn))
	}
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
	if sh, top, key, ok := categoryShare(t, c); ok {
		g.HasData, g.Raw, g.Score = true, fmt.Sprintf("top share %.0f%% of %s", sh*100, key), 1-sh
		return g, fmt.Sprintf("%s (%.0f%%)", top, sh*100)
	}
	return g, "no incumbent identified"
}

func categoryShare(t model.Tender, c Context) (float64, string, string, bool) {
	for _, k := range UnspscKeys(Codes(t.UNSPSC, t.GSIN, c.GsinUNSPSC)) {
		if sh, ok := c.CatShare[k]; ok && c.CatTop[k] != "" {
			return sh, c.CatTop[k], k, true
		}
	}
	return 0, "", "", false
}
func RecurKey(t model.Tender) string { return t.Org + "\x00" + normTitle(t.Title) }
func normTitle(s string) string {
	return strings.Join(strings.Fields(nonAlnum.ReplaceAllString(strings.ToLower(s), " ")), " ")
}
func NormTitle(s string) string { return normTitle(s) }

func Codes(unspsc, gsin []string, gsinMap map[string]string) []string {
	var codes []string
	for _, c := range unspsc {
		if c != "" {
			codes = append(codes, c)
		}
	}
	if len(codes) > 0 {
		return codes
	}
	for _, c := range gsin {
		if u := gsinMap[strings.TrimSpace(c)]; u != "" {
			codes = append(codes, u)
		}
	}
	return codes
}
func unspscDigits(code string) string {
	f := strings.FieldsFunc(code, func(r rune) bool { return r < '0' || r > '9' })
	if len(f) > 0 {
		return f[0]
	}
	return ""
}

func UnspscKeys(codes []string) []string {
	var out, class []string
	seen := map[string]bool{}
	add := func(k string) {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, c := range codes {
		d := unspscDigits(c)
		add(d)
		if len(d) > 6 {
			class = append(class, d[:6])
		}
	}
	for _, k := range class {
		add(k)
	}
	return out
}

var softwareTitleTerms = []string{"software", "platform", "saas", "cloud", "analytics", "machine learning", "portal", "dashboard", "api", "integration", "automation", "licence", "license", "application", "digital", "web", "online"}
var goodsTitleTerms = []string{"construction", "janitorial", "furniture", "vehicle", "ammunition", "snow", "roof", "paving", "uniform", "catering", "pest", "landscaping", "plumbing", "electrical", "forklift", "telehandler", "valve", "jacket", "helicopter", "submarine", "housing", "spare", "scanner", "switch", "heating", "steam", "design-build", "pcr", "microscope", "display", "indicator", "in car", "mail delivery"}

func countTerms(title string, terms []string) (n int) {
	hay := strings.ToLower(strings.ReplaceAll(title, "-", " "))
	for _, k := range terms {
		if strings.Contains(hay, strings.ReplaceAll(k, "-", " ")) {
			n++
		}
	}
	return
}

func breadth(t model.Tender, c Context) Signal {
	n := c.TitleDepts[NormTitle(t.Title)]
	return Signal{Name: "breadth", HasData: true, Raw: fmt.Sprintf("%d department(s)", n), Score: []float64{0.05, 0.15, 0.4, 0.4, 0.7, 0.7, 0.7, 1}[min(n, 7)]}
}

func concentration(t model.Tender, c Context) Signal {
	g := Signal{Name: "concentration", Raw: "no supplier data"}
	if sh, top, key, ok := categoryShare(t, c); ok {
		g.HasData, g.Score, g.Raw = true, sh, fmt.Sprintf("%s %.0f%% of %s", top, sh*100, key)
	}
	return g
}

func bandFit(solic string, cents map[string]int64) Signal {
	g := Signal{Name: "band-fit", Raw: "no award data"}
	v, ok := cents[solic]
	if !ok || solic == "" {
		return g
	}
	k, s := v/100000, 0.2
	switch {
	case k >= 50 && k < 250:
		s = 1
	case k < 50:
		s = 0.4
	case k < 1000:
		s = 0.6
	}
	return Signal{Name: "band-fit", HasData: true, Raw: fmt.Sprintf("$%d", v/100), Score: s}
}

func productFit(c Context, ref string) Signal {
	v, ok := c.ProductFit[ref]
	if !ok {
		return Signal{Name: "product-fit", Raw: "no cached enrichment"}
	}
	shape := strings.ToLower(strings.TrimSpace(c.FitShape[ref]))
	if shape == "resale" || shape == "staffing" || shape == "training" {
		v = min(v, 25)
	}
	return Signal{Name: "product-fit", HasData: true, Score: v / 100, Raw: fmt.Sprintf("%.0f shape %s", v, shape)}
}

// ClassOf returns the 6-digit UNSPSC class of one code, or "".
func ClassOf(code string) string {
	if d := unspscDigits(code); len(d) >= 6 {
		return d[:6]
	}
	return ""
}

// wedge rewards one buyer purchasing deeply in a class other departments
// also buy: buyer depth times category spread, at the notice's own UNSPSC
// granularity. HasData needs repeat buying (depth>=2) and neighbours (>=2).
// depth and spread must come from the same class, or a notice spanning
// several classes could borrow the best depth from one and the best spread
// from another with neither actually being a beachhead.
func wedge(t model.Tender, c Context) Signal {
	g := Signal{Name: "wedge", Raw: "no category repeat"}
	var bestDepth, bestSpread int
	var bestKey string
	bestScore := -1.0
	seen := map[string]bool{}
	for _, code := range Codes(t.UNSPSC, t.GSIN, c.GsinUNSPSC) {
		k := ClassOf(code)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		depth, spread := c.BuyerCat[t.Org+"\x00"+k], c.CatBuyers[k]
		if depth < 2 || spread < 2 {
			continue
		}
		if score := depthNorm(depth) * spreadNorm(spread); score > bestScore {
			bestScore, bestDepth, bestSpread, bestKey = score, depth, spread, k
		}
	}
	if bestScore < 0 {
		return g
	}
	g.HasData = true
	g.Score = bestScore
	g.Raw = fmt.Sprintf("buyerx%d in %s, %d dept(s)", bestDepth, bestKey, bestSpread)
	return g
}

func depthNorm(depth int) float64 {
	switch {
	case depth >= 10:
		return 1
	case depth >= 5:
		return 0.8
	case depth >= 3:
		return 0.6
	}
	return 0.4
}

func spreadNorm(spread int) float64 {
	switch {
	case spread >= 6:
		return 1
	case spread >= 3:
		return 0.7
	}
	return 0.5
}

// urgency scores a sooner future closing higher. Past or missing closings
// carry no data, so the biddable lens never ranks them.
func urgency(t model.Tender) Signal {
	g := Signal{Name: "urgency", Raw: "no closing date"}
	d, ok := closeDate(t.Closing)
	if !ok {
		return g
	}
	days := int(d.Sub(today()).Hours() / 24)
	if days < 0 {
		g.Raw = "closed " + t.Closing[:10]
		return g
	}
	g.HasData = true
	switch {
	case days <= 7:
		g.Score = 1
	case days <= 30:
		g.Score = 0.8
	case days <= 90:
		g.Score = 0.6
	default:
		g.Score = 0.35
	}
	g.Raw = fmt.Sprintf("closes in %dd", days)
	return g
}

// FutureClosing reports whether s closes today or later.
func FutureClosing(s string) bool {
	d, ok := closeDate(s)
	return ok && !d.Before(today())
}

func today() time.Time {
	y, m, d := Now().UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func closeDate(s string) (time.Time, bool) {
	if len(s) < 10 {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", s[:10])
	if err != nil {
		return time.Time{}, false
	}
	return d, true
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
