package lockfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/prolm/prolm/pkg/prolfile"
)

type depsHashInput struct {
	Dependencies    map[string]string `toml:"dependencies,omitempty"`
	DevDependencies map[string]string `toml:"dev-dependencies,omitempty"`
}

// ComputeProlfileHash returns the canonical SHA-256 content hash for the
// manifest's [dependencies] and [dev-dependencies] sections.
//
// The hash input is a deterministic TOML encoding of only those sections.
// For empty dependency sets, this yields SHA-256 of empty content.
func ComputeProlfileHash(pf *prolfile.ProlFile) (string, error) {
	if pf == nil {
		sum := sha256.Sum256(nil)
		return "sha256:" + hex.EncodeToString(sum[:]), nil
	}

	in := depsHashInput{
		Dependencies:    pf.Dependencies,
		DevDependencies: pf.DevDependencies,
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(in); err != nil {
		return "", fmt.Errorf("encoding dependency hash input: %w", err)
	}

	sum := sha256.Sum256(buf.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
