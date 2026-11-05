//go:build !windows

package projectdocs

import "golang.org/x/sys/unix"

const readFlags = unix.O_NONBLOCK
