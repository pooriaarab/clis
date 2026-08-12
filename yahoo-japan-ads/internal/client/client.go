package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yahoo-japan-ads-pp-cli/internal/config"
)

type Client struct {
	Config *config.Config
	HTTP   *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{Config: cfg, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) Request(ctx context.Context, method, path string, body []byte, contentType string, extra map[string]string) ([]byte, error) {
	url := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		url = strings.TrimRight(c.Config.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if c.Config.HasToken() {
		req.Header.Set("Authorization", "Bearer "+c.Config.Token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range extra {
		req.Header.Set(key, value)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned HTTP %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
