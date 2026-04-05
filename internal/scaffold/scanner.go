package scaffold

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prolm/prolm/internal/ui"
)

// useModuleRe matches Prolog use_module(library(...)) directives.
// It captures the library path (e.g. "clpfd" or "http/http_server").
var useModuleRe = regexp.MustCompile(`^\s*:-\s*use_module\(library\(([a-z][a-z0-9_/]*)\)\)`)

// swiBuiltins contains SWI-Prolog standard libraries that ship with the
// runtime and should not be treated as external dependencies.
var swiBuiltins = map[string]bool{
	"aggregate":             true,
	"ansi_term":             true,
	"apply":                 true,
	"assoc":                 true,
	"broadcast":             true,
	"check":                 true,
	"debug":                 true,
	"edinburgh":             true,
	"error":                 true,
	"lists":                 true,
	"main":                  true,
	"modules":               true,
	"nb_set":                true,
	"occurs":                true,
	"option":                true,
	"optparse":              true,
	"ordsets":               true,
	"pairs":                 true,
	"persistency":           true,
	"plunit":                true,
	"predicate_options":     true,
	"prolog_stack":          true,
	"quasi_quotations":      true,
	"readutil":              true,
	"record":                true,
	"settings":              true,
	"solution_sequences":    true,
	"statistics":            true,
	"strings":               true,
	"system":                true,
	"terms":                 true,
	"thread_pool":           true,
	"threadutil":            true,
	"pldoc":                 true,
	"dcg/basics":            true,
	"dcg/high_order":        true,
}

// ScanDeps walks all .pl files under dir and extracts external library
// dependencies from :- use_module(library(...)) directives.
// Returns a map of dependency names to "*" (any version).
// Built-in SWI-Prolog libraries are excluded.
func ScanDeps(dir string) (map[string]string, error) {
	deps := make(map[string]string)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable directories
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".pl") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil // skip unreadable files
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip commented-out lines.
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "%") {
				continue
			}

			m := useModuleRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}

			lib := m[1]
			// Use first path segment as the dependency name.
			name := strings.SplitN(lib, "/", 2)[0]

			if swiBuiltins[name] {
				continue
			}
			// Also check the full path for built-ins like dcg/basics.
			if swiBuiltins[lib] {
				continue
			}

			deps[name] = "*"
		}
		if err := scanner.Err(); err != nil {
			ui.Warn("scan error in %s: %s", path, err)
		}
		return nil
	})

	return deps, err
}
