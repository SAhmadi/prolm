package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validLockTOML = `[meta]
lock_version = 1
prolfile_hash = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

[[package]]
name = "alpha"
version = "1.0.0"
source = "swi-pack-index"
url = "https://example.com/alpha-1.0.0.tar.gz"
checksum = "sha256:aaa111"
dependencies = ["beta@2.0.0"]

[[package]]
name = "beta"
version = "2.0.0"
source = "swi-pack-index"
url = "https://example.com/beta-2.0.0.tar.gz"
checksum = "sha256:bbb222"
dependencies = []
`

func writeLock(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, LockFileName)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func sampleLockFile() *prolfile.LockFile {
	return &prolfile.LockFile{
		Meta: prolfile.LockMeta{
			LockVersion:  1,
			ProlfileHash: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		Packages: []prolfile.LockEntry{
			{
				Name:         "alpha",
				Version:      "1.0.0",
				Source:       "swi-pack-index",
				URL:          "https://example.com/alpha-1.0.0.tar.gz",
				Checksum:     "sha256:aaa111",
				Dependencies: []string{"beta@2.0.0"},
			},
			{
				Name:         "beta",
				Version:      "2.0.0",
				Source:       "swi-pack-index",
				URL:          "https://example.com/beta-2.0.0.tar.gz",
				Checksum:     "sha256:bbb222",
				Dependencies: []string{},
			},
		},
	}
}

// --- Load tests ---

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, validLockTOML)

	lf, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, lf)

	assert.Equal(t, 1, lf.Meta.LockVersion)
	assert.Contains(t, lf.Meta.ProlfileHash, "sha256:")
	require.Len(t, lf.Packages, 2)
	assert.Equal(t, "alpha", lf.Packages[0].Name)
	assert.Equal(t, "1.0.0", lf.Packages[0].Version)
	assert.Equal(t, "swi-pack-index", lf.Packages[0].Source)
	assert.Equal(t, "https://example.com/alpha-1.0.0.tar.gz", lf.Packages[0].URL)
	assert.Equal(t, "sha256:aaa111", lf.Packages[0].Checksum)
	assert.Equal(t, []string{"beta@2.0.0"}, lf.Packages[0].Dependencies)
	assert.Equal(t, "beta", lf.Packages[1].Name)
	assert.Empty(t, lf.Packages[1].Dependencies)
}

func TestLoad_MissingFileReturnsNil(t *testing.T) {
	lf, err := Load("/nonexistent/path/Prolfile.lock")
	assert.NoError(t, err)
	assert.Nil(t, lf)
}

func TestLoad_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, `[meta
broken toml`)

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestLoad_MissingLockVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, `[meta]
prolfile_hash = "sha256:abc"

[[package]]
name = "foo"
version = "1.0.0"
source = "swi-pack-index"
url = "https://example.com/foo.tar.gz"
checksum = "sha256:abc"
dependencies = []
`)

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock_version")
}

func TestLoad_LockVersionTooNew(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, `[meta]
lock_version = 99
prolfile_hash = "sha256:abc"
`)

	_, err := Load(path)
	require.Error(t, err)

	var vErr *LockVersionTooNewError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, 99, vErr.Found)
	assert.Equal(t, prolfile.CurrentLockVersion, vErr.Supported)
}

func TestLoad_GitConflictMarkers(t *testing.T) {
	markers := []struct {
		name    string
		content string
	}{
		{"opening marker", "[meta]\n<<<<<<< HEAD\nlock_version = 1\n"},
		{"separator", "[meta]\n=======\nlock_version = 1\n"},
		{"closing marker", "[meta]\n>>>>>>> branch\nlock_version = 1\n"},
		{"full conflict", `[meta]
<<<<<<< HEAD
lock_version = 1
prolfile_hash = "sha256:aaa"
=======
lock_version = 1
prolfile_hash = "sha256:bbb"
>>>>>>> feature
`},
	}

	for _, tc := range markers {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLock(t, dir, tc.content)

			_, err := Load(path)
			require.Error(t, err)

			var gErr *GitConflictError
			require.ErrorAs(t, err, &gErr)
			assert.Contains(t, gErr.Error(), "merge conflict")
		})
	}
}

func TestLoad_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	dir := t.TempDir()
	path := writeLock(t, dir, validLockTOML)
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { os.Chmod(path, 0644) })

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, "")

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock_version")
}

func TestLoad_SignatureFieldParsed(t *testing.T) {
	dir := t.TempDir()
	path := writeLock(t, dir, `[meta]
lock_version = 1
prolfile_hash = "sha256:abc"

[[package]]
name = "signed-pkg"
version = "1.0.0"
source = "swi-pack-index"
url = "https://example.com/signed.tar.gz"
checksum = "sha256:abc"
dependencies = []
signature = "sig:xyz"
`)

	lf, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "sig:xyz", lf.Packages[0].Signature)
}

