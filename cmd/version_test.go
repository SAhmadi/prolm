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

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func executeVersionTestCommand(args ...string) commandResult {
	cmd := newRootCmd()
	cmd.AddCommand(newVersionCmd())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	executedCmd, err := cmd.ExecuteC()
	if err != nil {
		printCommandError(cmd, executedCmd, err)
	}

	return commandResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
		err:    err,
	}
}

func TestExecute_Version(t *testing.T) {
	result := executeVersionTestCommand("version")
	require.NoError(t, result.err)

	assert.True(t, strings.HasPrefix(result.stdout, "prolm "), "expected output to start with 'prolm ', got: %q", result.stdout)
	assert.Contains(t, result.stdout, Version)
	assert.Empty(t, result.stderr)
}

func TestExecute_Version_UsesInjectedVersionVariable(t *testing.T) {
	originalVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = originalVersion })

	result := executeVersionTestCommand("version")
	require.NoError(t, result.err)

	assert.Equal(t, "prolm 1.2.3\n", result.stdout)
	assert.Empty(t, result.stderr)
}

func TestExecute_Help(t *testing.T) {
	// --help causes cobra to print and return nil.
	result := executeVersionTestCommand("--help")
	require.NoError(t, result.err)

	assert.Contains(t, result.stdout, "prolm")
	assert.Contains(t, result.stdout, "--verbose")
	assert.Contains(t, result.stdout, "--no-color")
	assert.Contains(t, result.stdout, "--json")
	assert.Empty(t, result.stderr)
}

func TestExecute_HelpWithGlobalFlag(t *testing.T) {
	result := executeVersionTestCommand("--no-color", "--help")
	require.NoError(t, result.err)

	assert.Contains(t, result.stdout, "Usage:")
	assert.Contains(t, result.stdout, "--no-color")
	assert.Contains(t, result.stdout, "Available Commands:")
	assert.Empty(t, result.stderr)
}

func TestExecute_CompletionHelp(t *testing.T) {
	result := executeVersionTestCommand("completion", "--help")
	require.NoError(t, result.err)

	assert.Contains(t, result.stdout, "Generate the autocompletion script")
	assert.Contains(t, result.stdout, "bash")
	assert.Contains(t, result.stdout, "zsh")
	assert.Contains(t, result.stdout, "fish")
	assert.Empty(t, result.stderr)
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
	result := executeVersionTestCommand("frob")
	require.Error(t, result.err)

	assert.Empty(t, result.stdout)
	assert.Contains(t, result.stderr, `unknown command "frob" for "prolm"`)
	assert.Contains(t, result.stderr, "Usage:")
	assert.Contains(t, result.stderr, "prolm [command]")
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
