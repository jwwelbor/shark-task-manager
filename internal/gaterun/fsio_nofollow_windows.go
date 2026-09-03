//go:build windows

package gaterun

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// This file is the Windows counterpart of the fd-relative primitives
// fsio_nofollow_unix.go implements via openat/linkat/renameat/unlinkat/
// fstatat. Every operation here delegates to ntfs_windows.go, which opens
// (or creates) its target relative to dh via NtCreateFile with
// FILE_OPEN_REPARSE_POINT and checks the resulting handle's own attributes
// — never a separate Lstat/GetFileAttributes-by-path call whose result is
// discarded in favor of reopening the same name by string. That is what
// closes the check-then-path-use TOCTOU window this package's previous
// Windows fallback (a plain Lstat, then a fresh path-joined os.Open/
// os.OpenFile/os.Link/os.Rename/os.Remove) left open (TD-187 / UAT-3-2).

func openRegularNoFollowAt(dh *os.File, name string) (*os.File, error) {
	h, err := openFileRelNoFollow(windows.Handle(dh.Fd()), name, windows.FILE_GENERIC_READ)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), name), nil
}

// openRegularNoFollowPath is the absolute/bare-path counterpart of
// openRegularNoFollowAt, for callers that only have a plain path (e.g. a
// CLI-flag-supplied file) rather than an already-open directory handle to
// open relative to — the same role fsio_nofollow_unix.go's
// openRegularNoFollowPath plays there. It opens path in a single
// CreateFile call with FILE_FLAG_OPEN_REPARSE_POINT (the Win32-layer
// equivalent of O_NOFOLLOW) and verifies the resulting handle's own
// attributes, so the final path component is never separately stat'd and
// then reopened by name. Like its Unix counterpart, this protects only the
// final path component — ancestor directories in an arbitrary CLI-supplied
// path are trusted the same way on both platforms; this is an unchanged,
// pre-existing scope boundary, not a new gap.
func openRegularNoFollowPath(path string) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("gaterun: encode %s: %w", path, err)
	}
	h, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		uint32(windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE),
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND || err == windows.ERROR_PATH_NOT_FOUND { //nolint:errorlint // syscall.Errno comparison by value is the documented pattern for these constants.
			return nil, fmt.Errorf("gaterun: open %s: %w", path, os.ErrNotExist)
		}
		return nil, fmt.Errorf("gaterun: open %s: %w", path, err)
	}
	f := os.NewFile(uintptr(h), path)

	info, ierr := getFileAttributesByHandle(h)
	if ierr != nil {
		_ = f.Close()
		return nil, ierr
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = f.Close()
		return nil, &UnsafePathError{Path: path, Reason: "refusing to follow symlink"}
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		_ = f.Close()
		return nil, &UnsafePathError{Path: path, Reason: "target is not a regular file"}
	}
	return f, nil
}

func createExclAt(dh *os.File, name string, _ os.FileMode) (*os.File, error) {
	h, err := createExclRel(windows.Handle(dh.Fd()), name)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), name), nil
}

func linkAt(dh *os.File, oldname, newname string) error {
	return linkRelNoFollow(windows.Handle(dh.Fd()), oldname, newname)
}

func renameAt(dh *os.File, oldname, newname string) error {
	return renameRelNoFollow(windows.Handle(dh.Fd()), oldname, newname)
}

func removeAt(dh *os.File, name string) error {
	return removeRelNoFollow(windows.Handle(dh.Fd()), name)
}

func existingTargetKindAt(dh *os.File, name string) (exists, isSymlink, isRegular bool, err error) {
	return statRelNoFollow(windows.Handle(dh.Fd()), name)
}
