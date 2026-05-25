package installer

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/prolm/prolm/internal/httputil"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/resolver"
	"github.com/prolm/prolm/internal/ui"
	"github.com/prolm/prolm/pkg/prolfile"
)

// defaultAllowedHosts are trusted registry hostnames for lockfile URL validation (SEC-14).
var defaultAllowedHosts = []string{
	"github.com",
	"www.swi-prolog.org",
}

// Options configures an install operation.
type Options struct {
	StoreDir string // override store directory (default: ~/.prolm/store/)
	CacheDir string // override cache directory (default: ~/.prolm/cache/)
}

// Install resolves and installs all dependencies from the manifest (SEC-1 through SEC-14).
// If lock is non-nil, it is used as the baseline for already-installed packages.
// Returns the new/updated lockfile. Never executes code from downloaded packs (SEC-3).
func Install(ctx context.Context, manifest *prolfile.ProlFile, lock *prolfile.LockFile, reg registry.Registry, opts Options) (*prolfile.LockFile, error) {
	store := NewStore(opts.StoreDir)
	if err := store.EnsureDir(); err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		cacheDir = filepath.Join(home, ".prolm", "cache")
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	// SEC-7: acquire file lock for write operations.
	unlock, err := store.Lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	var resolved *prolfile.LockFile
	if lockCoversInstalledManifest(lock, manifest, store) {
		resolved = cloneLock(lock)
		for i := range resolved.Packages {
			if err := validateLockURL(resolved.Packages[i].URL); err != nil {
				return nil, err
			}
			ui.Success("already installed: %s@%s", resolved.Packages[i].Name, resolved.Packages[i].Version)
		}
	} else {
		resolved, err = resolver.Resolve(ctx, manifest, reg)
		if err != nil {
			return nil, fmt.Errorf("resolving dependencies: %w", err)
		}
	}

	newLock := &prolfile.LockFile{
		Meta: prolfile.LockMeta{
			LockVersion: resolved.Meta.LockVersion,
		},
	}
	if newLock.Meta.LockVersion == 0 {
		newLock.Meta.LockVersion = prolfile.CurrentLockVersion
	}
	prolfileHash, err := lockfile.ComputeProlfileHash(manifest)
	if err != nil {
		return nil, fmt.Errorf("computing prolfile hash: %w", err)
	}
	newLock.Meta.ProlfileHash = prolfileHash

	for i := range resolved.Packages {
		entry, err := installResolved(ctx, resolved.Packages[i], lock, store, reg, cacheDir)
		if err != nil {
			return nil, fmt.Errorf("installing %s: %w", resolved.Packages[i].Name, err)
		}
		newLock.Packages = append(newLock.Packages, *entry)
	}

	// Deterministic output: sort packages by name.
	sort.Slice(newLock.Packages, func(i, j int) bool {
		return newLock.Packages[i].Name < newLock.Packages[j].Name
	})

	return newLock, nil
}

// installResolved handles the install flow for one resolved lock entry.
func installResolved(ctx context.Context, resolved prolfile.LockEntry, lock *prolfile.LockFile, store *Store, reg registry.Registry, cacheDir string) (*prolfile.LockEntry, error) {
	name := resolved.Name
	version := resolved.Version

	// Fast path: already in lock and installed in store — skip all cache I/O.
	// The pack is already unpacked; tarball cache integrity is irrelevant here.
	if existing := findLockEntry(lock, name); existing != nil {
		if existing.Version == version && store.IsInstalled(name, existing.Version) {
			// SEC-14: validate URL origin.
			if err := validateLockURL(existing.URL); err != nil {
				return nil, err
			}
			ui.Success("already installed: %s@%s", name, existing.Version)
			return existing, nil
		}
	}

	downloadURL := resolved.URL

	// SEC-14: validate download URL.
	if err := validateLockURL(downloadURL); err != nil {
		return nil, err
	}

	// Fetch tarball to cache.
	tarball := cachePath(cacheDir, name, version)
	ui.Info("fetching %s@%s", name, version)
	if err := Fetch(ctx, downloadURL, tarball); err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}

	// SEC-1 / §8.6: if the registry advertised a checksum, verify the
	// downloaded bytes against THAT before trusting any locally computed
	// hash. This closes the TOCTOU window on fresh installs where the
	// lockfile does not yet pin a sha256.
	registryChecksum, checksumWarning := registryIntegrityMetadata(ctx, reg, name, version, resolved.Checksum)
	if err := VerifyRegistryChecksum(tarball, registryChecksum); err != nil {
		return nil, fmt.Errorf("verifying registry checksum: %w", err)
	}
	if registryChecksum == "" && checksumWarning != "" {
		ui.Warn("%s", checksumWarning)
	}

	// SEC-1: compute sha256 for the lockfile.
	checksum, err := ComputeChecksum(tarball)
	if err != nil {
		return nil, fmt.Errorf("computing checksum: %w", err)
	}

	// If we have a lock entry with a checksum, verify against it.
	if existing := findLockEntry(lock, name); existing != nil && existing.Checksum != "" {
		if err := Verify(tarball, existing.Checksum); err != nil {
			return nil, err
		}
	}

	// Unpack to store.
	destDir := store.PackPath(name, version)
	if err := Unpack(tarball, destDir); err != nil {
		return nil, fmt.Errorf("unpacking: %w", err)
	}

	ui.Success("installed %s@%s", name, version)

	// Build lock entry.
	deps := resolved.Dependencies
	if deps == nil {
		deps = []string{}
	}
	entry := &prolfile.LockEntry{
		Name:         name,
		Version:      version,
		Source:       resolved.Source,
		URL:          downloadURL,
		Checksum:     checksum,
		Dependencies: deps,
	}
	return entry, nil
}

