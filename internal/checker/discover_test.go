package checker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_FindsSourceSkipsTestsAndVendor(t *testing.T) {
	dir := t.TempDir()
	write := func(p, body string) {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, p), []byte(body), 0644))
	}

	write("src/main.pl", "")
	write("src/util.pl", "")
	write("src/main_test.pl", "")    // excluded (test suffix)
	write("tests/test_foo.pl", "")   // excluded (test prefix)
	write("vendor/lib.pl", "")       // excluded (vendor)
	write(".git/hook.pl", "")        // excluded (.git)
	write("README.md", "")           // excluded (non-.pl)

	files, err := Discover(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "src", "main.pl"),
		filepath.Join(dir, "src", "util.pl"),
	}, files)
}

func TestDiscover_EmptyDir(t *testing.T) {
	files, err := Discover(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, files)
}
