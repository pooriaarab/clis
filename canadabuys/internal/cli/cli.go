package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Hardcoded CanadaBuys open-data registry. kind is how later PRs select files.
type dataset struct{ ID, Title, Kind, URL string }

type meta struct {
	URL          string `json:"url"`
	FetchedAt    string `json:"fetched-at"`
	LastModified string `json:"Last-Modified"`
	Bytes        int64  `json:"bytes"`
	SHA256       string `json:"sha256"`
}

type listRow struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	URL          string `json:"url"`
	Cached       bool   `json:"cached"`
	Bytes        int64  `json:"bytes,omitempty"`
	FetchedAt    string `json:"fetched_at,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	Age          string `json:"age,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

var catalog = []dataset{
	{"open-tenders", "Open tender notices", "tenders", "https://canadabuys.canada.ca/opendata/pub/openTenderNotice-ouvertAvisAppelOffres.csv"},
	{"new-tenders", "New tender notices, published today", "tenders", "https://canadabuys.canada.ca/opendata/pub/newTenderNotice-nouvelAvisAppelOffres.csv"},
	{"tenders", "Tender notices, 2022-08 onward", "tenders", "https://canadabuys.canada.ca/opendata/pub/tenderNoticeComplete-avisAppelOffresComplet.csv"},
	{"tenders-legacy", "Tender notices, 2009 to 2022", "tenders", "https://canadabuys.canada.ca/opendata/pub/2009-2022-tenderNoticeHistorical-AvisAppelOffresHistorique.csv"},
	{"awards", "Award notices, 2022-08 onward", "awards", "https://canadabuys.canada.ca/opendata/pub/awardNoticeComplete-avisAttributionComplet.csv"},
	{"awards-legacy", "Award notices, 2012 to 2022-08", "awards", "https://canadabuys.canada.ca/opendata/pub/2012-2022-awardNoticeHistorical-avisAttributionHistorique.csv"},
	{"contracts", "Contract history, 2023-06 onward", "contracts", "https://canadabuys.canada.ca/opendata/pub/contractHistoryComplete-contratsOctroyesComplet.csv"},
	{"contracts-legacy", "Contract history, 2009-01 to 2023-05", "contracts", "https://canadabuys.canada.ca/opendata/pub/2009-2023-contractHistoryHistorical-contratsOctroyesHistorique.csv"},
	{"gsin-unspsc", "Mapping of GSIN to UNSPSC codes", "reference", "https://donnees-data.tpsgc-pwgsc.gc.ca/ba2/aev-bas/nibsunspsc-gsinunspsc.csv"},
}

// idleTimeout bounds gaps between body reads. ResponseHeaderTimeout only
// covers the wait for headers, so a connection that goes silent mid-transfer
// without closing would otherwise hang fetchOne forever.
const idleTimeout = 60 * time.Second

var (
	flagCache string
	flagJSON  bool
	// No overall Timeout: a progressing body can outlast an hour.
	client = &http.Client{Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}}
)

func Execute() error { return root().Execute() }

func root() *cobra.Command {
	cmd := &cobra.Command{Use: "canadabuys", Short: "Fetch and cache CanadaBuys open CSVs", SilenceUsage: true, SilenceErrors: true}
	cmd.PersistentFlags().StringVar(&flagCache, "cache-dir", "", "cache directory")
	cmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "JSON on stdout")
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.AddCommand(datasetsCmd(), fetchCmd(), doctorCmd(), tendersCmd(), awardsCmd(), statsCmd(), opportunitiesCmd())
	return cmd
}

