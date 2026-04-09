package testrunner

import (
	"context"
	"errors"
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

func (f *fakeRT) Name() string                                          { return "swi" }
func (f *fakeRT) Detect() (*runtime.RuntimeInfo, error)                 { return nil, nil }
func (f *fakeRT) BuildRunArgs(string, []string, []string, string) []string { return nil }
func (f *fakeRT) BuildTestArgs(files, deps, flags []string) []string {
	f.gotFiles = files
	f.gotDeps = deps
	f.gotFlags = flags
	return []string{"-g", "run_tests", "-t", "halt"}
}
func (f *fakeRT) BuildCheckArgs([]string, []string) []string { return nil }
func (f *fakeRT) Exec([]string) error                        { return nil }

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
	res, err := r.Run(context.Background(), "/bin/swipl", []string{"a_test.pl"}, nil, &fakeRT{}, nil, Options{})
	// Non-zero exit from swipl is expected when tests fail; we surface the result,
	// not an error. The caller inspects Failed/Errors to decide the exit code.
	require.NoError(t, err)
	assert.NotNil(t, res)
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
