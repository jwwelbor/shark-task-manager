#!/usr/bin/env bash
# TC-111a / T-E40-F11-017 (spec.md REQ-F-012, AC-F11-38, AC-F11-39;
# test-plan.md TC-38/39): `setup_config` keys `scenario_roots` by
# `scenario_id`, never by `entity_family`, and `setup --scenario-index
# <path>` seeds a caller-chosen set of dedicated hierarchies through the
# production CLI (spec.md §2.1.4, §2.3.6).
#
# Scope: the families already admitted at this point in the build (feature,
# bug, change_card, tech_debt, task -- registered by T-E40-F11-008 -- and
# epic -- registered by T-E40-F11-009's py-epic-task-organization). All six
# families now have exactly one admitted seed each, so this file's default
# (no --scenario-index) run seeds all six. AC-F11-38a's synthetic
# duplicate-family collision proof and AC-F11-40/41/41a's full isolation +
# live-database non-mutation proof are T-E40-F11-011's scope -- not
# duplicated here (this task's own task file is explicit about that split).
#
# Caller-Path Contract: real filesystem, real git, the real Shark binary
# (bin/shark), and the real committed scenario packages. `setup` and `demo`
# are invoked exactly as an operator would invoke them, through
# bench/scripts/e40-benchmark.sh -- nothing is stubbed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"

fail() {
	echo "TC-111a FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh is missing or not executable"
[[ -x "$REPO_ROOT/bin/shark" ]] || fail "bin/shark is missing; run 'make shark' first"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# AC-F11-38 / TC-38: setup_config keys scenario_roots by scenario_id -- run
# the default (real) scenario index and assert the seeded key set is
# EXACTLY the admitted scenario_ids, never the family names the pre-rework
# `roots[family]` lookup used.
# ===========================================================================
DEFAULT_ROOT="$WORKDIR/default-setup"
"$OPERATOR" setup --out "$DEFAULT_ROOT" >"$WORKDIR/default-setup.out" 2>&1 \
	|| fail "default setup failed: $(cat "$WORKDIR/default-setup.out")"

python3 - "$DEFAULT_ROOT/setup-result.json" "$DEFAULT_ROOT/e40-demo.yaml" <<'PY'
import json
import sys

import yaml

result_path, config_path = sys.argv[1:3]
result = json.load(open(result_path, encoding="utf-8"))
config = yaml.safe_load(open(config_path, encoding="utf-8"))

expected_scenario_ids = {
    "py-bug-due-date-boundary",
    "py-change-priority-scale",
    "py-feature-recurring-tasks",
    "py-techdebt-consolidate-validation",
    "py-task-delete-task",
    "py-epic-task-organization",
}
family_names = {"bug", "change_card", "feature", "tech_debt", "epic", "task"}

root_keys = result["root_keys"]
if set(root_keys) != expected_scenario_ids:
    raise SystemExit(
        f"root_keys not keyed by the admitted scenario_ids: {sorted(root_keys)}"
    )
if set(root_keys) & family_names:
    raise SystemExit(f"root_keys is keyed by entity_family, not scenario_id: {sorted(root_keys)}")

scenario_roots = config["scenario_roots"]
if set(scenario_roots) != expected_scenario_ids:
    raise SystemExit(
        f"config scenario_roots not keyed by the admitted scenario_ids: {sorted(scenario_roots)}"
    )
if set(scenario_roots) & family_names:
    raise SystemExit(
        f"config scenario_roots is keyed by entity_family, not scenario_id: {sorted(scenario_roots)}"
    )

for scenario_id, entry in scenario_roots.items():
    if entry.get("root_key") != root_keys[scenario_id]:
        raise SystemExit(
            f"config scenario_roots.{scenario_id}.root_key disagrees with setup-result root_keys"
        )

# Six distinct entities -- no collision (only one scenario per family is
# admitted today; AC-F11-38a's synthetic duplicate-family proof is
# T-E40-F11-011's scope, not repeated here).
if len(set(root_keys.values())) != 6:
    raise SystemExit(f"root_keys values are not six distinct entities: {root_keys}")

print(
    "TC-111a(AC-F11-38): scenario_roots and root_keys are keyed by "
    "scenario_id, never entity_family -- PASS"
)
PY

# ===========================================================================
# AC-F11-39 / AC-F11-40 / TC-39: each already-admitted family gets the
# hierarchy shape REQ-F-012 specifies, and every key resolves to a
# real entity in the scratch project via `shark get --json` -- proving the
# key was actually captured from Shark's own JSON response, never
# constructed or guessed by the harness.
# ===========================================================================
SCRATCH_DIR="$DEFAULT_ROOT/scratch-template"
SCRATCH_BINARY="$SCRATCH_DIR/shark"
[[ -x "$SCRATCH_BINARY" ]] || fail "setup did not produce a scratch Shark binary"

shark_get() {
	# shark_get <key> -- runs `shark get <key> --json` against the isolated
	# scratch project only; SHARK_DB_URL/SHARK_BIN are stripped so a stray
	# inherited value cannot redirect this at anything else, and the
	# repository's live shark-tasks.db is never referenced.
	(
		cd "$SCRATCH_DIR" &&
			env -u SHARK_DB_URL -u SHARK_BIN SHARK_DB_BACKEND=sqlite \
				"$SCRATCH_BINARY" get "$1" --json
	)
}

result_field() {
	python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['root_keys'][sys.argv[2]])" \
		"$DEFAULT_ROOT/setup-result.json" "$1"
}

