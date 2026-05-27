package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVersion_StripsLeadingV(t *testing.T) {
	got, err := normalizeVersion("v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", got.String())
}

func TestConstraintAllows_DocumentedSyntax(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		{"star", "*", "9.9.9", true},
		{"caret", "^1.4", "1.9.0", true},
		{"caret rejects next major", "^1.4", "2.0.0", false},
		{"tilde", "~1.4.2", "1.4.9", true},
		{"tilde rejects next minor", "~1.4.2", "1.5.0", false},
		{"gte", ">=1.4", "1.4.0", true},
		{"exact", "=1.4.3", "1.4.3", true},
		{"bare version is compatible", "1.4.3", "1.9.0", true},
		{"bare version rejects next major", "1.4.3", "2.0.0", false},
		{"zero major caret", "^0.4.2", "0.4.9", true},
		{"zero major caret rejects next minor", "^0.4.2", "0.5.0", false},
		{"plain constraint excludes prerelease", "^1.4", "1.4.0-beta.1", false},
		{"explicit prerelease permits prerelease", ">=1.4.0-beta.1", "1.4.0-beta.2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parseConstraint("pkg", tt.constraint)
			require.NoError(t, err)
			v, err := normalizeVersion(tt.version)
			require.NoError(t, err)
			assert.Equal(t, tt.want, c.Check(v))
		})
	}
}

func TestParseConstraint_InvalidIncludesPackageContext(t *testing.T) {
	_, err := parseConstraint("badpkg", "not a version")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "badpkg")
	assert.Contains(t, err.Error(), "constraint")
}

func FuzzConstraint(f *testing.F) {
	for _, seed := range []string{"*", "^1.4", "~1.4.2", ">=1.0.0", "=1.2.3", "v1.2.3", "bad"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = parseConstraint("fuzzpkg", input)
	})
}
