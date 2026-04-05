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
	assert.Equal(t, map[string]string{"clpfd": "*"}, deps)
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
	writePl(t, dir, "main.pl", `:- use_module(library(clpfd)).
:- use_module(library(prosqlite)).
:- use_module(library(lists)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["clpfd"])
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Len(t, deps, 2) // lists is built-in
}

func TestScanDeps_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "src/main.pl", `:- use_module(library(clpfd)).
`)
	writePl(t, dir, "src/utils.pl", `:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["clpfd"])
	assert.Equal(t, "*", deps["prosqlite"])
	assert.Len(t, deps, 2)
}

func TestScanDeps_Deduplication(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "a.pl", `:- use_module(library(clpfd)).
`)
	writePl(t, dir, "b.pl", `:- use_module(library(clpfd)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "*", deps["clpfd"])
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
	writePl(t, dir, "main.pl", `  :- use_module(library(clpfd)).
	:- use_module(library(prosqlite)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Equal(t, "*", deps["clpfd"])
	assert.Equal(t, "*", deps["prosqlite"])
}

func TestScanDeps_CommentedOutDirective(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `% :- use_module(library(fake_dep)).
  % :- use_module(library(another_fake)).
:- use_module(library(clpfd)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "*", deps["clpfd"])
}

func TestScanDeps_BuiltinDcg(t *testing.T) {
	dir := t.TempDir()
	writePl(t, dir, "main.pl", `:- use_module(library(dcg/basics)).
`)
	deps, err := ScanDeps(dir)
	require.NoError(t, err)
	assert.Empty(t, deps)
}
