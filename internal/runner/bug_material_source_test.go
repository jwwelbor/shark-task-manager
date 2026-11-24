package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/projectdocs"
	"github.com/stretchr/testify/require"
)

// Pause only the decision-source payload. Other Git calls, including plan
// approval and the candidate scope check, must complete normally.
func ingestMaterialSourceStream(t *testing.T, request GateIngestRequest, scope bugCandidateScope, path, challenge string) error {
	t.Helper()
	if runtime.GOOS == "windows" {
		if challenge == "release" {
			_, err := IngestGateResult(context.Background(), request)
			return err
		}
		t.Skip("Git process observation uses a POSIX shell")
	}
	bin := t.TempDir()
	git, err := exec.LookPath("git")
	require.NoError(t, err)
	object := scope.CandidateCommit + ":" + path
	oid, err := exec.Command(git, "-C", scope.ProjectRoot, "rev-parse", object).Output()
	require.NoError(t, err)
	t.Setenv("MATERIAL_SOURCE_OID", strings.TrimSpace(string(oid)))
	t.Setenv("MATERIAL_SOURCE_REV_PATH", object)
	t.Setenv("MATERIAL_TEST_GIT", git)
	t.Setenv("MATERIAL_TEST_DIR", bin)
	t.Setenv("MATERIAL_TEST_CHALLENGE", challenge)
	if challenge != "release" {
		request.RunID += "-" + challenge
	}
	script := `#!/bin/sh
case " $* " in
  *" cat-file blob $MATERIAL_SOURCE_OID "*|*" show $MATERIAL_SOURCE_REV_PATH "*)
    if [ "$MATERIAL_TEST_CHALLENGE" = overflow ]; then
      "$MATERIAL_TEST_GIT" "$@" || exit 1
      echo started > "$MATERIAL_TEST_DIR/started"
      printf '!'
      while [ ! -f "$MATERIAL_TEST_DIR/release" ]; do sleep 0.02; done
      printf trailing-bytes
      exit 0
    fi
    echo started > "$MATERIAL_TEST_DIR/started"
    while [ ! -f "$MATERIAL_TEST_DIR/release" ]; do sleep 0.02; done ;;
esac
exec "$MATERIAL_TEST_GIT" "$@"
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var ingestErr error
	go func() {
		defer close(done)
		_, ingestErr = IngestGateResult(ctx, request)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, os.WriteFile(filepath.Join(bin, "release"), nil, 0600))
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("ingestion did not terminate during cleanup")
		}
	})
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(bin, "started"))
		return err == nil
	}, 10*time.Second, 10*time.Millisecond, "admission must reach the frozen source")
	switch challenge {
	case "cancel":
		cancel()
	case "release":
		require.NoError(t, os.WriteFile(filepath.Join(bin, "release"), nil, 0600))
	}
	// The overflow producer has offered exactly one extra byte and is still
	// holding stdout open. The verifier must return before cleanup releases it.
	select {
	case <-done:
		return ingestErr
	case <-time.After(5 * time.Second):
		t.Fatalf("admission still waiting for the frozen source in %s trial", challenge)
		return nil
	}
}

func TestMaterialSourceRejectsNonregularGitEntries(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) []byte {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
		return out
	}
	git("init")
	require.NoError(t, os.Mkdir(filepath.Join(root, "docs"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "decision"), []byte("approved"), 0600))
	git("add", "docs")
	// Stage real symlink-mode metadata portably, without creating a filesystem symlink.
	oid := strings.TrimSpace(string(git("rev-parse", ":docs/decision")))
	git("update-index", "--add", "--cacheinfo", "120000,"+oid+",decision-link")
	git("-c", "user.name=fixture", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "decision sources")
	commit := strings.TrimSpace(string(git("rev-parse", "HEAD")))
	scope := bugCandidateScope{ProjectRoot: root, CandidateCommit: commit}
	for _, path := range []string{"docs/decision", "docs", "decision-link"} {
		t.Run(path, func(t *testing.T) {
			content := git("show", commit+":"+path)
			source := bugReviewMaterialDecision{SourcePointer: path, SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(content))}
			trace := filepath.Join(t.TempDir(), "git-trace")
			t.Setenv("GIT_TRACE", trace)
			err := verifyBugMaterialDecisionSource(context.Background(), scope, source)
			if path == "docs/decision" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "regular blob")
				log, readErr := os.ReadFile(trace)
				require.NoError(t, readErr)
				require.Contains(t, string(log), "ls-tree ")
				require.NotContains(t, string(log), "cat-file blob ")
				require.NotContains(t, string(log), "git show ")
			}
		})
	}
}

func TestMaterialDecisionDocumentSizeBound(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init")
	for _, extra := range []int64{0, 1} {
		t.Run(fmt.Sprintf("limit_plus_%d", extra), func(t *testing.T) {
			// Build a large, compressible regular document in Git without a large
			// working copy or in-memory buffer.
			const statement = "Approved decision.\n"
			padding := projectdocs.MaxDocumentBytes + extra - int64(len(statement))
			content := io.MultiReader(strings.NewReader(statement), io.LimitReader(materialSpacePadding{}, padding))
			h := sha256.New()
			cmd := exec.Command("git", "-C", root, "hash-object", "-w", "--stdin")
			cmd.Stdin = io.TeeReader(content, h)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			oid := strings.TrimSpace(string(out))
			require.Equal(t, fmt.Sprint(projectdocs.MaxDocumentBytes+extra), git("cat-file", "-s", oid))
			git("update-index", "--add", "--cacheinfo", "100644,"+oid+",decision")
			git("-c", "user.name=test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "decision")
			scope := bugCandidateScope{ProjectRoot: root, CandidateCommit: git("rev-parse", "HEAD")}
			trace := filepath.Join(t.TempDir(), "trace")
			t.Setenv("GIT_TRACE", trace)
			err = verifyBugMaterialDecisionSource(context.Background(), scope, bugReviewMaterialDecision{SourcePointer: "decision", SourceSHA256: fmt.Sprintf("%x", h.Sum(nil))})
			log, readErr := os.ReadFile(trace)
			require.NoError(t, readErr)
			if extra > 0 {
				require.ErrorContains(t, err, "size")
				require.NotContains(t, string(log), "cat-file blob ")
				require.NotContains(t, string(log), "git show ")
			} else {
				require.NoError(t, err)
				meta, payload := strings.Index(string(log), "ls-tree "), strings.Index(string(log), "cat-file blob "+oid)
				require.GreaterOrEqual(t, meta, 0)
				require.Greater(t, payload, meta)
			}
		})
	}
}

type materialSpacePadding struct{}

func (materialSpacePadding) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}
