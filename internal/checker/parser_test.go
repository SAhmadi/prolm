package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_EmptyYieldsEmpty(t *testing.T) {
	res := Parse("", "")
	assert.Empty(t, res.Diagnostics)
	assert.Empty(t, res.Warnings())
	assert.Empty(t, res.Errors())
}

func TestParse_WarningWithLineAndCol(t *testing.T) {
	res := Parse("", "Warning: /tmp/foo.pl:10:5: Singleton variables: [X]\n")
	require.Len(t, res.Diagnostics, 1)
	d := res.Diagnostics[0]
	assert.Equal(t, SeverityWarning, d.Severity)
	assert.Equal(t, "/tmp/foo.pl", d.File)
	assert.Equal(t, 10, d.Line)
	assert.Equal(t, 5, d.Col)
	assert.Equal(t, "Singleton variables: [X]", d.Message)
}

func TestParse_WarningLineOnly(t *testing.T) {
	res := Parse("", "Warning: /tmp/foo.pl:20: Clauses of foo/1 are not together\n")
	require.Len(t, res.Diagnostics, 1)
	d := res.Diagnostics[0]
	assert.Equal(t, 20, d.Line)
	assert.Equal(t, 0, d.Col)
	assert.Equal(t, "/tmp/foo.pl", d.File)
}

func TestParse_Error(t *testing.T) {
	res := Parse("", "ERROR: /tmp/foo.pl:3:5: Syntax error: Operator expected\n")
	require.Len(t, res.Diagnostics, 1)
	assert.Equal(t, SeverityError, res.Diagnostics[0].Severity)
	assert.Len(t, res.Errors(), 1)
	assert.Empty(t, res.Warnings())
}

func TestParse_WarningWithoutLocation(t *testing.T) {
	res := Parse("", "Warning: Goal (directive) failed: user:foo\n")
	require.Len(t, res.Diagnostics, 1)
	d := res.Diagnostics[0]
	assert.Equal(t, "", d.File)
	assert.Equal(t, 0, d.Line)
	assert.Equal(t, "Goal (directive) failed: user:foo", d.Message)
}

func TestParse_ContinuationLineCoalesced(t *testing.T) {
	out := "Warning: /tmp/foo.pl:10:\n\tSingleton variables: [X]\n"
	res := Parse("", out)
	require.Len(t, res.Diagnostics, 1)
	assert.Contains(t, res.Diagnostics[0].Message, "Singleton variables: [X]")
}

func TestParse_MixedWarningsAndErrors(t *testing.T) {
	out := "Warning: /a.pl:1: w1\nERROR: /b.pl:2: e1\nWarning: /c.pl:3: w2\n"
	res := Parse("", out)
	assert.Len(t, res.Warnings(), 2)
	assert.Len(t, res.Errors(), 1)
}

func TestParse_IgnoresUnrecognisedLines(t *testing.T) {
	out := "hello world\nSome random thing\nWarning: /tmp/foo.pl:1: ok\n"
	res := Parse("", out)
	require.Len(t, res.Diagnostics, 1)
	assert.Equal(t, "ok", res.Diagnostics[0].Message)
}
