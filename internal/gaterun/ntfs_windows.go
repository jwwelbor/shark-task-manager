//go:build windows

package gaterun

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file provides the Windows counterparts of the fd-relative primitives
// fsio_nofollow_unix.go implements via openat/linkat/renameat/unlinkat/
// fstatat: every operation opens (or creates) its target via NtCreateFile
// relative to an already-open, verified parent directory handle (or, for
// the CLI-flag bare-path case, opens the leaf itself in one call), and
// checks the resulting handle's own attributes for a reparse point
// (symlink/junction/mount point) or wrong type — never a separate
// Lstat/GetFileAttributes-by-path call whose result is then discarded in
// favor of reopening the same name by string. That is what closes the
// check-then-path-use TOCTOU window this package's Windows fallback
// previously left open (see TD-187 / UAT-3-2): the safety verification and
// the real filesystem operation share one descriptor (or, where Windows has
// no handle-relative primitive at all — rename/link/delete — one
// RootDirectory-relative NtCreateFile/NtSetInformationFile call whose target
// name never gets re-resolved from a path string).
//
// FILE_OPEN_REPARSE_POINT is the Windows analogue of O_NOFOLLOW: it makes
// NtCreateFile open a reparse point (symlink, junction, mount point) itself
// rather than transparently traversing it, so the handle this function
// returns always refers to the exact filesystem object named — never
// whatever it points to.

// dirTraverseAccess is the access requested for ancestor directory
// components that are only ever descended into (never fsync'd or written
// through directly).
const dirTraverseAccess = uint32(windows.FILE_LIST_DIRECTORY | windows.FILE_READ_ATTRIBUTES | windows.FILE_TRAVERSE | windows.SYNCHRONIZE)

// dirFullAccess additionally requests write access, needed by the leaf run
// directory handle this package fsync's directly (fsyncDir) and uses as the
// RootDirectory for every create/link/rename/remove call.
const dirFullAccess = dirTraverseAccess | uint32(windows.FILE_GENERIC_WRITE)

const shareAll = uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE)

// fileRenameInformation mirrors the kernel's FILE_RENAME_INFORMATION /
// FILE_LINK_INFORMATION layout (they are identical): a flags word, an
// optional handle to the directory the new name is relative to, the name's
// byte length, and the inline UTF-16 name itself. golang.org/x/sys/windows
// exposes the NtSetInformationFile call and the FileRenameInformation /
// FileLinkInformation class constants but not this struct (its own test
// suite, TestNtCreateFileAndNtSetInformationFile, defines the identical
// layout locally for the same reason).
type fileRenameInformation struct {
	Flags          uint32
	RootDirectory  windows.Handle
	FileNameLength uint32
	FileName       [1]uint16
}

// ntCreateFileRel opens (or creates, depending on disposition) name
// relative to root via NtCreateFile, always with FILE_OPEN_REPARSE_POINT so
// a reparse point at name is opened rather than followed. root may be zero
// only when name is itself an absolute NT path; every caller in this
// package passes a real directory handle.
func ntCreateFileRel(root windows.Handle, name string, access, disposition, options uint32) (windows.Handle, uint32, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, 0, fmt.Errorf("gaterun: encode %q: %w", name, err)
	}
	oa := &windows.OBJECT_ATTRIBUTES{
		RootDirectory: root,
		ObjectName:    objectName,
		Attributes:    windows.OBJ_CASE_INSENSITIVE,
	}
	oa.Length = uint32(unsafe.Sizeof(*oa))

	var handle windows.Handle
	var iosb windows.IO_STATUS_BLOCK
	ntErr := windows.NtCreateFile(&handle, access, oa, &iosb, nil, windows.FILE_ATTRIBUTE_NORMAL,
		shareAll, disposition, options|windows.FILE_OPEN_REPARSE_POINT, 0, 0)
	if ntErr != nil {
		return 0, 0, ntErr
	}
	return handle, uint32(iosb.Information), nil
}

// classifyOpenErr maps the raw NTSTATUS from ntCreateFileRel to the
// sentinel errors this package's callers already check for with errors.Is.
func classifyOpenErr(err error, name string) error {
	if st, ok := err.(windows.NTStatus); ok {
		switch st {
		case windows.STATUS_OBJECT_NAME_NOT_FOUND, windows.STATUS_OBJECT_PATH_NOT_FOUND:
			return fmt.Errorf("gaterun: open %s: %w", name, os.ErrNotExist)
		case windows.STATUS_OBJECT_NAME_COLLISION:
			return fmt.Errorf("gaterun: open %s: %w", name, os.ErrExist)
		}
	}
	return fmt.Errorf("gaterun: open %s: %w", name, err)
}

