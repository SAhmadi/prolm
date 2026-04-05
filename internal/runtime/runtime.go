package runtime

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// RuntimeInfo holds the discovered binary path and parsed version string.
type RuntimeInfo struct {
	Path    string
	Version string
}

// Runtime is the interface all Prolog runtime backends must implement.
type Runtime interface {
	// Name returns the runtime identifier (e.g., "swi", "gnu", "scryer").
	Name() string

	// Detect searches for the runtime binary on the system.
	// On success it stores the result internally and returns RuntimeInfo.
	Detect() (*RuntimeInfo, error)

	// BuildRunArgs assembles command-line arguments for prolm run.
	// The binary path is NOT included; Exec prepends it.
	BuildRunArgs(entry string, deps []string, flags []string, goal string) []string

	// BuildTestArgs assembles arguments for prolm test (PlUnit).
	BuildTestArgs(testFiles []string, deps []string, flags []string) []string

	// BuildCheckArgs assembles arguments for prolm check (static analysis).
	BuildCheckArgs(files []string, deps []string) []string

	// Exec replaces the current process with the runtime binary (syscall.Exec).
	// Detect must be called first.
	Exec(args []string) error
}

// Detect is the factory function that returns a Runtime for the given name.
// An empty name defaults to "swi". Phase 1 only supports "swi".
func Detect(name string) (Runtime, error) {
	if name == "" {
		name = "swi"
	}
	switch name {
	case "swi":
		return NewSWIRuntime(), nil
	default:
		return nil, &ErrUnsupportedRuntime{Name: name}
	}
}

// CheckMinVersion verifies that info.Version satisfies the minVersion constraint.
// Returns nil if minVersion is empty or the version is sufficient.
// runtime is the identifier of the backend being checked (e.g. "swi", "gnu")
// and is included in ErrVersionTooOld so callers see an accurate error message.
func CheckMinVersion(info *RuntimeInfo, runtime, minVersion string) error {
	if minVersion == "" {
		return nil
	}
	found, err := semver.NewVersion(info.Version)
	if err != nil {
		return fmt.Errorf("parsing detected version %q: %w", info.Version, err)
	}
	required, err := semver.NewVersion(minVersion)
	if err != nil {
		return fmt.Errorf("parsing min_version %q: %w", minVersion, err)
	}
	if found.LessThan(required) {
		return &ErrVersionTooOld{
			Runtime:  runtime,
			Found:    info.Version,
			Required: minVersion,
		}
	}
	return nil
}

// --- Error types ---

// ErrRuntimeNotFound indicates the runtime binary was not found on PATH
// or any common installation location.
type ErrRuntimeNotFound struct {
	Runtime string
}

func (e *ErrRuntimeNotFound) Error() string {
	switch e.Runtime {
	case "swi":
		return "swipl not found on PATH.\n  Install SWI-Prolog: https://www.swi-prolog.org/download/stable\n  Or set PROLM_RUNTIME_PATH=/path/to/swipl"
	default:
		return fmt.Sprintf("%s runtime not found", e.Runtime)
	}
}

// ErrUnsupportedRuntime indicates the requested runtime name is not supported.
type ErrUnsupportedRuntime struct {
	Name string
}

func (e *ErrUnsupportedRuntime) Error() string {
	return fmt.Sprintf("unsupported runtime %q; supported runtimes: swi", e.Name)
}

// ErrVersionTooOld indicates the detected runtime version is below the
// minimum required by the project's [runtime.*] configuration.
type ErrVersionTooOld struct {
	Runtime  string
	Found    string
	Required string
}

func (e *ErrVersionTooOld) Error() string {
	return fmt.Sprintf("%s version %s found, but min_version %s is required", e.Runtime, e.Found, e.Required)
}

// ErrVersionParse indicates the runtime's --version output could not be parsed.
type ErrVersionParse struct {
	Output string
}

func (e *ErrVersionParse) Error() string {
	return fmt.Sprintf("could not parse version from swipl output: %q", e.Output)
}
