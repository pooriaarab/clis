package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrecedenceFlagsEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	fileKey := keyHex("file")
	envKey := keyHex("env")
	flagKey := keyHex("flag")
	ownerKey := keyHex("owner")
	if err := os.WriteFile(path, []byte(`
relay_url = "file-relay"
owner_key = "`+fileKey+`"

[identities]
file = "`+fileKey+`"
`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("BUZZ_RELAY_URL", "env-relay")
	t.Setenv("BUZZ_PRIVATE_KEY", envKey)
	t.Setenv("BUZZ_AUTH_TAG", `["auth","`+keyHex("auth owner")+`","","`+sigHex("auth sig")+`"]`)
	t.Setenv("BUZZ_OWNER_KEY", ownerKey)

	resolved, err := Resolve(Options{
		ConfigPath: path,
		RelayURL:   "flag-relay",
		PrivateKey: flagKey,
		Identity:   "file",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved.RelayURL != "flag-relay" {
		t.Fatalf("RelayURL = %q", resolved.RelayURL)
	}
	if resolved.PrivateKey != flagKey {
		t.Fatalf("PrivateKey did not prefer flag")
	}
	if resolved.OwnerKey != ownerKey {
		t.Fatalf("OwnerKey did not prefer env")
	}
	if resolved.AuthTag == "" {
		t.Fatalf("AuthTag not loaded from env")
	}
}

func TestSaveIdentityPersistsAuthTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	secret := keyHex("saved")
	cfg := File{RelayURL: "relay-name", Identities: map[string]string{}}
	if err := cfg.SaveIdentity(path, "agent-name", secret, "auth-tag-value"); err != nil {
		t.Fatalf("SaveIdentity() error = %v", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if loaded.Identities["agent-name"] != secret {
		t.Fatalf("identity not saved")
	}
	if loaded.AuthTags["agent-name"] != "auth-tag-value" {
		t.Fatalf("auth tag not saved")
	}
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
