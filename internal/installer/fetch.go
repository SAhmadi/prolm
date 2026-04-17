package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prolm/prolm/internal/httputil"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
)

// fetchRetryConfig holds backoff parameters for rate-limit retries (SEC-10).
// Package-level var so tests can override it.
var fetchRetryConfig = httputil.DefaultRetryConfig()

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

// Fetch downloads the resource at rawURL to destPath atomically (SEC-4, SEC-10, SEC-12).
// Uses progress bar via ui.NewDownloadBar. Enforces HTTPS, max tarball size, and retries on 429.
func Fetch(ctx context.Context, rawURL string, destPath string) error {
	// SEC-4: HTTPS only (allow localhost for test servers).
	if !httputil.IsLocalhostURL(rawURL) {
		if err := registry.ValidateHTTPS(rawURL); err != nil {
			return err
		}
	}

	client := &http.Client{Timeout: fetchTimeout}

	resp, err := fetchWithRetry(ctx, client, rawURL)
	if err != nil {
		// Convert httputil exhaustion error to registry.ErrRateLimited for API compatibility.
		var exhausted *httputil.ErrRetriesExhausted
		if errors.As(err, &exhausted) {
			return &registry.ErrRateLimited{RetryAfter: exhausted.RetryAfter}
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			if fallbackURL, ok := fallbackGitHubTagArchiveURL(rawURL); ok {
				resp.Body.Close()
				fallbackResp, fallbackErr := fetchWithRetry(ctx, client, fallbackURL)
				if fallbackErr == nil {
					resp = fallbackResp
				} else {
					return fallbackErr
				}
				defer resp.Body.Close()
				rawURL = fallbackURL
			}
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
		}
	}

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

func fetchWithRetry(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	return httputil.DoWithRetries(ctx, client, fetchRetryConfig, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	})
}

func fallbackGitHubTagArchiveURL(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(u.Hostname(), "github.com") && !httputil.IsLocalhostURL(rawURL) {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) < 6 {
		return "", false
	}
	if !strings.EqualFold(parts[2], "archive") || !strings.EqualFold(parts[3], "refs") || !strings.EqualFold(parts[4], "tags") {
		return "", false
	}
	tag := parts[5]
	if !strings.HasPrefix(tag, "v") || len(tag) == 1 {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(tag), ".tar.gz") {
		return "", false
	}
	parts[5] = strings.TrimPrefix(tag, "v")
	u.Path = "/" + strings.Join(parts, "/")
	return u.String(), true
}
