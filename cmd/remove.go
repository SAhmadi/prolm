package cmd

import (
	"fmt"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/spf13/cobra"
)

type removeRunner struct {
	sync func(pf *prolfile.ProlFile, manifestPath string) error
}

var defaultRemoveRunner = &removeRunner{
	sync: func(pf *prolfile.ProlFile, manifestPath string) error {
		return syncManifestDependencies(
			rootCmd.Context(),
			pf,
			manifestPath,
			func() (registry.Registry, error) { return registry.NewSWIRegistry() },
			func() installer.Options { return installer.Options{} },
			nil,
		)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <pack>",
	Short: "Remove a dependency and sync install/lockfile",
	Long: `remove deletes a dependency from [dependencies] in Prolfile.toml,
then runs the install flow to synchronize Prolfile.lock and local store state.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultRemoveRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func (r *removeRunner) run(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := manifest.ValidateName(name); err != nil {
		return err
	}

	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}
	original := cloneProlFile(pf)
	if pf.Dependencies == nil {
		pf.Dependencies = map[string]string{}
	}

	if _, ok := pf.Dependencies[name]; !ok {
		if _, inDev := pf.DevDependencies[name]; inDev {
			return fmt.Errorf(
				"dependency %q is declared in [dev-dependencies], not [dependencies]; --dev removal support lands in Phase 2",
				name,
			)
		}
		return fmt.Errorf(
			"dependency %q is not declared in [dependencies]; add it with `prolm add %s`",
			name, name,
		)
	}

	delete(pf.Dependencies, name)
	if err := manifest.Validate(pf); err != nil {
		return fmt.Errorf("updating Prolfile.toml: %w", err)
	}
	if err := manifest.Save(manifestPath, pf); err != nil {
		return fmt.Errorf("saving Prolfile.toml: %w", err)
	}
	if err := r.sync(pf, manifestPath); err != nil {
		_ = manifest.Save(manifestPath, original)
		return err
	}
	return nil
}
