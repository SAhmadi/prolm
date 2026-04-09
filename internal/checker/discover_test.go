package checker

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantLen int
	}{
		{
			name:    "empty directory",
			setup:   func(t *testing.T, dir string) {},
			wantLen: 0,
		},
		{
			name: "single main file",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.pl"), []byte("test."), 0644))
			},
			wantLen: 1,
		},
		{
			name: "mixed source and test files",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "test_helper.pl"), []byte("test."), 0644))
			},
			wantLen: 1,
		},
		{
			name: "nested structure",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "helper.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "tests", "main_test.pl"), []byte("test."), 0644))
			},
			wantLen: 2,
		},
		{
			name: "vendor and hidden dirs skipped",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "vendor", "dep.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "config.pl"), []byte("test."), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.pl"), []byte("test."), 0644))
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			got, err := Discover(dir)
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			// Verify sorted
			if len(got) > 0 {
				sorted := make([]string, len(got))
				copy(sorted, got)
				sort.Strings(sorted)
				assert.Equal(t, sorted, got)
				// Verify absolute paths
				for _, p := range got {
					assert.True(t, filepath.IsAbs(p))
				}
			}
		})
	}
}
