package cmd

import (
	"os"

	"github.com/prolm/prolm/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize prolm in an existing directory",
	Long: `Initialize a new Prolfile.toml in the current directory.

Use this to add prolm to an existing Prolog project. Unlike "prolm new",
this command does not create source files or directories.`,
	Example: `  prolm init
  prolm init --yes
  prolm init --yes --scan=false`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		scan, _ := cmd.Flags().GetBool("scan")

		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		return scaffold.InitProject(dir, scan, yes, os.Stdin)
	},
}

func init() {
	initCmd.Flags().Bool("yes", false, "accept all defaults non-interactively")
	initCmd.Flags().Bool("scan", true, "scan .pl files and auto-detect use_module deps")
	rootCmd.AddCommand(initCmd)
}
