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
	"github.com/prolm/prolm/internal/registry"
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

	deps := allDeps(manifest)

	// Deterministic install order: sort dependency names alphabetically (CLAUDE.md §6).
	names := make([]string, 0, len(deps))
	for n := range deps {
		names = append(names, n)
	}
	sort.Strings(names)

	newLock := &prolfile.LockFile{
		Meta: prolfile.LockMeta{
			LockVersion: prolfile.CurrentLockVersion,
		},
	}

	for _, name := range names {
		constraint := deps[name]
		entry, err := installOne(ctx, name, constraint, lock, store, reg, cacheDir)
		if err != nil {
			return nil, fmt.Errorf("installing %s: %w", name, err)
		}
		newLock.Packages = append(newLock.Packages, *entry)
	}

	// Deterministic output: sort packages by name.
	sort.Slice(newLock.Packages, func(i, j int) bool {
		return newLock.Packages[i].Name < newLock.Packages[j].Name
	})

	return newLock, nil
}

// installOne handles the install flow for a single dependency.
func installOne(ctx context.Context, name, constraint string, lock *prolfile.LockFile, store *Store, reg registry.Registry, cacheDir string) (*prolfile.LockEntry, error) {
	// Fast path: already in lock and installed in store — skip all cache I/O.
	// The pack is already unpacked; tarball cache integrity is irrelevant here.
	if existing := findLockEntry(lock, name); existing != nil {
		if store.IsInstalled(name, existing.Version) {
			// SEC-14: validate URL origin.
			if err := validateLockURL(existing.URL); err != nil {
				return nil, err
			}
			ui.Success("already installed: %s@%s", name, existing.Version)
			return existing, nil
		}
	}

	// Query registry for available versions.
	versions, err := reg.Versions(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("querying versions: %w", err)
	}
	if len(versions) == 0 {
		return nil, &registry.ErrPackageNotFound{Name: name, Registry: "swi-pack-index"}
	}

	// Phase 1: pick the first (latest) non-yanked version.
	// Phase 2 will add semver constraint resolution.
	var chosen *registry.PackageVersion
	for i := range versions {
		if !versions[i].Yanked {
			chosen = &versions[i]
			break
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("all versions of %s are yanked", name)
	}

	_ = constraint // Phase 1: constraint not used in resolution yet

	// Get download URL.
	downloadURL, err := reg.DownloadURL(ctx, name, chosen.Version)
	if err != nil {
		return nil, fmt.Errorf("getting download URL: %w", err)
	}

	// SEC-14: validate download URL.
	if err := validateLockURL(downloadURL); err != nil {
		return nil, err
	}

	// Fetch tarball to cache.
	tarball := cachePath(cacheDir, name, chosen.Version)
	ui.Info("fetching %s@%s", name, chosen.Version)
	if err := Fetch(ctx, downloadURL, tarball); err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}

	// SEC-1 / §8.6: if the registry advertised a checksum, verify the
	// downloaded bytes against THAT before trusting any locally computed
	// hash. This closes the TOCTOU window on fresh installs where the
	// lockfile does not yet pin a sha256.
	if err := VerifyRegistryChecksum(tarball, chosen.Checksum); err != nil {
		return nil, fmt.Errorf("verifying registry checksum: %w", err)
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
	destDir := store.PackPath(name, chosen.Version)
	if err := Unpack(tarball, destDir); err != nil {
		return nil, fmt.Errorf("unpacking: %w", err)
	}

	ui.Success("installed %s@%s", name, chosen.Version)

	// Build lock entry.
	deps := chosen.Dependencies
	if deps == nil {
		deps = []string{}
	}
	entry := &prolfile.LockEntry{
		Name:         name,
		Version:      chosen.Version,
		Source:       "swi-pack-index",
		URL:          downloadURL,
		Checksum:     checksum,
		Dependencies: deps,
	}
	return entry, nil
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
