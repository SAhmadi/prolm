package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRuntime is a stub Runtime that records what would be executed.
type fakeRuntime struct {
	name        string
	detectErr   error
	info        *runtime.RuntimeInfo
	gotRunEntry string
	gotRunDeps  []string
	gotRunFlags []string
	gotRunGoal  string
	gotExecArgs []string
	execErr     error
}

func (f *fakeRuntime) Name() string { return f.name }
func (f *fakeRuntime) Detect() (*runtime.RuntimeInfo, error) {
	if f.detectErr != nil {
		return nil, f.detectErr
	}
	if f.info == nil {
		f.info = &runtime.RuntimeInfo{Path: "/fake/swipl", Version: "9.2.1"}
	}
	return f.info, nil
}
func (f *fakeRuntime) BuildRunArgs(entry string, deps, flags []string, goal string) []string {
	f.gotRunEntry = entry
	f.gotRunDeps = append([]string(nil), deps...)
	f.gotRunFlags = append([]string(nil), flags...)
	f.gotRunGoal = goal
	return []string{"BUILT", entry, goal}
}
func (f *fakeRuntime) BuildTestArgs(_, _, _ []string) []string  { return nil }
func (f *fakeRuntime) BuildCheckArgs(_, _ []string) []string    { return nil }
func (f *fakeRuntime) Exec(args []string) error {
	f.gotExecArgs = append([]string(nil), args...)
	return f.execErr
}

// withTestRunRunner swaps defaultRunRunner for the duration of the test.
func withTestRunRunner(t *testing.T, runner *runRunner) {
	t.Helper()
	orig := defaultRunRunner
	defaultRunRunner = runner
	t.Cleanup(func() { defaultRunRunner = orig })
}

// resetRunCmd clears any flag state left by previous run-command tests.
func resetRunCmd(t *testing.T) *runRunner {
	t.Helper()
	runCmd.ResetFlags()
	runCmd.Flags().String("goal", "main", "Prolog goal to call after loading the entry")
	return defaultRunRunner
}

// writeRunProject creates a Prolfile.toml + Prolfile.lock + src/main.pl tree
// in dir, with the given runtime name and dependency list (deps already
// "installed" in storeDir).
func writeRunProject(t *testing.T, dir, storeDir string) {
	t.Helper()
	manifest := `[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name = "test-project"
version = "0.1.0"
entry = "src/main.pl"
runtime = "swi"

[dependencies]
clpfd = "*"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.toml"), []byte(manifest), 0644))

	lock := `[meta]
lock_version = 1
prolfile_hash = "sha256:deadbeef"

[[package]]
name = "clpfd"
version = "1.4.3"
source = "swi-pack-index"
url = "https://example.invalid/clpfd-1.4.3.tar.gz"
checksum = "sha256:abc"
dependencies = []
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Prolfile.lock"), []byte(lock), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.pl"), []byte(":- module(main, []).\n"), 0644))

	// Pre-populate the store: ~/.prolm-style layout — <storeDir>/<name>/<version>.
	require.NoError(t, os.MkdirAll(filepath.Join(storeDir, "clpfd", "1.4.3"), 0755))
}

func newFakeRunner(storeDir string, fr *fakeRuntime) *runRunner {
	return &runRunner{
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
	}
}

func TestExecute_Run_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withTestRunRunner(t, newFakeRunner(storeDir, fr))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run"})
	require.NoError(t, Execute())

	assert.Equal(t, filepath.Join(dir, "src", "main.pl"), fr.gotRunEntry)
	assert.Equal(t, []string{filepath.Join(storeDir, "clpfd", "1.4.3")}, fr.gotRunDeps)
	assert.Equal(t, "main", fr.gotRunGoal)
	assert.Equal(t, []string{"BUILT", fr.gotRunEntry, "main"}, fr.gotExecArgs)
}

func TestExecute_Run_NoLockfile(t *testing.T) {
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

	withTestRunRunner(t, newFakeRunner(filepath.Join(dir, "store"), &fakeRuntime{}))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Prolfile.lock not found")
}

func TestExecute_Run_DepNotInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	// Remove the pre-populated dep so the store-check fails.
	require.NoError(t, os.RemoveAll(filepath.Join(storeDir, "clpfd")))

	withTestRunRunner(t, newFakeRunner(storeDir, &fakeRuntime{}))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clpfd@1.4.3")
	assert.Contains(t, err.Error(), "not installed")
}

func TestExecute_Run_PositionalFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	other := filepath.Join(dir, "src", "other.pl")
	require.NoError(t, os.WriteFile(other, []byte(":- module(other, []).\n"), 0644))

	fr := &fakeRuntime{}
	withTestRunRunner(t, newFakeRunner(storeDir, fr))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run", "src/other.pl"})
	require.NoError(t, Execute())
	assert.Equal(t, other, fr.gotRunEntry)
}

func TestExecute_Run_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	withTestRunRunner(t, newFakeRunner(storeDir, &fakeRuntime{}))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run", "src/nope.pl"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExecute_Run_GoalOverride(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withTestRunRunner(t, newFakeRunner(storeDir, fr))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run", "--goal", "diagnose"})
	require.NoError(t, Execute())
	assert.Equal(t, "diagnose", fr.gotRunGoal)
}

func TestExecute_Run_ProlgArgsAfterDash(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{}
	withTestRunRunner(t, newFakeRunner(storeDir, fr))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run", "--", "--mode", "interactive"})
	require.NoError(t, Execute())
	// Last two args should be the forwarded prolog args.
	require.GreaterOrEqual(t, len(fr.gotExecArgs), 2)
	assert.Equal(t, []string{"--mode", "interactive"}, fr.gotExecArgs[len(fr.gotExecArgs)-2:])
}

func TestExecute_Run_RuntimeNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)

	fr := &fakeRuntime{detectErr: &runtime.ErrRuntimeNotFound{Runtime: "swi"}}
	withTestRunRunner(t, newFakeRunner(storeDir, fr))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "swipl not found")
}

func TestExecute_Run_ScriptNameNotSupported(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	storeDir := filepath.Join(dir, "store")
	writeRunProject(t, dir, storeDir)
	// Append a [scripts] section.
	manifestPath := filepath.Join(dir, "Prolfile.toml")
	body, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	body = append(body, []byte("\n[scripts]\nstart = \"prolm run src/main.pl\"\n")...)
	require.NoError(t, os.WriteFile(manifestPath, body, 0644))

	withTestRunRunner(t, newFakeRunner(storeDir, &fakeRuntime{}))
	resetRunCmd(t)
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"run", "start"})
	err = Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scripts")
}
