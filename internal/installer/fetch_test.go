package installer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/httputil"
	"github.com/prolm/prolm/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func overrideFetchBackoff(t *testing.T) {
	t.Helper()
	orig := fetchRetryConfig
	fetchRetryConfig = httputil.RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     30 * time.Millisecond,
		JitterMax:      1 * time.Millisecond,
	}
	t.Cleanup(func() { fetchRetryConfig = orig })
}

func TestFetch_Success(t *testing.T) {
	content := []byte("tarball-content-here")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), srv.URL+"/pack.tar.gz", dest)
	require.NoError(t, err)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	// No .tmp file left behind.
	_, statErr := os.Stat(dest + ".tmp")
	assert.True(t, os.IsNotExist(statErr))
}

func TestFetch_HTTPRejected(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), "http://evil.com/pack.tar.gz", dest)
	var httpsErr *registry.ErrHTTPSRequired
	require.ErrorAs(t, err, &httpsErr)
}

func TestFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until request is cancelled (avoids leaking goroutines on server close).
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(ctx, srv.URL+"/pack.tar.gz", dest)
	require.Error(t, err)
}

func TestFetch_429Backoff_ThenSuccess(t *testing.T) {
	overrideFetchBackoff(t)

	var attempts atomic.Int32
	content := []byte("success-after-retries")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write(content)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), srv.URL+"/pack.tar.gz", dest)
	require.NoError(t, err)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, got)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestFetch_429Exhausted(t *testing.T) {
	overrideFetchBackoff(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), srv.URL+"/pack.tar.gz", dest)
	var rateLimited *registry.ErrRateLimited
	require.ErrorAs(t, err, &rateLimited)
}

func TestFetch_SizeLimit(t *testing.T) {
	// Override max tarball size to a small value.
	t.Setenv(maxTarballSizeEnvVar, "100")

	// Serve more than 100 bytes.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 200))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), srv.URL+"/pack.tar.gz", dest)
	var limit *ErrExtractionLimit
	require.ErrorAs(t, err, &limit)

	// No file at dest after failure.
	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr))
}

func TestFetch_SizeLimitContentLength(t *testing.T) {
	// Override max tarball size to a small value.
	t.Setenv(maxTarballSizeEnvVar, "100")

	// Server declares Content-Length > limit but doesn't send the full body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "999999")
		w.Write([]byte("small"))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(context.Background(), srv.URL+"/pack.tar.gz", dest)
	var limit *ErrExtractionLimit
	require.ErrorAs(t, err, &limit)
}

func TestFetch_AtomicWrite_NoPartialOnFailure(t *testing.T) {
	// Server sends partial data then hangs until context is cancelled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		// Block until client gives up.
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(ctx, srv.URL+"/pack.tar.gz", dest)
	require.Error(t, err)

	// No file at dest (atomic — only appears on success).
	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr))
}

func TestFetch_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("late"))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	dir := t.TempDir()
	dest := filepath.Join(dir, "pack.tar.gz")

	err := Fetch(ctx, srv.URL+"/pack.tar.gz", dest)
	require.Error(t, err)
}

func TestMaxTarballSize_Default(t *testing.T) {
	t.Setenv(maxTarballSizeEnvVar, "")
	assert.Equal(t, int64(defaultMaxTarball), maxTarballSize())
}

func TestMaxTarballSize_Custom(t *testing.T) {
	t.Setenv(maxTarballSizeEnvVar, "1048576")
	assert.Equal(t, int64(1048576), maxTarballSize())
}

func TestMaxTarballSize_Invalid(t *testing.T) {
	t.Setenv(maxTarballSizeEnvVar, "not-a-number")
	assert.Equal(t, int64(defaultMaxTarball), maxTarballSize())
}
