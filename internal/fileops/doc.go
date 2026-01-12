// Package fileops provides unified file operations for entity (epic, feature, task) file management.
//
// This package consolidates duplicate file writing logic across epic, feature, and task creation
// into a single, well-tested implementation. It provides:
//
//   - Atomic write protection (prevents race conditions)
//   - Consistent error handling across all entity types
//   - Flexible file path resolution (absolute or relative)
//   - Optional verbose logging
//   - Force overwrite capability
//   - Task-specific CreateIfMissing behavior
//
// # Basic Usage
//
// Create a file writer and write an entity file:
//
//	writer := fileops.NewEntityFileWriter()
//	result, err := writer.WriteEntityFile(fileops.WriteOptions{
//		Content:        []byte("# My Epic\n\nDescription..."),
//		ProjectRoot:    "/path/to/project",
//		FilePath:       "docs/plan/epic.md",
//		EntityType:     "epic",
//		UseAtomicWrite: true,
//		Verbose:        true,
//		Logger:         log.Println,
//	})
//
// # File Path Resolution
//
// File paths can be absolute or relative to ProjectRoot:
//
//   - Relative: "docs/plan/epic.md" → resolved to ProjectRoot/docs/plan/epic.md
//   - Absolute: "/custom/path/epic.md" → used as-is
//
// # Atomic Writes
//
// When UseAtomicWrite is true, files are created with the O_EXCL flag,
// preventing race conditions when multiple processes try to create the same file.
//
// # Task-Specific Behavior
//
// For EntityType="task", the CreateIfMissing option controls whether files
// must exist before writing. When false, an error is returned if the file
// doesn't exist (requires --create flag in CLI).
//
// # Force Overwrite
//
// When Force=true, existing files are overwritten instead of being linked.
// This is useful for updating file content.
//
// # Test Coverage
//
// This package has 87.1% test coverage with comprehensive positive and negative test cases:
//
//   - Positive cases: new files, existing files, atomic writes, directory creation
//   - Negative cases: permission denied, invalid paths, missing files, write failures
//
// See writer_test.go for detailed test scenarios.
package fileops
