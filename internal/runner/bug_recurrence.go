package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// bugRecurrenceRecord is the retained instance evidence a successor review
// cites when a new fingerprint appears inside a completed sweep's scope.
type bugRecurrenceRecord struct {
	Kind                  string   `json:"kind"`
	PriorRunID            string   `json:"prior_run_id"`
	PriorInstanceFinger   string   `json:"prior_instance_fingerprint"`
	ClassKey              string   `json:"class_key"`
	Scope                 []string `json:"scope"`
	RenewedEnumeration    []string `json:"renewed_enumeration"`
	EnumeratedAtCandidate string   `json:"candidate,omitempty"`
}

// bugPriorSweep is a completed sweep from an accepted, applied result of the
// same bug, resolved from the owning project's history rather than from
// anything the successor submitted.
type bugPriorSweep struct {
	RunID string
	Sweep gateresult.DefectClassSweep
}

type bugOccurrence struct {
	class, fingerprint string
	sites, evidence    []string
}

// verifyBugRecurrence requires explicit recurrence evidence for every new
// occurrence inside the scope of the latest accepted completed sweep of its
// class, and blocks when the required history cannot be verified.
func verifyBugRecurrence(ctx context.Context, req gatepersist.Request, history gatepersist.HistoryReader) error {
	if req.EntityType != models.EntityTypeBug || req.Gate != "code_review" || req.Result == nil {
		return nil
	}
	occurrences := currentBugOccurrences(req.Result)
	if len(occurrences) == 0 {
		return nil
	}
	if history == nil {
		return fmt.Errorf("recurrence history is required to classify successor findings")
	}
	prior, err := acceptedBugSweeps(ctx, req, history)
	if err != nil {
		return fmt.Errorf("recurrence history unavailable: %w", err)
	}
	for _, occurrence := range occurrences {
		match := latestRecurrenceMatch(prior, occurrence)
		if match == nil {
			continue
		}
		if err := requireRecurrenceEvidence(req, occurrence, *match); err != nil {
			return fmt.Errorf("class %s fingerprint %s recurs inside accepted sweep %s: %w", occurrence.class, occurrence.fingerprint, match.RunID, err)
		}
	}
	return nil
}

func currentBugOccurrences(result *gateresult.GateResult) []bugOccurrence {
	var out []bugOccurrence
	for _, finding := range result.Findings {
		out = append(out, bugOccurrence{finding.ClassKey, finding.Fingerprint, finding.AffectedIDs, finding.EvidencePointers})
	}
	for _, sweep := range result.RemediationSweeps {
		for _, instance := range sweep.Instances {
			out = append(out, bugOccurrence{sweep.ClassKey, instance.Fingerprint, []string{instance.SitePointer}, instance.EvidencePointers})
		}
	}
	return out
}

// acceptedBugSweeps lists completed sweeps retained by applied results, oldest first.
func acceptedBugSweeps(ctx context.Context, req gatepersist.Request, history gatepersist.HistoryReader) ([]bugPriorSweep, error) {
	entries, err := history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	parent := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	var out []bugPriorSweep
	for _, entry := range entries {
		if entry == nil || entry.Notes == nil {
			continue
		}
		match := bugGateTransitionNote.FindStringSubmatch(*entry.Notes)
		if match == nil || match[1] != "code_review" {
			continue
		}
		fields := strings.Fields(*entry.Notes)
		runID, digest := fields[5], fields[7]
		sweeps, err := appliedRunSweeps(parent, req, runID, digest)
		if err != nil {
			return nil, err
		}
		for _, sweep := range sweeps {
			out = append(out, bugPriorSweep{runID, sweep})
		}
	}
	return out, nil
}

