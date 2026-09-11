package cli

import (
	"canadabuys-cli/internal/csvx"
	"canadabuys-cli/internal/llm"
	"canadabuys-cli/internal/model"
	"canadabuys-cli/internal/score"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
)

// oppOut is the JSON row when --llm is on. Without --llm we emit
// []score.Opportunity so that output stays byte-identical to #55.
type oppOut struct {
	score.Opportunity
	Enrichment *llm.Enrichment `json:"enrichment,omitempty"`
}

func opportunitiesCmd() *cobra.Command {
	var minAward, maxAward, since, category, explain, llmModel string
	var minScore float64
	var limit, llmLimit int
	var useLLM, noStaffing bool
	cmd := &cobra.Command{Use: "opportunities", Short: "Rank tender notices a small software team could win", RunE: func(*cobra.Command, []string) error {
		ctx, err := buildOppContext()
		if err != nil {
			return err
		}
		if explain != "" {
			return explainOpp(ctx, explain)
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
		full, err := queryTenders(&tenderFilter{})
		if err != nil {
			return err
		}
		addRecurYears(ctx, full)
		f := &tenderFilter{since: since}
		if category != "" {
			f.category = []string{category}
		}
		all, err := queryTenders(f)
		if err != nil {
			return err
		}
		out := []score.Opportunity{}
		staffing := 0
		for _, t := range all {
			o := score.Score(t, ctx)
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
			enr, skipped, e = applyLLM(out, llmModel, llmLimit)
			if e != nil {
				return e
			}
			fmt.Fprintf(os.Stderr, "%d staffing vehicles seen (--exclude-staffing=false to include), llm skipped %d\n", staffing, skipped)
		} else {
			fmt.Fprintf(os.Stderr, "%d staffing vehicles seen (--exclude-staffing=false to include)\n", staffing)
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
			return emit(rows)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
	cmd.Flags().BoolVar(&useLLM, "llm", false, "enrich the shortlist with the LLM provider")
	cmd.Flags().BoolVar(&noStaffing, "exclude-staffing", true, "hide staffing supply arrangements")
	return cmd
}

// applyLLM sends the first llmLimit shortlist rows to the provider.
// CANADABUYS_LLM_API_KEY wins over CEREBRAS_API_KEY when both are set.
func applyLLM(out []score.Opportunity, model string, llmLimit int) ([]*llm.Enrichment, int, error) {
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
	return (&llm.Client{Key: key, CacheDir: filepath.Join(dir, "llm")}).Enrich(out[:n], model)
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
		for _, c := range awardCats(a.Categories) {
			catTotal[c] += a.AmountCents
			if a.Supplier != "" {
				supCat[a.Supplier+"\x00"+c] += a.AmountCents
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
func explainOpp(ctx score.Context, ref string) error {
	all, err := queryTenders(&tenderFilter{})
	if err != nil {
		return err
	}
	addRecurYears(ctx, all)
	ex, err := queryTenders(&tenderFilter{exact: ref, limit: 1})
	if err != nil {
		return err
	}
	if len(ex) == 0 {
		return fmt.Errorf("no tender with reference number %q", ref)
	}
	o := score.Score(ex[0], ctx)
	if flagJSON {
		return emit(o)
	}
	fmt.Printf("reference: %s\ntitle: %s\nbuyer: %s\ncategory: %s\nclosing: %s\naward band: %s\nstaffing: %t\nscore: %.1f\n",
		o.Reference, o.Title, o.Buyer, o.Category, o.Closing, o.AwardBand, o.Staffing, o.Score*100)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SIGNAL\tRAW\tWEIGHT\tSCORE\tCONTRIBUTION")
	for _, s := range o.Signals {
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%.2f\t%.3f\n", s.Name, s.Raw, s.Weight, s.Score, s.Contribution)
	}
	return w.Flush()
}
