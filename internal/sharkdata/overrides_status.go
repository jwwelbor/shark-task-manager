package sharkdata

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Classification values for OverrideRow.Classification. These string
// spellings are part of REQ-NF-001's stability contract — an accidental
// rename fails TestOverrideRow_JSONFieldNames_Golden, not just at runtime.
const (
	ClassificationCurrent            = "current"
	ClassificationUpstreamChanged    = "upstream_changed"
	ClassificationIdenticalRedundant = "identical_redundant"
	ClassificationOrphaned           = "orphaned"
	ClassificationBaselineUnknown    = "baseline_unknown"
)

// OverrideRow describes the drift state of a single file under
// <dataRoot>/overrides/. JSON field names are part of REQ-NF-001's stability
// contract.
type OverrideRow struct {
	Path            string `json:"path"`
	Classification  string `json:"classification"`
	OverrideSHA256  string `json:"override_sha256"`
	CanonicalSHA256 string `json:"canonical_sha256"` // "" when orphaned
	BaselineSHA256  string `json:"baseline_sha256"`  // "" when baseline_unknown
	SuggestedAction string `json:"suggested_action"`
}

// OverrideStatusReport is the return value of OverrideStatusAt. JSON
// envelope keys ("overrides", "summary") are part of REQ-NF-001's stability
// contract.
type OverrideStatusReport struct {
	Rows    []OverrideRow  `json:"overrides"`
	Summary map[string]int `json:"summary"`
}

// newOverrideStatusReport returns a report with an empty (never nil) Rows
// slice and a Summary map carrying all five classification keys at zero, so
// JSON output is stable ({"overrides": [], "summary": {...all zero...}})
// even when overrides/ is empty or absent.
func newOverrideStatusReport() *OverrideStatusReport {
	return &OverrideStatusReport{
		Rows: []OverrideRow{},
		Summary: map[string]int{
			ClassificationCurrent:            0,
			ClassificationUpstreamChanged:    0,
			ClassificationIdenticalRedundant: 0,
			ClassificationOrphaned:           0,
			ClassificationBaselineUnknown:    0,
		},
	}
}

// normalizeOverrideRelPath rejects an absolute path or any path containing a
// ".." segment, returning the POSIX-normalized (forward-slash) relative path
// otherwise. This mirrors the relative-path normalization
// validatePromptIncludes already applies to prompt includes in embed.go, and
// must be called — and must reject unsafe input — before a walked override
// path is ever joined onto dataRoot or looked up via ReadEmbedded.
func normalizeOverrideRelPath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("sharkdata: override path must be relative: %q", relPath)
	}
	fsPath := filepath.ToSlash(relPath)
	for _, seg := range strings.Split(fsPath, "/") {
		if seg == ".." {
			return "", fmt.Errorf("sharkdata: override path must not contain %q segments: %q", "..", relPath)
		}
	}
	return fsPath, nil
}

// sanitizeWalkErrText extracts a path-free description of a filepath.WalkDir
// directory-entry error for use in a JSON-facing field (OverrideRow.
// SuggestedAction). WalkDir directory-read failures are reported as
// *fs.PathError wrapping the underlying syscall/errno error (e.g. "permission
// denied") — that inner error never carries a filesystem path, unlike the
// PathError's own Error() string, which embeds the absolute path it was
// operating on. The caller already knows the affected path in its
// already-relative form (relOS) and includes it separately, so only this
// path-free fragment is needed here. Per REQ-F-004, no other part of
// walkEntryErr is ever interpolated into a row field, since it could leak an
// absolute filesystem path. Any error that isn't a *fs.PathError falls back
// to a fixed, generic phrase rather than risk leaking path text from an
// unknown error shape.
func sanitizeWalkErrText(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) && pathErr.Err != nil {
		return pathErr.Err.Error()
	}
	return "unreadable path"
}

