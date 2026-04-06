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
func resetRootCmd(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	return buf
}

// withInstallSeams swaps the cmd seams for the duration of the test.
func withInstallSeams(t *testing.T, reg registry.Registry, storeDir, cacheDir string) {
	t.Helper()
	origReg := newRegistry
	origOpts := installOpts
	newRegistry = func() (registry.Registry, error) { return reg, nil }
	installOpts = func() installer.Options {
		return installer.Options{StoreDir: storeDir, CacheDir: cacheDir}
	}
	t.Cleanup(func() {
		newRegistry = origReg
		installOpts = origOpts
	})
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
	withInstallSeams(t, reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache"))

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

	withInstallSeams(t, &fakeRegistry{}, filepath.Join(dir, "store"), filepath.Join(dir, "cache"))
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
	withInstallSeams(t, reg, filepath.Join(dir, "store"), filepath.Join(dir, "cache"))

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-pack")
}

func TestExecute_Install_GitConflictedLockfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeManifest(t, dir, "clpfd")

	conflict := "<<<<<<< HEAD\n[meta]\nlock_version = 1\n=======\n[meta]\nlock_version = 1\n>>>>>>> branch\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.lock"), []byte(conflict), 0644))

	withInstallSeams(t, &fakeRegistry{}, filepath.Join(dir, "store"), filepath.Join(dir, "cache"))
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"install"})
	err := Execute()
	require.Error(t, err)
}
