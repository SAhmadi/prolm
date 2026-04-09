package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetCheckCmd(t *testing.T) {
	t.Helper()
	checkCmd.ResetFlags()
	checkCmd.Flags().Bool("strict", false, "")
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