BUG_KEY="$(result_field py-bug-due-date-boundary)"
CHANGE_KEY="$(result_field py-change-priority-scale)"
FEATURE_KEY="$(result_field py-feature-recurring-tasks)"
TECH_DEBT_KEY="$(result_field py-techdebt-consolidate-validation)"
TASK_KEY="$(result_field py-task-delete-task)"
EPIC_KEY="$(result_field py-epic-task-organization)"

[[ "$BUG_KEY" =~ ^B[0-9]{3}$ ]] || fail "bug scenario root_key is not a bug key: $BUG_KEY"
[[ "$CHANGE_KEY" =~ ^CC-[0-9]{3}$ ]] || fail "change-card scenario root_key is not a change-card key: $CHANGE_KEY"
[[ "$FEATURE_KEY" =~ ^E[0-9]{2}-F[0-9]{2}$ ]] || fail "feature scenario root_key is not a feature key: $FEATURE_KEY"
[[ "$TECH_DEBT_KEY" =~ ^TD-[0-9]{3}$ ]] || fail "tech-debt scenario root_key is not a tech-debt key: $TECH_DEBT_KEY"
[[ "$TASK_KEY" =~ ^T-E[0-9]{2}-F[0-9]{2}-[0-9]{3}$ ]] || fail "task scenario root_key is not a task key: $TASK_KEY"
[[ "$EPIC_KEY" =~ ^E[0-9]{2}$ ]] || fail "epic scenario root_key is not an epic key: $EPIC_KEY"
echo "TC-111a(AC-F11-39): every root_key matches its own family's key shape -- PASS"

shark_get "$BUG_KEY" >/dev/null || fail "seeded bug root_key does not resolve via shark get: $BUG_KEY"
shark_get "$CHANGE_KEY" >/dev/null || fail "seeded change-card root_key does not resolve via shark get: $CHANGE_KEY"
shark_get "$TECH_DEBT_KEY" >/dev/null || fail "seeded tech-debt root_key does not resolve via shark get: $TECH_DEBT_KEY"
TASK_GET="$(shark_get "$TASK_KEY")" || fail "seeded task scenario root_key does not resolve via shark get: $TASK_KEY"
echo "$TASK_GET" | python3 -c "
import json, sys
doc = json.load(sys.stdin)
# shark get <task-key> --json returns the task's fields flat at the top
# level (unlike shark get <feature-key>, which nests under 'feature').
assert doc['key'] == sys.argv[1], doc
" "$TASK_KEY" || fail "shark get did not return the seeded task entity: $TASK_GET"
FEATURE_GET="$(shark_get "$FEATURE_KEY")" || fail "seeded feature scenario root_key does not resolve via shark get: $FEATURE_KEY"
echo "$FEATURE_GET" | python3 -c "
import json, sys
doc = json.load(sys.stdin)
assert doc['feature']['key'] == sys.argv[1], doc
" "$FEATURE_KEY" || fail "shark get did not return the seeded feature entity: $FEATURE_GET"
EPIC_GET="$(shark_get "$EPIC_KEY")" || fail "seeded epic scenario root_key does not resolve via shark get: $EPIC_KEY"
echo "$EPIC_GET" | python3 -c "
import json, sys
doc = json.load(sys.stdin)
assert doc['epic']['key'] == sys.argv[1], doc
" "$EPIC_KEY" || fail "shark get did not return the seeded epic entity: $EPIC_GET"
echo "TC-111a(AC-F11-40): every root_key was captured from Shark's own JSON response and resolves in the scratch project -- PASS"

