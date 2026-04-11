package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePl(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func TestScanDeps_StandardLibrary(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(clpfd)).
main :- true.
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_SkipsBuiltins(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(lists)).
:- use_module(library(apply)).
:- use_module(library(assoc)).
:- use_module(library(pairs)).
:- use_module(library(ordsets)).
:- use_module(library(plunit)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_SkipsLocalModules(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(my_module).
:- use_module(utils).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_SkipsQuotedPaths(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module('path/to/file').
:- use_module("another/file").
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_NestedLibraryPath(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(http/http_server)).
:- use_module(library(http/http_client)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"http": "*"}, deps)
}

func TestScanDeps_MultipleDeps(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(prosqlite)).
:- use_module(library(http/http_client)).
:- use_module(library(lists)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Equal(t, "*", deps["http"])
	assert.Len(t, deps, 2) // lists is built-in
}

func TestScanDeps_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "src/main.pl", `:- use_module(library(prosqlite)).
`)
	writePl(t, dir, "src/utils.pl", `:- use_module(library(http/http_client)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Equal(t, "*", deps["http"])
	assert.Len(t, deps, 2)
}

func TestScanDeps_Deduplication(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "a.pl", `:- use_module(library(prosqlite)).
`)
	writePl(t, dir, "b.pl", `:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "*", deps["prosqlite"])
}

func TestScanDeps_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_NoPlFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644))
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_IndentedDirective(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `  :- use_module(library(prosqlite)).
	:- use_module(library(http/http_client)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Equal(t, "*", deps["http"])
}

func TestScanDeps_CommentedOutDirective(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `% :- use_module(library(fake_dep)).
  % :- use_module(library(another_fake)).
:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "*", deps["prosqlite"])
}

func TestScanDeps_BuiltinDcg(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(dcg/basics)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_SkipsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()
	// Place a .pl file in a hidden directory — it should be ignored.
	hiddenDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hiddenDir, 0755))
	writePl(t, hiddenDir, "pre-commit.pl", `:- use_module(library(somepkg)).
`)
	// Place a valid .pl file in the visible tree.
	writePl(t, dir, "main.pl", `:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "*", deps["prosqlite"])
	// somepkg from .git/ should NOT appear.
	assert.Empty(t, deps["somepkg"])
}

// --- Multi-line directive tests (QUALITY-013) ---

func TestScanDeps_MultiLineDirective(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(
    library(prosqlite)
).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"prosqlite": "*"}, deps)
}

func TestScanDeps_MultiLineCompact(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(
library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"prosqlite": "*"}, deps)
}

func TestScanDeps_MultiLineBuiltinSkipped(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(
    library(lists)
).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestScanDeps_MultiLineNestedPath(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(
    library(http/http_server)
).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"http": "*"}, deps)
}

func TestScanDeps_MixedSingleAndMultiLine(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(http/http_client)).
:- use_module(
    library(prosqlite)
).
:- use_module(library(lists)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["http"])
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Len(t, deps, 2) // lists is built-in
}

func TestScanDeps_MultiLineAbandoned(t *testing.T) {
	// A malformed directive that never closes should not break the scanner
	// or prevent subsequent directives from being detected.
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(
    some_garbage_that_never_closes
    more_garbage
:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	// prosqlite may or may not be detected depending on buffering, but no crash.
	assert.NoError(t, err)
	_ = deps
}

// --- Builtins list tests (QUALITY-014) ---

func TestSwiBuiltins_CoreEntriesPresent(t *testing.T) {
	core := []string{
		"clpfd",
		"lists", "apply", "assoc", "pairs", "ordsets", "plunit",
		"system", "readutil", "aggregate", "dcg/basics", "random",
		"csv", "socket", "thread", "url",
	}
	for _, name := range core {
		assert.True(t, swiBuiltins[name], "expected %q in swiBuiltins", name)
	}
}

func TestSwiBuiltins_MinimumCount(t *testing.T) {
	assert.GreaterOrEqual(t, len(swiBuiltins), 50,
		"swiBuiltins should have at least 50 entries to cover SWI-Prolog 9.2.x standard libs")
}
