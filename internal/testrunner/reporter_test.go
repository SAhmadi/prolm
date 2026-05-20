package testrunner

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_PassingSummary(t *testing.T) {
	var buf bytes.Buffer
	Report(&TestResult{Total: 3, Passed: 3, Duration: 42 * time.Millisecond}, &buf, ReportOptions{NoColor: true})
	out := buf.String()
	assert.Contains(t, out, "3 passed")
	assert.Contains(t, out, "0 failed")
}

func TestReport_FailingListsCases(t *testing.T) {
	var buf bytes.Buffer
	res := &TestResult{
		Total: 2, Passed: 1, Failed: 1,
		Cases: []TestCase{{Suite: "main", Name: "truth", Status: "fail", Message: "expected true"}},
	}
	Report(res, &buf, ReportOptions{NoColor: true})
	out := buf.String()
	assert.Contains(t, out, "main:truth")
	assert.Contains(t, out, "1 failed")
}

func TestReport_VerboseListsPassingCases(t *testing.T) {
	var buf bytes.Buffer
	res := &TestResult{
		Total: 1, Passed: 1,
		Cases: []TestCase{{Suite: "main", Name: "hello", Status: "pass"}},
	}
	Report(res, &buf, ReportOptions{Verbose: true, NoColor: true})
	out := buf.String()
	assert.Contains(t, out, "main:hello")
	assert.Contains(t, out, "1 passed")
}

func TestReport_NonVerboseOmitsPassingCases(t *testing.T) {
	var buf bytes.Buffer
	res := &TestResult{
		Total: 1, Passed: 1,
		Cases: []TestCase{{Suite: "main", Name: "hello", Status: "pass"}},
	}
	Report(res, &buf, ReportOptions{NoColor: true})
	assert.NotContains(t, buf.String(), "main:hello")
}

func TestReport_JSON(t *testing.T) {
	var buf bytes.Buffer
	res := &TestResult{Total: 1, Passed: 1}
	Report(res, &buf, ReportOptions{JSON: true})
	var decoded TestResult
	require.NoError(t, json.NewDecoder(&buf).Decode(&decoded))
	assert.Equal(t, 1, decoded.Total)
}

func TestReport_NoColorHasNoANSI(t *testing.T) {
	var buf bytes.Buffer
	Report(&TestResult{Total: 1, Passed: 1}, &buf, ReportOptions{NoColor: true})
	assert.False(t, strings.Contains(buf.String(), "\x1b["), "should not contain ANSI escapes")
}
