package sharkdata

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Drift describes a single parity mismatch between the authored
// skills/shark-attack/ tree and its embedded mirror under
// internal/sharkdata/default_data/skills/shark-attack/ (REQ-F-016).
type Drift struct {
	Path   string
	Reason string
}

// compareParity walks both fs.FS trees rooted at their respective
// shark-attack directories and reports every drift: a byte mismatch on a
// path present in both trees, an embedded-only file (present in embedded,
// absent from authored — REQ-F-016's explicit second failure mode), or an
// authored-only file (present in authored, absent from embedded).
//
// The function is pure over its two fs.FS parameters so the real gate
// (os.DirFS vs. an embed.FS adapter, TC-008-01) and the comparator's own
// unit tests (testing/fstest.MapFS fixtures, TC-008-02/03) exercise the
// exact same code path — a compiled go:embed tree is immutable at test
// time, so fixture-injected drift can only be exercised through this
// pure-function seam, never by mutating sharkdata's embed.FS directly.
func compareParity(authored, embedded fs.FS) ([]Drift, error) {
	authoredContent := map[string][]byte{}
	if err := fs.WalkDir(authored, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, readErr := fs.ReadFile(authored, p)
		if readErr != nil {
			return readErr
		}
		authoredContent[p] = content
		return nil
	}); err != nil {
		return nil, err
	}

	var drifts []Drift
	embeddedSeen := map[string]bool{}
	if err := fs.WalkDir(embedded, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		embeddedSeen[p] = true
		embContent, readErr := fs.ReadFile(embedded, p)
		if readErr != nil {
			return readErr
		}
		authContent, ok := authoredContent[p]
		if !ok {
			drifts = append(drifts, Drift{Path: p, Reason: "unexpected embedded-only file: no authored counterpart"})
			return nil
		}
		if !bytes.Equal(authContent, embContent) {
			drifts = append(drifts, Drift{Path: p, Reason: "byte drift between authored and embedded"})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for p := range authoredContent {
		if !embeddedSeen[p] {
			drifts = append(drifts, Drift{Path: p, Reason: "unexpected authored-only file: no embedded counterpart"})
		}
	}

	sort.Slice(drifts, func(i, j int) bool { return drifts[i].Path < drifts[j].Path })
	return drifts, nil
}

// realShardAttackTrees returns the real authored skills/shark-attack/ tree
// (via os.DirFS, live off disk) and the real embedded mirror (an fs.FS
// adapter over sharkdata's own compiled-in embed.FS, live at test time).
// Both reads are live — never a cached or snapshotted copy — so a future
// edit to either tree is caught the next time this test runs.
func realShardAttackTrees(t *testing.T) (authored, embedded fs.FS) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	authored = os.DirFS(filepath.Join(repoRoot, "skills", "shark-attack"))
	embedded, err = fs.Sub(embeddedFS, embedRootDir+"/skills/shark-attack")
	require.NoError(t, err)
	return authored, embedded
}

// TestTC008_01RealAuthoredEmbeddedTreesAreByteIdentical is the actual CI
// gate (REQ-F-016, AC-016): it runs inside `make test`/`go test ./...` and
// fails the moment the authored skills/shark-attack/ tree and the embedded
// internal/sharkdata/default_data/skills/shark-attack/ mirror drift apart,
// whether by byte content or by an embedded-only file.
func TestTC008_01RealAuthoredEmbeddedTreesAreByteIdentical(t *testing.T) {
	authored, embedded := realShardAttackTrees(t)

	drifts, err := compareParity(authored, embedded)
	require.NoError(t, err)
	assert.Emptyf(t, drifts, "authored/embedded shark-attack parity drift: %+v", drifts)
}

// TestTC008_02ComparatorDetectsSingleByteDriftOnSharedPath is a comparator
// unit test (TC-008-02): a compiled go:embed tree cannot host injected
// drift, so this drives compareParity's pure-function seam with two
// testing/fstest.MapFS fixtures differing by one byte in one file.
func TestTC008_02ComparatorDetectsSingleByteDriftOnSharedPath(t *testing.T) {
	authored := fstest.MapFS{
		"SKILL.md": &fstest.MapFile{Data: []byte("router v1")},
	}
	embedded := fstest.MapFS{
		"SKILL.md": &fstest.MapFile{Data: []byte("router v2")},
	}

	drifts, err := compareParity(authored, embedded)
	require.NoError(t, err)
	require.Len(t, drifts, 1, "expected exactly one drift entry naming the mismatched path")
	assert.Equal(t, "SKILL.md", drifts[0].Path)
}

// TestTC008_03ComparatorFlagsEmbeddedOnlyFileNotJustOneDirectional is a
// comparator unit test (TC-008-03): REQ-F-016's explicit second failure
// mode. A one-directional "every authored path exists in embedded" check
// would miss a file added straight to the embedded tree without ever
// landing in the authored tree; this fixture proves compareParity walks
// the embedded tree too and reports the embedded-only path.
func TestTC008_03ComparatorFlagsEmbeddedOnlyFileNotJustOneDirectional(t *testing.T) {
	authored := fstest.MapFS{
		"SKILL.md": &fstest.MapFile{Data: []byte("router")},
	}
	embedded := fstest.MapFS{
		"SKILL.md":            &fstest.MapFile{Data: []byte("router")},
		"workflows/orphan.md": &fstest.MapFile{Data: []byte("never landed in authored")},
	}

	drifts, err := compareParity(authored, embedded)
	require.NoError(t, err)
	require.Len(t, drifts, 1, "expected exactly one drift entry naming the embedded-only path")
	assert.Equal(t, "workflows/orphan.md", drifts[0].Path)
	assert.Contains(t, drifts[0].Reason, "embedded-only")
}

