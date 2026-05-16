package testrunner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRT implements runtime.Runtime just enough for the runner.
type fakeRT struct {
	gotFiles, gotDeps, gotFlags []string
}

func (f *fakeRT) Name() string                                             { return "swi" }
func (f *fakeRT) Detect() (*runtime.RuntimeInfo, error)                    { return nil, nil }
func (f *fakeRT) BuildRunArgs(string, []string, []string, string) []string { return nil }
func (f *fakeRT) BuildTestArgs(files, deps, flags []string) []string {
	f.gotFiles = files
	f.gotDeps = deps
	f.gotFlags = flags
	return []string{"-g", "run_tests", "-t", "halt"}
}
func (f *fakeRT) BuildCheckArgs([]string, []string, []string) []string { return nil }
func (f *fakeRT) Exec([]string) error                                  { return nil }

func TestRunner_NoFiles_NoExec(t *testing.T) {
	called := false
	r := &Runner{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		called = true
		return nil, nil, nil
	}}
	res, err := r.Run(context.Background(), "/bin/swipl", nil, nil, &fakeRT{}, nil, Options{})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Total)
	assert.False(t, called, "exec should not run when there are no test files")
}

func TestRunner_ParsesCannedOutput(t *testing.T) {
	r := &Runner{Exec: func(_ context.Context, _ string, _ ...string) ([]byte, []byte, error) {
		return []byte("% All 2 tests passed\n"), nil, nil
	}}
	res, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Passed)
	assert.Equal(t, 0, res.Failed)
}

func TestRunner_ExecError(t *testing.T) {
	r := &Runner{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		return nil, []byte("boom"), errors.New("exit 1")
	}}
	_, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	// A non-zero exit with no parsed PlUnit output (Total==0) is a genuine
	// runtime failure — not a test failure — so it must be surfaced as an error
	// (BUG-013). The captured stderr is included so the user sees the root cause.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

// BUG-013: an exec error must be surfaced when res.Total==0, even when swipl
// writes something to stderr (e.g. "Cannot find boot file").
func TestRunner_ExecError_SurfacedWhenStderrPresent(t *testing.T) {
	r := &Runner{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		// swipl writes a message to stderr then exits non-zero — no PlUnit output.
		return nil, []byte("Cannot find boot file"), errors.New("exit status 1")
	}}
	_, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	require.Error(t, err, "exec error must be surfaced when Total==0, even if stderr is non-empty")
	assert.Contains(t, err.Error(), "Cannot find boot file",
		"surfaced error must include the captured stderr so the user sees the root cause")
}

// BUG-013: a non-zero exit with partial test output (some tests ran) must NOT
// return an error — failing tests cause non-zero exits and that is normal.
func TestRunner_ExecError_NotSurfacedWhenTestsRan(t *testing.T) {
	r := &Runner{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		// PlUnit ran, reported failures, and swipl exited 1.
		stdout := "% test main:truth: failed\n% 1 test failed out of 2\n"
		return []byte(stdout), []byte("some stderr"), errors.New("exit status 1")
	}}
	res, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	require.NoError(t, err, "non-zero exit with parsed test results must not be an error")
	assert.Equal(t, 1, res.Failed)
}

func TestRunner_RejectsUnsafeFilter(t *testing.T) {
	r := &Runner{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		t.Fatal("exec should not be called when filter is invalid")
		return nil, nil, nil
	}}
	_, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{Filter: "bad'; injected"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filter")
}

func TestRunner_FilterUsesPlUnitCurrentTestGoal(t *testing.T) {
	var gotArgs []string
	r := &Runner{Exec: func(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte("% [1/1] main:hello .................................. passed (0.001 sec)\n"), nil, nil
	}}
	res, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{Filter: "hello"})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Passed)

	joined := strings.Join(gotArgs, " ")
	assert.NotContains(t, joined, "set_test_options")
	assert.Contains(t, joined, "plunit:current_test")
	assert.Contains(t, joined, "run_tests(Tests)")
}

func TestRunner_DefaultTimeoutApplied(t *testing.T) {
	var gotCtx context.Context
	r := &Runner{Exec: func(ctx context.Context, _ string, _ ...string) ([]byte, []byte, error) {
		gotCtx = ctx
		return []byte("% All 0 tests passed\n"), nil, nil
	}}
	start := time.Now()
	_, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	require.NoError(t, err)
	require.NotNil(t, gotCtx)
	dl, ok := gotCtx.Deadline()
	require.True(t, ok, "default timeout should be applied")
	assert.True(t, dl.After(start), "deadline should be in the future")
}
