package registry

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// PackageVersion holds metadata for a single version of a package
// as returned by a registry source.
type PackageVersion struct {
	Name         string
	Version      string
	URL          string
	Checksum     string // format: "sha1:<hex>" for SWI index; installer computes sha256 separately
	Dependencies []string
	Yanked       bool
}

// Registry is the interface that all package registry sources must implement.
// Phase 1 provides SWIRegistry; Phase 2+ adds GitHub and prolm registry.
type Registry interface {
	// Search returns packages whose name contains the search term.
	Search(ctx context.Context, query string) ([]PackageVersion, error)

	// Versions returns all known versions of a package, newest first.
	Versions(ctx context.Context, name string) ([]PackageVersion, error)

	// DownloadURL returns the HTTPS download URL for a specific version.
	DownloadURL(ctx context.Context, name, version string) (string, error)
}

// --- Error types ---

// ErrPackageNotFound indicates the registry has no package with the given name.
type ErrPackageNotFound struct {
	Name     string
	Registry string
}

func (e *ErrPackageNotFound) Error() string {
	return fmt.Sprintf("package %q not found in %s registry", e.Name, e.Registry)
}

// ErrVersionNotFound indicates the package exists but the requested version does not.
type ErrVersionNotFound struct {
	Name    string
	Version string
}

func (e *ErrVersionNotFound) Error() string {
	return fmt.Sprintf("version %q of package %q not found", e.Version, e.Name)
}

// ErrHTTPSRequired is returned when a URL uses http:// instead of https:// (SEC-4).
type ErrHTTPSRequired struct {
	URL string
}

func (e *ErrHTTPSRequired) Error() string {
	return fmt.Sprintf("HTTPS required but got insecure URL: %s", e.URL)
}

// ErrRateLimited is returned after exhausting all retry attempts on 429 responses (SEC-10).
type ErrRateLimited struct {
	RetryAfter time.Duration
}

func (e *ErrRateLimited) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited by registry; retry after %s", e.RetryAfter)
	}
	return "rate limited by registry; all retry attempts exhausted"
}

// ErrInvalidResponse is returned when registry responses cannot be parsed
// or contain invalid data (SEC-9).
type ErrInvalidResponse struct {
	Reason string
}

func (e *ErrInvalidResponse) Error() string {
	return fmt.Sprintf("invalid registry response: %s", e.Reason)
}

// --- Validation helpers ---

// nameRe matches valid package identifiers from registry responses.
// Slightly more lenient than manifest/validate.go: allows underscores because
// many SWI-Prolog packs use them (e.g., pro_sqlite3, list_util).
// Supports optional namespace: owner/name.
var nameRe = regexp.MustCompile(`^[a-z]([a-z0-9_-]*[a-z0-9])?(/[a-z]([a-z0-9_-]*[a-z0-9])?)?$`)

// sha1Re matches a valid lowercase hex SHA-1 hash (40 characters).
var sha1Re = regexp.MustCompile(`^[0-9a-f]{40}$`)

// maxNameLen is the maximum allowed length for package names.
const maxNameLen = 128

// ValidateHTTPS checks that rawURL uses the https scheme (SEC-4).
// Returns *ErrHTTPSRequired if the URL is not HTTPS.
func ValidateHTTPS(rawURL string) error {
	if rawURL == "" {
		return &ErrInvalidResponse{Reason: "empty URL"}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return &ErrInvalidResponse{Reason: fmt.Sprintf("malformed URL: %s", rawURL)}
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return &ErrHTTPSRequired{URL: rawURL}
	}
	return nil
}

// validatePackageName checks a package name extracted from a registry response (SEC-9).
func validatePackageName(name string) error {
	if name == "" {
		return &ErrInvalidResponse{Reason: "empty package name"}
	}
	if strings.ContainsRune(name, 0) {
		return &ErrInvalidResponse{Reason: "package name contains null byte"}
	}
	if len(name) > maxNameLen {
		return &ErrInvalidResponse{Reason: fmt.Sprintf("package name exceeds %d characters", maxNameLen)}
	}
	if !nameRe.MatchString(name) {
		return &ErrInvalidResponse{Reason: fmt.Sprintf("invalid package name: %q", name)}
	}
	return nil
}

// validateSHA1 checks that a checksum string is a valid 40-char hex SHA-1.
func validateSHA1(hash string) bool {
	return sha1Re.MatchString(hash)
}
