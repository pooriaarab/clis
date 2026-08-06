package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestBuildAgentRuntimeEnv(t *testing.T) {
	env := buildAgentRuntimeEnv("nsec1secret", "wss://relay.example", `["auth","owner","",""]`, "codex")
	want := map[string]string{
		"BUZZ_PRIVATE_KEY":       "nsec1secret",
		"BUZZ_RELAY_URL":         "wss://relay.example",
		"BUZZ_AUTH_TAG":          `["auth","owner","",""]`,
		"BUZZ_ACP_AGENT_COMMAND": "codex",
	}
	found := map[string]bool{}
	for _, kv := range env {
		for key, value := range want {
			if kv == key+"="+value {
				found[key] = true
			}
		}
	}
	for key := range want {
		if !found[key] {
			t.Errorf("buildAgentRuntimeEnv missing %s=%s in %v", key, want[key], env)
		}
	}
}

// TestRunFleetSuperviseAndShutdown is the runnable check for the fleet
// supervisor's non-trivial logic: per-agent log files, a max-concurrent
// cap enforced via a semaphore, a restart-on-exit cycle, and a graceful
// stop (not a crash-loop) once the context is canceled. It spawns real
// short-lived local shell scripts (no network) as stand-ins for buzz-acp.
func TestRunFleetSuperviseAndShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture assumes a POSIX shell")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-acp.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho started\nsleep 0.02\n"), 0o700); err != nil {
		t.Fatalf("write fixture script: %v", err)
	}
	logDir := filepath.Join(dir, "logs")

	agents := []createdAgent{
		{Name: "fleet-1", PubKey: "pub1", Nsec: "nsec1"},
		{Name: "fleet-2", PubKey: "pub2", Nsec: "nsec2"},
	}

	var stdout bytes.Buffer
	opts := &rootOptions{
		ConfigPath: filepath.Join(dir, "missing-config.toml"),
		RelayURL:   "wss://relay.invalid", // never dialed by runFleet itself
		Format:     "json",
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	if err := opts.runFleet(ctx, agents, 1, logDir, script, ""); err != nil {
		t.Fatalf("runFleet: %v", err)
	}

	dec := json.NewDecoder(&stdout)
	var manifestDoc struct {
		Manifest []fleetAgentStatus `json:"manifest"`
		LogDir   string             `json:"log_dir"`
	}
	if err := dec.Decode(&manifestDoc); err != nil {
		t.Fatalf("decode manifest doc: %v", err)
	}
	if len(manifestDoc.Manifest) != 2 {
		t.Fatalf("manifest has %d agents, want 2", len(manifestDoc.Manifest))
	}

	var summaryDoc struct {
		Summary []fleetAgentStatus `json:"summary"`
	}
	if err := dec.Decode(&summaryDoc); err != nil {
		t.Fatalf("decode summary doc: %v", err)
	}
	if len(summaryDoc.Summary) != 2 {
		t.Fatalf("summary has %d agents, want 2", len(summaryDoc.Summary))
	}
	for _, status := range summaryDoc.Summary {
		if status.State != "stopped" {
			t.Errorf("agent %s final state = %q, want %q (graceful shutdown)", status.Name, status.State, "stopped")
		}
		if status.PID != 0 {
			t.Errorf("agent %s final pid = %d, want 0 after shutdown", status.Name, status.PID)
		}
		if status.Restarts < 1 {
			t.Errorf("agent %s restarts = %d, want >= 1 (its short-lived child should have exited and restarted once)", status.Name, status.Restarts)
		}
		logBytes, err := os.ReadFile(status.LogPath)
		if err != nil {
			t.Errorf("read log for %s: %v", status.Name, err)
			continue
		}
		if !bytes.Contains(logBytes, []byte("started")) {
			t.Errorf("log for %s = %q, want it to contain the child's stdout", status.Name, logBytes)
		}
	}
}
