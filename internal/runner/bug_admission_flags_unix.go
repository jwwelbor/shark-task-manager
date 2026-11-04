//go:build !windows

package runner

import "golang.org/x/sys/unix"

// Inventory and provenance files must not be replaced by symlinks.
const bugInventoryOpenFlags = unix.O_NONBLOCK | unix.O_NOFOLLOW
