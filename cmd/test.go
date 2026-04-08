package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/internal/testrunner"
	"github.com/prolm/prolm/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// testCmdRunner holds the seams used by `prolm test` so unit tests can
// substitute a fake runtime, discovery function, and exec function without
// touching the real swipl binary.
type testCmdRunner struct {
	storeDir   string
	newRuntime func(name string) (runtime.Runtime, error)
	discover   func(projectDir string) ([]string, error)
	exec       testrunner.ExecFunc
}

var defaultTestCmdRunner = &testCmdRunner{
	newRuntime: runtime.NewRuntime,
	discover:   testrunner.Discover,
	exec:       testrunner.DefaultExec,
}

var testCmd = &cobra.Command{
	Use:   "test [file]",
	Short: "Discover and run PlUnit tests",
	Long: `test discovers **/*_test.pl and **/test_*.pl files in the project,
loads them into the configured Prolog runtime together with every locked
dependency, and runs the PlUnit suites. Exits with status 1 on any failure.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultTestCmdRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().String("filter", "", "Only run tests whose name matches this substring")
	testCmd.Flags().Bool("verbose", false, "Print each passing test, not just failures")
	testCmd.Flags().Duration("timeout", 30*time.Second, "Per-invocation timeout")
}

func (r *testCmdRunner) run(cmd *cobra.Command, args []string) error {
	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}

	depPaths, err := verifyStoreDeps(manifestPath, r.storeDir)
	if err != nil {
		return err
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
	var flags []string
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok {
			if cfg.MinVersion != "" {
				if err := runtime.CheckMinVersion(info, rt.Name(), cfg.MinVersion); err != nil {
					return fmt.Errorf("checking runtime version: %w", err)
				}
			}
			flags = cfg.Flags
		}
	}

	// Resolve test files: single positional file argument, or full discovery.
	projectDir := filepath.Dir(manifestPath)
	var testFiles []string
	if len(args) == 1 {
		p := args[0]
		if !filepath.IsAbs(p) {
			p = filepath.Join(projectDir, p)
		}
		if _, statErr := os.Stat(p); statErr != nil {
			return fmt.Errorf("test file %q not found: %w", args[0], statErr)
		}
		testFiles = []string{p}
	} else {
		testFiles, err = r.discover(projectDir)
		if err != nil {
			return fmt.Errorf("discovering test files: %w", err)
		}
	}

	if len(testFiles) == 0 {
		ui.Warn("No test files found (looked for *_test.pl and test_*.pl)")
		return nil
	}

	filter, _ := cmd.Flags().GetString("filter")
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	runner := &testrunner.Runner{Exec: r.exec}
	res, err := runner.Run(context.Background(), info.Path, testFiles, depPaths, rt, flags, testrunner.Options{
		Filter:  filter,
		Verbose: verbose,
		Timeout: timeout,
	})
	if err != nil {
		return err
	}

	testrunner.Report(res, ui.Out, testrunner.ReportOptions{
		Verbose: verbose,
		JSON:    viper.GetBool("json"),
		NoColor: ui.NoColor(),
	})

	if res.FailedCount() > 0 {
		return fmt.Errorf("%d test(s) failed", res.FailedCount())
	}
	return nil
}
