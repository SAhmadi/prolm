package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	smokeBinaryOnce sync.Once
	smokeBinaryPath string
	smokeBinaryErr  error
)

func buildSmokeBinary(t *testing.T) string {
	t.Helper()

	smokeBinaryOnce.Do(func() {
		testBinDir := filepath.Join(os.TempDir(), "prolm-smoke-bin")
		if err := os.MkdirAll(testBinDir, 0755); err != nil {
			smokeBinaryErr = err
			return
		}

		name := "prolm-smoke"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		smokeBinaryPath = filepath.Join(testBinDir, name)

		cwd, err := os.Getwd()
		if err != nil {
			smokeBinaryErr = err
			return
		}
		repoRoot := filepath.Dir(cwd)

		cmd := exec.Command("go", "build", "-o", smokeBinaryPath, ".")
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			smokeBinaryErr = fmt.Errorf("building smoke-test binary: %w\n%s", err, strings.TrimSpace(string(output)))
		}
	})

	require.NoError(t, smokeBinaryErr)
	return smokeBinaryPath
}

func runSmokeCommand(t *testing.T, binPath, dir string, env []string, args ...string) string {
	t.Helper()

	combined, err := runSmokeCommandRaw(binPath, dir, env, args...)

	require.NoErrorf(t, err, "command failed: %s %s\noutput:\n%s", binPath, strings.Join(args, " "), combined)
	return combined
}

func runSmokeCommandExpectError(t *testing.T, binPath, dir string, env []string, args ...string) (string, error) {
	t.Helper()

	combined, err := runSmokeCommandRaw(binPath, dir, env, args...)
	require.Error(t, err, "command was expected to fail: %s %s", binPath, strings.Join(args, " "))
	return combined, err
}

func runSmokeCommandRaw(binPath, dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	combined := stdout.String()
	if stderr.Len() > 0 {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += stderr.String()
	}

	return combined, err
}

func TestSmoke_EndToEndPhase1Acceptance(t *testing.T) {
	if _, err := exec.LookPath("swipl"); err != nil {
		t.Fatalf("swipl is required for the Phase 1 smoke test: %v", err)
	}

	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(),
		"HOME="+homeDir,
		"NO_COLOR=1",
	)

	newOut := runSmokeCommand(t, binPath, workspace, env, "new", "hello")
	assert.Contains(t, newOut, `Created project "hello"`)

	projectDir := filepath.Join(workspace, "hello")
	_, err := os.Stat(filepath.Join(projectDir, "Prolfile.toml"))
	require.NoError(t, err)

	installOut := runSmokeCommand(t, binPath, projectDir, env, "install")
	assert.Contains(t, installOut, "Installed 0 packages")

	_, err = os.Stat(filepath.Join(projectDir, "Prolfile.lock"))
	require.NoError(t, err)

	runOut := runSmokeCommand(t, binPath, projectDir, env, "run")
	assert.Contains(t, runOut, "Hello, world!")

	testOut := runSmokeCommand(t, binPath, projectDir, env, "test")
	assert.Contains(t, testOut, "1 passed, 0 failed")

	checkOut := runSmokeCommand(t, binPath, projectDir, env, "check")
	assert.Contains(t, checkOut, "0 error(s), 0 warning(s)")
}

func TestSmoke_New_MissingName_ShowsArgumentError(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(),
		"HOME="+homeDir,
		"NO_COLOR=1",
	)

	out, err := runSmokeCommandExpectError(t, binPath, workspace, env, "new")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.NotZero(t, exitErr.ExitCode())

	assert.Contains(t, out, "accepts 1 arg")
	assert.Contains(t, out, "prolm new <name>")
	assert.Contains(t, out, "Usage:")
	assert.NotContains(t, out, "Created project")
}

func TestSmoke_Install_EmptyLockfile_UsesContractEncoding(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	runSmokeCommand(t, binPath, workspace, env, "new", "hello")
	projectDir := filepath.Join(workspace, "hello")
	runSmokeCommand(t, binPath, projectDir, env, "install")

	lockBytes, err := os.ReadFile(filepath.Join(projectDir, "Prolfile.lock"))
	require.NoError(t, err)
	lockContent := string(lockBytes)

	assert.Contains(t, lockContent, `prolfile_hash = "sha256:`)
	assert.NotContains(t, lockContent, `prolfile_hash = ""`)
	assert.NotContains(t, lockContent, "package = []")
}
