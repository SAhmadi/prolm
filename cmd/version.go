package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current prolm version. Release builds override it with ldflags.
var Version = "0.1.1-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the prolm version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "prolm %s\n", Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
