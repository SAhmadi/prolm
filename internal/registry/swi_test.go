package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// --- Sample HTML fixtures ---

const packListHTML = `<!DOCTYPE html>
<html><body>
<table class="packlist">
<tr><th>Name</th><th>Version</th><th>Downloads</th><th>Rating</th><th>Description</th></tr>
<tr>
  <td><a href="/pack/list?p=clpfd">clpfd</a></td>
  <td>1.4.3</td><td>1200</td><td>4</td><td>Constraint logic programming</td>
</tr>
<tr>
  <td><a href="/pack/list?p=http">http</a></td>
  <td>7.1.2</td><td>5000</td><td>5</td><td>HTTP client and server</td>
</tr>
<tr>
  <td><a href="/pack/list?p=prosqlite">prosqlite</a></td>
  <td>0.9.11</td><td>300</td><td>3</td><td>SQLite interface</td>
</tr>
</table>
</body></html>`

const packListEmptyHTML = `<!DOCTYPE html>
<html><body>
<table class="packlist">
<tr><th>Name</th><th>Version</th><th>Downloads</th></tr>
</table>
</body></html>`

const packDetailHTML = `<!DOCTYPE html>
<html><body>
<h2>clpfd</h2>
<p>Constraint logic programming over finite domains</p>
<table>
<tr><th>Version</th><th>SHA1</th><th>#Downloads</th><th>URL</th></tr>
<tr>
  <td>1.4.3</td>
  <td>a3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3</td>
  <td>500</td>
  <td><a href="https://github.com/example/clpfd/archive/v1.4.3.tar.gz">download</a></td>
</tr>
<tr>
  <td>1.4.2</td>
  <td>b4e3d2c1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5</td>
  <td>400</td>
  <td><a href="https://github.com/example/clpfd/archive/v1.4.2.tar.gz">download</a></td>
</tr>
<tr>
  <td>1.3.0</td>
  <td>c5f4e3d2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6</td>
  <td>200</td>
  <td><a href="https://github.com/example/clpfd/archive/v1.3.0.tar.gz">download</a></td>
</tr>
</table>
</body></html>`

const prosqliteDetailHTML = `<!DOCTYPE html>
<html><body>
<h2>prosqlite</h2>
<table>
<tr><th>Version</th><th>SHA1</th><th>#Downloads</th><th>URL</th></tr>
<tr>
  <td>0.9.11</td>
  <td>a3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3</td>
  <td>300</td>
  <td><a href="https://github.com/example/prosqlite/archive/v0.9.11.tar.gz">download</a></td>
</tr>
</table>
</body></html>`

const aopGitDetailHTML = `<!DOCTYPE html>
<html><body>
<h2>aop</h2>
<table>
<tr><th>Version</th><th>SHA1</th><th>#Downloads</th><th>URL</th></tr>
<tr>
  <td>0.0.9</td>
  <td>f8cce8405c474b64e569a58634bf5dfd0b2c9ccc</td>
  <td>21</td>
  <td><a href="https://github.com/hargettp/aop.git">download</a></td>
</tr>
</table>
</body></html>`

const packNotFoundHTML = `<!DOCTYPE html>
<html><body>
<h2>Pack not found</h2>
<p>Sorry, I know nothing about pack "nonexistent".</p>
</body></html>`

// --- parsePackList tests ---

func TestParsePackList_Valid(t *testing.T) {
	results, err := parsePackList(strings.NewReader(packListHTML))
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, "clpfd", results[0].Name)
	assert.Equal(t, "1.4.3", results[0].Version)

	assert.Equal(t, "http", results[1].Name)
	assert.Equal(t, "7.1.2", results[1].Version)

	assert.Equal(t, "prosqlite", results[2].Name)
	assert.Equal(t, "0.9.11", results[2].Version)
}

