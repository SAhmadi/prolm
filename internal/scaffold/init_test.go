package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- sanitizeDirName tests ---

func TestSanitizeDirName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-app", "my-app"},
		{"My App", "my-app"},
		{"Hello_World", "hello_world"},
		{"my-app-123", "my-app-123"},
		{"UPPERCASE", "uppercase"},
		{"foo@bar#baz", "foo-bar-baz"},
		{"---trimmed---", "trimmed"},
		{"", ""},
		{"123-start", "123-start"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeDirName(tt.input))
		})
	}
}

// --- InitProject tests ---

func TestInitProject_Yes_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// Rename to a valid package name.
	projectDir := filepath.Join(dir, "my-app")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(projectDir, "Prolfile.toml"))
	assert.NoError(t, err, "Prolfile.toml should exist")

	_, err = os.Stat(filepath.Join(projectDir, "Prolfile.lock"))
	assert.NoError(t, err, "Prolfile.lock should exist")
}

func TestInitProject_Yes_ProlfileContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)

	assert.Equal(t, "test-project", pf.Package.Name)
	assert.Equal(t, "0.1.0", pf.Package.Version)
	assert.Equal(t, "src/main.pl", pf.Package.Entry)
	assert.Equal(t, "swi", pf.Package.Runtime)
	assert.Equal(t, 1, pf.Meta.ProlfileVersion)
}

func TestInitProject_Yes_NameSanitization(t *testing.T) {
	dir := t.TempDir()
	// Create a directory with a name that needs sanitizing.
	projectDir := filepath.Join(dir, "My Cool App")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "my-cool-app", pf.Package.Name)
}

func TestInitProject_Yes_InvalidDirName(t *testing.T) {
	dir := t.TempDir()
	// Directory name starting with digit is not a valid package name.
	projectDir := filepath.Join(dir, "123invalid")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project name")
}

func TestInitProject_ErrorIfProlfileExists(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "existing")
	require.NoError(t, os.Mkdir(projectDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "Prolfile.toml"), []byte("[package]\n"), 0644))

	err := InitProject(projectDir, false, true, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInitProject_ScanFindsDeps(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "scan-test")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	writePl(t, projectDir, "src/main.pl", `:- use_module(library(clpfd)).
:- use_module(library(lists)).
main :- true.
`)

	err := InitProject(projectDir, true, true, nil)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)

	assert.Equal(t, "*", pf.Dependencies["clpfd"])
	assert.NotContains(t, pf.Dependencies, "lists") // built-in
}

func TestInitProject_ScanFalse(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "no-scan")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	writePl(t, projectDir, "src/main.pl", `:- use_module(library(clpfd)).
`)

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)

	assert.Empty(t, pf.Dependencies)
}

func TestInitProject_EmptyLock(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "lock-test")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(projectDir, "Prolfile.lock"))
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestInitProject_Interactive(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "interactive")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	oldOut := ui.Out
	defer func() { ui.Out = oldOut }()
	ui.Out = &bytes.Buffer{}

	input := strings.NewReader("custom-name\n0.2.0\nsrc/app.pl\nscryer\n")
	err := InitProject(projectDir, false, false, input)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "custom-name", pf.Package.Name)
	assert.Equal(t, "0.2.0", pf.Package.Version)
	assert.Equal(t, "src/app.pl", pf.Package.Entry)
	assert.Equal(t, "scryer", pf.Package.Runtime)
}

func TestInitProject_Interactive_Defaults(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "defaults-test")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	oldOut := ui.Out
	defer func() { ui.Out = oldOut }()
	ui.Out = &bytes.Buffer{}

	input := strings.NewReader("\n\n\n\n")
	err := InitProject(projectDir, false, false, input)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "defaults-test", pf.Package.Name)
	assert.Equal(t, "0.1.0", pf.Package.Version)
	assert.Equal(t, "src/main.pl", pf.Package.Entry)
	assert.Equal(t, "swi", pf.Package.Runtime)
}

func TestInitProject_Interactive_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "bad-version")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	oldOut := ui.Out
	defer func() { ui.Out = oldOut }()
	ui.Out = &bytes.Buffer{}

	// First version is invalid, second is valid.
	input := strings.NewReader("test-app\nnot-a-version\n0.1.0\nsrc/main.pl\nswi\n")
	err := InitProject(projectDir, false, false, input)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", pf.Package.Version)
}

func TestInitProject_Interactive_InvalidEntry(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "bad-entry")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	oldOut := ui.Out
	defer func() { ui.Out = oldOut }()
	ui.Out = &bytes.Buffer{}

	// First entry is invalid (no .pl), second is valid.
	input := strings.NewReader("test-app\n0.1.0\n../../evil.py\nsrc/main.pl\nswi\n")
	err := InitProject(projectDir, false, false, input)
	require.NoError(t, err)

	pf, err := manifest.Load(filepath.Join(projectDir, "Prolfile.toml"), manifest.LoadOptions{})
	require.NoError(t, err)
	assert.Equal(t, "src/main.pl", pf.Package.Entry)
}

func TestInitProject_DoesNotCreateSourceFiles(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "no-source")
	require.NoError(t, os.Mkdir(projectDir, 0755))

	err := InitProject(projectDir, false, true, nil)
	require.NoError(t, err)

	// InitProject should NOT create src/, tests/, .gitignore, README.md, etc.
	_, err = os.Stat(filepath.Join(projectDir, "src"))
	assert.True(t, os.IsNotExist(err), "src/ should not be created by init")

	_, err = os.Stat(filepath.Join(projectDir, "tests"))
	assert.True(t, os.IsNotExist(err), "tests/ should not be created by init")

	_, err = os.Stat(filepath.Join(projectDir, ".gitignore"))
	assert.True(t, os.IsNotExist(err), ".gitignore should not be created by init")

	_, err = os.Stat(filepath.Join(projectDir, "README.md"))
	assert.True(t, os.IsNotExist(err), "README.md should not be created by init")
}
