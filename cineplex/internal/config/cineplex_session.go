// hand-authored novel config extension; preserve across regenerate

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cineplex-pp-cli/internal/cliutil"
	"github.com/pelletier/go-toml/v2"
)

const (
	sceneSessionTokenEnv = "CINEPLEX_SESSION_TOKEN"
	sceneSessionTokenKey = "scene_session_token"

	// The SCENE+ web session is a browser COOKIE (confirmed by capturing a real
	// checkout): authed ticketing endpoints (seat-availability-for-cart,
	// ticketing-cart, reserve-seats) authenticate via this cookie plus the
	// subscription key, NOT a bearer token.
	sceneSessionCookieEnv = "CINEPLEX_SESSION_COOKIE"
	sceneSessionCookieKey = "scene_session_cookie"
)

// SceneSessionCookie returns the SCENE+ session cookie for this loaded config,
// preferring the environment. This is the credential authed ticketing calls
// send in the Cookie header.
func (c *Config) SceneSessionCookie() string {
	if v := strings.TrimSpace(os.Getenv(sceneSessionCookieEnv)); v != "" {
		return v
	}
	if c == nil {
		return ""
	}
	if v := sceneConfigValue(c.Path, sceneSessionCookieKey); v != "" {
		return v
	}
	if c.legacySourcePath != "" && c.legacySourcePath != c.Path {
		return sceneConfigValue(c.legacySourcePath, sceneSessionCookieKey)
	}
	return ""
}

// SaveSceneSessionCookie stores the SCENE+ session cookie as a private config
// value without changing the generated Config struct.
func SaveSceneSessionCookie(configPath, cookie string) error {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return fmt.Errorf("session cookie is required")
	}
	return saveSceneConfigValue(configPath, sceneSessionCookieKey, cookie)
}

func sceneConfigValue(path, key string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return ""
	}
	if cliutil.VerifyCredsPerms(real) != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- app-owned config path.
	if err != nil {
		return ""
	}
	var values map[string]any
	if err := toml.Unmarshal(data, &values); err != nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	s, _ := value.(string)
	return strings.TrimSpace(s)
}

func saveSceneConfigValue(configPath, key, value string) error {
	path, _, err := resolveConfigPath(configPath)
	if err != nil {
		return err
	}
	values := map[string]any{}
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- app-owned config path.
	if err == nil {
		if err := toml.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config: %w", err)
	}
	values[key] = value
	data, err = toml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return cliutil.AtomicWritePrivateFile(path, data, 0o600, 0o700)
}

// SceneSessionToken returns the SCENE+ session token from the environment or
// the optional private config value. Environment values take precedence.
func SceneSessionToken() string {
	if token := strings.TrimSpace(os.Getenv(sceneSessionTokenEnv)); token != "" {
		return token
	}
	path, _, err := resolveConfigPath("")
	if err != nil {
		return ""
	}
	return sceneSessionTokenFromPath(path)
}

// SceneSessionToken returns the SCENE+ session token for this loaded config.
// It also checks the legacy source when Load used the legacy config path.
func (c *Config) SceneSessionToken() string {
	if token := strings.TrimSpace(os.Getenv(sceneSessionTokenEnv)); token != "" {
		return token
	}
	if c == nil {
		return ""
	}
	if token := sceneSessionTokenFromPath(c.Path); token != "" {
		return token
	}
	if c.legacySourcePath != "" && c.legacySourcePath != c.Path {
		return sceneSessionTokenFromPath(c.legacySourcePath)
	}
	return ""
}

// SaveSceneSessionToken stores the SCENE+ token as a private config value
// without changing the generated Config struct or its generated save logic.
func SaveSceneSessionToken(configPath, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("session token is required")
	}
	path, _, err := resolveConfigPath(configPath)
	if err != nil {
		return err
	}

	values := map[string]any{}
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- app-owned config path.
	if err == nil {
		if err := toml.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config: %w", err)
	}
	values[sceneSessionTokenKey] = token

	data, err = toml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return cliutil.AtomicWritePrivateFile(path, data, 0o600, 0o700)
}

func sceneSessionTokenFromPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return ""
	}
	if cliutil.VerifyCredsPerms(real) != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- app-owned config path.
	if err != nil {
		return ""
	}
	var values map[string]any
	if err := toml.Unmarshal(data, &values); err != nil {
		return ""
	}
	value, ok := values[sceneSessionTokenKey]
	if !ok {
		return ""
	}
	token, _ := value.(string)
	return strings.TrimSpace(token)
}
