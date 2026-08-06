package cli

// buzz mem — agent engram memory (NIP-AE). Kind 30174 (KIND_AGENT_ENGRAM,
// buzz-core/src/kind.rs), NIP-44 v2 encrypted, parameterized-replaceable by
// an HMAC d-tag. Mirrors
// /Users/parab/code/buzz/crates/buzz-core/src/engram.rs (conversation_key,
// d_tag, Body, build_event, validate_and_decrypt, select_head,
// monotonic_created_at) and crates/buzz-cli/src/commands/mem.rs.
//
// NIP-44 v2 crypto (ECDH conversation key + ChaCha20 + HMAC-SHA256 padding)
// is delegated to github.com/nbd-wtf/go-nostr/nip44 rather than
// hand-rolled — it is the standard Go implementation of the same NIP and
// its ConversationKey derivation is cross-checked against the Rust oracle's
// own test vectors in mem_test.go.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/spf13/cobra"
)

const (
	memCoreSlug          = "core"
	memSlugMaxLen        = 255
	memNIP44PlaintextMax = 65535
	memDTagDomain        = "agent-memory/v1/d-tag"
	memMemPrefix         = "mem/"
)

func memCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "mem", Short: "Agent engram memory commands"}

	ls := &cobra.Command{
		Use:   "ls",
		Short: "List non-tombstoned memory entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			agent, _ := cmd.Flags().GetString("agent")
			jsonOut, _ := cmd.Flags().GetBool("json")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			agentHex, ownerHex, theirPubHex, err := memResolveReader(resolved, keys, owner, agent)
			if err != nil {
				return err
			}
			filter := client.Filter{"kinds": []int{nostr.KindAgentEngram}, "authors": []string{agentHex}, "#p": []string{ownerHex}, "limit": 5000}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, keys, []client.Filter{filter})
			if err != nil {
				return err
			}
			var events []nostr.Event
			_ = json.Unmarshal(raw, &events)

			groups := map[string][]memCandidate{}
			for _, ev := range events {
				if ok, verr := ev.Verify(); verr != nil || !ok {
					continue
				}
				d, ok := memSingleTagValue(ev.Tags, "d")
				if !ok {
					continue
				}
				body, err := memValidateAndDecrypt(ev, agentHex, ownerHex, keys.SecretHex(), theirPubHex)
				if err != nil {
					continue
				}
				groups[d] = append(groups[d], memCandidate{Event: ev, Body: body})
			}

			type listing struct {
				Slug      string `json:"slug"`
				EventID   string `json:"event_id"`
				CreatedAt int64  `json:"created_at"`
			}
			var listings []listing
			for _, members := range groups {
				evs := make([]nostr.Event, len(members))
				for i, m := range members {
					evs[i] = m.Event
				}
				head := memSelectHead(evs)
				if head == nil {
					continue
				}
				var body memBody
				for _, m := range members {
					if m.Event.ID == head.ID {
						body = m.Body
						break
					}
				}
				if body.IsCore || body.Value == nil {
					continue
				}
				listings = append(listings, listing{Slug: body.Slug, EventID: head.ID, CreatedAt: head.CreatedAt})
			}
			sort.Slice(listings, func(i, j int) bool { return listings[i].Slug < listings[j].Slug })

			if jsonOut {
				if listings == nil {
					listings = []listing{}
				}
				return opts.writeJSON(listings)
			}
			if len(listings) == 0 {
				fmt.Fprintln(opts.stderr(), "(no memories besides core)")
				return nil
			}
			for _, l := range listings {
				fmt.Fprintf(opts.stdout(), "%s\t%d\t%s\n", l.Slug, l.CreatedAt, l.EventID)
			}
			return nil
		},
	}
	ls.Flags().String("owner", "", "owner pubkey (hex). Overrides BUZZ_AUTH_TAG")
	ls.Flags().String("agent", "", "agent pubkey (hex) to read as this key's owner")
	ls.Flags().Bool("json", false, "emit JSON instead of tab-delimited lines")

	get := &cobra.Command{
		Use:   "get <slug>",
		Short: "Print the value of a slug to stdout (no trailing newline)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := memNormalizeSlug(args[0])
			if err != nil {
				return err
			}
			owner, _ := cmd.Flags().GetString("owner")
			agent, _ := cmd.Flags().GetString("agent")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			agentHex, ownerHex, theirPubHex, err := memResolveReader(resolved, keys, owner, agent)
			if err != nil {
				return err
			}
			_, body, err := opts.memFetchHead(cmd.Context(), resolved, keys, agentHex, ownerHex, theirPubHex, slug)
			if err != nil {
				return err
			}
			if body == nil {
				return ExitError{Code: ExitOther, Message: "not found: " + slug}
			}
			if body.IsCore {
				_, err := opts.stdout().Write([]byte(body.Profile))
				return err
			}
			if body.Value == nil {
				return ExitError{Code: ExitOther, Message: "tombstoned: " + slug}
			}
			_, err = opts.stdout().Write([]byte(*body.Value))
			return err
		},
	}
	get.Flags().String("owner", "", "")
	get.Flags().String("agent", "", "agent pubkey (hex) to read as this key's owner")

	hash := &cobra.Command{
		Use:   "hash <slug>",
		Short: "Print sha256(value) in hex (use as `--base-hash` for `mem patch`)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := memNormalizeSlug(args[0])
			if err != nil {
				return err
			}
			owner, _ := cmd.Flags().GetString("owner")
			agent, _ := cmd.Flags().GetString("agent")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			agentHex, ownerHex, theirPubHex, err := memResolveReader(resolved, keys, owner, agent)
			if err != nil {
				return err
			}
			_, body, err := opts.memFetchHead(cmd.Context(), resolved, keys, agentHex, ownerHex, theirPubHex, slug)
			if err != nil {
				return err
			}
			value, err := memBodyValue(body, slug)
			if err != nil {
				return err
			}
			sum := sha256.Sum256([]byte(value))
			fmt.Fprintln(opts.stdout(), hex.EncodeToString(sum[:]))
			return nil
		},
	}
	hash.Flags().String("owner", "", "")
	hash.Flags().String("agent", "", "agent pubkey (hex) to read as this key's owner")

	set := &cobra.Command{
		Use:   "set <slug> <value>",
		Short: "Set a slug's value. Pass `-` to read the value from stdin",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := memNormalizeSlug(args[0])
			if err != nil {
				return err
			}
			allowEmpty, _ := cmd.Flags().GetBool("allow-empty")
			value := args[1]
			if value == "-" {
				data, err := io.ReadAll(io.LimitReader(os.Stdin, memNIP44PlaintextMax+1))
				if err != nil {
					return otherWrap("read stdin", err)
				}
				if len(data) > memNIP44PlaintextMax {
					return inputError(fmt.Sprintf("stdin value exceeds %d-byte NIP-44 plaintext limit", memNIP44PlaintextMax))
				}
				if len(data) == 0 && !allowEmpty {
					return inputError("refusing to write empty value from stdin (an upstream pipeline step likely failed). Pass --allow-empty to confirm, or use `buzz mem rm <slug>` to tombstone.")
				}
				value = string(data)
			}
			owner, _ := cmd.Flags().GetString("owner")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			ownerHex, err := memResolveOwner(resolved, owner)
			if err != nil {
				return err
			}
			agentHex := keys.PublicHex()
			head, _, err := opts.memFetchHead(cmd.Context(), resolved, keys, agentHex, ownerHex, ownerHex, slug)
			if err != nil {
				return err
			}
			createdAt := memMonotonicCreatedAt(time.Now().Unix(), head)

			var plaintext []byte
			if slug == memCoreSlug {
				plaintext, err = memEncodeCoreBody(value)
			} else {
				v := value
				plaintext, err = memEncodeMemoryBody(slug, &v)
			}
			if err != nil {
				return otherWrap("encode body", err)
			}
			event, err := memBuildEvent(keys, ownerHex, plaintext, createdAt, slug)
			if err != nil {
				return otherWrap("build event", err)
			}
			if err := opts.memSubmit(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			fmt.Fprintf(opts.stderr(), "wrote %s (event %s, created_at %d)\n", slug, event.ID, createdAt)
			return nil
		},
	}
	set.Flags().String("owner", "", "")
	set.Flags().Bool("allow-empty", false, "allow committing an empty value. Without this, a zero-byte stdin read is rejected to prevent silent data loss from upstream pipeline failures")

	patch := &cobra.Command{
		Use:   "patch <slug>",
		Short: "Apply a unified diff to a slug's current value (safer than set)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.memPatch(cmd, args[0])
		},
	}
	patch.Flags().String("patch-file", "", "read the patch from a file instead of stdin")
	patch.Flags().String("base-hash", "", "sha256 hex digest (lowercase) of the value the patch was generated against")
	patch.Flags().Bool("no-base-hash", false, "skip the base-hash check. Unsafe if concurrent edits are possible")
	patch.Flags().Bool("dry-run", false, "echo the input patch + resulting sha256 and exit without writing")
	patch.Flags().Bool("allow-empty", false, "allow committing an empty result")
	patch.Flags().String("owner", "", "")

	rm := &cobra.Command{
		Use:   "rm <slug>",
		Short: "Publish a tombstone for a slug (cannot be used on `core`)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := memNormalizeSlug(args[0])
			if err != nil {
				return err
			}
			if slug == memCoreSlug {
				return inputError("core cannot be tombstoned; overwrite it with `buzz mem set core ''` instead")
			}
			owner, _ := cmd.Flags().GetString("owner")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			ownerHex, err := memResolveOwner(resolved, owner)
			if err != nil {
				return err
			}
			agentHex := keys.PublicHex()
			head, _, err := opts.memFetchHead(cmd.Context(), resolved, keys, agentHex, ownerHex, ownerHex, slug)
			if err != nil {
				return err
			}
			createdAt := memMonotonicCreatedAt(time.Now().Unix(), head)
			plaintext, err := memEncodeMemoryBody(slug, nil)
			if err != nil {
				return otherWrap("encode body", err)
			}
			event, err := memBuildEvent(keys, ownerHex, plaintext, createdAt, slug)
			if err != nil {
				return otherWrap("build event", err)
			}
			if err := opts.memSubmit(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			fmt.Fprintf(opts.stderr(), "tombstoned %s (event %s, created_at %d)\n", slug, event.ID, createdAt)
			return nil
		},
	}
	rm.Flags().String("owner", "", "")

	cmd.AddCommand(ls, get, hash, set, patch, rm)
	return cmd
}

