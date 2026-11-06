//go:build windows

package fileops

import (
	"errors"

	"golang.org/x/sys/windows"
)

func renameNoReplace(from, to string) error {
	src, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	err = windows.MoveFileEx(src, dst, 0)
	if errors.Is(err, windows.ERROR_NOT_SUPPORTED) || errors.Is(err, windows.ERROR_CALL_NOT_IMPLEMENTED) {
		return ErrRenameUnsupported
	}
	return err
}
