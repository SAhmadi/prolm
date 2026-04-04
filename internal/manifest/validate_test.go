package manifest

import (
	"strings"
	"testing"

	"github.com/prolm/prolm/pkg/prolfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validProlFile returns a minimal valid ProlFile for testing.
func validProlFile() *prolfile.ProlFile {
	return &prolfile.ProlFile{
		Meta: prolfile.Meta{
			ProlfileVersion: 1,
			MinProlmVersion: "0.1.0",
		},
		Package: prolfile.Package{
			Name:    "my-project",
			Version: "0.1.0",
			Entry:   "src/main.pl",
			Runtime: "swi",
		},
	}
}

func TestValidate_ValidMinimal(t *testing.T) {
	pf := validProlFile()
	assert.NoError(t, Validate(pf))
}

func TestValidate_ValidFull(t *testing.T) {
	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			ProlfileVersion: 1,
			MinProlmVersion: "0.1.0",
		},
		Package: prolfile.Package{
			Name:        "my-project",
			Version:     "1.2.3",
			Description: "A test project",
			Authors:     []string{"Ada <ada@example.com>"},
			License:     "MIT",
			Homepage:    "https://example.com",
			Entry:       "src/main.pl",
			Runtime:     "swi",
		},
		Dependencies:    map[string]string{"clpfd": "^1.4", "http": "~7.0.1"},
		DevDependencies: map[string]string{"plunit": "*"},
		Runtime: map[string]prolfile.RuntimeConfig{
			"swi": {MinVersion: "9.0", Flags: []string{"-O"}},
		},
		Scripts: map[string]string{"start": "prolm run"},
	}
	assert.NoError(t, Validate(pf))
}

func TestValidate_EmptyName(t *testing.T) {
	pf := validProlFile()
	pf.Package.Name = ""
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package name is required")
}

func TestValidate_NameCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "hello", false},
		{"with-hyphens", "my-pack", false},
		{"single-char", "a", false},
		{"digits", "lib42", false},
		{"namespaced", "owner/pack", false},
		{"namespaced-hyphens", "my-org/my-pack", false},
		{"uppercase", "Hello", true},
		{"leading-hyphen", "-foo", true},
		{"trailing-hyphen", "foo-", true},
		{"double-hyphen", "foo--bar", false}, // allowed: internal hyphens
		{"spaces", "foo bar", true},
		{"special-chars", "foo@bar", true},
		{"empty-namespace", "/name", true},
		{"empty-name-after-slash", "owner/", true},
		{"double-slash", "a/b/c", true},
		{"underscore", "my_pack", false},              // SWI packs like list_util use underscores
		{"underscore-swi-style", "pro_sqlite3", false}, // real SWI pack name
		{"namespaced-underscore", "owner/my_pack", false},
		{"leading-underscore", "_foo", true},
		{"dot", "my.pack", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := validProlFile()
			pf.Package.Name = tt.input
			err := Validate(pf)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "name")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_NullByteInName(t *testing.T) {
	pf := validProlFile()
	pf.Package.Name = "hello\x00world"
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "null byte")
}

