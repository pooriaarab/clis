// Package kit renders the per-site copy kit as markdown + JSON.
package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pooriaarab/clis/submit/internal/sites"
)

// Render writes <out>/<id>.md and <out>/<id>.json for every site. It creates
// out when missing and returns the written paths.
func Render(sitesList []sites.Site, out string) ([]string, error) {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, fmt.Errorf("create kit dir %s: %w", out, err)
	}
	var paths []string
	for _, s := range sitesList {
		for _, ent := range []struct {
			path, body string
		}{
			{filepath.Join(out, s.ID+".md"), Markdown(s)},
			{filepath.Join(out, s.ID+".json"), JSON(s)},
		} {
			if err := os.WriteFile(ent.path, []byte(ent.body), 0o644); err != nil {
				return paths, fmt.Errorf("write %s: %w", ent.path, err)
			}
			paths = append(paths, ent.path)
		}
	}
	return paths, nil
}

// Planned returns the paths Render would write, without writing.
func Planned(sitesList []sites.Site, out string) []string {
	var paths []string
	for _, s := range sitesList {
		paths = append(paths, filepath.Join(out, s.ID+".md"), filepath.Join(out, s.ID+".json"))
	}
	return paths
}

// Markdown renders one site's copy kit.
func Markdown(s sites.Site) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", s.Name)
	fmt.Fprintf(&b, "- Site: %s\n", s.URL)
	fmt.Fprintf(&b, "- Contact: %s <%s>\n", s.Founder, s.Email)
	fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(s.Tags, ", "))
	fmt.Fprintf(&b, "\n## Tagline\n\n%s\n", s.Tagline)
	fmt.Fprintf(&b, "\n## One-liner\n\n%s\n", s.OneLiner)
	fmt.Fprintf(&b, "\n## Short (<=%d chars)\n\n%s\n", sites.MaxShort, s.Short)
	fmt.Fprintf(&b, "\n## Medium (<=%d chars)\n\n%s\n", sites.MaxMedium, s.Medium)
	fmt.Fprintf(&b, "\n## Headline (directory headline)\n\n%s\n", s.Headline)
	fmt.Fprintf(&b, "\n## Full (<=%d chars)\n\n%s\n", sites.MaxFull, s.Full2200)
	return b.String()
}

// JSON renders one site's copy kit as structured data.
func JSON(s sites.Site) string {
	v := map[string]any{
		"id": s.ID, "name": s.Name, "url": s.URL,
		"tagline": s.Tagline, "one_liner": s.OneLiner,
		"short": s.Short, "medium": s.Medium,
		"headline": s.Headline, "full2200": s.Full2200,
		"tags": s.Tags, "founder": s.Founder, "email": s.Email,
		"generated_by": "submit kit",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data) + "\n"
}
