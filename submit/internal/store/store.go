// Package store keeps the local (site x directory) submission tracker as one
// JSON file. Statuses: draft, submitted, live, rejected.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Statuses accepted by track set.
const (
	Draft     = "draft"
	Submitted = "submitted"
	Live      = "live"
	Rejected  = "rejected"
)

// DailyLimit is the max submissions per day. Automated-looking bursts get
// flagged, so set() refuses the 11th submission dated one day.
const DailyLimit = 10

// Record is one (site x directory) row.
type Record struct {
	Site       string `json:"site"`
	SiteURL    string `json:"site_url,omitempty"`
	Directory  string `json:"directory"`
	Status     string `json:"status"`
	ListingURL string `json:"listing_url,omitempty"`
	Date       string `json:"date"`
	UpdatedAt  string `json:"updated_at"`
	Note       string `json:"note,omitempty"`
}

// File is the on-disk shape.
type File struct {
	Version int               `json:"version"`
	Records map[string]Record `json:"records"`
}

// ValidStatus reports whether s is a known status.
func ValidStatus(s string) bool {
	switch s {
	case Draft, Submitted, Live, Rejected:
		return true
	}
	return false
}

// Counts as a submission for the daily rate limit: anything past draft.
func countsAsSubmission(s string) bool {
	return s == Submitted || s == Live || s == Rejected
}

func key(site, dir string) string { return site + "\x00" + dir }

// Load reads path. A missing file returns an empty store, not an error.
func Load(path string) (*File, error) {
	f := &File{Version: 1, Records: map[string]Record{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("tracker store %s is not valid JSON: %w", path, err)
	}
	if f.Records == nil {
		f.Records = map[string]Record{}
	}
	return f, nil
}

// Save writes f to path with mode 0600, creating the directory (0700).
func (f *File) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600) // WriteFile keeps the mode of a file that exists
}

// SubmittedOn counts non-draft rows dated day, excluding one key.
func (f *File) SubmittedOn(day, exclude string) int {
	n := 0
	for k, r := range f.Records {
		if k == exclude {
			continue
		}
		if r.Date == day && countsAsSubmission(r.Status) {
			n++
		}
	}
	return n
}

// SetInput is one upsert.
type SetInput struct {
	Site, SiteURL, Directory, Status, ListingURL, Date, Note string
	Now                                                      time.Time
}

// Set upserts a row. Moving a row to a submission status (submitted, live,
// rejected) enforces the daily limit and refuses past it.
func (f *File) Set(in SetInput) (Record, error) {
	if !ValidStatus(in.Status) {
		return Record{}, fmt.Errorf("status must be one of draft, submitted, live, rejected (got %q)", in.Status)
	}
	if in.Date == "" {
		in.Date = in.Now.Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		return Record{}, fmt.Errorf("date must be YYYY-MM-DD (got %q)", in.Date)
	}
	k := key(in.Site, in.Directory)
	prev, had := f.Records[k]
	rec := Record{
		Site: in.Site, SiteURL: in.SiteURL, Directory: in.Directory,
		Status: in.Status, ListingURL: in.ListingURL, Date: in.Date,
		UpdatedAt: in.Now.UTC().Format(time.RFC3339), Note: in.Note,
	}
	if had {
		// Empty fields keep the previous value, so `set` needs only --status.
		if rec.SiteURL == "" {
			rec.SiteURL = prev.SiteURL
		}
		if rec.ListingURL == "" {
			rec.ListingURL = prev.ListingURL
		}
		if rec.Note == "" {
			rec.Note = prev.Note
		}
	}
	alreadyCounted := had && prev.Date == rec.Date && countsAsSubmission(prev.Status)
	if countsAsSubmission(rec.Status) && !alreadyCounted {
		if n := f.SubmittedOn(rec.Date, k); n >= DailyLimit {
			return Record{}, fmt.Errorf("daily limit reached: %d submissions already dated %s (max %d/day); spread the rest over following days", n, rec.Date, DailyLimit)
		}
	}
	f.Records[k] = rec
	return rec, nil
}

// List returns rows matching the filters (empty means all), sorted by
// site then directory.
func (f *File) List(site, dir, status string) []Record {
	var out []Record
	for _, r := range f.Records {
		if site != "" && r.Site != site {
			continue
		}
		if dir != "" && r.Directory != dir {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Site != out[j].Site {
			return out[i].Site < out[j].Site
		}
		return out[i].Directory < out[j].Directory
	})
	return out
}

// Find returns the row for site+directory, or nil.
func (f *File) Find(site, dir string) *Record {
	if r, ok := f.Records[key(site, dir)]; ok {
		return &r
	}
	return nil
}

// FindBySiteURL returns rows whose site id or site URL matches s.
func (f *File) FindBySiteURL(s string) []Record {
	var out []Record
	for _, r := range f.Records {
		if r.Site == s || (r.SiteURL != "" && r.SiteURL == s) {
			out = append(out, r)
		}
	}
	return out
}
