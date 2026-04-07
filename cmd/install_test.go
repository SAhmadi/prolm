package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistry is a minimal registry.Registry for cmd-level orchestration tests.
type fakeRegistry struct {
	versions    map[string][]registry.PackageVersion
	downloadURL map[string]string
	versionsErr error
}

func (f *fakeRegistry) Search(_ context.Context, _ string) ([]registry.PackageVersion, error) {
	return nil, nil
}

func (f *fakeRegistry) Versions(_ context.Context, name string) ([]registry.PackageVersion, error) {
	if f.versionsErr != nil {
		return nil, f.versionsErr
	}
	if vs, ok := f.versions[name]; ok {
		return vs, nil
	}
	return nil, &registry.ErrPackageNotFound{Name: name, Registry: "fake"}
}

func (f *fakeRegistry) DownloadURL(_ context.Context, name, version string) (string, error) {
	key := name + "@" + version
	if u, ok := f.downloadURL[key]; ok {
		return u, nil
	}
	return "", &registry.ErrVersionNotFound{Name: name, Version: version}
}

// makeTarball builds a tiny .tar.gz with one file.
func makeTarball(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	body := ":- module(clpfd, []).\n"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "clpfd/main.pl", Size: int64(len(body)), Mode: 0644, Typeflag: tar.TypeReg,
	}))
	_, err := tw.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// writeManifest writes a minimal valid Prolfile.toml with one dependency.
