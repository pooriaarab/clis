// merx harvest walks one status through published-date RANGE slices and
// records a resumable local corpus. This file is the planner and store:
// it decides which slices to fetch. The fetch loop is a later change.
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"merx-cli/internal/search"
)

const (
	harvestDateLayout = "2006-01-02"
	// harvestEpoch is the documented --since default, not MERX's earliest notice.
	harvestEpoch = "2020-01-01"
)

// harvestStatus is the public --status names. The private search codes
// that MERX posts (OPEN, CLOSED, AWARD) belong with the fetch loop.
var harvestStatus = map[string]bool{
	"open": true, "closed": true, "awarded": true, "bid-results": true,
}

// Slice is one published-date RANGE query.
type Slice struct {
	Status string `json:"status"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

// Entry is one manifest row per finished leaf slice.
type Entry struct {
	Status      string `json:"status"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Reported    int    `json:"reported"`
	Captured    int    `json:"captured"`
	CompletedAt string `json:"completed_at"`
	Incomplete  bool   `json:"incomplete"`
}

// Summary is the machine-readable harvest result.
type Summary struct {
	Status     string `json:"status"`
	Slices     int    `json:"slices"`
	Reported   int    `json:"reported"`
	Captured   int    `json:"captured"`
	Incomplete int    `json:"incomplete"`
}

// Manifest is the on-disk harvest index. Since/Until are the requested
// coverage window, so a store that starts at 2020 documents that bound.
type Manifest struct {
	Since  string  `json:"since"`
	Until  string  `json:"until"`
	Slices []Entry `json:"slices"`
}

func key(s Slice) string { return s.Start + "\x00" + s.End }

// monthSlices covers since through until with calendar months.
func monthSlices(status string, since, until time.Time) []Slice {
	since = since.UTC().Truncate(24 * time.Hour)
	until = until.UTC().Truncate(24 * time.Hour)
	var out []Slice
	for cur := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.UTC); !cur.After(until); cur = cur.AddDate(0, 1, 0) {
		start, end := cur, cur.AddDate(0, 1, -1)
		if start.Before(since) {
			start = since
		}
		if end.After(until) {
			end = until
		}
		out = append(out, Slice{Status: status, Start: start.Format(harvestDateLayout), End: end.Format(harvestDateLayout)})
	}
	return out
}

// harvestDate parses a --since/--until value. Fail loudly on anything but YYYY-MM-DD.
func harvestDate(flag, value string) (time.Time, error) {
	t, err := time.Parse(harvestDateLayout, value)
	if err != nil || t.Format(harvestDateLayout) != value {
		return time.Time{}, fmt.Errorf("invalid --%s %q (want YYYY-MM-DD)", flag, value)
	}
	return t, nil
}

// split halves s on a day boundary. False when s is a single day.
func split(s Slice) (Slice, Slice, bool) {
	a, errA := time.Parse(harvestDateLayout, s.Start)
	b, errB := time.Parse(harvestDateLayout, s.End)
	if errA != nil || errB != nil || !a.Before(b) {
		return Slice{}, Slice{}, false
	}
	mid := a.AddDate(0, 0, int(b.Sub(a).Hours()/24)/2)
	l, r := s, s
	l.End = mid.Format(harvestDateLayout)
	r.Start = mid.AddDate(0, 0, 1).Format(harvestDateLayout)
	return l, r, true
}

// narrow splits s until every leaf reports at most 1,000 records.
// A leaf that is already a single day is returned as-is; the caller
// must record it incomplete. Returns leaves plus each leaf's reported total.
func narrow(s Slice, count func(Slice) (int, error)) ([]Slice, map[string]int, error) {
	n, err := count(s)
	if err != nil {
		return nil, nil, err
	}
	if n <= search.MaxReachable {
		return []Slice{s}, map[string]int{key(s): n}, nil
	}
	l, r, ok := split(s)
	if !ok {
		return []Slice{s}, map[string]int{key(s): n}, nil
	}
	ll, lm, err := narrow(l, count)
	if err != nil {
		return nil, nil, err
	}
	rl, rm, err := narrow(r, count)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range rm {
		lm[k] = v
	}
	return append(ll, rl...), lm, nil
}

