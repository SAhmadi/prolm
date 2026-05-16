package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstalledModulePath_NestedArchiveRoot(t *testing.T) {
	dir := t.TempDir()
	packPath := filepath.Join(dir, "aop", "0.0.9")
	modulePath := filepath.Join(packPath, "aop-0.0.9", "prolog", "aop.pl")
	require.NoError(t, os.MkdirAll(filepath.Dir(modulePath), 0755))
	require.NoError(t, os.WriteFile(modulePath, []byte(":- module(aop, []).\n"), 0644))

	got, err := resolveInstalledModulePath(packPath, "aop")
	require.NoError(t, err)
	assert.Equal(t, modulePath, got)
}

func TestResolveInstalledModulePath_ErrorsWhenModuleMissing(t *testing.T) {
	dir := t.TempDir()
	packPath := filepath.Join(dir, "aop", "0.0.9")
	require.NoError(t, os.MkdirAll(packPath, 0755))

	_, err := resolveInstalledModulePath(packPath, "aop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prolog/aop.pl")
}
