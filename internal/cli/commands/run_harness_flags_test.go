package commands

// Covers T-E34-F01-005 AC-T2: `shark run` registers the same three harness
// override flags as `shark next` and wires them into the run controller as
// the resolver's override identity (spec.md §3.3, REQ-F-006/AC-08).
//
// TestRunCmd_RegistersHarnessOverrideFlags exercises the real runCmd
// singleton's flag set directly (no DB, no cobra Execute needed). The
// wiring-into-RunOptions/RunControllerDeps assertions use a source scan,
// following TestNewNextAdapterCache_WiresHarnessResolver's precedent
// (next_harness_test.go) for production wiring lines that runRun itself
// can't exercise in a CLI test without a real project/DB (runRun calls
// cli.GetActionService/cli.FindProjectRoot directly, unlike next.go's
// injectable nextNewAdapterCache seam).

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunCmd_RegistersHarnessOverrideFlags covers AC-T2's first clause:
// `run.go` registers the same three override flags as `next.go`.
func TestRunCmd_RegistersHarnessOverrideFlags(t *testing.T) {
	for _, name := range []string{"harness", "harness-version", "harness-model"} {
		if runCmd.Flags().Lookup(name) == nil {
			t.Errorf("runCmd missing --%s flag; without it, precedence tier 1 (flags) has no entry point under `shark run` (REQ-F-006/AC-08)", name)
		}
	}
}

// TestRunRun_WiresHarnessOverrideAndResolver pins the production wiring
// runRun performs: reading the three override flags once via
// harnessOverrideFromFlags (the same helper next.go's runNext uses), passing
// the result as RunOptions.HarnessOverride, and constructing both the
// top-level and cascade-child RunControllerDeps with a real
// cli.GetHarnessResolver(). Without every one of these lines, `shark run`
// would silently resolve no harness identity, or resolve one but never see
// flag overrides, while every other test in this file still passes.
func TestRunRun_WiresHarnessOverrideAndResolver(t *testing.T) {
	src, err := os.ReadFile("run.go")
	require.NoError(t, err)
	body := string(src)

	assert.Contains(t, body, `harnessOverride, err := harnessOverrideFromFlags(cmd)`,
		"runRun must read the --harness/--harness-version/--harness-model flags via the shared harnessOverrideFromFlags helper")
	assert.Contains(t, body, `HarnessOverride: harnessOverride`,
		"runRun must pass the resolved override into RunOptions.HarnessOverride")

	// Both the top-level controller and the cascade-child controller must be
	// wired with a real resolver — a fix that adds this to only one call site
	// silently loses harness resolution partway down a cascade.
	occurrences := 0
	for i := 0; i+len(`HarnessResolver:   cli.GetHarnessResolver()`) <= len(body); i++ {
		if body[i:i+len(`HarnessResolver:   cli.GetHarnessResolver()`)] == `HarnessResolver:   cli.GetHarnessResolver()` {
			occurrences++
		}
	}
	assert.GreaterOrEqual(t, occurrences, 2,
		"expected HarnessResolver: cli.GetHarnessResolver() wired into both the top-level and cascade-child RunControllerDeps, got %d occurrence(s)", occurrences)
}
