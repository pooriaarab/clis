package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"canadabuys-cli/internal/csvx"
	"canadabuys-cli/internal/model"
)

var defaultTenderDatasets = []string{"tenders", "tenders-legacy"}

type tenderFilter struct {
	status, category, unspsc, gsin, org, region, notice, q, datasets []string
	since, until, closingAfter, closingBefore, sort, exact           string
	limit                                                            int
}

func tendersCmd() *cobra.Command {
	g := &cobra.Command{Use: "tenders", Short: "Query cached tender notices"}
	g.AddCommand(tendersListCmd(), tendersShowCmd())
	return g
}

func (f *tenderFilter) addFlags(cmd *cobra.Command) {
	s := cmd.Flags().StringSliceVar
	s(&f.status, "status", nil, "tender status (repeatable, OR)")
	s(&f.category, "category", nil, "procurement category code, e.g. *GD (repeatable, OR)")
	s(&f.unspsc, "unspsc", nil, "UNSPSC code (repeatable, OR)")
	s(&f.gsin, "gsin", nil, "GSIN code (repeatable, OR)")
	s(&f.org, "org", nil, "substring of contracting entity (repeatable, OR)")
	s(&f.region, "region", nil, "region of delivery (repeatable, OR)")
	s(&f.notice, "notice-type", nil, "substring of notice type (repeatable, OR)")
	s(&f.q, "q", nil, "substring over title and description (repeatable, OR)")
	s(&f.datasets, "dataset", nil, "cached datasets to read (default tenders,tenders-legacy)")
	cmd.Flags().StringVar(&f.since, "since", "", "published on or after YYYY-MM-DD")
	cmd.Flags().StringVar(&f.until, "until", "", "published on or before YYYY-MM-DD")
	cmd.Flags().StringVar(&f.closingAfter, "closing-after", "", "closes on or after YYYY-MM-DD")
	cmd.Flags().StringVar(&f.closingBefore, "closing-before", "", "closes on or before YYYY-MM-DD")
	cmd.Flags().StringVar(&f.sort, "sort", "", "publication|closing|reference|title, prefix - for descending")
	cmd.Flags().IntVar(&f.limit, "limit", 0, "max rows (0 for no limit)")
}

func tendersListCmd() *cobra.Command {
	var f tenderFilter
	cmd := &cobra.Command{Use: "list", Short: "List cached tender notices", RunE: func(*cobra.Command, []string) error {
		out, err := queryTenders(&f)
		if err != nil {
			return err
		}
		if flagJSON {
			return emitTenders(out)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "REFERENCE\tSTATUS\tPUBLISHED\tCLOSING\tORG\tTITLE")
		for _, t := range out {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				t.Reference, t.Status, shortDate(t.Publication), shortDate(t.Closing), t.Org, t.Title)
		}
		return w.Flush()
	}}
	f.addFlags(cmd)
	return cmd
}

func tendersShowCmd() *cobra.Command {
	var f tenderFilter
	cmd := &cobra.Command{Use: "show <referenceNumber>", Short: "Show one tender notice in full", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		f.exact, f.limit, f.sort = args[0], 1, ""
		out, err := queryTenders(&f)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return fmt.Errorf("no tender with reference number %q", args[0])
		}
		if flagJSON {
			return emit(out[0])
		}
		printTenderFull(out[0])
		return nil
	}}
	cmd.Flags().StringSliceVar(&f.datasets, "dataset", nil, "cached datasets to read (default tenders,tenders-legacy)")
	return cmd
}

func queryTenders(f *tenderFilter) ([]model.Tender, error) {
	for _, p := range [][2]string{{"--since", f.since}, {"--until", f.until}, {"--closing-after", f.closingAfter}, {"--closing-before", f.closingBefore}} {
		if p[1] != "" {
			if _, err := time.Parse("2006-01-02", p[1]); err != nil {
				return nil, fmt.Errorf("%s: %q is not YYYY-MM-DD", p[0], p[1])
			}
		}
	}
	paths, err := tenderPaths(f.datasets)
	if err != nil {
		return nil, err
	}
	out := []model.Tender{}
	seen, done := map[string]bool{}, false
	for _, p := range paths {
		if err := eachTender(p, func(t model.Tender) bool {
			if !seen[t.Reference] {
				seen[t.Reference] = true
				if f.match(t) {
					out = append(out, t)
				}
			}
			if f.sort == "" && f.limit > 0 && len(out) >= f.limit {
				done = true
				return false
			}
			return true
		}); err != nil {
			return nil, err
		}
		if done {
			break
		}
	}
	if err := f.sortTenders(out); err != nil {
		return nil, err
	}
	if f.limit > 0 && len(out) > f.limit {
		out = out[:f.limit]
	}
	return out, nil
}

func tenderPaths(ids []string) ([]string, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		ids = defaultTenderDatasets
	}
	var paths []string
	for _, id := range ids {
		d, err := lookup(id)
		if err != nil {
			return nil, err
		}
		if d.Kind != "tenders" {
			return nil, fmt.Errorf("%s is kind %s, not tenders", id, d.Kind)
		}
		p := csvPath(dir, id)
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("%s: %s is missing or unreadable (run: canadabuys fetch %s)", id, p, id)
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no cached tender datasets (run: fetch tenders tenders-legacy)")
	}
	return paths, nil
}

