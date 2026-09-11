package model

import "testing"

func TestSplitSetTreatsNewlineCellAsSet(t *testing.T) {
	if got := SplitSet("*GD\n*SRV"); len(got) != 2 || got[0] != "*GD" || got[1] != "*SRV" {
		t.Fatalf("got %q", got)
	}
	if got := SplitSet("*SRV\n*GD"); len(got) != 2 || got[0] != "*GD" || got[1] != "*SRV" {
		t.Fatalf("order changed the set: %q", got)
	}
	if got := SplitSet(""); got != nil {
		t.Fatalf("empty cell gave %q", got)
	}
}
