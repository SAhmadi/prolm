package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Version(t *testing.T) {
	// NOTE: This test mutates the package-level rootCmd (SetArgs, SetOut, SetErr)
	// and must NOT be run in parallel with other tests in this package.
	// TODO: Refactor to construct a fresh cobra.Command per test to enable parallelism.

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

func TestExecute_Version_UsesInjectedVersionVariable(t *testing.T) {
	originalVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = originalVersion })

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"version"})
	err := Execute()
	require.NoError(t, err)

	assert.Equal(t, "prolm 1.2.3\n", buf.String())
}

func TestExecute_Help(t *testing.T) {
	// NOTE: This test mutates the package-level rootCmd (SetArgs, SetOut, SetErr)
	// and must NOT be run in parallel with other tests in this package.
	// TODO: Refactor to construct a fresh cobra.Command per test to enable parallelism.

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

func TestExecute_HelpWithGlobalFlag(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"--no-color", "--help"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "--no-color")
	assert.Contains(t, out, "Available Commands:")
}

func TestExecute_CompletionHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"completion", "--help"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Generate the autocompletion script")
	assert.Contains(t, out, "bash")
	assert.Contains(t, out, "zsh")
	assert.Contains(t, out, "fish")
}

func TestExecute_CompletionScripts_BashAndZsh(t *testing.T) {
	binPath := buildSmokeBinary(t)
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))
	env := append(os.Environ(), "HOME="+homeDir, "NO_COLOR=1")

	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			out := runSmokeCommand(t, binPath, workspace, env, "completion", shell)
			assert.NotEmpty(t, out)
			assert.Contains(t, out, "prolm")
			if shell == "bash" {
				assert.Contains(t, out, "complete")
			}
			if shell == "zsh" {
				assert.Contains(t, out, "#compdef")
			}
		})
	}
}

func TestExecute_UnknownCommand_ShowsRootUsage(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"frob"})
	err := Execute()
	require.Error(t, err)

	out := buf.String()
	assert.Contains(t, out, `unknown command "frob" for "prolm"`)
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "prolm [command]")
}

func TestCommandCentralDocTracksPublicSurface(t *testing.T) {
	data, err := os.ReadFile("../docs/command-central.md")
	require.NoError(t, err)

	doc := string(data)
	assert.Contains(t, doc, "prolm completion")
	assert.Regexp(t, "(?m)(\\*\\*Status:\\*\\*|Status:)\\s+`Available`", doc)
	assert.Contains(t, doc, "prolm add <name|url>")
	assert.Contains(t, doc, "prolm remove <pack>")
	assert.Contains(t, doc, "macOS (zsh)")
	assert.Contains(t, doc, "Linux (bash)")
	assert.Contains(t, doc, "Linux (zsh)")
	assert.Contains(t, doc, "Windows completion setup is deferred")
}