func TestParsePackList_Empty(t *testing.T) {
	results, err := parsePackList(strings.NewReader(packListEmptyHTML))
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParsePackList_InvalidHTML(t *testing.T) {
	// html.Parse is very lenient — even broken HTML parses without error.
	// The result should be an empty list, not an error.
	results, err := parsePackList(strings.NewReader("<not>valid<html"))
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParsePackList_SkipsInvalidNames(t *testing.T) {
	html := `<table><tr>
		<td><a href="/pack/list?p=INVALID_UPPER">INVALID_UPPER</a></td>
		<td>1.0.0</td>
	</tr></table>`
	results, err := parsePackList(strings.NewReader(html))
	require.NoError(t, err)
	assert.Empty(t, results, "names with uppercase should be skipped")
}

// --- parsePackDetail tests ---

func TestParsePackDetail_Valid(t *testing.T) {
	versions, err := parsePackDetail(strings.NewReader(packDetailHTML), "clpfd")
	require.NoError(t, err)
	require.Len(t, versions, 3)

	assert.Equal(t, "clpfd", versions[0].Name)
	assert.Equal(t, "1.4.3", versions[0].Version)
	assert.Equal(t, "sha1:a3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3", versions[0].Checksum)
	assert.Equal(t, "https://github.com/example/clpfd/archive/v1.4.3.tar.gz", versions[0].URL)

	assert.Equal(t, "1.4.2", versions[1].Version)
	assert.Equal(t, "1.3.0", versions[2].Version)
}

func TestParsePackDetail_NotFound(t *testing.T) {
	_, err := parsePackDetail(strings.NewReader(packNotFoundHTML), "nonexistent")
	require.Error(t, err)

	var notFound *ErrPackageNotFound
	assert.True(t, errors.As(err, &notFound))
	assert.Equal(t, "nonexistent", notFound.Name)
}

func TestParsePackDetail_EmptyVersionTable(t *testing.T) {
	html := `<html><body><h2>mypkg</h2><table>
		<tr><th>Version</th><th>SHA1</th></tr>
	</table></body></html>`
	versions, err := parsePackDetail(strings.NewReader(html), "mypkg")
	// No versions found — returns ErrPackageNotFound (handled by Versions method).
	// The parse function itself returns empty slice.
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestParsePackDetail_GitRepoURL_DropsSWIChecksum(t *testing.T) {
	versions, err := parsePackDetail(strings.NewReader(aopGitDetailHTML), "aop")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "0.0.9", versions[0].Version)
	assert.Equal(t, "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz", versions[0].URL)
	assert.Empty(t, versions[0].Checksum)
	assert.Contains(t, versions[0].ChecksumWarning, "could not be cross-verified")
}

// --- SWIRegistry method tests (httptest.Server) ---

func newTestServer(t *testing.T, handler http.HandlerFunc) (*SWIRegistry, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	reg, err := NewSWIRegistry(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	require.NoError(t, err)
	return reg, srv
}

func TestSearch_Found(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(packListHTML))
	})

	results, err := reg.Search(context.Background(), "clp")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "clpfd", results[0].Name)
}

func TestSearch_NotFound(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packListHTML))
	})

	results, err := reg.Search(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packListHTML))
	})

	results, err := reg.Search(context.Background(), "HTTP")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "http", results[0].Name)
}

func TestVersions_Found(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	versions, err := reg.Versions(context.Background(), "clpfd")
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, "1.4.3", versions[0].Version)
}

// BUG-018: Versions must request /pack/list with ?p=<name> as a real query
// parameter. Encoding '?p=' into the path causes SWI to return not-found pages.
func TestVersions_UsesPackQueryParameter(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pack/list", r.URL.Path)
		assert.Equal(t, "prosqlite", r.URL.Query().Get("p"))
		w.Write([]byte(prosqliteDetailHTML))
	})

	versions, err := reg.Versions(context.Background(), "prosqlite")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "prosqlite", versions[0].Name)
	assert.Equal(t, "0.9.11", versions[0].Version)
}

func TestVersions_PackNotFound(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packNotFoundHTML))
	})

	_, err := reg.Versions(context.Background(), "nonexistent")
	require.Error(t, err)

	var notFound *ErrPackageNotFound
	assert.True(t, errors.As(err, &notFound))
}

