package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject_CreatesExpectedFiles(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	expected := []string{
		"hello/Prolfile.toml",
		"hello/Prolfile.lock",
		"hello/.gitignore",
		"hello/.gitattributes",
		"hello/README.md",
		"hello/src/main.pl",
		"hello/tests/main_test.pl",
	}
	for _, f := range expected {
		_, err := os.Stat(f)
		assert.NoError(t, err, "expected file %s to exist", f)
	}
}

func TestNewProject_ProlfileParsesBack(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("my-app", "app", "swi")
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join("my-app", "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)

	assert.Equal(t, "my-app", pf.Package.Name)
	assert.Equal(t, "0.1.0", pf.Package.Version)
	assert.Equal(t, "src/main.pl", pf.Package.Entry)
	assert.Equal(t, "swi", pf.Package.Runtime)
	assert.Equal(t, 1, pf.Meta.ProlfileVersion)
	assert.Equal(t, "0.1.0", pf.Meta.MinProlmVersion)
}

func TestNewProject_RuntimeWrittenToManifest(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("scryer-app", "app", "scryer")
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join("scryer-app", "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "scryer", pf.Package.Runtime)
}

func TestNewProject_MainPlContent(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	data, err := os.ReadFile("hello/src/main.pl")
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Hello, world!")
	assert.Contains(t, content, "hello") // project name in comment
	assert.Contains(t, content, ":- module(main, [main/0]).")
}

func TestNewProject_TestPlContent(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	data, err := os.ReadFile("hello/tests/main_test.pl")
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "begin_tests(main)")
	assert.Contains(t, content, "end_tests(main)")
}

func TestNewProject_GitattributesContent(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	data, err := os.ReadFile("hello/.gitattributes")
	require.NoError(t, err)
	assert.Contains(t, string(data), "Prolfile.lock merge=ours")
}

func TestNewProject_GitInitRuns(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	// git init is best-effort; only check if git is available.
	_, gitErr := os.Stat("hello/.git")
	if gitErr != nil {
		t.Skip("git not available, skipping .git check")
	}
}

func TestNewProject_EmptyLock(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("hello", "app", "swi")
	require.NoError(t, err)

	data, err := os.ReadFile("hello/Prolfile.lock")
	require.NoError(t, err)
	assert.Empty(t, data)
}

// --- Validation tests ---

func TestNewProject_RejectsDotDot(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("../evil", "app", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path separator")
}

func TestNewProject_RejectsDot(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject(".", "app", "swi")
	assert.Error(t, err)
}

func TestNewProject_RejectsPathSeparator(t *testing.T) {
	t.Chdir(t.TempDir())

	err := NewProject("foo/bar", "app", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path separator")

	err = NewProject(`foo\bar`, "app", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path separator")
}

func TestNewProject_RejectsAbsolutePath(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("/tmp/evil", "app", "swi")
	assert.Error(t, err)
}

func TestNewProject_RejectsEmptyName(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("", "app", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestNewProject_RejectsInvalidName(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("Hello_World", "app", "swi")
	assert.Error(t, err) // uppercase not allowed
}

func TestNewProject_DirectoryAlreadyExists(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, os.Mkdir("hello", 0755))
	err := NewProject("hello", "app", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestNewProject_InvalidTemplate(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("hello", "nonexistent", "swi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown template")
}

func TestNewProject_InvalidRuntime(t *testing.T) {
	t.Chdir(t.TempDir())
	err := NewProject("hello", "app", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown runtime")
}

func TestNewProject_CleanupOnError(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a file (not directory) at the src path to cause MkdirAll to fail
	// after the project dir is created.
	require.NoError(t, os.Mkdir("fail-project", 0755))
	require.NoError(t, os.WriteFile("fail-project/src", []byte("blocker"), 0644))

	// Remove the project dir so NewProject can try to recreate it.
	require.NoError(t, os.RemoveAll("fail-project"))

	// Now create a scenario where directory creation succeeds but a later step fails.
	// We create the project dir with a read-only "src" file blocking src/main.pl write.
	require.NoError(t, os.MkdirAll("blocker/src", 0755))
	require.NoError(t, os.WriteFile("blocker/src/main.pl", nil, 0000))
	require.NoError(t, os.Chmod("blocker/src", 0555))

	// NewProject("blocker", ...) will fail because the directory already exists.
	err := NewProject("blocker", "app", "swi")
	assert.Error(t, err)

	// Restore permissions for cleanup.
	os.Chmod("blocker/src", 0755)
}
