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

// useModuleRe matches a complete use_module(library(...)) directive,
// tolerating internal whitespace so that collapsed multi-line text matches.
var useModuleRe = regexp.MustCompile(`^\s*:-\s*use_module\(\s*library\(\s*([a-z][a-z0-9_/]*)\s*\)\s*\)`)

// directiveStartRe matches the opening of a use_module directive.
// Used to detect multi-line directives when useModuleRe does not match.
var directiveStartRe = regexp.MustCompile(`^\s*:-\s*use_module\(`)

// swiBuiltins contains SWI-Prolog standard libraries that ship with the
// runtime and should not be treated as external dependencies.
//
// Validated against SWI-Prolog 9.2.x standard library list.
// To regenerate, run:
//
//	swipl -g "forall((file_search_path(library,D), exists_directory(D),
//	  directory_files(D,Fs), member(F,Fs), file_name_extension(Base,pl,F)),
//	  writeln(Base)), halt."
//
// This list is intentionally conservative: prefer false positives (user
// sees a built-in listed as a dep and removes it) over false negatives
// (real external dep silently excluded).
var swiBuiltins = map[string]bool{
	"aggregate":          true,
	"ansi_term":          true,
	"apply":              true,
	"assoc":              true,
	"broadcast":          true,
	"char_type":          true,
	"check":              true,
	"csv":                true,
	"debug":              true,
	"dicts":              true,
	"edinburgh":          true,
	"error":              true,
	"gensym":             true,
	"listing":            true,
	"lists":              true,
	"main":               true,
	"modules":            true,
	"nb_set":             true,
	"occurs":             true,
	"option":             true,
	"optparse":           true,
	"ordsets":            true,
	"pairs":              true,
	"persistency":        true,
	"pldoc":              true,
	"plunit":             true,
	"predicate_options":  true,
	"prolog_colour":      true,
	"prolog_stack":       true,
	"prolog_xref":        true,
	"pure_input":         true,
	"quasi_quotations":   true,
	"random":             true,
	"readutil":           true,
	"record":             true,
	"settings":           true,
	"socket":             true,
	"solution_sequences": true,
	"statistics":         true,
	"strings":            true,
	"system":             true,
	"terms":              true,
	"thread":             true,
	"thread_pool":        true,
	"threadutil":         true,
	"url":                true,
	"varnumbers":         true,
	"www_browser":        true,
	"xpath":              true,
	"dcg/basics":         true,
	"dcg/high_order":     true,
}

// addIfExternal checks whether lib refers to an external dependency and,
// if so, adds it to deps. Built-in SWI-Prolog libraries are skipped.
func addIfExternal(deps map[string]string, lib string) {
	name := strings.SplitN(lib, "/", 2)[0]
	if swiBuiltins[name] || swiBuiltins[lib] {
		return
	}
	deps[name] = "*"
}

// maxDirectiveLen is the maximum accumulated length (in bytes) before
// abandoning a multi-line directive to avoid runaway buffering.
const maxDirectiveLen = 500

// ScanDeps walks all .pl files under dir and extracts external library
// dependencies from :- use_module(library(...)) directives.
// Returns a map of dependency names to "*" (any version).
// Built-in SWI-Prolog libraries are excluded.
//
// Multi-line directives are supported: when a line starts a use_module
// directive but does not close it, subsequent lines are accumulated
// until the closing "). " is found.
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

		var buf strings.Builder
		inDirective := false

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// While accumulating a multi-line directive, append lines
			// until the closing ")." is found.
			if inDirective {
				buf.WriteString(" ")
				buf.WriteString(trimmed)

				if strings.Contains(trimmed, ").") {
					if m := useModuleRe.FindStringSubmatch(buf.String()); m != nil {
						addIfExternal(deps, m[1])
					}
					buf.Reset()
					inDirective = false
				} else if buf.Len() > maxDirectiveLen {
					buf.Reset()
					inDirective = false
				}
				continue
			}

			// Skip commented-out lines.
			if strings.HasPrefix(trimmed, "%") {
				continue
			}

			// Fast path: single-line directive.
			if m := useModuleRe.FindStringSubmatch(line); m != nil {
				addIfExternal(deps, m[1])
				continue
			}

			// Check for the start of a multi-line directive.
			if directiveStartRe.MatchString(line) {
				buf.Reset()
				buf.WriteString(trimmed)
				inDirective = true
			}
		}
		if err := scanner.Err(); err != nil {
			ui.Warn("scan error in %s: %s", path, err)
		}
		return nil
	})

	return deps, err
}