// ── slug grammar ────────────────────────────────────────────────────────────

func memNormalizeSlug(raw string) (string, error) {
	slug := raw
	if raw != memCoreSlug && !strings.HasPrefix(raw, memMemPrefix) {
		slug = memMemPrefix + raw
	}
	if err := memValidateSlug(slug); err != nil {
		return "", inputWrap("invalid slug", err)
	}
	return slug, nil
}

func memValidateSlug(slug string) error {
	if slug == memCoreSlug {
		return nil
	}
	if len(slug) > memSlugMaxLen {
		return fmt.Errorf("length %d exceeds %d", len(slug), memSlugMaxLen)
	}
	rest, ok := strings.CutPrefix(slug, memMemPrefix)
	if !ok {
		return fmt.Errorf("expected `core` or `mem/…`, got %q", slug)
	}
	if rest == "" {
		return errors.New("empty after `mem/`")
	}
	for i, seg := range strings.Split(rest, "/") {
		if err := memValidateSegment(seg); err != nil {
			return fmt.Errorf("segment %d (%q): %w", i+1, seg, err)
		}
	}
	return nil
}

func memValidateSegment(s string) error {
	if s == "" {
		return errors.New("empty")
	}
	if len(s) > 64 {
		return errors.New("longer than 64 bytes")
	}
	if !memIsLowerAlnum(s[0]) {
		return errors.New("first byte must be [a-z0-9]")
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(memIsLowerAlnum(c) || c == '_' || c == '-') {
			return errors.New("only [a-z0-9_-] allowed after the first byte")
		}
	}
	return nil
}

