package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestSpacing(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	delay := 40 * time.Millisecond
	c := &Client{HTTP: srv.Client(), Delay: delay}
	start := time.Now()
	for i := 0; i < 3; i++ {
		req, err := NewRequest(http.MethodGet, srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	elapsed := time.Since(start)
	if n.Load() != 3 {
		t.Fatalf("hits %d", n.Load())
	}
	min := 2 * delay
	if elapsed < min {
		t.Fatalf("elapsed %s, want at least %s between 3 requests", elapsed, min)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Delay: 10 * time.Millisecond}
	req, err := NewRequest(http.MethodGet, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if n.Load() != 2 {
		t.Fatalf("hits %d, want 2", n.Load())
	}
}

func TestRetryResendsBody(t *testing.T) {
	var n atomic.Int32
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		got = append(got, string(body))
		if n.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Delay: 10 * time.Millisecond}
	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if n.Load() != 2 {
		t.Fatalf("hits %d, want 2", n.Load())
	}
	for i, body := range got {
		if body != "payload" {
			t.Fatalf("attempt %d body %q, want %q", i, body, "payload")
		}
	}
}

type recordSendTimes struct {
	mu    sync.Mutex
	sends []time.Time
	base  http.RoundTripper
}

func (r *recordSendTimes) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.sends = append(r.sends, time.Now())
	r.mu.Unlock()
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func (r *recordSendTimes) snapshot() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.sends...)
}

func TestRedirectHopsArePaced(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		if r.URL.Path != "/b" {
			http.Redirect(w, r, "/b", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	delay := 40 * time.Millisecond
	httpClient := srv.Client()
	rec := &recordSendTimes{base: httpClient.Transport}
	httpClient.Transport = rec
	c := &Client{HTTP: httpClient, Delay: delay}
	req, err := NewRequest(http.MethodGet, srv.URL+"/a")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	if n.Load() != 2 {
		t.Fatalf("hops %d, want 2", n.Load())
	}
	// Send times are recorded client-side in the RoundTripper wrapped
	// beneath pacedTransport, so each stamp is taken after the pacing
	// sleep for that hop and before the request hits the wire. Transport
	// latency can only push a server-side stamp later, never an earlier
	// send stamp, so the gaps cannot drift under the delay.
	got := rec.snapshot()
	if len(got) != 2 {
		t.Fatalf("sends %d, want 2", len(got))
	}
	for i := 1; i < len(got); i++ {
		if gap := got[i].Sub(got[i-1]); gap < delay {
			t.Fatalf("gap between hops %d and %d %s, want at least %s", i-1, i, gap, delay)
		}
	}
}

func TestGivesUpAfterMaxAttempts(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Delay: 10 * time.Millisecond}
	req, err := NewRequest(http.MethodGet, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Do(req)
	if err == nil {
		t.Fatal("want error after exhausting retries")
	}
	if n.Load() != maxAttempts {
		t.Fatalf("hits %d, want %d", n.Load(), maxAttempts)
	}
}

func TestDelayFromEnvTooLarge(t *testing.T) {
	maxSeconds := int64(^uint64(0)>>1) / int64(time.Second)
	t.Setenv("MERX_CRAWL_DELAY", strconv.FormatInt(maxSeconds+1, 10))
	d, err := delayFromEnv()
	if err == nil {
		t.Fatal("expected error for oversized MERX_CRAWL_DELAY")
	}
	if d != 0 {
		t.Fatalf("delay %s, want 0", d)
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error %q", err)
	}
}
