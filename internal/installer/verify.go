package installer

import (
	"crypto/sha1"
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

// sha1HexRe matches a bare 40-character lowercase hex SHA-1.
var sha1HexRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

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

// ComputeSHA1 returns the SHA-1 hash of the file at filePath as a lowercase
// hex string (no algorithm prefix). Used only for cross-checking against
// SWI-Prolog pack index checksums, which are advertised as raw sha1.
// SHA-1 is NOT suitable for security-critical integrity — see ComputeChecksum
// for the SHA-256 used in lockfile pinning.
func ComputeSHA1(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file for sha1: %w", err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("computing sha1: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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

	if got != expectedChecksum {
		os.Remove(filePath) // best-effort delete on mismatch
		return &ErrChecksumMismatch{
			FilePath: filePath,
			Expected: expectedChecksum,
			Got:      got,
		}
	}

	return nil
}

// VerifyRegistryChecksum cross-checks a downloaded tarball against a checksum
// advertised by the registry's API response (SEC-1). This closes the TOCTOU
// window between version lookup and lock entry creation.
//
// Accepted formats for advertised:
//   - "sha256:<64 hex>"  — verified via SHA-256
//   - "sha1:<40 hex>"    — verified via SHA-1
//   - "<40 hex>"         — bare SHA-1 (SWI pack index format)
//
// On mismatch, the file is deleted and *ErrChecksumMismatch is returned.
// If advertised is empty, the function is a no-op (registry did not provide
// a checksum). Unknown formats return an error rather than silently passing.
func VerifyRegistryChecksum(filePath, advertised string) error {
	if advertised == "" {
		return nil
	}

	switch {
	case checksumRe.MatchString(advertised):
		return Verify(filePath, advertised)

	case strings.HasPrefix(advertised, "sha1:") && sha1HexRe.MatchString(strings.TrimPrefix(advertised, "sha1:")):
		expected := strings.TrimPrefix(advertised, "sha1:")
		return verifySHA1(filePath, expected, advertised)

	case sha1HexRe.MatchString(advertised):
		return verifySHA1(filePath, advertised, "sha1:"+advertised)

	default:
		return fmt.Errorf("unrecognized registry checksum format: %q", advertised)
	}
}

func verifySHA1(filePath, expectedHex, displayed string) error {
	got, err := ComputeSHA1(filePath)
	if err != nil {
		return err
	}
	if got != expectedHex {
		os.Remove(filePath) // best-effort delete on mismatch (SEC-1)
		return &ErrChecksumMismatch{
			FilePath: filePath,
			Expected: displayed,
			Got:      "sha1:" + got,
		}
	}
	return nil
}
