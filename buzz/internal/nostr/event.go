package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcutil/bech32"
)

const (
	KindProfile             = 0
	KindTextNote            = 1
	KindContactList         = 3
	KindDeletion            = 5
	KindReaction            = 7
	KindChannelMessage      = 9
	KindAuth                = 22242
	KindHTTPAuth            = 27235
	KindLongForm            = 30023
	KindEmojiSet            = 30030
	KindPersona             = 30175
	KindManagedAgent        = 30177
	KindChannelMetadata     = 39000
	KindChannelMembers      = 39002
	KindNIP29CreateGroup    = 9007
	KindNIP29PutUser        = 9000
	KindNIP29RemoveUser     = 9001
	KindNIP29JoinRequest    = 9021
	KindSetWorkspaceProfile = 9033
	KindStatus              = 30315
	KindPresence            = 20001
	KindPresenceSnapshot    = 40902
	KindDMCreated           = 41001
	KindDMOpen              = 41010
	KindDMAddMember         = 41011
	KindDMHide              = 41012
	KindCanvas              = 40100

	// NIP-29 channel-metadata commands (buzz-core/src/kind.rs).
	KindNIP29EditMetadata = 9002 // KIND_NIP29_EDIT_METADATA — update/topic/purpose/archive/unarchive
	KindNIP29DeleteEvent  = 9005 // KIND_NIP29_DELETE_EVENT — messages delete (buzz-sdk build_delete_message_with_options)
	KindNIP29DeleteGroup  = 9008 // KIND_NIP29_DELETE_GROUP — channels delete
	KindNIP29LeaveRequest = 9022 // KIND_NIP29_LEAVE_REQUEST — channels leave

	// NIP-IA: identity archive (buzz-core/src/kind.rs).
	KindIAArchiveRequest   = 9035  // KIND_IA_ARCHIVE_REQUEST
	KindIAUnarchiveRequest = 9036  // KIND_IA_UNARCHIVE_REQUEST
	KindIAArchivedList     = 13535 // KIND_IA_ARCHIVED_LIST

	// Stream messages / forum (buzz-core/src/kind.rs).
	KindStreamMessageV2 = 40002 // KIND_STREAM_MESSAGE_V2
	KindMessageEdit     = 40003 // KIND_STREAM_MESSAGE_EDIT — messages edit
	KindMessageDiff     = 40008 // KIND_STREAM_MESSAGE_DIFF — messages send-diff
	KindForumPost       = 45001 // KIND_FORUM_POST
	KindForumVote       = 45002 // KIND_FORUM_VOTE — messages vote
	KindForumComment    = 45003 // KIND_FORUM_COMMENT

	// Agent observer frames (buzz-core/src/kind.rs; NIP-44 encrypted content).
	KindAgentObserverFrame = 24200 // KIND_AGENT_OBSERVER_FRAME — agents draft-create/draft-update
	KindAgentProfile       = 10100 // KIND_AGENT_PROFILE — channels set-add-policy

	// NIP-34: git collaboration (repos.rs kind::KIND_GIT_* in crates/buzz-core/src/kind.rs).
	KindGitRepoAnnouncement = 30617
	KindGitPatch            = 1617
	KindGitPullRequest      = 1618
	KindGitPrUpdate         = 1619
	KindGitIssue            = 1621
	KindGitStatusOpen       = 1630
	KindGitStatusMerged     = 1631
	KindGitStatusClosed     = 1632
	KindGitStatusDraft      = 1633

	// NIP-MP: multi-repo projects.
	KindProject = 30621

	// Workflow engine (buzz-core/src/kind.rs).
	KindWorkflowDef           = 30620
	KindWorkflowTrigger       = 46020
	KindApprovalGrant         = 46030
	KindApprovalDeny          = 46031
	KindWorkflowTriggered     = 46001
	KindWorkflowStepStarted   = 46002
	KindWorkflowStepCompleted = 46003
	KindWorkflowStepFailed    = 46004
	KindWorkflowCompleted     = 46005
	KindWorkflowFailed        = 46006
	KindWorkflowCancelled     = 46007

	// Community moderation commands (buzz-core/src/kind.rs, is_moderation_command_kind).
	KindReport              = 1984
	KindModerationBan       = 9040
	KindModerationUnban     = 9041
	KindModerationTimeout   = 9042
	KindModerationUntimeout = 9043
	KindModerationResolve   = 9044

	// Media / Blossom (BUD-01/BUD-02).
	KindBlossomAuth = 24242

	// NIP-AE: Agent Engram.
	KindAgentEngram = 30174
)