func datasetsCmd() *cobra.Command {
	g := &cobra.Command{Use: "datasets", Short: "Dataset registry"}
	var kind string
	list := &cobra.Command{Use: "list", Short: "Print the dataset registry", RunE: func(*cobra.Command, []string) error {
		dir, err := cacheDir()
		if err != nil {
			return err
		}
		rs, err := listRows(dir, kind)
		if err != nil {
			return err
		}
		if flagJSON {
			return emit(rs)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tKIND\tCACHED\tAGE\tBYTES\tTITLE")
		for _, r := range rs {
			age, bytes := "-", "-"
			if r.Cached {
				age, bytes = r.Age, fmt.Sprintf("%d", r.Bytes)
			}
			fmt.Fprintf(w, "%s\t%s\t%t\t%s\t%s\t%s\n", r.ID, r.Kind, r.Cached, age, bytes, r.Title)
		}
		return w.Flush()
	}}
	list.Flags().StringVar(&kind, "kind", "", "filter by kind")
	g.AddCommand(list)
	return g
}

func fetchCmd() *cobra.Command {
	var all, force bool
	var kind string
	cmd := &cobra.Command{Use: "fetch [id...]", Short: "Download datasets into the cache", Args: cobra.ArbitraryArgs, RunE: func(_ *cobra.Command, ids []string) error {
		dir, err := cacheDir()
		if err != nil {
			return err
		}
		ds, err := pick(ids, all, kind)
		if err != nil {
			return err
		}
		type result struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Bytes  int64  `json:"bytes,omitempty"`
			SHA256 string `json:"sha256,omitempty"`
		}
		out := make([]result, 0, len(ds))
		for _, d := range ds {
			status, m, err := fetchOne(dir, d, force)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s %s (%d bytes)\n", status, d.ID, m.Bytes)
			out = append(out, result{d.ID, status, m.Bytes, m.SHA256})
		}
		if flagJSON {
			return emit(out)
		}
		return nil
	}}
	cmd.Flags().BoolVar(&all, "all", false, "fetch every dataset")
	cmd.Flags().BoolVar(&force, "force", false, "re-download even if fresh")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by kind")
	return cmd
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Report cache configuration and reachability", RunE: func(*cobra.Command, []string) error {
		dir, err := cacheDir()
		if err != nil {
			return err
		}
		rs, err := listRows(dir, "")
		if err != nil {
			return err
		}
		used, err := diskUsed(dir)
		if err != nil {
			return err
		}
		ok := ping()
		if flagJSON {
			return emit(map[string]any{"ok": ok, "cache_dir": dir, "bytes": used, "reachable": ok, "host": "https://canadabuys.canada.ca/", "datasets": rs})
		}
		fmt.Printf("cache: %s\ndisk: %d bytes\nreachable: %t\n", dir, used, ok)
		for _, r := range rs {
			if r.Cached {
				fmt.Printf("%s  cached  %s  %d\n", r.ID, r.Age, r.Bytes)
			} else {
				fmt.Printf("%s  missing\n", r.ID)
			}
		}
		return nil
	}}
}

