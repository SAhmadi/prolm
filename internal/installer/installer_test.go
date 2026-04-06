package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRegistry implements registry.Registry for testing.
type mockRegistry struct {
	versions    map[string][]registry.PackageVersion
	downloadURL map[string]string // key: "name@version"
}

func (m *mockRegistry) Search(_ context.Context, query string) ([]registry.PackageVersion, error) {
	return nil, nil
}

func (m *mockRegistry) Versions(_ context.Context, name string) ([]registry.PackageVersion, error) {
	if vs, ok := m.versions[name]; ok {
		return vs, nil
	}
	return nil, &registry.ErrPackageNotFound{Name: name, Registry: "mock"}
}

func (m *mockRegistry) DownloadURL(_ context.Context, name, version string) (string, error) {
	key := name + "@" + version
	if url, ok := m.downloadURL[key]; ok {
		return url, nil
	}
	return "", &registry.ErrVersionNotFound{Name: name, Version: version}
}

// makeTarball creates a valid .tar.gz in memory with a single .pl file.
func makeTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func setupTestInstall(t *testing.T) (storeDir, cacheDir string) {
	t.Helper()
	dir := t.TempDir()
	storeDir = filepath.Join(dir, "store")
	cacheDir = filepath.Join(dir, "cache")
	return storeDir, cacheDir
}

func TestInstall_FreshInstall(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	tarballContent := makeTarball(t, map[string]string{
		"clpfd/main.pl": ":- module(clpfd, [in/2]).\n",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarballContent)
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"clpfd": {{Name: "clpfd", Version: "1.4.3", URL: srv.URL + "/clpfd-1.4.3.tar.gz"}},
		},
		downloadURL: map[string]string{
			"clpfd@1.4.3": srv.URL + "/clpfd-1.4.3.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"clpfd": "^1.4"},
	}

	lf, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)

	assert.Equal(t, "clpfd", lf.Packages[0].Name)
	assert.Equal(t, "1.4.3", lf.Packages[0].Version)
	assert.Equal(t, "swi-pack-index", lf.Packages[0].Source)
	assert.NotEmpty(t, lf.Packages[0].Checksum)
	assert.True(t, strings.HasPrefix(lf.Packages[0].Checksum, "sha256:"))

	// Verify pack is in store.
	store := NewStore(storeDir)
	assert.True(t, store.IsInstalled("clpfd", "1.4.3"))
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	// Pre-create the store directory to simulate already-installed.
	store := NewStore(storeDir)
	require.NoError(t, store.EnsureDir())
	require.NoError(t, os.MkdirAll(store.PackPath("clpfd", "1.4.3"), 0755))

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"clpfd": "^1.4"},
	}
	lock := &prolfile.LockFile{
		Packages: []prolfile.LockEntry{
			{
				Name:     "clpfd",
				Version:  "1.4.3",
				Source:   "swi-pack-index",
				URL:      "https://www.swi-prolog.org/pack/clpfd-1.4.3.tar.gz",
				Checksum: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	}

	lf, err := Install(context.Background(), manifest, lock, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)
	assert.Equal(t, "clpfd", lf.Packages[0].Name)
	assert.Equal(t, "1.4.3", lf.Packages[0].Version)
}

func TestInstall_RegistryError(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"nonexistent": "*"},
	}

	_, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestInstall_InvalidLockURL(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	// Pre-create the store to hit the fast path.
	store := NewStore(storeDir)
	require.NoError(t, store.EnsureDir())
	require.NoError(t, os.MkdirAll(store.PackPath("clpfd", "1.4.3"), 0755))

	reg := &mockRegistry{}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"clpfd": "^1.4"},
	}
	lock := &prolfile.LockFile{
		Packages: []prolfile.LockEntry{
			{
				Name:    "clpfd",
				Version: "1.4.3",
				URL:     "https://evil.com/clpfd-1.4.3.tar.gz", // not in allowed hosts
			},
		},
	}

	_, err := Install(context.Background(), manifest, lock, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SEC-14")
}

