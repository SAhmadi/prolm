package checker

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// skipDirs is the set of directory names Discover never descends into.
var skipDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	".prolm":       {},
	"vendor":       {},
}

// Discover walks projectDir and returns absolute paths of Prolog source files
// suitable for `prolm check`.
//
// Test files (**/*_test.pl, **/test_*.pl) are excluded — `prolm test` covers
// those. Symlinks are not followed (avoids path-traversal surprises, mirrors
// testrunner.Discover). Results are sorted for deterministic reporting.
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
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".pl") {
			return nil
		}
		if strings.HasSuffix(name, "_test.pl") || strings.HasPrefix(name, "test_") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
