package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"buzz-cli/internal/nostr"
)

func TestPersonaContentFieldOrder(t *testing.T) {
	content := PersonaEventContent{
		DisplayName:        "Agent",
		SystemPrompt:       ptr("Prompt"),
		AvatarURL:          ptr("avatar"),
		Runtime:            ptr("runtime"),
		Model:              ptr("model"),
		Provider:           ptr("provider"),
		NamePool:           []string{"one"},
		RespondTo:          ptr("owner-only"),
		RespondToAllowlist: []string{"a"},
		Parallelism:        ptr(uint32(2)),
	}
	got, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"display_name":"Agent","system_prompt":"Prompt","avatar_url":"avatar","runtime":"runtime","model":"model","provider":"provider","name_pool":["one"],"respond_to":"owner-only","respond_to_allowlist":["a"],"parallelism":2}`
	if string(got) != want {
		t.Fatalf("persona JSON mismatch\nwant %s\n got %s", want, got)
	}
}

func TestManagedAgentProjectionFieldOrder(t *testing.T) {
	content := ManagedAgentEventContent{
		Name:                 "Agent",
		PersonaID:            ptr("persona"),
		SystemPrompt:         ptr("Prompt"),
		Model:                ptr("model"),
		Provider:             ptr("provider"),
		PersonaSourceVersion: ptr("hash"),
		Parallelism:          3,
		RespondTo:            RespondToOwnerOnly,
		RespondToAllowlist:   []string{"a"},
	}
	got, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"name":"Agent","persona_id":"persona","system_prompt":"Prompt","model":"model","provider":"provider","persona_source_version":"hash","parallelism":3,"respond_to":"owner-only","respond_to_allowlist":["a"]}`
	if string(got) != want {
		t.Fatalf("managed-agent JSON mismatch\nwant %s\n got %s", want, got)
	}
}

func TestBuildAgentCreateEventsShapes(t *testing.T) {
	events, err := BuildManagedAgentCreateEvents(ManagedAgentCreateInput{
		AgentPubKey:  keyHex("agent"),
		OwnerPubKey:  keyHex("owner"),
		Name:         "Agent",
		SystemPrompt: "Prompt",
		AvatarURL:    "avatar",
		Runtime:      "runtime",
		Model:        "model",
		Provider:     "provider",
		Parallelism:  2,
		RespondTo:    RespondToOwnerOnly,
		Channels:     []string{"channel-id"},
		AuthTag:      []string{"auth", keyHex("owner"), "", sigHex("auth")},
	})
	if err != nil {
		t.Fatalf("BuildManagedAgentCreateEvents() error = %v", err)
	}
	if events.Profile.Kind != 0 || events.Persona.Kind != 30175 || events.ManagedAgent.Kind != 30177 {
		t.Fatalf("unexpected event kinds: %#v", events)
	}
	if !reflect.DeepEqual(events.Persona.Tags, nostr.Tags{{"d", "agent"}}) {
		t.Fatalf("persona tags = %#v", events.Persona.Tags)
	}
	if !reflect.DeepEqual(events.ManagedAgent.Tags, nostr.Tags{{"d", keyHex("agent")}}) {
		t.Fatalf("managed-agent tags = %#v", events.ManagedAgent.Tags)
	}
	if len(events.ChannelMemberships) != 1 || events.ChannelMemberships[0].Kind != 9000 {
		t.Fatalf("channel memberships = %#v", events.ChannelMemberships)
	}
}

func ptr[T any](v T) *T { return &v }

func stringsOf(s string, n int) string {
	var out strings.Builder
	for out.Len() < n {
		out.WriteString(s)
	}
	return out.String()[:n]
}

func keyHex(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func sigHex(label string) string {
	first := sha256.Sum256([]byte(label + ":1"))
	second := sha256.Sum256([]byte(label + ":2"))
	return hex.EncodeToString(append(first[:], second[:]...))
}
