// Package config locates the submit config directory and tracker store.
package config

import (
	"os"
	"path/filepath"
)

// Dir returns the config directory. SUBMIT_CONFIG_DIR wins, then
// ~/.config/submit.
func Dir(getenv func(string) string) (string, error) {
	if d := getenv("SUBMIT_CONFIG_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "submit"), nil
}

// StorePath returns the tracker store file path inside dir.
func StorePath(dir string) string {
	return filepath.Join(dir, "store.json")
}
