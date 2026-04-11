package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/prolm/prolm/internal/checker"
	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type checkCmdRunner struct {
	storeDir   string
	newRuntime func(name string) (runtime.Runtime, error)
	discover   func(projectDir string) ([]string, error)
	exec       checker.ExecFunc
}

var defaultCheckCmdRunner = &checkCmdRunner{
	newRuntime: runtime.NewRuntime,
	discover:   checker.Discover,
	exec:       checker.DefaultExec,
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Static analysis of Prolog source files",
	Long: `check loads all .pl files in the project via the configured Prolog runtime,
capturing and reporting warnings such as undefined predicates, singleton variables,
missing imports, and missing module declarations.

TRUST BOUNDARY: prolm check invokes the Prolog runtime, which executes
initialization directives (:- initialization/1), term_expansion/2, and
goal_expansion/2 hooks present in the project source and its locked
dependencies. It does NOT sandbox execution. Only run prolm check on
projects and dependencies you trust.

Use --no-deps to analyse only the project source files without loading
dependency modules (reduces execution scope at the cost of cross-module
analysis).

By default, warnings do not cause failure. Use --strict to treat warnings as errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultCheckCmdRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("strict", false, "Treat warnings as errors")
	checkCmd.Flags().Bool("no-deps", false, "Analyse project source only; do not load dependency modules")
	checkCmd.Flags().Duration("timeout", 30*time.Second, "Per-invocation timeout (default 30s)")
}

func (r *checkCmdRunner) run(cmd *cobra.Command, args []string) error {
	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}

	noDeps, _ := cmd.Flags().GetBool("no-deps")

	depPaths, err := verifyStoreDeps(manifestPath, r.storeDir)
	if err != nil {
		return err
	}
	if noDeps {
		depPaths = nil
	}

	// Runtime: --runtime flag > [package].runtime > factory default.
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
		if cfg, ok := pf.Runtime[rt.Name()]; ok {
			if cfg.MinVersion != "" {
				if err := runtime.CheckMinVersion(info, rt.Name(), cfg.MinVersion); err != nil {
					return fmt.Errorf("checking runtime version: %w", err)
				}
			}
		}
	}

	// BUG-014: use filepath.Dir so --config with any path works correctly,
	// instead of fragile string slicing that assumed the path ended with
	// "Prolfile.toml" at a fixed offset.
	srcFiles, err := r.discover(filepath.Dir(manifestPath))
	if err != nil {
		return fmt.Errorf("discovering source files: %w", err)
	}

	if len(srcFiles) == 0 {
		ui.Warn("No source files found (looked for *.pl, excluding *_test.pl and test_*.pl)")
		return nil
	}

	// BUG-015: forward [runtime.*].flags from Prolfile.toml to BuildCheckArgs.
	var runtimeFlags []string
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok {
			runtimeFlags = cfg.Flags
		}
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")

	c := &checker.Checker{Exec: r.exec}
	res, err := c.Check(context.Background(), info.Path, srcFiles, depPaths, rt, checker.Options{
		RuntimeFlags: runtimeFlags,
		Timeout:      timeout,
	})
	if err != nil {
		return err
	}

	strict, _ := cmd.Flags().GetBool("strict")
	opts := checker.ReportOptions{
		Strict:  strict,
		JSON:    viper.GetBool("json"),
		NoColor: viper.GetBool("no-color"),
	}
	failing := checker.Report(res, ui.Out, opts)
	if failing {
		if len(res.Errors()) > 0 {
			return fmt.Errorf("%d error(s) found", len(res.Errors()))
		}
		return fmt.Errorf("%d warning(s) found (--strict mode)", len(res.Warnings()))
	}
	return nil
}
