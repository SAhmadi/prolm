package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fastConfig = RetryConfig{
	MaxRetries:     3,
	InitialBackoff: 1 * time.Millisecond,
	MaxBackoff:     50 * time.Millisecond,
	JitterMax:      1 * time.Millisecond,
}

func TestDoWithRetries_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	resp, err := DoWithRetries(context.Background(), srv.Client(), fastConfig, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoWithRetries_429ThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	resp, err := DoWithRetries(context.Background(), srv.Client(), fastConfig, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDoWithRetries_429Exhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	_, err := DoWithRetries(context.Background(), srv.Client(), fastConfig, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	require.Error(t, err)

	var exhausted *ErrRetriesExhausted
	require.ErrorAs(t, err, &exhausted)
}

func TestDoWithRetries_ContextCancellation(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     10 * time.Second,
		JitterMax:      1 * time.Millisecond,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := DoWithRetries(ctx, srv.Client(), cfg, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	})
	require.Error(t, err)
}

func TestDoWithRetries_Non429Passthrough(t *testing.T) {
	tests := []int{http.StatusNotFound, http.StatusNotModified, http.StatusInternalServerError}
	for _, code := range tests {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			t.Cleanup(srv.Close)

			resp, err := DoWithRetries(context.Background(), srv.Client(), fastConfig, func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, srv.URL, nil)
			})
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, code, resp.StatusCode)
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"empty", "", 0},
		{"integer seconds", "5", 5 * time.Second},
		{"zero", "0", 0},
		{"negative", "-1", 0},
		{"invalid", "abc", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseRetryAfter(tt.value))
		})
	}
}