func TestDownloadURL_Found(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	url, err := reg.DownloadURL(context.Background(), "clpfd", "1.4.3")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/example/clpfd/archive/v1.4.3.tar.gz", url)
}

func TestDownloadURL_VersionNotFound(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	_, err := reg.DownloadURL(context.Background(), "clpfd", "9.9.9")
	require.Error(t, err)

	var notFound *ErrVersionNotFound
	assert.True(t, errors.As(err, &notFound))
	assert.Equal(t, "9.9.9", notFound.Version)
}

func TestDownloadURL_StripsVPrefix(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	url, err := reg.DownloadURL(context.Background(), "clpfd", "v1.4.3")
	require.NoError(t, err)
	assert.Contains(t, url, "v1.4.3.tar.gz")
}

func TestDownloadURL_GitRepoURL_ConvertedToTaggedArchive(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(aopGitDetailHTML))
	})

	url, err := reg.DownloadURL(context.Background(), "aop", "0.0.9")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz", url)
}

// --- Security tests ---

func TestHTTPS_BaseURLEnforcement(t *testing.T) {
	_, err := NewSWIRegistry(WithBaseURL("http://insecure.example.com"))
	require.Error(t, err)

	var httpsErr *ErrHTTPSRequired
	assert.True(t, errors.As(err, &httpsErr))
}

func TestHTTPS_DownloadURLEnforcement(t *testing.T) {
	html := `<html><body><table>
		<tr><td>1.0.0</td><td>a3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3</td><td>10</td>
		<td><a href="http://insecure.example.com/pkg.tar.gz">download</a></td></tr>
	</table></body></html>`

	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	})

	_, err := reg.DownloadURL(context.Background(), "testpkg", "1.0.0")
	require.Error(t, err)

	var httpsErr *ErrHTTPSRequired
	assert.True(t, errors.As(err, &httpsErr))
}

func TestTimeout_Respected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(packListHTML))
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Timeout: 100 * time.Millisecond}
	reg, err := NewSWIRegistry(
		WithBaseURL(srv.URL),
		WithHTTPClient(client),
	)
	require.NoError(t, err)

	_, err = reg.Search(context.Background(), "test")
	require.Error(t, err)
}

func TestInputValidation_InvalidPackageName(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	_, err := reg.Versions(context.Background(), "INVALID")
	require.Error(t, err)

	var invalid *ErrInvalidResponse
	assert.True(t, errors.As(err, &invalid))
}

func TestInputValidation_NullByteInName(t *testing.T) {
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(packDetailHTML))
	})

	_, err := reg.Versions(context.Background(), "test\x00pkg")
	require.Error(t, err)

	var invalid *ErrInvalidResponse
	assert.True(t, errors.As(err, &invalid))
}

// --- Rate limiting tests ---

func TestRateLimit_429ThenSuccess(t *testing.T) {
	// Override backoff for fast tests.
	orig := swiRetryConfig
	swiRetryConfig = httputil.RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     30 * time.Millisecond,
		JitterMax:      1 * time.Millisecond,
	}
	t.Cleanup(func() { swiRetryConfig = orig })

	var attempts atomic.Int32
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(packListHTML))
	})

	results, err := reg.Search(context.Background(), "http")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestRateLimit_429Exhausted(t *testing.T) {
	orig := swiRetryConfig
	swiRetryConfig = httputil.RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     30 * time.Millisecond,
		JitterMax:      1 * time.Millisecond,
	}
	t.Cleanup(func() { swiRetryConfig = orig })

	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := reg.Search(context.Background(), "test")
	require.Error(t, err)

	var rateLimited *ErrRateLimited
	assert.True(t, errors.As(err, &rateLimited))
}