// OverrideStatusAt walks <dataRoot>/overrides/ and classifies every regular
// file's drift state relative to its canonical counterpart (see
// classifyOverride for the five-state decision table). A missing or empty
// overrides/ directory is not an error — it produces an empty report.
//
// Error tolerance: a per-file read failure and a per-directory walk failure
// (e.g. permission denied) both degrade to a single baseline_unknown row for
// the affected path and the walk continues over sibling paths — neither
// aborts the whole report. Only a failure to stat overrides/ itself is
// returned as an error.
//
// Read-only: no file under overrides/ or the embedded canonical tree is ever
// opened in write mode, and a symlink under overrides/ is never followed.
func OverrideStatusAt(dataRoot string) (*OverrideStatusReport, error) {
	report := newOverrideStatusReport()

	overridesDir := filepath.Join(dataRoot, "overrides")
	info, err := os.Stat(overridesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return report, nil
		}
		return nil, fmt.Errorf("failed to stat overrides directory %q: %w", overridesDir, err)
	}
	if !info.IsDir() {
		return report, nil
	}

	// A missing or invalid manifest must never fail the whole status call —
	// every affected path simply has no trustworthy baseline entry, which
	// classifyOverride already treats as baseline_unknown.
	manifest, manifestErr := LoadOverrideBaselines(dataRoot)
	if manifestErr != nil {
		manifest = &OverrideBaselineManifest{
			SchemaVersion: baselineManifestSchemaVersion,
			Baselines:     map[string]string{},
		}
	}

	walkErr := filepath.WalkDir(overridesDir, func(p string, d fs.DirEntry, walkEntryErr error) error {
		if walkEntryErr != nil {
			// A directory-entry-level walk error (e.g. permission denied
			// listing a subdirectory, or an Lstat failure) must degrade to a
			// per-entry baseline_unknown row exactly like a file-level read
			// failure below (Step 0/os.ReadFile branch), rather than
			// propagating the error out of WalkDir and discarding every row
			// already collected for sibling paths. d is nil only when the
			// root path itself could not be Lstat'd (a TOCTOU race after the
			// caller's own os.Stat above); guard against that before calling
			// d.IsDir().
			relOS, relErr := filepath.Rel(overridesDir, p)
			if relErr != nil {
				relOS = p
			}
			report.addRow(OverrideRow{
				Path:            filepath.ToSlash(relOS),
				Classification:  ClassificationBaselineUnknown,
				SuggestedAction: fmt.Sprintf("failed to walk path %q: %s", filepath.ToSlash(relOS), sanitizeWalkErrText(walkEntryErr)),
			})
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		relOS, relErr := filepath.Rel(overridesDir, p)
		if relErr != nil {
			return relErr
		}

		// Path-safety normalization must run before this path is joined
		// onto dataRoot again (for reading the override file) or used as a
		// lookup key against the embedded canonical tree.
		safePath, safeErr := normalizeOverrideRelPath(relOS)
		if safeErr != nil {
			report.addRow(OverrideRow{
				Path:            filepath.ToSlash(relOS),
				Classification:  ClassificationBaselineUnknown,
				SuggestedAction: fmt.Sprintf("unsafe override path rejected: %v", safeErr),
			})
			return nil
		}

		report.addRow(classifyOverride(overridesDir, safePath, d, manifest))
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("failed to walk overrides directory %q: %w", overridesDir, walkErr)
	}

	sort.Slice(report.Rows, func(i, j int) bool {
		return report.Rows[i].Path < report.Rows[j].Path
	})

	return report, nil
}

// addRow appends row to the report and increments its classification's
// summary count.
func (r *OverrideStatusReport) addRow(row OverrideRow) {
	r.Rows = append(r.Rows, row)
	r.Summary[row.Classification]++
}

// classifyOverride applies the five-state classification decision table to a
// single regular (or non-regular) file found under overrides/, following
// spec.md's step order:
//
//  0. symlink or any other non-regular file -> baseline_unknown, never opened.
//  1. no canonical counterpart -> orphaned.
//  2. override bytes equal canonical bytes -> identical_redundant, checked
//     before any baseline-entry lookup (bytes equality is definitive
//     regardless of baseline state — decision #4 in spec.md's "Key technical
//     decisions").
//  3. no trustworthy baseline entry for this path -> baseline_unknown.
//  4. baseline SHA-256 differs from current canonical SHA-256 ->
//     upstream_changed.
//  5. baseline SHA-256 equals current canonical SHA-256 -> current.
func classifyOverride(overridesDir, relPath string, d fs.DirEntry, manifest *OverrideBaselineManifest) OverrideRow {
	row := OverrideRow{Path: relPath}

	// Step 0: symlink (or any other non-regular file) — never read, and
	// classified before any counterpart check so a symlinked path is never
	// misclassified as orphaned when a counterpart does exist.
	if !d.Type().IsRegular() {
		row.Classification = ClassificationBaselineUnknown
		if d.Type()&fs.ModeSymlink != 0 {
			row.SuggestedAction = "replace symlink with a regular file"
		} else {
			row.SuggestedAction = "replace non-regular file with a regular file"
		}
		return row
	}

	overridePath := filepath.Join(overridesDir, filepath.FromSlash(relPath))
	overrideBytes, err := os.ReadFile(overridePath)
	if err != nil {
		row.Classification = ClassificationBaselineUnknown
		row.SuggestedAction = fmt.Sprintf("failed to read override file %q", relPath)
		return row
	}
	row.OverrideSHA256 = sha256Hex(overrideBytes)

	canonicalBytes, err := ReadEmbedded(relPath)
	if err != nil {
		// Step 1: no canonical counterpart -> orphaned.
		row.Classification = ClassificationOrphaned
		row.SuggestedAction = "no canonical counterpart exists; consider removing this override"
		return row
	}
	row.CanonicalSHA256 = sha256Hex(canonicalBytes)

	// Step 2 (spec.md step 3): bytes equality is definitive and checked
	// before any baseline-entry lookup.
	if row.OverrideSHA256 == row.CanonicalSHA256 {
		row.Classification = ClassificationIdenticalRedundant
		if baselineSHA, ok := manifest.Baselines[relPath]; ok {
			row.BaselineSHA256 = baselineSHA
		}
		row.SuggestedAction = "override is byte-identical to canonical; consider removing it"
		return row
	}

	baselineSHA, hasBaseline := manifest.Baselines[relPath]
	if !hasBaseline {
		// Step 3 (spec.md step 2): no trustworthy baseline entry.
		row.Classification = ClassificationBaselineUnknown
		row.SuggestedAction = "no recorded baseline for this override; review then run 'shark admin overrides acknowledge'"
		return row
	}
	row.BaselineSHA256 = baselineSHA

	if baselineSHA != row.CanonicalSHA256 {
		// Step 4: canonical has moved since the recorded baseline.
		row.Classification = ClassificationUpstreamChanged
		row.SuggestedAction = "review upstream canonical change before rebasing this override"
		return row
	}

	// Step 5: baseline matches current canonical (and bytes already
	// confirmed to differ from canonical, else step 2 above would have
	// matched).
	row.Classification = ClassificationCurrent
	return row
}

// sha256Hex returns the lowercase hex-encoded SHA-256 digest of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
