package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_NewCache_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "cache")
	c, err := NewCache(dir)
	require.NoError(t, err)
	require.NotNil(t, c)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCache_PutAndGet(t *testing.T) {
	c, err := NewCache(t.TempDir())
	require.NoError(t, err)

	entry := &cacheEntry{
		URL:        "https://example.com/pack/list",
		ETag:       `"abc123"`,
		StatusCode: 200,
		FetchedAt:  time.Now().Truncate(time.Second),
	}
	body := []byte("<html>test content</html>")

	err = c.Put("https://example.com/pack/list", entry, body)
	require.NoError(t, err)

	gotEntry, gotBody, err := c.Get("https://example.com/pack/list")
	require.NoError(t, err)
	require.NotNil(t, gotEntry)
	assert.Equal(t, entry.URL, gotEntry.URL)
	assert.Equal(t, entry.ETag, gotEntry.ETag)
	assert.Equal(t, entry.StatusCode, gotEntry.StatusCode)
	assert.Equal(t, body, gotBody)
}

func TestCache_Miss(t *testing.T) {
	c, err := NewCache(t.TempDir())
	require.NoError(t, err)

	entry, body, err := c.Get("https://example.com/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Nil(t, body)
}

func TestCache_CorruptedMetadata_TreatedAsMiss(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	require.NoError(t, err)

	// Write corrupted metadata.
	key := cacheKey("https://example.com/test")
	os.WriteFile(filepath.Join(dir, key+".json"), []byte("not json"), 0644)
	os.WriteFile(filepath.Join(dir, key+".body"), []byte("body"), 0644)

	entry, body, err := c.Get("https://example.com/test")
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Nil(t, body)
}

func TestCache_MissingBody_TreatedAsMiss(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	require.NoError(t, err)

	// Write metadata but no body file.
	key := cacheKey("https://example.com/test")
	os.WriteFile(filepath.Join(dir, key+".json"), []byte(`{"url":"https://example.com/test"}`), 0644)

	entry, body, err := c.Get("https://example.com/test")
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Nil(t, body)
}

func TestCache_AtomicWrite_NoTmpFiles(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	require.NoError(t, err)

	entry := &cacheEntry{URL: "https://example.com/test", StatusCode: 200, FetchedAt: time.Now()}
	err = c.Put("https://example.com/test", entry, []byte("body"))
	require.NoError(t, err)

	// Check no .tmp files remain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp", "leftover .tmp file: %s", e.Name())
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c, err := NewCache(t.TempDir())
	require.NoError(t, err)

	// Each goroutine writes to a different key to avoid inter-write races,
	// verifying that concurrent operations don't corrupt each other.
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			url := fmt.Sprintf("https://example.com/test/%d", i)
			entry := &cacheEntry{URL: url, StatusCode: 200, FetchedAt: time.Now()}
			_ = c.Put(url, entry, []byte("body"))
			got, body, err := c.Get(url)
			if err == nil && got != nil {
				assert.Equal(t, []byte("body"), body)
			}
		}(i)
	}
	wg.Wait()

	// Verify all entries are readable after concurrent writes.
	for i := range 10 {
		url := fmt.Sprintf("https://example.com/test/%d", i)
		entry, body, err := c.Get(url)
		require.NoError(t, err)
		require.NotNil(t, entry, "entry %d should exist", i)
		assert.Equal(t, []byte("body"), body)
	}
}

// --- ETag integration test with httptest ---

func TestCache_ETagFlow(t *testing.T) {
	orig := swiRetryConfig
	swiRetryConfig = httputil.RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     30 * time.Millisecond,
		JitterMax:      1 * time.Millisecond,
	}
	t.Cleanup(func() { swiRetryConfig = orig })

	var requestCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("If-None-Match") == `"etag-v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"etag-v1"`)
		w.Write([]byte(packDetailHTML))
	}))
	t.Cleanup(srv.Close)

	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	require.NoError(t, err)

	reg, err := NewSWIRegistry(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithCache(cache),
	)
	require.NoError(t, err)

	// First request: cache miss, gets full response.
	// Use Versions() to test HTTP-level ETag caching without the
	// in-memory pack list cache (PERF-001) short-circuiting the second call.
	versions, err := reg.Versions(context.Background(), "clpfd")
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	assert.Equal(t, 1, requestCount)

	// Second request: sends If-None-Match, gets 304, uses cached body.
	versions, err = reg.Versions(context.Background(), "clpfd")
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	assert.Equal(t, 2, requestCount)
}
