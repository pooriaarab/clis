package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	EnvConfigPath  = "BUZZ_CONFIG"
	EnvRelayURL    = "BUZZ_RELAY_URL"
	EnvPrivateKey  = "BUZZ_PRIVATE_KEY"
	EnvAuthTag     = "BUZZ_AUTH_TAG"
	EnvOwnerKey    = "BUZZ_OWNER_KEY"
	defaultDirName = "buzz-cli"
	defaultFile    = "config.toml"
)

type File struct {
	RelayURL   string                 `toml:"relay_url,omitempty"`
	OwnerKey   string                 `toml:"owner_key,omitempty"`
	Identities map[string]string      `toml:"identities,omitempty"`
	AuthTags   map[string]string      `toml:"auth_tags,omitempty"`
	Agents     map[string]AgentRecord `toml:"agents,omitempty"`
	Invites    []InviteRecord         `toml:"invites,omitempty"`
}

// AgentRecord is everything needed to run a managed agent later, keyed by
// agent name in File.Agents. ACPCommand/HarnessCommand are optional: when
// unset, `agents run`/`fleet` fall back to their own command-line defaults.
type AgentRecord struct {
	Nsec           string `toml:"nsec"`
	AuthTag        string `toml:"auth_tag,omitempty"`
	RelayURL       string `toml:"relay,omitempty"`
	ACPCommand     string `toml:"acp_command,omitempty"`
	HarnessCommand string `toml:"harness_command,omitempty"`
}

type InviteRecord struct {
	Code          string `toml:"code"`
	URL           string `toml:"url,omitempty"`
	ExpiresAt     int64  `toml:"expires_at,omitempty"`
	MaxUses       int    `toml:"max_uses,omitempty"`
	UsesRemaining int    `toml:"uses_remaining,omitempty"`
	CreatedAt     int64  `toml:"created_at"`
}

type Options struct {
	ConfigPath string
	RelayURL   string
	Identity   string
	PrivateKey string
	AuthTag    string
	OwnerKey   string
}

type Resolved struct {
	ConfigPath string
	RelayURL   string
	Identity   string
	PrivateKey string
	AuthTag    string
	OwnerKey   string
	File       File
}

func Resolve(opts Options) (Resolved, error) {
	path := opts.ConfigPath
	if path == "" {
		path = os.Getenv(EnvConfigPath)
	}
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Resolved{}, err
		}
	}

	file, err := LoadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Resolved{}, err
		}
		file = File{}
	}
	file.ensureMaps()

	identity := strings.TrimSpace(opts.Identity)
	resolved := Resolved{
		ConfigPath: path,
		Identity:   identity,
		File:       file,
	}
	resolved.RelayURL = firstNonEmpty(opts.RelayURL, os.Getenv(EnvRelayURL), file.RelayURL)
	resolved.PrivateKey = firstNonEmpty(opts.PrivateKey, os.Getenv(EnvPrivateKey))
	if resolved.PrivateKey == "" && identity != "" {
		resolved.PrivateKey = file.Identities[identity]
	}
	resolved.AuthTag = firstNonEmpty(opts.AuthTag, os.Getenv(EnvAuthTag))
	if resolved.AuthTag == "" && identity != "" {
		resolved.AuthTag = file.AuthTags[identity]
	}
	resolved.OwnerKey = firstNonEmpty(opts.OwnerKey, os.Getenv(EnvOwnerKey), file.OwnerKey)
	return resolved, nil
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config dir: %w", err)
	}
	return filepath.Join(dir, defaultDirName, defaultFile), nil
}

func LoadFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var file File
	if err := toml.Unmarshal(b, &file); err != nil {
		return File{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	file.ensureMaps()
	return file, nil
}

// SaveFile writes the config atomically (temp file + rename in the same
// directory, so a crash mid-write can't corrupt the existing file) and
// chmods it 0600 since it holds secrets (nsecs, auth tags, owner key).
func SaveFile(path string, file File) error {
	file.ensureMaps()
	b, err := toml.Marshal(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (f File) Save(path string) error {
	return SaveFile(path, f)
}

func (f File) SaveIdentity(path, name, secret, authTag string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("identity name is required")
	}
	f.ensureMaps()
	f.Identities[name] = secret
	if authTag != "" {
		f.AuthTags[name] = authTag
	}
	return SaveFile(path, f)
}

// SaveAgent persists a managed agent's identity so it can be run later by
// name alone: the nsec/auth tag also land in the legacy Identities/AuthTags
// maps (so --identity lookups elsewhere keep working), and the full record
// (including relay + runtime commands) lands in Agents[name].
func (f File) SaveAgent(path, name string, record AgentRecord) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("identity name is required")
	}
	f.ensureMaps()
	f.Identities[name] = record.Nsec
	if record.AuthTag != "" {
		f.AuthTags[name] = record.AuthTag
	}
	f.Agents[name] = record
	return SaveFile(path, f)
}

func (f File) AppendInvite(path string, record InviteRecord) error {
	f.ensureMaps()
	f.Invites = append(f.Invites, record)
	return SaveFile(path, f)
}

func SaveIdentity(path, name, secret, authTag string) error {
	file, err := LoadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		file = File{}
	}
	return file.SaveIdentity(path, name, secret, authTag)
}

func (f *File) ensureMaps() {
	if f.Identities == nil {
		f.Identities = map[string]string{}
	}
	if f.AuthTags == nil {
		f.AuthTags = map[string]string{}
	}
	if f.Agents == nil {
		f.Agents = map[string]AgentRecord{}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
