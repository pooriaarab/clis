package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
)

const (
	RespondToOwnerOnly = "owner-only"
	RespondToAllowlist = "allowlist"
	RespondToAnyone    = "anyone"
)

type PersonaEventContent struct {
	DisplayName        string   `json:"display_name"`
	SystemPrompt       *string  `json:"system_prompt,omitempty"`
	AvatarURL          *string  `json:"avatar_url,omitempty"`
	Runtime            *string  `json:"runtime,omitempty"`
	Model              *string  `json:"model,omitempty"`
	Provider           *string  `json:"provider,omitempty"`
	NamePool           []string `json:"name_pool,omitempty"`
	RespondTo          *string  `json:"respond_to,omitempty"`
	RespondToAllowlist []string `json:"respond_to_allowlist,omitempty"`
	Parallelism        *uint32  `json:"parallelism,omitempty"`
}

type ManagedAgentEventContent struct {
	Name                 string   `json:"name"`
	PersonaID            *string  `json:"persona_id,omitempty"`
	SystemPrompt         *string  `json:"system_prompt,omitempty"`
	Model                *string  `json:"model,omitempty"`
	Provider             *string  `json:"provider,omitempty"`
	PersonaSourceVersion *string  `json:"persona_source_version,omitempty"`
	Parallelism          uint32   `json:"parallelism"`
	RespondTo            string   `json:"respond_to"`
	RespondToAllowlist   []string `json:"respond_to_allowlist,omitempty"`
}

type ManagedAgentCreateInput struct {
	AgentPubKey        string
	OwnerPubKey        string
	Name               string
	SystemPrompt       string
	AvatarURL          string
	Runtime            string
	Model              string
	Provider           string
	PersonaID          string
	PersonaSlug        string
	PersonaShared      bool
	Parallelism        uint32
	RespondTo          string
	RespondToAllowlist []string
	Channels           []string
	AuthTag            nostr.Tag
	CreatedAt          int64
}

type ManagedAgentCreateEvents struct {
	Profile            nostr.Event
	Persona            nostr.Event
	ManagedAgent       nostr.Event
	ChannelMemberships []nostr.Event
}

func BuildManagedAgentCreateEvents(input ManagedAgentCreateInput) (ManagedAgentCreateEvents, error) {
	if err := input.validate(); err != nil {
		return ManagedAgentCreateEvents{}, err
	}
	createdAt := input.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}

	profileContent, err := marshalProfileContent(input.Name, input.AvatarURL)
	if err != nil {
		return ManagedAgentCreateEvents{}, err
	}
	profileTags := nostr.Tags{}
	if len(input.AuthTag) > 0 {
		profileTags = append(profileTags, input.AuthTag)
	}
	profile := nostr.NewUnsignedEvent(nostr.KindProfile, input.AgentPubKey, profileContent, profileTags, createdAt)

	personaContent := PersonaEventContent{
		DisplayName:        input.Name,
		SystemPrompt:       ptrValue(input.SystemPrompt),
		AvatarURL:          ptrIfNotEmpty(input.AvatarURL),
		Runtime:            ptrIfNotEmpty(input.Runtime),
		Model:              ptrIfNotEmpty(input.Model),
		Provider:           ptrIfNotEmpty(input.Provider),
		RespondTo:          ptrIfNotEmpty(input.respondTo()),
		RespondToAllowlist: compactStrings(input.RespondToAllowlist),
		Parallelism:        ptrValue(input.parallelism()),
	}
	personaBody, err := json.Marshal(personaContent)
	if err != nil {
		return ManagedAgentCreateEvents{}, err
	}
	personaTags := nostr.Tags{{"d", input.personaDTag()}}
	if input.PersonaShared {
		personaTags = append(personaTags, nostr.Tag{"shared", "true"})
	}
	persona := nostr.NewUnsignedEvent(nostr.KindPersona, input.OwnerPubKey, string(personaBody), personaTags, createdAt)

	managedContent := ManagedAgentEventContent{
		Name:                 input.Name,
		PersonaID:            ptrIfNotEmpty(input.PersonaID),
		SystemPrompt:         ptrIfNotEmpty(input.SystemPrompt),
		Model:                ptrIfNotEmpty(input.Model),
		Provider:             ptrIfNotEmpty(input.Provider),
		PersonaSourceVersion: nil,
		Parallelism:          input.parallelism(),
		RespondTo:            input.respondTo(),
		RespondToAllowlist:   compactStrings(input.RespondToAllowlist),
	}
	if managedContent.PersonaID != nil {
		managedContent.SystemPrompt = nil
		managedContent.Model = nil
		managedContent.Provider = nil
	}
	managedBody, err := json.Marshal(managedContent)
	if err != nil {
		return ManagedAgentCreateEvents{}, err
	}
	managed := nostr.NewUnsignedEvent(nostr.KindManagedAgent, input.OwnerPubKey, string(managedBody), nostr.Tags{{"d", strings.ToLower(input.AgentPubKey)}}, createdAt)

	memberships := make([]nostr.Event, 0, len(input.Channels))
	for _, channel := range compactStrings(input.Channels) {
		memberships = append(memberships, nostr.NewUnsignedEvent(
			nostr.KindNIP29PutUser,
			input.OwnerPubKey,
			"",
			nostr.Tags{{"h", channel}, {"p", strings.ToLower(input.AgentPubKey)}},
			createdAt,
		))
	}

	return ManagedAgentCreateEvents{
		Profile:            profile,
		Persona:            persona,
		ManagedAgent:       managed,
		ChannelMemberships: memberships,
	}, nil
}

