package prolfile

// Current format versions for Prolfile.toml and Prolfile.lock.
const (
	CurrentProlfileVersion = 1
	CurrentLockVersion     = 1
)

// Meta holds format versioning metadata from the [meta] section of Prolfile.toml.
// Fields are ordered alphabetically by TOML key for deterministic serialization.
type Meta struct {
	MinProlmVersion string `toml:"min_prolm_version"`
	ProlfileVersion int    `toml:"prolfile_version"`
}

// Package holds project identity and configuration from the [package] section.
// Fields are ordered alphabetically by TOML key for deterministic serialization.
type Package struct {
	Authors     []string `toml:"authors"`
	Description string   `toml:"description"`
	Entry       string   `toml:"entry"`
	Homepage    string   `toml:"homepage"`
	License     string   `toml:"license"`
	Name        string   `toml:"name"`
	Runtime     string   `toml:"runtime"`
	Version     string   `toml:"version"`
}

// RuntimeConfig holds per-runtime settings (e.g. [runtime.swi]).
type RuntimeConfig struct {
	MinVersion string   `toml:"min_version"`
	Flags      []string `toml:"flags"`
}

// ProlFile represents a parsed Prolfile.toml.
// Field order matches the desired TOML section order for deterministic serialization.
type ProlFile struct {
	Meta            Meta                       `toml:"meta"`
	Package         Package                    `toml:"package"`
	Dependencies    map[string]string          `toml:"dependencies"`
	DevDependencies map[string]string          `toml:"dev-dependencies"`
	Runtime         map[string]RuntimeConfig   `toml:"runtime"`
	Scripts         map[string]string          `toml:"scripts"`
}

// LockMeta holds lockfile versioning metadata from the [meta] section of Prolfile.lock.
type LockMeta struct {
	LockVersion  int    `toml:"lock_version"`
	ProlfileHash string `toml:"prolfile_hash"`
}

// LockEntry represents a single resolved dependency in the lockfile.
// The Signature field is a placeholder for future package signing (see 8.11).
// Dependency names may be namespaced as owner/name (see 8.12).
type LockEntry struct {
	Name         string   `toml:"name"`
	Version      string   `toml:"version"`
	Source       string   `toml:"source"`
	URL          string   `toml:"url"`
	Checksum     string   `toml:"checksum"`
	Dependencies []string `toml:"dependencies"`
	Signature    string   `toml:"signature,omitempty"`
}

// LockFile represents a parsed Prolfile.lock.
type LockFile struct {
	Meta     LockMeta    `toml:"meta"`
	Packages []LockEntry `toml:"package"`
}

// Dependency represents a rich dependency specification.
// In Phase 1, dependencies use simple map[string]string (name -> version constraint).
// This struct supports Phase 2 features: path deps, source overrides.
// Names may be namespaced as owner/name (see 8.12).
type Dependency struct {
	Name     string `toml:"name"`
	Version  string `toml:"version,omitempty"`
	Path     string `toml:"path,omitempty"`
	Source   string `toml:"source,omitempty"`
	Registry string `toml:"registry,omitempty"`
}
