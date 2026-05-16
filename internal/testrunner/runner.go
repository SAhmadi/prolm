package testrunner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/prolm/prolm/internal/runtime"
)

// defaultTimeout is the wall-clock limit for a single swipl test invocation
// when the caller does not set Options.Timeout. Matches ROADMAP §1.12.
const defaultTimeout = 30 * time.Second

// Options controls runner behaviour.
type Options struct {
	Filter  string
	Verbose bool
	Timeout time.Duration
}

// ExecFunc is the seam used to run the runtime binary. Real callers use
// DefaultExec; tests inject a stub to avoid touching a real swipl.
type ExecFunc func(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)

// DefaultExec runs name+args via os/exec and returns stdout, stderr, and any
// non-zero-exit error.
func DefaultExec(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// Runner orchestrates test execution. The Exec seam is swapped in tests.
type Runner struct {
	Exec ExecFunc
}

// safeFilter restricts test filter strings to a conservative alphabet so they
// can be embedded in a Prolog goal without shell/atom escaping surprises.
// Validated at call time; anything outside this set is rejected (SEC-9).
var safeFilter = regexp.MustCompile(`^[A-Za-z0-9_:/.-]+$`)

// Run invokes the Prolog runtime against testFiles+depPaths and parses the
// output into a TestResult.
//
// Behaviour:
//   - Empty testFiles → returns a zero TestResult without invoking swipl.
//   - opts.Filter is validated against safeFilter; invalid input is a hard error.
//   - The call is bounded by opts.Timeout (default 30s) via context.
//   - Non-zero exit from swipl is NOT translated to a Go error; failing tests
//     routinely cause non-zero exits and callers decide the exit code from
//     res.Failed+res.Errors.
func (r *Runner) Run(
	parent context.Context,
	binary string,
	testFiles []string,
	depPaths []string,
	rt runtime.Runtime,
	flags []string,
	opts Options,
) (*TestResult, error) {
	if len(testFiles) == 0 {
		return &TestResult{}, nil
	}

	if opts.Filter != "" && !safeFilter.MatchString(opts.Filter) {
		return nil, fmt.Errorf("invalid --filter %q: only [A-Za-z0-9_:/.-] allowed", opts.Filter)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	args := rt.BuildTestArgs(testFiles, depPaths, flags)
	if opts.Filter != "" {
		args = replaceRunTestsGoal(args, filteredRunTestsGoal(opts.Filter))
	}

	execFn := r.Exec
	if execFn == nil {
		execFn = DefaultExec
	}

	start := time.Now()
	stdout, stderr, runErr := execFn(ctx, binary, args...)
	res := Parse(string(stdout), string(stderr))
	res.Duration = time.Since(start)

	// A timeout is a hard error (tests did not complete). A non-zero exit is
	// normal for failing tests and is reflected in res.Failed/res.Errors.
	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("test run timed out after %s", timeout)
	}
	// Surface runErr when no PlUnit results were parsed (Total==0). This catches
	// genuine runtime failures (e.g. "Cannot find boot file") that cause swipl
	// to exit non-zero without emitting any PlUnit output. The captured stderr
	// is included in the message so the user sees the root cause (BUG-013).
	//
	// We intentionally do NOT check len(stdout)==0 or len(stderr)==0 here:
	// a runtime that writes to stderr then exits non-zero with no PlUnit output
	// is a hard failure, regardless of whether stderr is empty.
	if runErr != nil && res.Total == 0 {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			return res, fmt.Errorf("running %s: %w", binary, runErr)
		}
		return res, fmt.Errorf("running %s: %w\n%s", binary, runErr, msg)
	}

	return res, nil
}

func filteredRunTestsGoal(filter string) string {
	escaped := strings.ReplaceAll(filter, "'", "''")
	return fmt.Sprintf(
		"findall(Unit:Test,(plunit:current_test(Unit,Test,_,_,_),atom_string(Test,S),sub_string(S,_,_,_,'%s')),Tests),run_tests(Tests)",
		escaped,
	)
}

func replaceRunTestsGoal(args []string, goal string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "-g" && out[i+1] == "run_tests" {
			out[i+1] = goal
			return out
		}
	}
	return append(out, "-g", goal)
}
