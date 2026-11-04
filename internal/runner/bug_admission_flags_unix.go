//go:build !windows

package runner

import "golang.org/x/sys/unix"

// Nonblocking open prevents a malicious FIFO from hanging admission.
const bugPlanOpenFlags = unix.O_NONBLOCK