func memIsLowerAlnum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// ── NIP-44 conversation key + HMAC d-tag ────────────────────────────────────

// memConversationKey derives K_c per NIP-44 v2 ECDH. Symmetric: deriving
// from either side of the (agent, owner) pair yields the same key.
func memConversationKey(mySecHex, theirPubHex string) ([32]byte, error) {
	return nip44.GenerateConversationKey(theirPubHex, mySecHex)
}

// memDTag computes d = lower_hex(HMAC-SHA256(K_c, "agent-memory/v1/d-tag" ||
// 0x00 || slug)).
func memDTag(kC [32]byte, slug string) string {
	mac := hmac.New(sha256.New, kC[:])
	mac.Write([]byte(memDTagDomain))
	mac.Write([]byte{0})
	mac.Write([]byte(slug))
	return hex.EncodeToString(mac.Sum(nil))
}

// ── body encode/decode ──────────────────────────────────────────────────────

type memMemoryJSON struct {
	Slug  string  `json:"slug"`
	Value *string `json:"value"`
}

type memCoreJSON struct {
	Slug    string `json:"slug"`
	Profile string `json:"profile"`
}

func memEncodeMemoryBody(slug string, value *string) ([]byte, error) {
	return json.Marshal(memMemoryJSON{Slug: slug, Value: value})
}

