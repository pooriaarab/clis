package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"canadabuys-cli/internal/csvx"
	"canadabuys-cli/internal/model"
)

func awardsCmd() *cobra.Command {
	g := &cobra.Command{Use: "awards", Short: "Query cached award notices"}
	g.AddCommand(awardsListCmd())
	return g
}

type awardFilter struct {
	supplier, org, unspsc, category, currency []string
	since, until, minAmount, maxAmount        string
	minCents, maxCents                        int64
	limit                                     int
}

func (f *awardFilter) addFlags(cmd *cobra.Command) {
	s := cmd.Flags().StringSliceVar
	s(&f.supplier, "supplier", nil, "substring of supplier name (repeatable, OR)")
	s(&f.org, "org", nil, "substring of contracting entity (repeatable, OR)")
	s(&f.unspsc, "unspsc", nil, "UNSPSC code (repeatable, OR)")
	s(&f.category, "category", nil, "procurement category, e.g. *GD (repeatable, OR)")
	s(&f.currency, "currency", nil, "contract currency, e.g. CAD (repeatable, OR)")
	cmd.Flags().StringVar(&f.since, "since", "", "awarded on or after YYYY-MM-DD")
	cmd.Flags().StringVar(&f.until, "until", "", "awarded on or before YYYY-MM-DD")
	cmd.Flags().StringVar(&f.minAmount, "min-amount", "", "minimum contract amount in dollars")
	cmd.Flags().StringVar(&f.maxAmount, "max-amount", "", "maximum contract amount in dollars")
	cmd.Flags().IntVar(&f.limit, "limit", 0, "max rows (0 for no limit)")
}

func (f *awardFilter) boundFlags() error {
	for _, b := range []struct {
		name string
		raw  string
		dst  *int64
	}{{"min-amount", f.minAmount, &f.minCents}, {"max-amount", f.maxAmount, &f.maxCents}} {
		if b.raw == "" {
			continue
		}
		cents, kind := model.ParseCents(b.raw)
		if kind != model.AmountPositive {
			return fmt.Errorf("%s must be a positive amount in dollars, got %q", b.name, b.raw)
		}
		*b.dst = cents
	}
	return nil
}

func awardsListCmd() *cobra.Command {
	var f awardFilter
	cmd := &cobra.Command{Use: "list", Short: "List cached award notices", RunE: func(*cobra.Command, []string) error {
		if err := f.boundFlags(); err != nil {
			return err
		}
		path, err := awardPath()
		if err != nil {
			return err
		}
		out := []model.Award{}
		if err := eachAward(path, func(a model.Award) bool {
			if f.match(a) {
				out = append(out, a)
			}
			return f.limit <= 0 || len(out) < f.limit
		}); err != nil {
			return err
		}
		if flagJSON {
			return emit(out)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "AWARDED\tREFERENCE\tSUPPLIER\tORG\tCURR\tAMOUNT")
		for _, a := range out {
			amt := "(" + a.AmountKind + ")"
			if a.AmountKind == model.AmountPositive {
				amt = fmtDollars(a.AmountCents)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				shortDate(a.AwardDate), a.Reference, a.Supplier, a.Org, a.Currency, amt)
		}
		return w.Flush()
	}}
	f.addFlags(cmd)
	return cmd
}

// match ANDs the fields; a repeated flag ORs within its field.
// Amount bounds only match priced rows: blank is not zero.
func (f *awardFilter) match(a model.Award) bool {
	switch {
	case len(f.supplier) > 0 && !anyMatch(f.supplier, a.Supplier, has):
		return false
	case len(f.org) > 0 && !anyMatch(f.org, a.Org, has):
		return false
	case len(f.unspsc) > 0 && !anySet(a.UNSPSC, f.unspsc, strings.EqualFold):
		return false
	case len(f.category) > 0 && !anySet(a.Categories, f.category, strings.EqualFold):
		return false
	case len(f.currency) > 0 && !anyMatch(f.currency, a.Currency, strings.EqualFold):
		return false
	case !dateWithin(a.AwardDate, f.since, f.until):
		return false
	case f.minAmount != "" && (a.AmountKind != model.AmountPositive || a.AmountCents < f.minCents):
		return false
	case f.maxAmount != "" && (a.AmountKind != model.AmountPositive || a.AmountCents > f.maxCents):
		return false
	}
	return true
}

func awardPath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	p := csvPath(dir, "awards")
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("awards not cached (run: fetch awards)")
	}
	return p, nil
}

// eachAward streams awards through the shared csvx reader: no per-row
// map and no row collection, so the pass stays at reader RSS.
func eachAward(path string, fn func(model.Award) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := csvx.New(f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for {
		err := r.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !fn(model.ParseAward(r.Get)) {
			return nil
		}
	}
}

// fmtDollars truncates cents and groups thousands: 15246578 -> $152,465.
func fmtDollars(c int64) string {
	s := fmt.Sprintf("%d", c/100)
	if len(s) <= 3 {
		return "$" + s
	}
	var b strings.Builder
	b.WriteString("$")
	if r := len(s) % 3; r == 0 {
		b.WriteString(s[:3])
		s = s[3:]
	} else {
		b.WriteString(s[:r])
		s = s[r:]
	}
	for len(s) > 0 {
		b.WriteByte(',')
		b.WriteString(s[:3])
		s = s[3:]
	}
	return b.String()
}
