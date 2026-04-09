package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/checker"
	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetCheckCmd(t *testing.T) {
	t.Helper()
	checkCmd.ResetFlags()
	checkCmd.Flags().Bool("strict", false, "")
	checkCmd.Flags().Bool("no-deps", false, "")
}

func withCheckCmdRunner(t *testing.T, r *checkCmdRunner) {
	t.Helper()
	orig := defaultCheckCmdRunner
	defaultCheckCmdRunner = r
	t.Cleanup(func() { defaultCheckCmdRunner = orig })
}

func newFakeCheckRunner(storeDir string, fr *fakeRuntime, stdout string, discoverFiles []string) *checkCmdRunner {
	return &checkCmdRunner{
		storeDir: storeDir,
		newRuntime: func(name string) (runtime.Runtime, error) {
			if fr.name == "" {
				if name == "" {
					name = "swi"
				}
				fr.name = name
			}
			return fr, nil
		},
		discover: func(string) ([]string, error) { return discoverFiles, nil },
		exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
			return []byte(stdout), nil, nil
		},
	}
}

func TestExecute_Check_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	srcFile := filepath.Join(dir, "src", "main.pl")
	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, "", []string{srcFile}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())
}

func TestExecute_Check_NoSourceFilesWarns(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, "", nil))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())
}

func TestExecute_Check_ErrorExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	swiplOutput := "ERROR:   /path/file.pl:10:5: Undefined predicate foo/1\n"
	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, swiplOutput, []string{srcFile}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error")
}

func TestExecute_Check_WarningExitsZeroByDefault(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	swiplOutput := "Warning: /path/file.pl:10:5: Singleton variable 'X'\n"
	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, swiplOutput, []string{srcFile}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())
}

func TestExecute_Check_WarningExitsNonZeroWithStrict(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	swiplOutput := "Warning: /path/file.pl:10:5: Singleton variable 'X'\n"
	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, swiplOutput, []string{srcFile}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check", "--strict"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "warning")
}

func TestExecute_Check_RuntimeNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{detectErr: &runtime.ErrRuntimeNotFound{Runtime: "swi"}}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, "", nil))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "swipl")
}

func TestExecute_Check_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	manifestBody := `[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name = "p"
version = "0.1.0"
entry = "src/main.pl"
runtime = "swi"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.toml"), []byte(manifestBody), 0644))

	withCheckCmdRunner(t, newFakeCheckRunner(filepath.Join(dir, "store"), &fakeRuntime{}, "", nil))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prolfile.lock")
}

// SEC-017: --no-deps must pass an empty dep list to the checker so dependency
// modules are never loaded (and their initialization directives never executed).
func TestExecute_Check_NoDeps_PassesEmptyDepPaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	var gotArgs []string
	runner := &checkCmdRunner{
		storeDir:   storeDir,
		newRuntime: func(name string) (runtime.Runtime, error) { return &fakeRuntime{}, nil },
		discover:   func(string) ([]string, error) { return []string{srcFile}, nil },
		exec: func(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
			gotArgs = args
			return nil, nil, nil
		},
	}
	withCheckCmdRunner(t, runner)
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check", "--no-deps"})
	require.NoError(t, Execute())

	// With --no-deps the args must not contain any use_module call.
	for _, a := range gotArgs {
		assert.NotContains(t, a, "use_module", "--no-deps must suppress dep loading; got args: %v", gotArgs)
	}
}

// DRY-005 / QUALITY-023: --json output must be valid JSON decodable into
// checker.CheckResult. The old hand-rolled serialiser used fmt %q (Go string
// quoting, not JSON escaping) and would fail this round-trip.
func TestExecute_Check_JSONOutput_IsValid(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	// A warning message containing characters that differ between Go %q and
	// JSON encoding — a backslash triggers the difference.
	swiplOutput := "Warning: /path/file.pl:10:5: Singleton variable 'X\\Y'\n"
	fr := &fakeRuntime{}
	runner := newFakeCheckRunner(storeDir, fr, swiplOutput, []string{srcFile})
	withCheckCmdRunner(t, runner)

	var buf bytes.Buffer
	origOut := ui.Out
	ui.Out = &buf
	t.Cleanup(func() { ui.Out = origOut })

	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check", "--json"})
	_ = Execute() // may return non-zero; we only care about output validity

	var decoded checker.CheckResult
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err, "check --json output must be valid JSON; got: %s", buf.String())
	assert.Len(t, decoded.Diagnostics, 1)
}

// ERR-004 (cmd layer): when the exec function returns a spawn error,
// the check command must surface it — not silently report "No issues found".
func TestExecute_Check_SpawnError_SurfacedToCaller(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	srcFile := filepath.Join(dir, "src", "main.pl")

	spawnErr := fmt.Errorf("fork/exec /fake/swipl: no such file or directory")
	runner := &checkCmdRunner{
		storeDir:   storeDir,
		newRuntime: func(name string) (runtime.Runtime, error) { return &fakeRuntime{}, nil },
		discover:   func(string) ([]string, error) { return []string{srcFile}, nil },
		exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
			return nil, nil, spawnErr
		},
	}
	withCheckCmdRunner(t, runner)
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err, "spawn error must propagate to caller")
}
