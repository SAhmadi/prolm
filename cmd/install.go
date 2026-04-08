package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
	"github.com/spf13/cobra"
)

// installRunner holds the seams used by the install command.
// Tests construct their own installRunner instead of mutating package globals,
// which eliminates the global-state race described in QUALITY-017.
type installRunner struct {
	newRegistry func() (registry.Registry, error)
	installOpts func() installer.Options
}

// defaultInstallRunner is the production runner used by the Cobra command.
var defaultInstallRunner = &installRunner{
	newRegistry: func() (registry.Registry, error) {
		return registry.NewSWIRegistry()
	},
	installOpts: func() installer.Options { return installer.Options{} },
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all dependencies from Prolfile.toml",
	Long: `Install resolves and installs every dependency declared in Prolfile.toml.

If Prolfile.lock exists and all locked packages are present in the local store,
the install is a fast no-op. Otherwise prolm queries the registry, downloads
each tarball, verifies its SHA-256 checksum, unpacks it into the store, and
writes a fresh Prolfile.lock.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultInstallRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// --frozen and --offline are Phase 2 features (CLAUDE.md §7 / QUALITY-015).
	// Registering them now as stubs lets scripts fail fast with a useful message
	// instead of cobra's "unknown flag" error.
	installCmd.Flags().Bool("frozen", false, "[Phase 2] Fail if Prolfile.lock would change (not yet implemented)")
	installCmd.Flags().Bool("offline", false, "[Phase 2] Use local cache only, no network requests (not yet implemented)")
	installCmd.Flags().Bool("no-verify", false, "[Phase 2] Skip checksum verification — DANGEROUS, dev only (not yet implemented)")
}

func (r *installRunner) run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Reject Phase 2 stub flags with a clear message rather than silently ignoring them.
	for _, flag := range []string{"frozen", "offline", "no-verify"} {
		if f := cmd.Flags().Lookup(flag); f != nil && f.Changed {
			return fmt.Errorf("--%s is not yet implemented; it will be available in Phase 2", flag)
		}
	}

	pf, manifestPath, err := loadManifest(cmd)
	if err != nil {
		return err
	}

	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		// §8.19: auto-heal a lockfile containing Git merge conflict markers
		// by discarding it and re-resolving from Prolfile.toml.
		var conflictErr *lockfile.GitConflictError
		if errors.As(err, &conflictErr) {
			ui.Warn("Lockfile had merge conflicts — re-resolved from Prolfile.toml")
			lock = nil
		} else {
			return fmt.Errorf("reading Prolfile.lock: %w", err)
		}
	}

	// Capture the on-disk bytes before the install so we can compare afterwards.
	// If the resulting lockfile is byte-identical to what is already on disk, we
	// skip the write to avoid spurious mtime/inode churn (QUALITY-016).
	var existingLockBytes []byte
	if existingData, readErr := os.ReadFile(lockPath); readErr == nil {
		existingLockBytes = existingData
	}

	reg, err := r.newRegistry()
	if err != nil {
		return fmt.Errorf("initializing registry: %w", err)
	}

	start := time.Now()
	newLock, err := installer.Install(ctx, pf, lock, reg, r.installOpts())
	if err != nil {
		return err
	}

	// Encode the new lockfile and compare against the current on-disk content.
	// Only write when the content has actually changed (QUALITY-016).
	newLockBytes, err := lockfile.Encode(newLock)
	if err != nil {
		return fmt.Errorf("encoding lockfile: %w", err)
	}

	if string(newLockBytes) != string(existingLockBytes) {
		if err := lockfile.Save(lockPath, newLock); err != nil {
			return fmt.Errorf("saving lockfile: %w", err)
		}
	}

	ui.Success("Installed %d packages in %s", len(newLock.Packages), time.Since(start).Round(time.Millisecond))
	return nil
}