// --- Save tests ---

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	original := sampleLockFile()
	require.NoError(t, Save(path, original))

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, original.Meta.LockVersion, loaded.Meta.LockVersion)
	assert.Equal(t, original.Meta.ProlfileHash, loaded.Meta.ProlfileHash)
	require.Len(t, loaded.Packages, len(original.Packages))
	for i := range original.Packages {
		assert.Equal(t, original.Packages[i].Name, loaded.Packages[i].Name)
		assert.Equal(t, original.Packages[i].Version, loaded.Packages[i].Version)
		assert.Equal(t, original.Packages[i].Source, loaded.Packages[i].Source)
		assert.Equal(t, original.Packages[i].URL, loaded.Packages[i].URL)
		assert.Equal(t, original.Packages[i].Checksum, loaded.Packages[i].Checksum)
		assert.Equal(t, original.Packages[i].Dependencies, loaded.Packages[i].Dependencies)
	}
}

func TestSave_AtomicNoTmpFileRemains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	require.NoError(t, Save(path, sampleLockFile()))

	tmpPath := path + ".tmp"
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), ".tmp file should not remain after successful save")
}

func TestSave_PackagesSortedByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1, ProlfileHash: "sha256:abc"},
		Packages: []prolfile.LockEntry{
			{Name: "zeta", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/z.tar.gz", Checksum: "sha256:zzz", Dependencies: []string{}},
			{Name: "alpha", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/a.tar.gz", Checksum: "sha256:aaa", Dependencies: []string{}},
			{Name: "mid", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/m.tar.gz", Checksum: "sha256:mmm", Dependencies: []string{}},
		},
	}

	require.NoError(t, Save(path, lf))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	posA := strings.Index(content, `name = "alpha"`)
	posM := strings.Index(content, `name = "mid"`)
	posZ := strings.Index(content, `name = "zeta"`)
	require.NotEqual(t, -1, posA)
	require.NotEqual(t, -1, posM)
	require.NotEqual(t, -1, posZ)
	assert.Less(t, posA, posM, "alpha must appear before mid")
	assert.Less(t, posM, posZ, "mid must appear before zeta")
}

func TestSave_Deterministic(t *testing.T) {
	dir := t.TempDir()

	lf := sampleLockFile()
	path1 := filepath.Join(dir, "a.lock")
	path2 := filepath.Join(dir, "b.lock")

	require.NoError(t, Save(path1, lf))
	require.NoError(t, Save(path2, lf))

	data1, _ := os.ReadFile(path1)
	data2, _ := os.ReadFile(path2)
	assert.Equal(t, data1, data2, "Save must produce identical output")
}

func TestSave_KeyOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	require.NoError(t, Save(path, sampleLockFile()))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// For each [[package]] block, verify key ordering matches the spec:
	// name, version, source, url, checksum, dependencies
	keys := []string{"name", "version", "source", "url", "checksum", "dependencies"}

	// Find the first [[package]] block and verify order within it.
	pkgStart := strings.Index(content, "[[package]]")
	require.NotEqual(t, -1, pkgStart)
	block := content[pkgStart:]

	var positions []int
	for _, key := range keys {
		pos := strings.Index(block, key+" = ")
		require.NotEqual(t, -1, pos, "key %q not found in package block", key)
		positions = append(positions, pos)
	}
	for i := 1; i < len(positions); i++ {
		assert.Less(t, positions[i-1], positions[i],
			"key %q must appear before %q", keys[i-1], keys[i])
	}
}

func TestSave_HeaderComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	require.NoError(t, Save(path, sampleLockFile()))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "# This file is auto-generated by prolm."),
		"lockfile must start with auto-generated comment")
	assert.Contains(t, content, "Commit this file to version control")
}

func TestSave_EmptyPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1, ProlfileHash: "sha256:abc"},
	}

	require.NoError(t, Save(path, lf))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.True(t, IsEmpty(loaded))
}

func TestSave_SignatureOmittedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1, ProlfileHash: "sha256:abc"},
		Packages: []prolfile.LockEntry{
			{Name: "pkg", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/pkg.tar.gz", Checksum: "sha256:aaa", Dependencies: []string{}},
		},
	}

	require.NoError(t, Save(path, lf))
	data, _ := os.ReadFile(path)
	assert.NotContains(t, string(data), "signature", "signature must be omitted when empty")
}

