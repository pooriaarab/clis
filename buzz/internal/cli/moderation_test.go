package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func newModerationTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Uint64("expires-in", 0, "")
	cmd.Flags().Uint64("expires-at", 0, "")
	return cmd
}

func TestResolveExpiryNeitherSet(t *testing.T) {
	cmd := newModerationTestCmd()
	got, err := resolveExpiry(cmd)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestResolveExpiryExpiresAt(t *testing.T) {
	cmd := newModerationTestCmd()
	if err := cmd.Flags().Set("expires-at", "1783500000"); err != nil {
		t.Fatal(err)
	}
	got, err := resolveExpiry(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != 1783500000 {
		t.Fatalf("got %v", got)
	}
}

func TestResolveExpiryExpiresInIsRelativeToNow(t *testing.T) {
	cmd := newModerationTestCmd()
	if err := cmd.Flags().Set("expires-in", "3600"); err != nil {
		t.Fatal(err)
	}
	before := uint64(time.Now().Unix())
	got, err := resolveExpiry(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected a value")
	}
	if *got < before+3600 || *got > before+3600+5 {
		t.Fatalf("got %d, want ~%d", *got, before+3600)
	}
}

func TestResolveExpiryMutuallyExclusive(t *testing.T) {
	cmd := newModerationTestCmd()
	if err := cmd.Flags().Set("expires-in", "60"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("expires-at", "100"); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveExpiry(cmd); err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
}

func TestModerationTargetTagsValidatesPubkey(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("pubkey", "", "")
	if err := cmd.Flags().Set("pubkey", "not-hex"); err != nil {
		t.Fatal(err)
	}
	if _, err := moderationTargetTags(cmd); err == nil {
		t.Fatal("expected validation error")
	}
}
