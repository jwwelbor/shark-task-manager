//go:build windows

package fileops

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows"
)

func renameNoReplace(from, to string) error {
	supportsLongPaths := windowsSupportsLongPaths()
	normalizedFrom, err := normalizeWindowsPath(from, supportsLongPaths)
	if err != nil {
		return err
	}
	normalizedTo, err := normalizeWindowsPath(to, supportsLongPaths)
	if err != nil {
		return err
	}
	src, err := windows.UTF16PtrFromString(normalizedFrom)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(normalizedTo)
	if err != nil {
		return err
	}
	err = windows.MoveFileEx(src, dst, 0)
	if errors.Is(err, windows.ERROR_NOT_SUPPORTED) || errors.Is(err, windows.ERROR_CALL_NOT_IMPLEMENTED) {
		return ErrRenameUnsupported
	}
	return err
}

const windowsLegacyPathLimit = 248

func windowsSupportsLongPaths() bool {
	version := windows.RtlGetVersion()
	return version.MajorVersion > 10 ||
		(version.MajorVersion == 10 && (version.MinorVersion > 0 ||
			(version.MinorVersion == 0 && version.BuildNumber >= 15063)))
}

// normalizeWindowsPath follows Go's long-path behavior before calling a
// Windows API directly. Go enables native long-path handling on Windows 10
// build 15063 and later; older systems need the extended-length prefix that
// Go's os operations add before calling file APIs.
func normalizeWindowsPath(path string, supportsLongPaths bool) (string, error) {
	if supportsLongPaths || hasWindowsExtendedPrefix(path) || hasWindowsDevicePrefix(path) {
		return path, nil
	}

	absPath, err := windows.FullPath(path)
	if err != nil {
		return "", err
	}
	if len(absPath) < windowsLegacyPathLimit {
		return path, nil
	}
	if hasWindowsUNCPathPrefix(absPath) {
		return `\\?\UNC\` + strings.TrimLeft(absPath, `\/`), nil
	}
	return `\\?\` + absPath, nil
}

func hasWindowsExtendedPrefix(path string) bool {
	if len(path) < 4 {
		return false
	}
	return (isWindowsPathSeparator(path[0]) && isWindowsPathSeparator(path[1]) && path[2] == '?' && isWindowsPathSeparator(path[3])) ||
		(isWindowsPathSeparator(path[0]) && path[1] == '?' && path[2] == '?' && isWindowsPathSeparator(path[3]))
}

func hasWindowsDevicePrefix(path string) bool {
	return len(path) >= 4 && isWindowsPathSeparator(path[0]) && isWindowsPathSeparator(path[1]) && path[2] == '.' && isWindowsPathSeparator(path[3])
}

func hasWindowsUNCPathPrefix(path string) bool {
	return len(path) >= 2 && isWindowsPathSeparator(path[0]) && isWindowsPathSeparator(path[1])
}

func isWindowsPathSeparator(char byte) bool {
	return char == '\\' || char == '/'
}
