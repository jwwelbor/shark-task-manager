// Package runner — transcript file capture.
//
// This file implements the opt-in per-dispatch transcript writer used by the
// run controller when ObservabilityConfig.CaptureAgentTranscripts is true.
//
// The writer is intentionally side-effect free at package level: all filesystem
// work is scoped to a caller-supplied project root. This lets unit tests drive
// the writer against t.TempDir() without touching the real project.
//
// Exact on-disk format (bytes — no trailing newline):
//
//	COMMAND: <cmd>
//	EXIT: <code>
//	DURATION: <ms>ms
//	---STDOUT---
//	<stdout>
//	---STDERR---
//	<stderr>
//
// File path layout (relative to project root):
//
//	.shark/runs/<run_id>/<entity_key>/<stage_n>-<status>-<provider>.log
//
// The entity_key directory level exists because cascade children inherit
// their parent's run_id unchanged (RunID is a stable per-invocation
// correlation identifier across the whole cascade tree — see
// controller.go's RunOptions.RunID doc comment) while each child's own
// Run() independently restarts its stage_n counter at 1. Without per-entity
// nesting, two sibling cascade children dispatching in the same
// status/provider would collide on stage_n and silently overwrite each
// other's transcript.
//
// Permissions:
//   - Parent directory: 0o755
//   - File:              0o644
package runner

import (
	"fmt"
	"os"
	"path/filepath"
)

// transcriptDirMode is the permission bitmask applied when MkdirAll-ing the
// per-run directory ".shark/runs/<run_id>/".
const transcriptDirMode os.FileMode = 0o755

// transcriptFileMode is the permission bitmask applied when creating each
// transcript .log file.
const transcriptFileMode os.FileMode = 0o644

// relTranscriptPath returns the project-relative transcript file path for a
// single dispatch under the given run. Callers use this value as the
// transcript_path attribute on slog events so the filename is portable across
// project roots.
//
// entityKey nests each entity (parent and every cascade child) under its own
// subdirectory so sibling cascade children — which inherit the same runID
// but each restart their own stageN counter at 1 — can never collide on
// filename (see the package doc comment above for the full rationale).
//
// The caller is responsible for passing a non-empty runID, a non-empty
// entityKey, a positive stageN, and non-empty status/provider; this helper
// does not validate its inputs.
func relTranscriptPath(runID, entityKey string, stageN int, status, provider string) string {
	return filepath.Join(".shark", "runs", runID, entityKey, fmt.Sprintf("%d-%s-%s.log", stageN, status, provider))
}

// writeTranscript writes a single dispatch transcript to the filesystem under
// root and returns the project-relative path.
//
// Behavior:
//   - Creates the parent directory `<root>/.shark/runs/<runID>/` with
//     os.MkdirAll (mode 0o755) if it does not exist.
//   - Writes the content atomically via os.WriteFile with mode 0o644.
//   - The format string is part of the documented on-disk contract; any
//     drift will break consumers who parse the file offline.
//
// Arguments:
//   - root:       absolute path to the project root (from RunOptions.ProjectRoot).
//   - runID:      per-run identifier (directory name under .shark/runs/).
//   - entityKey:  the entity this dispatch belongs to (per-entity subdirectory
//     name, so sibling cascade children never collide — see relTranscriptPath).
//   - stageN:     1-based stage counter within the run (filename prefix).
//   - status:     the current (from) status that drove this dispatch.
//   - provider:   agent provider key (e.g. "anthropic", "codex").
//   - command:    the exact command line passed to the agent CLI. May be "".
//   - exitCode:   the process exit code (0 for success).
//   - durationMS: dispatch duration in whole milliseconds.
//   - stdout:     captured stdout (already truncated by the caller).
//   - stderr:     captured stderr (already truncated by the caller).
//
// Returns the project-relative path on success, or ("", error) if either
// MkdirAll or WriteFile fails. Callers treat errors as non-fatal — they log a
// single run.transcript.warning and disable further writes for the run.
func writeTranscript(
	root, runID, entityKey string,
	stageN int,
	status, provider, command string,
	exitCode int,
	durationMS int64,
	stdout, stderr string,
) (string, error) {
	rel := relTranscriptPath(runID, entityKey, stageN, status, provider)
	abs := filepath.Join(root, rel)

	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, transcriptDirMode); err != nil {
		return "", fmt.Errorf("create transcript dir %s: %w", dir, err)
	}

	// On-disk EXACT format. Do not add a trailing newline after <stderr>.
	content := fmt.Sprintf(
		"COMMAND: %s\nEXIT: %d\nDURATION: %dms\n---STDOUT---\n%s\n---STDERR---\n%s",
		command, exitCode, durationMS, stdout, stderr,
	)

	if err := os.WriteFile(abs, []byte(content), transcriptFileMode); err != nil {
		return "", fmt.Errorf("write transcript %s: %w", abs, err)
	}

	return rel, nil
}
