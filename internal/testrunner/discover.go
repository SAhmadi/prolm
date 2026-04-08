package testrunner

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// skipDirs is the set of directory names that Discover never descends into.
// These are the usual suspects for vendor/cache/VCS content which must not
// produce test results and would slow discovery in large repos.
var skipDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	".prolm":       {},
	"vendor":       {},
}

// Discover walks projectDir and returns absolute paths of Prolog test files.
//
// A file is considered a test if its base name ends with "_test.pl" or starts
// with "test_" (and ends with ".pl"). Matches ROADMAP §1.12. Symlinks are not
// followed; hidden/vendor/VCS directories are skipped for speed and safety.
// Results are sorted for deterministic reporting.
func Discover(projectDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == projectDir {
				return nil
			}
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		// Only regular files — ignore symlinks to avoid traversal surprises.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".pl") {
			return nil
		}
		if strings.HasSuffix(name, "_test.pl") || strings.HasPrefix(name, "test_") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
