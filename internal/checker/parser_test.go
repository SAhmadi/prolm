package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		stdout        string
		stderr        string
		wantDiagCount int
		wantWarnings  int
		wantErrors    int
		wantMsg       string
	}{
		{
			name:          "empty output",
			stdout:        "",
			stderr:        "",
			wantDiagCount: 0,
			wantWarnings:  0,
			wantErrors:    0,
		},
		{
			name: "single warning with location",
			stdout: "Warning: /path/to/file.pl:10:5: Singleton variable 'X'\n",
			stderr: "",
			wantDiagCount: 1,
			wantWarnings: 1,
			wantMsg: "Singleton variable 'X'",
		},
		{
			name: "error without location",
			stdout: "ERROR:   Undefined predicate: foo/1\n",
			stderr: "",
			wantDiagCount: 1,
			wantErrors: 1,
			wantMsg: "Undefined predicate: foo/1",
		},
		{
			name: "warning without column",
			stdout: "Warning: /path/main.pl:20: Missing module declaration\n",
			stderr: "",
			wantDiagCount: 1,
			wantWarnings: 1,
		},
		{
			name: "mixed warnings and errors",
			stdout: `Warning: /path/a.pl:1:1: First warning
ERROR:   /path/b.pl:2:2: An error
Warning: /path/c.pl:3:3: Second warning
`,
			stderr: "",
			wantDiagCount: 3,
			wantWarnings: 2,
			wantErrors: 1,
		},
		{
			name: "stderr and stdout combined",
			stdout: "Warning: /path/file.pl:10:5: msg1\n",
			stderr: "ERROR:   /path/other.pl:20:10: msg2\n",
			wantDiagCount: 2,
			wantWarnings: 1,
			wantErrors: 1,
		},
		{
			name: "non-matching lines ignored",
			stdout: `Warning: /path/file.pl:10:5: valid warning
some random output
another line with no structure
ERROR:   /path/other.pl:20:10: valid error
`,
			stderr: "",
			wantDiagCount: 2,
			wantWarnings: 1,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Parse(tt.stdout, tt.stderr)
			assert.Equal(t, tt.wantDiagCount, len(res.Diagnostics))
			assert.Equal(t, tt.wantWarnings, len(res.Warnings()))
			assert.Equal(t, tt.wantErrors, len(res.Errors()))
			if tt.wantMsg != "" && len(res.Diagnostics) > 0 {
				assert.Contains(t, res.Diagnostics[0].Message, tt.wantMsg)
			}
		})
	}
}

func TestParse_ExtractsLocationInfo(t *testing.T) {
	res := Parse("Warning: /home/user/src/main.pl:42:17: Test message\n", "")
	require.Equal(t, 1, len(res.Diagnostics))
	d := res.Diagnostics[0]
	assert.Equal(t, SeverityWarning, d.Severity)
	assert.Equal(t, "/home/user/src/main.pl", d.File)
	assert.Equal(t, 42, d.Line)
	assert.Equal(t, 17, d.Col)
	assert.Equal(t, "Test message", d.Message)
}

func TestParse_ContinuationLines(t *testing.T) {
	stdout := `Warning: /path/file.pl:10:5: First line
  continuation with more details
  and even more info
ERROR:   /path/other.pl:20:10: Another diagnostic
`
	res := Parse(stdout, "")
	assert.Equal(t, 2, len(res.Diagnostics))
	assert.Contains(t, res.Diagnostics[0].Message, "continuation")
	assert.Contains(t, res.Diagnostics[0].Message, "more info")
}

func TestCheckResult_FilterMethods(t *testing.T) {
	res := &CheckResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "w1"},
			{Severity: SeverityError, Message: "e1"},
			{Severity: SeverityWarning, Message: "w2"},
			{Severity: SeverityError, Message: "e2"},
		},
	}
	warnings := res.Warnings()
	assert.Equal(t, 2, len(warnings))
	assert.Equal(t, SeverityWarning, warnings[0].Severity)
	assert.Equal(t, SeverityWarning, warnings[1].Severity)

	errors := res.Errors()
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, SeverityError, errors[0].Severity)
	assert.Equal(t, SeverityError, errors[1].Severity)
}
