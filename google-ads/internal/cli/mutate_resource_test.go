// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0.

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// One runnable check that the generic mutate command builds the right path
// for a resource none of the printed wrapper commands ever covered
// (Experiment), proving the escape hatch actually reaches resources beyond
// the six hardcoded promoted_*.go commands.
func TestMutateResourceCmd_BuildsPathForUnwrappedResource(t *testing.T) {
	flags := &rootFlags{}
	root := newRootCmd(flags)
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{
		"mutate", "experiments", "1234567890",
		"--dry-run", "--json",
		"--operations", `[{"create":{"name":"Q3 bid test","type":"SEARCH_CUSTOM"}}]`,
	})

	// The dry-run request echo writes to the real os.Stderr inside
	// internal/client (not through cmd.SetErr), so capture it directly.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	execErr := root.Execute()
	w.Close()
	os.Stderr = origStderr
	stderrBytes, _ := io.ReadAll(r)
	if execErr != nil {
		t.Fatalf("Execute: %v", execErr)
	}

	if !strings.Contains(string(stderrBytes), "/v24/customers/1234567890/experiments:mutate") {
		t.Fatalf("expected experiments:mutate path in dry-run echo, got:\n%s", stderrBytes)
	}

	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not the JSON envelope: %v\noutput:\n%s", err, out.String())
	}
	results, ok := envelope["results"].(map[string]any)
	if !ok || results["dry_run"] != true {
		t.Fatalf("expected results.dry_run=true in envelope, got %+v", envelope)
	}
}

func TestMutateResourceCmd_RequiresBothArgs(t *testing.T) {
	flags := &rootFlags{}
	root := newRootCmd(flags)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"mutate", "experiments"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error when customerId is missing, got nil")
	}
}
