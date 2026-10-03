package e2e

import (
	"strings"
	"testing"
)

func TestDirectoriesList(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("directories", "list")
	wantExit(t, r, 0)
	if n := strings.Count(r.Stdout, "\n"); n != 33 { // header + 32
		t.Errorf("list lines = %d, want 33", n)
	}
	wantContains(t, "stdout", r.Stdout, "launching-next")
	wantContains(t, "stdout", r.Stdout, "crunchbase")

	r = sb.Run("directories", "list", "--wave", "1")
	wantExit(t, r, 0)
	if n := strings.Count(r.Stdout, "\n"); n != 11 { // header + 10
		t.Errorf("wave 1 lines = %d, want 11", n)
	}

	r = sb.Run("directories", "list", "--wave", "9")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "unknown wave")

	r = sb.Run("directories", "list", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	dirs, _ := v["directories"].([]any)
	if len(dirs) != 32 {
		t.Errorf("directories = %d, want 32", len(dirs))
	}
}