func getFileAttributesByHandle(h windows.Handle) (windows.ByHandleFileInformation, error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return info, fmt.Errorf("gaterun: query file information: %w", err)
	}
	return info, nil
}

// openDirRelNoFollow opens name as a directory relative to root, creating
// it (FILE_OPEN_IF) when create is true and it does not yet exist, and
// verifies — via GetFileInformationByHandle on the just-opened handle, not
// a separate stat — that the result is a real (non-reparse-point)
// directory.
func openDirRelNoFollow(root windows.Handle, name string, create bool, full bool) (windows.Handle, error) {
	disposition := uint32(windows.FILE_OPEN)
	if create {
		disposition = windows.FILE_OPEN_IF
	}
	access := dirTraverseAccess
	if full {
		access = dirFullAccess
	}
	options := uint32(windows.FILE_DIRECTORY_FILE | windows.FILE_SYNCHRONOUS_IO_NONALERT)

	h, _, err := ntCreateFileRel(root, name, access, disposition, options)
	if err != nil {
		if st, ok := err.(windows.NTStatus); ok {
			switch st {
			case windows.STATUS_OBJECT_NAME_NOT_FOUND, windows.STATUS_OBJECT_PATH_NOT_FOUND:
				return 0, fmt.Errorf("gaterun: open %s: %w", name, os.ErrNotExist)
			case windows.STATUS_NOT_A_DIRECTORY, windows.STATUS_INVALID_PARAMETER:
				return 0, &UnsafePathError{Path: name, Reason: "refusing to follow symlink or non-directory ancestor"}
			}
		}
		return 0, fmt.Errorf("gaterun: open %s: %w", name, err)
	}

	info, ierr := getFileAttributesByHandle(h)
	if ierr != nil {
		_ = windows.CloseHandle(h)
		return 0, ierr
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = windows.CloseHandle(h)
		return 0, &UnsafePathError{Path: name, Reason: "refusing to follow symlink or non-directory ancestor"}
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		_ = windows.CloseHandle(h)
		return 0, &UnsafePathError{Path: name, Reason: "refusing to follow symlink or non-directory ancestor"}
	}
	return h, nil
}

// openFileRelNoFollow opens name as a non-directory file relative to root,
// verifying — via the resulting handle — that it is a real (non-reparse-
// point, non-directory) file.
func openFileRelNoFollow(root windows.Handle, name string, access uint32) (windows.Handle, error) {
	options := uint32(windows.FILE_NON_DIRECTORY_FILE | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	h, _, err := ntCreateFileRel(root, name, access, windows.FILE_OPEN, options)
	if err != nil {
		return 0, classifyOpenErr(err, name)
	}

	info, ierr := getFileAttributesByHandle(h)
	if ierr != nil {
		_ = windows.CloseHandle(h)
		return 0, ierr
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = windows.CloseHandle(h)
		return 0, &UnsafePathError{Path: name, Reason: "refusing to follow symlink"}
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		_ = windows.CloseHandle(h)
		return 0, &UnsafePathError{Path: name, Reason: "target is not a regular file"}
	}
	return h, nil
}

// createExclRel creates name relative to root, failing with os.ErrExist if
// it already exists (regardless of what it is) — the disposition check and
// the create share one NtCreateFile call, so there is no separate
// exists-check step to race.
func createExclRel(root windows.Handle, name string) (windows.Handle, error) {
	options := uint32(windows.FILE_NON_DIRECTORY_FILE | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	h, _, err := ntCreateFileRel(root, name, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE, windows.FILE_CREATE, options)
	if err != nil {
		return 0, classifyOpenErr(err, name)
	}
	return h, nil
}

// encodeRelativeName UTF-16-encodes name (without a NUL terminator) for use
// in a fileRenameInformation buffer.
func encodeRelativeName(name string) ([]uint16, error) {
	u16, err := windows.UTF16FromString(name)
	if err != nil {
		return nil, fmt.Errorf("gaterun: encode %q: %w", name, err)
	}
	return u16[:len(u16)-1], nil // drop the implicit NUL terminator
}

func buildRenameOrLinkBuffer(root windows.Handle, flags uint32, newname string) ([]byte, error) {
	nameU16, err := encodeRelativeName(newname)
	if err != nil {
		return nil, err
	}
	nameLen := len(nameU16) * 2
	var dummy fileRenameInformation
	bufSize := int(unsafe.Offsetof(dummy.FileName)) + nameLen
	buf := make([]byte, bufSize)
	info := (*fileRenameInformation)(unsafe.Pointer(&buf[0]))
	info.Flags = flags
	info.RootDirectory = root
	info.FileNameLength = uint32(nameLen)
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(&info.FileName[0])), len(nameU16))
	copy(dst, nameU16)
	return buf, nil
}

