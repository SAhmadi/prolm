package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache provides HTTP response caching with ETag / Last-Modified support.
// Cache entries are stored as <key>.json (metadata) + <key>.body (response body)
// under the configured directory. All writes are atomic (SEC-8).
type Cache struct {
	dir string
}

// cacheEntry holds metadata for a cached HTTP response.
type cacheEntry struct {
	URL          string    `json:"url"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	StatusCode   int       `json:"status_code"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// NewCache creates a cache rooted at dir, creating the directory if needed (0755).
func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}
	return &Cache{dir: dir}, nil
}

// Get retrieves a cached entry for the given URL.
// Returns (nil, nil, nil) on cache miss.
func (c *Cache) Get(rawURL string) (*cacheEntry, []byte, error) {
	key := cacheKey(rawURL)
	metaPath := filepath.Join(c.dir, key+".json")
	bodyPath := filepath.Join(c.dir, key+".body")

	metaData, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reading cache metadata: %w", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(metaData, &entry); err != nil {
		// Corrupted cache entry — treat as miss, clean up.
		os.Remove(metaPath)
		os.Remove(bodyPath)
		return nil, nil, nil
	}

	body, err := os.ReadFile(bodyPath)
	if os.IsNotExist(err) {
		// Metadata without body — treat as miss, clean up.
		os.Remove(metaPath)
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reading cache body: %w", err)
	}

	return &entry, body, nil
}

// Put stores a response in the cache. Writes are atomic (SEC-8):
// data is written to a .tmp file first, then renamed into place.
func (c *Cache) Put(rawURL string, entry *cacheEntry, body []byte) error {
	key := cacheKey(rawURL)
	metaPath := filepath.Join(c.dir, key+".json")
	bodyPath := filepath.Join(c.dir, key+".body")

	metaData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshalling cache entry: %w", err)
	}

	if err := atomicWrite(bodyPath, body); err != nil {
		return fmt.Errorf("writing cache body: %w", err)
	}
	if err := atomicWrite(metaPath, metaData); err != nil {
		os.Remove(bodyPath) // best-effort cleanup
		return fmt.Errorf("writing cache metadata: %w", err)
	}
	return nil
}

// cacheKey returns a hex-encoded SHA-256 hash of the URL, used as filename.
func cacheKey(rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(h[:])
}

// atomicWrite writes data to path via a temporary file and rename (SEC-8).
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return err
	}
	return nil
}
