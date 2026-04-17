package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubResolver_PicksLatestStableTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/hargettp/aop/tags":
			_, _ = w.Write([]byte(`[
  {"name":"v0.0.8"},
  {"name":"v0.0.9-beta.1"},
  {"name":"v0.0.9"}
]`))
		case "/repos/hargettp/aop/releases":
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	resolver := &gitHubRepoResolver{
		client:  srv.Client(),
		apiBase: srv.URL,
	}
	got, err := resolver.Resolve(context.Background(), "hargettp/aop")
	require.NoError(t, err)
	assert.Equal(t, "aop", got.Name)
	assert.Equal(t, "0.0.9", got.Version)
	assert.Equal(t, "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz", got.URL)
}

func TestGitHubResolver_FallsBackToReleasesWhenTagsLackSemver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/example/pkg/tags":
			_, _ = w.Write([]byte(`[{"name":"latest"}]`))
		case "/repos/example/pkg/releases":
			_, _ = w.Write([]byte(`[
  {"tag_name":"v1.0.0-rc.1"},
  {"tag_name":"v1.0.0"}
]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	resolver := &gitHubRepoResolver{
		client:  srv.Client(),
		apiBase: srv.URL,
	}
	got, err := resolver.Resolve(context.Background(), "example/pkg")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", got.Version)
	assert.Equal(t, "https://github.com/example/pkg/archive/refs/tags/v1.0.0.tar.gz", got.URL)
}

func TestGitHubResolver_NoStableSemverTagsOrReleasesFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/hargettp/aop/tags":
			_, _ = w.Write([]byte(`[{"name":"latest"},{"name":"dev-preview"}]`))
		case "/repos/hargettp/aop/releases":
			_, _ = w.Write([]byte(`[{"tag_name":"beta"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	resolver := &gitHubRepoResolver{
		client:  srv.Client(),
		apiBase: srv.URL,
	}
	_, err := resolver.Resolve(context.Background(), "hargettp/aop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no stable semver tags/releases")
	assert.Contains(t, err.Error(), "explicit tagged archive URL")
}

func TestFetchGitHubTagNames_SetsHeaders_AndUsesGitHubToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret-token")
	var gotUserAgent string
	var gotAuth string
	var gotAccept string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		_ = json.NewEncoder(w).Encode([]map[string]string{{"name": "v1.2.3"}})
	}))
	t.Cleanup(srv.Close)

	client := srv.Client()
	names, err := fetchGitHubTagNames(context.Background(), client, srv.URL, "tags")
	require.NoError(t, err)
	require.Equal(t, []string{"v1.2.3"}, names)
	assert.Equal(t, gitHubAPIUserAgent, gotUserAgent)
	assert.Equal(t, "Bearer secret-token", gotAuth)
	assert.Equal(t, "application/vnd.github+json", gotAccept)
}

func TestFetchGitHubTagNames_ForbiddenIncludesTokenGuidance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	t.Cleanup(srv.Close)

	_, err := fetchGitHubTagNames(context.Background(), srv.Client(), srv.URL, "tags")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 403")
	assert.Contains(t, err.Error(), "GITHUB_TOKEN")
	assert.Contains(t, err.Error(), "rate limit")
}

func TestFetchGitHubReleaseTagNames_SetsUserAgentWithoutToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	var gotUserAgent string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode([]map[string]string{{"tag_name": "v1.0.0"}})
	}))
	t.Cleanup(srv.Close)

	names, err := fetchGitHubReleaseTagNames(context.Background(), srv.Client(), srv.URL)
	require.NoError(t, err)
	require.Equal(t, []string{"v1.0.0"}, names)
	assert.Equal(t, gitHubAPIUserAgent, gotUserAgent)
	assert.Empty(t, gotAuth)
}
