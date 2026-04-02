package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Version(t *testing.T) {
	// Capture stdout by swapping the command's output writer.
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	}()

	rootCmd.SetArgs([]string{"version"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "prolm "), "expected output to start with 'prolm ', got: %q", out)
	assert.Contains(t, out, Version)
}

func TestExecute_Help(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"--help"})
	// --help causes cobra to print and return nil.
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "prolm")
	assert.Contains(t, out, "--verbose")
	assert.Contains(t, out, "--no-color")
	assert.Contains(t, out, "--json")
}