func TestValidate_OverlongName(t *testing.T) {
	pf := validProlFile()
	pf.Package.Name = strings.Repeat("a", maxNameLen+1)
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestValidate_InvalidVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid", "1.2.3", false},
		{"valid-prerelease", "1.0.0-beta.1", false},
		{"valid-v-prefix", "v1.2.3", false},
		{"empty", "", true},
		{"not-semver", "abc", true},
		{"partial", "1.2", false}, // Masterminds/semver accepts "1.2" as "1.2.0"
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := validProlFile()
			pf.Package.Version = tt.version
			err := Validate(pf)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "version")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_Entry(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantErr bool
		errMsg  string
	}{
		{"valid", "src/main.pl", false, ""},
		{"empty-ok", "", false, ""},
		{"not-pl", "main.py", true, ".pl"},
		{"absolute", "/etc/main.pl", true, "relative"},
		{"path-traversal", "../../etc/passwd.pl", true, ".."},
		{"dot-dot-middle", "src/../etc/main.pl", true, ".."},
		{"null-byte", "src/\x00main.pl", true, "null byte"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := validProlFile()
			pf.Package.Entry = tt.entry
			err := Validate(pf)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_OverlongEntry(t *testing.T) {
	pf := validProlFile()
	pf.Package.Entry = "src/" + strings.Repeat("a", maxEntryLen) + ".pl"
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestValidate_Runtime(t *testing.T) {
	tests := []struct {
		runtime string
		wantErr bool
	}{
		{"swi", false},
		{"gnu", false},
		{"scryer", false},
		{"", false},
		{"python", true},
		{"SWI", true},
	}
	for _, tt := range tests {
		t.Run(tt.runtime, func(t *testing.T) {
			pf := validProlFile()
			pf.Package.Runtime = tt.runtime
			err := Validate(pf)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "runtime")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_DepVersionConstraints(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		wantErr    bool
	}{
		{"caret", "^1.4", false},
		{"tilde", "~1.4.2", false},
		{"gte", ">=1.0.0", false},
		{"exact", "=1.4.3", false},
		{"wildcard", "*", false},
		{"bare-version", "1.4.3", false},
		{"invalid", "not-a-version", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := validProlFile()
			pf.Dependencies = map[string]string{"clpfd": tt.constraint}
			err := Validate(pf)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "constraint")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_DepNameValidation(t *testing.T) {
	pf := validProlFile()
	pf.Dependencies = map[string]string{"INVALID": "^1.0"}
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid identifier")
}

func TestValidate_DepNameWithUnderscore(t *testing.T) {
	// SWI-Prolog packs such as list_util and pro_sqlite3 use underscores.
	// The manifest must accept them, matching the registry's nameRe.
	pf := validProlFile()
	pf.Dependencies = map[string]string{"list_util": "^1.0", "pro_sqlite3": "^0.9"}
	err := Validate(pf)
	assert.NoError(t, err)
}

func TestValidate_DevDepValidation(t *testing.T) {
	pf := validProlFile()
	pf.DevDependencies = map[string]string{"bad name": "*"}
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev-dependencies")
}

func TestValidate_InvalidRuntimeConfig(t *testing.T) {
	pf := validProlFile()
	pf.Runtime = map[string]prolfile.RuntimeConfig{
		"python": {MinVersion: "3.0"},
	}
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime.python")
}

func TestValidate_MultipleErrors(t *testing.T) {
	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{ProlfileVersion: 1},
		Package: prolfile.Package{
			Name:    "", // error 1
			Version: "", // error 2
			Entry:   "main.py", // error 3
			Runtime: "python",  // error 4
		},
	}
	err := Validate(pf)
	require.Error(t, err)

	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	assert.GreaterOrEqual(t, len(ve.Errors), 4, "expected at least 4 validation errors")
}

// --- QUALITY-004: Null byte checks on free-text string fields ---

func TestValidate_NullByteInDescription(t *testing.T) {
	pf := validProlFile()
	pf.Package.Description = "hello\x00world"
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description contains null byte")
}

func TestValidate_NullByteInLicense(t *testing.T) {
	pf := validProlFile()
	pf.Package.License = "MIT\x00"
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "license contains null byte")
}

func TestValidate_NullByteInHomepage(t *testing.T) {
	pf := validProlFile()
	pf.Package.Homepage = "https://example.com\x00"
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "homepage contains null byte")
}

func TestValidate_NullByteInAuthors(t *testing.T) {
	pf := validProlFile()
	pf.Package.Authors = []string{"Ada\x00Evil"}
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authors[0] contains null byte")
}

func TestValidate_NullByteInScripts(t *testing.T) {
	pf := validProlFile()
	pf.Scripts = map[string]string{"start": "prolm run\x00"}
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[scripts].start contains null byte")
}

func TestValidate_OverlongDescription(t *testing.T) {
	pf := validProlFile()
	pf.Package.Description = strings.Repeat("a", maxStringFieldLen+1)
	err := Validate(pf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description exceeds")
}

func TestValidationError_SingleMessage(t *testing.T) {
	ve := &ValidationError{Errors: []string{"name is required"}}
	assert.Equal(t, "validation error: name is required", ve.Error())
}

func TestValidationError_MultipleMessages(t *testing.T) {
	ve := &ValidationError{Errors: []string{"a", "b"}}
	msg := ve.Error()
	assert.Contains(t, msg, "validation errors:")
	assert.Contains(t, msg, "a")
	assert.Contains(t, msg, "b")
}
