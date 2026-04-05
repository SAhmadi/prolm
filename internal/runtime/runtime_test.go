package runtime

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Factory tests
// ---------------------------------------------------------------------------

func TestDetectFactory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"swi explicit", "swi", false},
		{"empty defaults to swi", "", false},
		{"gnu unsupported", "gnu", true},
		{"scryer unsupported", "scryer", true},
		{"unknown", "foobar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt, err := Detect(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				var target *ErrUnsupportedRuntime
				assert.True(t, errors.As(err, &target))
				assert.Nil(t, rt)
			} else {
				require.NoError(t, err)
				require.NotNil(t, rt)
				assert.IsType(t, &SWIRuntime{}, rt)
				assert.Equal(t, "swi", rt.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SWIRuntime.Detect tests
// ---------------------------------------------------------------------------

func newTestSWI(lookPathResult string, lookPathErr error, versionOutput string, versionErr error) *SWIRuntime {
	return &SWIRuntime{
		lookPath: func(name string) (string, error) {
			return lookPathResult, lookPathErr
		},
		runCmd: func(name string, args ...string) ([]byte, error) {
			return []byte(versionOutput), versionErr
		},
		execReplace: func(binary string, argv []string, env []string) error {
			return nil
		},
		fileExists: func(path string) bool {
			return false // default: no fallback paths exist
		},
	}
}

func TestSWIRuntime_Detect(t *testing.T) {
	notFound := &exec.Error{Name: "swipl", Err: exec.ErrNotFound}

	tests := []struct {
		name          string
		swi           *SWIRuntime
		envPath       string
		wantPath      string
		wantVersion   string
		wantErrNotFound bool
		wantErrParse    bool
		wantErrSubstr   string
	}{
		{
			name:        "happy path via PATH",
			swi:         newTestSWI("/usr/local/bin/swipl", nil, "SWI-Prolog version 9.2.1 for x86_64-darwin\n", nil),
			wantPath:    "/usr/local/bin/swipl",
			wantVersion: "9.2.1",
		},
		{
			name: "fallback to /opt/homebrew/bin",
			swi: func() *SWIRuntime {
				s := newTestSWI("", notFound, "SWI-Prolog version 9.3.0 for arm64-darwin\n", nil)
				s.fileExists = func(path string) bool {
					return path == "/opt/homebrew/bin/swipl"
				}
				return s
			}(),
			wantPath:    "/opt/homebrew/bin/swipl",
			wantVersion: "9.3.0",
		},
		{
			name: "fallback to /snap/bin",
			swi: func() *SWIRuntime {
				s := newTestSWI("", notFound, "SWI-Prolog version 9.0.4 for x86_64-linux\n", nil)
				s.fileExists = func(path string) bool {
					return path == "/snap/bin/swipl"
				}
				return s
			}(),
			wantPath:    "/snap/bin/swipl",
			wantVersion: "9.0.4",
		},
		{
			name:            "not found anywhere",
			swi:             newTestSWI("", notFound, "", nil),
			wantErrNotFound: true,
		},
		{
			name:         "garbled version output",
			swi:          newTestSWI("/usr/bin/swipl", nil, "not a version string\n", nil),
			wantErrParse: true,
		},
		{
			name:          "version command fails",
			swi:           newTestSWI("/usr/bin/swipl", nil, "", fmt.Errorf("exit status 1")),
			wantErrSubstr: "running /usr/bin/swipl --version",
		},
		{
			name: "PROLM_RUNTIME_PATH override",
			swi: func() *SWIRuntime {
				s := newTestSWI("", notFound, "SWI-Prolog version 9.1.0 for x86_64-linux\n", nil)
				s.fileExists = func(path string) bool {
					return path == "/custom/path/swipl"
				}
				return s
			}(),
			envPath:     "/custom/path/swipl",
			wantPath:    "/custom/path/swipl",
			wantVersion: "9.1.0",
		},
		{
			name: "PROLM_RUNTIME_PATH file not found",
			swi: func() *SWIRuntime {
				s := newTestSWI("", notFound, "", nil)
				s.fileExists = func(path string) bool { return false }
				return s
			}(),
			envPath:       "/nonexistent/swipl",
			wantErrSubstr: "PROLM_RUNTIME_PATH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envPath != "" {
				t.Setenv("PROLM_RUNTIME_PATH", tt.envPath)
			} else {
				t.Setenv("PROLM_RUNTIME_PATH", "")
			}

			info, err := tt.swi.Detect()

			if tt.wantErrNotFound {
				require.Error(t, err)
				var target *ErrRuntimeNotFound
				assert.True(t, errors.As(err, &target), "expected ErrRuntimeNotFound, got %T", err)
				assert.Nil(t, info)
				return
			}
			if tt.wantErrParse {
				require.Error(t, err)
				var target *ErrVersionParse
				assert.True(t, errors.As(err, &target), "expected ErrVersionParse, got %T", err)
				assert.Nil(t, info)
				return
			}
			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.wantPath, info.Path)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, info, tt.swi.Info())
		})
	}
}