func memEncodeCoreBody(profile string) ([]byte, error) {
	return json.Marshal(memCoreJSON{Slug: memCoreSlug, Profile: profile})
}

// memBody is the decoded form of either body variant. IsCore discriminates;
// for a memory body, Value == nil means a tombstone.
type memBody struct {
	Slug    string
	IsCore  bool
	Profile string
	Value   *string
}

func memBodySlug(b memBody) string {
	if b.IsCore {
		return memCoreSlug
	}
	return b.Slug
}

func memDecodeBody(plaintext []byte) (memBody, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(plaintext, &raw); err != nil {
		return memBody{}, fmt.Errorf("invalid JSON body: %w", err)
	}
	slugRaw, ok := raw["slug"]
	if !ok {
		return memBody{}, errors.New("missing `slug`")
	}
	var slug string
	if err := json.Unmarshal(slugRaw, &slug); err != nil {
		return memBody{}, errors.New("`slug` is not a string")
	}
	if err := memValidateSlug(slug); err != nil {
		return memBody{}, fmt.Errorf("invalid slug: %w", err)
	}
	if slug == memCoreSlug {
		profRaw, ok := raw["profile"]
		if !ok {
			return memBody{}, errors.New("core missing `profile`")
		}
		var profile string
		if err := json.Unmarshal(profRaw, &profile); err != nil {
			return memBody{}, errors.New("`profile` is not a string")
		}
		return memBody{Slug: slug, IsCore: true, Profile: profile}, nil
	}
	valRaw, ok := raw["value"]
	if !ok {
		return memBody{}, errors.New("memory missing `value`")
	}
	if string(valRaw) == "null" {
		return memBody{Slug: slug}, nil
	}
	var value string
	if err := json.Unmarshal(valRaw, &value); err != nil {
		return memBody{}, errors.New("`value` is not a string")
	}
	return memBody{Slug: slug, Value: &value}, nil
}

func memBodyValue(body *memBody, slug string) (string, error) {
	if body == nil {
		return "", ExitError{Code: ExitOther, Message: "not found: " + slug}
	}
	if body.IsCore {
		return body.Profile, nil
	}
	if body.Value == nil {
		return "", ExitError{Code: ExitOther, Message: "tombstoned: " + slug}
	}
	return *body.Value, nil
}

// ── event build / validate ──────────────────────────────────────────────────

// memBuildEvent signs a kind:30174 engram event: content is NIP-44
// encrypted plaintext, tags are d=HMAC(K_c, slug) and p=owner.
func memBuildEvent(agentKeys *nostr.KeyPair, ownerPubHex string, plaintext []byte, createdAt int64, slug string) (nostr.Event, error) {
	if len(plaintext) > memNIP44PlaintextMax {
		return nostr.Event{}, fmt.Errorf("body exceeds %d-byte plaintext limit (%d bytes)", memNIP44PlaintextMax, len(plaintext))
	}
	kC, err := memConversationKey(agentKeys.SecretHex(), ownerPubHex)
	if err != nil {
		return nostr.Event{}, err
	}
	ciphertext, err := nip44.Encrypt(string(plaintext), kC)
	if err != nil {
		return nostr.Event{}, err
	}
	d := memDTag(kC, slug)
	event := nostr.NewUnsignedEvent(nostr.KindAgentEngram, agentKeys.PublicHex(), ciphertext, nostr.Tags{{"d", d}, {"p", strings.ToLower(ownerPubHex)}}, createdAt)
	if err := event.Sign(agentKeys); err != nil {
		return nostr.Event{}, err
	}
	return event, nil
}

