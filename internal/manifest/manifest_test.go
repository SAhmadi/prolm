package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTOML = `[meta]
min_prolm_version = "0.1.0"
prolfile_version = 1

[package]
authors = ["Ada <ada@example.com>"]
description = "A test project"
entry = "src/main.pl"
homepage = "https://example.com"
license = "MIT"
name = "test-project"
runtime = "swi"
version = "1.0.0"

[dependencies]
clpfd = "^1.4"
http = "~7.0.1"

[dev-dependencies]
plunit = "*"

[runtime]

  [runtime.swi]
    flags = ["-O"]
    min_version = "9.0"

[scripts]
start = "prolm run"
`

func writeTOML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, prolfileName)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// --- Discover tests ---

func TestDiscover_FindsInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, validTOML)

	got, err := Discover(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, prolfileName), got)
}

func TestDiscover_WalksUpward(t *testing.T) {
	root := t.TempDir()
	writeTOML(t, root, validTOML)

	child := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(child, 0755))

	got, err := Discover(child)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, prolfileName), got)
}

func TestDiscover_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDiscover_SkipsDirectory(t *testing.T) {
	dir := t.TempDir()
	// Create a directory named Prolfile.toml (not a file)
	require.NoError(t, os.Mkdir(filepath.Join(dir, prolfileName), 0755))

	_, err := Discover(dir)
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Load tests ---

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, validTOML)

	pf, err := Load(path, LoadOptions{ProlmVersion: "1.0.0"})
	require.NoError(t, err)

	assert.Equal(t, "test-project", pf.Package.Name)
	assert.Equal(t, "1.0.0", pf.Package.Version)
	assert.Equal(t, "^1.4", pf.Dependencies["clpfd"])
	assert.Equal(t, "*", pf.DevDependencies["plunit"])
	assert.Equal(t, "9.0", pf.Runtime["swi"].MinVersion)
}

func TestLoad_AutoDiscover(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, validTOML)

	// Change into dir so Discover finds it
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(orig)

	pf, err := Load("", LoadOptions{ProlmVersion: "1.0.0"})
	require.NoError(t, err)
	assert.Equal(t, "test-project", pf.Package.Name)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/Prolfile.toml", LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestLoad_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta
broken toml`)

	_, err := Load(path, LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestLoad_MissingProlfileVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[package]
name = "test"
version = "1.0.0"
`)
	_, err := Load(path, LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prolfile_version")
}

func TestLoad_VersionTooNew(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta]
prolfile_version = 99

[package]
name = "test"
version = "1.0.0"
`)
	_, err := Load(path, LoadOptions{})
	require.Error(t, err)

	var vErr *VersionTooNewError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, 99, vErr.Found)
}

func TestLoad_ProlmTooOld(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta]
prolfile_version = 1
min_prolm_version = "99.0.0"

[package]
name = "test"
version = "1.0.0"
`)
	_, err := Load(path, LoadOptions{ProlmVersion: "0.1.0"})
	require.Error(t, err)

	var pErr *ProlmTooOldError
	require.ErrorAs(t, err, &pErr)
	assert.Equal(t, "99.0.0", pErr.Required)
}

func TestLoad_ProlmVersionDev(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name = "test"
version = "1.0.0"
entry = "src/main.pl"
runtime = "swi"
`)
	// "0.1.0-dev" should satisfy min_prolm_version = "0.1.0"
	pf, err := Load(path, LoadOptions{ProlmVersion: "0.1.0-dev"})
	require.NoError(t, err)
	assert.Equal(t, "test", pf.Package.Name)
}

func TestLoad_SkipsVersionCheckWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta]
prolfile_version = 1
min_prolm_version = "99.0.0"

[package]
name = "test"
version = "1.0.0"
`)
	// Empty ProlmVersion skips the min_prolm_version check entirely
	_, err := Load(path, LoadOptions{})
	// Should not get a ProlmTooOldError
	var pErr *ProlmTooOldError
	assert.False(t, errors.As(err, &pErr), "should not get ProlmTooOldError when ProlmVersion is empty")
}

func TestLoad_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, `[meta]
prolfile_version = 1

[package]
name = "INVALID"
version = "1.0.0"
`)
	_, err := Load(path, LoadOptions{})
	require.Error(t, err)

	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
}

// --- Save tests ---

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, prolfileName)

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			MinProlmVersion: "0.1.0",
			ProlfileVersion: 1,
		},
		Package: prolfile.Package{
			Name:    "test-project",
			Version: "1.0.0",
			Entry:   "src/main.pl",
			Runtime: "swi",
		},
		Dependencies: map[string]string{
			"clpfd": "^1.4",
			"http":  "~7.0.1",
		},
	}

	require.NoError(t, Save(path, pf))

	loaded, err := Load(path, LoadOptions{ProlmVersion: "1.0.0"})
	require.NoError(t, err)

	assert.Equal(t, pf.Package.Name, loaded.Package.Name)
	assert.Equal(t, pf.Package.Version, loaded.Package.Version)
	assert.Equal(t, pf.Dependencies["clpfd"], loaded.Dependencies["clpfd"])
	assert.Equal(t, pf.Dependencies["http"], loaded.Dependencies["http"])
}

