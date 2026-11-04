//go:build !windows

package runner

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInventoryRejectsSwappedFileWithoutFollowingOrBlocking(t *testing.T) {
	for _, replacement := range []string{"symlink", "fifo"} {
		t.Run(replacement, func(t *testing.T) {
			rootPath := t.TempDir()
			name := filepath.Join(rootPath, "entry")
			require.NoError(t, os.WriteFile(name, []byte("original"), 0600))
			root, err := os.OpenRoot(rootPath)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, root.Close()) })
			before, err := root.Lstat("entry")
			require.NoError(t, err)
			require.NoError(t, os.Remove(name))
			if replacement == "fifo" {
				require.NoError(t, syscall.Mkfifo(name, 0600))
			} else {
				outside := filepath.Join(t.TempDir(), "outside")
				require.NoError(t, os.WriteFile(outside, []byte("secret"), 0600))
				require.NoError(t, os.Symlink(outside, name))
			}
			done := make(chan error, 1)
			go func() {
				f, err := bugOpenVerifiedInventoryFile(root, "entry", before)
				if f != nil {
					err = errors.Join(err, f.Close())
				}
				done <- err
			}()
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				require.FailNow(t, "inventory open blocked on swapped file")
			}
		})
	}
}
