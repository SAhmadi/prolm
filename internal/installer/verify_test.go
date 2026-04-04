package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestComputeChecksum_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello, prolm!")
	path := writeFile(t, dir, "test.tar.gz", content)

	got, err := ComputeChecksum(path)
	require.NoError(t, err)
	assert.Equal(t, sha256Hex(content), got)
}

func TestComputeChecksum_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "empty.tar.gz", []byte{})

	got, err := ComputeChecksum(path)
	require.NoError(t, err)

	// SHA-256 of empty input is well-known.
	assert.Equal(t, sha256Hex([]byte{}), got)
}

func TestComputeChecksum_NonexistentFile(t *testing.T) {
	_, err := ComputeChecksum("/nonexistent/file.tar.gz")
	assert.Error(t, err)
}

func TestVerify_Match(t *testing.T) {
	dir := t.TempDir()
	content := []byte("verified content")
	path := writeFile(t, dir, "pack.tar.gz", content)

	err := Verify(path, sha256Hex(content))
	assert.NoError(t, err)

	// File still exists after successful verification.
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr)
}

func TestVerify_Mismatch_DeletesFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("actual content")
	path := writeFile(t, dir, "pack.tar.gz", content)

	wrongChecksum := sha256Hex([]byte("different content"))
	err := Verify(path, wrongChecksum)

	var mismatch *ErrChecksumMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, path, mismatch.FilePath)
	assert.Equal(t, wrongChecksum, mismatch.Expected)
	assert.Equal(t, sha256Hex(content), mismatch.Got)

	// File must be deleted on mismatch.
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestVerify_TruncatedFile(t *testing.T) {
	dir := t.TempDir()
	full := []byte("this is the full content of the tarball")
	truncated := full[:10]
	path := writeFile(t, dir, "trunc.tar.gz", truncated)

	err := Verify(path, sha256Hex(full))
	var mismatch *ErrChecksumMismatch
	assert.ErrorAs(t, err, &mismatch)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestVerify_BitFlippedContent(t *testing.T) {
	dir := t.TempDir()
	original := []byte("original bytes here")
	flipped := make([]byte, len(original))
	copy(flipped, original)
	flipped[5] ^= 0x01 // flip one bit

	path := writeFile(t, dir, "flipped.tar.gz", flipped)

	err := Verify(path, sha256Hex(original))
	var mismatch *ErrChecksumMismatch
	assert.ErrorAs(t, err, &mismatch)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestVerify_InvalidChecksumFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "pack.tar.gz", []byte("data"))

	tests := []struct {
		name     string
		checksum string
	}{
		{"missing prefix", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"},
		{"wrong prefix", "sha1:abcdef1234567890abcdef1234567890abcdef12"},
		{"too short", "sha256:abcdef"},
		{"too long", "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890aa"},
		{"empty string", ""},
		{"uppercase hex", "sha256:ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify(path, tc.checksum)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid checksum format")
		})
	}
}

func TestVerify_EmptyFileCorrectChecksum(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "empty.tar.gz", []byte{})

	err := Verify(path, sha256Hex([]byte{}))
	assert.NoError(t, err)
}
