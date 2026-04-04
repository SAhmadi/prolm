package installer

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
)

// Default backoff parameters for rate-limit retries (SEC-10).
// Package-level vars so tests can override them.
// TODO: Phase 2 — extract shared retry logic to internal/httputil/.
var (
	fetchMaxRetries     = 5
	fetchInitialBackoff = 1 * time.Second
	fetchMaxBackoff     = 30 * time.Second
	fetchJitterMax      = 500 * time.Millisecond
)

const (
	fetchTimeout         = 30 * time.Second
	defaultMaxTarball    = 50 << 20 // 50 MB
	maxTarballSizeEnvVar = "PROLM_MAX_TARBALL_SIZE"
)

// maxTarballSize returns the configured maximum tarball size in bytes.
// Defaults to 50 MB, configurable via PROLM_MAX_TARBALL_SIZE env var.
func maxTarballSize() int64 {
	if v := os.Getenv(maxTarballSizeEnvVar); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil && n > 0 {
			return n
		}
		ui.Warn("invalid %s=%q, using default %d bytes", maxTarballSizeEnvVar, v, defaultMaxTarball)
	}
	return defaultMaxTarball
}

// isLocalhostURL checks if a URL points to localhost (for test servers).
func isLocalhostURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// Fetch downloads the resource at rawURL to destPath atomically (SEC-4, SEC-10, SEC-12).
// Uses progress bar via ui.NewDownloadBar. Enforces HTTPS, max tarball size, and retries on 429.
func Fetch(ctx context.Context, rawURL string, destPath string) error {
	// SEC-4: HTTPS only (allow localhost for test servers).
	if !isLocalhostURL(rawURL) {
		if err := registry.ValidateHTTPS(rawURL); err != nil {
			return err
		}
	}

	client := &http.Client{Timeout: fetchTimeout}
	resp, err := doFetchWithRetries(ctx, client, rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	maxSize := maxTarballSize()

	// Check Content-Length if available.
	if resp.ContentLength > maxSize {
		return &ErrExtractionLimit{
			Reason: fmt.Sprintf("tarball size %d bytes exceeds limit %d bytes", resp.ContentLength, maxSize),
		}
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmp) // best-effort cleanup; no-op if renamed successfully
	}()

	// Progress bar wraps the response body.
	bar := ui.NewDownloadBar(resp.ContentLength, filepath.Base(destPath))
	reader := io.TeeReader(resp.Body, bar)

	// Limit read to maxSize+1 to detect oversized responses.
	limited := io.LimitReader(reader, maxSize+1)
	written, err := io.Copy(f, limited)
	if err != nil {
		return fmt.Errorf("downloading tarball: %w", err)
	}
	if written > maxSize {
		return &ErrExtractionLimit{
			Reason: fmt.Sprintf("tarball size exceeds limit %d bytes", maxSize),
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic placement.
	if err := os.Rename(tmp, destPath); err != nil {
		return fmt.Errorf("moving tarball to destination: %w", err)
	}
	return nil
}

// doFetchWithRetries performs an HTTP GET with exponential backoff on 429 responses (SEC-10).
func doFetchWithRetries(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	backoff := fetchInitialBackoff

	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt >= fetchMaxRetries {
				return nil, &registry.ErrRateLimited{}
			}

			wait := backoff
			if ra := parseRetryAfterHeader(resp.Header.Get("Retry-After")); ra > 0 {
				wait = ra
			}
			jitter := time.Duration(rand.Int64N(int64(fetchJitterMax)))
			wait += jitter
			if wait > fetchMaxBackoff {
				wait = fetchMaxBackoff
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			backoff *= 2
			continue
		}

		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}
}

// parseRetryAfterHeader parses a Retry-After header value (seconds only).
func parseRetryAfterHeader(val string) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	secs, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return time.Duration(secs) * time.Second
}
