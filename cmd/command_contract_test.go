package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Root_BareGlobalFlag_ShowsGuidanceAndHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"--no-color"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "No command specified")
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "Available Commands:")
}

func TestExecute_HelpOutput_HasNoInternalAuthoringRefs(t *testing.T) {
	cases := [][]string{
		{"--help"},
		{"--no-color", "--help"},
		{"new", "--help"},
		{"check", "-h"},
		{"completion", "--help"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			defer func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
				rootCmd.SetArgs(nil)
			}()

			rootCmd.SetArgs(args)
			err := Execute()
			require.NoError(t, err)
			out := strings.ToLower(buf.String())
			assert.NotContains(t, out, "claude.md")
			assert.NotContains(t, out, "codex")
			assert.NotContains(t, out, "chatgpt")
		})
	}
}

func TestParseAddTarget(t *testing.T) {
	t.Run("package_name", func(t *testing.T) {
		got, err := parseAddTarget("clpfd")
		require.NoError(t, err)
		assert.Equal(t, "clpfd", got.Name)
		assert.Empty(t, got.SourceURL)
	})

	t.Run("swi_listing_url", func(t *testing.T) {
		got, err := parseAddTarget("https://www.swi-prolog.org/pack/list?p=clpfd")
		require.NoError(t, err)
		assert.Equal(t, "clpfd", got.Name)
		assert.Empty(t, got.SourceURL)
	})

	t.Run("swi_tarball_url", func(t *testing.T) {
		raw := "https://www.swi-prolog.org/pack/file_details?path=clpfd-1.2.3.tgz"
		got, err := parseAddTarget(raw)
		require.NoError(t, err)
		assert.Equal(t, "clpfd", got.Name)
		assert.Equal(t, raw, got.SourceURL)
	})

	t.Run("github_repo_url", func(t *testing.T) {
		got, err := parseAddTarget("https://github.com/SWI-Prolog/packages-clpfd")
		require.NoError(t, err)
		assert.Equal(t, "packages-clpfd", got.Name)
		assert.Empty(t, got.SourceURL)
	})

	t.Run("github_archive_url", func(t *testing.T) {
		raw := "https://github.com/SWI-Prolog/packages-clpfd/archive/refs/tags/v1.0.0.tar.gz"
		got, err := parseAddTarget(raw)
		require.NoError(t, err)
		assert.Equal(t, "packages-clpfd", got.Name)
		assert.Equal(t, raw, got.SourceURL)
	})

	t.Run("http_rejected", func(t *testing.T) {
		_, err := parseAddTarget("http://www.swi-prolog.org/pack/list?p=clpfd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTPS")
	})
}

func TestExecute_Add_UpdatesManifestAndSyncs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	var called bool
	var gotOverride map[string]string
	withAddRunner(t, &addRunner{
		sync: func(_ *prolfile.ProlFile, _ string, overrides map[string]string) error {
			called = true
			gotOverride = overrides
			return nil
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "clpfd"})
	require.NoError(t, Execute())
	require.True(t, called, "add must sync manifest -> lock/install")
	assert.Empty(t, gotOverride)

	pf, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	require.NotNil(t, pf.Dependencies)
	assert.Equal(t, "*", pf.Dependencies["clpfd"])
}

func TestExecute_Add_URLOverride_PassesThroughToSync(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	var gotOverride map[string]string
	withAddRunner(t, &addRunner{
		sync: func(_ *prolfile.ProlFile, _ string, overrides map[string]string) error {
			gotOverride = overrides
			return nil
		},
	})

	raw := "https://www.swi-prolog.org/pack/file_details?path=clpfd-1.2.3.tgz"
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", raw})
	require.NoError(t, Execute())

	assert.Equal(t, map[string]string{"clpfd": raw}, gotOverride)
}

func TestExecute_Add_SyncFailureRollsBackManifest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		sync: func(_ *prolfile.ProlFile, _ string, _ map[string]string) error {
			return assert.AnError
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "clpfd"})
	err := Execute()
	require.Error(t, err)

	pf, _, loadErr := loadManifest(rootCmd)
	require.NoError(t, loadErr)
	assert.NotContains(t, pf.Dependencies, "clpfd", "manifest must roll back when sync fails")
}

func TestExecute_Remove_UpdatesManifestAndSyncs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)
	pf, manifestPath, err := loadManifest(rootCmd)
	require.NoError(t, err)
	if pf.Dependencies == nil {
		pf.Dependencies = map[string]string{}
	}
	pf.Dependencies["clpfd"] = "*"
	require.NoError(t, manifest.Save(manifestPath, pf))

	var called bool
	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error {
			called = true
			return nil
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "clpfd"})
	require.NoError(t, Execute())
	require.True(t, called, "remove must sync manifest -> lock/install")

	pf2, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	assert.NotContains(t, pf2.Dependencies, "clpfd")
}

func TestExecute_Remove_MissingDependencyReturnsActionableError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error { return nil },
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "clpfd"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not declared")
	assert.Contains(t, err.Error(), "prolm add clpfd")
}

func TestExecute_Remove_SyncFailureRollsBackManifest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)
	pf, manifestPath, err := loadManifest(rootCmd)
	require.NoError(t, err)
	if pf.Dependencies == nil {
		pf.Dependencies = map[string]string{}
	}
	pf.Dependencies["clpfd"] = "*"
	require.NoError(t, manifest.Save(manifestPath, pf))

	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error {
			return assert.AnError
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "clpfd"})
	err = Execute()
	require.Error(t, err)

	pf2, _, loadErr := loadManifest(rootCmd)
	require.NoError(t, loadErr)
	assert.Equal(t, "*", pf2.Dependencies["clpfd"], "manifest must roll back when sync fails")
}

func writeBasicManifest(t *testing.T, dir string) {
	t.Helper()
	content := `[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name = "phase15-project"
version = "0.1.0"
entry = "src/main.pl"
runtime = "swi"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.toml"), []byte(content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.lock"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.pl"), []byte("main :- true.\n"), 0644))
}

func withAddRunner(t *testing.T, r *addRunner) {
	t.Helper()
	orig := defaultAddRunner
	defaultAddRunner = r
	t.Cleanup(func() { defaultAddRunner = orig })
}

func withRemoveRunner(t *testing.T, r *removeRunner) {
	t.Helper()
	orig := defaultRemoveRunner
	defaultRemoveRunner = r
	t.Cleanup(func() { defaultRemoveRunner = orig })
}
