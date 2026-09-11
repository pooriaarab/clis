// Package llm enriches a deterministic shortlist with product-fit notes.
package llm

import (
	"bytes"
	"canadabuys-cli/internal/score"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Enrichment struct {
	Reference   string  `json:"reference"`
	Deliverable float64 `json:"deliverable"`
	ProductFit  float64 `json:"productFit"`
	Shape       string  `json:"shape"`
	Thesis      string  `json:"thesis"`
	Category    string  `json:"category"`
}
type Provider interface {
	Enrich(notices []score.Opportunity, model string) (enr []*Enrichment, skipped int, err error)
}

// Client enriches notices through an OpenAI-compatible chat API.
// Concurrency bounds how many batch requests fly at once. Zero means the
// default of 8. One is the serial case: there is no separate serial path.
type Client struct {
	BaseURL, Key, CacheDir string
	Concurrency            int
}

// enrichBatchSize stays at ten. A malformed reply costs the whole batch,
// so a bigger batch loses more.
const enrichBatchSize = 10

const defaultConcurrency = 8

// maxBatchAttempts bounds retries of one batch. A rate-limited reply is
// retried, never counted as a skip: conflating the two would silently drop
// notices and report success.
const maxBatchAttempts = 5

// retryBaseDelay is the base of the exponential backoff between batch
// retries (doubled per attempt, plus jitter, capped at 30s). A variable
// so tests can shrink it.
var retryBaseDelay = time.Second

func (c *Client) concurrency() int {
	if c.Concurrency <= 0 {
		return defaultConcurrency
	}
	return c.Concurrency
}

// instructionVersion is hashed into the cache key so a new prompt cannot reuse old entries.
const instructionVersion = "product-fit-v2"

const instruction = `Rate each procurement notice for PRODUCT FIT, not how easily a small team could fulfil the contract.

A product is something you build once and sell many times. Name exactly one shape:
- product   a platform, SaaS, or software system the team would own and resell
- resale    the deliverable is existing third-party licences, datasets, or hardware you would buy and pass through
- staffing  consulting or developer time billed by the day
- training  course or training delivery

A notice asking for a platform, portal, or software system is product, even if vendors already sell something similar. Resale is only pass-through of someone else's licences or data.

Examples:
- "Exceed TurboX Premium Licences and Maintenance for SSC" → resale
- "Request for Qualifications - Web Development Consultants" → staffing
- "Online ArcGIS training" → training
- "Specialized Investigative Management Software" → product
- "User Testing and Research Platform" → product
- "Board Meeting Portal Management Software" → product

Split the rating. Do not collapse them into one number:
- deliverable  0-100: could a small team deliver this contract
- productFit   0-100: is this a repeatable product

Hard rule: if shape is resale, staffing, or training, productFit MUST be 0-25. Only shape "product" may score above 40.

Reply with one JSON object {"results":[{"reference":..., "deliverable":0-100, "productFit":0-100, "shape":"product|resale|staffing|training", "thesis":one product line, "category":closest commercial category}]} and nothing else.`

func (c *Client) Enrich(notices []score.Opportunity, m string) ([]*Enrichment, int, error) {
	if c.Key == "" {
		c.Key = os.Getenv("CANADABUYS_LLM_API_KEY")
	}
	if c.Key == "" {
		c.Key = os.Getenv("CEREBRAS_API_KEY")
	}
	if c.Key == "" {
		return nil, 0, fmt.Errorf("set CANADABUYS_LLM_API_KEY or CEREBRAS_API_KEY to use --llm; the deterministic path needs no key")
	}
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return nil, 0, err
	}
	out := make([]*Enrichment, len(notices))
	var pending []int
	for i, b := range notices {
		if e, ok := c.cached(b, m); ok {
			e := e
			clampShape(&e)
			out[i] = &e
		} else {
			pending = append(pending, i)
		}
	}
	// Partition the pending notices into disjoint batches up front, so two
	// workers never fetch the same notice.
	var batches [][]int
	for i := 0; i < len(pending); i += enrichBatchSize {
		batches = append(batches, pending[i:min(i+enrichBatchSize, len(pending))])
	}
	// A fixed pool of workers drains the batch queue. The pool size is the
	// bound: a batch never spawns its own goroutine, so 10,000 notices
	// mean 10,000 queue entries, not 10,000 requests in flight.
	workers := min(c.concurrency(), max(len(batches), 1))
	jobs := make(chan []int)
	var skipped atomic.Int64
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				batch := make([]score.Opportunity, 0, len(idx))
				for _, j := range idx {
					batch = append(batch, notices[j])
				}
				got, n, err := c.batch(batch, m)
				skipped.Add(int64(n))
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("llm: batch starting at %s: %w", notices[idx[0]].Reference, err)
					}
					mu.Unlock()
					continue
				}
				for _, j := range idx {
					e, ok := got[notices[j].Reference]
					// validReply runs on every reply, including this
					// concurrent path. A consulting/1000 answer is a
					// skip, not a clamp, and is not cached.
					if !ok || e.Thesis == "" || !validReply(e) {
						skipped.Add(1)
						continue
					}
					clampShape(&e)
					if werr := c.store(notices[j], m, e); werr != nil {
						fmt.Fprintf(os.Stderr, "llm: cache write failed for %s: %v\n", notices[j].Reference, werr)
					}
					// Copy onto the heap so the pointer outlives the
					// loop. Results land by index, so completion order
					// never leaks into printed order.
					held := e
					out[j] = &held
				}
			}
		}()
	}
	for _, idx := range batches {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()
	// Complete and report: every batch runs even when a sibling fails, so
	// one bad batch neither abandons the others silently nor loses their
	// cache writes. The first error is still returned, so the run fails
	// loudly instead of reporting partial success as success.
	return out, int(skipped.Load()), firstErr
}