# Feature scenario and task scenario EACH get a DEDICATED host epic that is
# never reused (the task scenario's own host epic + host feature, per
# T-E40-F11-017's seed_scenario_root; T-E40-F11-008 AC-F11-29); the epic
# scenario's own root IS itself a dedicated epic (T-E40-F11-009,
# seed_scenario_root's family == "epic" branch needs no separate "host"
# wrapper above it): the scratch project holds exactly three epics -- one
# per hierarchical/epic-root scenario -- never a shared one.
EPIC_COUNT="$(
	cd "$SCRATCH_DIR" &&
		env -u SHARK_DB_URL -u SHARK_BIN SHARK_DB_BACKEND=sqlite \
			"$SCRATCH_BINARY" epic list --json | python3 -c "import json,sys; print(json.load(sys.stdin)['count'])"
)"
[[ "$EPIC_COUNT" == "3" ]] || fail "expected exactly three dedicated epics (feature scenario host epic + task scenario host epic + epic scenario's own root), found $EPIC_COUNT"
echo "TC-111a(AC-F11-39): feature and task scenarios each have their own single dedicated host epic, and the epic scenario its own dedicated root epic, never shared -- PASS"

# ===========================================================================
# `setup --scenario-index <path>` (spec.md §2.3.6): the seam is reachable
# through the production CLI, not by editing harness internals. A custom
# index naming a STRICT SUBSET of the four admitted packages produces a
# strict subset of seeded roots -- proving the flag actually changes what
# gets seeded, not merely accepted and ignored.
# ===========================================================================
CUSTOM_INDEX="$WORKDIR/custom-scenario-index.yaml"
cat >"$CUSTOM_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $BENCH_DIR/scenarios/packages/py-bug-due-date-boundary
  - $BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation
EOF

CUSTOM_ROOT="$WORKDIR/custom-setup"
"$OPERATOR" setup --out "$CUSTOM_ROOT" --scenario-index "$CUSTOM_INDEX" \
	>"$WORKDIR/custom-setup.out" 2>&1 \
	|| fail "setup --scenario-index failed: $(cat "$WORKDIR/custom-setup.out")"

python3 - "$CUSTOM_ROOT/setup-result.json" "$CUSTOM_ROOT/e40-demo.yaml" "$CUSTOM_INDEX" <<'PY'
import json
import sys

import yaml

result_path, config_path, custom_index = sys.argv[1:4]
result = json.load(open(result_path, encoding="utf-8"))
config = yaml.safe_load(open(config_path, encoding="utf-8"))

expected = {"py-bug-due-date-boundary", "py-techdebt-consolidate-validation"}
if set(result["root_keys"]) != expected:
    raise SystemExit(
        f"--scenario-index did not seed exactly the requested subset: {sorted(result['root_keys'])}"
    )
if len(result["scenario_matrix"]) != 2:
    raise SystemExit(
        f"--scenario-index did not restrict the scenario matrix to 2 entries: {len(result['scenario_matrix'])}"
    )
if config["scenario_index"] != custom_index:
    raise SystemExit(
        f"config scenario_index was not persisted as the requested custom index: {config['scenario_index']!r}"
    )
