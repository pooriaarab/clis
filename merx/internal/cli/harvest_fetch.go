package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/html"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
	"merx-cli/internal/session"
)

const privateSearchPath = "/private/supplier/solicitations/search"

var privateStatus = map[string]string{
	"open": "OPEN", "closed": "CLOSED", "awarded": "AWARD", "bid-results": "AWARD",
}

type harvestDeps struct {
	dir, status, since, until string
	slices                    []Slice
	count                     func(Slice) (int, error)
	page                      func(Slice, int) (search.Page, error)
	detail                    func(string) error
	resume                    bool
}

func executePlan(d harvestDeps) (Summary, error) {
	sum := Summary{Status: d.status}
	store, manifest := harvestPaths(d.dir, d.status)
	if !d.resume {
		os.Remove(store)
		os.Remove(manifest)
	}
	known, err := loadIDs(store)
	if err != nil {
		return sum, err
	}
	man, err := loadManifest(manifest)
	if err != nil {
		return sum, err
	}
	if man.Since == "" {
		man.Since = d.since
	}
	if man.Until == "" {
		man.Until = d.until
	}
	done := map[string]bool{}
	for _, e := range man.Slices {
		done[e.Start+"\x00"+e.End] = true
	}
	leaves, totals, err := plan(d.slices, d.count, done)
	if err != nil {
		return sum, err
	}
	if err := os.MkdirAll(filepath.Dir(store), 0o755); err != nil {
		return sum, err
	}
	f, err := os.OpenFile(store, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return sum, err
	}
	defer f.Close()
	// Append the record, then checkpoint. Reverse order loses rows on crash.
	for _, leaf := range leaves {
		rep := totals[key(leaf)]
		pages := (rep + search.PageSize - 1) / search.PageSize
		if pages > search.MaxPage {
			pages = search.MaxPage
		}
		got := 0
		failed := false
		for p := 1; p <= pages; p++ {
			pg, err := d.page(leaf, p)
			if err != nil {
				return sum, err
			}
			for _, r := range pg.Records {
				if known[r.InternalID] {
					continue
				}
				if err := d.detail(r.DetailURL); err != nil {
					fmt.Fprintf(os.Stderr, "detail %s: %v\n", r.InternalID, err)
					failed = true
					continue
				}
				if err := appendRecord(f, r); err != nil {
					return sum, err
				}
				known[r.InternalID] = true
				got++
			}
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s page %d/%d\n", d.status, leaf.Start, leaf.End, p, pages)
		}
		e := Entry{Status: d.status, Start: leaf.Start, End: leaf.End, Reported: rep, Captured: got,
			CompletedAt: time.Now().UTC().Format(time.RFC3339), Incomplete: incomplete(leaf, rep)}
		sum.Slices++
		sum.Reported += rep
		sum.Captured += got
		if e.Incomplete {
			sum.Incomplete++
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s incomplete: reported %d, captured %d\n", d.status, leaf.Start, leaf.End, rep, got)
		}
		if failed {
			// A detail fetch failed for at least one record: leave the leaf
			// out of the manifest so a later --resume retries the records
			// that were not stored, instead of marking the slice done with
			// a permanent gap.
			fmt.Fprintf(os.Stderr, "harvest %s %s..%s not recorded: a detail fetch failed, rerun with --resume to retry\n", d.status, leaf.Start, leaf.End)
			continue
		}
		man.Slices = append(man.Slices, e)
		if err := saveManifest(manifest, man); err != nil {
			return sum, err
		}
	}
	return sum, nil
}

func loadSearchForm(c *httpx.Client, portal string) (url.Values, error) {
	req, err := httpx.NewRequest(http.MethodGet, strings.TrimRight(portal, "/")+privateSearchPath)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search page: got HTTP %d", resp.StatusCode)
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	_, fields, ok := session.FindForm(doc, "_csrf", "searchAction")
	if !ok || fields["_csrf"] == "" {
		return nil, fmt.Errorf("search page has no _csrf field")
	}
	v := make(url.Values, len(fields))
	for k, val := range fields {
		v.Set(k, val)
	}
	return v, nil
}

