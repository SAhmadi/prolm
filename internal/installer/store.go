package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultLockTimeout = 30 * time.Second

// ErrStoreLocked indicates another process holds the store lock.
type ErrStoreLocked struct {
	LockPath string
}

func (e *ErrStoreLocked) Error() string {
	return fmt.Sprintf("another prolm process is running (lock: %s); wait for it to finish or remove the lock file if stale", e.LockPath)
}

// Store manages the local package store at a base directory (~/.prolm/store/).
type Store struct {
	baseDir     string
	lockTimeout time.Duration
}

// NewStore creates a Store. If baseDir is empty, defaults to ~/.prolm/store/.
func NewStore(baseDir string) *Store {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		baseDir = filepath.Join(home, ".prolm", "store")
	}
	return &Store{
		baseDir:     baseDir,
		lockTimeout: defaultLockTimeout,
	}
}

// PackPath returns the directory path for an installed package version.
func (s *Store) PackPath(name, version string) string {
	return filepath.Join(s.baseDir, name, version)
}

// IsInstalled reports whether the package at name@version exists in the store.
func (s *Store) IsInstalled(name, version string) bool {
	info, err := os.Stat(s.PackPath(name, version))
	return err == nil && info.IsDir()
}

// EnsureDir creates the store base directory with permissions 0755 (SEC-6).
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// Lock acquires an exclusive file lock on <baseDir>/.lock (SEC-7).
// Returns an unlock function that must be called (typically via defer).
// Uses O_CREATE|O_EXCL for atomic lock creation without external dependencies.
func (s *Store) Lock() (unlock func(), err error) {
	lockPath := filepath.Join(s.baseDir, ".lock")

	// Ensure base directory exists before attempting lock.
	if err := s.EnsureDir(); err != nil {
		return nil, fmt.Errorf("creating store directory: %w", err)
	}

	deadline := time.Now().Add(s.lockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating lock file: %w", err)
		}
		if time.Now().After(deadline) {
			return nil, &ErrStoreLocked{LockPath: lockPath}
		}
		time.Sleep(200 * time.Millisecond)
	}
}
