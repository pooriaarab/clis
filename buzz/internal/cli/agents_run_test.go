package cli

import (
	"testing"

	"buzz-cli/internal/config"
)

// TestResolveAgentRuntimeLoadsFromConfig is the runnable check that
// `agents run <name>` (and fleet supervision) can find an agent's key,
// auth tag, relay, and runtime commands from its persisted config.AgentRecord
// when the caller supplies nothing but the name — the whole point of
// persisting on create. No network, no process spawn: resolveAgentRuntime is
// pure.
func TestResolveAgentRuntimeLoadsFromConfig(t *testing.T) {
	resolved := config.Resolved{
		File: config.File{
			Agents: map[string]config.AgentRecord{
				"agent-1": {
					Nsec:           "nsec-from-config",
					AuthTag:        "auth-tag-from-config",
					RelayURL:       "wss://relay.example",
					ACPCommand:     "configured-acp",
					HarnessCommand: "configured-harness",
				},
			},
		},
	}

	privateKey, relayURL, authTag, acp, harness := resolveAgentRuntime(resolved, "agent-1", "", "", false, false)
	if privateKey != "nsec-from-config" {
		t.Fatalf("privateKey = %q, want config value", privateKey)
	}
	if authTag != "auth-tag-from-config" {
		t.Fatalf("authTag = %q, want config value", authTag)
	}
	if relayURL != "wss://relay.example" {
		t.Fatalf("relayURL = %q, want config value", relayURL)
	}
	if acp != "configured-acp" {
		t.Fatalf("acp = %q, want config value", acp)
	}
	if harness != "configured-harness" {
		t.Fatalf("harness = %q, want config value", harness)
	}
}

func TestResolveAgentRuntimeExplicitFlagsWin(t *testing.T) {
	resolved := config.Resolved{
		PrivateKey: "flag-nsec",
		RelayURL:   "wss://flag-relay",
		AuthTag:    "flag-auth-tag",
		File: config.File{
			Agents: map[string]config.AgentRecord{
				"agent-1": {
					Nsec:           "nsec-from-config",
					AuthTag:        "auth-tag-from-config",
					RelayURL:       "wss://relay.example",
					ACPCommand:     "configured-acp",
					HarnessCommand: "configured-harness",
				},
			},
		},
	}

	privateKey, relayURL, authTag, acp, harness := resolveAgentRuntime(resolved, "agent-1", "flag-acp", "flag-harness", true, true)
	if privateKey != "flag-nsec" || relayURL != "wss://flag-relay" || authTag != "flag-auth-tag" {
		t.Fatalf("explicit resolved values were overridden by config: %q %q %q", privateKey, relayURL, authTag)
	}
	if acp != "flag-acp" {
		t.Fatalf("acp = %q, want explicit flag to win over config", acp)
	}
	if harness != "flag-harness" {
		t.Fatalf("harness = %q, want explicit flag to win over config", harness)
	}
}

func TestResolveAgentRuntimeUnknownAgentPassesThrough(t *testing.T) {
	resolved := config.Resolved{PrivateKey: "flag-nsec"}
	privateKey, relayURL, authTag, acp, harness := resolveAgentRuntime(resolved, "unknown", "flag-acp", "", true, false)
	if privateKey != "flag-nsec" || relayURL != "" || authTag != "" || acp != "flag-acp" || harness != "" {
		t.Fatalf("unexpected resolution for unknown agent: %q %q %q %q %q", privateKey, relayURL, authTag, acp, harness)
	}
}
