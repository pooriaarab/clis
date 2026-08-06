package desktopstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type SecretStore interface {
	Load(key string) (value string, ok bool, err error)
	Store(key, value string) error
	Delete(key string) error
}

// MemorySecretStore is an in-memory SecretStore for tests only.
type MemorySecretStore struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{data: map[string]string{}}
}

func (m *MemorySecretStore) Load(key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.data[key]
	return value, ok, nil
}

func (m *MemorySecretStore) Store(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = map[string]string{}
	}
	m.data[key] = value
	return nil
}

func (m *MemorySecretStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

type darwinKeychainStore struct {
	service string
}

const keychainBlobAccount = "secrets"

var errKeychainUnavailable = errors.New("macOS keychain (security CLI) unavailable")

// NewDarwinKeychainStore returns a SecretStore backed by the macOS `security`
// CLI for the given service name.
func NewDarwinKeychainStore(service string) SecretStore {
	return &darwinKeychainStore{service: service}
}

func (d *darwinKeychainStore) checkAvailable() error {
	if runtime.GOOS != "darwin" {
		return errKeychainUnavailable
	}
	if _, err := exec.LookPath("security"); err != nil {
		return errKeychainUnavailable
	}
	return nil
}

func (d *darwinKeychainStore) readBlob() (map[string]string, error) {
	var stderr strings.Builder
	cmd := exec.Command("security", "find-generic-password", "-a", keychainBlobAccount, "-s", d.service, "-w")
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		combined := strings.TrimSpace(string(stdout) + "\n" + stderr.String())
		if strings.Contains(strings.ToLower(combined), "could not be found") {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read macOS keychain blob: %w: %s", err, combined)
	}
	if strings.TrimSpace(string(stdout)) == "" {
		return map[string]string{}, nil
	}
	var blob map[string]string
	if err := json.Unmarshal(stdout, &blob); err != nil {
		return nil, fmt.Errorf("parse macOS keychain blob: %w", err)
	}
	if blob == nil {
		blob = map[string]string{}
	}
	return blob, nil
}

func (d *darwinKeychainStore) writeBlob(blob map[string]string) error {
	b, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	out, err := exec.Command("security", "add-generic-password", "-U", "-a", keychainBlobAccount, "-s", d.service, "-w", string(b)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("write macOS keychain blob: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *darwinKeychainStore) Load(key string) (string, bool, error) {
	if err := d.checkAvailable(); err != nil {
		return "", false, err
	}
	blob, err := d.readBlob()
	if err != nil {
		return "", false, err
	}
	value, ok := blob[key]
	return value, ok, nil
}

func (d *darwinKeychainStore) Store(key, value string) error {
	if err := d.checkAvailable(); err != nil {
		return err
	}
	lockPath := fmt.Sprintf("/tmp/buzz-keychain-%d-%s.lock", os.Getuid(), d.service)
	return WithFileLock(lockPath, 5*time.Second, func() error {
		blob, err := d.readBlob()
		if err != nil {
			return err
		}
		blob[key] = value
		return d.writeBlob(blob)
	})
}

func (d *darwinKeychainStore) Delete(key string) error {
	if err := d.checkAvailable(); err != nil {
		return err
	}
	lockPath := fmt.Sprintf("/tmp/buzz-keychain-%d-%s.lock", os.Getuid(), d.service)
	return WithFileLock(lockPath, 5*time.Second, func() error {
		blob, err := d.readBlob()
		if err != nil {
			return err
		}
		delete(blob, key)
		return d.writeBlob(blob)
	})
}
