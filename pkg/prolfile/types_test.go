package prolfile

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProlFile_ZeroValue(t *testing.T) {
	var pf ProlFile

	assert.Equal(t, 0, pf.Meta.ProlfileVersion)
	assert.Empty(t, pf.Meta.MinProlmVersion)
	assert.Empty(t, pf.Package.Name)
	assert.Empty(t, pf.Package.Version)
	assert.Empty(t, pf.Package.Authors)
	assert.Nil(t, pf.Dependencies)
	assert.Nil(t, pf.DevDependencies)
	assert.Nil(t, pf.Runtime)
	assert.Nil(t, pf.Scripts)
}

func TestProlFile_DecodeTOML(t *testing.T) {
	input := `
[meta]
prolfile_version = 1
min_prolm_version = "0.1.0"

[package]
name        = "my-expert-system"
version     = "0.1.0"
description = "A diagnostic expert system for network faults"
authors     = ["Ada Lovelace <ada@example.com>"]
license     = "MIT"
homepage    = "https://github.com/ada/my-expert-system"
entry       = "src/main.pl"
runtime     = "swi"

[dependencies]
clpfd       = "^1.4"
prosqlite   = "^0.9"
http        = "^7.0"

[dev-dependencies]
plunit      = "*"

[runtime.swi]
min_version = "9.0"
flags       = ["-O", "--stack-limit=2g"]

[runtime.scryer]
min_version = "0.9"

[scripts]
start       = "prolm run src/main.pl"
diagnose    = "prolm run src/diagnose.pl -- --mode interactive"
`

	var pf ProlFile
	_, err := toml.Decode(input, &pf)
	require.NoError(t, err)

	// Meta
	assert.Equal(t, 1, pf.Meta.ProlfileVersion)
	assert.Equal(t, "0.1.0", pf.Meta.MinProlmVersion)

	// Package
	assert.Equal(t, "my-expert-system", pf.Package.Name)
	assert.Equal(t, "0.1.0", pf.Package.Version)
	assert.Equal(t, "A diagnostic expert system for network faults", pf.Package.Description)
	assert.Equal(t, []string{"Ada Lovelace <ada@example.com>"}, pf.Package.Authors)
	assert.Equal(t, "MIT", pf.Package.License)
	assert.Equal(t, "https://github.com/ada/my-expert-system", pf.Package.Homepage)
	assert.Equal(t, "src/main.pl", pf.Package.Entry)
	assert.Equal(t, "swi", pf.Package.Runtime)

	// Dependencies
	assert.Equal(t, "^1.4", pf.Dependencies["clpfd"])
	assert.Equal(t, "^0.9", pf.Dependencies["prosqlite"])
	assert.Equal(t, "^7.0", pf.Dependencies["http"])

	// Dev-dependencies
	assert.Equal(t, "*", pf.DevDependencies["plunit"])

	// Runtime configs
	require.Contains(t, pf.Runtime, "swi")
	assert.Equal(t, "9.0", pf.Runtime["swi"].MinVersion)
	assert.Equal(t, []string{"-O", "--stack-limit=2g"}, pf.Runtime["swi"].Flags)
	require.Contains(t, pf.Runtime, "scryer")
	assert.Equal(t, "0.9", pf.Runtime["scryer"].MinVersion)

	// Scripts
	assert.Equal(t, "prolm run src/main.pl", pf.Scripts["start"])
	assert.Equal(t, "prolm run src/diagnose.pl -- --mode interactive", pf.Scripts["diagnose"])
}

