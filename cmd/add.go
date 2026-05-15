package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/prolm/prolm/internal/installer"
	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/internal/ui"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/spf13/cobra"
)

type addTarget struct {
	Name          string
	SourceURL     string
	SourceVersion string
	GitHubRepoRef string
}

type addRunner struct {
	resolveGitHubRepo func(ctx context.Context, repoRef string) (registry.PackageVersion, error)
	sync              func(pf *prolfile.ProlFile, manifestPath string, sourceOverrides map[string]registry.PackageVersion) error
}

var defaultAddRunner = &addRunner{
	resolveGitHubRepo: func(ctx context.Context, repoRef string) (registry.PackageVersion, error) {
		return newGitHubRepoResolver().Resolve(ctx, repoRef)
	},
	sync: func(pf *prolfile.ProlFile, manifestPath string, sourceOverrides map[string]registry.PackageVersion) error {
		return syncManifestDependencies(
			rootCmd.Context(),
			pf,
			manifestPath,
			func() (registry.Registry, error) { return registry.NewSWIRegistry() },
			func() installer.Options { return installer.Options{} },
			sourceOverrides,
		)
	},
}

var addCmd = &cobra.Command{
	Use:   "add <name|url>",
	Short: "Add a dependency and sync install/lockfile",
	Long: `add adds a dependency to [dependencies] in Prolfile.toml, then runs
the install flow to synchronize Prolfile.lock and local store state.

Supported inputs in Phase 1.5:
  - SWI package names (for example: aop)
  - SWI package URLs
  - GitHub package URLs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultAddRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func (r *addRunner) run(cmd *cobra.Command, args []string) error {
	target, err := parseAddTarget(args[0])
	if err != nil {
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
	if _, exists := pf.Dependencies[target.Name]; exists {
		ui.Warn("Dependency %q already exists in [dependencies]", target.Name)
		ui.Hint("Run `prolm install` if you want to refresh local packages")
		return nil
	}
	if target.GitHubRepoRef != "" {
		pv, err := r.resolveGitHubRepo(cmd.Context(), target.GitHubRepoRef)
		if err != nil {
			return fmt.Errorf("resolving github repository %s: %w", target.GitHubRepoRef, err)
		}
		target.SourceURL = pv.URL
		target.SourceVersion = pv.Version
	}

	pf.Dependencies[target.Name] = "*"
	if err := manifest.Validate(pf); err != nil {
		return fmt.Errorf("updating Prolfile.toml: %w", err)
	}

	overrides := map[string]registry.PackageVersion{}
	if target.SourceURL != "" {
		sourceVersion := target.SourceVersion
		if sourceVersion == "" {
			sourceVersion = inferStableVersion(target.SourceURL)
		}
		if sourceVersion == "" {
			sourceVersion = "0.0.0"
		}
		overrides[target.Name] = registry.PackageVersion{
			Name:    target.Name,
			Version: sourceVersion,
			URL:     target.SourceURL,
		}
	}
	if err := r.sync(pf, manifestPath, overrides); err != nil {
		return err
	}

	version, err := installedVersionFor(manifestPath, target.Name)
	if err != nil {
		return err
	}
	if version == "" {
		version = target.SourceVersion
	}
	constraint := "*"
	switch {
	case version == "" || version == "0.0.0":
		ui.Warn("Could not resolve a stable version for %s; leaving dependency constraint as \"*\"", target.Name)
	case manifest.ValidateVersion(version) != nil:
		ui.Warn("Resolved version %q for %s is not valid semver; leaving dependency constraint as \"*\"", version, target.Name)
	default:
		constraint = "^" + version
	}
	pf.Dependencies[target.Name] = constraint
	if err := manifest.Validate(pf); err != nil {
		return fmt.Errorf("pinning dependency version: %w", err)
	}
	if err := manifest.Save(manifestPath, pf); err != nil {
		if saveErr := manifest.Save(manifestPath, original); saveErr != nil {
			return fmt.Errorf("saving Prolfile.toml: %w (rollback failed: %v)", err, saveErr)
		}
		return fmt.Errorf("saving Prolfile.toml: %w", err)
	}

	ui.Success("Added %s", target.Name)
	return nil
}

func parseAddTarget(raw string) (addTarget, error) {
	if !strings.Contains(raw, "://") {
		if err := manifest.ValidateName(raw); err != nil {
			return addTarget{}, err
		}
		return addTarget{Name: raw}, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return addTarget{}, fmt.Errorf("invalid dependency URL %q: %w", raw, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return addTarget{}, fmt.Errorf("dependency URLs must use HTTPS: %s", raw)
	}

	host := strings.ToLower(u.Hostname())
	switch host {
	case "www.swi-prolog.org", "swi-prolog.org":
		name := strings.TrimSpace(u.Query().Get("p"))
		if name == "" {
			name = deriveNameFromPathOrQuery(u)
		}
		if err := manifest.ValidateName(name); err != nil {
			return addTarget{}, fmt.Errorf("could not infer package name from SWI URL %q: %w", raw, err)
		}
		// If this is a listing URL with ?p=<name>, resolve through SWI index by name.
		if strings.TrimSpace(u.Query().Get("p")) != "" {
			return addTarget{Name: name}, nil
		}
		return addTarget{Name: name, SourceURL: raw, SourceVersion: inferStableVersion(raw)}, nil

	case "github.com":
		parts := splitPathParts(u.Path)
		if len(parts) < 2 {
			return addTarget{}, fmt.Errorf("github URL must include owner/repo: %s", raw)
		}
		owner := parts[0]
		repo := strings.TrimSuffix(parts[1], ".git")
		if err := manifest.ValidateName(repo); err != nil {
			return addTarget{}, fmt.Errorf("could not infer package name from GitHub URL %q: %w", raw, err)
		}
		if strings.HasSuffix(strings.ToLower(u.Path), ".tar.gz") || strings.HasSuffix(strings.ToLower(u.Path), ".tgz") {
			return addTarget{Name: repo, SourceURL: raw, SourceVersion: inferStableVersion(raw)}, nil
		}
		// For repository URLs, resolve to a stable semver tag via GitHub API.
		return addTarget{Name: repo, GitHubRepoRef: owner + "/" + repo}, nil
	}

	return addTarget{}, fmt.Errorf("unsupported dependency URL host %q; use a SWI or GitHub URL", host)
}

func inferStableVersion(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		if qp := strings.TrimSpace(u.Query().Get("path")); qp != "" {
			if v := versionFromBase(stripArchiveExt(path.Base(qp))); v != "" {
				return v
			}
		}
		if v := versionFromBase(stripArchiveExt(path.Base(u.Path))); v != "" {
			return v
		}
	}
	base := stripArchiveExt(path.Base(rawURL))
	return versionFromBase(base)
}

func stripArchiveExt(s string) string {
	name := strings.TrimSpace(s)
	for _, suffix := range []string{".tar.gz", ".tgz", ".zip"} {
		name = strings.TrimSuffix(name, suffix)
	}
	for _, suffix := range []string{".tar", ".gz"} {
		name = strings.TrimSuffix(name, suffix)
	}
	name = strings.TrimSuffix(name, ".git")
	return name
}

func versionFromBase(base string) string {
	if i := strings.LastIndex(base, "-"); i > 0 && i+1 < len(base) {
		candidate := strings.TrimPrefix(base[i+1:], "v")
		if manifest.ValidateVersion(candidate) == nil {
			return candidate
		}
	}
	base = strings.TrimPrefix(base, "v")
	if manifest.ValidateVersion(base) == nil {
		return base
	}
	return ""
}

func installedVersionFor(manifestPath, name string) (string, error) {
	lockPath := filepath.Join(filepath.Dir(manifestPath), lockfile.LockFileName)
	lf, err := lockfile.Load(lockPath)
	if err != nil {
		return "", fmt.Errorf("reading Prolfile.lock: %w", err)
	}
	if lf == nil {
		return "", nil
	}
	for _, p := range lf.Packages {
		if p.Name == name {
			return p.Version, nil
		}
	}
	return "", nil
}

func deriveNameFromPathOrQuery(u *url.URL) string {
	if p := strings.TrimSpace(u.Query().Get("path")); p != "" {
		return trimArchiveSuffixes(p)
	}
	base := path.Base(strings.Trim(u.Path, "/"))
	return trimArchiveSuffixes(base)
}

func trimArchiveSuffixes(s string) string {
	name := strings.TrimSpace(s)
	for _, suffix := range []string{".tar.gz", ".tgz", ".zip"} {
		name = strings.TrimSuffix(name, suffix)
	}
	for _, suffix := range []string{".tar", ".gz"} {
		name = strings.TrimSuffix(name, suffix)
	}
	name = strings.TrimSuffix(name, ".git")
	// Heuristic for pack filenames like "clpfd-1.2.3".
	if i := strings.LastIndex(name, "-"); i > 0 && i+1 < len(name) {
		tail := name[i+1:]
		if hasDigit(tail) {
			return name[:i]
		}
	}
	return name
}

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func splitPathParts(p string) []string {
	rawParts := strings.Split(strings.Trim(p, "/"), "/")
	out := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
