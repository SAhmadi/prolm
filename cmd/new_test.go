package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingArgs_ShowsUsage(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	out, err := runSmokeCommandExpectError(t, binPath, workspace, env, "new")
	require.Error(t, err)
	assert.Contains(t, out, "accepts 1 arg")
	assert.Contains(t, out, "prolm new <name>")
	assert.Contains(t, out, "Usage:")
}

func TestNew_InvalidName_IsRejected(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	out, err := runSmokeCommandExpectError(t, binPath, workspace, env, "new", "../evil")
	require.Error(t, err)
	assert.Contains(t, out, "path separator")
}

func TestNew_HappyPath_CreatesExpectedFiles(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	out := runSmokeCommand(t, binPath, workspace, env, "new", "my-project")
	assert.Contains(t, out, `Created project "my-project"`)

	projectDir := filepath.Join(workspace, "my-project")
	_, err := os.Stat(projectDir)
	require.NoError(t, err, "project directory should exist")
	_, err = os.Stat(filepath.Join(projectDir, "Prolfile.toml"))
	require.NoError(t, err, "Prolfile.toml should exist")
	_, err = os.Stat(filepath.Join(projectDir, "src", "main.pl"))
	require.NoError(t, err, "src/main.pl should exist")
}

func TestNew_WithRuntimeFlag_WritesRuntimeToManifest(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	runSmokeCommand(t, binPath, workspace, env, "new", "scryer-project", "--runtime", "scryer")

	data, readErr := os.ReadFile(filepath.Join(workspace, "scryer-project", "Prolfile.toml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `runtime = "scryer"`)
}

func TestNew_Help_ExplainsTemplateStatus(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	out := runSmokeCommand(t, binPath, workspace, env, "new", "--help")
	outLower := strings.ToLower(out)

	assert.Contains(t, outLower, "templates:")
	assert.Contains(t, out, "app      Available in Phase 1 / Phase 1.5")
	assert.Contains(t, out, "library  Planned for Phase 2")
	assert.Contains(t, out, "cli      Planned for Phase 2")
	assert.Contains(t, out, "--runtime string")
	assert.Contains(t, out, "project template: app (Phase 1 / 1.5) | library (Phase 2) | cli (Phase 2)")
}
