package cli

import (
	"canadabuys-cli/internal/csvx"
	"canadabuys-cli/internal/llm"
	"canadabuys-cli/internal/model"
	"canadabuys-cli/internal/score"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// oppOut is the JSON row when --llm is on. Without --llm we emit
// []score.Opportunity so that output stays byte-identical to #55.
type oppOut struct {
	score.Opportunity
	Enrichment *llm.Enrichment `json:"enrichment,omitempty"`
}

var coreSignal = map[string]string{"displace": "concentration", "bootstrap": "band-fit", "wedge": "wedge", "biddable": "urgency"}

func opportunitiesCmd() *cobra.Command {
	var minAward, maxAward, since, category, explain, llmModel, lensName string
	var minScore float64
	var limit, llmLimit, llmConcurrency int
	var useLLM, noStaffing, groupSimilar bool
	cmd := &cobra.Command{Use: "opportunities", Short: "Rank tender notices a small software team could win", RunE: func(*cobra.Command, []string) error {
		lens, err := score.LensByName(lensName)
		if err != nil {
			return err
		}
		if useLLM && lens.Name != "default" {
			return fmt.Errorf("--llm issues model requests; lens %q re-ranks cached enrichment only (omit --llm)", lens.Name)
		}
		ctx, err := buildOppContext()
		if err != nil {
			return err
		}
		full, err := queryTenders(&tenderFilter{})
		if err != nil {
			return err
		}
		addRecurYears(ctx, full)
		aggBase, _ := keepLatestAmendments(full)
		addLensAggregates(&ctx, aggBase)
		if lens.Name != "default" {
			loadEnrichCache(&ctx)
		}
		if explain != "" {
			return explainOpp(ctx, explain, lens)
		}
		var minC, maxC int64
		for _, b := range []struct {
			raw, name string
			dst       *int64
		}{{minAward, "min-award", &minC}, {maxAward, "max-award", &maxC}} {
			if b.raw == "" {
				continue
			}
			cents, kind := model.ParseCents(b.raw)
			if kind != model.AmountPositive {
				return fmt.Errorf("%s must be a positive amount in dollars, got %q", b.name, b.raw)
			}
			*b.dst = cents
		}
		f := &tenderFilter{since: since}
		if category != "" {
			f.category = []string{category}
		}
		all, err := queryTenders(f)
		if err != nil {
			return err
		}
		// Collapse amendments before scoring so one solicitation is not rated twice.
		all, amendments := keepLatestAmendments(all)
		similar := 0
		if groupSimilar {
			all, similar = groupSimilarNotices(all)
		}
		out := []score.Opportunity{}
		staffing := 0
		for _, t := range all {
			o, _ := score.ScoreLens(t, ctx, lens.Name) // lens validated above
			if sig, ok := coreSignal[lens.Name]; ok && !haveSignal(o, sig) {
				continue
			}
			if minScore >= 0 && o.Score*100 < minScore {
				continue
			}
			if cents, ok := ctx.AwardCents[t.Solicitation]; minAward+maxAward != "" && (!ok || t.Solicitation == "" || minAward != "" && cents < minC || maxAward != "" && cents > maxC) {
				continue
			}
			if o.Staffing {
				staffing++
				if noStaffing {
					continue
				}
			}
			out = append(out, o)
		}
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Score != out[j].Score {
				return out[i].Score > out[j].Score
			}
			return out[i].Reference < out[j].Reference
		})
		if limit > 0 && len(out) > limit {
			out = out[:limit]
		}
		// Enrichment runs only on the already-ranked shortlist, never
		// the full tender corpus. --llm-limit is the send cap.
		var enr []*llm.Enrichment
		skipped := 0
		if useLLM {
			var e error
			enr, skipped, e = applyLLM(out, llmModel, llmLimit, llmConcurrency)
			if e != nil {
				return e
			}
			fmt.Fprintln(os.Stderr, oppSummary(amendments, similar, staffing, skipped, groupSimilar, true))
		} else {
			fmt.Fprintln(os.Stderr, oppSummary(amendments, similar, staffing, 0, groupSimilar, false))
		}
		if flagJSON {
			if !useLLM {
				return emit(out)
			}
			rows := make([]oppOut, len(out))
			for i, o := range out {
				rows[i] = oppOut{Opportunity: o}
				if i < len(enr) {
					rows[i].Enrichment = enr[i]
				}
			}
			sort.SliceStable(rows, moreProductFit(rows))
			return emit(rows)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if useLLM {
			rows := make([]oppOut, len(out))
			for i, o := range out {
				rows[i] = oppOut{Opportunity: o}
				if i < len(enr) {
					rows[i].Enrichment = enr[i]
				}
			}
			sort.SliceStable(rows, moreProductFit(rows))
			fmt.Fprintln(w, "FIT\tSHAPE\tREFERENCE\tBUYER\tTITLE")
			for _, r := range rows {
				fit, shape := 0.0, ""
				if r.Enrichment != nil {
					fit, shape = r.Enrichment.ProductFit, r.Enrichment.Shape
				}
				fmt.Fprintf(w, "%.0f\t%s\t%s\t%s\t%s\n", fit, shape, r.Reference, r.Buyer, r.Title)
			}
			return w.Flush()
		}
		if lens.Name == "displace" {
			fmt.Fprintln(w, "SCORE\tREFERENCE\tBUYER\tBAND\tINCUMBENT\tTITLE")
			for _, o := range out {
				fmt.Fprintf(w, "%.1f\t%s\t%s\t%s\t%s\t%s\n", o.Score*100, o.Reference, o.Buyer, o.AwardBand, o.Incumbent, o.Title)
			}
			return w.Flush()
		}
		if lens.Name == "biddable" {
			fmt.Fprintln(w, "SCORE\tREFERENCE\tBUYER\tCLOSING\tTITLE")
			for _, o := range out {
				fmt.Fprintf(w, "%.1f\t%s\t%s\t%s\t%s\n", o.Score*100, o.Reference, o.Buyer, o.Closing[:10], o.Title)
			}
			return w.Flush()
		}
		fmt.Fprintln(w, "SCORE\tREFERENCE\tBUYER\tBAND\tTITLE")
		for _, o := range out {
			fmt.Fprintf(w, "%.1f\t%s\t%s\t%s\t%s\n", o.Score*100, o.Reference, o.Buyer, o.AwardBand, o.Title)
		}
		return w.Flush()
	}}
	cmd.Flags().Float64Var(&minScore, "min-score", -1, "minimum score 0-100")
	cmd.Flags().StringVar(&category, "category", "", "filter by category code")
	cmd.Flags().StringVar(&minAward, "min-award", "", "minimum joined award in dollars")
	cmd.Flags().StringVar(&maxAward, "max-award", "", "maximum joined award in dollars")
	cmd.Flags().StringVar(&since, "since", "", "published on or after YYYY-MM-DD")
	cmd.Flags().StringVar(&explain, "explain", "", "print every signal for one referenceNumber")
	cmd.Flags().StringVar(&llmModel, "llm-model", "gpt-oss-120b", "LLM model for --llm enrichment")
	cmd.Flags().IntVar(&limit, "limit", 20, "max rows")
	cmd.Flags().IntVar(&llmLimit, "llm-limit", 100, "max shortlist notices sent to the LLM")
	cmd.Flags().IntVar(&llmConcurrency, "llm-concurrency", 8, "parallel LLM batch requests; 1 is serial")
	cmd.Flags().StringVar(&lensName, "lens", "default", "ranking lens: default|horizontal|displace|bootstrap|wedge|recurring|biddable")
	cmd.Flags().BoolVar(&useLLM, "llm", false, "enrich the shortlist with the LLM provider")
	cmd.Flags().BoolVar(&noStaffing, "exclude-staffing", true, "hide staffing supply arrangements")
	cmd.Flags().BoolVar(&groupSimilar, "group-similar", false, "also collapse notices that share a buyer and title")
	return cmd
}