var nsecPrefix = "nsec" + "1"

type Tag []string

type Tags []Tag

type Event struct {
	ID        string `json:"id"`
	PubKey    string `json:"pubkey"`
	CreatedAt int64  `json:"created_at"`
	Kind      int    `json:"kind"`
	Tags      Tags   `json:"tags"`
	Content   string `json:"content"`
	Sig       string `json:"sig"`
}

type KeyPair struct {
	Private *btcec.PrivateKey
	Public  *btcec.PublicKey
}

func NewKeyPair() (*KeyPair, error) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}
	return &KeyPair{Private: priv, Public: priv.PubKey()}, nil
}

func ParsePrivateKey(value string) (*KeyPair, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("private key is required")
	}

	var raw []byte
	var err error
	if strings.HasPrefix(strings.ToLower(value), nsecPrefix) {
		raw, err = DecodeNsec(value)
		if err != nil {
			return nil, err
		}
	} else {
		if len(value) != 64 {
			return nil, fmt.Errorf("private key must be nsec or 64 hex chars")
		}
		raw, err = hex.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("decode private key hex: %w", err)
		}
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("private key must decode to 32 bytes")
	}
	if allZero(raw) {
		return nil, errors.New("private key must be non-zero")
	}

	priv, pub := btcec.PrivKeyFromBytes(raw)
	return &KeyPair{Private: priv, Public: pub}, nil
}

func DecodeNsec(value string) ([]byte, error) {
	hrp, data, err := bech32.Decode(value)
	if err != nil {
		return nil, fmt.Errorf("decode nsec: %w", err)
	}
	if hrp != "nsec" {
		return nil, fmt.Errorf("bech32 human-readable part %q is not nsec", hrp)
	}
	raw, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return nil, fmt.Errorf("decode nsec data: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("nsec decoded to %d bytes, want 32", len(raw))
	}
	return raw, nil
}

// ParseNpub decodes a bech32 "npub1..." string into a lowercase 64-char hex
// pubkey. Used by `messages search --author`, which accepts hex, npub, or a
// display name (buzz-cli/src/commands/messages.rs resolve_author).
func ParseNpub(value string) (string, error) {
	hrp, data, err := bech32.Decode(value)
	if err != nil {
		return "", fmt.Errorf("decode npub: %w", err)
	}
	if hrp != "npub" {
		return "", fmt.Errorf("bech32 human-readable part %q is not npub", hrp)
	}
	raw, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("decode npub data: %w", err)
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("npub decoded to %d bytes, want 32", len(raw))
	}
	return hex.EncodeToString(raw), nil
}

func EncodeNsec(raw []byte) (string, error) {
	if len(raw) != 32 {
		return "", fmt.Errorf("nsec input must be 32 bytes")
	}
	data, err := bech32.ConvertBits(raw, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("encode nsec data: %w", err)
	}
	return bech32.Encode("nsec", data)
}

func (k *KeyPair) SecretHex() string {
	return hex.EncodeToString(k.Private.Serialize())
}

func (k *KeyPair) Nsec() (string, error) {
	return EncodeNsec(k.Private.Serialize())
}

func (k *KeyPair) PublicHex() string {
	return hex.EncodeToString(schnorr.SerializePubKey(k.Public))
}

func ParsePublicKeyHex(value string) (*btcec.PublicKey, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode pubkey hex: %w", err)
	}
	if len(raw) != schnorr.PubKeyBytesLen {
		return nil, fmt.Errorf("pubkey must be 32 bytes")
	}
	return schnorr.ParsePubKey(raw)
}