print(
    "TC-111a(setup --scenario-index): a caller-supplied index seeds exactly "
    "its own subset through the production CLI -- PASS"
)
PY

# A re-run against the SAME operator root with the SAME --scenario-index is
# the existing idempotent "already_prepared" path -- unaffected.
"$OPERATOR" setup --out "$CUSTOM_ROOT" --scenario-index "$CUSTOM_INDEX" \
	>"$WORKDIR/custom-setup-repeat.out" 2>&1 \
	|| fail "idempotent re-run with the same --scenario-index failed: $(cat "$WORKDIR/custom-setup-repeat.out")"
grep -q '"already_prepared"' "$WORKDIR/custom-setup-repeat.out" \
	|| fail "same --scenario-index re-run did not report already_prepared: $(cat "$WORKDIR/custom-setup-repeat.out")"
echo "TC-111a(setup --scenario-index): identical re-run against the same operator root is idempotent -- PASS"

# A re-run against the SAME operator root with a DIFFERENT --scenario-index
# must be refused, naming both paths -- never silently accepted and
# dropped (the exact "accepted but silently ignored" defect class this
# feature exists to close).
OTHER_INDEX="$WORKDIR/other-scenario-index.yaml"
cat >"$OTHER_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $BENCH_DIR/scenarios/packages/py-bug-due-date-boundary
EOF
mismatch_out="$WORKDIR/custom-setup-mismatch.out"
mismatch_rc=0
"$OPERATOR" setup --out "$CUSTOM_ROOT" --scenario-index "$OTHER_INDEX" \
	>"$mismatch_out" 2>&1 || mismatch_rc=$?
[[ "$mismatch_rc" -ne 0 ]] || fail "setup accepted a different --scenario-index against an already-prepared operator root: $(cat "$mismatch_out")"
grep -q "different scenario_index" "$mismatch_out" || fail "mismatch refusal did not name the scenario_index conflict: $(cat "$mismatch_out")"
grep -qF "$OTHER_INDEX" "$mismatch_out" || fail "mismatch refusal did not name the requested index: $(cat "$mismatch_out")"
grep -qF "$CUSTOM_INDEX" "$mismatch_out" || fail "mismatch refusal did not name the existing index: $(cat "$mismatch_out")"
echo "TC-111a(setup --scenario-index): a different --scenario-index against an already-prepared root is refused, naming both paths -- PASS"

# ===========================================================================
# cmd_demo must still work: per spec.md §2.1.2(iii) it synthesizes a `setup`
# namespace internally and calls cmd_setup directly, so that namespace must
# carry a scenario_index attribute or every `demo` invocation raises
# AttributeError.
# ===========================================================================
DEMO_ROOT="$WORKDIR/demo-setup"
demo_out="$WORKDIR/demo.out"
demo_rc=0
timeout 120 "$OPERATOR" demo --out "$DEMO_ROOT" --scenario py-bug-due-date-boundary \
	>"$demo_out" 2>&1 || demo_rc=$?
grep -qi "AttributeError" "$demo_out" &&
	fail "demo raised AttributeError (scenario_index missing from the synthesized setup namespace): $(cat "$demo_out")"
grep -qi "^Traceback" "$demo_out" &&
	fail "demo raised an unexpected exception: $(cat "$demo_out")"
# T-E40-F11-010 (AC-F11-09): `demo` wraps a real `setup` + `preflight`; this
# scaffold's synthesized config configures no runtime adapter/i05_bundle_dir,
# so the fail-closed gate correctly reports status=="blocked" (exit 1) --
# what this test actually needs is that cmd_demo's internal setup namespace
# is well-formed (no AttributeError/Traceback above), not that preflight
# reaches "pass".
[[ "$demo_rc" -eq 1 ]] || fail "demo exited $demo_rc, want 1 (blocked): $(cat "$demo_out")"
echo "TC-111a(cmd_demo): demo's synthesized setup namespace carries scenario_index, no AttributeError -- PASS"

echo "TC-111a: scenario-root isolation foundation (keyed setup + --scenario-index) -- ALL PASS"