// cacheDir: --cache-dir, then CANADABUYS_CACHE_DIR, then XDG, then ~/.cache.
func cacheDir() (string, error) {
	if flagCache != "" {
		return flagCache, nil
	}
	if e := os.Getenv("CANADABUYS_CACHE_DIR"); e != "" {
		return e, nil
	}
	if e := os.Getenv("XDG_CACHE_HOME"); e != "" {
		return filepath.Join(e, "canadabuys"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "canadabuys"), nil
}

func filterKind(kind string) ([]dataset, error) {
	if kind == "" {
		return catalog, nil
	}
	var out []dataset
	for _, d := range catalog {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("unknown kind %q (tenders, awards, contracts, reference)", kind)
	}
	return out, nil
}

func lookup(id string) (dataset, error) {
	for _, d := range catalog {
		if d.ID == id {
			return d, nil
		}
	}
	return dataset{}, fmt.Errorf("unknown dataset %q", id)
}

func pick(ids []string, all bool, kind string) ([]dataset, error) {
	if len(ids) > 0 && all {
		return nil, fmt.Errorf("pass dataset ids or --all, not both")
	}
	if len(ids) == 0 && !all && kind == "" {
		return nil, fmt.Errorf("pass dataset ids, --all, or --kind")
	}
	if len(ids) > 0 {
		out := make([]dataset, 0, len(ids))
		for _, id := range ids {
			d, err := lookup(id)
			if err != nil {
				return nil, err
			}
			if kind != "" && d.Kind != kind {
				return nil, fmt.Errorf("%s is kind %s, not %s", id, d.Kind, kind)
			}
			out = append(out, d)
		}
		return out, nil
	}
	return filterKind(kind)
}

func csvPath(dir, id string) string  { return filepath.Join(dir, id+".csv") }
func metaPath(dir, id string) string { return filepath.Join(dir, id+".json") }

func loadMeta(dir, id string) (meta, bool) {
	b, err := os.ReadFile(metaPath(dir, id))
	if err != nil {
		return meta{}, false
	}
	var m meta
	if json.Unmarshal(b, &m) != nil {
		return meta{}, false
	}
	if _, err := os.Stat(csvPath(dir, id)); err != nil {
		return meta{}, false
	}
	return m, true
}

func listRows(dir, kind string) ([]listRow, error) {
	ds, err := filterKind(kind)
	if err != nil {
		return nil, err
	}
	out := make([]listRow, 0, len(ds))
	for _, d := range ds {
		r := listRow{ID: d.ID, Title: d.Title, Kind: d.Kind, URL: d.URL}
		if m, ok := loadMeta(dir, d.ID); ok {
			r.Cached, r.Bytes, r.FetchedAt, r.LastModified, r.SHA256 = true, m.Bytes, m.FetchedAt, m.LastModified, m.SHA256
			if t, err := time.Parse(time.RFC3339, m.FetchedAt); err == nil {
				r.Age = time.Since(t).Truncate(time.Second).String()
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func diskUsed(dir string) (int64, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var n int64
	for _, e := range ents {
		if info, err := e.Info(); err == nil {
			n += info.Size()
		}
	}
	return n, nil
}

func ping() bool {
	c := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodHead, "https://canadabuys.canada.ca/", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "canadabuys-cli")
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// fetchOne streams to a temp file, hashes on the way, then renames into place.
func fetchOne(dir string, d dataset, force bool) (string, meta, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", meta{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.URL, nil)
	if err != nil {
		return "", meta{}, err
	}
	req.Header.Set("User-Agent", "canadabuys-cli")
	if m, ok := loadMeta(dir, d.ID); ok && !force && m.LastModified != "" {
		req.Header.Set("If-Modified-Since", m.LastModified)
	}
	fmt.Fprintf(os.Stderr, "fetching %s\n", d.ID)
	// The Transport's own Dial/TLS/ResponseHeader timeouts bound everything up
	// to here; the idle timer only needs to cover the body-read phase, where a
	// connection can go silent without closing.
	resp, err := client.Do(req)
	if err != nil {
		return "", meta{}, err
	}
	defer resp.Body.Close()
	idle := time.AfterFunc(idleTimeout, cancel)
	defer idle.Stop()
	if resp.StatusCode == http.StatusNotModified {
		m, _ := loadMeta(dir, d.ID)
		return "skipped", m, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", meta{}, fmt.Errorf("%s: HTTP %s", d.ID, resp.Status)
	}
	f, err := os.CreateTemp(dir, d.ID+".csv.*.tmp")
	if err != nil {
		return "", meta{}, err
	}
	tmp := f.Name()
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h, &progress{id: d.ID, onWrite: func() { idle.Reset(idleTimeout) }}), resp.Body)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		if copyErr != nil {
			return "", meta{}, copyErr
		}
		return "", meta{}, closeErr
	}
	// Drop the sidecar first so a crash cannot pair a new CSV with old metadata.
	if err := os.Remove(metaPath(dir, d.ID)); err != nil && !os.IsNotExist(err) {
		os.Remove(tmp)
		return "", meta{}, err
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		os.Remove(tmp)
		return "", meta{}, err
	}
	if err := os.Rename(tmp, csvPath(dir, d.ID)); err != nil {
		os.Remove(tmp)
		return "", meta{}, err
	}
	m := meta{URL: d.URL, FetchedAt: time.Now().UTC().Format(time.RFC3339), LastModified: resp.Header.Get("Last-Modified"), Bytes: n, SHA256: hex.EncodeToString(h.Sum(nil))}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", meta{}, err
	}
	if err := os.WriteFile(metaPath(dir, d.ID), append(b, '\n'), 0o644); err != nil {
		return "", meta{}, err
	}
	return "fetched", m, nil
}

type progress struct {
	id      string
	n, last int64
	onWrite func()
}

func (p *progress) Write(b []byte) (int, error) {
	p.onWrite()
	p.n += int64(len(b))
	if p.n-p.last >= 8<<20 {
		fmt.Fprintf(os.Stderr, "%s: %d MB\n", p.id, p.n>>20)
		p.last = p.n
	}
	return len(b), nil
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