// store writes one cache entry atomically: temp file plus rename, the way
// the dataset fetcher does, so concurrent workers cannot corrupt a file.
func (c *Client) store(b score.Opportunity, m string, e Enrichment) error {
	raw, _ := json.Marshal(e)
	raw = append(raw, '\n')
	f, err := os.CreateTemp(c.CacheDir, ".llm-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(raw); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(c.CacheDir, c.key(b, m))); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// validReply accepts only a known shape and scores in [0, 100].
func validReply(e Enrichment) bool {
	s := strings.ToLower(strings.TrimSpace(e.Shape))
	ok := s == "product" || s == "resale" || s == "staffing" || s == "training"
	return ok && e.ProductFit >= 0 && e.ProductFit <= 100 && e.Deliverable >= 0 && e.Deliverable <= 100
}

// clampShape caps a named non-product shape so it cannot outrank a product.
func clampShape(e *Enrichment) {
	e.Shape = strings.ToLower(strings.TrimSpace(e.Shape))
	switch e.Shape {
	case "resale", "staffing", "training":
		if e.ProductFit > 25 {
			e.ProductFit = 25
		}
	}
}

func promptFor(b score.Opportunity) string {
	return fmt.Sprintf("reference: %s\nbuyer: %s\ntitle: %s\ndescription: %s", b.Reference, b.Buyer, b.Title, b.Description[:min(1500, len(b.Description))])
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// safeRefComponent strips path separators and other filesystem-meaningful
// characters from a reference sourced from external tender data, so it
// cannot be used to escape CacheDir when embedded in a cache filename.
func safeRefComponent(ref string) string {
	s := unsafeFilenameChars.ReplaceAllString(ref, "_")
	if s == "" {
		return "_"
	}
	return s
}

func (c *Client) key(b score.Opportunity, m string) string {
	h := sha256.Sum256([]byte(instructionVersion + "\x00" + m + "\x00" + instruction + "\x00" + promptFor(b)))
	return "llm-" + safeRefComponent(b.Reference) + "-" + hex.EncodeToString(h[:])[:16] + ".json"
}
func (c *Client) cached(b score.Opportunity, m string) (Enrichment, bool) {
	var e Enrichment
	raw, err := os.ReadFile(filepath.Join(c.CacheDir, c.key(b, m)))
	if err != nil || json.Unmarshal(raw, &e) != nil || e.Reference != b.Reference || !validReply(e) {
		return Enrichment{}, false
	}
	return e, true
}
func (c *Client) batch(batch []score.Opportunity, m string) (map[string]Enrichment, int, error) {
	var lastErr error
	for attempt := 0; attempt < maxBatchAttempts; attempt++ {
		got, n, retry, err := c.batchOnce(batch, m)
		if err == nil {
			return got, n, nil
		}
		if !retry {
			return nil, 0, err
		}
		lastErr = err
		time.Sleep(backoff(attempt))
	}
	return nil, 0, fmt.Errorf("llm provider: giving up after %d attempts: %w", maxBatchAttempts, lastErr)
}

// backoff is exponential in the attempt number with full jitter, capped
// at 30 seconds so a long rate-limit storm still makes progress.
func backoff(attempt int) time.Duration {
	d := retryBaseDelay << attempt
	if d > 30*time.Second || d <= 0 {
		d = 30 * time.Second
	}
	return d/2 + time.Duration(rand.Int63n(int64(d)/2+1))
}

// retryableStatus reports whether a failed request is worth repeating.
// 429 and 5xx are transient under concurrency; any other 4xx is a real
// rejection and fails immediately.
func retryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code/100 == 5
}

func (c *Client) batchOnce(batch []score.Opportunity, m string) (map[string]Enrichment, int, bool, error) {
	var sb strings.Builder
	sb.WriteString(instruction)
	for _, b := range batch {
		sb.WriteString("\n---\n" + promptFor(b))
	}
	body, _ := json.Marshal(map[string]any{"model": m,
		"messages":        []map[string]string{{"role": "system", "content": "Reply with structured JSON only."}, {"role": "user", "content": sb.String()}},
		"response_format": map[string]string{"type": "json_object"}})
	base := "https://api.cerebras.ai/v1"
	if c.BaseURL != "" {
		base = c.BaseURL
	} else if e := os.Getenv("CANADABUYS_LLM_BASE_URL"); e != "" {
		base = e
	}
	req, err := http.NewRequest("POST", strings.TrimSuffix(base, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, 0, false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, true, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, true, fmt.Errorf("llm provider: reading response body: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		err := fmt.Errorf("llm provider: HTTP %s: %s", resp.Status, strings.TrimSpace(string(raw)))
		return nil, 0, retryableStatus(resp.StatusCode), err
	}
	var env struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &env) != nil || len(env.Choices) == 0 {
		return nil, 0, false, nil
	}
	var doc struct {
		Results []Enrichment `json:"results"`
	}
	if json.Unmarshal([]byte(env.Choices[0].Message.Content), &doc) != nil {
		return nil, 0, false, nil
	}
	byRef := map[string]Enrichment{}
	for _, e := range doc.Results {
		byRef[e.Reference] = e
	}
	return byRef, 0, false, nil
}
