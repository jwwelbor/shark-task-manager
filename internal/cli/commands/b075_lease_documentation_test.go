package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestB075ExternalAdapterLeaseLifecycleContract prevents the external Rider
// loop from regressing to a one-shot claim followed by an unprotected
// --apply-result call. The parent owns the lease for the entire worker
// lifetime; the worker must never be asked to repair a lost lease itself.
func TestB075ExternalAdapterLeaseLifecycleContract(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))

	read := func(relativePath string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(projectRoot, relativePath))
		if err != nil {
			t.Fatalf("read %s: %v", relativePath, err)
		}
		return string(content)
	}

	run := read(filepath.Join("skills", "shark-rider", "verbs", "run.md"))
	contract := read(filepath.Join("skills", "shark-rider", "context", "host-adapter-contract.md"))

	for _, marker := range []string{
		"lease supervisor",
		"before spawning the worker",
		"TTL/3",
		"heartbeat failure",
		"stop the worker",
		"do not apply the result",
		"final heartbeat",
		"--apply-result",
	} {
		if !strings.Contains(run, marker) {
			t.Errorf("run.md omits B075 lease-lifecycle requirement %q", marker)
		}
	}

	for _, marker := range []string{
		"parent owns the lease",
		"same session",
		"If renewal fails",
		"never apply",
		"re-claim",
	} {
		if !strings.Contains(contract, marker) {
			t.Errorf("host-adapter-contract.md omits B075 ownership requirement %q", marker)
		}
	}

	supervisor := strings.Index(run, "Before spawning the worker, start a parent-owned lease supervisor")
	spawn := strings.Index(run, "Spawn the host worker using")
	finalHeartbeat := strings.Index(run, "final heartbeat")
	applyResult := strings.Index(run, "--apply-result")
	if supervisor < 0 || spawn < 0 || supervisor > spawn {
		t.Errorf("run.md must start the lease supervisor before spawning the worker")
	}
	if finalHeartbeat < 0 || applyResult < 0 || finalHeartbeat > applyResult {
		t.Errorf("run.md must require a final heartbeat before --apply-result")
	}
}
