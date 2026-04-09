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

// checkCmdRunner holds the seams used by `prolm check` so tests can substitute
// a fake runtime, discovery function, and exec function without touching real
// swipl. Mirrors testCmdRunner.
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
	Short: "Run static analysis on the project's Prolog source files",
	Long: `check loads every .pl file in the project (excluding test files) into
the configured Prolog runtime with every locked dependency available, and
reports warnings and errors emitted by the runtime: undefined predicates,
singleton variables, missing imports, syntax errors, and load-time goal
failures. Exits with status 1 on any error, or on any warning when --strict
is passed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultCheckCmdRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("strict", false, "Treat warnings as errors")
	checkCmd.Flags().Duration("timeout", 30*time.Second, "Per-invocation timeout")
}

func (r *checkCmdRunner) run(cmd *cobra.Command, _ []string) error {
	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}

	depPaths, err := verifyStoreDeps(manifestPath, r.storeDir)
	if err != nil {
		return err
	}

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

	projectDir := filepath.Dir(manifestPath)
	srcFiles, err := r.discover(projectDir)
	if err != nil {
		return fmt.Errorf("discovering source files: %w", err)
	}
	if len(srcFiles) == 0 {
		ui.Warn("No Prolog source files found to check")
		return nil
	}

	strict, _ := cmd.Flags().GetBool("strict")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	chk := &checker.Checker{Exec: r.exec}
	res, err := chk.Check(context.Background(), info.Path, srcFiles, depPaths, rt, checker.Options{
		Timeout: timeout,
	})
	if err != nil {
		return err
	}

	fail := checker.Report(res, ui.Out, checker.ReportOptions{
		Strict:  strict,
		JSON:    viper.GetBool("json"),
		NoColor: ui.NoColor(),
	})
	if fail {
		return fmt.Errorf("check failed: %d error(s), %d warning(s)",
			len(res.Errors()), len(res.Warnings()))
	}
	return nil
}
