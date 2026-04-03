package manifest

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/prolm/prolm/pkg/prolfile"
)

// maxNameLen is the maximum allowed length for package and dependency names.
const maxNameLen = 128

// maxEntryLen is the maximum allowed length for the entry file path.
const maxEntryLen = 256

// validRuntimes is the set of supported Prolog runtimes.
var validRuntimes = map[string]bool{
	"swi":    true,
	"gnu":    true,
	"scryer": true,
}

// nameRe matches valid package/dependency identifiers.
// Supports optional namespace: owner/name. Both segments must be lowercase
// alphanumeric with internal hyphens only.
var nameRe = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?(/[a-z]([a-z0-9-]*[a-z0-9])?)?$`)

// ValidationError collects all validation failures so the user sees every
// problem at once rather than fixing them one at a time.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	switch len(e.Errors) {
	case 1:
		return fmt.Sprintf("validation error: %s", e.Errors[0])
	default:
		return fmt.Sprintf("validation errors:\n  - %s", strings.Join(e.Errors, "\n  - "))
	}
}

// Validate checks all fields of pf for correctness. It returns a
// *ValidationError containing every problem found, or nil if valid.
func Validate(pf *prolfile.ProlFile) error {
	var errs []string

	errs = validateName(errs, pf.Package.Name)
	errs = validateVersion(errs, pf.Package.Version)
	errs = validateEntry(errs, pf.Package.Entry)
	errs = validateRuntime(errs, pf.Package.Runtime)
	errs = validateDeps(errs, "dependencies", pf.Dependencies)
	errs = validateDeps(errs, "dev-dependencies", pf.DevDependencies)
	errs = validateRuntimeConfigs(errs, pf.Runtime)

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

func validateName(errs []string, name string) []string {
	if name == "" {
		return append(errs, "package name is required")
	}
	if containsNullByte(name) {
		return append(errs, "package name contains null byte")
	}
	if len(name) > maxNameLen {
		return append(errs, fmt.Sprintf("package name exceeds %d characters", maxNameLen))
	}
	if !nameRe.MatchString(name) {
		return append(errs, fmt.Sprintf("package name %q is invalid: must be lowercase alphanumeric with hyphens, optionally namespaced as owner/name", name))
	}
	return errs
}

func validateVersion(errs []string, version string) []string {
	if version == "" {
		return append(errs, "package version is required")
	}
	v := strings.TrimPrefix(version, "v")
	if _, err := semver.NewVersion(v); err != nil {
		return append(errs, fmt.Sprintf("package version %q is not valid semver: %v", version, err))
	}
	return errs
}

func validateEntry(errs []string, entry string) []string {
	if entry == "" {
		return errs // entry is optional
	}
	if containsNullByte(entry) {
		return append(errs, "entry path contains null byte")
	}
	if len(entry) > maxEntryLen {
		return append(errs, fmt.Sprintf("entry path exceeds %d characters", maxEntryLen))
	}
	if filepath.IsAbs(entry) {
		return append(errs, fmt.Sprintf("entry %q must be a relative path", entry))
	}
	if containsDotDot(entry) {
		return append(errs, fmt.Sprintf("entry %q must not contain '..' path traversal", entry))
	}
	if !strings.HasSuffix(entry, ".pl") {
		return append(errs, fmt.Sprintf("entry %q must end with .pl", entry))
	}
	return errs
}

func validateRuntime(errs []string, runtime string) []string {
	if runtime == "" {
		return errs // empty means default (swi)
	}
	if !validRuntimes[runtime] {
		return append(errs, fmt.Sprintf("runtime %q is not supported; must be one of: swi, gnu, scryer", runtime))
	}
	return errs
}

func validateDeps(errs []string, section string, deps map[string]string) []string {
	for name, constraint := range deps {
		if containsNullByte(name) {
			errs = append(errs, fmt.Sprintf("[%s] key contains null byte", section))
			continue
		}
		if len(name) > maxNameLen {
			errs = append(errs, fmt.Sprintf("[%s] key %q exceeds %d characters", section, name, maxNameLen))
			continue
		}
		if !nameRe.MatchString(name) {
			errs = append(errs, fmt.Sprintf("[%s] key %q is not a valid identifier", section, name))
		}
		if constraint != "*" {
			if _, err := semver.NewConstraint(constraint); err != nil {
				errs = append(errs, fmt.Sprintf("[%s] %s: version constraint %q is invalid: %v", section, name, constraint, err))
			}
		}
	}
	return errs
}

func validateRuntimeConfigs(errs []string, runtimes map[string]prolfile.RuntimeConfig) []string {
	for name := range runtimes {
		if !validRuntimes[name] {
			errs = append(errs, fmt.Sprintf("[runtime.%s] is not a supported runtime; must be one of: swi, gnu, scryer", name))
		}
	}
	return errs
}

func containsNullByte(s string) bool {
	return strings.ContainsRune(s, 0)
}

func containsDotDot(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