func TestInstall_MultiplePackages(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	tarballA := makeTarball(t, map[string]string{"a/main.pl": "a."})
	tarballB := makeTarball(t, map[string]string{"b/main.pl": "b."})
	tarballC := makeTarball(t, map[string]string{"c/main.pl": "c."})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "pack-a"):
			w.Write(tarballA)
		case strings.Contains(r.URL.Path, "pack-b"):
			w.Write(tarballB)
		case strings.Contains(r.URL.Path, "pack-c"):
			w.Write(tarballC)
		}
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"pack-a": {{Name: "pack-a", Version: "1.0.0"}},
			"pack-b": {{Name: "pack-b", Version: "2.0.0"}},
			"pack-c": {{Name: "pack-c", Version: "0.1.0"}},
		},
		downloadURL: map[string]string{
			"pack-a@1.0.0": srv.URL + "/pack-a-1.0.0.tar.gz",
			"pack-b@2.0.0": srv.URL + "/pack-b-2.0.0.tar.gz",
			"pack-c@0.1.0": srv.URL + "/pack-c-0.1.0.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies:    map[string]string{"pack-a": "^1.0", "pack-b": "^2.0"},
		DevDependencies: map[string]string{"pack-c": "*"},
	}

	lf, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 3)

	// Packages must be sorted by name.
	assert.Equal(t, "pack-a", lf.Packages[0].Name)
	assert.Equal(t, "pack-b", lf.Packages[1].Name)
	assert.Equal(t, "pack-c", lf.Packages[2].Name)
}

func TestInstall_ContextCancellation(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until client disconnects
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"slow-pack": {{Name: "slow-pack", Version: "1.0.0"}},
		},
		downloadURL: map[string]string{
			"slow-pack@1.0.0": srv.URL + "/slow.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"slow-pack": "*"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := Install(ctx, manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.Error(t, err)
}

func TestInstall_YankedVersionSkipped(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	tarball := makeTarball(t, map[string]string{"pkg/main.pl": "ok."})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"mypack": {
				{Name: "mypack", Version: "2.0.0", Yanked: true},
				{Name: "mypack", Version: "1.0.0", Yanked: false},
			},
		},
		downloadURL: map[string]string{
			"mypack@1.0.0": srv.URL + "/mypack-1.0.0.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"mypack": "*"},
	}

	lf, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)
	assert.Equal(t, "1.0.0", lf.Packages[0].Version) // Skipped yanked 2.0.0.
}

// TestInstall_RegistryChecksumMismatch verifies SEC-016 / CLAUDE.md §8.6:
// when the registry advertises a checksum, the downloaded tarball must be
// verified against THAT checksum (not just hashed and trusted) to close the
// TOCTOU window between version lookup and lock entry creation.
func TestInstall_RegistryChecksumMismatch(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	// Server returns real bytes; registry advertises a sha1 of *different* bytes.
	tarball := makeTarball(t, map[string]string{"pkg/main.pl": "real."})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	// 40-char hex that is NOT the sha1 of `tarball`.
	bogusSHA1 := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"mypack": {{
				Name:     "mypack",
				Version:  "1.0.0",
				URL:      srv.URL + "/mypack-1.0.0.tar.gz",
				Checksum: bogusSHA1,
			}},
		},
		downloadURL: map[string]string{
			"mypack@1.0.0": srv.URL + "/mypack-1.0.0.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"mypack": "*"},
	}

	_, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum")

	// Cache file must have been deleted on mismatch.
	cachedTarball := filepath.Join(cacheDir, "mypack", "1.0.0.tar.gz")
	_, statErr := os.Stat(cachedTarball)
	assert.True(t, os.IsNotExist(statErr), "cache file should be deleted on mismatch")

	// Pack must NOT have been unpacked into the store.
	store := NewStore(storeDir)
	assert.False(t, store.IsInstalled("mypack", "1.0.0"))
}