func postSlice(c *httpx.Client, portal string, form url.Values, priv string, s Slice, page int) (search.Page, error) {
	v := make(url.Values, len(form))
	for k, vs := range form {
		v[k] = append([]string(nil), vs...)
	}
	v.Set("status", priv)
	v.Set("publishedDate.dateType", "RANGE")
	v.Set("publishedDate.localRangeStart", s.Start)
	v.Set("publishedDate.localRangeEnd", s.End)
	v.Set("pageNumber", strconv.Itoa(page))
	v.Set("pageSize", strconv.Itoa(search.PageSize))
	enc := v.Encode()
	req, err := httpx.NewRequest(http.MethodPost, strings.TrimRight(portal, "/")+privateSearchPath)
	if err != nil {
		return search.Page{}, err
	}
	req.ContentLength = int64(len(enc))
	req.Body = io.NopCloser(strings.NewReader(enc))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(enc)), nil }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		return search.Page{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return search.Page{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return search.Page{}, fmt.Errorf("slice %s..%s page %d: got HTTP %d", s.Start, s.End, page, resp.StatusCode)
	}
	return search.Parse(strings.NewReader(string(body)))
}

func fetchDetail(c *httpx.Client, detailURL string) error {
	if detailURL == "" {
		return fmt.Errorf("empty detail URL")
	}
	req, err := httpx.NewRequest(http.MethodGet, detailURL)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, copyErr := io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", detailURL, resp.StatusCode)
	}
	return copyErr
}

var exitFunc = os.Exit

// logoutOnce runs Logout at most once so the signal handler and runHarvest's
// deferred cleanup cannot both fire it for the same session.
func logoutOnce(once *sync.Once, c *httpx.Client, portal string) {
	once.Do(func() {
		if err := session.Logout(c, portal); err != nil {
			fmt.Fprintln(os.Stderr, "logout:", err)
		}
	})
}

// installLogout logs out on SIGINT/SIGTERM. Caller must stop() on success or a
// later signal still overrides the exit code. #96 is unverified.
func installLogout(once *sync.Once, c *httpx.Client, portal string) func() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		fmt.Fprintln(os.Stderr, "interrupted: logging out")
		logoutOnce(once, c, portal)
		exitFunc(1)
	}()
	return func() { signal.Stop(ch) }
}

func runHarvest(status, since, until string, resume bool, seeds []Slice) error {
	priv := privateStatus[status]
	j, c, jarPath, err := openSession()
	if err != nil {
		return err
	}
	valid := false
	if _, err := os.Stat(jarPath); err == nil {
		valid, _ = session.Valid(c, session.Production.Portal)
	}
	if !valid {
		user, pass := os.Getenv("MERX_USERNAME"), os.Getenv("MERX_PASSWORD")
		if user == "" || pass == "" {
			return fmt.Errorf("missing credentials: export MERX_USERNAME and MERX_PASSWORD")
		}
		if err := session.Login(c, session.Production, user, pass); err != nil {
			return err
		}
	}
	// A lost jar cannot be logged out. Write it before the first fetch.
	if err := j.Save(); err != nil {
		return err
	}
	portal := session.Production.Portal
	var once sync.Once
	stop := installLogout(&once, c, portal)
	defer func() {
		stop()
		logoutOnce(&once, c, portal)
		os.Remove(jarPath)
	}()
	form, err := loadSearchForm(c, portal)
	if err != nil {
		return err
	}
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	sum, err := executePlan(harvestDeps{dir: dir, status: status, since: since, until: until, slices: seeds, resume: resume,
		count:  func(s Slice) (int, error) { pg, err := postSlice(c, portal, form, priv, s, 1); return pg.Total, err },
		page:   func(s Slice, p int) (search.Page, error) { return postSlice(c, portal, form, priv, s, p) },
		detail: func(u string) error { return fetchDetail(c, u) }})
	if err != nil {
		return err
	}
	if flagJSON {
		return emit(sum)
	}
	fmt.Printf("%s: %d slices, reported %d, captured %d, incomplete %d\n", sum.Status, sum.Slices, sum.Reported, sum.Captured, sum.Incomplete)
	return nil
}
