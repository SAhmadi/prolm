package testrunner

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TestResult is the structured outcome of a single `prolm test` invocation.
// It is intentionally friendly to JSON marshalling and templated reporting.
type TestResult struct {
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Errors    int           `json:"errors"`
	Cases     []TestCase    `json:"cases,omitempty"`
	Duration  time.Duration `json:"duration_ns"`
	RawStdout string        `json:"-"`
	RawStderr string        `json:"-"`
}

// TestCase is a single PlUnit test result. Only failing/erroring cases are
// reported individually; passing cases are only counted (PlUnit does not
// print each passing test by default).
type TestCase struct {
	Suite   string `json:"suite"`
	Name    string `json:"name"`
	Status  string `json:"status"` // pass | fail | error
	Message string `json:"message,omitempty"`
}

// Failed reports whether any case failed or errored.
func (r *TestResult) FailedCount() int { return r.Failed + r.Errors }

// reAllPassed matches lines like "% All 3 tests passed".
var reAllPassed = regexp.MustCompile(`(?m)^%\s*All\s+(\d+)\s+tests?\s+passed`)

// reFailedOutOf matches lines like "% 1 test failed out of 3" or
// "% 2 tests failed out of 5".
var reFailedOutOf = regexp.MustCompile(`(?m)^%\s*(\d+)\s+tests?\s+failed\s+out\s+of\s+(\d+)`)

// reCaseFail matches lines like "% test main:truth: failed" or
// "ERROR: test main:truth: <message>".
var reCaseFail = regexp.MustCompile(`test\s+([A-Za-z0-9_]+):([A-Za-z0-9_]+):\s*(.*)`)

// Parse turns swipl/PlUnit stdout+stderr into a structured TestResult.
//
// PlUnit's output format is not a stable contract, so this parser is
// deliberately tolerant: unrecognised lines are ignored rather than
// producing errors, and an empty input yields a zeroed result.
func Parse(stdout, stderr string) *TestResult {
	res := &TestResult{RawStdout: stdout, RawStderr: stderr}

	combined := stdout + "\n" + stderr

	if m := reAllPassed.FindStringSubmatch(combined); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		res.Total = n
		res.Passed = n
	} else if m := reFailedOutOf.FindStringSubmatch(combined); len(m) == 3 {
		failed, _ := strconv.Atoi(m[1])
		total, _ := strconv.Atoi(m[2])
		res.Total = total
		res.Failed = failed
		res.Passed = total - failed
		if res.Passed < 0 {
			res.Passed = 0
		}
	}

	// Capture individual failing cases.
	seen := map[string]struct{}{}
	for _, line := range strings.Split(combined, "\n") {
		if !strings.Contains(line, "test ") {
			continue
		}
		m := reCaseFail.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		status := "fail"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			status = "error"
		}
		key := m[1] + ":" + m[2]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res.Cases = append(res.Cases, TestCase{
			Suite:   m[1],
			Name:    m[2],
			Status:  status,
			Message: strings.TrimSpace(m[3]),
		})
	}

	return res
}