func (t Tags) Equal(other Tags) bool {
	if len(t) != len(other) {
		return false
	}
	for i := range t {
		if len(t[i]) != len(other[i]) {
			return false
		}
		for j := range t[i] {
			if t[i][j] != other[i][j] {
				return false
			}
		}
	}
	return true
}

func (e Event) CanonicalJSON() ([]byte, error) {
	return json.Marshal([]any{0, e.PubKey, e.CreatedAt, e.Kind, e.Tags, e.Content})
}

func (e *Event) ComputeID() (string, error) {
	canonical, err := e.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func (e *Event) Sign(keys *KeyPair) error {
	if keys == nil || keys.Private == nil {
		return errors.New("signing key is required")
	}
	e.PubKey = keys.PublicHex()
	if e.CreatedAt == 0 {
		e.CreatedAt = time.Now().Unix()
	}
	id, err := e.ComputeID()
	if err != nil {
		return err
	}
	digest, err := hex.DecodeString(id)
	if err != nil {
		return err
	}
	sig, err := schnorr.Sign(keys.Private, digest)
	if err != nil {
		return fmt.Errorf("sign event: %w", err)
	}
	e.ID = id
	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}

func (e Event) Verify() (bool, error) {
	id, err := e.ComputeID()
	if err != nil {
		return false, err
	}
	if e.ID != "" && e.ID != id {
		return false, nil
	}
	if e.Sig == "" {
		return false, errors.New("signature is empty")
	}
	pub, err := ParsePublicKeyHex(e.PubKey)
	if err != nil {
		return false, err
	}
	sigBytes, err := hex.DecodeString(e.Sig)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return false, fmt.Errorf("parse signature: %w", err)
	}
	digest, err := hex.DecodeString(id)
	if err != nil {
		return false, err
	}
	return sig.Verify(digest, pub), nil
}

func (e Event) MustJSON() []byte {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return b
}

func NewUnsignedEvent(kind int, pubkey string, content string, tags Tags, createdAt int64) Event {
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	return Event{
		PubKey:    strings.ToLower(pubkey),
		CreatedAt: createdAt,
		Kind:      kind,
		Tags:      tags,
		Content:   content,
	}
}

func BuildAuthEvent(challenge, relayURL string, keys *KeyPair, authTags ...Tag) (Event, error) {
	tags := Tags{{"relay", relayURL}, {"challenge", challenge}}
	for _, tag := range authTags {
		if len(tag) > 0 {
			tags = append(tags, tag)
		}
	}
	event := NewUnsignedEvent(KindAuth, "", "", tags, 0)
	if err := event.Sign(keys); err != nil {
		return Event{}, err
	}
	return event, nil
}

func BuildProfileEvent(keys *KeyPair, displayName, name, picture, about, nip05 string, authTags ...Tag) (Event, error) {
	content := profileContent{
		DisplayName: omitEmpty(displayName),
		Name:        omitEmpty(name),
		Picture:     omitEmpty(picture),
		About:       omitEmpty(about),
		NIP05:       omitEmpty(nip05),
	}
	body, err := json.Marshal(content)
	if err != nil {
		return Event{}, err
	}
	tags := Tags{}
	for _, tag := range authTags {
		if len(tag) > 0 {
			tags = append(tags, tag)
		}
	}
	event := NewUnsignedEvent(KindProfile, "", string(body), tags, 0)
	if err := event.Sign(keys); err != nil {
		return Event{}, err
	}
	return event, nil
}

type profileContent struct {
	DisplayName *string `json:"display_name,omitempty"`
	Name        *string `json:"name,omitempty"`
	Picture     *string `json:"picture,omitempty"`
	About       *string `json:"about,omitempty"`
	NIP05       *string `json:"nip05,omitempty"`
}

func omitEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func MintAuthTag(owner *KeyPair, agentPubHex, conditions string) (Tag, error) {
	if owner == nil || owner.Private == nil {
		return nil, errors.New("owner key is required")
	}
	agentPubHex = strings.ToLower(strings.TrimSpace(agentPubHex))
	if _, err := ParsePublicKeyHex(agentPubHex); err != nil {
		return nil, fmt.Errorf("agent pubkey: %w", err)
	}
	if err := ValidateAuthConditions(conditions); err != nil {
		return nil, err
	}
	ownerPub := owner.PublicHex()
	if ownerPub == agentPubHex {
		return nil, errors.New("owner key cannot attest itself")
	}
	digest := hashAuthPreimage(agentPubHex, conditions)
	sig, err := schnorr.Sign(owner.Private, digest)
	if err != nil {
		return nil, fmt.Errorf("sign auth tag: %w", err)
	}
	return Tag{"auth", ownerPub, conditions, hex.EncodeToString(sig.Serialize())}, nil
}

func VerifyAuthTag(tag Tag, agentPubHex string) (string, error) {
	agentPubHex = strings.ToLower(strings.TrimSpace(agentPubHex))
	if _, err := ParsePublicKeyHex(agentPubHex); err != nil {
		return "", fmt.Errorf("agent pubkey: %w", err)
	}
	if len(tag) != 4 || tag[0] != "auth" {
		return "", errors.New("auth tag must be [auth, owner_pubkey, conditions, sig]")
	}
	ownerPubHex := strings.ToLower(strings.TrimSpace(tag[1]))
	conditions := tag[2]
	sigHex := strings.ToLower(strings.TrimSpace(tag[3]))
	if ownerPubHex == agentPubHex {
		return "", errors.New("auth tag cannot be self-attested")
	}
	ownerPub, err := ParsePublicKeyHex(ownerPubHex)
	if err != nil {
		return "", fmt.Errorf("owner pubkey: %w", err)
	}
	if err := ValidateAuthConditions(conditions); err != nil {
		return "", err
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("decode auth signature: %w", err)
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return "", fmt.Errorf("parse auth signature: %w", err)
	}
	if !sig.Verify(hashAuthPreimage(agentPubHex, conditions), ownerPub) {
		return "", errors.New("auth tag signature is invalid")
	}
	return ownerPubHex, nil
}

func AuthTagJSON(tag Tag) (string, error) {
	b, err := json.Marshal(tag)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ParseAuthTagJSON(value string) (Tag, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var tag Tag
	if err := json.Unmarshal([]byte(value), &tag); err != nil {
		return nil, fmt.Errorf("parse auth tag JSON: %w", err)
	}
	return tag, nil
}

func hashAuthPreimage(agentPubHex, conditions string) []byte {
	preimage := "nostr:agent-auth:" + agentPubHex + ":" + conditions
	sum := sha256.Sum256([]byte(preimage))
	return sum[:]
}

func ValidateAuthConditions(value string) error {
	if value == "" {
		return nil
	}
	for _, r := range value {
		if unicode.IsSpace(r) {
			return errors.New("auth tag conditions must not contain whitespace")
		}
	}
	for _, clause := range strings.Split(value, "&") {
		if clause == "" {
			return errors.New("auth tag condition contains an empty clause")
		}
		if strings.HasPrefix(clause, "kind=") {
			n, err := parseCanonicalUint(clause[len("kind="):])
			if err != nil {
				return fmt.Errorf("invalid kind condition: %w", err)
			}
			if n > 65535 {
				return errors.New("kind condition exceeds 65535")
			}
			continue
		}
		if strings.HasPrefix(clause, "created_at<") || strings.HasPrefix(clause, "created_at>") {
			n, err := parseCanonicalUint(clause[len("created_at<"):])
			if err != nil {
				return fmt.Errorf("invalid created_at condition: %w", err)
			}
			if n == 0 {
				return errors.New("created_at condition must be positive")
			}
			continue
		}
		return fmt.Errorf("unsupported auth tag condition %q", clause)
	}
	return nil
}

func parseCanonicalUint(value string) (uint64, error) {
	if value == "" {
		return 0, errors.New("empty number")
	}
	if len(value) > 1 && value[0] == '0' {
		return 0, errors.New("number must be canonical without leading zeros")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, errors.New("number contains non-digits")
		}
	}
	return strconv.ParseUint(value, 10, 64)
}

func allZero(raw []byte) bool {
	for _, b := range raw {
		if b != 0 {
			return false
		}
	}
	return true
}
