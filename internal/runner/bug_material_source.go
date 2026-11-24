package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
	"github.com/jwwelbor/shark-task-manager/internal/projectdocs"
)

func verifyBugMaterialDecisionSource(ctx context.Context, scope bugCandidateScope, decision bugReviewMaterialDecision) error {
	root, _, err := candidateRepository(ctx, scope.ProjectRoot)
	if err != nil {
		return fmt.Errorf("resolve material decision source repository: %w", err)
	}
	path := filepath.ToSlash(filepath.Clean(decision.SourcePointer))
	if path == "." || !filepath.IsLocal(path) {
		return fmt.Errorf("invalid material decision source path %q", path)
	}
	entry, err := gitutil.Command(ctx, "--literal-pathspecs", "-C", root, "ls-tree", "-l", "-z", "--full-tree", scope.CandidateCommit, "--", path).Output()
	if err != nil {
		return fmt.Errorf("inspect material decision source %q: %w", path, err)
	}
	oid, size, err := materialBlobEntry(string(entry), path)
	if err != nil {
		return err
	}
	digest, err := materialBlobDigest(ctx, root, oid, size)
	if err != nil {
		return fmt.Errorf("read frozen material decision source %q: %w", path, err)
	}
	if !strings.EqualFold(digest, decision.SourceSHA256) {
		return fmt.Errorf("material decision source digest does not match frozen candidate")
	}
	return nil
}

func materialBlobEntry(entry, path string) (string, int64, error) {
	meta, name, _ := strings.Cut(entry, "\t")
	fields := strings.Fields(meta)
	if name != path+"\x00" || len(fields) != 4 {
		return "", 0, fmt.Errorf("material decision source %q has no matching frozen tree entry", path)
	}
	if (fields[0] != "100644" && fields[0] != "100755") || fields[1] != "blob" || !validGitHash(fields[2]) {
		return "", 0, fmt.Errorf("material decision source %q must be a regular blob", path)
	}
	size, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil || size < 0 || size > projectdocs.MaxDocumentBytes {
		return "", 0, fmt.Errorf("material decision source %q has unsupported size %q (limit %d)", path, fields[3], projectdocs.MaxDocumentBytes)
	}
	return fields[2], size, nil
}

func materialBlobDigest(ctx context.Context, root, oid string, size int64) (string, error) {
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := gitutil.Command(readCtx, "-C", root, "cat-file", "blob", oid)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	hash := sha256.New()
	n, readErr := io.Copy(hash, io.LimitReader(out, size+1))
	if readErr != nil || n != size {
		// Stop the producer before waiting if it offered more than its tree entry.
		cancel()
	}
	waitErr := cmd.Wait()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if readErr != nil {
		return "", readErr
	}
	if n != size {
		return "", fmt.Errorf("blob size differs from frozen entry (%d bytes expected, read %d)", size, n)
	}
	if waitErr != nil {
		return "", waitErr
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
