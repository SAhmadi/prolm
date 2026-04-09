package checker

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/prolm/prolm/internal/runtime"
)

// defaultTimeout bounds a single swipl check invocation when the caller does
// not supply one. Matches the 30s convention used by testrunner.
const defaultTimeout = 30 * time.Second

// Options controls Checker behaviour.
type Options struct {
	Timeout time.Duration
}

// ExecFunc is the seam used to run the runtime binary. Tests inject a stub so
// the checker can be unit-tested without swipl installed.
type ExecFunc func(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)

// DefaultExec runs name+args via os/exec and returns stdout, stderr, and any
// non-zero-exit error. Non-zero exit is NOT fatal — swipl exits non-zero on
// warnings that have nothing to say about correctness; the caller inspects the
// parsed CheckResult to make the verdict.
func DefaultExec(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// Checker orchestrates static analysis. Exec is swapped in tests.
type Checker struct {
	Exec ExecFunc
}

// Check loads srcFiles into rt with deps available, captures the resulting
// swipl diagnostics, and returns them parsed.
//
// Behaviour:
//   - Empty srcFiles → returns an empty CheckResult without invoking swipl.
//   - The call is bounded by opts.Timeout (default 30s) via context.
//   - A non-zero exit from swipl is tolerated: the parsed output is still
//     returned so callers can render warnings/errors.
func (c *Checker) Check(
	parent context.Context,
	binary string,
	srcFiles []string,
	depPaths []string,
	rt runtime.Runtime,
	opts Options,
) (*CheckResult, error) {
	if len(srcFiles) == 0 {
		return &CheckResult{}, nil
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	args := rt.BuildCheckArgs(srcFiles, depPaths)

	execFn := c.Exec
	if execFn == nil {
		execFn = DefaultExec
	}

	stdout, stderr, _ := execFn(ctx, binary, args...)
	return Parse(string(stdout), string(stderr)), nil
}
