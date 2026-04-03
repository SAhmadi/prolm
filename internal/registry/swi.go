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
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Default backoff parameters for rate-limit retries (SEC-10).
// Package-level vars so tests can override them.
var (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	jitterMax      = 500 * time.Millisecond
)

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
	if !isLocalhostURL(r.baseURL) {
		if err := ValidateHTTPS(r.baseURL); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Search returns packages whose name contains the query string.
func (r *SWIRegistry) Search(ctx context.Context, query string) ([]PackageVersion, error) {
	body, err := r.fetch(ctx, "/pack/list")
	if err != nil {
		return nil, fmt.Errorf("fetching pack list: %w", err)
	}
	defer body.Close()

	all, err := parsePackList(body)
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

	path := "/pack/list?p=" + url.QueryEscape(name)
	body, err := r.fetch(ctx, path)
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
	return versions, nil
}

// DownloadURL returns the HTTPS download URL for a specific package version.
func (r *SWIRegistry) DownloadURL(ctx context.Context, name, version string) (string, error) {
	versions, err := r.Versions(ctx, name)
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
			if !isLocalhostURL(pv.URL) {
				if err := ValidateHTTPS(pv.URL); err != nil {
					return "", err
				}
			}
			return pv.URL, nil
		}
	}
	return "", &ErrVersionNotFound{Name: name, Version: version}
}

// maxRegistryResponseBytes is the upper bound for any single registry HTML
// response body (BUG-006 / SEC-12 spirit). A misbehaving or malicious server
// cannot exhaust memory by streaming a huge response.
const maxRegistryResponseBytes = 10 * 1024 * 1024 // 10 MB

// fetch performs an HTTP GET with caching, rate-limit retries, and context support.
func (r *SWIRegistry) fetch(ctx context.Context, path string) (io.ReadCloser, error) {
	// BUG-003: use url.JoinPath so a trailing slash on baseURL and a leading
	// slash on path never produce a double-slash URL.
	u, err := url.Parse(r.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	fullURL := u.JoinPath(path).String()

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

	result, err := r.doRequestWithRetries(ctx, fullURL, cached)
	if err != nil {
		return nil, err
	}

	// If body is nil, the server returned 304 — use cached body.
	if result.body == nil && cachedBody != nil {
		return io.NopCloser(strings.NewReader(string(cachedBody))), nil
	}
	if result.body == nil {
		return nil, &ErrInvalidResponse{Reason: "304 response without cached body"}
	}

	// Read and cache the response body if caching is enabled.
	// BUG-006: cap the read at maxRegistryResponseBytes to prevent memory
	// exhaustion from a misbehaving or malicious registry server.
	if r.cache != nil {
		limited := io.LimitReader(result.body, maxRegistryResponseBytes+1)
		data, readErr := io.ReadAll(limited)
		result.body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading response body: %w", readErr)
		}
		if int64(len(data)) > maxRegistryResponseBytes {
			return nil, &ErrInvalidResponse{Reason: "registry response too large"}
		}

		entry := &cacheEntry{
			URL:          fullURL,
			ETag:         result.etag,
			LastModified: result.lastModified,
			StatusCode:   200,
			FetchedAt:    time.Now(),
		}
		_ = r.cache.Put(fullURL, entry, data) // best-effort

		return io.NopCloser(strings.NewReader(string(data))), nil
	}

	return result.body, nil
}

// fetchResult holds the response from a successful HTTP request.
type fetchResult struct {
	body         io.ReadCloser
	etag         string
	lastModified string
}

// doRequestWithRetries executes an HTTP GET with rate-limit backoff (SEC-10).
// Returns a result with nil body on 304 Not Modified.
func (r *SWIRegistry) doRequestWithRetries(ctx context.Context, rawURL string, cached *cacheEntry) (*fetchResult, error) {
	backoff := initialBackoff

	for attempt := range maxRetries + 1 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		// Add conditional headers from cache.
		if cached != nil {
			if cached.ETag != "" {
				req.Header.Set("If-None-Match", cached.ETag)
			}
			if cached.LastModified != "" {
				req.Header.Set("If-Modified-Since", cached.LastModified)
			}
		}

		resp, err := r.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return &fetchResult{
				body:         resp.Body,
				etag:         resp.Header.Get("ETag"),
				lastModified: resp.Header.Get("Last-Modified"),
			}, nil

		case http.StatusNotModified:
			resp.Body.Close()
			return &fetchResult{}, nil

		case http.StatusTooManyRequests:
			resp.Body.Close()
			if attempt == maxRetries {
				retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
				return nil, &ErrRateLimited{RetryAfter: retryAfter}
			}

			// Determine wait duration from Retry-After header or backoff.
			wait := backoff
			if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
				wait = ra
			}

			// Add jitter.
			jitter := time.Duration(rand.Int64N(int64(jitterMax)))
			wait += jitter

			// Cap at maxBackoff.
			if wait > maxBackoff {
				wait = maxBackoff
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}

		case http.StatusNotFound:
			resp.Body.Close()
			return nil, &ErrInvalidResponse{
				Reason: fmt.Sprintf("HTTP 404 from %s", rawURL),
			}

		default:
			resp.Body.Close()
			return nil, &ErrInvalidResponse{
				Reason: fmt.Sprintf("unexpected HTTP status %d from %s", resp.StatusCode, rawURL),
			}
		}
	}

	// Unreachable — the loop handles all exit conditions — but satisfies the compiler.
	return nil, &ErrRateLimited{}
}

