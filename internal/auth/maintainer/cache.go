package maintainer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// sessionEntry is the JSON data model for the on-disk cache file.
// Unrecognized fields are ignored on read (forward-compatible).
//
// Spec reference: spec.md §2.3 (data model).
type sessionEntry struct {
	LastSuccess time.Time `json:"last_success"`
	PassHash    string    `json:"pass_hash"`
}

// cacheDir returns the directory that holds the per-project session file.
// The path is:
//
//	<cache-root>/shark/<project-hash>
//
// where <cache-root> is $XDG_CACHE_HOME if set and non-empty, otherwise
// os.UserCacheDir(). <project-hash> is a lowercase-hex SHA-256 of the
// absolute project root path.
//
// Spec reference: spec.md REQ-F-004, F02-D6.
func cacheDir(projectRoot string) (string, error) {
	root, err := cacheRoot()
	if err != nil {
		return "", fmt.Errorf("maintainer cache: resolve cache root: %w", err)
	}
	hash := projectHash(projectRoot)
	return filepath.Join(root, "shark", hash), nil
}

// cacheRoot returns the base cache directory: $XDG_CACHE_HOME if set and
// non-empty, otherwise os.UserCacheDir().
func cacheRoot() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("os.UserCacheDir: %w", err)
	}
	return dir, nil
}

// projectHash returns a stable lowercase-hex SHA-256 digest of the absolute
// project root path. This guarantees cache isolation between projects even
// when they share the same base name.
//
// Spec reference: spec.md F02-D6.
func projectHash(projectRoot string) string {
	sum := sha256.Sum256([]byte(projectRoot))
	return fmt.Sprintf("%x", sum)
}

// sessionPath returns the full path to the session file for the given project root.
func sessionPath(projectRoot string) (string, error) {
	dir, err := cacheDir(projectRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "maintainer.session"), nil
}

// readSession reads and parses the session file at path. If the file does not
// exist or is malformed (invalid JSON, wrong types, truncated), it returns nil
// without an error — treating the file as a cache miss per REQ-NF-003.
func readSession(path string) *sessionEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read — treat as cache miss.
		return nil
	}
	var entry sessionEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Malformed JSON — treat as cache miss (REQ-NF-003).
		return nil
	}
	return &entry
}

// writeSession atomically writes the session entry to path.
// It creates parent directories (mode 0700) if needed, then writes to a temp
// file and renames onto the final path so that concurrent readers never observe
// a partially-written file.
//
// Spec reference: spec.md F02-D2, REQ-NF-003, REQ-F-003.
func writeSession(path string, entry *sessionEntry) error {
	dir := filepath.Dir(path)
	// Create the per-project directory with mode 0700 (owner read/write/exec only).
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("maintainer cache: create directory: %w", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("maintainer cache: marshal session: %w", err)
	}

	// Write to a temp file in the same directory so os.Rename is always
	// same-filesystem (POSIX rename is atomic within the same directory).
	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	// Create the file with mode 0600 (owner read/write only).
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("maintainer cache: create temp file: %w", err)
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("maintainer cache: write temp file: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("maintainer cache: close temp file: %w", closeErr)
	}

	// Atomic rename to final path.
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("maintainer cache: rename temp file: %w", err)
	}
	return nil
}
