package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extraction limits (SEC-12).
const (
	maxExtractedSize  = 200 << 20 // 200 MB total extracted
	maxFileCount      = 10_000
	maxSingleFileSize = 20 << 20 // 20 MB per file
)

// ErrPathTraversal indicates an illegal path was found in a tarball (SEC-2).
type ErrPathTraversal struct {
	EntryPath string
}

func (e *ErrPathTraversal) Error() string {
	return fmt.Sprintf("illegal path in tarball: %s", e.EntryPath)
}

// ErrExtractionLimit indicates a size, count, or single-file limit was exceeded (SEC-12).
type ErrExtractionLimit struct {
	Reason string
}

func (e *ErrExtractionLimit) Error() string {
	return fmt.Sprintf("extraction limit exceeded: %s", e.Reason)
}

// Unpack extracts a .tar.gz tarball to destDir with full security checks (SEC-2, SEC-12, SEC-13).
// On any violation, partial extraction is deleted and a hard error is returned.
func Unpack(tarballPath string, destDir string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("opening tarball: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("decompressing tarball: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	// Create destDir; on any error, clean up partial extraction.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(destDir)
		}
	}()

	var totalSize int64
	var fileCount int

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tarball entry: %w", err)
		}

		// Reject null bytes in entry name.
		if strings.ContainsRune(header.Name, 0) {
			return &ErrPathTraversal{EntryPath: "(contains null byte)"}
		}

		// Reject absolute paths before safeExtract (filepath.Join replaces
		// first arg when second is absolute on some platforms).
		if filepath.IsAbs(header.Name) {
			return &ErrPathTraversal{EntryPath: header.Name}
		}

		dest, err := safeExtract(destDir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", header.Name, err)
			}

		case tar.TypeReg:
			if header.Size > maxSingleFileSize {
				return &ErrExtractionLimit{
					Reason: fmt.Sprintf("file %s is %d bytes (max %d)", header.Name, header.Size, maxSingleFileSize),
				}
			}
			totalSize += header.Size
			if totalSize > maxExtractedSize {
				return &ErrExtractionLimit{
					Reason: fmt.Sprintf("total extracted size exceeds %d bytes", maxExtractedSize),
				}
			}
			fileCount++
			if fileCount > maxFileCount {
				return &ErrExtractionLimit{
					Reason: fmt.Sprintf("file count exceeds %d", maxFileCount),
				}
			}
			if err := extractRegularFile(dest, tr, header.Size); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := validateSymlink(destDir, dest, header.Linkname); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, dest); err != nil {
				return fmt.Errorf("creating symlink %s: %w", header.Name, err)
			}

		case tar.TypeLink:
			// Validate that the hardlink target stays within destDir (SEC-015).
			// Use the validated path directly — never reconstruct from raw header.
			linkTarget, err := safeExtract(destDir, header.Linkname)
			if err != nil {
				return err
			}
			if err := os.Link(linkTarget, dest); err != nil {
				return fmt.Errorf("creating hardlink %s: %w", header.Name, err)
			}

		default:
			// Reject device files, named pipes, sockets, and other special types.
			return &ErrPathTraversal{
				EntryPath: fmt.Sprintf("%s (unsupported type: %d)", header.Name, header.Typeflag),
			}
		}
	}

	success = true
	return nil
}

// safeExtract validates that entryPath resolves safely within destDir.
// Implements the exact pattern from CLAUDE.md 8.5 (SEC-2).
func safeExtract(destDir, entryPath string) (string, error) {
	dest := filepath.Join(destDir, entryPath)
	dest = filepath.Clean(dest)
	if !strings.HasPrefix(dest, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", &ErrPathTraversal{EntryPath: entryPath}
	}
	return dest, nil
}

// validateSymlink checks that a symlink target resolves within destDir (SEC-13).
func validateSymlink(destDir, linkPath, target string) error {
	// Resolve symlink target relative to the link's parent directory.
	var resolved string
	if filepath.IsAbs(target) {
		return &ErrPathTraversal{EntryPath: fmt.Sprintf("symlink %s -> %s (absolute target)", linkPath, target)}
	}
	resolved = filepath.Join(filepath.Dir(linkPath), target)
	resolved = filepath.Clean(resolved)

	if !strings.HasPrefix(resolved, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return &ErrPathTraversal{
			EntryPath: fmt.Sprintf("symlink target escapes destination: %s -> %s", linkPath, target),
		}
	}
	return nil
}

// extractRegularFile writes a tar entry to dest with 0644 permissions.
// Uses io.LimitReader to enforce size limits at the I/O level.
func extractRegularFile(dest string, r io.Reader, size int64) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", dest, err)
	}
	defer f.Close()

	// LimitReader prevents reading more than declared size (defense in depth).
	written, err := io.Copy(f, io.LimitReader(r, size+1))
	if err != nil {
		return fmt.Errorf("writing file %s: %w", dest, err)
	}
	if written > size {
		return &ErrExtractionLimit{
			Reason: fmt.Sprintf("file %s wrote more bytes than declared size %d", dest, size),
		}
	}

	return nil
}
