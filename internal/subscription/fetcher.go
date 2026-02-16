package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	pkgerrors "raycoon/pkg/errors"
)

// Fetcher handles HTTP requests for subscriptions with retry logic
type Fetcher struct {
	client       *http.Client
	userAgent    string
	timeout      time.Duration
	maxRetries   int
	retryDelay   time.Duration
}

// FetcherConfig represents fetcher configuration
type FetcherConfig struct {
	UserAgent  string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultFetcherConfig returns default fetcher configuration
func DefaultFetcherConfig() FetcherConfig {
	return FetcherConfig{
		UserAgent:  "Raycoon/1.0",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}
}

// NewFetcher creates a new subscription fetcher
func NewFetcher(config FetcherConfig) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		userAgent:  config.UserAgent,
		timeout:    config.Timeout,
		maxRetries: config.MaxRetries,
		retryDelay: config.RetryDelay,
	}
}

// Fetch fetches subscription content from a URL with retry logic
func (f *Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(f.retryDelay * time.Duration(attempt)):
			}
		}

		content, err := f.doFetch(ctx, url)
		if err == nil {
			return content, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Don't retry on client errors (4xx)
		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
				break
			}
		}
	}

	return nil, &pkgerrors.SubscriptionError{
		URL: url,
		Err: fmt.Errorf("fetch failed after %d attempts: %w", f.maxRetries+1, lastErr),
	}
}

// doFetch performs a single fetch attempt
func (f *Fetcher) doFetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "*/*")

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        url,
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// HTTPError represents an HTTP error
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d %s for %s", e.StatusCode, e.Status, e.URL)
}
