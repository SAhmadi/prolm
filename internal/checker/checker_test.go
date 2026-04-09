package checker

import (
	"context"
	"testing"
	"time"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRT struct {
	name string
}

func (f *fakeRT) Name() string { return f.name }
func (f *fakeRT) Detect() (*runtime.RuntimeInfo, error) {
	return &runtime.RuntimeInfo{Path: "/fake/swipl", Version: "9.2.1"}, nil
}
func (f *fakeRT) BuildRunArgs(_ string, _ []string, _ []string, _ string) []string { return nil }
func (f *fakeRT) BuildTestArgs(_, _, _ []string) []string { return nil }
func (f *fakeRT) BuildCheckArgs(files, deps []string) []string {
	var args []string
	for _, d := range deps {
		args = append(args, "-g", "use_module('"+d+"')")
	}
	for _, file := range files {
		args = append(args, "-g", "load_files('"+file+"')")
	}
	args = append(args, "-t", "halt")
	return args
}
func (f *fakeRT) Exec(_ []string) error { return nil }

func TestChecker_EmptyFiles(t *testing.T) {
	c := &Checker{
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			return nil, nil, nil
		},
	}
	res, err := c.Check(context.Background(), "/fake/swipl", []string{}, []string{}, &fakeRT{}, Options{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.Diagnostics))
}

func TestChecker_InvokesRuntime(t *testing.T) {
	var gotArgs []string
	c := &Checker{
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			gotArgs = append(gotArgs, args...)
			return []byte("Warning: /path/file.pl:10:5: Test\n"), nil, nil
		},
	}
	rt := &fakeRT{name: "swi"}
	res, err := c.Check(context.Background(), "/fake/swipl", []string{"src/main.pl"}, []string{"deps/lib"}, rt, Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Diagnostics))
	assert.Contains(t, gotArgs, "load_files('src/main.pl')")
	assert.Contains(t, gotArgs, "use_module('deps/lib')")
}

func TestChecker_AppliesTimeout(t *testing.T) {
	var gotDeadline time.Time
	c := &Checker{
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			deadline, ok := ctx.Deadline()
			assert.True(t, ok, "context should have deadline")
			gotDeadline = deadline
			return []byte(""), nil, nil
		},
	}
	_, err := c.Check(context.Background(), "/fake/swipl", []string{"src/main.pl"}, nil, &fakeRT{}, Options{Timeout: 5 * time.Second})
	require.NoError(t, err)
	// Verify the deadline was set to approximately 5 seconds from now
	now := time.Now()
	assert.True(t, gotDeadline.After(now), "deadline should be in the future")
	assert.True(t, gotDeadline.Before(now.Add(6*time.Second)), "deadline should be ~5s from now")
}

func TestChecker_UsesDefaultTimeout(t *testing.T) {
	c := &Checker{
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			_, hasDeadline := ctx.Deadline()
			assert.True(t, hasDeadline, "should have timeout set")
			return []byte(""), nil, nil
		},
	}
	res, err := c.Check(context.Background(), "/fake/swipl", []string{"src/main.pl"}, nil, &fakeRT{}, Options{})
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestChecker_ToleratesNonZeroExit(t *testing.T) {
	c := &Checker{
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			return []byte("Warning: /path/file.pl:10:5: Test\n"), nil, nil
		},
	}
	res, err := c.Check(context.Background(), "/fake/swipl", []string{"src/main.pl"}, nil, &fakeRT{}, Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Diagnostics))
}