// parseRetryAfter parses a Retry-After header value as seconds.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	secs, err := strconv.Atoi(value)
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// --- HTML Parsing ---

// parsePackList parses the SWI pack list page (/pack/list) and extracts
// package names, latest versions, and descriptions from the HTML table.
func parsePackList(r io.Reader) ([]PackageVersion, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, &ErrInvalidResponse{Reason: fmt.Sprintf("HTML parse error: %v", err)}
	}

	var results []PackageVersion

	// Walk the DOM looking for table rows with pack data.
	// The pack list page has a table with columns: Name, Version, Downloads, Rating, Description.
	// Pack names are in <a> tags linking to /pack/list?p=<name>.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if pv, ok := parsePackListRow(n); ok {
				if err := validatePackageName(pv.Name); err != nil {
					// Skip entries with invalid names rather than failing entirely (SEC-9).
					return
				}
				results = append(results, pv)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return results, nil
}

// parsePackListRow attempts to extract a PackageVersion from a <tr> element.
// Returns (pv, true) if the row contains pack data, (zero, false) otherwise.
func parsePackListRow(tr *html.Node) (PackageVersion, bool) {
	var cells []string
	var packName string

	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
			continue
		}
		text := extractText(td)
		cells = append(cells, strings.TrimSpace(text))

		// Look for <a> tag with href containing /pack/list?p= to get pack name.
		if packName == "" {
			packName = findPackLink(td)
		}
	}

	// Need at least name and version columns.
	if packName == "" || len(cells) < 2 {
		return PackageVersion{}, false
	}

	pv := PackageVersion{
		Name:    packName,
		Version: normalizeVersion(cells[1]),
	}
	return pv, true
}

// parsePackDetail parses an individual pack detail page (/pack/list?p=<name>)
// and extracts all versions with their download URLs, checksums, and dependencies.
func parsePackDetail(r io.Reader, name string) ([]PackageVersion, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, &ErrInvalidResponse{Reason: fmt.Sprintf("HTML parse error: %v", err)}
	}

	// Check for "not found" page — look for text indicating the pack is unknown.
	if containsNotFoundText(doc) {
		return nil, &ErrPackageNotFound{Name: name, Registry: "swi-pack-index"}
	}

	var versions []PackageVersion
	tables := findElements(doc, "table")

	for _, table := range tables {
		rows := findElements(table, "tr")
		for _, row := range rows {
			if pv, ok := parseDetailRow(row, name); ok {
				versions = append(versions, pv)
			}
		}
	}

	return versions, nil
}

// parseDetailRow extracts version info from a detail page table row.
// Expects columns like: Version, SHA1/SHA256, #Downloads, URL.
func parseDetailRow(tr *html.Node, name string) (PackageVersion, bool) {
	var cells []string
	var links []string

	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
			continue
		}
		text := strings.TrimSpace(extractText(td))
		cells = append(cells, text)

		// Collect all href links in this cell.
		for _, href := range findHrefs(td) {
			links = append(links, href)
		}
	}

	// Skip header rows and rows without enough data.
	if len(cells) < 2 {
		return PackageVersion{}, false
	}

	// Try to identify version string in first cell.
	version := normalizeVersion(cells[0])
	if version == "" {
		return PackageVersion{}, false
	}

	pv := PackageVersion{
		Name:    name,
		Version: version,
	}

	// Look for SHA1 hash in cells.
	for _, cell := range cells[1:] {
		if validateSHA1(strings.ToLower(cell)) {
			pv.Checksum = "sha1:" + strings.ToLower(cell)
			break
		}
	}

	// Use the first valid download link.
	for _, href := range links {
		if isDownloadURL(href) {
			pv.URL = href
			break
		}
	}

	return pv, true
}

// --- HTML utility functions ---

// extractText recursively extracts all text content from a node.
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

// findPackLink looks for an <a> tag whose href matches /pack/list?p=<name>
// and returns the pack name.
func findPackLink(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				if name := extractPackNameFromHref(attr.Val); name != "" {
					return name
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if name := findPackLink(c); name != "" {
			return name
		}
	}
	return ""
}

// extractPackNameFromHref extracts a pack name from /pack/list?p=<name>.
func extractPackNameFromHref(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/pack/list") {
		return ""
	}
	return u.Query().Get("p")
}

// findHrefs returns all href attribute values from <a> tags under n.
func findHrefs(n *html.Node) []string {
	var hrefs []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					hrefs = append(hrefs, attr.Val)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return hrefs
}

// findElements finds all descendant elements with the given tag name.
func findElements(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			result = append(result, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

// containsNotFoundText checks if the page contains text indicating
// the pack was not found (e.g., "Sorry, I know nothing about...").
// Only SWI-Prolog-specific phrases are matched to avoid false positives
// from pack descriptions or page footers that happen to include the
// common phrase "not found" (BUG-004).
func containsNotFoundText(doc *html.Node) bool {
	text := strings.ToLower(extractText(doc))
	return strings.Contains(text, "no packs match") ||
		strings.Contains(text, "know nothing about")
}

// normalizeVersion strips "v" prefix and validates the string looks like a version.
// Returns empty string if it doesn't look like a version at all.
func normalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return ""
	}
	// Basic check: must start with a digit.
	if s[0] < '0' || s[0] > '9' {
		return ""
	}
	return s
}

// isDownloadURL checks if a URL looks like a download link (archive file).
func isDownloadURL(href string) bool {
	lower := strings.ToLower(href)
	return strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".zip")
}

// isLocalhostURL checks if a URL points to localhost (for test servers).
func isLocalhostURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}
