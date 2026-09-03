package gaterun

import (
	"fmt"
	"io"
)

// ReadBoundedRegularFile reads path after verifying, via a no-follow open
// plus an fstat on the resulting descriptor, that it is a real (non-symlink)
// regular file, bounding the read at maxBytes+1 bytes so an oversized target
// is rejected rather than exhausting memory. It never blocks indefinitely on
// a FIFO or other special file (see openRegularNoFollowPath).
//
// This is the CLI-flag-supplied-absolute-path counterpart of
// readRegularBounded (fsio.go): that function opens a name relative to an
// already-open, no-follow-verified run-directory handle, which requires a
// directory descriptor this package's other callers don't have. Callers that
// only hold a bare path from a CLI flag (run_apply_result.go's
// --apply-result, impact.go's --impact-file) use this instead, sharing the
// exact same no-follow-open + fstat-regular-file-check safety property
// without duplicating it.
func ReadBoundedRegularFile(path string, maxBytes int) ([]byte, error) {
	f, err := openRegularNoFollowPath(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("gaterun: read %s: %w", path, err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("gaterun: %s exceeds the %d byte bound", path, maxBytes)
	}
	return data, nil
}
