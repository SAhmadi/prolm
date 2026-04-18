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
and a git repository.

Templates:
  app      Available in Phase 1 / Phase 1.5. Creates an executable-style
           project with src/main.pl and a matching test file.
  library  Planned for Phase 2. Will create a reusable module/library layout.
  cli      Planned for Phase 2. Will create a command-line oriented project
           with argument handling patterns.`,
	Example: `  prolm new my-expert-system
  prolm new my-app --template app
  prolm new my-app --runtime scryer`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		tmpl, _ := cmd.Flags().GetString("template")
		runtime, _ := cmd.Flags().GetString("runtime")
		// --runtime is a global flag; default to SWI for `prolm new` when unset.
		if runtime == "" {
			runtime = "swi"
		}
		return scaffold.NewProject(name, tmpl, runtime)
	},
}

func init() {
	newCmd.Flags().String("runtime", "swi", "target runtime: swi | gnu | scryer")
	newCmd.Flags().String("template", "app", "project template: app (Phase 1 / 1.5) | library (Phase 2) | cli (Phase 2)")
	rootCmd.AddCommand(newCmd)
}
