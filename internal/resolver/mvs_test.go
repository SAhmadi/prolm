package resolver

import (
	"testing"

	"github.com/prolm/prolm/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectVersion_ChoosesMinimumSatisfyingVersion(t *testing.T) {
	reqs := []requirement{
		{Dependent: "root", Constraint: "^1.0"},
		{Dependent: "alpha", Constraint: ">=1.4.0"},
	}
	versions := []registry.PackageVersion{
		{Name: "shared", Version: "2.0.0"},
		{Name: "shared", Version: "1.5.0"},
		{Name: "shared", Version: "1.4.0"},
		{Name: "shared", Version: "1.0.0"},
	}

	got, err := selectVersion("shared", reqs, versions)
	require.NoError(t, err)
	assert.Equal(t, "1.4.0", got.Version)
}

func TestSelectVersion_SkipsYankedVersions(t *testing.T) {
	reqs := []requirement{{Dependent: "root", Constraint: "*"}}
	versions := []registry.PackageVersion{
		{Name: "pkg", Version: "1.0.0", Yanked: true},
		{Name: "pkg", Version: "1.1.0"},
	}

	got, err := selectVersion("pkg", reqs, versions)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", got.Version)
}

func TestSelectVersion_ConflictNamesDependents(t *testing.T) {
	reqs := []requirement{
		{Dependent: "alpha", Constraint: "^1.0"},
		{Dependent: "bravo", Constraint: "^2.0"},
	}
	versions := []registry.PackageVersion{
		{Name: "shared", Version: "2.0.0"},
		{Name: "shared", Version: "1.0.0"},
	}

	_, err := selectVersion("shared", reqs, versions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shared")
	assert.Contains(t, err.Error(), "alpha")
	assert.Contains(t, err.Error(), "bravo")
}

func TestSelectVersion_AllVersionsYanked(t *testing.T) {
	reqs := []requirement{{Dependent: "root", Constraint: "*"}}
	versions := []registry.PackageVersion{
		{Name: "pkg", Version: "1.0.0", Yanked: true},
	}

	_, err := selectVersion("pkg", reqs, versions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all versions")
	assert.Contains(t, err.Error(), "yanked")
}
