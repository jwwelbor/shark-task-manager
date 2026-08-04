package sharkdata

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SyncSharkAttackTree copies the canonical embedded-source tree to the
// authored mirror and removes authored-only files. Callers must pass the
// Shark Attack subtree roots, in that direction: embedded source first,
// authored destination second. `make sync-shark-attack-skill` supplies the
// repository's fixed roots for the supported repair operation.
func SyncSharkAttackTree(embeddedSource, authoredDestination string) error {
	canonicalFiles, err := copyCanonicalFiles(embeddedSource, authoredDestination)
	if err != nil {
		return err
	}

	if err := removeAuthoredOnlyFiles(authoredDestination, canonicalFiles); err != nil {
		return err
	}
	return nil
}

func copyCanonicalFiles(source, destination string) (map[string]struct{}, error) {
	canonicalFiles := make(map[string]struct{})
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("canonical source contains unsupported symlink %q", path)
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("derive canonical relative path for %q: %w", path, err)
		}
		canonicalFiles[rel] = struct{}{}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read canonical file %q: %w", path, err)
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read canonical mode for %q: %w", path, err)
		}
		mode := info.Mode().Perm()
		target := filepath.Join(destination, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create authored directory for %q: %w", target, err)
		}
		if err := writeFileAtomically(target, contents, mode); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copy canonical Shark Attack tree: %w", err)
	}
	return canonicalFiles, nil
}

func removeAuthoredOnlyFiles(destination string, canonicalFiles map[string]struct{}) error {
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(destination, path)
		if err != nil {
			return fmt.Errorf("derive authored relative path for %q: %w", path, err)
		}
		if _, exists := canonicalFiles[rel]; exists {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove authored-only file %q: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove authored-only Shark Attack files: %w", err)
	}
	return nil
}

func writeFileAtomically(path string, contents []byte, mode fs.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".shark-attack-sync-*")
	if err != nil {
		return fmt.Errorf("create temporary authored file for %q: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(mode); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return fmt.Errorf("close temporary authored file after mode failure for %q: %w", path, closeErr)
		}
		return fmt.Errorf("set mode on temporary authored file for %q: %w", path, err)
	}
	if _, err := temporary.Write(contents); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return fmt.Errorf("close temporary authored file after write failure for %q: %w", path, closeErr)
		}
		return fmt.Errorf("write temporary authored file for %q: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary authored file for %q: %w", path, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace authored file %q: %w", path, err)
	}
	return nil
}
