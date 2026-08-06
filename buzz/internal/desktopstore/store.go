package desktopstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// DefaultStorePath returns the real desktop app's managed-agents.json path.
// Callers must NEVER call this in a test - tests always pass an explicit
// temp-dir path via --store-path.
func DefaultStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "xyz.block.buzz.app", "agents", "managed-agents.json"), nil
}

// LoadRaw reads the managed-agents.json array as opaque elements, so callers
// can splice one out or append one without risking dropping fields this
// package's ManagedAgentRecord struct doesn't model.
func LoadRaw(path string) ([]json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return []json.RawMessage{}, nil
	}
	var records []json.RawMessage
	if err := json.Unmarshal(b, &records); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if records == nil {
		return []json.RawMessage{}, nil
	}
	return records, nil
}

// SaveRaw writes records back to path with a required backup of the existing
// file and an atomic temp-file rename.
func SaveRaw(path string, records []json.RawMessage) error {
	if records == nil {
		records = []json.RawMessage{}
	}
	dir := filepath.Dir(path)
	if b, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", b, 0o600); err != nil {
			return fmt.Errorf("backup %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".managed-agents-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

// FileLock is a best-effort advisory lock on lockPath, held for the duration
// of a load-mutate-save cycle.
func WithFileLock(lockPath string, timeout time.Duration, fn func() error) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fn()
	}
	defer f.Close()

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			return fn()
		}
		if time.Now().After(deadline) {
			return fn()
		}
		time.Sleep(50 * time.Millisecond)
	}
}
