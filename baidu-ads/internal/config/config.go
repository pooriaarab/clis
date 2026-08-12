package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BaseURL string
	Token   string
	Env     string
	Name    string
}

func Load(env, baseURL, name string) *Config {
	c := &Config{BaseURL: strings.TrimRight(baseURL, "/"), Env: env, Name: name}
	if value := os.Getenv(env + "_BASE_URL"); value != "" {
		c.BaseURL = strings.TrimRight(value, "/")
	}
	c.Token = os.Getenv(env + "_ACCESS_TOKEN")
	if c.Token == "" {
		c.Token = os.Getenv(env + "_TOKEN")
	}
	if c.Token == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			if data, readErr := os.ReadFile(filepath.Join(home, ".config", name, "token")); readErr == nil {
				c.Token = strings.TrimSpace(string(data))
			}
		}
	}
	return c
}

func SaveToken(name, token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "token"), []byte(strings.TrimSpace(token)+"\n"), 0600)
}

func (c *Config) HasToken() bool { return strings.TrimSpace(c.Token) != "" }
