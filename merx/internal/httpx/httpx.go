// Package httpx is the shared HTTP client for merx.
//
// Every request uses an honest User-Agent. Bare curl is HTTP 403; this
// client identifies as merx-cli. Requests are spaced by at least 5s
// (robots.txt Crawl-delay: 5). MERX_CRAWL_DELAY raises that interval.
// 429 and 5xx are retried with exponential backoff, then given up.
package httpx

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	UserAgent    = "merx-cli/0.1 (+https://github.com/pooriaarab/clis)"
	DefaultDelay = 5 * time.Second
	maxAttempts  = 3
)

// Client spaces requests and retries 429/5xx. Do not share across tests
// that need different delays; construct one per command process.
type Client struct {
	HTTP  *http.Client
	Delay time.Duration
	mu    sync.Mutex
	last  time.Time
}

func New() (*Client, error) {
	d, err := delayFromEnv()
	if err != nil {
		return nil, err
	}
	c := &Client{Delay: d}
	c.HTTP = &http.Client{Transport: &pacedTransport{
		c: c,
		base: &http.Transport{
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}}
	return c, nil
}

func NewRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}

func delayFromEnv() (time.Duration, error) {
	s := os.Getenv("MERX_CRAWL_DELAY")
	if s == "" {
		return DefaultDelay, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("MERX_CRAWL_DELAY: %w", err)
	}
	if n < 5 {
		return 0, fmt.Errorf("MERX_CRAWL_DELAY must be >= 5 (robots.txt Crawl-delay: 5), got %d", n)
	}
	maxSeconds := int64(^uint64(0)>>1) / int64(time.Second)
	if int64(n) > maxSeconds {
		return 0, fmt.Errorf("MERX_CRAWL_DELAY is too large, got %d", n)
	}
	return time.Duration(n) * time.Second, nil
}

// pacedTransport applies crawl-delay before every hop, including redirects.
// http.Client.Do only sees the first request; RoundTrip sees each hop.
type pacedTransport struct {
	c    *Client
	base http.RoundTripper
}

func (t *pacedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.c.pace(req.Context()); err != nil {
		return nil, err
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func (c *Client) ensurePaced() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.HTTP == nil {
		c.HTTP = &http.Client{Transport: &pacedTransport{c: c}}
		return
	}
	if _, ok := c.HTTP.Transport.(*pacedTransport); ok {
		return
	}
	clone := *c.HTTP
	clone.Transport = &pacedTransport{c: c, base: c.HTTP.Transport}
	c.HTTP = &clone
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)
	c.ensurePaced()
	ctx := req.Context()
	// A request body can only be replayed if GetBody can rebuild it; otherwise
	// the body is consumed by the first attempt and retrying would send it
	// empty, so don't retry at all.
	attempts := maxAttempts
	if req.Body != nil && req.GetBody == nil {
		attempts = 1
	}
	var lastStatus int
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoff := c.Delay * time.Duration(1<<uint(attempt-1))
			if err := sleep(ctx, backoff); err != nil {
				return nil, err
			}
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}
		lastStatus = resp.StatusCode
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return nil, fmt.Errorf("HTTP %d: gave up after %d attempts", lastStatus, attempts)
}

func (c *Client) pace(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.last.IsZero() {
		if wait := c.Delay - time.Since(c.last); wait > 0 {
			if err := sleep(ctx, wait); err != nil {
				return err
			}
		}
	}
	c.last = time.Now()
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
