package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetCheckCmd(t *testing.T) {
	t.Helper()
	checkCmd.ResetFlags()
	checkCmd.Flags().Bool("strict", false, "")
	checkCmd.Flags().Duration("timeout", 30*time.Second, "")
}

func withCheckCmdRunner(t *testing.T, r *checkCmdRunner) {
	t.Helper()
	orig := defaultCheckCmdRunner
	defaultCheckCmdRunner = r
	t.Cleanup(func() { defaultCheckCmdRunner = orig })
}

func newFakeCheckRunner(storeDir string, fr *fakeRuntime, stderr string, discoverFiles []string) *checkCmdRunner {
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
			return nil, []byte(stderr), nil
		},
	}
}

func TestExecute_Check_CleanProjectSucceeds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	src := filepath.Join(dir, "src", "main.pl")
	require.NoError(t, os.WriteFile(src, []byte(":- module(main,[]).\n"), 0644))

	fr := &fakeRuntime{}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, "", []string{src}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())
}

func TestExecute_Check_ErrorsFailNonZero(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	src := filepath.Join(dir, "src", "main.pl")
	require.NoError(t, os.WriteFile(src, []byte(""), 0644))

	fr := &fakeRuntime{}
	stderr := "ERROR: /tmp/main.pl:3:5: Syntax error: Operator expected\n"
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, stderr, []string{src}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check failed")
}

func TestExecute_Check_WarningsOnlyPassUnlessStrict(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	src := filepath.Join(dir, "src", "main.pl")
	require.NoError(t, os.WriteFile(src, []byte(""), 0644))

	fr := &fakeRuntime{}
	stderr := "Warning: /tmp/main.pl:10:1: Singleton variables: [X]\n"
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, stderr, []string{src}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())

	// Strict mode: warnings become failures.
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, &fakeRuntime{}, stderr, []string{src}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check", "--strict"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check failed")
}

func TestExecute_Check_NoFilesWarnsButSucceeds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, &fakeRuntime{}, "", nil))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	require.NoError(t, Execute())
}

func TestExecute_Check_RuntimeNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	src := filepath.Join(dir, "src", "main.pl")
	require.NoError(t, os.WriteFile(src, []byte(""), 0644))

	fr := &fakeRuntime{detectErr: &runtime.ErrRuntimeNotFound{Runtime: "swi"}}
	withCheckCmdRunner(t, newFakeCheckRunner(storeDir, fr, "", []string{src}))
	resetCheckCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"check"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "swipl not found")
}