// keepLatestAmendments keeps one row per real solicitation; placeholders use a namespaced reference.
func keepLatestAmendments(ts []model.Tender) ([]model.Tender, int) {
	return collapseBy(ts, func(t model.Tender) string {
		s := strings.TrimSpace(t.Solicitation)
		switch strings.ToLower(s) {
		case "", "n/a", "na", "tbd", "-", "--", "none", "nil":
			return "\x00" + t.Reference
		}
		return s
	}, newerAmendment)
}

// groupSimilarNotices is opt-in and keyed on buyer plus title, not title alone.
func groupSimilarNotices(ts []model.Tender) ([]model.Tender, int) {
	return collapseBy(ts, func(t model.Tender) string { return t.Org + "\x00" + t.Title }, newerPublication)
}

func collapseBy(ts []model.Tender, key func(model.Tender) string, newer func(a, b model.Tender) bool) ([]model.Tender, int) {
	best := make(map[string]model.Tender, len(ts))
	for _, t := range ts {
		k := key(t)
		if prev, ok := best[k]; !ok || newer(t, prev) {
			best[k] = t
		}
	}
	out := make([]model.Tender, 0, len(best))
	seen := make(map[string]bool, len(best))
	for _, t := range ts {
		k := key(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, best[k])
	}
	return out, len(ts) - len(out)
}

