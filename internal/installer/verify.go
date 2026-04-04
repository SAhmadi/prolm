package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// checksumRe matches a valid sha256 checksum string: "sha256:" followed by 64 hex chars.
var checksumRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ErrChecksumMismatch indicates the computed checksum does not match the expected value.
// The file at FilePath is deleted when this error is returned.
type ErrChecksumMismatch struct {
	FilePath string
	Expected string
	Got      string
}

func (e *ErrChecksumMismatch) Error() string {
	return fmt.Sprintf("checksum mismatch for %s: expected %s, got %s", e.FilePath, e.Expected, e.Got)
}

// ComputeChecksum returns the SHA-256 checksum of the file at filePath
// in the format "sha256:<hex>".
func ComputeChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("computing checksum: %w", err)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// Verify computes the SHA-256 checksum of filePath and compares it to
// expectedChecksum (format "sha256:<hex>"). On mismatch, the file is
// deleted and *ErrChecksumMismatch is returned (SEC-1).
func Verify(filePath string, expectedChecksum string) error {
	if !checksumRe.MatchString(expectedChecksum) {
		return fmt.Errorf("invalid checksum format: %q (expected sha256:<64 hex chars>)", expectedChecksum)
	}

	got, err := ComputeChecksum(filePath)
	if err != nil {
		return err
	}

	if !strings.EqualFold(got, expectedChecksum) {
		os.Remove(filePath) // best-effort delete on mismatch
		return &ErrChecksumMismatch{
			FilePath: filePath,
			Expected: expectedChecksum,
			Got:      got,
		}
	}

	return nil
}