// BuildArchiveIdentityRequest builds an unsigned NIP-IA kind:9035 archive
// request for targetPubKeyHex, owner-signed by ownerPubHex's key (caller
// signs). Mirrors desktop/src-tauri/src/events.rs
// build_archive_identity_request / identity_archive_tags for the "retired"
// reason path used by `buzz desktop delete`.
func BuildArchiveIdentityRequest(ownerPubHex, targetPubKeyHex string, authTag nostr.Tag, createdAt int64) (nostr.Event, error) {
	target := strings.ToLower(strings.TrimSpace(targetPubKeyHex))
	if !isHex64(target) {
		return nostr.Event{}, errors.New("target pubkey must be 64 hex characters")
	}
	tags := nostr.Tags{{"-"}, {"p", target}, {"reason", "retired"}}
	if len(authTag) == 4 {
		tags = append(tags, authTag)
	}
	return nostr.NewUnsignedEvent(nostr.KindIAArchiveRequest, ownerPubHex, "", tags, createdAt), nil
}

func (i ManagedAgentCreateInput) validate() error {
	if !isHex64(i.AgentPubKey) {
		return errors.New("agent pubkey must be 64 hex characters")
	}
	if !isHex64(i.OwnerPubKey) {
		return errors.New("owner pubkey must be 64 hex characters")
	}
	if strings.TrimSpace(i.Name) == "" {
		return errors.New("agent name is required")
	}
	if i.respondTo() != RespondToOwnerOnly && i.respondTo() != RespondToAllowlist && i.respondTo() != RespondToAnyone {
		return fmt.Errorf("unsupported respond-to mode %q", i.respondTo())
	}
	return nil
}

func (i ManagedAgentCreateInput) personaDTag() string {
	if strings.TrimSpace(i.PersonaSlug) != "" {
		return NormalizeDTag(i.PersonaSlug)
	}
	return NormalizeDTag(i.Name)
}

func (i ManagedAgentCreateInput) respondTo() string {
	if strings.TrimSpace(i.RespondTo) == "" {
		return RespondToOwnerOnly
	}
	return strings.TrimSpace(i.RespondTo)
}

func (i ManagedAgentCreateInput) parallelism() uint32 {
	if i.Parallelism == 0 {
		return 1
	}
	return i.Parallelism
}

func NormalizeDTag(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		valid := unicode.IsLetter(r) || unicode.IsDigit(r)
		if valid {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && out.Len() > 0 {
			out.WriteByte('-')
			lastDash = true
		}
		if out.Len() >= 64 {
			break
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = "agent"
	}
	if len(result) > 64 {
		result = result[:64]
		result = strings.TrimRight(result, "-")
	}
	if result == "" || !isASCIIAlnum(rune(result[0])) {
		result = "a" + result
	}
	return result
}

func BuildChannelCreateEvent(pubkey, name, channelType, visibility, description string, ttlSeconds int64) (nostr.Event, error) {
	name = CanonicalChannelName(name)
	if strings.TrimSpace(name) == "" {
		return nostr.Event{}, errors.New("channel name is required")
	}
	channelID := uuid.NewString()
	tags := nostr.Tags{{"h", channelID}, {"name", name}}
	if visibility != "" {
		tags = append(tags, nostr.Tag{"visibility", visibility})
	}
	if channelType != "" {
		tags = append(tags, nostr.Tag{"channel_type", channelType})
	}
	if description != "" {
		tags = append(tags, nostr.Tag{"about", description})
	}
	if ttlSeconds > 0 {
		tags = append(tags, nostr.Tag{"ttl", fmt.Sprintf("%d", ttlSeconds)})
	}
	return nostr.NewUnsignedEvent(nostr.KindNIP29CreateGroup, pubkey, "", tags, 0), nil
}

func CanonicalChannelName(name string) string {
	name = strings.TrimLeftFunc(name, func(r rune) bool {
		return r == '#' || unicode.IsSpace(r)
	})
	return strings.TrimRightFunc(name, unicode.IsSpace)
}

func marshalProfileContent(displayName, picture string) (string, error) {
	content := struct {
		DisplayName *string `json:"display_name,omitempty"`
		Name        *string `json:"name,omitempty"`
		Picture     *string `json:"picture,omitempty"`
		About       *string `json:"about,omitempty"`
		NIP05       *string `json:"nip05,omitempty"`
	}{
		DisplayName: ptrIfNotEmpty(displayName),
		Picture:     ptrIfNotEmpty(picture),
	}
	body, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ptrValue[T any](v T) *T {
	return &v
}

func ptrIfNotEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	value = strings.TrimSpace(value)
	return &value
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

var hex64Pattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

func isHex64(value string) bool {
	return hex64Pattern.MatchString(value)
}

func isASCIIAlnum(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
}
