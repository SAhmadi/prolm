package resolver

import (
	"context"
	"testing"

	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRegistry struct {
	versions    map[string][]registry.PackageVersion
	downloadURL map[string]string
}

func (m *mockRegistry) Search(_ context.Context, _ string) ([]registry.PackageVersion, error) {
	return nil, nil
}

func (m *mockRegistry) Versions(_ context.Context, name string) ([]registry.PackageVersion, error) {
	if vs, ok := m.versions[name]; ok {
		return vs, nil
	}
	return nil, &registry.ErrPackageNotFound{Name: name, Registry: "mock"}
}

func (m *mockRegistry) DownloadURL(_ context.Context, name, version string) (string, error) {
	if u, ok := m.downloadURL[name+"@"+version]; ok {
		return u, nil
	}
	return "", &registry.ErrVersionNotFound{Name: name, Version: version}
}

func TestResolve_DiamondDependency(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"alpha": {
				{Name: "alpha", Version: "1.0.0", Dependencies: []string{"shared@>=1.2.0"}},
			},
			"bravo": {
				{Name: "bravo", Version: "1.0.0", Dependencies: []string{"shared@>=1.4.0"}},
			},
			"shared": {
				{Name: "shared", Version: "2.0.0"},
				{Name: "shared", Version: "1.5.0"},
				{Name: "shared", Version: "1.4.0"},
				{Name: "shared", Version: "1.2.0"},
			},
		},
		downloadURL: map[string]string{
			"alpha@1.0.0":  "https://www.swi-prolog.org/pack/alpha-1.0.0.tar.gz",
			"bravo@1.0.0":  "https://www.swi-prolog.org/pack/bravo-1.0.0.tar.gz",
			"shared@1.4.0": "https://www.swi-prolog.org/pack/shared-1.4.0.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"bravo": "*", "alpha": "*"},
	}

	lf, err := Resolve(context.Background(), manifest, reg)
	require.NoError(t, err)
	require.Len(t, lf.Packages, 3)
	assert.Equal(t, []string{"alpha", "bravo", "shared"}, []string{
		lf.Packages[0].Name,
		lf.Packages[1].Name,
		lf.Packages[2].Name,
	})
	assert.Equal(t, "1.4.0", lf.Packages[2].Version)
	assert.Equal(t, []string{"shared@1.4.0"}, lf.Packages[0].Dependencies)
	assert.Equal(t, []string{"shared@1.4.0"}, lf.Packages[1].Dependencies)
}

func TestResolve_PrunesDependenciesFromReplacedVersion(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"alpha": {
				{Name: "alpha", Version: "1.0.0", Dependencies: []string{"shared@>=1.0.0"}},
			},
			"bravo": {
				{Name: "bravo", Version: "1.0.0", Dependencies: []string{"shared@>=2.0.0"}},
			},
			"shared": {
				{Name: "shared", Version: "2.0.0"},
				{Name: "shared", Version: "1.0.0", Dependencies: []string{"legacy@*"}},
			},
			"legacy": {
				{Name: "legacy", Version: "1.0.0"},
			},
		},
		downloadURL: map[string]string{
			"alpha@1.0.0":  "https://www.swi-prolog.org/pack/alpha-1.0.0.tar.gz",
			"bravo@1.0.0":  "https://www.swi-prolog.org/pack/bravo-1.0.0.tar.gz",
			"shared@2.0.0": "https://www.swi-prolog.org/pack/shared-2.0.0.tar.gz",
			"legacy@1.0.0": "https://www.swi-prolog.org/pack/legacy-1.0.0.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"bravo": "*", "alpha": "*"},
	}

	lf, err := Resolve(context.Background(), manifest, reg)
	require.NoError(t, err)

	require.Len(t, lf.Packages, 3)
	assert.Equal(t, []string{"alpha", "bravo", "shared"}, []string{
		lf.Packages[0].Name,
		lf.Packages[1].Name,
		lf.Packages[2].Name,
	})
	assert.Equal(t, "2.0.0", lf.Packages[2].Version)
	assert.Equal(t, []string{"shared@2.0.0"}, lf.Packages[0].Dependencies)
	assert.Equal(t, []string{"shared@2.0.0"}, lf.Packages[1].Dependencies)
}

