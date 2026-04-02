package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/prolm/prolm/pkg/prolfile"
)

const prolfileName = "Prolfile.toml"

// ErrNotFound is returned when Discover cannot find a Prolfile.toml.
var ErrNotFound = errors.New("Prolfile.toml not found")

// VersionTooNewError indicates prolfile_version exceeds what this prolm supports.
type VersionTooNewError struct {
	Found     int
	Supported int
}

func (e *VersionTooNewError) Error() string {
	return fmt.Sprintf(
		"this Prolfile.toml requires a newer format (prolfile_version = %d), but prolm only supports version %d; run `prolm self-update` to upgrade",
		e.Found, e.Supported,
	)
}

// ProlmTooOldError indicates the running prolm version is below min_prolm_version.
type ProlmTooOldError struct {
	Required string
	Running  string
}

func (e *ProlmTooOldError) Error() string {
	return fmt.Sprintf(
		"this project requires prolm >= %s (you have %s); run `prolm self-update` or visit https://prolm.dev/install",
		e.Required, e.Running,
	)
}

// LoadOptions configures the behaviour of Load.
type LoadOptions struct {
	// ProlmVersion is the running prolm CLI version string (semver).
	// Used to check against [meta].min_prolm_version.
	// If empty, the min_prolm_version check is skipped.
	ProlmVersion string
}

// Load reads and parses a Prolfile.toml. If path is empty, it walks upward
// from the current working directory to find one. After parsing, it checks
// prolfile_version and min_prolm_version compatibility, then validates all fields.
func Load(path string, opts LoadOptions) (*prolfile.ProlFile, error) {
	if path == "" {
		discovered, err := Discover("")
		if err != nil {
			return nil, err
		}
		path = discovered
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var pf prolfile.ProlFile
	if err := toml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if err := checkProlfileVersion(pf.Meta.ProlfileVersion); err != nil {
		return nil, err
	}

	if err := checkMinProlmVersion(pf.Meta.MinProlmVersion, opts.ProlmVersion); err != nil {
		return nil, err
	}

	if err := Validate(&pf); err != nil {
		return nil, err
	}

	return &pf, nil
}

// Save writes a ProlFile to the given path with deterministic key ordering.
// It uses BurntSushi/toml's encoder which preserves struct field order and
// sorts map keys alphabetically.
func Save(path string, pf *prolfile.ProlFile) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(pf); err != nil {
		return fmt.Errorf("encoding Prolfile.toml: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// Discover walks upward from startDir looking for Prolfile.toml. If startDir
// is empty, the current working directory is used. Returns the absolute path
// to the first Prolfile.toml found, or ErrNotFound if none exists.
func Discover(startDir string) (string, error) {
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
		startDir = wd
	}

	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		candidate := filepath.Join(dir, prolfileName)
		info, err := os.Stat(candidate)
		switch {
		case err == nil && info.Mode().IsRegular():
			return candidate, nil
		case err == nil:
			// exists but is not a regular file (directory, etc.) — skip
		case os.IsNotExist(err):
			// not here, keep walking
		default:
			return "", fmt.Errorf("cannot access %s: %w", candidate, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

func checkProlfileVersion(version int) error {
	if version == 0 {
		return errors.New("Prolfile.toml is missing [meta].prolfile_version")
	}
	if version > prolfile.CurrentProlfileVersion {
		return &VersionTooNewError{
			Found:     version,
			Supported: prolfile.CurrentProlfileVersion,
		}
	}
	return nil
}

func checkMinProlmVersion(required, running string) error {
	if required == "" || running == "" {
		return nil
	}

	reqVer, err := semver.NewVersion(strings.TrimPrefix(required, "v"))
	if err != nil {
		return fmt.Errorf("invalid min_prolm_version %q: %w", required, err)
	}

	runVer, err := semver.NewVersion(strings.TrimPrefix(running, "v"))
	if err != nil {
		return fmt.Errorf("invalid prolm version %q: %w", running, err)
	}

	// Compare only major.minor.patch so that pre-release builds like
	// "0.1.0-dev" satisfy min_prolm_version = "0.1.0".
	runCore, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", runVer.Major(), runVer.Minor(), runVer.Patch()))

	if runCore.LessThan(reqVer) {
		return &ProlmTooOldError{
			Required: required,
			Running:  running,
		}
	}
	return nil
}