func appliedRunSweeps(parent string, req gatepersist.Request, runID, digest string) ([]gateresult.DefectClassSweep, error) {
	dir, err := gaterun.RunDir(parent, runID)
	if err != nil {
		return nil, err
	}
	state, exists, err := gaterun.LoadOperationState(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("accepted run %s operation state is unavailable", runID)
	}
	if state.PersistenceState != gaterun.PersistenceStateTransitioned || state.OperationDigest != digest || !strings.EqualFold(state.EntityKey, req.EntityKey) {
		return nil, nil // not an accepted, applied result: it supplies no authority
	}
	data, exists, err := gaterun.ReadResult(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("accepted run %s result is unavailable", runID)
	}
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("accepted run %s identity is unavailable", runID)
	}
	computed, err := gaterun.ComputeBoundOperationDigest(req.EntityKey, string(req.EntityType), "code_review", "code_review", data, identity.PlanRoot, identity.PlanCommit, identity.PlanBlob)
	if err != nil || computed != digest {
		return nil, fmt.Errorf("accepted run %s result does not match its receipt", runID)
	}
	envelope, err := workercontrol.Decode(data)
	if err != nil {
		return nil, err
	}
	result, err := gateresult.Decode(envelope.GateResult)
	if err != nil {
		return nil, err
	}
	var complete []gateresult.DefectClassSweep
	for _, sweep := range result.RemediationSweeps {
		// A claimed completion earns historical credit only while its paired
		// guard evidence still resolves; otherwise it supplies no authority.
		if sweep.Status == "complete" && verifyBugSweepGuard(req, sweep, false) == nil {
			complete = append(complete, sweep)
		}
	}
	return complete, nil
}

func sitePath(site string) string {
	if i := strings.LastIndex(site, ":"); i > 0 {
		return site[:i]
	}
	return site
}

func scopeCovers(scope []string, site string) bool {
	path := sitePath(site)
	for _, member := range scope {
		if member == path || member == site || strings.HasPrefix(path, strings.TrimSuffix(member, "/")+"/") {
			return true
		}
	}
	return false
}

// latestRecurrenceMatch returns the newest accepted completed sweep of the same
// class whose scope covers a site of a fingerprint it has not already seen.
func latestRecurrenceMatch(prior []bugPriorSweep, occurrence bugOccurrence) *bugPriorSweep {
	for _, candidate := range prior {
		if candidate.Sweep.ClassKey == occurrence.class && seenInSweep(candidate.Sweep, occurrence.fingerprint) {
			return nil // a recorded fingerprint follows the existing same-fingerprint path
		}
	}
	for i := len(prior) - 1; i >= 0; i-- {
		candidate := prior[i]
		if candidate.Sweep.ClassKey != occurrence.class {
			continue
		}
		for _, site := range occurrence.sites {
			if scopeCovers(candidate.Sweep.SearchScope, site) {
				return &candidate
			}
		}
	}
	return nil
}

func seenInSweep(sweep gateresult.DefectClassSweep, fingerprint string) bool {
	for _, instance := range sweep.Instances {
		if instance.Fingerprint == fingerprint {
			return true
		}
	}
	return false
}

func requireRecurrenceEvidence(req gatepersist.Request, occurrence bugOccurrence, match bugPriorSweep) error {
	root := bugEvidenceProject(req)
	for _, pointer := range occurrence.evidence {
		if !filepath.IsLocal(pointer) {
			continue
		}
		data, err := readBoundedBugEvidence(root, pointer, bugEvidenceLimit(""))
		if err != nil {
			continue
		}
		var record bugRecurrenceRecord
		if decodeStrict(data, &record) != nil || record.Kind != "recurrence" {
			continue
		}
		return checkRecurrenceRecord(record, match)
	}
	return fmt.Errorf("explicit recurrence evidence citing the prior sweep, its class and scope is required")
}

func checkRecurrenceRecord(record bugRecurrenceRecord, match bugPriorSweep) error {
	if record.PriorRunID != match.RunID || record.ClassKey != match.Sweep.ClassKey || !seenInSweep(match.Sweep, record.PriorInstanceFinger) {
		return fmt.Errorf("recurrence cites a different sweep than the latest accepted one")
	}
	if !sameMembers(record.Scope, match.Sweep.SearchScope) {
		return fmt.Errorf("recurrence scope differs from the prior sweep's recorded scope")
	}
	enumerated := make(map[string]bool, len(record.RenewedEnumeration))
	for _, member := range record.RenewedEnumeration {
		enumerated[member] = true
	}
	for _, member := range match.Sweep.SearchScope {
		if !enumerated[member] {
			return fmt.Errorf("renewed enumeration omits %s from the prior scope", member)
		}
	}
	return nil
}

func sameMembers(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	if len(set) != len(b) {
		return false
	}
	for _, v := range b {
		if !set[v] {
			return false
		}
	}
	return true
}
