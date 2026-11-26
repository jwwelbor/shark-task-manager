package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
)

// A frozen decision source declares its own obligations. The parent enumerates
// them from the candidate tree, so neither the report markers nor the writer's
// expected packet can add or remove a controlling decision.
const (
	materialSourceMarker   = "MATERIAL_DECISION_SOURCE:"
	materialArtifactMarker = "MATERIAL_AFFECTS_ARTIFACT:"
	materialCriterionMark  = "MATERIAL_AFFECTS_CRITERION:"
)

type frozenMaterialDecision struct {
	ID        string
	Source    string
	Artifacts map[string]bool
	Criteria  map[materialCriterionKey]bool
}

// enumerateFrozenMaterialDecisions lists every decision declared in the frozen
// candidate tree, keyed by decision ID.
func enumerateFrozenMaterialDecisions(ctx context.Context, scope bugCandidateScope) (map[string]*frozenMaterialDecision, error) {
	root, _, err := candidateRepository(ctx, scope.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve material decision repository: %w", err)
	}
	out, err := gitutil.Command(ctx, "--literal-pathspecs", "-C", root, "grep", "-z", "-l", "-F", "-e", materialSourceMarker, scope.CandidateCommit, "--").Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("enumerate frozen material decisions: %w", err)
	}
	found := make(map[string]*frozenMaterialDecision)
	prefix := scope.CandidateCommit + ":"
	for _, entry := range bytes.Split(out, []byte{0}) {
		path, ok := strings.CutPrefix(string(entry), prefix)
		if !ok {
			continue
		}
		content, err := readFrozenMaterialBlob(ctx, scope, path)
		if err != nil {
			return nil, fmt.Errorf("read frozen decision source %q: %w", path, err)
		}
		if err := parseFrozenMaterialSource(path, string(content), found); err != nil {
			return nil, err
		}
	}
	return found, nil
}

func parseFrozenMaterialSource(path, content string, found map[string]*frozenMaterialDecision) error {
	var current *frozenMaterialDecision
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, materialSourceMarker):
			id := strings.TrimSpace(strings.TrimPrefix(line, materialSourceMarker))
			if id == "" || found[id] != nil {
				return fmt.Errorf("frozen decision source %s declares an invalid or duplicate decision", path)
			}
			current = &frozenMaterialDecision{ID: id, Source: path, Artifacts: map[string]bool{}, Criteria: map[materialCriterionKey]bool{}}
			found[id] = current
		case strings.HasPrefix(line, materialArtifactMarker) && current != nil:
			current.Artifacts[strings.TrimSpace(strings.TrimPrefix(line, materialArtifactMarker))] = true
		case strings.HasPrefix(line, materialCriterionMark) && current != nil:
			fields := strings.Fields(strings.TrimPrefix(line, materialCriterionMark))
			if len(fields) != 3 {
				return fmt.Errorf("frozen decision source %s declares a malformed criterion", path)
			}
			current.Criteria[materialCriterionKey{fields[0], fields[1], fields[2]}] = true
		}
	}
	return nil
}

// compareFrozenMaterialDecisions requires the writer's expected packet to equal
// the independently enumerated set, including each decision's obligations.
func compareFrozenMaterialDecisions(enumerated map[string]*frozenMaterialDecision, expected []bugReviewMaterialDecision) error {
	byID := make(map[string]bugReviewMaterialDecision, len(expected))
	for _, decision := range expected {
		byID[decision.DecisionID] = decision
	}
	ids := make([]string, 0, len(enumerated))
	for id := range enumerated {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		decision, ok := byID[id]
		if !ok {
			return fmt.Errorf("frozen source %s declares material decision %s that the expected impacts omit", enumerated[id].Source, id)
		}
		if err := compareFrozenMaterialMembers(enumerated[id], decision); err != nil {
			return err
		}
		delete(byID, id)
	}
	for id := range byID {
		return fmt.Errorf("expected impacts list material decision %s that no frozen source declares", id)
	}
	return nil
}

func compareFrozenMaterialMembers(source *frozenMaterialDecision, decision bugReviewMaterialDecision) error {
	if decision.SourcePointer != source.Source {
		return fmt.Errorf("%s: expected source %s differs from frozen source %s", source.ID, decision.SourcePointer, source.Source)
	}
	artifacts := make(map[string]bool, len(decision.AffectedArtifacts))
	for _, artifact := range decision.AffectedArtifacts {
		artifacts[artifact.Path] = true
		if !source.Artifacts[artifact.Path] {
			return fmt.Errorf("%s: expected artifact %s is not declared by the frozen source", source.ID, artifact.Path)
		}
	}
	for path := range source.Artifacts {
		if !artifacts[path] {
			return fmt.Errorf("%s: frozen source declares artifact %s that the expected impacts omit", source.ID, path)
		}
	}
	criteria := make(map[materialCriterionKey]bool, len(decision.AffectedCriteria))
	for _, criterion := range decision.AffectedCriteria {
		key := materialCriterionKey{criterion.EntityKey, criterion.ProductionCallerPath, criterion.AcceptanceCriterionID}
		criteria[key] = true
		if !source.Criteria[key] {
			return fmt.Errorf("%s: expected criterion %s is not declared by the frozen source", source.ID, key)
		}
	}
	for key := range source.Criteria {
		if !criteria[key] {
			return fmt.Errorf("%s: frozen source declares criterion %s that the expected impacts omit", source.ID, key)
		}
	}
	return nil
}
