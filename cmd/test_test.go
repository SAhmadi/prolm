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

// resetTestCmd clears any leftover flag state from a prior test.
func resetTestCmd(t *testing.T) {
	t.Helper()
	testCmd.ResetFlags()
	testCmd.Flags().String("filter", "", "")
	testCmd.Flags().Bool("verbose", false, "")
	testCmd.Flags().Duration("timeout", 30*time.Second, "")
}

func withTestCmdRunner(t *testing.T, r *testCmdRunner) {
	t.Helper()
	orig := defaultTestCmdRunner
	defaultTestCmdRunner = r
	t.Cleanup(func() { defaultTestCmdRunner = orig })
}

func newFakeTestRunner(storeDir string, fr *fakeRuntime, stdout string, discoverFiles []string) *testCmdRunner {
	return &testCmdRunner{
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

func TestExecute_Test_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	testFile := filepath.Join(dir, "tests", "main_test.pl")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte(":- begin_tests(main).\n:- end_tests(main).\n"), 0644))

	fr := &fakeRuntime{}
	withTestCmdRunner(t, newFakeTestRunner(storeDir, fr, "% All 2 tests passed\n", []string{testFile}))
	resetTestCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"test"})
	require.NoError(t, Execute())
}

func TestExecute_Test_FailingExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	testFile := filepath.Join(dir, "tests", "bad_test.pl")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte(""), 0644))

	fr := &fakeRuntime{}
	withTestCmdRunner(t, newFakeTestRunner(storeDir, fr,
		"% test main:truth: failed\n% 1 test failed out of 2\n",
		[]string{testFile}))
	resetTestCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"test"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestExecute_Test_NoFilesWarnsButSucceeds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withTestCmdRunner(t, newFakeTestRunner(storeDir, fr, "", nil))
	resetTestCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"test"})
	require.NoError(t, Execute())
}

func TestExecute_Test_PositionalFileNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withTestCmdRunner(t, newFakeTestRunner(storeDir, fr, "", nil))
	resetTestCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"test", "tests/nope.pl"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExecute_Test_RejectsBadFilter(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	testFile := filepath.Join(dir, "tests", "main_test.pl")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte(""), 0644))

	fr := &fakeRuntime{}
	withTestCmdRunner(t, newFakeTestRunner(storeDir, fr, "", []string{testFile}))
	resetTestCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"test", "--filter", "bad space"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filter")
}

