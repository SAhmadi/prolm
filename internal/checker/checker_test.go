package checker

import (
	"context"
	"testing"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRT is a minimal Runtime used only by Checker.Check.
type fakeRT struct {
	gotFiles []string
	gotDeps  []string
}

func (f *fakeRT) Name() string                                          { return "fake" }
func (f *fakeRT) Detect() (*runtime.RuntimeInfo, error)                 { return nil, nil }
func (f *fakeRT) BuildRunArgs(string, []string, []string, string) []string { return nil }
func (f *fakeRT) BuildTestArgs([]string, []string, []string) []string   { return nil }
func (f *fakeRT) BuildCheckArgs(files, deps []string) []string {
	f.gotFiles = append([]string(nil), files...)
	f.gotDeps = append([]string(nil), deps...)
	return []string{"-t", "halt"}
}
func (f *fakeRT) Exec([]string) error { return nil }

func TestChecker_EmptyFilesSkipsExec(t *testing.T) {
	execCalled := false
	c := &Checker{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		execCalled = true
		return nil, nil, nil
	}}
	res, err := c.Check(context.Background(), "/fake/swipl", nil, nil, &fakeRT{}, Options{})
	require.NoError(t, err)
	assert.False(t, execCalled)
	assert.Empty(t, res.Diagnostics)
}

func TestChecker_ParsesStderrDiagnostics(t *testing.T) {
	rt := &fakeRT{}
	c := &Checker{Exec: func(_ context.Context, _ string, _ ...string) ([]byte, []byte, error) {
		return nil,
			[]byte("Warning: /tmp/a.pl:1:1: Singleton variables: [X]\nERROR: /tmp/a.pl:2: boom\n"),
			nil
	}}
	res, err := c.Check(context.Background(), "/fake/swipl",
		[]string{"/tmp/a.pl"}, []string{"/store/dep"}, rt, Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{"/tmp/a.pl"}, rt.gotFiles)
	assert.Equal(t, []string{"/store/dep"}, rt.gotDeps)
	assert.Len(t, res.Warnings(), 1)
	assert.Len(t, res.Errors(), 1)
}

func TestChecker_TolerantToNonZeroExit(t *testing.T) {
	c := &Checker{Exec: func(context.Context, string, ...string) ([]byte, []byte, error) {
		return nil, []byte("Warning: /tmp/a.pl:1: ok\n"), assert.AnError
	}}
	res, err := c.Check(context.Background(), "/fake/swipl",
		[]string{"/tmp/a.pl"}, nil, &fakeRT{}, Options{})
	require.NoError(t, err)
	assert.Len(t, res.Warnings(), 1)
}
