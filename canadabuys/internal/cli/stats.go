package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"canadabuys-cli/internal/model"
)

func statsCmd() *cobra.Command {
	g := &cobra.Command{Use: "stats", Short: "Statistics over cached notices"}
	g.AddCommand(statsAwardsCmd(), statsLeaderCmd("buyers", "org", "ENTITY"), statsLeaderCmd("suppliers", "supplier", "SUPPLIER"))
	return g
}

type awardScan struct {
	rows, blank, zero, negative, invalid int
	priced                               []int64
	total                                int64
	byCcy                                map[string][2]int64 // count, total cents
}

// scanAwards streams one pass: priced cents for percentiles plus the
// blank/zero/negative/invalid counters and per-currency subtotals.
func scanAwards(path string) (*awardScan, error) {
	s := &awardScan{byCcy: map[string][2]int64{}}
	err := eachAward(path, func(a model.Award) bool {
		s.rows++
		if a.AmountKind != model.AmountPositive {
			switch a.AmountKind {
			case model.AmountBlank:
				s.blank++
			case model.AmountZero:
				s.zero++
			case model.AmountNegative:
				s.negative++
			default:
				s.invalid++
			}
			return true
		}
		s.priced = append(s.priced, a.AmountCents)
		s.total += a.AmountCents
		g := s.byCcy[unknown(a.Currency)]
		s.byCcy[unknown(a.Currency)] = [2]int64{g[0] + 1, g[1] + a.AmountCents}
		return true
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(s.priced, func(i, j int) bool { return s.priced[i] < s.priced[j] })
	return s, nil
}

// percentile is nearest-rank over ascending cents: index ceil(p*n/100)-1.
func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if i := (p*len(sorted)+99)/100 - 1; i > 0 {
		return sorted[i]
	}
	return sorted[0]
}

type ccyRow struct {
	Currency   string `json:"currency"`
	Notices    int    `json:"notices"`
	TotalCents int64  `json:"total_cents"`
	Total      string `json:"total"`
}

type awardSummary struct {
	Rows        int      `json:"rows"`
	Priced      int      `json:"priced"`
	Blank       int      `json:"blank"`
	Zero        int      `json:"zero"`
	Negative    int      `json:"negative"`
	Invalid     int      `json:"invalid"`
	Total       string   `json:"total"`
	TotalCents  int64    `json:"total_cents"`
	Currency    string   `json:"currency"`
	Mean        string   `json:"mean"`
	MeanCents   int64    `json:"mean_cents"`
	Median      string   `json:"median"`
	MedianCents int64    `json:"median_cents"`
	P25         string   `json:"p25"`
	P25Cents    int64    `json:"p25_cents"`
	P75         string   `json:"p75"`
	P75Cents    int64    `json:"p75_cents"`
	P90         string   `json:"p90"`
	P90Cents    int64    `json:"p90_cents"`
	P95         string   `json:"p95"`
	P95Cents    int64    `json:"p95_cents"`
	P99         string   `json:"p99"`
	P99Cents    int64    `json:"p99_cents"`
	Max         string   `json:"max"`
	MaxCents    int64    `json:"max_cents"`
	ByCurrency  []ccyRow `json:"by_currency"`
}

func (s *awardScan) summary() awardSummary {
	n := len(s.priced)
	out := awardSummary{Currency: "mixed", Rows: s.rows, Priced: n}
	out.Blank, out.Zero, out.Negative, out.Invalid = s.blank, s.zero, s.negative, s.invalid
	out.TotalCents, out.Total = s.total, fmtBillions(s.total)
	if n > 0 {
		out.MeanCents = (s.total + int64(n)/2) / int64(n)
		out.MedianCents, out.P25Cents, out.P75Cents = percentile(s.priced, 50), percentile(s.priced, 25), percentile(s.priced, 75)
		out.P90Cents, out.P95Cents, out.P99Cents = percentile(s.priced, 90), percentile(s.priced, 95), percentile(s.priced, 99)
		out.MaxCents = s.priced[n-1]
	}
	out.Mean, out.Median = fmtDollars(out.MeanCents), fmtDollars(out.MedianCents)
	out.P25, out.P75 = fmtDollars(out.P25Cents), fmtDollars(out.P75Cents)
	out.P90, out.P95 = fmtDollars(out.P90Cents), fmtDollars(out.P95Cents)
	out.P99, out.Max = fmtDollars(out.P99Cents), fmtDollars(out.MaxCents)
	for _, k := range sortedCcy(s.byCcy) {
		g := s.byCcy[k]
		out.ByCurrency = append(out.ByCurrency, ccyRow{k, int(g[0]), g[1], fmtBillions(g[1])})
	}
	return out
}

func sortedCcy(m map[string][2]int64) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func statsAwardsCmd() *cobra.Command {
	var by string
	var limit int
	cmd := &cobra.Command{Use: "awards", Short: "Award size statistics", RunE: func(*cobra.Command, []string) error {
		path, err := awardPath()
		if err != nil {
			return err
		}
		if by == "" {
			s, err := scanAwards(path)
			if err != nil {
				return err
			}
			sum := s.summary()
			if flagJSON {
				return emit(sum)
			}
			fmt.Printf("rows: %d  priced: %d  blank: %d  zero: %d  negative: %d  invalid: %d\n",
				sum.Rows, sum.Priced, sum.Blank, sum.Zero, sum.Negative, sum.Invalid)
			fmt.Printf("total (%s currencies): %s\n", sum.Currency, sum.Total)
			fmt.Printf("mean: %s  median: %s  p25: %s  p75: %s  p90: %s  p95: %s  p99: %s  max: %s\n",
				sum.Mean, sum.Median, sum.P25, sum.P75, sum.P90, sum.P95, sum.P99, sum.Max)
			fmt.Println("by currency:")
			for _, c := range sum.ByCurrency {
				fmt.Printf("  %s: %d notices, %s\n", c.Currency, c.Notices, c.Total)
			}
			fmt.Println("note: the total mixes contract currencies; use awards list --currency to isolate one.")
			return nil
		}
		switch by {
		case "category", "unspsc", "gsin", "org", "supplier", "year":
		default:
			return fmt.Errorf("unknown --by %q (category, unspsc, gsin, org, supplier, year)", by)
		}
		bs, err := rankAwards(path, by)
		if err != nil {
			return err
		}
		return printBuckets(bs, "GROUP", by == "supplier", limit)
	}}
	cmd.Flags().StringVar(&by, "by", "", "group by category|unspsc|gsin|org|supplier|year")
	cmd.Flags().IntVar(&limit, "limit", 0, "max groups (0 for no limit)")
	return cmd
}

type bucket struct {
	Name            string  `json:"name"`
	Notices         int     `json:"notices"`
	TotalCents      int64   `json:"total_cents"`
	Share           float64 `json:"share_of_category,omitempty"`
	ShareKey        string  `json:"share_category,omitempty"`
	ShareTotalCents int64   `json:"share_category_total_cents,omitempty"`
}

// rankAwards aggregates priced rows by one dimension, largest total first.
// A multi-valued row (category, unspsc, gsin) credits its full amount to
// each listed key, so group totals can exceed the corpus total.
// With by=supplier each row also carries its share of its top category.
func rankAwards(path string, by string) ([]bucket, error) {
	agg := map[string]*bucket{}
	catOf, catTotals := map[string]map[string]int64{}, map[string]int64{}
	err := eachAward(path, func(a model.Award) bool {
		if a.AmountKind != model.AmountPositive {
			return true
		}
		keys := rankKeys(by, a)
		for _, k := range keys {
			b := agg[k]
			if b == nil {
				b = &bucket{Name: k}
				agg[k] = b
			}
			b.Notices++
			b.TotalCents += a.AmountCents
		}
		if by == "supplier" {
			for _, c := range awardCats(a.Categories) {
				catTotals[c] += a.AmountCents
				m := catOf[keys[0]]
				if m == nil {
					m = map[string]int64{}
					catOf[keys[0]] = m
				}
				m[c] += a.AmountCents
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	out := make([]bucket, 0, len(agg))
	for _, b := range agg {
		if by == "supplier" {
			cats := make([]string, 0, len(catOf[b.Name]))
			for c := range catOf[b.Name] {
				cats = append(cats, c)
			}
			sort.Strings(cats)
			top, topTotal := "", int64(0)
			for _, c := range cats {
				if v := catOf[b.Name][c]; v > topTotal {
					top, topTotal = c, v
				}
			}
			b.Share, b.ShareKey, b.ShareTotalCents = model.CategoryShare(topTotal, catTotals[top]), top, catTotals[top]
		}
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalCents > out[j].TotalCents })
	return out, nil
}

func rankKeys(by string, a model.Award) []string {
	switch by {
	case "category":
		return awardCats(a.Categories)
	case "unspsc":
		return awardCats(a.UNSPSC)
	case "gsin":
		return awardCats(a.GSIN)
	case "org":
		return []string{unknown(a.Org)}
	case "supplier":
		return []string{unknown(a.Supplier)}
	case "year":
		if len(a.AwardDate) >= 4 {
			return []string{a.AwardDate[:4]}
		}
		return []string{"unknown"}
	}
	return nil
}

// awardCats is a multi-value cell as grouping keys. An empty cell is
// one "unknown" group; each listed value is its own group.
func awardCats(vals []string) []string {
	if len(vals) == 0 {
		return []string{"unknown"}
	}
	return vals
}

func unknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func statsLeaderCmd(use, by, title string) *cobra.Command {
	var limit int
	share := by == "supplier"
	cmd := &cobra.Command{Use: use, Short: title + " ranked by award value", RunE: func(*cobra.Command, []string) error {
		path, err := awardPath()
		if err != nil {
			return err
		}
		bs, err := rankAwards(path, by)
		if err != nil {
			return err
		}
		return printBuckets(bs, title, share, limit)
	}}
	cmd.Flags().IntVar(&limit, "limit", 0, "max rows (0 for no limit)")
	return cmd
}

func printBuckets(bs []bucket, title string, share bool, limit int) error {
	if limit > 0 && len(bs) > limit {
		bs = bs[:limit]
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if share {
		fmt.Fprintln(w, title+"\tNOTICES\tTOTAL\tTOP CATEGORY\tCATEGORY TOTAL\tSHARE")
	} else {
		fmt.Fprintln(w, title+"\tNOTICES\tTOTAL")
	}
	for _, b := range bs {
		if share {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%.1f%%\n",
				b.Name, b.Notices, fmtDollars(b.TotalCents), b.ShareKey, fmtDollars(b.ShareTotalCents), b.Share*100)
		} else {
			fmt.Fprintf(w, "%s\t%d\t%s\n", b.Name, b.Notices, fmtDollars(b.TotalCents))
		}
	}
	return w.Flush()
}

// fmtBillions is display only; the running total stays integer cents.
func fmtBillions(c int64) string { return fmt.Sprintf("$%.2fB", float64(c)/1e11) }
