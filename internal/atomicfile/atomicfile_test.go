package atomicfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prolm/prolm/internal/atomicfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	err := atomicfile.Write(path, []byte("hello"), 0644)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))

	// No .tmp file left behind.
	_, err = os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))
}

func TestWrite_InvalidDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "file.txt")

	err := atomicfile.Write(path, []byte("data"), 0644)
	require.Error(t, err)

	// No .tmp file left behind.
	_, err = os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))
}

func TestWrite_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	require.NoError(t, atomicfile.Write(path, []byte("first"), 0644))
	require.NoError(t, atomicfile.Write(path, []byte("second"), 0644))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "second", string(got))
}

func TestWrite_Permissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")

	require.NoError(t, atomicfile.Write(path, []byte("secret"), 0600))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