func TestRateLimit_ContextCancellation(t *testing.T) {
	orig := swiRetryConfig
	swiRetryConfig = httputil.RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     10 * time.Second,
		JitterMax:      1 * time.Millisecond,
	}
	t.Cleanup(func() { swiRetryConfig = orig })

	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := reg.Search(ctx, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context"))
}

// --- Error type tests ---

func TestErrorTypes_Messages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{&ErrPackageNotFound{Name: "foo", Registry: "swi"}, `package "foo" not found in swi registry`},
		{&ErrVersionNotFound{Name: "foo", Version: "1.0"}, `version "1.0" of package "foo" not found`},
		{&ErrHTTPSRequired{URL: "http://x"}, "HTTPS required but got insecure URL: http://x"},
		{&ErrRateLimited{RetryAfter: 5 * time.Second}, "rate limited by registry; retry after 5s"},
		{&ErrRateLimited{}, "rate limited by registry; all retry attempts exhausted"},
		{&ErrInvalidResponse{Reason: "bad"}, "invalid registry response: bad"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.err.Error())
	}
}

func TestErrorTypes_ErrorsAs(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &ErrPackageNotFound{Name: "x", Registry: "swi"})
	var target *ErrPackageNotFound
	assert.True(t, errors.As(err, &target))
	assert.Equal(t, "x", target.Name)
}

// --- Validation helper tests ---

func TestValidateHTTPS(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com/pack", false},
		{"HTTPS://EXAMPLE.COM", false},
		{"http://example.com", true},
		{"ftp://example.com", true},
		{"", true},
		{"://bad", true},
	}
	for _, tt := range tests {
		err := ValidateHTTPS(tt.url)
		if tt.wantErr {
			assert.Error(t, err, "URL: %s", tt.url)
		} else {
			assert.NoError(t, err, "URL: %s", tt.url)
		}
	}
}

func TestValidateSHA1(t *testing.T) {
	assert.True(t, validateSHA1("a3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3"))
	assert.False(t, validateSHA1("tooshort"))
	assert.False(t, validateSHA1("A3F2C1D9E8B7A6F5E4D3C2B1A0F9E8D7C6B5A4F3")) // uppercase
	assert.False(t, validateSHA1(""))
	assert.False(t, validateSHA1("g3f2c1d9e8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3")) // non-hex
}

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"clpfd", false},
		{"my-package", false},
		{"owner/name", false},
		{"", true},
		{"UPPER", true},
		{"has space", true},
		{"null\x00byte", true},
		{strings.Repeat("a", 200), true},
	}
	for _, tt := range tests {
		err := validatePackageName(tt.name)
		if tt.wantErr {
			assert.Error(t, err, "name: %q", tt.name)
		} else {
			assert.NoError(t, err, "name: %q", tt.name)
		}
	}
}

// --- HTML utility tests ---

func TestNormalizeVersion(t *testing.T) {
	assert.Equal(t, "1.4.3", normalizeVersion("1.4.3"))
	assert.Equal(t, "1.4.3", normalizeVersion("v1.4.3"))
	assert.Equal(t, "1.4.3", normalizeVersion("  v1.4.3  "))
	assert.Equal(t, "", normalizeVersion(""))
	assert.Equal(t, "", normalizeVersion("abc"))
	assert.Equal(t, "", normalizeVersion("Version"))
}

func TestIsDownloadURL(t *testing.T) {
	assert.True(t, isDownloadURL("https://example.com/pkg.tar.gz"))
	assert.True(t, isDownloadURL("https://example.com/pkg.tgz"))
	assert.True(t, isDownloadURL("https://example.com/pkg.zip"))
	assert.False(t, isDownloadURL("https://example.com/page.html"))
	assert.False(t, isDownloadURL("https://example.com/"))
}