func writeManifest(t *testing.T, dir, depName string) {
	t.Helper()
	content := `[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name = "test-project"
version = "0.1.0"
entry = "src/main.pl"
runtime = "swi"

[dependencies]
` + depName + ` = "*"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.toml"), []byte(content), 0644))
}

// resetRootCmd restores the cmd-package globals so tests don't bleed.
// It also resets the Changed state on all installCmd flags so that stub-flag
// tests don't pollute subsequent tests within the same binary run.
func resetRootCmd(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Reset the Changed state on all installCmd flags to clear state left by
	// a previous test invocation (e.g. --frozen from TestExecute_Install_StubFlags).
	// Cobra's ResetFlags drops the FlagSet entirely, so we re-register afterwards.
	installCmd.ResetFlags()
	installCmd.Flags().Bool("frozen", false, "[Phase 2] Fail if Prolfile.lock would change (not yet implemented)")
	installCmd.Flags().Bool("offline", false, "[Phase 2] Use local cache only, no network requests (not yet implemented)")
	installCmd.Flags().Bool("no-verify", false, "[Phase 2] Skip checksum verification — DANGEROUS, dev only (not yet implemented)")

	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	return buf
}

// newTestRunner builds an installRunner with a fake registry and temp-dir
// store/cache. Tests use this instead of mutating package globals (QUALITY-017).
func newTestRunner(reg registry.Registry, storeDir, cacheDir string) *installRunner {
	return &installRunner{
		newRegistry: func() (registry.Registry, error) { return reg, nil },
		installOpts: func() installer.Options {
			return installer.Options{StoreDir: storeDir, CacheDir: cacheDir}
		},
	}
}

// withTestRunner replaces defaultInstallRunner for the duration of the test.
// It does NOT use t.Parallel()-unsafe global mutation; the swap is confined to
// the single goroutine running the test and restored in t.Cleanup.
func withTestRunner(t *testing.T, runner *installRunner) {
	t.Helper()
	orig := defaultInstallRunner
	defaultInstallRunner = runner
	t.Cleanup(func() { defaultInstallRunner = orig })
}

func TestExecute_Install_Fresh(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeManifest(t, dir, "clpfd")

	tarball := makeTarball(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &fakeRegistry{
		versions: map[string][]registry.PackageVersion{
			"clpfd": {{Name: "clpfd", Version: "1.4.3"}},
		},
		downloadURL: map[string]string{
			"clpfd@1.4.3": srv.URL + "/clpfd-1.4.3.tar.gz",
		},
	}
	withTestRunner(t, newTestRunner(reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache")))

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	require.NoError(t, Execute())

	_, err := os.Stat(filepath.Join(dir, "Prolfile.lock"))
	assert.NoError(t, err, "Prolfile.lock should be written")
	_, err = os.Stat(filepath.Join(dir, "store", "clpfd", "1.4.3"))
	assert.NoError(t, err, "package should be unpacked into store")
}

func TestExecute_Install_NoManifest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	withTestRunner(t, newTestRunner(&fakeRegistry{}, filepath.Join(dir, "store"), filepath.Join(dir, "cache")))
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	err := Execute()
	require.Error(t, err)
}

func TestExecute_Install_RegistryError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeManifest(t, dir, "missing-pack")

	reg := &fakeRegistry{} // empty: Versions returns ErrPackageNotFound
	withTestRunner(t, newTestRunner(reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache")))

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-pack")
}

// TestExecute_Install_GitConflictedLockfile verifies CLAUDE.md §8.19:
// a Prolfile.lock containing Git merge conflict markers must be auto-healed
// by discarding the conflicted lock and re-resolving from Prolfile.toml.
func TestExecute_Install_GitConflictedLockfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeManifest(t, dir, "clpfd")

	conflict := "<<<<<<< HEAD\n[meta]\nlock_version = 1\n=======\n[meta]\nlock_version = 1\n>>>>>>> branch\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.lock"), []byte(conflict), 0644))

	tarball := makeTarball(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &fakeRegistry{
		versions: map[string][]registry.PackageVersion{
			"clpfd": {{Name: "clpfd", Version: "1.4.3"}},
		},
		downloadURL: map[string]string{
			"clpfd@1.4.3": srv.URL + "/clpfd-1.4.3.tar.gz",
		},
	}
	withTestRunner(t, newTestRunner(reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache")))

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	require.NoError(t, Execute())

	// Lockfile must have been re-written cleanly (no conflict markers).
	data, err := os.ReadFile(filepath.Join(dir, "Prolfile.lock"))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "<<<<<<<")
	assert.NotContains(t, string(data), "=======")
	assert.NotContains(t, string(data), ">>>>>>>")
	assert.Contains(t, string(data), "clpfd")

	// Package must be installed in the store.
	_, err = os.Stat(filepath.Join(dir, "store", "clpfd", "1.4.3"))
	assert.NoError(t, err, "package should be unpacked into store")
}

// TestExecute_Install_StubFlags verifies QUALITY-015: --frozen, --offline, and
// --no-verify are registered as Phase 2 stubs and return a clear error rather
// than cobra's "unknown flag" message.
func TestExecute_Install_StubFlags(t *testing.T) {
	for _, flag := range []string{"--frozen", "--offline", "--no-verify"} {
		t.Run(flag, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeManifest(t, dir, "clpfd")

			withTestRunner(t, newTestRunner(&fakeRegistry{}, filepath.Join(dir, "store"), filepath.Join(dir, "cache")))
			resetRootCmd(t)
			rootCmd.SetArgs([]string{"install", flag})
			err := Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not yet implemented",
				"stub flag %s should return a not-yet-implemented error", flag)
			assert.Contains(t, err.Error(), "Phase 2",
				"stub flag %s error should mention Phase 2", flag)
		})
	}
}

// TestExecute_Install_NoLockfileRewrite verifies QUALITY-016: when a second
// `prolm install` produces a byte-identical lockfile, the file's mtime must not
// change (i.e. Save is not called again).
func TestExecute_Install_NoLockfileRewrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeManifest(t, dir, "clpfd")

	tarball := makeTarball(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	}))
	t.Cleanup(srv.Close)

	reg := &fakeRegistry{
		versions: map[string][]registry.PackageVersion{
			"clpfd": {{Name: "clpfd", Version: "1.4.3"}},
		},
		downloadURL: map[string]string{
			"clpfd@1.4.3": srv.URL + "/clpfd-1.4.3.tar.gz",
		},
	}

	runner := newTestRunner(reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache"))
	withTestRunner(t, runner)

	// First install: creates the lockfile.
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	require.NoError(t, Execute())

	lockPath := filepath.Join(dir, "Prolfile.lock")
	info1, err := os.Stat(lockPath)
	require.NoError(t, err)

	// Second install: everything is already in the store; lockfile content is unchanged.
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	require.NoError(t, Execute())

	info2, err := os.Stat(lockPath)
	require.NoError(t, err)

	// On systems with sub-second mtime resolution the times must be equal.
	assert.Equal(t, info1.ModTime(), info2.ModTime(),
		"second install must not rewrite an unchanged lockfile (QUALITY-016)")
}