// ---------------------------------------------------------------------------
// BuildRunArgs tests
// ---------------------------------------------------------------------------

func TestSWIRuntime_BuildRunArgs(t *testing.T) {
	s := NewSWIRuntime()

	tests := []struct {
		name  string
		entry string
		deps  []string
		flags []string
		goal  string
		want  []string
	}{
		{
			name:  "full invocation",
			entry: "src/main",
			deps:  []string{"/store/clpfd/1.4.3"},
			flags: []string{"-O", "--stack-limit=2g"},
			goal:  "main",
			want: []string{
				"-O", "--stack-limit=2g",
				"-g", "use_module('/store/clpfd/1.4.3')",
				"-g", "use_module('src/main')",
				"-g", "main",
				"-t", "halt",
			},
		},
		{
			name:  "no deps no flags",
			entry: "src/main",
			deps:  nil,
			flags: nil,
			goal:  "main",
			want: []string{
				"-g", "use_module('src/main')",
				"-g", "main",
				"-t", "halt",
			},
		},
		{
			name:  "multiple deps",
			entry: "src/app",
			deps:  []string{"dep1", "dep2", "dep3"},
			flags: []string{"-O"},
			goal:  "start",
			want: []string{
				"-O",
				"-g", "use_module('dep1')",
				"-g", "use_module('dep2')",
				"-g", "use_module('dep3')",
				"-g", "use_module('src/app')",
				"-g", "start",
				"-t", "halt",
			},
		},
		{
			name:  "custom goal",
			entry: "src/diagnose",
			deps:  nil,
			flags: nil,
			goal:  "diagnose(network)",
			want: []string{
				"-g", "use_module('src/diagnose')",
				"-g", "diagnose(network)",
				"-t", "halt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.BuildRunArgs(tt.entry, tt.deps, tt.flags, tt.goal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// BuildTestArgs tests
// ---------------------------------------------------------------------------

func TestSWIRuntime_BuildTestArgs(t *testing.T) {
	s := NewSWIRuntime()

	tests := []struct {
		name      string
		testFiles []string
		deps      []string
		flags     []string
		want      []string
	}{
		{
			name:      "single test file with dep",
			testFiles: []string{"tests/main_test"},
			deps:      []string{"dep1"},
			flags:     nil,
			want: []string{
				"-g", "use_module(library(plunit))",
				"-g", "use_module('dep1')",
				"-g", "use_module('tests/main_test')",
				"-g", "run_tests",
				"-t", "halt",
			},
		},
		{
			name:      "multiple test files no deps",
			testFiles: []string{"t1", "t2"},
			deps:      nil,
			flags:     nil,
			want: []string{
				"-g", "use_module(library(plunit))",
				"-g", "use_module('t1')",
				"-g", "use_module('t2')",
				"-g", "run_tests",
				"-t", "halt",
			},
		},
		{
			name:      "with flags",
			testFiles: []string{"tests/rules_test"},
			deps:      []string{"dep1"},
			flags:     []string{"-O"},
			want: []string{
				"-O",
				"-g", "use_module(library(plunit))",
				"-g", "use_module('dep1')",
				"-g", "use_module('tests/rules_test')",
				"-g", "run_tests",
				"-t", "halt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.BuildTestArgs(tt.testFiles, tt.deps, tt.flags)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// BuildCheckArgs tests
// ---------------------------------------------------------------------------

func TestSWIRuntime_BuildCheckArgs(t *testing.T) {
	s := NewSWIRuntime()

	tests := []struct {
		name string
		files []string
		deps  []string
		want  []string
	}{
		{
			name:  "single file with dep",
			files: []string{"src/main.pl"},
			deps:  []string{"dep1"},
			want: []string{
				"-g", "use_module('dep1')",
				"-g", "load_files('src/main.pl')",
				"-t", "halt",
			},
		},
		{
			name:  "multiple files no deps",
			files: []string{"src/a.pl", "src/b.pl"},
			deps:  nil,
			want: []string{
				"-g", "load_files('src/a.pl')",
				"-g", "load_files('src/b.pl')",
				"-t", "halt",
			},
		},
		{
			name:  "no files no deps",
			files: nil,
			deps:  nil,
			want:  []string{"-t", "halt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.BuildCheckArgs(tt.files, tt.deps)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// CheckMinVersion tests
// ---------------------------------------------------------------------------

func TestCheckMinVersion(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		minVersion     string
		wantErr        bool
		wantVersionOld bool
	}{
		{"version above min", "9.2.1", "9.0.0", false, false},
		{"version equals min", "9.2.1", "9.2.1", false, false},
		{"version below min", "9.0.0", "9.2.0", true, true},
		{"empty min version", "9.2.1", "", false, false},
		{"invalid found version", "invalid", "9.0.0", true, false},
		{"invalid min version", "9.2.1", "bad", true, false},
		{"patch below", "9.2.0", "9.2.1", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &RuntimeInfo{Path: "/usr/bin/swipl", Version: tt.version}
			err := CheckMinVersion(info, "swi", tt.minVersion)

			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.wantVersionOld {
				var target *ErrVersionTooOld
				if assert.True(t, errors.As(err, &target), "expected ErrVersionTooOld, got %T", err) {
					assert.Equal(t, "swi", target.Runtime, "runtime name must propagate into ErrVersionTooOld")
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Exec tests
// ---------------------------------------------------------------------------

func TestSWIRuntime_Exec(t *testing.T) {
	t.Run("exec before detect", func(t *testing.T) {
		s := NewSWIRuntime()
		err := s.Exec([]string{"-g", "main", "-t", "halt"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Detect()")
	})

	t.Run("exec after detect", func(t *testing.T) {
		var capturedBinary string
		var capturedArgv []string

		s := &SWIRuntime{
			lookPath: func(name string) (string, error) {
				return "/usr/local/bin/swipl", nil
			},
			runCmd: func(name string, args ...string) ([]byte, error) {
				return []byte("SWI-Prolog version 9.2.1 for x86_64-darwin\n"), nil
			},
			execReplace: func(binary string, argv []string, env []string) error {
				capturedBinary = binary
				capturedArgv = argv
				return nil
			},
			fileExists: func(path string) bool { return false },
		}
		t.Setenv("PROLM_RUNTIME_PATH", "")

		_, err := s.Detect()
		require.NoError(t, err)

		err = s.Exec([]string{"-g", "main", "-t", "halt"})
		require.NoError(t, err)

		assert.Equal(t, "/usr/local/bin/swipl", capturedBinary)
		assert.Equal(t, []string{"/usr/local/bin/swipl", "-g", "main", "-t", "halt"}, capturedArgv)
	})
}

// ---------------------------------------------------------------------------
// Error message tests
// ---------------------------------------------------------------------------

func TestErrorMessages(t *testing.T) {
	t.Run("ErrRuntimeNotFound swi", func(t *testing.T) {
		err := &ErrRuntimeNotFound{Runtime: "swi"}
		assert.Contains(t, err.Error(), "swipl not found")
		assert.Contains(t, err.Error(), "swi-prolog.org")
		assert.Contains(t, err.Error(), "PROLM_RUNTIME_PATH")
	})

	t.Run("ErrRuntimeNotFound other", func(t *testing.T) {
		err := &ErrRuntimeNotFound{Runtime: "gnu"}
		assert.Contains(t, err.Error(), "gnu runtime not found")
	})

	t.Run("ErrUnsupportedRuntime", func(t *testing.T) {
		err := &ErrUnsupportedRuntime{Name: "foobar"}
		assert.Contains(t, err.Error(), "foobar")
		assert.Contains(t, err.Error(), "unsupported runtime")
	})

	t.Run("ErrVersionTooOld", func(t *testing.T) {
		err := &ErrVersionTooOld{Runtime: "swi", Found: "8.0.0", Required: "9.0.0"}
		assert.Contains(t, err.Error(), "8.0.0")
		assert.Contains(t, err.Error(), "9.0.0")
	})

	t.Run("ErrVersionParse", func(t *testing.T) {
		err := &ErrVersionParse{Output: "garbage"}
		assert.Contains(t, err.Error(), "garbage")
		assert.Contains(t, err.Error(), "could not parse")
	})
}
