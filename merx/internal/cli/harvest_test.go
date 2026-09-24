package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"merx-cli/internal/search"
)

// daysPer records 100 notices per day in the slice.
func daysPer(s Slice) (int, error) {
	a, err := time.Parse(harvestDateLayout, s.Start)
	if err != nil {
		return 0, err
	}
	b, err := time.Parse(harvestDateLayout, s.End)
	if err != nil {
		return 0, err
	}
	return 100 * (int(b.Sub(a).Hours()/24) + 1), nil
}

func TestNarrowSplitsOverflow(t *testing.T) {
	s := Slice{Status: "open", Start: "2020-01-01", End: "2020-01-31"}
	leaves, totals, err := plan([]Slice{s}, daysPer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaves) < 2 {
		t.Fatalf("3100-record month was not narrowed: %v", leaves)
	}
	cover := 0
	prev := ""
	for i, l := range leaves {
		n := totals[key(l)]
		if n > search.MaxReachable {
			t.Fatalf("leaf %v still reports %d", l, n)
		}
		a, _ := time.Parse(harvestDateLayout, l.Start)
		b, _ := time.Parse(harvestDateLayout, l.End)
		cover += int(b.Sub(a).Hours()/24) + 1
		if i > 0 && l.Start <= prev {
			t.Fatalf("leaves overlap or gap: %v", leaves)
		}
		prev = l.End
	}
	if leaves[0].Start != "2020-01-01" || prev != "2020-01-31" || cover != 31 {
		t.Fatalf("narrowing truncated the range: %v", leaves)
	}
}

func TestUnnarrowableSliceIncomplete(t *testing.T) {
	dir := t.TempDir()
	const rep = 5000
	count := func(Slice) (int, error) { return rep, nil }
	s := Slice{Status: "open", Start: "2020-01-01", End: "2020-01-01"}
	leaves, totals, err := plan([]Slice{s}, count, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaves) != 1 {
		t.Fatalf("single-day slice must stay one leaf: %v", leaves)
	}
	got := totals[key(leaves[0])]
	if !incomplete(leaves[0], got) {
		t.Fatalf("single-day %d-record slice must be incomplete", got)
	}
	e := Entry{Status: "open", Start: s.Start, End: s.End,
		Reported: got, Captured: capturedCount(got), Incomplete: true}
	_, manifest := harvestPaths(dir, "open")
	if err := saveManifest(manifest, Manifest{Slices: []Entry{e}}); err != nil {
		t.Fatal(err)
	}
	stored, err := loadManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Slices) != 1 {
		t.Fatalf("entries %v", stored.Slices)
	}
	if !stored.Slices[0].Incomplete || stored.Slices[0].Reported != rep || stored.Slices[0].Captured != search.MaxReachable {
		t.Fatalf("entry must carry both numbers: %+v", stored.Slices[0])
	}
}

func writeStore(t *testing.T, dir, status string, ids ...string) {
	t.Helper()
	store, _ := harvestPaths(dir, status)
	if err := os.MkdirAll(filepath.Dir(store), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(store, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, id := range ids {
		if err := appendRecord(f, search.Record{InternalID: id, DetailURL: "https://www.merx.com/d/" + id}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResumeSkipsStoredIDs(t *testing.T) {
	dir := t.TempDir()
	writeStore(t, dir, "open", "old-1", "old-2")
	store, manifest := harvestPaths(dir, "open")
	known, err := loadIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 2 || !known["old-1"] || !known["old-2"] {
		t.Fatalf("store ids %v", known)
	}
	done := []Entry{{Status: "open", Start: "2020-01-01", End: "2020-01-31",
		Reported: 50, Captured: 50, CompletedAt: time.Now().UTC().Format(time.RFC3339)}}
	if err := saveManifest(manifest, Manifest{Slices: done}); err != nil {
		t.Fatal(err)
	}
	skip := map[string]bool{}
	for _, e := range done {
		skip[e.Start+"\x00"+e.End] = true
	}
	seeds := []Slice{
		{Status: "open", Start: "2020-01-01", End: "2020-01-31"},
		{Status: "open", Start: "2020-02-01", End: "2020-02-29"},
	}
	leaves, _, err := plan(seeds, func(Slice) (int, error) { return 3, nil }, skip)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range leaves {
		if l.Start == "2020-01-01" {
			t.Fatal("ids already in the store were planned again")
		}
	}
	if len(leaves) != 1 || leaves[0].Start != "2020-02-01" {
		t.Fatalf("planned %v, want only February", leaves)
	}
	again, err := loadIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 || !again["old-1"] || !again["old-2"] {
		t.Fatalf("store ids changed: %v", again)
	}
}

func TestManifestRecordsCoverageWindow(t *testing.T) {
	dir := t.TempDir()
	_, manifest := harvestPaths(dir, "open")
	want := Manifest{Since: harvestEpoch, Until: "2020-06-30", Slices: []Entry{{
		Status: "open", Start: "2020-01-01", End: "2020-01-01",
		Reported: 5000, Captured: 1000, CompletedAt: "2020-01-02T00:00:00Z", Incomplete: true,
	}}}
	if err := saveManifest(manifest, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Since != want.Since || got.Until != want.Until || len(got.Slices) != 1 || got.Slices[0] != want.Slices[0] {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestHarvestFlagErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--status", "won"}, "unknown --status"},
		{[]string{"--since", "nope", "--dry-run"}, "invalid --since"},
		{[]string{"--since", "2021-01-01", "--until", "2020-01-01", "--dry-run"}, "after --until"},
	}
	for _, tc := range cases {
		cmd := harvestCmd()
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("args %v: error %v, want %s", tc.args, err, tc.want)
		}
	}
}

func TestHarvestSincePlansFrom2015(t *testing.T) {
	cmd := harvestCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--status", "closed", "--since", "2015-01-01", "--until", "2015-02-28", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "closed 2015-01-01") || strings.Contains(got, "2020-01") {
		t.Fatalf("got %q", got)
	}
}
