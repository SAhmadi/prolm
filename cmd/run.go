package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/spf13/cobra"
)

// runRunner holds the seams used by the run command so tests can substitute
// the store directory and runtime factory without mutating package globals.
type runRunner struct {
	storeDir   string
	newRuntime func(name string) (runtime.Runtime, error)
}

var defaultRunRunner = &runRunner{
	storeDir:   "",
	newRuntime: runtime.NewRuntime,
}

var runCmd = &cobra.Command{
	Use:   "run [entry] [-- prolog-args...]",
	Short: "Run the project entry point with all dependencies loaded",
	Long: `run invokes the configured Prolog runtime with every locked
dependency loaded via use_module/1, then evaluates the project entry
point and the configured goal.

If no entry argument is given, [package].entry from Prolfile.toml is used.
A positional argument is treated as a path to a .pl file.

Use '--' to separate arguments destined for the Prolog runtime, e.g.

    prolm run -- --mode interactive`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultRunRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().String("goal", "main", "Prolog goal to call after loading the entry")
}

func (r *runRunner) run(cmd *cobra.Command, args []string) error {
	// Split positional args at '--': everything before is consumed by prolm,
	// everything after is forwarded verbatim to the Prolog runtime.
	dashAt := cmd.ArgsLenAtDash()
	var entryArgs, prologArgs []string
	if dashAt < 0 {
		entryArgs = args
	} else {
		entryArgs = args[:dashAt]
		prologArgs = args[dashAt:]
	}

	// Resolve manifest via shared helper (DRY-003).
	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}

	// Resolve lockfile and verify every locked package exists in the local store.
	depPaths, err := verifyStoreDeps(manifestPath, r.storeDir)
	if err != nil {
		return err
	}

	// Resolve entry point (and any prolog args embedded in a [scripts] value).
	entry, scriptPrologArgs, err := resolveRunEntry(pf, manifestPath, entryArgs)
	if err != nil {
		return err
	}
	// Script prolog args come first so that explicit user args (after --) win.
	prologArgs = append(scriptPrologArgs, prologArgs...)

	// Pick runtime: --runtime flag > [package].runtime > factory default.
	runtimeName, _ := cmd.Flags().GetString("runtime")
	if runtimeName == "" {
		runtimeName = pf.Package.Runtime
	}
	rt, err := r.newRuntime(runtimeName)
	if err != nil {
		return fmt.Errorf("selecting runtime %q: %w", runtimeName, err)
	}
	info, err := rt.Detect()
	if err != nil {
		return fmt.Errorf("detecting %s runtime: %w", rt.Name(), err)
	}
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok && cfg.MinVersion != "" {
			if err := runtime.CheckMinVersion(info, rt.Name(), cfg.MinVersion); err != nil {
				return fmt.Errorf("checking runtime version: %w", err)
			}
		}
	}

	// Assemble args and exec.
	var flags []string
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok {
			flags = cfg.Flags
		}
	}
	goal, _ := cmd.Flags().GetString("goal")

	rtArgs := rt.BuildRunArgs(entry, depPaths, flags, goal)
	rtArgs = append(rtArgs, prologArgs...)

	if err := rt.Exec(rtArgs); err != nil {
		return fmt.Errorf("executing %s: %w", rt.Name(), err)
	}
	return nil
}

// resolveRunEntry picks the entry point file to load and any extra Prolog args
// that come from a [scripts] entry. The third return value is non-nil only when
// a named script with embedded "-- <args>" is resolved.
//
// Precedence:
//  1. No positional args => [package].entry from Prolfile.toml.
//  2. First positional arg matches a [scripts] key => parse as "prolm run <entry> [-- <args>]".
//  3. First positional arg is a path to a .pl file => use it directly.
func resolveRunEntry(pf *prolfile.ProlFile, manifestPath string, args []string) (entry string, scriptPrologArgs []string, err error) {
	if len(args) == 0 {
		e := pf.Package.Entry
		if e == "" {
			return "", nil, fmt.Errorf("no entry point: pass a .pl file or set [package].entry in Prolfile.toml")
		}
		resolved, err := resolveEntryPath(manifestPath, e)
		return resolved, nil, err
	}

	first := args[0]
	if pf.Scripts != nil {
		if scriptVal, ok := pf.Scripts[first]; ok {
			return resolveScriptEntry(manifestPath, first, scriptVal)
		}
	}
	resolved, err := resolveEntryPath(manifestPath, first)
	return resolved, nil, err
}

// resolveScriptEntry parses a [scripts] value of the form
// "prolm run <entry> [-- <prolog-args>...]" and returns the resolved entry
// path together with any Prolog args after the "--" separator.
//
// Only "prolm run ..." script values are supported. Any other format is
// rejected with a clear error so users understand the expected syntax.
func resolveScriptEntry(manifestPath, scriptName, scriptVal string) (string, []string, error) {
	// Tokenise by splitting on spaces; we do not need full shell quoting
	// because Prolfile.toml script values follow a controlled syntax.
	tokens := splitScriptTokens(scriptVal)

	// Expect: "prolm" "run" <entry> [-- <prolog-args>...]
	if len(tokens) < 3 || tokens[0] != "prolm" || tokens[1] != "run" {
		return "", nil, fmt.Errorf(
			"script %q value %q is not in the expected format: must start with \"prolm run <entry>\"",
			scriptName, scriptVal,
		)
	}

	entryToken := tokens[2]
	rest := tokens[3:]

	// Find "--" separator if present.
	var scriptPrologArgs []string
	for i, tok := range rest {
		if tok == "--" {
			scriptPrologArgs = rest[i+1:]
			break
		}
	}

	resolved, err := resolveEntryPath(manifestPath, entryToken)
	if err != nil {
		return "", nil, err
	}
	return resolved, scriptPrologArgs, nil
}

// splitScriptTokens splits a script string on whitespace runs. This is
// intentionally simple: Prolfile.toml scripts follow a restricted syntax
// ("prolm run <entry> [-- args...]") that does not require shell quoting.
func splitScriptTokens(s string) []string {
	var tokens []string
	start := -1
	for i := 0; i <= len(s); i++ {
		isSpace := i == len(s) || s[i] == ' ' || s[i] == '\t'
		if isSpace {
			if start >= 0 {
				tokens = append(tokens, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	return tokens
}

// resolveEntryPath resolves an entry path relative to the manifest directory
// and verifies that the file exists. Absolute paths are accepted as-is.
func resolveEntryPath(manifestPath, entry string) (string, error) {
	full := entry
	if !filepath.IsAbs(full) {
		full = filepath.Join(filepath.Dir(manifestPath), entry)
	}
	if _, err := os.Stat(full); err != nil {
		return "", fmt.Errorf("entry file %q not found: %w", entry, err)
	}
	return full, nil
}
