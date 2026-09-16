// Reconcile the current non-terminal Shark tech-debt collection against the
// 2026-09-16 residual-closure plan. Run with the unmodified JSON emitted by
// `shark td list --json`.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type techDebtStatus string

const (
	statusIdentified techDebtStatus = "identified"
	statusResearch   techDebtStatus = "research"
	statusInProgress techDebtStatus = "in_progress"
)

type techDebt struct {
	Key    string         `json:"key"`
	Status techDebtStatus `json:"status"`
}

type ownership struct {
	Key            string         `json:"key"`
	Batch          string         `json:"batch"`
	ExpectedStatus techDebtStatus `json:"expected_status"`
}

type reconciliation struct {
	Active           int      `json:"active"`
	Planned          int      `json:"planned"`
	Duplicates       []string `json:"duplicates"`
	InvalidRecords   []string `json:"invalid_records"`
	Missing          []string `json:"missing"`
	Extra            []string `json:"extra"`
	StatusMismatched []string `json:"status_mismatched"`
}

//go:embed residual-closure-manifest.json
var manifestJSON []byte

func loadPlanned() ([]ownership, error) {
	var planned []ownership
	if err := json.Unmarshal(manifestJSON, &planned); err != nil {
		return nil, fmt.Errorf("decode residual closure manifest: %w", err)
	}
	return planned, nil
}

func sorted(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func indexLive(live []techDebt) (map[string]techDebtStatus, map[string]bool, []string) {
	active := map[string]techDebtStatus{}
	duplicates := map[string]bool{}
	invalidRecords := []string{}
	for index, item := range live {
		if item.Key == "" {
			invalidRecords = append(invalidRecords, fmt.Sprintf("record[%d]: blank key", index))
			continue
		}
		if _, exists := active[item.Key]; exists {
			duplicates[item.Key] = true
			continue
		}
		active[item.Key] = item.Status
	}
	return active, duplicates, invalidRecords
}

func indexOwnership(planned []ownership, duplicates map[string]bool) map[string]string {
	owners := map[string]string{}
	for _, item := range planned {
		if _, exists := owners[item.Key]; exists {
			duplicates[item.Key] = true
		}
		owners[item.Key] = item.Batch
	}
	return owners
}

func compareKeys(active map[string]techDebtStatus, owners map[string]string) (map[string]bool, map[string]bool) {
	missing, extra := map[string]bool{}, map[string]bool{}
	for key := range active {
		if _, exists := owners[key]; !exists {
			extra[key] = true
		}
	}
	for key := range owners {
		if _, exists := active[key]; !exists {
			missing[key] = true
		}
	}
	return missing, extra
}

func findStatusMismatches(active map[string]techDebtStatus, planned []ownership) map[string]bool {
	statusMismatched := map[string]bool{}
	for _, item := range planned {
		if actual, exists := active[item.Key]; exists && actual != item.ExpectedStatus {
			statusMismatched[item.Key] = true
		}
	}
	return statusMismatched
}

func reconcile(live []techDebt, planned []ownership) reconciliation {
	active, duplicates, invalidRecords := indexLive(live)
	owners := indexOwnership(planned, duplicates)
	missing, extra := compareKeys(active, owners)
	statusMismatched := findStatusMismatches(active, planned)
	return reconciliation{
		Active:           len(active),
		Planned:          len(owners),
		Duplicates:       sorted(duplicates),
		InvalidRecords:   invalidRecords,
		Missing:          sorted(missing),
		Extra:            sorted(extra),
		StatusMismatched: sorted(statusMismatched),
	}
}

func (report reconciliation) valid() bool {
	return len(report.Duplicates) == 0 && len(report.InvalidRecords) == 0 && len(report.Missing) == 0 && len(report.Extra) == 0 && len(report.StatusMismatched) == 0
}

func validateLiveInputPath(input string) (string, error) {
	resolved, err := filepath.EvalSymlinks(input)
	if err != nil {
		return "", fmt.Errorf("resolve live collection path: %w", err)
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return "", fmt.Errorf("resolve temporary directory: %w", err)
	}
	relative, err := filepath.Rel(tempRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("live collection path must resolve under %s", tempRoot)
	}
	return resolved, nil
}

func loadLive(input string) ([]techDebt, error) {
	path, err := validateLiveInputPath(input)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read live collection: %w", err)
	}
	var live []techDebt
	if err := json.Unmarshal(data, &live); err != nil {
		return nil, fmt.Errorf("decode live collection: %w", err)
	}
	return live, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: reconcile-residual-tech-debt.go <shark-td-list.json>")
		os.Exit(2)
	}
	live, err := loadLive(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	planned, err := loadPlanned()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report := reconcile(live, planned)
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !report.valid() {
		os.Exit(1)
	}
}
