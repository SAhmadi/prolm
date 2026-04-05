package cmd

import (
	"github.com/prolm/prolm/internal/scaffold"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new Prolog project",
	Long: `Create a new Prolog project in a new directory.

The project is initialized with a Prolfile.toml, source files, test files,
and a git repository.`,
	Example: `  prolm new my-expert-system
  prolm new my-app --template app
  prolm new my-app --runtime scryer`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		tmpl, _ := cmd.Flags().GetString("template")
		runtime, _ := cmd.Flags().GetString("runtime")
		if runtime == "" {
			runtime = "swi"
		}
		return scaffold.NewProject(name, tmpl, runtime)
	},
}

func init() {
	newCmd.Flags().String("template", "app", "project template: app | library | cli")
	rootCmd.AddCommand(newCmd)
}