func TestResolve_DropsRequirementsFromReplacedVersion(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"alpha": {
				{Name: "alpha", Version: "1.0.0", Dependencies: []string{"shared@>=1.0.0"}},
			},
			"beta": {
				{Name: "beta", Version: "1.0.0", Dependencies: []string{"legacy@>=1.0.0"}},
			},
			"zulu": {
				{Name: "zulu", Version: "1.0.0", Dependencies: []string{"shared@>=2.0.0"}},
			},
			"shared": {
				{Name: "shared", Version: "2.0.0"},
				{Name: "shared", Version: "1.0.0", Dependencies: []string{"legacy@>=2.0.0"}},
			},
			"legacy": {
				{Name: "legacy", Version: "2.0.0"},
				{Name: "legacy", Version: "1.0.0"},
			},
		},
		downloadURL: map[string]string{
			"alpha@1.0.0":  "https://www.swi-prolog.org/pack/alpha-1.0.0.tar.gz",
			"beta@1.0.0":   "https://www.swi-prolog.org/pack/beta-1.0.0.tar.gz",
			"zulu@1.0.0":   "https://www.swi-prolog.org/pack/zulu-1.0.0.tar.gz",
			"shared@2.0.0": "https://www.swi-prolog.org/pack/shared-2.0.0.tar.gz",
			"legacy@2.0.0": "https://www.swi-prolog.org/pack/legacy-2.0.0.tar.gz",
			"legacy@1.0.0": "https://www.swi-prolog.org/pack/legacy-1.0.0.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{
		Dependencies: map[string]string{"zulu": "*", "beta": "*", "alpha": "*"},
	}

	lf, err := Resolve(context.Background(), manifest, reg)
	require.NoError(t, err)

	require.Len(t, lf.Packages, 5)
	assert.Equal(t, "legacy", lf.Packages[2].Name)
	assert.Equal(t, "1.0.0", lf.Packages[2].Version)
	assert.Equal(t, []string{"legacy@1.0.0"}, lf.Packages[1].Dependencies)
	assert.Equal(t, []string{"shared@2.0.0"}, lf.Packages[0].Dependencies)
	assert.Equal(t, []string{"shared@2.0.0"}, lf.Packages[4].Dependencies)
}

func TestResolve_CircularDependencyReportsPath(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"alpha": {{Name: "alpha", Version: "1.0.0", Dependencies: []string{"bravo@*"}}},
			"bravo": {{Name: "bravo", Version: "1.0.0", Dependencies: []string{"alpha@*"}}},
		},
		downloadURL: map[string]string{
			"alpha@1.0.0": "https://www.swi-prolog.org/pack/alpha-1.0.0.tar.gz",
			"bravo@1.0.0": "https://www.swi-prolog.org/pack/bravo-1.0.0.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"alpha": "*"}}

	_, err := Resolve(context.Background(), manifest, reg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
	assert.Contains(t, err.Error(), "alpha -> bravo -> alpha")
}

func TestResolve_MissingPackage(t *testing.T) {
	reg := &mockRegistry{versions: map[string][]registry.PackageVersion{}}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"missing": "*"}}

	_, err := Resolve(context.Background(), manifest, reg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestResolve_NoSatisfyingVersion(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"pkg": {{Name: "pkg", Version: "1.0.0"}},
		},
		downloadURL: map[string]string{
			"pkg@1.0.0": "https://www.swi-prolog.org/pack/pkg-1.0.0.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"pkg": "^2.0"}}

	_, err := Resolve(context.Background(), manifest, reg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pkg")
	assert.Contains(t, err.Error(), "^2.0")
}

func TestResolve_NormalizesVPrefixInLock(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"pkg": {{Name: "pkg", Version: "v1.2.3"}},
		},
		downloadURL: map[string]string{
			"pkg@1.2.3": "https://www.swi-prolog.org/pack/pkg-1.2.3.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"pkg": "*"}}

	lf, err := Resolve(context.Background(), manifest, reg)
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)
	assert.Equal(t, "1.2.3", lf.Packages[0].Version)
}

func TestResolve_UsesRegistryRowURLForShortVersion(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"pkg": {{
				Name:    "pkg",
				Version: "1.2",
				URL:     "https://www.swi-prolog.org/pack/pkg-1.2.tar.gz",
			}},
		},
		downloadURL: map[string]string{
			"pkg@1.2": "https://www.swi-prolog.org/pack/pkg-1.2.tar.gz",
		},
	}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"pkg": "*"}}

	lf, err := Resolve(context.Background(), manifest, reg)
	require.NoError(t, err)
	require.Len(t, lf.Packages, 1)
	assert.Equal(t, "1.2.0", lf.Packages[0].Version)
	assert.Equal(t, "https://www.swi-prolog.org/pack/pkg-1.2.tar.gz", lf.Packages[0].URL)
}

func TestResolve_RejectsRegistryPackageNameMismatch(t *testing.T) {
	reg := &mockRegistry{
		versions: map[string][]registry.PackageVersion{
			"pkg": {{Name: "other", Version: "1.0.0"}},
		},
	}
	manifest := &prolfile.ProlFile{Dependencies: map[string]string{"pkg": "*"}}

	_, err := Resolve(context.Background(), manifest, reg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "other")
	assert.Contains(t, err.Error(), "pkg")
}