func TestSave_SignatureIncludedWhenSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1, ProlfileHash: "sha256:abc"},
		Packages: []prolfile.LockEntry{
			{Name: "pkg", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/pkg.tar.gz", Checksum: "sha256:aaa", Dependencies: []string{}, Signature: "sig:xyz"},
		},
	}

	require.NoError(t, Save(path, lf))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), `signature = "sig:xyz"`)
}

func TestSave_ErrorOnWriteToNonExistentDir(t *testing.T) {
	// The tmp write fails immediately because the directory does not exist.
	// No .tmp file is created, so there is nothing to clean up here.
	// The rename-failure cleanup path (os.Remove after failed os.Rename) is
	// exercised implicitly but cannot be triggered portably without OS mocking.
	err := Save("/nonexistent-dir/Prolfile.lock", sampleLockFile())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing temporary lockfile")
}

func TestSave_NilDependenciesBecomesEmptySlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1, ProlfileHash: "sha256:abc"},
		Packages: []prolfile.LockEntry{
			{Name: "pkg", Version: "1.0.0", Source: "swi-pack-index", URL: "https://example.com/pkg.tar.gz", Checksum: "sha256:aaa", Dependencies: nil},
		},
	}

	require.NoError(t, Save(path, lf))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "dependencies = []", "nil dependencies must be serialized as empty array")
}

// --- IsEmpty tests ---

func TestIsEmpty_Nil(t *testing.T) {
	assert.True(t, IsEmpty(nil))
}

func TestIsEmpty_NoPackages(t *testing.T) {
	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1},
	}
	assert.True(t, IsEmpty(lf))
}

func TestIsEmpty_WithPackages(t *testing.T) {
	assert.False(t, IsEmpty(sampleLockFile()))
}

// --- containsConflictMarkers tests ---

func TestContainsConflictMarkers_NoMarkers(t *testing.T) {
	assert.False(t, containsConflictMarkers([]byte(validLockTOML)))
}

func TestContainsConflictMarkers_InlineValuesNotDetected(t *testing.T) {
	// A value containing "=======" mid-line should NOT trigger detection,
	// because conflict markers only appear at the start of lines.
	content := `[meta]
lock_version = 1
prolfile_hash = "sha256:======="
`
	assert.False(t, containsConflictMarkers([]byte(content)))
}

func TestContainsConflictMarkers_SeparatorExactMatch(t *testing.T) {
	// The ======= separator must be the entire line content (trimmed).
	// Lines that merely start with "=======" but have additional text
	// should NOT be detected as conflict markers.
	assert.False(t, containsConflictMarkers([]byte("======= some text\n")))
	assert.False(t, containsConflictMarkers([]byte("========\n")))
	// Exact separator with optional whitespace should still be detected.
	assert.True(t, containsConflictMarkers([]byte("=======\n")))
	assert.True(t, containsConflictMarkers([]byte("=======  \n")))
	assert.True(t, containsConflictMarkers([]byte("  =======\n")))
}

// --- Error type tests ---

func TestLockVersionTooNewError_Message(t *testing.T) {
	err := &LockVersionTooNewError{Found: 5, Supported: 1}
	assert.Contains(t, err.Error(), "lock_version 5")
	assert.Contains(t, err.Error(), "version 1")
	assert.Contains(t, err.Error(), "self-update")
}

func TestGitConflictError_Message(t *testing.T) {
	err := &GitConflictError{Path: "/tmp/Prolfile.lock"}
	assert.Contains(t, err.Error(), "merge conflict")
	assert.Contains(t, err.Error(), "/tmp/Prolfile.lock")
	assert.Contains(t, err.Error(), "prolm install")
}

func TestLoad_GitConflictError_ErrorsAs(t *testing.T) {
	// Load returns *GitConflictError directly so errors.As can match it.
	dir := t.TempDir()
	path := writeLock(t, dir, "<<<<<<< HEAD\nfoo\n")

	_, err := Load(path)
	require.Error(t, err)

	var gErr *GitConflictError
	assert.True(t, errors.As(err, &gErr))
}

// TestSave_ConcurrentWrites verifies that simultaneous Save calls targeting
// the same path do not corrupt the lockfile (SEC-7 / QUALITY-003).
// Run with -race to catch data races in the atomic rename path.
func TestSave_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Prolfile.lock")

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{LockVersion: 1},
		Packages: []prolfile.LockEntry{
			{
				Name:     "clpfd",
				Version:  "1.4.3",
				Source:   "swi-pack-index",
				URL:      "https://example.com/clpfd.tar.gz",
				Checksum: "sha256:abc123",
			},
		},
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = Save(path, lf)
		}()
	}
	wg.Wait()

	// The final file must be valid, well-formed TOML — not a partial write.
	got, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.Meta.LockVersion)
	require.Len(t, got.Packages, 1)
	assert.Equal(t, "clpfd", got.Packages[0].Name)
}