func newerAmendment(a, b model.Tender) bool {
	an, aerr := strconv.Atoi(strings.TrimSpace(a.Amendment))
	bn, berr := strconv.Atoi(strings.TrimSpace(b.Amendment))
	if aerr == nil && berr == nil && an != bn {
		return an > bn
	}
	if a.AmendmentDate != b.AmendmentDate {
		return a.AmendmentDate > b.AmendmentDate
	}
	if a.Publication != b.Publication {
		return a.Publication > b.Publication
	}
	return a.Reference > b.Reference
}

// newerPublication picks the latest published notice, then closing, then reference.
func newerPublication(a, b model.Tender) bool {
	if a.Publication != b.Publication {
		return a.Publication > b.Publication
	}
	if a.Closing != b.Closing {
		return a.Closing > b.Closing
	}
	return a.Reference > b.Reference
}

func productFitOf(r oppOut) float64 {
	if r.Enrichment == nil {
		return -1
	}
	return r.Enrichment.ProductFit
}

func moreProductFit(rows []oppOut) func(int, int) bool {
	return func(i, j int) bool {
		if pi, pj := productFitOf(rows[i]), productFitOf(rows[j]); pi != pj {
			return pi > pj
		}
		return rows[i].Reference < rows[j].Reference
	}
}

func oppSummary(amendments, similar, staffing, skipped int, groupSimilar, useLLM bool) string {
	s := fmt.Sprintf("%d amendment rows collapsed, %d staffing vehicles seen (--exclude-staffing=false to include)", amendments, staffing)
	if groupSimilar {
		s = fmt.Sprintf("%d amendment rows collapsed, %d similar notices grouped, %d staffing vehicles seen (--exclude-staffing=false to include)", amendments, similar, staffing)
	}
	if useLLM {
		s += fmt.Sprintf(", llm skipped %d", skipped)
	}
	return s
}