// incomplete reports whether a leaf still overflows what MERX can serve.
func incomplete(s Slice, reported int) bool {
	_, _, ok := split(s)
	return reported > search.MaxReachable && !ok
}

func harvestPaths(dir, status string) (string, string) {
	base := filepath.Join(dir, "harvest", status)
	return base + ".ndjson", base + ".manifest.json"
}

func loadIDs(path string) (map[string]bool, error) {
	known := map[string]bool{}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return known, nil
		}
		return nil, err
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var r search.Record
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("stored record is corrupt: %w", err)
		}
		known[r.InternalID] = true
	}
	return known, nil
}

func loadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, nil
		}
		return Manifest{}, err
	}
	var out Manifest
	if err := json.Unmarshal(raw, &out); err != nil {
		return Manifest{}, fmt.Errorf("stored manifest is corrupt: %w", err)
	}
	return out, nil
}

// saveManifest writes atomically so a killed run keeps a valid manifest.
func saveManifest(path string, m Manifest) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func appendRecord(f *os.File, r search.Record) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// plan narrows each seed with count and drops leaves already finished
// (done is keyed the same way as key). Tests inject count so planning
// never needs the network.
func plan(seeds []Slice, count func(Slice) (int, error), done map[string]bool) ([]Slice, map[string]int, error) {
	var out []Slice
	totals := map[string]int{}
	for _, s := range seeds {
		leaves, n, err := narrow(s, count)
		if err != nil {
			return nil, nil, err
		}
		for _, leaf := range leaves {
			k := key(leaf)
			if done[k] {
				continue
			}
			out = append(out, leaf)
			totals[k] = n[k]
		}
	}
	return out, totals, nil
}

// capturedCount is how many records MERX can actually serve from reported.
func capturedCount(reported int) int {
	if reported > search.MaxReachable {
		return search.MaxReachable
	}
	return reported
}

func harvestCmd() *cobra.Command {
	var status, since, until string
	var dry bool
	cmd := &cobra.Command{Use: "harvest", Short: "Build a resumable notice corpus for one status (--since / --until bound the window)", RunE: func(cmd *cobra.Command, _ []string) error {
		if !harvestStatus[status] {
			return fmt.Errorf("unknown --status %q (open, awarded, bid-results, closed)", status)
		}
		start, err := harvestDate("since", since)
		if err != nil {
			return err
		}
		if until == "" {
			until = time.Now().UTC().Format(harvestDateLayout)
		}
		end, err := harvestDate("until", until)
		if err != nil {
			return err
		}
		if start.After(end) {
			return fmt.Errorf("--since %s is after --until %s", since, until)
		}
		if !dry {
			return fmt.Errorf("harvest fetch is not implemented; pass --dry-run to print the slice plan")
		}
		seeds := monthSlices(status, start, end)
		// Zero count: every calendar month is already a leaf. No network.
		leaves, _, err := plan(seeds, func(Slice) (int, error) { return 0, nil }, nil)
		if err != nil {
			return err
		}
		if flagJSON {
			return emit(map[string]any{"status": status, "since": since, "until": until, "slices": leaves})
		}
		for _, s := range leaves {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", s.Status, s.Start, s.End)
		}
		return nil
	}}
	cmd.Flags().StringVar(&status, "status", "open", "harvest: open, awarded, bid-results, closed")
	cmd.Flags().BoolVar(&dry, "dry-run", false, "print planned slices without fetching")
	cmd.Flags().StringVar(&since, "since", harvestEpoch, "first published date YYYY-MM-DD")
	cmd.Flags().StringVar(&until, "until", "", "last published date YYYY-MM-DD (default today)")
	return cmd
}
