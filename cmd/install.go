package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
	"github.com/spf13/cobra"
)

// newRegistry is a package-level seam so tests can inject a fake registry.
var newRegistry = func() (registry.Registry, error) {
	return registry.NewSWIRegistry()
}

// installOpts is a package-level seam so tests can redirect store/cache dirs
// away from the user's home directory.
var installOpts = func() installer.Options { return installer.Options{} }

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all dependencies from Prolfile.toml",
	Long: `Install resolves and installs every dependency declared in Prolfile.toml.

If Prolfile.lock exists and all locked packages are present in the local store,
the install is a fast no-op. Otherwise prolm queries the registry, downloads
each tarball, verifies its SHA-256 checksum, unpacks it into the store, and
writes a fresh Prolfile.lock.`,
	Args: cobra.NoArgs,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	configPath, _ := cmd.Flags().GetString("config")

	// Resolve the manifest path first so we can place Prolfile.lock beside it.
	manifestPath := configPath
	if manifestPath == "" {
		discovered, err := manifest.Discover("")
		if err != nil {
			return err
		}
		manifestPath = discovered
	}

	pf, err := manifest.Load(manifestPath, manifest.LoadOptions{ProlmVersion: Version})
	if err != nil {
		return err
	}

	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		return err
	}

	reg, err := newRegistry()
	if err != nil {
		return fmt.Errorf("initializing registry: %w", err)
	}

	start := time.Now()
	newLock, err := installer.Install(ctx, pf, lock, reg, installOpts())
	if err != nil {
		return err
	}

	if err := lockfile.Save(lockPath, newLock); err != nil {
		return fmt.Errorf("saving lockfile: %w", err)
	}

	ui.Success("Installed %d packages in %s", len(newLock.Packages), time.Since(start).Round(time.Millisecond))
	return nil
}
