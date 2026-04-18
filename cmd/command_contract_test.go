package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/registry"
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
		got, err := parseAddTarget("aop")
		require.NoError(t, err)
		assert.Equal(t, "aop", got.Name)
		assert.Empty(t, got.SourceURL)
	})

	t.Run("swi_listing_url", func(t *testing.T) {
		got, err := parseAddTarget("https://www.swi-prolog.org/pack/list?p=aop")
		require.NoError(t, err)
		assert.Equal(t, "aop", got.Name)
		assert.Empty(t, got.SourceURL)
	})

	t.Run("swi_tarball_url", func(t *testing.T) {
		raw := "https://www.swi-prolog.org/pack/file_details?path=aop-0.0.9.tgz"
		got, err := parseAddTarget(raw)
		require.NoError(t, err)
		assert.Equal(t, "aop", got.Name)
		assert.Equal(t, raw, got.SourceURL)
	})

	t.Run("github_repo_url", func(t *testing.T) {
		raw := "https://github.com/SWI-Prolog/packages-clpfd"
		got, err := parseAddTarget(raw)
		require.NoError(t, err)
		assert.Equal(t, "packages-clpfd", got.Name)
		assert.Equal(t, "SWI-Prolog/packages-clpfd", got.GitHubRepoRef)
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
		_, err := parseAddTarget("http://www.swi-prolog.org/pack/list?p=aop")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTPS")
	})
}

func TestExecute_Add_UpdatesManifestAndSyncs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	var called bool
	var gotOverride map[string]registry.PackageVersion
	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, nil
		},
		sync: func(_ *prolfile.ProlFile, manifestPath string, overrides map[string]registry.PackageVersion) error {
			called = true
			gotOverride = overrides
			manifestContent, err := os.ReadFile(manifestPath)
			require.NoError(t, err)
			assert.NotContains(t, string(manifestContent), "aop = \"*\"")
			assert.NotContains(t, string(manifestContent), "aop = \"^")

			lockContent := `[meta]
lock_version = 1
prolfile_hash = ""

[[package]]
name = "aop"
version = "0.0.9"
source = "swi-pack-index"
url = "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz"
checksum = "sha256:abc"
dependencies = []
`
			return os.WriteFile(filepath.Join(filepath.Dir(manifestPath), "Prolfile.lock"), []byte(lockContent), 0644)
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "aop"})
	require.NoError(t, Execute())
	require.True(t, called, "add must sync manifest -> lock/install")
	assert.Empty(t, gotOverride)

	pf, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	require.NotNil(t, pf.Dependencies)
	assert.Equal(t, "^0.0.9", pf.Dependencies["aop"])
}

func TestExecute_Add_URLOverride_PassesThroughToSync(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	var gotOverride map[string]registry.PackageVersion
	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, nil
		},
		sync: func(_ *prolfile.ProlFile, _ string, overrides map[string]registry.PackageVersion) error {
			gotOverride = overrides
			return nil
		},
	})

	raw := "https://www.swi-prolog.org/pack/file_details?path=aop-0.0.9.tgz"
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", raw})
	require.NoError(t, Execute())

	require.Contains(t, gotOverride, "aop")
	assert.Equal(t, raw, gotOverride["aop"].URL)
	assert.Equal(t, "0.0.9", gotOverride["aop"].Version)
}

func TestExecute_Add_SyncFailureRollsBackManifest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, nil
		},
		sync: func(_ *prolfile.ProlFile, _ string, _ map[string]registry.PackageVersion) error {
			return assert.AnError
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "aop"})
	err := Execute()
	require.Error(t, err)

	pf, _, loadErr := loadManifest(rootCmd)
	require.NoError(t, loadErr)
	assert.NotContains(t, pf.Dependencies, "aop", "manifest must roll back when sync fails")
}

