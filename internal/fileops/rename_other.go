//go:build !linux && !darwin && !windows

package fileops

func renameNoReplace(_, _ string) error { return ErrRenameUnsupported }
