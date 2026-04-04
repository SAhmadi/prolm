// Package atomicfile provides atomic file writing (SEC-8).
// Content is written to a temporary file first, then renamed into place.
// A crash mid-write never leaves a partial file at the target path.
package atomicfile

import (
	"fmt"
	"os"
)

// Write atomically writes data to path with the given permissions.
// It writes to a .tmp sibling first, then renames into place.
// On rename failure the temporary file is removed (best-effort).
func Write(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("writing temporary file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("renaming temporary file: %w", err)
	}
	return nil
}
