package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_New_MissingArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"new"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestExecute_New_InvalidName(t *testing.T) {
	t.Chdir(t.TempDir())

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"new", "../evil"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path separator")
}

func TestExecute_New_HappyPath(t *testing.T) {
	t.Chdir(t.TempDir())

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"new", "my-project"})
	err := Execute()
	require.NoError(t, err)

	_, statErr := os.Stat("my-project")
	assert.NoError(t, statErr, "project directory should exist")

	_, statErr = os.Stat("my-project/Prolfile.toml")
	assert.NoError(t, statErr, "Prolfile.toml should exist")

	_, statErr = os.Stat("my-project/src/main.pl")
	assert.NoError(t, statErr, "src/main.pl should exist")
}

func TestExecute_New_WithRuntimeFlag(t *testing.T) {
	t.Chdir(t.TempDir())

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"new", "scryer-project", "--runtime", "scryer"})
	err := Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile("scryer-project/Prolfile.toml")
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `runtime = "scryer"`)
}