// TestInstall_RegistryChecksumMatch verifies that a correct sha1 advertised
// by the registry passes verification and the install proceeds.
func TestInstall_RegistryChecksumMatch(t *testing.T) {
	storeDir, cacheDir := setupTestInstall(t)

	tarball := makeTarball(t, map[string]string{"pkg/main.pl": "ok."})

	// Compute the real sha1 of the tarball bytes.
	tmp := filepath.Join(t.TempDir(), "t.tar.gz")
	require.NoError(t, os.WriteFile(tmp, tarball, 0644))
	sha1Hex, err := ComputeSHA1(tmp)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"mypack": {{
				Name:     "mypack",
				Version:  "1.0.0",
				URL:      srv.URL + "/mypack-1.0.0.tar.gz",
				Checksum: sha1Hex,
			}},
		},
		downloadURL: map[string]string{
			"mypack@1.0.0": srv.URL + "/mypack-1.0.0.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"mypack": "*"},
	}

	lf, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)
}

func TestInstall_DeterministicOrder(t *testing.T) {
	// Verify that Install processes packages in alphabetical order regardless of
	// map iteration order. We record the server-side request order to confirm.
	storeDir, cacheDir := setupTestInstall(t)

	tarball := makeTarball(t, map[string]string{"pkg/main.pl": "ok."})

	var requestOrder []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract package name from path, e.g. "/alpha-1.0.0.tar.gz" → "alpha"
		for _, name := range []string{"alpha", "bravo", "charlie"} {
			if strings.Contains(r.URL.Path, name) {
				requestOrder = append(requestOrder, name)
			}
		}
		w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"alpha":   {{Name: "alpha", Version: "1.0.0"}},
			"bravo":   {{Name: "bravo", Version: "1.0.0"}},
			"charlie": {{Name: "charlie", Version: "1.0.0"}},
		},
		downloadURL: map[string]string{
			"alpha@1.0.0":   srv.URL + "/alpha-1.0.0.tar.gz",
			"bravo@1.0.0":   srv.URL + "/bravo-1.0.0.tar.gz",
			"charlie@1.0.0": srv.URL + "/charlie-1.0.0.tar.gz",
		},
	}

	manifest := &prolfile.ProlFile{
		// Deliberately declared out of alphabetical order.
		Dependencies: map[string]string{"charlie": "*", "alpha": "*", "bravo": "*"},
	}

	lf, err := Install(context.Background(), manifest, nil, reg, Options{
		StoreDir: storeDir,
		CacheDir: cacheDir,
	})
	require.NoError(t, err)
	require.Len(t, lf.Packages, 3)

	// Lockfile packages must be sorted alphabetically.
	assert.Equal(t, "alpha", lf.Packages[0].Name)
	assert.Equal(t, "bravo", lf.Packages[1].Name)
	assert.Equal(t, "charlie", lf.Packages[2].Name)

	// Downloads must have happened in alphabetical order.
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, requestOrder)
}

func TestValidateLockURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"allowed swi host", "https://www.swi-prolog.org/pack/test.tar.gz", false},
		{"allowed github", "https://github.com/user/repo/archive/v1.tar.gz", false},
		{"localhost allowed", "http://localhost:8080/test.tar.gz", false},
		{"127.0.0.1 allowed", "http://127.0.0.1:9090/test.tar.gz", false},
		{"unknown host blocked", "https://evil.com/pack.tar.gz", true},
		{"empty URL", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLockURL(tc.url)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLockURL_CustomAllowedHosts(t *testing.T) {
	t.Setenv("PROLM_ALLOWED_HOSTS", "internal.corp.com, packages.example.org")

	assert.NoError(t, validateLockURL("https://internal.corp.com/pack.tar.gz"))
	assert.NoError(t, validateLockURL("https://packages.example.org/pack.tar.gz"))
	assert.Error(t, validateLockURL("https://www.swi-prolog.org/pack.tar.gz")) // default no longer applies
}