// memCandidate pairs a raw event with its decoded body for head selection.
type memCandidate struct {
	Event nostr.Event
	Body  memBody
}

// memValidateAndDecrypt validates an event against the NIP-AE head-selection
// rules (kind, single p/d tags, owner match, d re-derivation) and returns
// its decrypted body. Caller must verify the event signature beforehand.
func memValidateAndDecrypt(ev nostr.Event, expectedAgentHex, expectedOwnerHex, mySecHex, theirPubHex string) (memBody, error) {
	if ev.Kind != nostr.KindAgentEngram {
		return memBody{}, fmt.Errorf("wrong kind: %d", ev.Kind)
	}
	if !strings.EqualFold(ev.PubKey, expectedAgentHex) {
		return memBody{}, errors.New("pubkey != expected_agent")
	}
	dValue, ok := memSingleTagValue(ev.Tags, "d")
	if !ok {
		return memBody{}, errors.New("missing or duplicate d tag")
	}
	if len(dValue) != 64 || !isLowerHex(dValue) {
		return memBody{}, errors.New("d tag must be 64 lowercase hex chars")
	}
	pValue, ok := memSingleTagValue(ev.Tags, "p")
	if !ok {
		return memBody{}, errors.New("missing or duplicate p tag")
	}
	if !strings.EqualFold(pValue, expectedOwnerHex) {
		return memBody{}, errors.New("p tag != expected_owner")
	}

	kC, err := memConversationKey(mySecHex, theirPubHex)
	if err != nil {
		return memBody{}, err
	}
	plaintext, err := nip44.Decrypt(ev.Content, kC)
	if err != nil {
		return memBody{}, errors.New("decrypt failed")
	}
	body, err := memDecodeBody([]byte(plaintext))
	if err != nil {
		return memBody{}, err
	}
	if derived := memDTag(kC, memBodySlug(body)); derived != dValue {
		return memBody{}, errors.New("body slug does not re-derive to d tag")
	}
	return body, nil
}

func memSingleTagValue(tags nostr.Tags, key string) (string, bool) {
	value := ""
	count := 0
	for _, t := range tags {
		if len(t) >= 2 && t[0] == key {
			count++
			value = t[1]
		}
	}
	if count != 1 {
		return "", false
	}
	return value, true
}

// memSelectHead picks the head from same-slug candidates: greatest
// created_at, ties broken by lowest event id (NIP-01 convention).
func memSelectHead(events []nostr.Event) *nostr.Event {
	if len(events) == 0 {
		return nil
	}
	head := events[0]
	for _, e := range events[1:] {
		switch {
		case e.CreatedAt > head.CreatedAt:
			head = e
		case e.CreatedAt == head.CreatedAt && e.ID < head.ID:
			head = e
		}
	}
	return &head
}

// memMonotonicCreatedAt computes max(now, prior_head.created_at + 1).
func memMonotonicCreatedAt(now int64, priorHead *nostr.Event) int64 {
	if priorHead == nil {
		return now
	}
	next := priorHead.CreatedAt + 1
	if next > now {
		return next
	}
	return now
}

// ── owner/reader resolution ─────────────────────────────────────────────────

// memResolveOwner resolves the agent's owner pubkey: --owner wins, else the
// NIP-OA BUZZ_AUTH_TAG (whose slot 1 carries the owner pubkey).
func memResolveOwner(resolved config.Resolved, ownerFlag string) (string, error) {
	if ownerFlag != "" {
		return validateHex64(ownerFlag)
	}
	if resolved.AuthTag == "" {
		return "", inputError("owner pubkey required (set BUZZ_AUTH_TAG with a NIP-OA attestation or pass --owner)")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return "", inputWrap("parse auth tag", err)
	}
	if len(tag) < 2 {
		return "", inputError("BUZZ_AUTH_TAG is malformed")
	}
	return validateHex64(tag[1])
}

