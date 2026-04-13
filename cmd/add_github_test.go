package cmd

import (
	"context"
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
