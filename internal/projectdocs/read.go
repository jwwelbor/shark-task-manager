package projectdocs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxDocumentBytes is the existing maximum size for review evidence.
const MaxDocumentBytes int64 = 128 * 1024 * 1024

// ReadBounded reads a regular project-relative file without blocking on FIFOs.
func ReadBounded(rootPath, path string, limit int64) ([]byte, error) {
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("evidence path must be local")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }() // Read-only handle; close does not affect the verified bytes.
	file, err := root.OpenFile(path, os.O_RDONLY|readFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open evidence: %w", err)
	}
	defer func() { _ = file.Close() }() // Read-only handle; close does not affect the verified bytes.
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("evidence must be a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("cannot read bounded evidence")
	}
	return data, nil
}
