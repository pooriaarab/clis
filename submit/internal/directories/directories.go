// Package directories ships the ranked third-party directory list as built-in
// data. Names, URLs, costs and requirements of third parties are public facts.
// Our own sites never appear here; the CLI reads them from a user YAML file.
package directories

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed directories.json
var raw []byte

// Entry is one third-party directory.
type Entry struct {
	Rank         int    `json:"rank"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Bucket       string `json:"bucket"`
	SiteURL      string `json:"site_url"`
	SubmitURL    string `json:"submit_url"`
	Cost         string `json:"cost"`
	Requirements string `json:"requirements"`
	Authority    string `json:"authority"`
	Verticals    string `json:"verticals"`
	Automation   string `json:"automation"`
	Why          string `json:"why"`
	Wave         string `json:"wave"`
}

// Wave is one runbook wave.
type Wave struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Note  string `json:"note"`
}

// Data is the whole built-in file.
type Data struct {
	GeneratedFrom string  `json:"generated_from"`
	DailyLimit    int     `json:"daily_limit"`
	Waves         []Wave  `json:"waves"`
	Directories   []Entry `json:"directories"`
}

// Load parses the embedded data file.
func Load() (Data, error) {
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return Data{}, fmt.Errorf("built-in directory data is corrupt: %w", err)
	}
	return d, nil
}

// ByID returns the entry with id (case-insensitive), or nil.
func ByID(d Data, id string) *Entry {
	for i := range d.Directories {
		if strings.EqualFold(d.Directories[i].ID, id) {
			return &d.Directories[i]
		}
	}
	return nil
}

// ByName returns the entry whose id or name matches (case-insensitive), or nil.
func ByName(d Data, name string) *Entry {
	for i := range d.Directories {
		if strings.EqualFold(d.Directories[i].ID, name) || strings.EqualFold(d.Directories[i].Name, name) {
			return &d.Directories[i]
		}
	}
	return nil
}
