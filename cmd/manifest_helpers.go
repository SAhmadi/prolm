package cmd

import (
	"fmt"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/spf13/cobra"
)

// loadManifest resolves the Prolfile.toml path from the --config flag or by
// auto-discovery, then loads and validates it. It returns the parsed ProlFile
// and the absolute path to the manifest so callers can derive sibling paths
// (e.g. Prolfile.lock, store, cache).
//
// All cmd/ commands that need the manifest should call this helper instead of
// duplicating the discover → load pattern (DRY-003).
func loadManifest(cmd *cobra.Command) (*prolfile.ProlFile, string, error) {
	configPath, _ := cmd.Flags().GetString("config")

	manifestPath := configPath
	if manifestPath == "" {
		discovered, err := manifest.Discover("")
		if err != nil {
			return nil, "", fmt.Errorf("discovering Prolfile.toml: %w", err)
		}
		manifestPath = discovered
	}

	pf, err := manifest.Load(manifestPath, manifest.LoadOptions{ProlmVersion: Version})
	if err != nil {
		return nil, "", fmt.Errorf("reading Prolfile.toml: %w", err)
	}

	return pf, manifestPath, nil
}