func TestProlFile_RoundTrip(t *testing.T) {
	original := ProlFile{
		Meta: Meta{
			ProlfileVersion: 1,
			MinProlmVersion: "0.2.0",
		},
		Package: Package{
			Name:        "test-project",
			Version:     "1.0.0",
			Description: "A test project",
			Authors:     []string{"Test Author <test@example.com>"},
			License:     "Apache-2.0",
			Homepage:    "https://example.com",
			Entry:       "src/main.pl",
			Runtime:     "swi",
		},
		Dependencies: map[string]string{
			"clpfd": "^1.4",
			"http":  "^7.0",
		},
		DevDependencies: map[string]string{
			"plunit": "*",
		},
		Runtime: map[string]RuntimeConfig{
			"swi": {MinVersion: "9.0", Flags: []string{"-O"}},
		},
		Scripts: map[string]string{
			"start": "prolm run",
		},
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(original)
	require.NoError(t, err)

	var decoded ProlFile
	_, err = toml.Decode(buf.String(), &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestLockFile_ZeroValue(t *testing.T) {
	var lf LockFile

	assert.Equal(t, 0, lf.Meta.LockVersion)
	assert.Empty(t, lf.Meta.ProlfileHash)
	assert.Nil(t, lf.Packages)
}

func TestLockFile_DecodeTOML(t *testing.T) {
	input := `
[meta]
lock_version  = 1
prolfile_hash = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

[[package]]
name         = "clpfd"
version      = "1.4.3"
source       = "swi-pack-index"
url          = "https://example.com/clpfd-1.4.3.tar.gz"
checksum     = "sha256:a3f2c1d9e4b5a6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1"
dependencies = ["http@7.1.2"]

[[package]]
name         = "http"
version      = "7.1.2"
source       = "swi-pack-index"
url          = "https://example.com/http-7.1.2.tar.gz"
checksum     = "sha256:f1d2e3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2"
dependencies = []

[[package]]
name         = "prosqlite"
version      = "0.9.11"
source       = "swi-pack-index"
url          = "https://example.com/prosqlite-0.9.11.tar.gz"
checksum     = "sha256:b8e91d4fa1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8"
dependencies = []
`

	var lf LockFile
	_, err := toml.Decode(input, &lf)
	require.NoError(t, err)

	assert.Equal(t, 1, lf.Meta.LockVersion)
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", lf.Meta.ProlfileHash)
	require.Len(t, lf.Packages, 3)

	// First package
	assert.Equal(t, "clpfd", lf.Packages[0].Name)
	assert.Equal(t, "1.4.3", lf.Packages[0].Version)
	assert.Equal(t, "swi-pack-index", lf.Packages[0].Source)
	assert.Equal(t, []string{"http@7.1.2"}, lf.Packages[0].Dependencies)

	// Second package
	assert.Equal(t, "http", lf.Packages[1].Name)
	assert.Equal(t, "7.1.2", lf.Packages[1].Version)
	assert.Empty(t, lf.Packages[1].Dependencies)

	// Third package
	assert.Equal(t, "prosqlite", lf.Packages[2].Name)
}

func TestLockFile_RoundTrip(t *testing.T) {
	original := LockFile{
		Meta: LockMeta{
			LockVersion:  1,
			ProlfileHash: "sha256:abcdef1234567890",
		},
		Packages: []LockEntry{
			{
				Name:         "alpha",
				Version:      "1.0.0",
				Source:       "swi-pack-index",
				URL:          "https://example.com/alpha-1.0.0.tar.gz",
				Checksum:     "sha256:aaa111",
				Dependencies: []string{"beta@2.0.0"},
			},
			{
				Name:         "beta",
				Version:      "2.0.0",
				Source:       "swi-pack-index",
				URL:          "https://example.com/beta-2.0.0.tar.gz",
				Checksum:     "sha256:bbb222",
				Dependencies: []string{},
			},
		},
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(original)
	require.NoError(t, err)

	var decoded LockFile
	_, err = toml.Decode(buf.String(), &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestLockEntry_SignatureOmitEmpty(t *testing.T) {
	// Empty signature should be omitted from TOML output.
	entry := LockEntry{
		Name:     "test-pack",
		Version:  "1.0.0",
		Source:   "swi-pack-index",
		URL:      "https://example.com/test.tar.gz",
		Checksum: "sha256:abc123",
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(entry)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "signature")

	// Non-empty signature should appear.
	entry.Signature = "sigstore:xyz789"
	buf.Reset()
	err = toml.NewEncoder(&buf).Encode(entry)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "signature")
	assert.Contains(t, buf.String(), "sigstore:xyz789")
}

func TestProlFile_NamespacedDependencies(t *testing.T) {
	input := `
[meta]
prolfile_version = 1

[package]
name = "test"
version = "0.1.0"

[dependencies]
"ada/clpfd" = "^1.4"
"community/json-parser" = "^2.0"
`

	var pf ProlFile
	_, err := toml.Decode(input, &pf)
	require.NoError(t, err)

	assert.Equal(t, "^1.4", pf.Dependencies["ada/clpfd"])
	assert.Equal(t, "^2.0", pf.Dependencies["community/json-parser"])

	// Round-trip preserves namespaced keys.
	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(pf)
	require.NoError(t, err)

	var decoded ProlFile
	_, err = toml.Decode(buf.String(), &decoded)
	require.NoError(t, err)

	assert.Equal(t, pf.Dependencies, decoded.Dependencies)
}

func TestProlFile_MultipleRuntimes(t *testing.T) {
	input := `
[meta]
prolfile_version = 1

[package]
name = "multi-runtime"
version = "0.1.0"

[runtime.swi]
min_version = "9.0"
flags = ["-O"]

[runtime.scryer]
min_version = "0.9"

[runtime.gnu]
min_version = "1.5"
`

	var pf ProlFile
	_, err := toml.Decode(input, &pf)
	require.NoError(t, err)

	require.Len(t, pf.Runtime, 3)
	assert.Equal(t, "9.0", pf.Runtime["swi"].MinVersion)
	assert.Equal(t, []string{"-O"}, pf.Runtime["swi"].Flags)
	assert.Equal(t, "0.9", pf.Runtime["scryer"].MinVersion)
	assert.Equal(t, "1.5", pf.Runtime["gnu"].MinVersion)
}

func TestDependency_Construction(t *testing.T) {
	tests := []struct {
		name string
		dep  Dependency
	}{
		{
			name: "version only",
			dep:  Dependency{Name: "clpfd", Version: "^1.4"},
		},
		{
			name: "path only",
			dep:  Dependency{Name: "my-lib", Path: "../my-lib"},
		},
		{
			name: "version with source",
			dep:  Dependency{Name: "http", Version: "^7.0", Source: "swi-pack-index"},
		},
		{
			name: "namespaced with registry",
			dep:  Dependency{Name: "ada/clpfd", Version: "^1.4", Registry: "prolm-registry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify round-trip through TOML.
			var buf bytes.Buffer
			err := toml.NewEncoder(&buf).Encode(tt.dep)
			require.NoError(t, err)

			var decoded Dependency
			_, err = toml.Decode(buf.String(), &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.dep, decoded)
		})
	}
}
