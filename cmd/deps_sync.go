package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
	"github.com/prolm/prolm/pkg/prolfile"
)

// syncManifestDependencies installs dependencies from pf and writes Prolfile.lock.
// add/remove call this after mutating Prolfile.toml to keep manifest+lock in sync.
func syncManifestDependencies(
	ctx context.Context,
	pf *prolfile.ProlFile,
	manifestPath string,
	newRegistry func() (registry.Registry, error),
	installOpts func() installer.Options,
	sourceOverrides map[string]string,
) error {
	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		var conflictErr *lockfile.GitConflictError
		if errors.As(err, &conflictErr) {
			ui.Warn("Lockfile had merge conflicts — re-resolved from Prolfile.toml")
			lock = nil
		} else {
			return fmt.Errorf("reading Prolfile.lock: %w", err)
		}
	}

	reg, err := newRegistry()
	if err != nil {
		return fmt.Errorf("initializing registry: %w", err)
	}
	if len(sourceOverrides) > 0 {
		reg = &sourceOverrideRegistry{
			base:      reg,
			overrides: sourceOverrides,
		}
	}

	newLock, err := installer.Install(ctx, pf, lock, reg, installOpts())
	if err != nil {
		return err
	}
	if err := lockfile.Save(lockPath, newLock); err != nil {
		return fmt.Errorf("saving lockfile: %w", err)
	}
	return nil
}

// sourceOverrideRegistry allows add <url> to install from a direct tarball URL
// while keeping the rest of resolution on the default SWI registry.
type sourceOverrideRegistry struct {
	base      registry.Registry
	overrides map[string]string
}

func (r *sourceOverrideRegistry) Search(ctx context.Context, query string) ([]registry.PackageVersion, error) {
	return r.base.Search(ctx, query)
}

func (r *sourceOverrideRegistry) Versions(ctx context.Context, name string) ([]registry.PackageVersion, error) {
	if u, ok := r.overrides[name]; ok {
		return []registry.PackageVersion{{
			Name:    name,
			Version: "0.0.0",
			URL:     u,
		}}, nil
	}
	return r.base.Versions(ctx, name)
}

func (r *sourceOverrideRegistry) DownloadURL(ctx context.Context, name, version string) (string, error) {
	if u, ok := r.overrides[name]; ok {
		return u, nil
	}
	return r.base.DownloadURL(ctx, name, version)
}