func registryIntegrityMetadata(ctx context.Context, reg registry.Registry, name, version, fallbackChecksum string) (string, string) {
	versions, err := reg.Versions(ctx, name)
	if err != nil {
		return fallbackChecksum, ""
	}
	for _, pv := range versions {
		if strings.TrimPrefix(pv.Version, "v") == version {
			return pv.Checksum, pv.ChecksumWarning
		}
	}
	return fallbackChecksum, ""
}

// allDeps merges manifest.Dependencies and manifest.DevDependencies.
func allDeps(manifest *prolfile.ProlFile) map[string]string {
	merged := make(map[string]string)
	for k, v := range manifest.Dependencies {
		merged[k] = v
	}
	for k, v := range manifest.DevDependencies {
		merged[k] = v
	}
	return merged
}

func lockCoversInstalledManifest(lock *prolfile.LockFile, manifest *prolfile.ProlFile, store *Store) bool {
	if lock == nil || len(lock.Packages) == 0 {
		return false
	}
	prolfileHash, err := lockfile.ComputeProlfileHash(manifest)
	if err != nil || lock.Meta.ProlfileHash != prolfileHash {
		return false
	}
	for name := range allDeps(manifest) {
		if findLockEntry(lock, name) == nil {
			return false
		}
	}
	for i := range lock.Packages {
		if !store.IsInstalled(lock.Packages[i].Name, lock.Packages[i].Version) {
			return false
		}
	}
	return true
}

func cloneLock(lock *prolfile.LockFile) *prolfile.LockFile {
	if lock == nil {
		return nil
	}
	clone := &prolfile.LockFile{
		Meta:     lock.Meta,
		Packages: append([]prolfile.LockEntry(nil), lock.Packages...),
	}
	sort.Slice(clone.Packages, func(i, j int) bool {
		return clone.Packages[i].Name < clone.Packages[j].Name
	})
	return clone
}

// findLockEntry searches lock for a matching package name.
func findLockEntry(lock *prolfile.LockFile, name string) *prolfile.LockEntry {
	if lock == nil {
		return nil
	}
	for i := range lock.Packages {
		if lock.Packages[i].Name == name {
			return &lock.Packages[i]
		}
	}
	return nil
}

// cachePath returns the cache file path for a package tarball.
func cachePath(cacheDir, name, version string) string {
	return filepath.Join(cacheDir, name, version+".tar.gz")
}

// validateLockURL checks that a URL's hostname is in the allowed hosts list (SEC-14).
// Configurable via PROLM_ALLOWED_HOSTS env var (comma-separated).
func validateLockURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty download URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %s", rawURL)
	}

	// Allow localhost for tests.
	host := u.Hostname()
	if httputil.IsLocalhostURL(rawURL) {
		return nil
	}

	allowed := defaultAllowedHosts
	if env := os.Getenv("PROLM_ALLOWED_HOSTS"); env != "" {
		allowed = strings.Split(env, ",")
		for i := range allowed {
			allowed[i] = strings.TrimSpace(allowed[i])
		}
	}

	for _, h := range allowed {
		if strings.EqualFold(host, h) {
			return nil
		}
	}

	return fmt.Errorf("URL host %q not in allowed hosts (SEC-14); set PROLM_ALLOWED_HOSTS to override", host)
}
