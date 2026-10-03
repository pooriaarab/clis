// Package sites loads the user-supplied sites YAML file.
//
// The file holds OUR sites (names, copy, contact) and is never shipped with
// the repo. See README for the documented shape and example.
package sites

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Limits from the submission targets. Launching Next caps the full
// description at 2500 characters and the headline at ~100.
const (
	MaxFull     = 2500
	MaxShort    = 280
	MaxMedium   = 500
	MaxHeadline = 100
	MaxName     = 100
)

// Funding maps to the Launching Next funding radio values.
var Funding = map[string]string{
	"side":         "1",
	"bootstrapped": "2",
	"funded":       "3",
	"none":         "0",
}

// Site is one of our directory websites.
type Site struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	URL       string   `yaml:"url" json:"url"`
	Tagline   string   `yaml:"tagline" json:"tagline"`
	OneLiner  string   `yaml:"one_liner" json:"one_liner"`
	Short     string   `yaml:"short" json:"short"`
	Medium    string   `yaml:"medium" json:"medium"`
	Headline  string   `yaml:"headline" json:"headline"`
	Full2200  string   `yaml:"full2200" json:"full2200"`
	Tags      []string `yaml:"tags" json:"tags"`
	Funding   string   `yaml:"funding" json:"funding"`
	Marketing string   `yaml:"marketing_budget" json:"marketing_budget"`
	Founder   string   `yaml:"founder" json:"founder"`
	Email     string   `yaml:"email" json:"email"`
}

// File is the top level of the sites YAML file.
type File struct {
	Sites []Site `yaml:"sites"`
}

// Load reads and validates path. It returns every problem found, so one run
// lists all fixes.
func Load(path string) ([]Site, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sites file %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("sites file %s is not valid YAML: %w", path, err)
	}
	if len(f.Sites) == 0 {
		return nil, fmt.Errorf("sites file %s lists no sites", path)
	}
	var problems []string
	seen := map[string]bool{}
	for i := range f.Sites {
		s := &f.Sites[i]
		where := fmt.Sprintf("site #%d", i+1)
		if s.ID != "" {
			where = fmt.Sprintf("site %q", s.ID)
		}
		if s.ID == "" {
			problems = append(problems, where+": id is required")
		} else if seen[s.ID] {
			problems = append(problems, where+": duplicate id")
		}
		seen[s.ID] = true
		if s.Name == "" {
			problems = append(problems, where+": name is required")
		} else if len(s.Name) > MaxName {
			problems = append(problems, fmt.Sprintf("%s: name is %d characters, max %d", where, len(s.Name), MaxName))
		}
		if s.URL == "" {
			problems = append(problems, where+": url is required")
		} else if u, err := url.Parse(s.URL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			problems = append(problems, where+": url must be an absolute http(s) URL")
		}
		if s.Headline == "" {
			s.Headline = s.Tagline // headline defaults to tagline
		}
		if len(s.Headline) > MaxHeadline {
			problems = append(problems, fmt.Sprintf("%s: headline is %d characters, max %d", where, len(s.Headline), MaxHeadline))
		}
		if len(s.Short) > MaxShort {
			problems = append(problems, fmt.Sprintf("%s: short is %d characters, max %d", where, len(s.Short), MaxShort))
		}
		if len(s.Medium) > MaxMedium {
			problems = append(problems, fmt.Sprintf("%s: medium is %d characters, max %d", where, len(s.Medium), MaxMedium))
		}
		if len(s.Full2200) > MaxFull {
			problems = append(problems, fmt.Sprintf("%s: full2200 is %d characters, max %d", where, len(s.Full2200), MaxFull))
		}
		if len(s.Tags) < 5 || len(s.Tags) > 10 {
			problems = append(problems, fmt.Sprintf("%s: want 5-10 tags, got %d", where, len(s.Tags)))
		}
		if s.Funding == "" {
			s.Funding = "none"
		} else if _, ok := Funding[s.Funding]; !ok {
			problems = append(problems, where+": funding must be one of side, bootstrapped, funded, none")
		}
		if s.Marketing == "" {
			s.Marketing = "$0"
		}
		if s.Founder == "" {
			problems = append(problems, where+": founder is required")
		}
		if !strings.Contains(s.Email, "@") {
			problems = append(problems, where+": email must contain @")
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("sites file %s has %d problem(s):\n- %s", path, len(problems), strings.Join(problems, "\n- "))
	}
	return f.Sites, nil
}

// ByID returns the site with id, or nil.
func ByID(sites []Site, id string) *Site {
	for i := range sites {
		if sites[i].ID == id {
			return &sites[i]
		}
	}
	return nil
}
