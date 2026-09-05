package sharkdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// baselineManifestSchemaVersion is the only schema_version this binary
// recognizes. A manifest with any other value is treated as invalid so that
// callers classify affected paths as baseline_unknown rather than silently
// upgrading or misreading an incompatible schema.
const baselineManifestSchemaVersion = 1

// baselineManifestFileName is the name of the baseline manifest file stored
// directly under the resolved data root (not under overrides/).
const baselineManifestFileName = ".shark-override-baselines.json"

// ErrInvalidBaselineManifest is returned by LoadOverrideBaselines when the
// manifest file exists but cannot be trusted: corrupt JSON, or a
// schema_version this binary does not recognize. Callers must treat every
// path as baseline_unknown rather than surfacing this as a hard failure.
var ErrInvalidBaselineManifest = errors.New("invalid override baseline manifest")

// OverrideBaselineManifest records, for each overridden canonical file path,
// the SHA-256 digest of the canonical file at the time the override was last
// acknowledged. It never contains override bytes — only paths and digests.
type OverrideBaselineManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Baselines     map[string]string `json:"baselines"`
}

// LoadOverrideBaselines reads the baseline manifest at
// <dataRoot>/.shark-override-baselines.json. A missing file returns an empty,
// valid manifest (SchemaVersion: 1, Baselines: {}) and a nil error, so status
// still functions on a project with zero acknowledgements. A file that exists
// but fails to parse, or whose schema_version is not exactly 1, returns
// ErrInvalidBaselineManifest.
func LoadOverrideBaselines(dataRoot string) (*OverrideBaselineManifest, error) {
	path := filepath.Join(dataRoot, baselineManifestFileName)

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &OverrideBaselineManifest{
			SchemaVersion: baselineManifestSchemaVersion,
			Baselines:     map[string]string{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read override baseline manifest %q: %w", path, err)
	}

	var manifest OverrideBaselineManifest
	if jsonErr := json.Unmarshal(raw, &manifest); jsonErr != nil {
		return nil, fmt.Errorf("%w: %q: %v", ErrInvalidBaselineManifest, path, jsonErr)
	}
	if manifest.SchemaVersion != baselineManifestSchemaVersion {
		return nil, fmt.Errorf("%w: %q: unrecognized schema_version %d", ErrInvalidBaselineManifest, path, manifest.SchemaVersion)
	}
	if manifest.Baselines == nil {
		manifest.Baselines = map[string]string{}
	}

	return &manifest, nil
}

// Save writes the manifest to <dataRoot>/.shark-override-baselines.json.
// encoding/json sorts map keys on marshal, so the file is diff-stable in
// version control.
func (m *OverrideBaselineManifest) Save(dataRoot string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal override baseline manifest: %w", err)
	}

	path := filepath.Join(dataRoot, baselineManifestFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write override baseline manifest %q: %w", path, err)
	}

	return nil
}

// AcknowledgeOverrides records a fresh baseline digest for each of paths,
// per spec.md REQ-F-002 (acknowledge): reject-before-write. Every path is
// validated — a regular override file must exist at
// <dataRoot>/overrides/<path> and a canonical counterpart must exist via
// ReadEmbedded — before anything is written. A single invalid path aborts
// the whole call with zero manifest mutation (no partial writes). On
// success, the manifest is loaded, every path's baseline is set to the
// current canonical SHA-256, the manifest is saved exactly once, and the
// refreshed OverrideStatusAt report is returned so callers can observe the
// acknowledged paths reclassified as current. Acknowledge never opens an
// override file in write mode and never touches override bytes.
func AcknowledgeOverrides(dataRoot string, paths []string) (*OverrideStatusReport, error) {
	canonicalHashes := make(map[string]string, len(paths))

	for _, p := range paths {
		// Path-safety normalization must run before this caller-controlled
		// path is joined onto dataRoot or looked up anywhere — same
		// invariant OverrideStatusAt enforces for walked paths (see
		// normalizeOverrideRelPath in overrides_status.go). Validating first
		// avoids both a path-existence oracle (Lstat on a raw, unvalidated
		// path) and storing an un-normalized key in the manifest.
		safePath, safeErr := normalizeOverrideRelPath(p)
		if safeErr != nil {
			return nil, fmt.Errorf("sharkdata: acknowledge failed for %q: %w", p, safeErr)
		}

		overridePath := filepath.Join(dataRoot, "overrides", filepath.FromSlash(safePath))
		info, statErr := os.Lstat(overridePath)
		if statErr != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("sharkdata: acknowledge failed for %q: no regular override file exists at %q", safePath, overridePath)
		}

		canonicalBytes, readErr := ReadEmbedded(safePath)
		if readErr != nil {
			return nil, fmt.Errorf("sharkdata: acknowledge failed for %q: no canonical counterpart exists: %w", safePath, readErr)
		}
		canonicalHashes[safePath] = sha256Hex(canonicalBytes)
	}

	manifest, loadErr := LoadOverrideBaselines(dataRoot)
	if loadErr != nil {
		return nil, fmt.Errorf("sharkdata: acknowledge aborted: %w", loadErr)
	}
	if manifest.Baselines == nil {
		manifest.Baselines = map[string]string{}
	}
	for p, hash := range canonicalHashes {
		manifest.Baselines[p] = hash
	}

	if err := manifest.Save(dataRoot); err != nil {
		return nil, fmt.Errorf("failed to save updated baseline manifest: %w", err)
	}

	return OverrideStatusAt(dataRoot)
}
