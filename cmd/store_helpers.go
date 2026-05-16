package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/ui"
)

// verifyStoreDeps loads Prolfile.lock next to manifestPath and verifies that
// every locked package exists in storeDir. On success it returns the absolute
// path of each installed pack (suitable for use_module/1 arguments) in lock
// order. On failure it emits a user-friendly hint and returns an error.
//
// Shared by `prolm run` and `prolm test` (QUALITY-020, DRY-003).
func verifyStoreDeps(manifestPath, storeDir string) ([]string, error) {
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
		depPath, err := resolveInstalledModulePath(store.PackPath(pkg.Name, pkg.Version), pkg.Name)
		if err != nil {
			return nil, fmt.Errorf("resolving package %s@%s module path: %w", pkg.Name, pkg.Version, err)
		}
		depPaths = append(depPaths, depPath)
	}
	return depPaths, nil
}

// resolveInstalledModulePath returns the concrete Prolog module file to load
// for a package installed in the store. SWI packs are commonly extracted with
// one archive root that contains prolog/<pack>.pl; loading the version
// directory itself fails because it is not a module.
func resolveInstalledModulePath(packPath, name string) (string, error) {
	candidates := []string{
		filepath.Join(packPath, "prolog", name+".pl"),
		filepath.Join(packPath, name+".pl"),
		filepath.Join(packPath, "pack.pl"),
	}
	for _, candidate := range candidates {
		if isRegularFile(candidate) {
			return candidate, nil
		}
	}

	var matches []string
	err := filepath.WalkDir(packPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == name+".pl" && filepath.Base(filepath.Dir(path)) == "prolog" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		return matches[0], nil
	}

	return "", fmt.Errorf("could not find prolog/%s.pl under %s", name, packPath)
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
