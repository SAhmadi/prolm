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
