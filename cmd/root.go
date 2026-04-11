package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "prolm",
	Short: "prolm — the Prolog project manager",
	Long: `prolm is a project manager and build tool for Prolog.

It provides dependency management, reproducible builds, and a unified
workflow across SWI-Prolog, GNU Prolog, and Scryer Prolog.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Bare global-flag invocation contract (Phase 1.5):
		// running `prolm --no-color` (or similar) with no subcommand should
		// clearly explain what happened, then show help.
		if len(args) == 0 {
			if cmd.Flags().Lookup("no-color").Changed ||
				cmd.Flags().Lookup("json").Changed ||
				cmd.Flags().Lookup("verbose").Changed ||
				cmd.Flags().Lookup("runtime").Changed ||
				cmd.Flags().Lookup("config").Changed {
				_, _ = cmd.OutOrStdout().Write([]byte("No command specified; showing help.\n\n"))
			}
			return cmd.Help()
		}
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Bind all flags (persistent + local) into Viper so they are
		// accessible via viper.GetString/GetBool throughout internal packages.
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}
		viper.AutomaticEnv()
		viper.SetEnvPrefix("PROLM") // PROLM_TOKEN, PROLM_VERBOSE, etc.

		// Respect NO_COLOR environment variable (https://no-color.org/).
		if os.Getenv("NO_COLOR") != "" {
			if err := cmd.Flags().Set("no-color", "true"); err == nil {
				viper.Set("no-color", true)
			}
		}

		return nil
	},
}

// Execute is the entry point called from main.go.
func Execute() error {
	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		printCommandError(executedCmd, err)
	}
	return err
}

func printCommandError(executedCmd *cobra.Command, err error) {
	_, _ = rootCmd.ErrOrStderr().Write([]byte(err.Error() + "\n"))
	if isUsageError(err) {
		usageCmd := usageCommand(executedCmd)
		_, _ = rootCmd.ErrOrStderr().Write([]byte("\n" + usageCmd.UsageString()))
	}
}

func usageCommand(executedCmd *cobra.Command) *cobra.Command {
	if executedCmd != nil {
		return executedCmd
	}
	return rootCmd
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, marker := range []string{
		"unknown command",
		"unknown flag",
		"accepts ",
		"requires at least",
		"requires at most",
		"requires exactly",
		"argument",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "path to Prolfile.toml (default: auto-discover upward)")
	rootCmd.PersistentFlags().String("runtime", "", "override runtime: swi | gnu | scryer")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose/debug output")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colour output (also respects NO_COLOR env var)")
	rootCmd.PersistentFlags().Bool("json", false, "machine-readable JSON output")
}
