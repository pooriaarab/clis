// Package config stores credentials under ~/.config/search-console.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the config directory. SEARCH_CONSOLE_CONFIG_DIR wins, then
// ~/.config/search-console.
func Dir(getenv func(string) string) (string, error) {
	if d := getenv("SEARCH_CONSOLE_CONFIG_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "search-console"), nil
}

// Load reads dir/name.json into v. It reports false when the file is absent.
func Load(dir, name string, v any) (bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("%s.json is not valid JSON: %w", name, err)
	}
	return true, nil
}

// Save writes v to dir/name.json with mode 0600. It creates dir with mode 0700.
func Save(dir, name string, v any) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600) // WriteFile keeps the mode of a file that exists
}
