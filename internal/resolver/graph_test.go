package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDependencySpec(t *testing.T) {
	tests := []struct {
		spec           string
		wantName       string
		wantConstraint string
	}{
		{"beta@^1.2", "beta", "^1.2"},
		{"beta", "beta", "*"},
		{"owner/beta@~0.4.2", "owner/beta", "~0.4.2"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			got, err := parseDependencySpec(tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantConstraint, got.Constraint)
		})
	}
}

func TestParseDependencySpec_RejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		"BadName@^1.0",
		"../evil@^1.0",
		"pkg@not a version",
		"pkg\x00@^1.0",
	}

	for _, spec := range tests {
		t.Run(spec, func(t *testing.T) {
			_, err := parseDependencySpec(spec)
			require.Error(t, err)
		})
	}
}
