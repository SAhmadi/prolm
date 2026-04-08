package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/ui"
	"github.com/prolm/prolm/pkg/prolfile"
)

// verifyStoreDeps loads Prolfile.lock next to manifestPath and verifies that
// every locked package exists in storeDir. On success it returns the absolute
// path of each installed pack (suitable for use_module/1 arguments) in lock
// order. On failure it emits a user-friendly hint and returns an error.
//
// Shared by `prolm run` and `prolm test` (addresses DRY-004).
func verifyStoreDeps(_ *prolfile.ProlFile, manifestPath, storeDir string) ([]string, error) {
	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		return nil, fmt.Errorf("reading Prolfile.lock: %w", err)
	}
	if lock == nil {
		ui.Hint("Run `prolm install` first to download dependencies")
		return nil, fmt.Errorf("prolfile.lock not found at %s", lockPath)
	}

	store := installer.NewStore(storeDir)
	depPaths := make([]string, 0, len(lock.Packages))
	for _, pkg := range lock.Packages {
		if !store.IsInstalled(pkg.Name, pkg.Version) {
			ui.Hint("Run `prolm install` first to download dependencies")
			return nil, fmt.Errorf("package %s@%s not installed in store", pkg.Name, pkg.Version)
		}
		depPaths = append(depPaths, store.PackPath(pkg.Name, pkg.Version))
	}
	return depPaths, nil
}