func eachTender(path string, fn func(model.Tender) bool) error {
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
		if err := r.Next(); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !fn(model.Parse(r.Get)) {
			return nil
		}
	}
}

// emitTenders writes one object at a time. Bytes match json.Encoder
// with SetIndent("", "  "), including the trailing newline.
func emitTenders(ts []model.Tender) error {
	if len(ts) == 0 {
		_, err := io.WriteString(os.Stdout, "[]\n")
		return err
	}
	if _, err := io.WriteString(os.Stdout, "[\n"); err != nil {
		return err
	}
	for i, t := range ts {
		if i > 0 {
			if _, err := io.WriteString(os.Stdout, ",\n"); err != nil {
				return err
			}
		}
		b, err := json.MarshalIndent(t, "  ", "  ")
		if err != nil {
			return err
		}
		if _, err := os.Stdout.Write(append([]byte("  "), b...)); err != nil {
			return err
		}
	}
	_, err := io.WriteString(os.Stdout, "\n]\n")
	return err
}

// match ANDs the fields; a repeated flag ORs within its field.
func (f *tenderFilter) match(t model.Tender) bool {
	switch {
	case f.exact != "" && t.Reference != f.exact:
		return false
	case len(f.status) > 0 && !anyMatch(f.status, t.Status, strings.EqualFold):
		return false
	case len(f.category) > 0 && !anySet(t.Categories, f.category, strings.EqualFold):
		return false
	case len(f.unspsc) > 0 && !anySet(model.SplitSet(t.UNSPSC), f.unspsc, strings.EqualFold):
		return false
	case len(f.gsin) > 0 && !anySet(model.SplitSet(t.GSIN), f.gsin, strings.EqualFold):
		return false
	case len(f.org) > 0 && !anyMatch(f.org, t.Org, has):
		return false
	case len(f.region) > 0 && !anySet(t.Regions, f.region, has):
		return false
	case len(f.notice) > 0 && !anyMatch(f.notice, t.NoticeType, has):
		return false
	case !dateWithin(t.Publication, f.since, f.until):
		return false
	case !dateWithin(t.Closing, f.closingAfter, f.closingBefore):
		return false
	}
	if len(f.q) > 0 {
		hay := t.Title + "\n" + t.TitleFR + "\n" + t.Description + "\n" + t.DescriptionFR
		return anyMatch(f.q, hay, has)
	}
	return true
}

func (f *tenderFilter) sortTenders(out []model.Tender) error {
	if f.sort == "" {
		return nil
	}
	key, desc := f.sort, false
	if strings.HasPrefix(key, "-") {
		key, desc = key[1:], true
	}
	val := map[string]func(model.Tender) string{
		"publication": func(t model.Tender) string { return t.Publication },
		"closing":     func(t model.Tender) string { return t.Closing },
		"reference":   func(t model.Tender) string { return t.Reference },
		"title":       func(t model.Tender) string { return t.Title },
	}[key]
	if val == nil {
		return fmt.Errorf("unknown sort %q (publication, closing, reference, title)", f.sort)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if desc {
			return val(out[j]) < val(out[i])
		}
		return val(out[i]) < val(out[j])
	})
	return nil
}

func printTenderFull(t model.Tender) {
	rows := [][2]string{
		{"Reference", t.Reference}, {"Solicitation", t.Solicitation},
		{"Title", t.Title}, {"Title (FR)", t.TitleFR},
		{"Status", t.Status}, {"Categories", strings.Join(t.Categories, "; ")},
		{"UNSPSC", t.UNSPSC}, {"UNSPSC description", t.UNSPSCDesc},
		{"GSIN", t.GSIN}, {"GSIN description", t.GSINDesc},
		{"Notice type", t.NoticeType}, {"Method", t.Method},
		{"Contracting entity", t.Org}, {"End user", t.EndUser},
		{"Regions of delivery", strings.Join(t.Regions, "; ")}, {"Trade agreements", strings.Join(t.TradeAgreements, "; ")},
		{"Published", t.Publication}, {"Closing", t.Closing},
		{"Contract start", t.ContractStart}, {"Description", t.Description},
		{"Description (FR)", t.DescriptionFR},
	}
	for _, r := range rows {
		fmt.Printf("%-18s %s\n", r[0]+":", r[1])
	}
}

func anyMatch(vals []string, s string, rel func(a, b string) bool) bool {
	for _, v := range vals {
		if rel(v, s) {
			return true
		}
	}
	return false
}

func anySet(set, vals []string, rel func(a, b string) bool) bool {
	for _, m := range set {
		if anyMatch(vals, m, rel) {
			return true
		}
	}
	return false
}

func has(a, b string) bool { return strings.Contains(strings.ToLower(b), strings.ToLower(a)) }

func dateWithin(v, since, until string) bool {
	if since == "" && until == "" {
		return true
	}
	d := shortDate(v)
	return d != "" && (since == "" || d >= since) && (until == "" || d <= until)
}

func shortDate(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}
