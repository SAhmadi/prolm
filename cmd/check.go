package cmd

import (
	"context"
	"fmt"

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

By default, warnings do not cause failure. Use --strict to treat warnings as errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultCheckCmdRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("strict", false, "Treat warnings as errors")
}

func (r *checkCmdRunner) run(cmd *cobra.Command, args []string) error {
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
	if pf.Runtime != nil {
		if cfg, ok := pf.Runtime[rt.Name()]; ok {
			if cfg.MinVersion != "" {
				if err := runtime.CheckMinVersion(info, rt.Name(), cfg.MinVersion); err != nil {
					return fmt.Errorf("checking runtime version: %w", err)
				}
			}
		}
	}

	// Discover all source files (excluding test files).
	srcFiles, err := r.discover(manifestPath[:len(manifestPath)-len("Prolfile.toml")])
	if err != nil {
		return fmt.Errorf("discovering source files: %w", err)
	}

	if len(srcFiles) == 0 {
		ui.Warn("No source files found (looked for *.pl, excluding *_test.pl and test_*.pl)")
		return nil
	}

	c := &checker.Checker{Exec: r.exec}
	res, err := c.Check(context.Background(), info.Path, srcFiles, depPaths, rt, checker.Options{})
	if err != nil {
		return err
	}

	// Report diagnostics.
	reportCheck(res, viper.GetBool("json"))

	// Determine exit code.
	strict, _ := cmd.Flags().GetBool("strict")
	if len(res.Errors()) > 0 {
		return fmt.Errorf("%d error(s) found", len(res.Errors()))
	}
	if strict && len(res.Warnings()) > 0 {
		return fmt.Errorf("%d warning(s) found (--strict mode)", len(res.Warnings()))
	}

	return nil
}

// reportCheck formats and prints check output.
func reportCheck(res *checker.CheckResult, jsonMode bool) {
	if jsonMode {
		// JSON output: emit the full CheckResult.
		fmt.Fprintf(ui.Out, "{\"diagnostics\":[")
		for i, d := range res.Diagnostics {
			if i > 0 {
				fmt.Fprint(ui.Out, ",")
			}
			fmt.Fprintf(ui.Out, "{\"severity\":%q,\"file\":%q,\"line\":%d,\"col\":%d,\"message\":%q}",
				d.Severity, d.File, d.Line, d.Col, d.Message)
		}
		fmt.Fprintf(ui.Out, "]}\n")
		return
	}

	// Text output: print each diagnostic with colour.
	warnings := res.Warnings()
	errors := res.Errors()

	for _, d := range errors {
		ui.Error("%s:%d:%d: %s", d.File, d.Line, d.Col, d.Message)
	}
	for _, d := range warnings {
		ui.Warn("%s:%d:%d: %s", d.File, d.Line, d.Col, d.Message)
	}

	// Summary.
	if len(errors) == 0 && len(warnings) == 0 {
		ui.Success("No issues found")
	} else if len(errors) > 0 {
		ui.Error("%d error(s), %d warning(s)", len(errors), len(warnings))
	} else {
		ui.Info("%d warning(s)", len(warnings))
	}
}
