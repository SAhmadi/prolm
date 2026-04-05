// Package scaffold implements `prolm new` and `prolm init` project creation.
package scaffold

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/prolm/prolm/internal/manifest"
	"github.com/prolm/prolm/internal/ui"
	"github.com/prolm/prolm/pkg/prolfile"
)

//go:embed all:templates/app
var appTemplates embed.FS

// validTemplates lists templates available in Phase 1.
var validTemplates = map[string]bool{
	"app": true,
}

// validRuntimes lists supported Prolog runtimes.
var validRuntimes = map[string]bool{
	"swi":    true,
	"gnu":    true,
	"scryer": true,
}

// testAfterCreate is a hook for testing cleanup behavior.
// It is called (when non-nil) immediately after the project directory is
// created and created=true is set. Only set in tests (same-package access).
var testAfterCreate func(projectDir string) error

// templateData holds values passed to all templates.
type templateData struct {
	Name    string
	Runtime string
}

// templateMapping maps embedded template paths to output file paths.
var templateMapping = []struct {
	src  string // path inside embed.FS
	dest string // path relative to project root
}{
	{"templates/app/main.pl.tmpl", "src/main.pl"},
	{"templates/app/main_test.pl.tmpl", "tests/main_test.pl"},
	{"templates/app/.gitignore.tmpl", ".gitignore"},
	{"templates/app/README.md.tmpl", "README.md"},
}

// NewProject creates a new Prolog project in a subdirectory named after name.
// It creates the directory structure, generates Prolfile.toml, renders template
// files, and optionally initialises a git repository.
func NewProject(name, tmpl, runtime string) error {
	if err := validateNewProjectName(name); err != nil {
		return err
	}
	if !validTemplates[tmpl] {
		ui.Error("unknown template %q; available: app", tmpl)
		ui.Hint("Use --template app (the only template available in this release)")
		return fmt.Errorf("unknown template %q", tmpl)
	}
	if !validRuntimes[runtime] {
		ui.Error("unknown runtime %q; must be one of: swi, gnu, scryer", runtime)
		ui.Hint("Use --runtime swi, --runtime gnu, or --runtime scryer")
		return fmt.Errorf("unknown runtime %q", runtime)
	}

	projectDir := filepath.Join(".", name)

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	// Track whether we created the directory so we can clean up on error.
	created := false
	defer func() {
		if created {
			// An error occurred after directory creation; clean up.
			os.RemoveAll(projectDir)
		}
	}()

	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}
	created = true

	if testAfterCreate != nil {
		if err := testAfterCreate(projectDir); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "tests"), 0755); err != nil {
		return fmt.Errorf("creating tests directory: %w", err)
	}

	// Generate Prolfile.toml via manifest.Save() for deterministic serialization.
	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			ProlfileVersion: prolfile.CurrentProlfileVersion,
			MinProlmVersion: "0.1.0",
		},
		Package: prolfile.Package{
			Name:    name,
			Version: "0.1.0",
			Entry:   "src/main.pl",
			Runtime: runtime,
		},
	}
	if err := manifest.Save(filepath.Join(projectDir, "Prolfile.toml"), pf); err != nil {
		return fmt.Errorf("writing Prolfile.toml: %w", err)
	}

	// Render template files.
	data := templateData{Name: name, Runtime: runtime}
	for _, m := range templateMapping {
		if err := renderTemplate(projectDir, m.src, m.dest, data); err != nil {
			return fmt.Errorf("rendering %s: %w", m.dest, err)
		}
	}

	// Create empty Prolfile.lock.
	if err := os.WriteFile(filepath.Join(projectDir, "Prolfile.lock"), nil, 0644); err != nil {
		return fmt.Errorf("creating Prolfile.lock: %w", err)
	}

	// Create .gitattributes with merge strategy (per 8.19).
	gitattrs := []byte("Prolfile.lock merge=ours\n")
	if err := os.WriteFile(filepath.Join(projectDir, ".gitattributes"), gitattrs, 0644); err != nil {
		return fmt.Errorf("creating .gitattributes: %w", err)
	}

	// Run git init (best-effort).
	gitInit(projectDir)

	// Success — prevent cleanup.
	created = false

	ui.Success("Created project %q", name)
	ui.Hint("cd %s && prolm install", name)
	return nil
}

// validateNewProjectName checks that name is safe for use as a directory name
// and is a valid package identifier.
func validateNewProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("project name %q is not allowed", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("project name %q must not contain path separators", name)
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("project name %q must not be an absolute path", name)
	}
	return manifest.ValidateName(name)
}

// renderTemplate reads a template from the embedded FS, executes it with data,
// and writes the result to destPath inside projectDir.
func renderTemplate(projectDir, src, dest string, data templateData) error {
	content, err := appTemplates.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading embedded template %s: %w", src, err)
	}

	tmpl, err := template.New(filepath.Base(src)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", src, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template %s: %w", src, err)
	}

	outPath := filepath.Join(projectDir, dest)
	return os.WriteFile(outPath, []byte(buf.String()), 0644)
}

// gitInit runs `git init` in dir. Errors are silently ignored since git
// availability is not required.
func gitInit(dir string) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return
	}
	cmd := exec.Command(gitPath, "init", dir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	_ = cmd.Run()
}
