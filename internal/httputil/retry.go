package httputil

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig holds backoff parameters for rate-limit retries (SEC-10).
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	JitterMax      time.Duration
}

// DefaultRetryConfig returns the standard retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		JitterMax:      500 * time.Millisecond,
	}
}

// ErrRetriesExhausted is returned after exhausting all retry attempts on 429 responses.
type ErrRetriesExhausted struct {
	MaxRetries int
	RetryAfter time.Duration
}

func (e *ErrRetriesExhausted) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited; all %d retry attempts exhausted; retry after %s", e.MaxRetries, e.RetryAfter)
	}
	return "rate limited; all retry attempts exhausted"
}

// DoWithRetries performs an HTTP request with exponential backoff on 429 responses (SEC-10).
//
// makeReq is called on each attempt to create a fresh request. This allows callers
// to customise headers (e.g. ETag/If-Modified-Since for caching).
//
// Contract:
//   - On HTTP 429: retries with exponential backoff + jitter, respects context.
//   - On 429 exhaustion: returns (nil, *ErrRetriesExhausted).
//   - On any other status (200, 304, 404, etc.): returns (*http.Response, nil) — caller handles.
//   - On transport error: returns (nil, err).
func DoWithRetries(ctx context.Context, client *http.Client, cfg RetryConfig, makeReq func() (*http.Request, error)) (*http.Response, error) {
	backoff := cfg.InitialBackoff

	for attempt := range cfg.MaxRetries + 1 {
		req, err := makeReq()
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// 429 Too Many Requests — retry with backoff.
		resp.Body.Close()

		if attempt == cfg.MaxRetries {
			retryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
			return nil, &ErrRetriesExhausted{MaxRetries: cfg.MaxRetries, RetryAfter: retryAfter}
		}

		wait := backoff
		if ra := ParseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
			wait = ra
		}

		jitter := time.Duration(rand.Int64N(int64(cfg.JitterMax)))
		wait += jitter

		if wait > cfg.MaxBackoff {
			wait = cfg.MaxBackoff
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}

		backoff *= 2
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	// Unreachable — the loop handles all exit conditions — but satisfies the compiler.
	return nil, &ErrRetriesExhausted{MaxRetries: cfg.MaxRetries}
}

// ParseRetryAfter parses a Retry-After header value (RFC 9110 section 10.2.4).
// Supports both integer seconds and HTTP-date format.
// Returns 0 if the value is empty, unparseable, or a date in the past.
func ParseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	// Try integer seconds first.
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	// Try HTTP-date format (e.g. "Wed, 21 Oct 2025 07:28:00 GMT").
	if t, err := http.ParseTime(value); err == nil {
		if delay := time.Until(t); delay > 0 {
			return delay
		}
		return 0
	}
	return 0
}
