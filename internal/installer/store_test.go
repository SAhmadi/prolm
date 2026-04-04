package installer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore_DefaultDir(t *testing.T) {
	s := NewStore("")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".prolm", "store"), s.baseDir)
}

func TestNewStore_CustomDir(t *testing.T) {
	s := NewStore("/tmp/custom-store")
	assert.Equal(t, "/tmp/custom-store", s.baseDir)
}

func TestStore_EnsureDir(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	s := NewStore(storeDir)

	require.NoError(t, s.EnsureDir())

	info, err := os.Stat(storeDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestStore_PackPath(t *testing.T) {
	s := NewStore("/tmp/store")
	assert.Equal(t, "/tmp/store/clpfd/1.4.3", s.PackPath("clpfd", "1.4.3"))
}

func TestStore_IsInstalled_True(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	packDir := s.PackPath("clpfd", "1.4.3")
	require.NoError(t, os.MkdirAll(packDir, 0755))

	assert.True(t, s.IsInstalled("clpfd", "1.4.3"))
}

func TestStore_IsInstalled_False(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	assert.False(t, s.IsInstalled("clpfd", "1.4.3"))
}

func TestStore_IsInstalled_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	// Create a file (not a directory) at the pack path.
	packDir := filepath.Dir(s.PackPath("clpfd", "1.4.3"))
	require.NoError(t, os.MkdirAll(packDir, 0755))
	require.NoError(t, os.WriteFile(s.PackPath("clpfd", "1.4.3"), []byte("not a dir"), 0644))

	assert.False(t, s.IsInstalled("clpfd", "1.4.3"))
}

func TestStore_Lock_Success(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.lockTimeout = 2 * time.Second

	unlock, err := s.Lock()
	require.NoError(t, err)

	// Lock file should exist.
	lockPath := filepath.Join(dir, ".lock")
	_, statErr := os.Stat(lockPath)
	assert.NoError(t, statErr)

	unlock()

	// Lock file should be removed after unlock.
	_, statErr = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestStore_Lock_Contention(t *testing.T) {
	dir := t.TempDir()

	s1 := NewStore(dir)
	s1.lockTimeout = 5 * time.Second
	s2 := NewStore(dir)
	s2.lockTimeout = 5 * time.Second

	// First goroutine acquires the lock.
	unlock1, err := s1.Lock()
	require.NoError(t, err)

	// Second goroutine tries to acquire; should block then succeed.
	var wg sync.WaitGroup
	wg.Add(1)
	var lock2Err error
	go func() {
		defer wg.Done()
		unlock2, err := s2.Lock()
		lock2Err = err
		if err == nil {
			unlock2()
		}
	}()

	// Release the first lock after a short delay.
	time.Sleep(300 * time.Millisecond)
	unlock1()

	wg.Wait()
	assert.NoError(t, lock2Err)
}

func TestStore_Lock_Timeout(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.lockTimeout = 500 * time.Millisecond

	// Pre-create the lock file to simulate another process.
	lockPath := filepath.Join(dir, ".lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("12345\n"), 0644))

	_, err := s.Lock()
	var locked *ErrStoreLocked
	require.ErrorAs(t, err, &locked)
	assert.Equal(t, lockPath, locked.LockPath)
}

func TestStore_Lock_UnlockIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.lockTimeout = 2 * time.Second

	unlock, err := s.Lock()
	require.NoError(t, err)

	// Calling unlock multiple times should not panic.
	unlock()
	unlock()
}