// renameRelNoFollow atomically renames oldname to newname, both relative to
// dh, replacing any existing target — the mutable-sidecar-replace case
// (operation-state.json).
func renameRelNoFollow(dh windows.Handle, oldname, newname string) error {
	h, err := openFileRelNoFollow(dh, oldname, windows.DELETE|windows.SYNCHRONIZE)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()

	buf, err := buildRenameOrLinkBuffer(dh, windows.FILE_RENAME_REPLACE_IF_EXISTS, newname)
	if err != nil {
		return err
	}
	var iosb windows.IO_STATUS_BLOCK
	if ntErr := windows.NtSetInformationFile(h, &iosb, &buf[0], uint32(len(buf)), windows.FileRenameInformation); ntErr != nil {
		return fmt.Errorf("gaterun: rename %s to %s: %w", oldname, newname, ntErr)
	}
	return nil
}

// linkRelNoFollow hard-links oldname to newname, both relative to dh,
// without replacing an existing target — CreateResult's first-writer-wins
// primitive. An existing newname reports os.ErrExist.
func linkRelNoFollow(dh windows.Handle, oldname, newname string) error {
	h, err := openFileRelNoFollow(dh, oldname, windows.FILE_GENERIC_READ)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()

	buf, err := buildRenameOrLinkBuffer(dh, 0, newname) // Flags=0: never replace an existing target.
	if err != nil {
		return err
	}
	var iosb windows.IO_STATUS_BLOCK
	if ntErr := windows.NtSetInformationFile(h, &iosb, &buf[0], uint32(len(buf)), windows.FileLinkInformation); ntErr != nil {
		if st, ok := ntErr.(windows.NTStatus); ok && st == windows.STATUS_OBJECT_NAME_COLLISION {
			return fmt.Errorf("gaterun: link %s: %w", newname, os.ErrExist)
		}
		return fmt.Errorf("gaterun: link %s to %s: %w", oldname, newname, ntErr)
	}
	return nil
}

// removeRelNoFollow deletes name relative to dh. Because the handle is
// opened with FILE_OPEN_REPARSE_POINT, a symlink at name is deleted itself
// (never its target) — the same guarantee unlinkat gives on Unix.
func removeRelNoFollow(dh windows.Handle, name string) error {
	options := uint32(windows.FILE_SYNCHRONOUS_IO_NONALERT)
	h, _, err := ntCreateFileRel(dh, name, windows.DELETE|windows.SYNCHRONIZE, windows.FILE_OPEN, options)
	if err != nil {
		return classifyOpenErr(err, name)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	type fileDispositionInformationEx struct {
		Flags uint32
	}
	disp := fileDispositionInformationEx{Flags: windows.FILE_DISPOSITION_DELETE | windows.FILE_DISPOSITION_POSIX_SEMANTICS}
	var iosb windows.IO_STATUS_BLOCK
	if ntErr := windows.NtSetInformationFile(h, &iosb, (*byte)(unsafe.Pointer(&disp)), uint32(unsafe.Sizeof(disp)), windows.FileDispositionInformationEx); ntErr != nil {
		return fmt.Errorf("gaterun: remove %s: %w", name, ntErr)
	}
	return nil
}

// statRelNoFollow reports, for name relative to dh, whether it exists and —
// if so — whether it is a reparse point (symlink/junction) or a regular
// file, via a single no-follow open plus a handle-based attribute query. It
// never opens the target for read and never issues a separate by-path stat.
func statRelNoFollow(dh windows.Handle, name string) (exists, isSymlink, isRegular bool, err error) {
	options := uint32(windows.FILE_SYNCHRONOUS_IO_NONALERT)
	h, _, ntErr := ntCreateFileRel(dh, name, windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE, windows.FILE_OPEN, options)
	if ntErr != nil {
		if st, ok := ntErr.(windows.NTStatus); ok && (st == windows.STATUS_OBJECT_NAME_NOT_FOUND || st == windows.STATUS_OBJECT_PATH_NOT_FOUND) {
			return false, false, false, nil
		}
		return false, false, false, fmt.Errorf("gaterun: stat %s: %w", name, ntErr)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	info, ierr := getFileAttributesByHandle(h)
	if ierr != nil {
		return false, false, false, ierr
	}
	isSymlink = info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
	isRegular = !isSymlink && info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0
	return true, isSymlink, isRegular, nil
}