func TestSave_Deterministic(t *testing.T) {
	dir := t.TempDir()

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			MinProlmVersion: "0.1.0",
			ProlfileVersion: 1,
		},
		Package: prolfile.Package{
			Name:    "test",
			Version: "1.0.0",
			Entry:   "src/main.pl",
			Runtime: "swi",
		},
		Dependencies: map[string]string{
			"zlib": "^1.0",
			"alib": "^2.0",
		},
	}

	path1 := filepath.Join(dir, "a.toml")
	path2 := filepath.Join(dir, "b.toml")

	require.NoError(t, Save(path1, pf))
	require.NoError(t, Save(path2, pf))

	data1, _ := os.ReadFile(path1)
	data2, _ := os.ReadFile(path2)
	assert.Equal(t, data1, data2, "Save must produce identical output")
}

func TestSave_SectionOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, prolfileName)

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			MinProlmVersion: "0.1.0",
			ProlfileVersion: 1,
		},
		Package: prolfile.Package{
			Name:    "test",
			Version: "1.0.0",
			Entry:   "src/main.pl",
			Runtime: "swi",
		},
		Dependencies:    map[string]string{"clpfd": "^1.4"},
		DevDependencies: map[string]string{"plunit": "*"},
		Runtime: map[string]prolfile.RuntimeConfig{
			"swi": {MinVersion: "9.0"},
		},
		Scripts: map[string]string{"start": "prolm run"},
	}

	require.NoError(t, Save(path, pf))
	data, _ := os.ReadFile(path)
	content := string(data)

	// Verify section ordering by checking string positions
	sections := []string{"[meta]", "[package]", "[dependencies]", "[dev-dependencies]", "[runtime]", "[scripts]"}
	var positions []int
	for _, s := range sections {
		pos := strings.Index(content, s)
		require.NotEqual(t, -1, pos, "section %s not found in output", s)
		positions = append(positions, pos)
	}
	for i := 1; i < len(positions); i++ {
		assert.Less(t, positions[i-1], positions[i],
			"section %s must appear before %s", sections[i-1], sections[i])
	}
}

func TestSave_AlphabeticMapKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, prolfileName)

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{ProlfileVersion: 1},
		Package: prolfile.Package{
			Name:    "test",
			Version: "1.0.0",
		},
		Dependencies: map[string]string{
			"zlib":  "^1.0",
			"alib":  "^2.0",
			"mlib":  "^3.0",
		},
	}

	require.NoError(t, Save(path, pf))
	data, _ := os.ReadFile(path)
	content := string(data)

	posA := strings.Index(content,"alib")
	posM := strings.Index(content,"mlib")
	posZ := strings.Index(content,"zlib")
	assert.Less(t, posA, posM)
	assert.Less(t, posM, posZ)
}

func TestSave_OmitsEmptySections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, prolfileName)

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{ProlfileVersion: 1},
		Package: prolfile.Package{
			Name:    "test",
			Version: "1.0.0",
		},
		// No dependencies, dev-dependencies, runtime, or scripts
	}

	require.NoError(t, Save(path, pf))
	data, _ := os.ReadFile(path)
	content := string(data)

	assert.NotContains(t, content, "[dependencies]")
	assert.NotContains(t, content, "[dev-dependencies]")
	assert.NotContains(t, content, "[scripts]")
}

func TestSave_NamespacedDeps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, prolfileName)

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{ProlfileVersion: 1},
		Package: prolfile.Package{
			Name:    "test",
			Version: "1.0.0",
		},
		Dependencies: map[string]string{
			"ada/clpfd": "^1.4",
		},
	}

	require.NoError(t, Save(path, pf))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "ada/clpfd")
}

// --- checkMinProlmVersion tests ---

func TestCheckMinProlmVersion_DevSatisfiesRelease(t *testing.T) {
	assert.NoError(t, checkMinProlmVersion("0.1.0", "0.1.0-dev"))
}

func TestCheckMinProlmVersion_OlderFails(t *testing.T) {
	err := checkMinProlmVersion("2.0.0", "1.0.0")
	require.Error(t, err)
	var pErr *ProlmTooOldError
	require.ErrorAs(t, err, &pErr)
}

func TestCheckMinProlmVersion_EmptySkips(t *testing.T) {
	assert.NoError(t, checkMinProlmVersion("", "1.0.0"))
	assert.NoError(t, checkMinProlmVersion("1.0.0", ""))
}
