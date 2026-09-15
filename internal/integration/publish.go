// Package integration — see run.go's package doc.
package integration

import (
	"os"
)

// atomicLinkPublish creates path as a hard link to the already-complete
// tempPath. It reports whether this caller won the create-if-absent race;
// callers retain ownership of their typed read-back and error contracts.
func atomicLinkPublish(tempPath, path string) (bool, error) {
	if err := os.Link(tempPath, path); err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// writeTempFileSynced writes data to path and synchronizes it before the
// caller publishes it. Failed writes leave no partial temporary file.
func writeTempFileSynced(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, runFileMode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		// The write error is authoritative; best-effort cleanup cannot make it recoverable.
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	if err := f.Sync(); err != nil {
		// The sync error is authoritative; best-effort cleanup cannot make it recoverable.
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		// Close already failed; removal is best-effort cleanup of an unusable temp file.
		_ = os.Remove(path)
		return err
	}
	return nil
}