// memResolveReader resolves the read perspective for ls/get/hash: normal
// agent-side reads use the CLI identity as the agent and --owner/
// BUZZ_AUTH_TAG for the owner; --agent flips to owner-side recovery (CLI
// identity is treated as the owner, decrypting the named agent's engrams).
func memResolveReader(resolved config.Resolved, keys *nostr.KeyPair, ownerFlag, agentFlag string) (agentHex, ownerHex, theirPubHex string, err error) {
	if agentFlag != "" {
		if ownerFlag != "" {
			return "", "", "", inputError("--owner and --agent are mutually exclusive for read commands")
		}
		agentHex, err = validateHex64(agentFlag)
		if err != nil {
			return "", "", "", err
		}
		if agentHex == keys.PublicHex() {
			return "", "", "", inputError("--agent must differ from the CLI identity; omit --agent for agent-side reads")
		}
		return agentHex, keys.PublicHex(), agentHex, nil
	}
	agentHex = keys.PublicHex()
	ownerHex, err = memResolveOwner(resolved, ownerFlag)
	if err != nil {
		return "", "", "", err
	}
	return agentHex, ownerHex, ownerHex, nil
}

// memFetchHead queries the relay for the head engram event on slug and
// returns its decoded body, or (nil, nil, nil) if none exists.
func (opts *rootOptions) memFetchHead(ctx context.Context, resolved config.Resolved, readerKeys *nostr.KeyPair, agentHex, ownerHex, theirPubHex, slug string) (*nostr.Event, *memBody, error) {
	kC, err := memConversationKey(readerKeys.SecretHex(), theirPubHex)
	if err != nil {
		return nil, nil, otherWrap("derive conversation key", err)
	}
	d := memDTag(kC, slug)
	filter := client.Filter{"kinds": []int{nostr.KindAgentEngram}, "authors": []string{agentHex}, "#d": []string{d}, "#p": []string{ownerHex}, "limit": 16}
	raw, err := opts.fetchQuery(ctx, resolved, readerKeys, []client.Filter{filter})
	if err != nil {
		return nil, nil, err
	}
	var events []nostr.Event
	if err := json.Unmarshal(raw, &events); err != nil || len(events) == 0 {
		return nil, nil, nil
	}

	var valid []memCandidate
	for _, ev := range events {
		if ok, verr := ev.Verify(); verr != nil || !ok {
			continue
		}
		body, err := memValidateAndDecrypt(ev, agentHex, ownerHex, readerKeys.SecretHex(), theirPubHex)
		if err != nil {
			continue
		}
		valid = append(valid, memCandidate{Event: ev, Body: body})
	}
	if len(valid) == 0 {
		return nil, nil, nil
	}
	evs := make([]nostr.Event, len(valid))
	for i, c := range valid {
		evs[i] = c.Event
	}
	head := memSelectHead(evs)
	for _, c := range valid {
		if c.Event.ID == head.ID {
			return head, &c.Body, nil
		}
	}
	return head, nil, nil
}

func (opts *rootOptions) memSubmit(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	return relayPublishError(raw)
}

// ── mem patch: unified diff apply ───────────────────────────────────────────

