// Package-level implementation note:
//
// The SWI-Prolog pack index (https://www.swi-prolog.org/pack/) does not expose
// a public JSON or stable machine-readable API. The /pack/query endpoint speaks
// Prolog terms (application/x-prolog), which would require a full Prolog term
// parser in Go — a significant Phase 1 scope increase. HTML scraping of the
// /pack/list and /pack/list?p=<name> pages is therefore used for Phase 1.
//
// Consequence: if SWI-Prolog changes their page HTML structure, the parse
// functions (parsePackList, parsePackDetail) will need to be updated. Tests
// use captured HTML fixtures to detect regressions early.
package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/prolm/prolm/internal/httputil"
)

// swiRetryConfig holds backoff parameters for rate-limit retries (SEC-10).
// Package-level var so tests can override it.
var swiRetryConfig = httputil.DefaultRetryConfig()

// defaultTimeout is the per-request HTTP timeout.
const defaultTimeout = 30 * time.Second

// SWIOption configures a SWIRegistry.
type SWIOption func(*SWIRegistry)

// WithBaseURL overrides the default SWI pack index base URL (for testing).
func WithBaseURL(u string) SWIOption {
	return func(r *SWIRegistry) { r.baseURL = u }
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(c *http.Client) SWIOption {
	return func(r *SWIRegistry) { r.client = c }
}

// WithCache enables HTTP response caching.
func WithCache(c *Cache) SWIOption {
	return func(r *SWIRegistry) { r.cache = c }
}

// SWIRegistry queries the SWI-Prolog pack index at swi-prolog.org.
// It implements the Registry interface.
type SWIRegistry struct {
	baseURL string
	client  *http.Client
	cache   *Cache

	// In-memory caches for parsed results (per-session, no TTL needed).
	mu            sync.Mutex
	versionCache  map[string][]PackageVersion // keyed by package name
	packListCache []PackageVersion            // cached /pack/list results
}

// NewSWIRegistry creates a new SWI pack index client.
// The base URL defaults to https://www.swi-prolog.org.
func NewSWIRegistry(opts ...SWIOption) (*SWIRegistry, error) {
	r := &SWIRegistry{
		baseURL: "https://www.swi-prolog.org",
		client:  &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(r)
	}
	// SEC-4: validate base URL is HTTPS.
	// isLocalhostURL bypass exists solely for httptest.Server in unit tests,
	// which always bind to 127.0.0.1. It must never be used with non-test URLs.
	if !httputil.IsLocalhostURL(r.baseURL) {
		if err := ValidateHTTPS(r.baseURL); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Search returns packages whose name contains the query string.
func (r *SWIRegistry) Search(ctx context.Context, query string) ([]PackageVersion, error) {
	all, err := r.cachedPackList(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []PackageVersion
	for _, pv := range all {
		if strings.Contains(strings.ToLower(pv.Name), query) {
			results = append(results, pv)
		}
	}
	return results, nil
}

// Versions returns all known versions of a package, newest first.
func (r *SWIRegistry) Versions(ctx context.Context, name string) ([]PackageVersion, error) {
	if err := validatePackageName(name); err != nil {
		return nil, err
	}

	body, err := r.fetch(ctx, "/pack/list", url.Values{"p": []string{name}})
	if err != nil {
		return nil, fmt.Errorf("fetching pack detail for %q: %w", name, err)
	}
	defer body.Close()

	versions, err := parsePackDetail(body, name)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, &ErrPackageNotFound{Name: name, Registry: "swi-pack-index"}
	}

	// Populate the in-memory cache so subsequent cachedVersions / DownloadURL
	// calls for the same package skip the network entirely (BUG-009).
	r.mu.Lock()
	if r.versionCache == nil {
		r.versionCache = make(map[string][]PackageVersion)
	}
	r.versionCache[name] = versions
	r.mu.Unlock()

	return versions, nil
}

// DownloadURL returns the HTTPS download URL for a specific package version.
func (r *SWIRegistry) DownloadURL(ctx context.Context, name, version string) (string, error) {
	versions, err := r.cachedVersions(ctx, name)
	if err != nil {
		return "", err
	}

	version = strings.TrimPrefix(version, "v")
	for _, pv := range versions {
		if pv.Version == version {
			if pv.URL == "" {
				return "", &ErrInvalidResponse{Reason: fmt.Sprintf("no download URL for %s@%s", name, version)}
			}
			// SEC-4: validate download URL is HTTPS.
			// isLocalhostURL bypass is for httptest-based download URLs in tests only.
			if !httputil.IsLocalhostURL(pv.URL) {
				if err := ValidateHTTPS(pv.URL); err != nil {
					return "", err
				}
			}
			return pv.URL, nil
		}
	}
	return "", &ErrVersionNotFound{Name: name, Version: version}
}

// cachedVersions returns versions for a package, using the in-memory cache
// if available. This avoids re-fetching and re-parsing the pack detail page
// when DownloadURL is called after Versions for the same package (BUG-008).
func (r *SWIRegistry) cachedVersions(ctx context.Context, name string) ([]PackageVersion, error) {
	r.mu.Lock()
	if r.versionCache != nil {
		if cached, ok := r.versionCache[name]; ok {
			r.mu.Unlock()
			return cached, nil
		}
	}
	r.mu.Unlock()

	versions, err := r.Versions(ctx, name)
	if err != nil {
		return nil, err
	}

	// Versions() already wrote to r.versionCache[name] (QUALITY-007).
	return versions, nil
}

// cachedPackList returns the full pack list, using the in-memory cache if
// available. This avoids re-fetching /pack/list on repeated Search calls
// within the same session (PERF-001).
func (r *SWIRegistry) cachedPackList(ctx context.Context) ([]PackageVersion, error) {
	r.mu.Lock()
	if r.packListCache != nil {
		cached := r.packListCache
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	body, err := r.fetch(ctx, "/pack/list", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching pack list: %w", err)
	}
	defer body.Close()

	all, err := parsePackList(body)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.packListCache = all
	r.mu.Unlock()

	return all, nil
}

// maxRegistryResponseBytes is the upper bound for any single registry HTML
// response body (BUG-006 / SEC-12 spirit). A misbehaving or malicious server
// cannot exhaust memory by streaming a huge response.
const maxRegistryResponseBytes = 10 * 1024 * 1024 // 10 MB

// fetch performs an HTTP GET with caching, rate-limit retries, and context support.
func (r *SWIRegistry) fetch(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	// BUG-003: use url.JoinPath so a trailing slash on baseURL and a leading
	// slash on path never produce a double-slash URL.
	u, err := url.Parse(r.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	joined := u.JoinPath(path)
	if query != nil {
		joined.RawQuery = query.Encode()
	}
	fullURL := joined.String()

	// Check cache first.
	var cached *cacheEntry
	var cachedBody []byte
	if r.cache != nil {
		var cacheErr error
		cached, cachedBody, cacheErr = r.cache.Get(fullURL)
		if cacheErr != nil {
			// Cache read error is non-fatal; proceed without cache.
			// BUG-002: reset both cached and cachedBody so a stale cachedBody
			// is never served if the server unexpectedly returns 304.
			cached = nil
			cachedBody = nil
		}
	}

	resp, err := httputil.DoWithRetries(ctx, r.client, swiRetryConfig, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		if cached != nil {
			if cached.ETag != "" {
				req.Header.Set("If-None-Match", cached.ETag)
			}
			if cached.LastModified != "" {
				req.Header.Set("If-Modified-Since", cached.LastModified)
			}
		}
		return req, nil
	})
	if err != nil {
		var exhausted *httputil.ErrRetriesExhausted
		if errors.As(err, &exhausted) {
			return nil, &ErrRateLimited{RetryAfter: exhausted.RetryAfter}
		}
		return nil, err
	}

	// Convert the raw response into a fetchResult.
	var result *fetchResult
	switch resp.StatusCode {
	case http.StatusOK:
		result = &fetchResult{
			body:         resp.Body,
			etag:         resp.Header.Get("ETag"),
			lastModified: resp.Header.Get("Last-Modified"),
		}
	case http.StatusNotModified:
		resp.Body.Close()
		result = &fetchResult{}
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, &ErrInvalidResponse{
			Reason: fmt.Sprintf("HTTP 404 from %s", fullURL),
		}
	default:
		resp.Body.Close()
		return nil, &ErrInvalidResponse{
			Reason: fmt.Sprintf("unexpected HTTP status %d from %s", resp.StatusCode, fullURL),
		}
	}

	// If body is nil, the server returned 304 — use cached body.
	if result.body == nil && cachedBody != nil {
		return io.NopCloser(strings.NewReader(string(cachedBody))), nil
	}
	if result.body == nil {
		return nil, &ErrInvalidResponse{Reason: "304 response without cached body"}
	}

	// BUG-006 / BUG-007: cap the read at maxRegistryResponseBytes to prevent
	// memory exhaustion from a misbehaving or malicious registry server.
	// Applied unconditionally — caching status does not affect security limits.
	limited := io.LimitReader(result.body, maxRegistryResponseBytes+1)
	data, readErr := io.ReadAll(limited)
	result.body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading response body: %w", readErr)
	}
	if int64(len(data)) > maxRegistryResponseBytes {
		return nil, &ErrInvalidResponse{Reason: "registry response too large"}
	}

	// Cache the response if caching is enabled.
	if r.cache != nil {
		entry := &cacheEntry{
			URL:          fullURL,
			ETag:         result.etag,
			LastModified: result.lastModified,
			StatusCode:   200,
			FetchedAt:    time.Now(),
		}
		_ = r.cache.Put(fullURL, entry, data) // best-effort
	}

	return io.NopCloser(strings.NewReader(string(data))), nil
}

// fetchResult holds the response from a successful HTTP request.
type fetchResult struct {
	body         io.ReadCloser
	etag         string
	lastModified string
}
