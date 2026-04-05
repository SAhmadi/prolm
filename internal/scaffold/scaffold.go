// Package scaffold implements `prolm new` and `prolm init` project creation.
package scaffold

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	if runtime == "" || manifest.ValidateRuntime(runtime) != nil {
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

	// Generate Prolfile.toml programmatically via manifest.Save() for
	// deterministic serialization (per 8.28). There is intentionally no
	// Prolfile.toml.tmpl template; the manifest package owns the format.
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

// sanitizeRe matches characters not allowed in a package name.
var sanitizeRe = regexp.MustCompile(`[^a-z0-9_-]+`)

// sanitizeDirName converts a directory name into a valid package name
// candidate by lowercasing and replacing invalid characters with hyphens.
func sanitizeDirName(name string) string {
	name = strings.ToLower(name)
	name = sanitizeRe.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

// InitProject initialises a prolm project in an existing directory by
// creating Prolfile.toml and an empty Prolfile.lock. Unlike NewProject,
// it does not create source files, directories, or run git init.
func InitProject(dir string, scan, yes bool) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	prolfilePath := filepath.Join(absDir, "Prolfile.toml")
	if _, err := os.Stat(prolfilePath); err == nil {
		ui.Error("Prolfile.toml already exists in %s", absDir)
		ui.Hint("Use `prolm install` to install dependencies from the existing Prolfile.toml")
		return fmt.Errorf("Prolfile.toml already exists")
	}

	// Derive defaults from the directory name.
	defaultName := sanitizeDirName(filepath.Base(absDir))
	defaultVersion := "0.1.0"
	defaultEntry := "src/main.pl"
	defaultRuntime := "swi"

	var name, version, entry, runtime string

	if yes {
		// Non-interactive: use defaults, fail if name is invalid.
		name = defaultName
		if err := manifest.ValidateName(name); err != nil {
			ui.Error("cannot derive a valid project name from directory %q", filepath.Base(absDir))
			ui.Hint("Run `prolm init` without --yes to enter a name interactively")
			return fmt.Errorf("invalid project name %q: %w", name, err)
		}
		version = defaultVersion
		entry = defaultEntry
		runtime = defaultRuntime
	} else {
		// Interactive: prompt for each value.
		p := newPrompter()

		name = p.ask("Project name", defaultName)
		if err := manifest.ValidateName(name); err != nil {
			ui.Error("invalid project name %q: %s", name, err)
			// Re-prompt once.
			name = p.ask("Project name", defaultName)
			if err := manifest.ValidateName(name); err != nil {
				return fmt.Errorf("invalid project name %q: %w", name, err)
			}
		}

		version = p.ask("Version", defaultVersion)

		entry = p.ask("Entry point", defaultEntry)

		runtime = p.ask("Runtime (swi, gnu, scryer)", defaultRuntime)
		if err := manifest.ValidateRuntime(runtime); err != nil {
			ui.Error("invalid runtime %q: %s", runtime, err)
			runtime = p.ask("Runtime (swi, gnu, scryer)", defaultRuntime)
			if err := manifest.ValidateRuntime(runtime); err != nil {
				return fmt.Errorf("invalid runtime %q: %w", runtime, err)
			}
		}
	}

	// Scan for dependencies if requested.
	var deps map[string]string
	if scan {
		deps, err = ScanDeps(absDir)
		if err != nil {
			ui.Warn("dependency scan failed: %s", err)
			deps = nil
		}
	}

	pf := &prolfile.ProlFile{
		Meta: prolfile.Meta{
			ProlfileVersion: prolfile.CurrentProlfileVersion,
			MinProlmVersion: "0.1.0",
		},
		Package: prolfile.Package{
			Name:    name,
			Version: version,
			Entry:   entry,
			Runtime: runtime,
		},
		Dependencies: deps,
	}

	if err := manifest.Save(prolfilePath, pf); err != nil {
		return fmt.Errorf("writing Prolfile.toml: %w", err)
	}

	// Create empty Prolfile.lock.
	lockPath := filepath.Join(absDir, "Prolfile.lock")
	if err := os.WriteFile(lockPath, nil, 0644); err != nil {
		return fmt.Errorf("creating Prolfile.lock: %w", err)
	}

	ui.Success("Initialized project %q", name)
	if len(deps) > 0 {
		ui.Info("Detected %d dependencies from .pl files", len(deps))
		ui.Hint("Review [dependencies] in Prolfile.toml — the scanner may have missed some")
	}
	ui.Hint("Run `prolm install` to install dependencies")
	return nil
}
