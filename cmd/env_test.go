package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/runtime"
	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withEnvCmdRunner(t *testing.T, r *envCmdRunner) {
	t.Helper()
	orig := defaultEnvCmdRunner
	defaultEnvCmdRunner = r
	t.Cleanup(func() { defaultEnvCmdRunner = orig })
}

func newTestEnvRunner(fr runtime.Runtime, discoverErr error, discoverPath string, pf *prolfile.ProlFile, loadErr error, storePath string, storeCount int, storeErr error, env map[string]string) *envCmdRunner {
	return &envCmdRunner{
		newRuntime: func(string) (runtime.Runtime, error) { return fr, nil },
		discoverManifest: func(string) (string, error) {
			if discoverErr != nil {
				return "", discoverErr
			}
			return discoverPath, nil
		},
		loadManifest: func(string, manifest.LoadOptions) (*prolfile.ProlFile, error) {
			if loadErr != nil {
				return nil, loadErr
			}
			return pf, nil
		},
		storePath: func() (string, error) {
			return storePath, nil
		},
		countStorePacks: func(string) (int, error) {
			if storeErr != nil {
				return 0, storeErr
			}
			return storeCount, nil
		},
		getenv: func(key string) string {
			return env[key]
		},
	}
}

func TestExecute_Env_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	pf := &prolfile.ProlFile{}
	pf.Package.Name = "my-expert-system"
	pf.Package.Version = "0.1.0"

	runner := newTestEnvRunner(
		fr,
		nil,
		filepath.Join(dir, "Prolfile.toml"),
		pf,
		nil,
		"/Users/test/.prolm/store",
		12,
		nil,
		map[string]string{
			"HTTP_PROXY":  "http://user:pass@proxy.corp:8080",
			"HTTPS_PROXY": "https://proxy.corp:8443",
		},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "prolm      "+Version)
	assert.Contains(t, out, "runtime    swi -> /usr/local/bin/swipl (9.2.1) [✓]")
	assert.Contains(t, out, "store      /Users/test/.prolm/store (12 packs installed)")
	assert.Contains(t, out, "project    my-expert-system 0.1.0")
	assert.Contains(t, out, "proxy      HTTP_PROXY=http://REDACTED:REDACTED@proxy.corp:8080, HTTPS_PROXY=https://proxy.corp:8443")
}

func TestExecute_Env_RuntimeMissing_IsGraceful(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", detectErr: &runtime.ErrRuntimeNotFound{Runtime: "swi"}}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "runtime    swi -> not found [✗]")
}

func TestExecute_Env_NoProject_IsGraceful(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "project    not in a project")
}

func TestExecute_Env_ProxyUnset(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "proxy      HTTP_PROXY=unset, HTTPS_PROXY=unset")
}

func TestExecute_Env_ProxyCredentialsAreRedactedForSchemelessValues(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{
			"HTTP_PROXY":  "user:supersecret@proxy.corp:8080",
			"HTTPS_PROXY": "alice@proxy.corp:8443",
		},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.NotContains(t, out, "supersecret")
	assert.NotContains(t, out, "alice@proxy.corp")
	assert.Contains(t, out, "HTTP_PROXY=REDACTED:REDACTED@proxy.corp:8080")
	assert.Contains(t, out, "HTTPS_PROXY=REDACTED@proxy.corp:8443")
}

func TestExecute_Env_StoreCountRendering(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		3,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	buf := resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "store      /Users/test/.prolm/store (3 packs installed)")
}

func TestExecute_Env_RuntimeDetectUnexpectedError_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", detectErr: errors.New("permission denied")}
	runner := newTestEnvRunner(
		fr,
		manifest.ErrNotFound,
		"",
		nil,
		nil,
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detecting swi runtime")
}

func TestExecute_Env_ProjectLoadUnexpectedError_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fr := &fakeRuntime{name: "swi", info: &runtime.RuntimeInfo{Path: "/usr/local/bin/swipl", Version: "9.2.1"}}
	runner := newTestEnvRunner(
		fr,
		nil,
		filepath.Join(dir, "Prolfile.toml"),
		nil,
		errors.New("invalid toml"),
		"/Users/test/.prolm/store",
		0,
		nil,
		map[string]string{},
	)
	withEnvCmdRunner(t, runner)

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"env"})
	err := Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading project manifest")
}

func TestRedactProxyValue(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "valid URL with username and password",
			raw:      "http://alice:secret@proxy.corp:8080",
			expected: "http://REDACTED:REDACTED@proxy.corp:8080",
		},
		{
			name:     "valid URL with username only",
			raw:      "http://alice@proxy.corp:8080",
			expected: "http://REDACTED@proxy.corp:8080",
		},
		{
			name:     "schemeless with username and password",
			raw:      "alice:secret@proxy.corp:8080",
			expected: "REDACTED:REDACTED@proxy.corp:8080",
		},
		{
			name:     "schemeless with username only",
			raw:      "alice@proxy.corp:8080",
			expected: "REDACTED@proxy.corp:8080",
		},
		{
			name:     "plain proxy without credentials",
			raw:      "https://proxy.corp:8443",
			expected: "https://proxy.corp:8443",
		},
		{
			name:     "empty value",
			raw:      "",
			expected: "unset",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := redactProxyValue(tc.raw)
			assert.Equal(t, tc.expected, got)
			assert.False(t, strings.Contains(got, "secret"))
		})
	}
}

func TestCountInstalledPacks(t *testing.T) {
	storeDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(storeDir, "clpfd", "1.4.3"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(storeDir, "clpfd", "1.4.4"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(storeDir, "lists", "2.0.0"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(storeDir, ".lock"), []byte("123"), 0644))

	count, err := countInstalledPacks(storeDir)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}