func (opts *rootOptions) memPatch(cmd *cobra.Command, rawSlug string) error {
	slug, err := memNormalizeSlug(rawSlug)
	if err != nil {
		return err
	}
	patchFile, _ := cmd.Flags().GetString("patch-file")
	baseHash, _ := cmd.Flags().GetString("base-hash")
	noBaseHash, _ := cmd.Flags().GetBool("no-base-hash")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	allowEmpty, _ := cmd.Flags().GetBool("allow-empty")
	owner, _ := cmd.Flags().GetString("owner")
	hasBaseHash := cmd.Flags().Changed("base-hash")

	if hasBaseHash && noBaseHash {
		return inputError("--base-hash and --no-base-hash are mutually exclusive")
	}
	if !hasBaseHash && !noBaseHash {
		return inputError("missing --base-hash <hex> (run `buzz mem hash <slug>` to get it). Pass --no-base-hash to skip this check at your own risk.")
	}
	if hasBaseHash && (len(baseHash) != 64 || !isLowerHexOrUpper(baseHash)) {
		return inputError("--base-hash must be a 64-character hex sha256 digest")
	}

	var diffText string
	if patchFile != "" {
		data, err := os.ReadFile(patchFile)
		if err != nil {
			return inputWrap("failed to read --patch-file "+patchFile, err)
		}
		diffText = string(data)
	} else {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, memNIP44PlaintextMax+1))
		if err != nil {
			return otherWrap("read stdin", err)
		}
		if len(data) == 0 {
			return inputError("refusing to apply empty patch from stdin (an upstream pipeline step likely failed)")
		}
		diffText = string(data)
	}

	resolved, keys, err := opts.resolveKeys(true)
	if err != nil {
		return err
	}
	ownerHex, err := memResolveOwner(resolved, owner)
	if err != nil {
		return err
	}
	agentHex := keys.PublicHex()

	head, body, err := opts.memFetchHead(cmd.Context(), resolved, keys, agentHex, ownerHex, ownerHex, slug)
	if err != nil {
		return err
	}
	if head == nil {
		return ExitError{Code: ExitOther, Message: "not found: " + slug}
	}
	current, err := memBodyValue(body, slug)
	if err != nil {
		return err
	}

	if hasBaseHash {
		sum := sha256.Sum256([]byte(current))
		actual := hex.EncodeToString(sum[:])
		if !strings.EqualFold(actual, baseHash) {
			return ExitError{Code: ExitConflict, Message: fmt.Sprintf("slug `%s` has changed since patch was generated (expected sha256 %s, got %s). Re-fetch and regenerate the patch.", slug, baseHash, actual)}
		}
	}

	fileHeaders := 0
	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "--- ") {
			fileHeaders++
		}
	}
	if fileHeaders > 1 {
		return inputError(fmt.Sprintf("multi-file patch not supported (found %d `--- ` headers); a memory slug is a single virtual file", fileHeaders))
	}

	hunks, err := memParsePatch(diffText)
	if err != nil {
		return inputWrap("malformed unified diff", err)
	}
	newValue, err := memApplyPatch(current, hunks)
	if err != nil {
		return inputError(fmt.Sprintf("patch did not apply cleanly to slug `%s`: %s. Context must match the current value verbatim at the declared line numbers — no fuzz, no offset.", slug, err.Error()))
	}

	if len(newValue) > memNIP44PlaintextMax {
		return inputError(fmt.Sprintf("patched value would exceed %d-byte NIP-44 plaintext limit (got %d bytes)", memNIP44PlaintextMax, len(newValue)))
	}
	if newValue == "" && !allowEmpty {
		return inputError("refusing to write empty value (patch result is empty). Pass --allow-empty to confirm, or use `buzz mem rm <slug>` to tombstone.")
	}

	newHash := sha256.Sum256([]byte(newValue))
	newHashHex := hex.EncodeToString(newHash[:])
	fmt.Fprintln(opts.stderr(), strings.TrimRight(diffText, "\n"))
	fmt.Fprintln(opts.stderr())
	if dryRun {
		fmt.Fprintf(opts.stderr(), "(dry run — slug `%s` not modified; would write sha256 %s)\n", slug, newHashHex)
		return nil
	}

	var plaintext []byte
	if slug == memCoreSlug {
		plaintext, err = memEncodeCoreBody(newValue)
	} else {
		v := newValue
		plaintext, err = memEncodeMemoryBody(slug, &v)
	}
	if err != nil {
		return otherWrap("encode body", err)
	}
	createdAt := memMonotonicCreatedAt(time.Now().Unix(), head)
	event, err := memBuildEvent(keys, ownerHex, plaintext, createdAt, slug)
	if err != nil {
		return otherWrap("build event", err)
	}
	if err := opts.memSubmit(cmd.Context(), resolved, keys, event); err != nil {
		return err
	}
	fmt.Fprintf(opts.stderr(), "wrote %s (event %s, created_at %d, sha256 %s)\n", slug, event.ID, createdAt, newHashHex)
	return nil
}

