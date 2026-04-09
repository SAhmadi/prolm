package testrunner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_FindsSuffixAndPrefix(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(":- module(x,[])."), 0644))
	}
	write("tests/main_test.pl")
	write("tests/test_helpers.pl")
	write("src/main.pl")        // not a test
	write("src/other_test.pl")  // suffix match even outside tests/
	write("notes.md")           // ignored

	got, err := Discover(dir)
	require.NoError(t, err)
	want := []string{
		filepath.Join(dir, "src", "other_test.pl"),
		filepath.Join(dir, "tests", "main_test.pl"),
		filepath.Join(dir, "tests", "test_helpers.pl"),
	}
	assert.Equal(t, want, got)
}

func TestDiscover_EmptyDir(t *testing.T) {
	got, err := Discover(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDiscover_SkipsHiddenAndVendorDirs(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{
		".git/ignored_test.pl",
		"node_modules/x_test.pl",
		".prolm/store/pkg/1.0.0/foo_test.pl",
		"tests/real_test.pl",
	} {
		p := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(""), 0644))
	}
	got, err := Discover(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "tests", "real_test.pl")}, got)
}

func TestDiscover_NonExistentDir(t *testing.T) {
	_, err := Discover(filepath.Join(t.TempDir(), "nope"))
	assert.Error(t, err)
}