func TestParseRetryAfter(t *testing.T) {
	// Integer seconds.
	assert.Equal(t, 5*time.Second, httputil.ParseRetryAfter("5"))
	assert.Equal(t, time.Duration(0), httputil.ParseRetryAfter("0"))
	assert.Equal(t, time.Duration(0), httputil.ParseRetryAfter(""))
	assert.Equal(t, time.Duration(0), httputil.ParseRetryAfter("not-a-number"))
	assert.Equal(t, time.Duration(0), httputil.ParseRetryAfter("-1"))

	// HTTP-date in the future (BUG-005).
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	d := httputil.ParseRetryAfter(future)
	assert.Greater(t, d, 20*time.Second, "expected ~30s delay for future HTTP-date")
	assert.Less(t, d, 35*time.Second)

	// HTTP-date in the past returns 0.
	past := time.Now().Add(-10 * time.Second).UTC().Format(http.TimeFormat)
	assert.Equal(t, time.Duration(0), httputil.ParseRetryAfter(past))
}

// --- BUG-008: DownloadURL caches Versions results ---

func TestDownloadURL_CachesVersions(t *testing.T) {
	var fetchCount atomic.Int32
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fetchCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(packDetailHTML))
	})

	// First call fetches from server.
	url1, err := reg.DownloadURL(context.Background(), "clpfd", "1.4.3")
	require.NoError(t, err)
	assert.Contains(t, url1, "1.4.3")

	// Second call for a different version should use cached versions.
	url2, err := reg.DownloadURL(context.Background(), "clpfd", "1.4.2")
	require.NoError(t, err)
	assert.Contains(t, url2, "1.4.2")

	// Only one HTTP request for the detail page.
	assert.Equal(t, int32(1), fetchCount.Load())
}

// --- BUG-009: Versions() populates the in-memory cache ---

func TestVersions_PopulatesCache(t *testing.T) {
	var fetchCount atomic.Int32
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fetchCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(packDetailHTML))
	})

	// Direct Versions() call — fetches from server.
	versions, err := reg.Versions(context.Background(), "clpfd")
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	assert.Equal(t, int32(1), fetchCount.Load())

	// DownloadURL for the same package must use the cache, not re-fetch.
	url1, err := reg.DownloadURL(context.Background(), "clpfd", "1.4.3")
	require.NoError(t, err)
	assert.Contains(t, url1, "1.4.3")

	// Still only one HTTP request.
	assert.Equal(t, int32(1), fetchCount.Load())
}

// --- PERF-001: Search caches pack list ---

func TestSearch_CachesPackList(t *testing.T) {
	var fetchCount atomic.Int32
	reg, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fetchCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(packListHTML))
	})

	// First search fetches from server.
	results1, err := reg.Search(context.Background(), "clp")
	require.NoError(t, err)
	require.NotEmpty(t, results1)

	// Second search with different query uses cached pack list.
	results2, err := reg.Search(context.Background(), "http")
	require.NoError(t, err)
	require.NotEmpty(t, results2)

	// Only one HTTP fetch for /pack/list.
	assert.Equal(t, int32(1), fetchCount.Load())
}

// --- DRY-002: walkDOM helper ---

func TestWalkDOM_ShortCircuit(t *testing.T) {
	doc, err := html.Parse(strings.NewReader("<div><p>a</p><p>b</p></div>"))
	require.NoError(t, err)

	var visited int
	walkDOM(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "p" {
			visited++
			return true // skip children of <p>
		}
		return false
	})
	assert.Equal(t, 2, visited)
}

// --- BUG-007: Response size limit must apply even without cache ---

func TestFetch_ResponseSizeLimit_NoCacheEnabled(t *testing.T) {
	// Serve a response larger than maxRegistryResponseBytes (10 MB).
	// The registry is created WITHOUT WithCache, so the no-cache code path is used.
	oversized := strings.Repeat("x", 10*1024*1024+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(oversized))
	}))
	t.Cleanup(srv.Close)

	reg, err := NewSWIRegistry(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		// Deliberately no WithCache — this is the bug scenario.
	)
	require.NoError(t, err)

	_, err = reg.Search(context.Background(), "anything")
	require.Error(t, err)

	var invalidResp *ErrInvalidResponse
	assert.True(t, errors.As(err, &invalidResp), "expected ErrInvalidResponse, got: %v", err)
}
