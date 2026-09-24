// Package docs stores one solicitation's attachments. A missing acceptance is not an acceptance.
package docs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
)

type File struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	SourceURL string `json:"source_url"`
	FetchedAt string `json:"fetched_at"`
}
type Acceptance struct {
	SolicitationID string `json:"solicitation_id"`
	Filename       string `json:"filename"`
	AcceptedAt     string `json:"accepted_at"`
}
type Skip struct {
	SolicitationID string `json:"solicitation_id"`
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	At             string `json:"at"`
}
type Manifest struct {
	SolicitationID string       `json:"solicitation_id"`
	Documents      []File       `json:"documents"`
	Acceptances    []Acceptance `json:"acceptances"`
	SkippedGates   []Skip       `json:"skipped_gates,omitempty"`
}

const gateSkip, manFile = "gated-and-skipped", "manifest.json"

func Fetch(c *httpx.Client, portal, id, dest string) (*Manifest, error) {
	if id == "" || strings.ContainsAny(id, "/\\.?#") {
		return nil, fmt.Errorf("invalid solicitation id")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	man, err := loadManifest(dest)
	if err != nil {
		return nil, err
	}
	man.SolicitationID = id
	list, err := abs(portal, "/public/solicitations/"+id+"/abstract/docs-items")
	if err != nil {
		return nil, err
	}
	st, final, body, _, _, err := hop(c, http.MethodGet, list, nil, true, 8)
	if err != nil {
		return nil, err
	}
	if strings.Contains(final, "/req-ack") {
		skip, err := acceptOrSkip(c, portal, final, id, dest, &man)
		if err != nil || skip {
			return &man, err
		}
		if st, final, body, _, _, err = hop(c, http.MethodGet, list, nil, true, 8); err != nil {
			return &man, err
		}
	}
	if st != http.StatusOK {
		return &man, fmt.Errorf("GET %s: HTTP %d", final, st)
	}
	entries, err := search.ParseDocList(bytes.NewReader(body))
	if err != nil {
		return &man, err
	}
	var files []File
next:
	for _, e := range entries {
		for _, f := range man.Documents {
			if raw, err := os.ReadFile(filepath.Join(dest, filepath.Base(f.Filename))); err == nil && f.ID == e.ID && f.SHA256 != "" && sum(raw) == f.SHA256 {
				fmt.Fprintf(os.Stderr, "merx: skip %s (checksum match)\n", e.Filename)
				files = append(files, f)
				continue next
			}
		}
		f, err := download(c, portal, dest, e)
		if err != nil {
			return &man, err
		}
		files = append(files, f)
	}
	man.Documents = files
	return &man, writeManifest(dest, &man)
}

func acceptOrSkip(c *httpx.Client, portal, page, id, dest string, man *Manifest) (bool, error) {
	st, final, body, _, _, err := hop(c, http.MethodGet, page, nil, false, 8)
	if err != nil {
		return false, err
	}
	if st != http.StatusOK {
		return false, fmt.Errorf("GET %s: HTTP %d", final, st)
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	ack, err := search.ParseAck(bytes.NewReader(body))
	if err != nil || ack.AcceptName == "" {
		man.SkippedGates = append(man.SkippedGates, Skip{id, gateSkip, "unparseable acknowledgement page", ts})
		fmt.Fprintf(os.Stderr, "merx: gated-and-skipped %s at %s (acknowledgement page could not be parsed)\n", id, ts)
		return true, writeManifest(dest, man)
	}
	vals := url.Values{}
	for k, v := range ack.Fields {
		vals.Set(k, v)
	}
	vals.Set(ack.AcceptName, ack.AcceptValue)
	target, err := abs(final, ack.Action)
	if err != nil {
		return false, err
	}
	if st, _, _, _, _, err = hop(c, http.MethodPost, target, vals, false, 8); err != nil {
		return false, err
	} else if st >= 400 {
		return false, fmt.Errorf("accept POST %s: HTTP %d", target, st)
	}
	man.Acceptances = append(man.Acceptances, Acceptance{id, ack.Filename, ts})
	fmt.Fprintf(os.Stderr, "merx: accepted %s for %s at %s\n", ack.Filename, id, ts)
	return false, writeManifest(dest, man)
}

func download(c *httpx.Client, portal, dest string, e search.Doc) (File, error) {
	u, err := abs(portal, e.URL)
	if err != nil {
		return File{}, err
	}
	fmt.Fprintf(os.Stderr, "merx: downloading %s\n", e.Filename)
	var body []byte
	var ct, disp, final, href string
	for i := 0; i < 2; i++ {
		var st int
		st, final, body, ct, disp, err = hop(c, http.MethodGet, u, nil, true, 8)
		if err != nil {
			return File{}, err
		}
		if st != http.StatusOK || i > 0 && (strings.Contains(ct, "html") || strings.Contains(ct, "javascript")) {
			return File{}, fmt.Errorf("GET %s: HTTP %d (%s)", final, st, ct)
		}
		if i > 0 {
			break
		}
		links, err := search.ParseDocList(bytes.NewReader(body))
		if err != nil {
			return File{}, err
		}
		for _, d := range links {
			if strings.Contains(d.URL, "attachment-preview-download") {
				href = d.URL
				break
			}
		}
		if href == "" {
			return File{}, fmt.Errorf("no download link in preview dialog")
		}
		if u, err = abs(portal, href); err != nil {
			return File{}, err
		}
	}
	name := e.Filename
	if _, p, err := mime.ParseMediaType(disp); err == nil && p["filename"] != "" {
		name = p["filename"]
	}
	for _, s := range []string{name, e.Filename, "download"} {
		s = filepath.Base(strings.ReplaceAll(s, "\\", "/"))
		if s != "" && s != "." && s != ".." {
			name = s
			break
		}
		name = "download"
	}
	if err := os.WriteFile(filepath.Join(dest, name), body, 0o644); err != nil {
		return File{}, err
	}
	return File{e.ID, name, int64(len(body)), sum(body), u, time.Now().UTC().Format(time.RFC3339)}, nil
}

func hop(c *httpx.Client, method, rawurl string, vals url.Values, xhr bool, max int) (int, string, []byte, string, string, error) {
	cur := rawurl
	for n := 0; n < max; n++ {
		req, err := httpx.NewRequest(method, cur)
		if err != nil {
			return 0, "", nil, "", "", err
		}
		if vals != nil {
			enc := vals.Encode()
			req.ContentLength, req.Body = int64(len(enc)), io.NopCloser(strings.NewReader(enc))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if xhr {
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
		}
		resp, err := c.Do(req)
		if err != nil {
			return 0, "", nil, "", "", err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		st, loc, ct, disp := resp.StatusCode, resp.Header.Get("Location"), resp.Header.Get("Content-Type"), resp.Header.Get("Content-Disposition")
		if err != nil || st < 300 || st > 399 || max == 1 {
			return st, cur, body, ct, disp, err
		}
		next, err := abs(cur, loc)
		if loc == "" || err != nil {
			return st, cur, body, ct, disp, fmt.Errorf("redirect: %w", err)
		}
		if strings.Contains(next, "/req-ack") {
			return st, next, body, ct, disp, nil
		}
		if strings.Contains(next, "/authentication/login") || strings.Contains(next, "/saml/login") {
			return st, next, body, ct, disp, fmt.Errorf("not authenticated: run merx login")
		}
		cur, vals, method = next, nil, http.MethodGet
	}
	return 0, "", nil, "", "", fmt.Errorf("too many redirects")
}

func loadManifest(dir string) (Manifest, error) {
	var man Manifest
	raw, err := os.ReadFile(filepath.Join(dir, manFile))
	if os.IsNotExist(err) {
		return man, nil
	}
	if err != nil {
		return man, err
	}
	return man, json.Unmarshal(raw, &man)
}
func writeManifest(dir string, man *Manifest) error {
	man.Acceptances, man.Documents = append([]Acceptance{}, man.Acceptances...), append([]File{}, man.Documents...)
	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manFile), append(raw, '\n'), 0o644)
}
func abs(portal, rawurl string) (string, error) {
	p, err := url.Parse(portal)
	if err != nil {
		return "", err
	}
	u, err := p.Parse(rawurl)
	if err != nil || u.Host != p.Host || u.Scheme != p.Scheme {
		if err == nil {
			err = fmt.Errorf("refused off-host URL")
		}
		return "", err
	}
	return u.String(), nil
}
func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
