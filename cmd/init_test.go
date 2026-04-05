package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Init_HappyPath(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory with a valid package name to chdir into.
	projectDir := filepath.Join(dir, "my-init-project")
	require.NoError(t, os.Mkdir(projectDir, 0755))
	t.Chdir(projectDir)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"init", "--yes"})
	err := Execute()
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(projectDir, "Prolfile.toml"))
	assert.NoError(t, statErr, "Prolfile.toml should exist")

	_, statErr = os.Stat(filepath.Join(projectDir, "Prolfile.lock"))
	assert.NoError(t, statErr, "Prolfile.lock should exist")
}

func TestExecute_Init_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "existing-project")
	require.NoError(t, os.Mkdir(projectDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "Prolfile.toml"), []byte("[package]\n"), 0644))
	t.Chdir(projectDir)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"init", "--yes"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestExecute_Init_NoArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	// init should reject positional arguments.
	rootCmd.SetArgs([]string{"init", "some-arg", "--yes"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestExecute_Init_WithScan(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "scan-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "src", "main.pl"),
		[]byte(`:- use_module(library(clpfd)).
main :- true.
`),
		0644,
	))
	t.Chdir(projectDir)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"init", "--yes"}) // --scan defaults to true
	err := Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(projectDir, "Prolfile.toml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "clpfd")
}

func TestExecute_Init_NoScan(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "noscan-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "src", "main.pl"),
		[]byte(`:- use_module(library(clpfd)).
`),
		0644,
	))
	t.Chdir(projectDir)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs([]string{"init", "--yes", "--scan=false"})
	err := Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(projectDir, "Prolfile.toml"))
	require.NoError(t, readErr)
	assert.NotContains(t, string(data), "clpfd")
}
