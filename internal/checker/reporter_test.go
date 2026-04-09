package checker

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_CleanResultPassesAndPrintsZeroSummary(t *testing.T) {
	var buf bytes.Buffer
	fail := Report(&CheckResult{}, &buf, ReportOptions{NoColor: true})
	assert.False(t, fail)
	assert.Contains(t, buf.String(), "0 error(s), 0 warning(s)")
}

func TestReport_WarningDoesNotFailUnlessStrict(t *testing.T) {
	res := &CheckResult{Diagnostics: []Diagnostic{
		{Severity: SeverityWarning, File: "a.pl", Line: 1, Message: "w"},
	}}
	var buf bytes.Buffer
	assert.False(t, Report(res, &buf, ReportOptions{NoColor: true}))
	buf.Reset()
	assert.True(t, Report(res, &buf, ReportOptions{NoColor: true, Strict: true}))
	assert.Contains(t, buf.String(), "a.pl:1")
}

func TestReport_ErrorAlwaysFails(t *testing.T) {
	res := &CheckResult{Diagnostics: []Diagnostic{
		{Severity: SeverityError, Message: "e"},
	}}
	var buf bytes.Buffer
	assert.True(t, Report(res, &buf, ReportOptions{NoColor: true}))
}

func TestReport_JSON(t *testing.T) {
	res := &CheckResult{Diagnostics: []Diagnostic{
		{Severity: SeverityWarning, File: "a.pl", Line: 1, Message: "w"},
	}}
	var buf bytes.Buffer
	Report(res, &buf, ReportOptions{JSON: true, NoColor: true})
	var decoded CheckResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Len(t, decoded.Diagnostics, 1)
}
