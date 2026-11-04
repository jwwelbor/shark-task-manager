//go:build !windows

package runner

import "golang.org/x/sys/unix"

// Nonblocking open prevents a malicious FIFO from hanging admission.
const bugPlanOpenFlags = unix.O_NONBLOCK

// Inventory and provenance files must not be replaced by symlinks.
const bugInventoryOpenFlags = unix.O_NONBLOCK | unix.O_NOFOLLOW
