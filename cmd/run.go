package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/internal/ui"
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

	// Resolve manifest (mirrors cmd/install.go discover-or-explicit pattern).
	configPath, _ := cmd.Flags().GetString("config")
	manifestPath := configPath
	if manifestPath == "" {
		discovered, err := manifest.Discover("")
		if err != nil {
			return fmt.Errorf("discovering Prolfile.toml: %w", err)
		}
		manifestPath = discovered
	}

	pf, err := manifest.Load(manifestPath, manifest.LoadOptions{ProlmVersion: Version})
	if err != nil {
		return fmt.Errorf("reading Prolfile.toml: %w", err)
	}

	// Resolve lockfile. nil lock => first install hasn't happened yet.
	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		return fmt.Errorf("reading Prolfile.lock: %w", err)
	}
	if lock == nil {
		ui.Hint("Run `prolm install` first to download dependencies")
		return fmt.Errorf("Prolfile.lock not found at %s", lockPath)
	}

	// Verify every locked package exists in the local store.
	store := installer.NewStore(r.storeDir)
	depPaths := make([]string, 0, len(lock.Packages))
	for _, pkg := range lock.Packages {
		if !store.IsInstalled(pkg.Name, pkg.Version) {
			ui.Hint("Run `prolm install` first to download dependencies")
			return fmt.Errorf("package %s@%s not installed in store", pkg.Name, pkg.Version)
		}
		depPaths = append(depPaths, store.PackPath(pkg.Name, pkg.Version))
	}

	// Resolve entry point.
	entry, err := resolveRunEntry(pf, manifestPath, entryArgs)
	if err != nil {
		return err
	}

	// Pick runtime: --runtime flag > [package].runtime > factory default.
	runtimeName, _ := cmd.Flags().GetString("runtime")
	if runtimeName == "" {
		runtimeName = pf.Package.Runtime
	}
	rt, err := r.newRuntime(runtimeName)
	if err != nil {
		return err
	}
	info, err := rt.Detect()
	if err != nil {
		return err
	}
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok && cfg.MinVersion != "" {
			if err := runtime.CheckMinVersion(info, rt.Name(), cfg.MinVersion); err != nil {
				return err
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

	return rt.Exec(rtArgs)
}

// resolveRunEntry picks the entry point file to load.
//
// Precedence:
//  1. No positional args => [package].entry from Prolfile.toml.
//  2. First positional arg is a path to a .pl file => use it.
//
// Named [scripts] entries are NOT yet supported (tracked as QUALITY-018);
// script values are free-text shell commands and require a small command
// parser/runner that is out of scope for sub-phase 1.11.
func resolveRunEntry(pf *prolfile.ProlFile, manifestPath string, args []string) (string, error) {
	if len(args) == 0 {
		entry := pf.Package.Entry
		if entry == "" {
			return "", fmt.Errorf("no entry point: pass a .pl file or set [package].entry in Prolfile.toml")
		}
		return resolveEntryPath(manifestPath, entry)
	}

	first := args[0]
	if pf.Scripts != nil {
		if _, ok := pf.Scripts[first]; ok {
			return "", fmt.Errorf("running named [scripts] entries is not yet supported (Phase 2)")
		}
	}
	return resolveEntryPath(manifestPath, first)
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
