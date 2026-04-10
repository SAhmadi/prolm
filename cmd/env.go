package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/spf13/cobra"
)

type envCmdRunner struct {
	newRuntime       func(name string) (runtime.Runtime, error)
	discoverManifest func(startDir string) (string, error)
	loadManifest     func(path string, opts manifest.LoadOptions) (*prolfile.ProlFile, error)
	storePath        func() (string, error)
	countStorePacks  func(storePath string) (int, error)
	getenv           func(key string) string
}

var defaultEnvCmdRunner = &envCmdRunner{
	newRuntime:       runtime.NewRuntime,
	discoverManifest: manifest.Discover,
	loadManifest:     manifest.Load,
	storePath:        resolveDefaultStorePath,
	countStorePacks:  countInstalledPacks,
	getenv:           os.Getenv,
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print diagnostic environment information",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return defaultEnvCmdRunner.run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}

func (r *envCmdRunner) run(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	runtimeLine, err := r.renderRuntimeLine()
	if err != nil {
		return err
	}

	storePath, err := r.storePath()
	if err != nil {
		return fmt.Errorf("resolving store path: %w", err)
	}
	packCount, err := r.countStorePacks(storePath)
	if err != nil {
		return fmt.Errorf("counting installed packs in %s: %w", storePath, err)
	}

	projectLine, err := r.renderProjectLine()
	if err != nil {
		return err
	}

	httpProxy := redactProxyValue(r.getenv("HTTP_PROXY"))
	httpsProxy := redactProxyValue(r.getenv("HTTPS_PROXY"))

	fmt.Fprintf(out, "prolm      %s\n", Version)
	fmt.Fprintf(out, "runtime    %s\n", runtimeLine)
	fmt.Fprintf(out, "store      %s (%d packs installed)\n", storePath, packCount)
	fmt.Fprintf(out, "project    %s\n", projectLine)
	fmt.Fprintf(out, "proxy      HTTP_PROXY=%s, HTTPS_PROXY=%s\n", httpProxy, httpsProxy)

	return nil
}

func (r *envCmdRunner) renderRuntimeLine() (string, error) {
	rt, err := r.newRuntime("swi")
	if err != nil {
		return "", fmt.Errorf("selecting runtime %q: %w", "swi", err)
	}

	info, err := rt.Detect()
	if err != nil {
		var notFound *runtime.ErrRuntimeNotFound
		if errors.As(err, &notFound) {
			return "swi -> not found [✗]", nil
		}
		return "", fmt.Errorf("detecting %s runtime: %w", rt.Name(), err)
	}

	return fmt.Sprintf("swi -> %s (%s) [✓]", info.Path, info.Version), nil
}

func (r *envCmdRunner) renderProjectLine() (string, error) {
	manifestPath, err := r.discoverManifest("")
	if err != nil {
		if errors.Is(err, manifest.ErrNotFound) {
			return "not in a project", nil
		}
		return "", fmt.Errorf("discovering Prolfile.toml: %w", err)
	}

	pf, err := r.loadManifest(manifestPath, manifest.LoadOptions{ProlmVersion: Version})
	if err != nil {
		return "", fmt.Errorf("loading project manifest: %w", err)
	}

	return fmt.Sprintf("%s %s", pf.Package.Name, pf.Package.Version), nil
}

func resolveDefaultStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".prolm", "store"), nil
}

func countInstalledPacks(storePath string) (int, error) {
	nameEntries, err := os.ReadDir(storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, nameEntry := range nameEntries {
		if !nameEntry.IsDir() {
			continue
		}
		versionPath := filepath.Join(storePath, nameEntry.Name())
		versionEntries, err := os.ReadDir(versionPath)
		if err != nil {
			return 0, err
		}
		for _, versionEntry := range versionEntries {
			if versionEntry.IsDir() {
				count++
			}
		}
	}

	return count, nil
}

func redactProxyValue(raw string) string {
	if raw == "" {
		return "unset"
	}

	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return raw
	}

	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword("REDACTED", "REDACTED")
			return u.String()
		}
		u.User = url.User("REDACTED")
	}

	return u.String()
}
