package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// readBugEvidence resolves one digest-bound, project-local regular file.
func readBugEvidence(req gatepersist.Request, kind string) (workercontrol.EvidenceRef, []byte, error) {
	envelope, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return workercontrol.EvidenceRef{}, nil, err
	}
	var found *workercontrol.EvidenceRef
	for i := range envelope.Evidence {
		if envelope.Evidence[i].Kind == kind {
			if found != nil {
				return workercontrol.EvidenceRef{}, nil, fmt.Errorf("exactly one %s reference is required", kind)
			}
			found = &envelope.Evidence[i]
		}
	}
	if found == nil {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("%s evidence is required", kind)
	}
	if !filepath.IsLocal(found.Pointer) || found.Pointer == "." {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("evidence path must be local to project")
	}
	if !strings.HasPrefix(found.Summary, "sha256:") {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("evidence digest is required")
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(found.Summary, "sha256:"))
	if err != nil || len(expected) != sha256.Size {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("invalid evidence digest")
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return workercontrol.EvidenceRef{}, nil, err
	}
	defer func() { _ = root.Close() }()
	file, err := root.OpenFile(found.Pointer, os.O_RDONLY|bugPlanOpenFlags, 0)
	if err != nil {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("open evidence: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2*1024*1024 {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("evidence must be a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("cannot read bounded evidence")
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), hex.EncodeToString(expected)) {
		return workercontrol.EvidenceRef{}, nil, fmt.Errorf("evidence digest is stale")
	}
	return *found, data, nil
}

// readBugReviewScope binds a gate result to the exact saved scope and verifies
// that the source still produces the reviewed tree before any transition.
func readBugReviewScope(ctx context.Context, req gatepersist.Request) (workercontrol.EvidenceRef, error) {
	ref, data, err := readBugEvidence(req, "bug_review_scope")
	if err != nil {
		return ref, err
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	if err := verifyBugCandidate(ctx, root, data); err != nil {
		return ref, fmt.Errorf("review candidate is stale or invalid: %w", err)
	}
	return ref, nil
}