// applyLLM sends the first llmLimit shortlist rows to the provider.
// CANADABUYS_LLM_API_KEY wins over CEREBRAS_API_KEY when both are set.
func applyLLM(out []score.Opportunity, model string, llmLimit, concurrency int) ([]*llm.Enrichment, int, error) {
	key := os.Getenv("CANADABUYS_LLM_API_KEY")
	if key == "" {
		key = os.Getenv("CEREBRAS_API_KEY")
	}
	if key == "" {
		return nil, 0, fmt.Errorf("set CANADABUYS_LLM_API_KEY or CEREBRAS_API_KEY to use --llm; the deterministic path needs no key")
	}
	dir, err := cacheDir()
	if err != nil {
		return nil, 0, err
	}
	n := min(max(llmLimit, 0), len(out))
	return (&llm.Client{Key: key, CacheDir: filepath.Join(dir, "llm"), Concurrency: concurrency}).Enrich(out[:n], model)
}
func buildOppContext() (score.Context, error) {
	ctx := score.Context{GsinUNSPSC: map[string]string{}, AwardCents: map[string]int64{}, RecurYears: map[string]int{}, CatShare: map[string]float64{}, CatTop: map[string]string{}}
	dir, err := cacheDir()
	if err != nil {
		return ctx, err
	}
	if _, err := os.Stat(csvPath(dir, "gsin-unspsc")); err == nil {
		ctx.GsinUNSPSC = loadGsinMap(csvPath(dir, "gsin-unspsc"))
	}
	ap := csvPath(dir, "awards")
	if _, err := os.Stat(ap); err != nil {
		return ctx, fmt.Errorf("awards not cached (run: fetch awards)")
	}
	catTotal, supCat := map[string]int64{}, map[string]int64{}
	if err := eachAward(ap, func(a model.Award) bool {
		if a.AmountKind != model.AmountPositive {
			return true
		}
		if a.Solicitation != "" {
			ctx.AwardCents[a.Solicitation] = max(ctx.AwardCents[a.Solicitation], a.AmountCents)
		}
		for _, k := range score.UnspscKeys(score.Codes(a.UNSPSC, a.GSIN, ctx.GsinUNSPSC)) {
			catTotal[k] += a.AmountCents
			if a.Supplier != "" {
				supCat[a.Supplier+"\x00"+k] += a.AmountCents
			}
		}
		return true
	}); err != nil {
		return ctx, err
	}
	keys := make([]string, 0, len(supCat))
	for k := range supCat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	topCents := map[string]int64{}
	for _, k := range keys {
		sup, cat, _ := strings.Cut(k, "\x00")
		if supCat[k] > topCents[cat] {
			topCents[cat], ctx.CatTop[cat] = supCat[k], sup
		}
	}
	for cat, tot := range catTotal {
		ctx.CatShare[cat] = model.CategoryShare(topCents[cat], tot)
	}
	return ctx, nil
}
func loadGsinMap(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	r, err := csvx.New(f)
	if err != nil {
		return out
	}
	for {
		if err := r.Next(); err == io.EOF {
			return out
		} else if err != nil {
			return out
		}
		if g, u := r.Get("GSINCode-NIBSCode"), r.Get("UNSPSC-Code"); g != "" && u != "" {
			out[g] = u
		}
	}
}
func addRecurYears(ctx score.Context, all []model.Tender) {
	years := map[string]map[string]bool{}
	for _, t := range all {
		if len(t.Publication) < 4 {
			continue
		}
		k := score.RecurKey(t)
		if years[k] == nil {
			years[k] = map[string]bool{}
		}
		years[k][t.Publication[:4]] = true
	}
	for k, m := range years {
		ctx.RecurYears[k] = len(m)
	}
}
func explainOpp(ctx score.Context, ref string, lens score.Lens) error {
	ex, err := queryTenders(&tenderFilter{exact: ref, limit: 1})
	if err != nil {
		return err
	}
	if len(ex) == 0 {
		return fmt.Errorf("no tender with reference number %q", ref)
	}
	o, _ := score.ScoreLens(ex[0], ctx, lens.Name) // lens validated above
	if flagJSON {
		return emit(o)
	}
	ws := []string{}
	for _, s := range o.Signals {
		ws = append(ws, fmt.Sprintf("%s=%.0f", s.Name, s.Weight))
	}
	fmt.Printf("lens: %s (%s)\nweights: %s\nreference: %s\ntitle: %s\nbuyer: %s\ncategory: %s\nclosing: %s\naward band: %s\nstaffing: %t\nscore: %.1f\n",
		lens.Name, lens.Why, strings.Join(ws, " "), o.Reference, o.Title, o.Buyer, o.Category, o.Closing, o.AwardBand, o.Staffing, o.Score*100)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SIGNAL\tRAW\tWEIGHT\tSCORE\tCONTRIBUTION")
	for _, s := range o.Signals {
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%.2f\t%.3f\n", s.Name, s.Raw, s.Weight, s.Score, s.Contribution)
	}
	return w.Flush()
}

func haveSignal(o score.Opportunity, name string) bool {
	for _, s := range o.Signals {
		if s.Name == name {
			return s.HasData
		}
	}
	return false
}

func addLensAggregates(ctx *score.Context, all []model.Tender) {
	depts := map[string]map[string]bool{}
	ctx.BuyerCat, ctx.CatBuyers = map[string]int{}, map[string]int{}
	seen := map[string]map[string]bool{}
	for _, t := range all {
		if ok, _ := score.IsStaffing(t); ok {
			continue
		}
		k := score.NormTitle(t.Title)
		if depts[k] == nil {
			depts[k] = map[string]bool{}
		}
		depts[k][t.Org] = true
		for _, code := range score.Codes(t.UNSPSC, t.GSIN, ctx.GsinUNSPSC) {
			k := score.ClassOf(code)
			if k == "" {
				continue
			}
			ctx.BuyerCat[t.Org+"\x00"+k]++
			if seen[k] == nil {
				seen[k] = map[string]bool{}
			}
			if !seen[k][t.Org] {
				seen[k][t.Org] = true
				ctx.CatBuyers[k]++
			}
		}
	}
	ctx.TitleDepts = map[string]int{}
	for k, m := range depts {
		ctx.TitleDepts[k] = len(m)
	}
}

func loadEnrichCache(ctx *score.Context) {
	dir, err := cacheDir()
	if err != nil {
		return
	}
	names, _ := filepath.Glob(filepath.Join(dir, "llm", "*"))
	sort.Strings(names)
	ctx.ProductFit, ctx.FitShape = map[string]float64{}, map[string]string{}
	for _, n := range names {
		raw, err := os.ReadFile(n)
		var v llm.Enrichment
		if err != nil || json.Unmarshal(raw, &v) != nil || v.Reference == "" {
			continue
		}
		if _, seen := ctx.ProductFit[v.Reference]; !seen {
			ctx.ProductFit[v.Reference] = max(0, min(100, v.ProductFit))
			ctx.FitShape[v.Reference] = v.Shape
		}
	}
}