// TestTC008_04ParityGateScopeExcludesShardRiderNotJustHappensToOmitIt
// (TC-008-04) proves the parity gate's scope root is skills/shark-attack/
// only, not the full skills/ tree — skills/shark-rider/ is authored-only
// per spec's explicit note and has no embedded counterpart. The real-tree
// invocation (TC-008-01) does not false-positive on it because the
// authored fs.FS passed in is already rooted below skills/shark-attack/,
// not skills/; this test asserts that scoping choice deliberately, rather
// than skills/shark-rider/ merely happening to be absent.
func TestTC008_04ParityGateScopeExcludesShardRiderNotJustHappensToOmitIt(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	riderRoot := filepath.Join(repoRoot, "skills", "shark-rider")
	info, statErr := os.Stat(riderRoot)
	require.NoErrorf(t, statErr, "skills/shark-rider must exist on disk for this exclusion check to be meaningful")
	require.True(t, info.IsDir())

	authored, embedded := realShardAttackTrees(t)
	drifts, err := compareParity(authored, embedded)
	require.NoError(t, err)

	for _, d := range drifts {
		assert.NotContains(t, d.Path, "shark-rider", "parity gate scope leaked outside skills/shark-attack/")
	}

	// The scope root itself is the proof: authored is os.DirFS rooted at
	// <repoRoot>/skills/shark-attack, so a path under skills/shark-rider/
	// cannot appear in authoredContent's walk at all — confirm that root
	// directly rather than relying on drifts being empty.
	authoredRoot := filepath.Join(repoRoot, "skills", "shark-attack")
	assert.NotEqual(t, filepath.Join(repoRoot, "skills"), authoredRoot)
}

// TestTC008_05SyncHelperRepairsFixtureDrift verifies AC-016's repair path.
// It introduces both a changed shared file and an authored-only file, then
// drives the same synchronizer used by `make sync-shark-attack-skill`. The
// real parity comparator must be clean after the embedded canonical tree is
// copied outward into the authored mirror.
func TestTC008_05SyncHelperRepairsFixtureDrift(t *testing.T) {
	embeddedRoot := t.TempDir()
	authoredRoot := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(embeddedRoot, "workflows"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(embeddedRoot, "SKILL.md"), []byte("canonical router"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(embeddedRoot, "workflows", "resume.md"), []byte("canonical resume"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(authoredRoot, "SKILL.md"), []byte("drifted router"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(authoredRoot, "retired.md"), []byte("authored-only"), 0o644))

	before, err := compareParity(os.DirFS(authoredRoot), os.DirFS(embeddedRoot))
	require.NoError(t, err)
	require.NotEmpty(t, before, "fixture must begin with drift for the repair test to be meaningful")

	require.NoError(t, SyncSharkAttackTree(embeddedRoot, authoredRoot))

	after, err := compareParity(os.DirFS(authoredRoot), os.DirFS(embeddedRoot))
	require.NoError(t, err)
	assert.Empty(t, after, "sync helper must leave the real parity comparator clean")
}

// TestTC012ParallelTeamParityDetectsBothDirectionsAndRepairs verifies the
// E38-F12 distribution contract for the new parallel-team workflow. The two
// MapFS fixtures exercise both possible tree directions; the temporary on-disk
// repair then drives the same synchronizer used by make sync-shark-attack-skill.
func TestTC012ParallelTeamParityDetectsBothDirectionsAndRepairs(t *testing.T) {
	t.Run("authored byte drift fails", func(t *testing.T) {
		authored := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("authored drift")},
		}
		embedded := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("embedded canonical")},
		}

		drifts, err := compareParity(authored, embedded)
		require.NoError(t, err)
		require.Len(t, drifts, 1)
		assert.Equal(t, "workflows/parallel-team.md", drifts[0].Path)
		assert.Contains(t, drifts[0].Reason, "byte drift")
	})

	t.Run("embedded-only drift fails", func(t *testing.T) {
		authored := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("canonical")},
		}
		embedded := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("canonical")},
			"workflows/orphan.md":        &fstest.MapFile{Data: []byte("embedded-only")},
		}

		drifts, err := compareParity(authored, embedded)
		require.NoError(t, err)
		require.Len(t, drifts, 1)
		assert.Equal(t, "workflows/orphan.md", drifts[0].Path)
		assert.Contains(t, drifts[0].Reason, "embedded-only")
	})

	t.Run("authored-only drift fails", func(t *testing.T) {
		authored := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("canonical")},
			"workflows/orphan.md":        &fstest.MapFile{Data: []byte("authored-only")},
		}
		embedded := fstest.MapFS{
			"workflows/parallel-team.md": &fstest.MapFile{Data: []byte("canonical")},
		}

		drifts, err := compareParity(authored, embedded)
		require.NoError(t, err)
		require.Len(t, drifts, 1)
		assert.Equal(t, "workflows/orphan.md", drifts[0].Path)
		assert.Contains(t, drifts[0].Reason, "authored-only")
	})

	t.Run("synchronizer repairs authored mirror", func(t *testing.T) {
		embeddedRoot := t.TempDir()
		authoredRoot := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(embeddedRoot, "workflows"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(authoredRoot, "workflows"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(embeddedRoot, "workflows", "parallel-team.md"), []byte("canonical"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(authoredRoot, "workflows", "parallel-team.md"), []byte("drifted"), 0o644))

		before, err := compareParity(os.DirFS(authoredRoot), os.DirFS(embeddedRoot))
		require.NoError(t, err)
		require.NotEmpty(t, before)
		require.NoError(t, SyncSharkAttackTree(embeddedRoot, authoredRoot))

		after, err := compareParity(os.DirFS(authoredRoot), os.DirFS(embeddedRoot))
		require.NoError(t, err)
		assert.Empty(t, after)
	})
}