func TestExecute_Add_GitHubRepoURL_PinsResolvedVersion(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{
				Name:    "aop",
				Version: "0.0.9",
				URL:     "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz",
			}, nil
		},
		sync: func(pf *prolfile.ProlFile, manifestPath string, _ map[string]registry.PackageVersion) error {
			lockContent := `[meta]
lock_version = 1
prolfile_hash = ""

[[package]]
name = "aop"
version = "0.0.9"
source = "github"
url = "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz"
checksum = "sha256:abc"
dependencies = []
`
			return os.WriteFile(filepath.Join(filepath.Dir(manifestPath), "Prolfile.lock"), []byte(lockContent), 0644)
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "https://github.com/hargettp/aop"})
	require.NoError(t, Execute())

	pf, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	assert.Equal(t, "^0.0.9", pf.Dependencies["aop"])
}

func TestExecute_Add_GitHubRepoURL_NoStableTagFailsWithGuidance(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, assert.AnError
		},
		sync: func(_ *prolfile.ProlFile, _ string, _ map[string]registry.PackageVersion) error {
			return nil
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "https://github.com/hargettp/aop"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving github repository")
}

func TestExecute_Add_SWIListingURL_PinsVersionFromLockfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, nil
		},
		sync: func(_ *prolfile.ProlFile, manifestPath string, _ map[string]registry.PackageVersion) error {
			lockContent := `[meta]
lock_version = 1
prolfile_hash = ""

[[package]]
name = "aop"
version = "0.0.9"
source = "swi-pack-index"
url = "https://github.com/hargettp/aop/archive/refs/tags/v0.0.9.tar.gz"
checksum = "sha256:abc"
dependencies = []
`
			return os.WriteFile(filepath.Join(filepath.Dir(manifestPath), "Prolfile.lock"), []byte(lockContent), 0644)
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "https://www.swi-prolog.org/pack/list?p=aop"})
	require.NoError(t, Execute())

	pf, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	assert.Equal(t, "^0.0.9", pf.Dependencies["aop"])
}

func TestExecute_Add_UnresolvedVersionLeavesWildcard(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withAddRunner(t, &addRunner{
		resolveGitHubRepo: func(_ context.Context, _ string) (registry.PackageVersion, error) {
			return registry.PackageVersion{}, nil
		},
		sync: func(_ *prolfile.ProlFile, _ string, _ map[string]registry.PackageVersion) error {
			return nil
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"add", "aop"})
	require.NoError(t, Execute())

	pf, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	assert.Equal(t, "*", pf.Dependencies["aop"])
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
	pf.Dependencies["aop"] = "*"
	require.NoError(t, manifest.Save(manifestPath, pf))

	var called bool
	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error {
			called = true
			return nil
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "aop"})
	require.NoError(t, Execute())
	require.True(t, called, "remove must sync manifest -> lock/install")

	pf2, _, err := loadManifest(rootCmd)
	require.NoError(t, err)
	assert.NotContains(t, pf2.Dependencies, "aop")
}

func TestExecute_Remove_MissingDependencyReturnsActionableError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeBasicManifest(t, dir)

	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error { return nil },
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "aop"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not declared")
	assert.Contains(t, err.Error(), "prolm add aop")
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
	pf.Dependencies["aop"] = "*"
	require.NoError(t, manifest.Save(manifestPath, pf))

	withRemoveRunner(t, &removeRunner{
		sync: func(_ *prolfile.ProlFile, _ string) error {
			return assert.AnError
		},
	})

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"remove", "aop"})
	err = Execute()
	require.Error(t, err)

	pf2, _, loadErr := loadManifest(rootCmd)
	require.NoError(t, loadErr)
	assert.Equal(t, "*", pf2.Dependencies["aop"], "manifest must roll back when sync fails")
}

func TestAddCommandHelp_UsesInstallableExamplePack(t *testing.T) {
	assert.Contains(t, addCmd.Long, "for example: aop")
	assert.NotContains(t, addCmd.Long, "for example: clpfd")
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