type memDiffLine struct {
	op         byte // ' ', '-', or '+'
	text       string
	hasNewline bool
}

type memDiffHunk struct {
	oldStart int
	lines    []memDiffLine
}

var memHunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// memParsePatch parses a unified diff into hunks. Tolerant of "--- "/"+++ "
// file headers (skipped) and "\ No newline at end of file" markers.
func memParsePatch(diffText string) ([]memDiffHunk, error) {
	var hunks []memDiffHunk
	var cur *memDiffHunk
	for _, line := range strings.Split(diffText, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			m := memHunkHeaderRe.FindStringSubmatch(line)
			if m == nil {
				return nil, fmt.Errorf("malformed hunk header: %s", line)
			}
			oldStart, _ := strconv.Atoi(m[1])
			hunks = append(hunks, memDiffHunk{oldStart: oldStart})
			cur = &hunks[len(hunks)-1]
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "):
			// file headers — not needed to apply a single-file patch.
		case line == `\ No newline at end of file`:
			if cur != nil && len(cur.lines) > 0 {
				cur.lines[len(cur.lines)-1].hasNewline = false
			}
		default:
			if cur == nil || line == "" {
				continue
			}
			op := line[0]
			if op != ' ' && op != '-' && op != '+' {
				return nil, fmt.Errorf("unexpected diff line: %q", line)
			}
			cur.lines = append(cur.lines, memDiffLine{op: op, text: line[1:], hasNewline: true})
		}
	}
	if len(hunks) == 0 {
		return nil, errors.New("no hunks found in patch")
	}
	return hunks, nil
}

// memApplyPatch verifies each hunk's preimage (context + delete lines)
// matches current at its declared line number exactly — no fuzz, no
// offset-sliding — then applies it. Multi-hunk patches reference the
// *original* file's line numbers independently (not cumulative).
func memApplyPatch(current string, hunks []memDiffHunk) (string, error) {
	currentLines := memSplitInclusiveNewline(current)
	var out strings.Builder
	origIdx := 0

	for hi, hunk := range hunks {
		var preimage []string
		for _, l := range hunk.lines {
			if l.op == ' ' || l.op == '-' {
				preimage = append(preimage, memDiffLineText(l))
			}
		}

		declaredStart := 0
		if len(preimage) == 0 {
			if hunk.oldStart != 0 {
				return "", fmt.Errorf("hunk #%d has empty preimage at line %d; pure no-context insertions into non-empty values are not supported (regenerate the patch with `diff -u` to include surrounding context)", hi+1, hunk.oldStart)
			}
		} else {
			if hunk.oldStart == 0 {
				return "", fmt.Errorf("hunk #%d has invalid line number 0", hi+1)
			}
			declaredStart = hunk.oldStart - 1
			end := declaredStart + len(preimage)
			if end > len(currentLines) {
				return "", fmt.Errorf("hunk #%d expects %d preimage line(s) starting at line %d, but the value only has %d line(s)", hi+1, len(preimage), declaredStart+1, len(currentLines))
			}
			for offset, expected := range preimage {
				actual := currentLines[declaredStart+offset]
				if expected != actual {
					return "", fmt.Errorf("hunk #%d preimage mismatch at line %d: patch expects %q but value has %q", hi+1, declaredStart+offset+1, expected, actual)
				}
			}
		}

		for origIdx < declaredStart {
			out.WriteString(currentLines[origIdx])
			origIdx++
		}
		for _, l := range hunk.lines {
			switch l.op {
			case ' ':
				out.WriteString(memDiffLineText(l))
				origIdx++
			case '-':
				origIdx++
			case '+':
				out.WriteString(memDiffLineText(l))
			}
		}
	}
	for origIdx < len(currentLines) {
		out.WriteString(currentLines[origIdx])
		origIdx++
	}
	return out.String(), nil
}

func memDiffLineText(l memDiffLine) string {
	if l.hasNewline {
		return l.text + "\n"
	}
	return l.text
}

// memSplitInclusiveNewline splits s into lines, each retaining its trailing
// "\n" except possibly the last (mirrors Rust's split_inclusive('\n')).
func memSplitInclusiveNewline(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
