package extractor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const fetchMaxBodyBytes = 20 << 20 // 20 MiB — generous for the deep tier

// FetchResult holds the raw bytes and content-type of a fetched URL.
type FetchResult struct {
	Content     []byte
	ContentType string
}

// ContentFetcher fetches raw bytes from URLs for the deep processing tier.
// It applies a larger body cap and longer timeout than the skim-tier URLExtractor.
// Safe for concurrent use.
type ContentFetcher struct {
	client *http.Client
}

// NewContentFetcher returns a ContentFetcher with a 30 s timeout and max 5 redirects.
func NewContentFetcher() *ContentFetcher {
	return &ContentFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// Fetch downloads rawURL and returns the body bytes and content-type header.
// A non-nil error means the URL could not be fetched or returned a non-2xx status.
func (f *ContentFetcher) Fetch(ctx context.Context, rawURL string) (FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return FetchResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "SelfSystems/1.0 (content-indexer)")

	resp, err := f.client.Do(req)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FetchResult{}, fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBodyBytes))
	if err != nil {
		return FetchResult{}, fmt.Errorf("read body: %w", err)
	}

	return FetchResult{
		Content:     body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}
