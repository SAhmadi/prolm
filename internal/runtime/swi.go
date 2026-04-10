package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

// swiVersionRe matches the version number in swipl --version output.
// Example: "SWI-Prolog version 9.2.1 for x86_64-darwin"
var swiVersionRe = regexp.MustCompile(`SWI-Prolog version (\d+\.\d+\.\d+)`)

// swiFallbackPaths lists common swipl install locations checked when
// the binary is not on PATH.
var swiFallbackPaths = []string{
	"/usr/local/bin/swipl",
	"/opt/homebrew/bin/swipl",
	"/snap/bin/swipl",
	"/usr/bin/swipl",
}

// SWIRuntime implements the Runtime interface for SWI-Prolog.
type SWIRuntime struct {
	// lookPath locates a binary on PATH. Defaults to exec.LookPath.
	lookPath func(string) (string, error)

	// runCmd executes a command and returns combined output.
	// Defaults to a wrapper around exec.Command.CombinedOutput().
	runCmd func(name string, args ...string) ([]byte, error)

	// execReplace replaces the current process. Defaults to syscall.Exec.
	execReplace func(binary string, argv []string, env []string) error

	// fileExists reports whether path exists and is a regular file.
	// Defaults to an os.Stat-based check.
	fileExists func(path string) bool

	// info is populated by Detect and used by Exec.
	info *RuntimeInfo
}

// NewSWIRuntime creates a SWIRuntime with production defaults.
func NewSWIRuntime() *SWIRuntime {
	return &SWIRuntime{
		lookPath: exec.LookPath,
		runCmd: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		},
		execReplace: syscall.Exec,
		fileExists: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && !info.IsDir()
		},
	}
}

// Name returns "swi".
func (s *SWIRuntime) Name() string { return "swi" }

// Info returns the RuntimeInfo populated by Detect, or nil if Detect
// has not been called yet.
func (s *SWIRuntime) Info() *RuntimeInfo { return s.info }

// Detect searches for swipl and parses its version.
//
// Search order:
//  1. PROLM_RUNTIME_PATH env var
//  2. exec.LookPath("swipl") (PATH)
//  3. Fallback locations: /usr/local/bin, /opt/homebrew/bin, /snap/bin, /usr/bin
func (s *SWIRuntime) Detect() (*RuntimeInfo, error) {
	path, err := s.findBinary()
	if err != nil {
		return nil, err
	}

	version, err := s.parseVersion(path)
	if err != nil {
		return nil, err
	}

	s.info = &RuntimeInfo{Path: path, Version: version}
	return s.info, nil
}

// findBinary locates the swipl binary using the search order.
func (s *SWIRuntime) findBinary() (string, error) {
	// 1. Explicit env override.
	if envPath := os.Getenv("PROLM_RUNTIME_PATH"); envPath != "" {
		if s.fileExists(envPath) {
			return envPath, nil
		}
		return "", fmt.Errorf("PROLM_RUNTIME_PATH=%q: file not found", envPath)
	}

	// 2. PATH lookup.
	if p, err := s.lookPath("swipl"); err == nil {
		return p, nil
	}

	// 3. Fallback locations.
	for _, p := range swiFallbackPaths {
		if s.fileExists(p) {
			return p, nil
		}
	}

	return "", &ErrRuntimeNotFound{Runtime: "swi"}
}

// parseVersion runs swipl --version and extracts the semver string.
func (s *SWIRuntime) parseVersion(path string) (string, error) {
	out, err := s.runCmd(path, "--version")
	if err != nil {
		return "", fmt.Errorf("running %s --version: %w", path, err)
	}

	output := strings.TrimSpace(string(out))
	matches := swiVersionRe.FindStringSubmatch(output)
	if len(matches) < 2 {
		return "", &ErrVersionParse{Output: output}
	}
	return matches[1], nil
}

// escapePrologAtom escapes a string for use inside a single-quoted Prolog atom
// by replacing every single-quote with two single-quotes (ISO 6.3.3.3).
func escapePrologAtom(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// BuildRunArgs assembles swipl arguments for prolm run.
//
// Result: [flags...] [-g "use_module('dep')" ...] -g "use_module('entry')" -g "goal" -t halt
func (s *SWIRuntime) BuildRunArgs(entry string, deps []string, flags []string, goal string) []string {
	var args []string

	args = append(args, flags...)

	for _, dep := range deps {
		args = append(args, "-g", fmt.Sprintf("use_module('%s')", escapePrologAtom(dep)))
	}

	args = append(args, "-g", fmt.Sprintf("use_module('%s')", escapePrologAtom(entry)))
	args = append(args, "-g", goal)
	args = append(args, "-t", "halt")

	return args
}

// BuildTestArgs assembles swipl arguments for prolm test (PlUnit).
//
// Result: [flags...] -g "use_module(library(plunit))" [-g "use_module('dep')" ...]
//
//	[-g "load_files(['test'],[if(true)])" ...] -g "run_tests" -t halt
func (s *SWIRuntime) BuildTestArgs(testFiles []string, deps []string, flags []string) []string {
	var args []string

	args = append(args, flags...)

	// Load PlUnit framework.
	args = append(args, "-g", "use_module(library(plunit))")

	for _, dep := range deps {
		args = append(args, "-g", fmt.Sprintf("use_module('%s')", escapePrologAtom(dep)))
	}

	for _, tf := range testFiles {
		args = append(args, "-g", fmt.Sprintf("load_files(['%s'],[if(true)])", escapePrologAtom(tf)))
	}

	args = append(args, "-g", "run_tests")
	args = append(args, "-t", "halt")

	return args
}

// BuildCheckArgs assembles swipl arguments for prolm check (static analysis).
//
// BUG-015: flags from [runtime.swi].flags are prepended (mirrors BuildTestArgs).
// BUG-016: load_files/2 is used with [if(true),autoload(false)] instead of
// bare load_files/1. The autoload(false) option prevents swipl from
// auto-loading undefined predicates during the analysis pass, which reduces
// the execution surface. Note: :- initialization(Goal) directives present in
// the loaded source are still executed by swipl; this is an inherent
// limitation of SWI-Prolog's load mechanism documented in CLAUDE.md §8.15.
//
// Result: [flags...] [-g "use_module('dep')" ...] [-g "load_files(['file'],[if(true),autoload(false)])" ...] -t halt
func (s *SWIRuntime) BuildCheckArgs(files []string, deps []string, flags []string) []string {
	var args []string

	args = append(args, flags...)

	for _, dep := range deps {
		args = append(args, "-g", fmt.Sprintf("use_module('%s')", escapePrologAtom(dep)))
	}

	for _, f := range files {
		args = append(args, "-g", fmt.Sprintf("load_files(['%s'],[if(true),autoload(false)])", escapePrologAtom(f)))
	}

	args = append(args, "-t", "halt")

	return args
}

// Exec replaces the current process with swipl via syscall.Exec.
// Detect must be called successfully before Exec.
func (s *SWIRuntime) Exec(args []string) error {
	if s.info == nil {
		return fmt.Errorf("runtime not detected; call Detect() first")
	}
	argv := make([]string, 0, 1+len(args))
	argv = append(argv, s.info.Path)
	argv = append(argv, args...)
	return s.execReplace(s.info.Path, argv, os.Environ())
}
