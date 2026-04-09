package testrunner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_AllPassed(t *testing.T) {
	stdout := `% PL-Unit: main .. done
% All 2 tests passed
`
	res := Parse(stdout, "")
	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 2, res.Passed)
	assert.Equal(t, 0, res.Failed)
	assert.Equal(t, 0, res.Errors)
}

func TestParse_OneFailure(t *testing.T) {
	stdout := `% PL-Unit: main
% test main:truth: failed
% 1 test failed out of 3
`
	res := Parse(stdout, "")
	assert.Equal(t, 3, res.Total)
	assert.Equal(t, 1, res.Failed)
	assert.Equal(t, 2, res.Passed)
	if assert.NotEmpty(t, res.Cases) {
		assert.Equal(t, "truth", res.Cases[0].Name)
		assert.Equal(t, "main", res.Cases[0].Suite)
		assert.Equal(t, "fail", res.Cases[0].Status)
	}
}

func TestParse_Empty(t *testing.T) {
	res := Parse("", "")
	assert.Equal(t, 0, res.Total)
	assert.NotNil(t, res)
}

func TestParse_DoesNotPanicOnMalformed(t *testing.T) {
	assert.NotPanics(t, func() {
		Parse("random garbage\n% tests\nfoo", "ERROR: something")
	})
}

// BUG-012: suite and case names with hyphens/dots must be captured.
func TestParse_HyphenatedSuiteAndCase(t *testing.T) {
	stdout := "% test my-suite:case-1: failed\n% 1 test failed out of 2\n"
	res := Parse(stdout, "")
	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 1, res.Failed)
	if assert.Len(t, res.Cases, 1, "hyphenated case must appear in Cases") {
		assert.Equal(t, "my-suite", res.Cases[0].Suite)
		assert.Equal(t, "case-1", res.Cases[0].Name)
		assert.Equal(t, "fail", res.Cases[0].Status)
	}
}

func TestParse_DottedSuiteAndCase(t *testing.T) {
	stdout := "% test suite.v2:test.3: failed\n% 1 test failed out of 1\n"
	res := Parse(stdout, "")
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, 1, res.Failed)
	if assert.Len(t, res.Cases, 1, "dotted case must appear in Cases") {
		assert.Equal(t, "suite.v2", res.Cases[0].Suite)
		assert.Equal(t, "test.3", res.Cases[0].Name)
	}
}

// BUG-012: when only per-case lines are emitted (no summary line), FailedCount
// must still reflect the failures so the process exits non-zero.
func TestParse_NoSummaryLine_FailedCountFromCases(t *testing.T) {
	// swipl can omit the "N tests failed out of M" summary line in certain
	// configurations; ensure FailedCount > 0 so prolm test exits 1.
	stdout := "% test my-suite:bad-case: failed\n"
	res := Parse(stdout, "")
	// Total/Passed may be 0 if no summary line was emitted, but Failed must
	// reflect the captured case.
	if assert.Len(t, res.Cases, 1, "case must be captured even without summary line") {
		assert.Equal(t, "fail", res.Cases[0].Status)
	}
	assert.Greater(t, res.FailedCount(), 0,
		"FailedCount must be > 0 when per-case failure lines are present without a summary")
}
