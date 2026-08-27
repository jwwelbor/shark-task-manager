#!/usr/bin/env bash
# TC-095 (B057): verify-lifecycle-run.sh must accept the candidate.identity_digest
# that run-lifecycle.sh actually produces. Before the fix, run-lifecycle.sh's
# refresh_candidate() folded scratch_content_digest into identity_digest (7
# components) while verify-lifecycle-run.sh's expected_identity recomputed
# over only the original six identity components, so every real run failed
# identity_mismatch.
#
# This builds a minimal, schema-valid lifecycle record by hand (rather than
# driving the full dispatch loop) so the assertion is isolated to the
# identity_digest contract and is not tripped up by unrelated
# producer/validator gaps elsewhere in the record. The candidate object
# itself is produced by executing run-lifecycle.sh's own
# candidate_identity()/refresh_candidate() functions (extracted from the real
# file, not reimplemented), so this exercises the actual producer code.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
VERIFIER="$SCRIPTS_DIR/verify-lifecycle-run.sh"
SCHEMA="$SCRIPTS_DIR/../runs/i07-schema.yaml"
fail() { echo "TC-095 FAIL: $1" >&2; exit 1; }
[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable"
[[ -x "$VERIFIER" ]] || fail "verify-lifecycle-run.sh missing or not executable"
[[ -f "$SCHEMA" ]] || fail "i07-schema.yaml missing"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/scratch"
printf 'hello' >"$WORKDIR/scratch/file.txt"

RECORD="$WORKDIR/lifecycle.jsonl"

python3 - "$RUNNER" "$REPO_ROOT" "$WORKDIR/scratch" "$RECORD" <<'PY'
import json
import re
import sys

runner_path, repo_root, scratch, record_path = sys.argv[1:5]

# Extract the embedded python body (between the `python3 - "$@" <<'PY'` heredoc
# markers) up to the trailing `try: raise SystemExit(main(...))` driver, so
# importing it defines candidate_identity()/refresh_candidate() without
# running main() or requiring a --scenario/--root CLI invocation.
source = open(runner_path, encoding="utf-8").read()
match = re.search(r"<<'PY'\n(.*)\ntry:\n    raise SystemExit\(main", source, re.DOTALL)
assert match, "could not locate run-lifecycle.sh's embedded python body"
namespace = {"__name__": "run_lifecycle_under_test"}
exec(compile(match.group(1), runner_path, "exec"), namespace)

candidate = namespace["candidate_identity"](repo_root)
namespace["refresh_candidate"](candidate, scratch)
assert "scratch_content_digest" in candidate, "refresh_candidate did not add scratch_content_digest"

digest64 = lambda seed: __import__("hashlib").sha256(seed.encode()).hexdigest()

record = {
    "identity": {
        "schema_version": "1.0",
        "run_id": "tc095",
        "scenario_id": "tc095-scenario",
        "scenario_version": "1",
        "fixture_id": "tc095-fixture",
        "fixture_digest": digest64("fixture"),
        "adapter_id": "tc095-adapter",
        "adapter_version": "1.0.0",
        "shark_binary_digest": digest64("binary"),
        "shark_content_digest": digest64("content"),
        "roots": {
            "agent_fixture_checkout": "/tmp/checkout",
            "scratch_shark_project": scratch,
            "evaluator_only": "/tmp/evaluator",
        },
    },
    "entity_graph": {
        "root_key": "X",
        "root_type": "task",
        "resolved_via": "scenario_root",
        "fork_candidates": [],
        "selected_keys": [],
        "selected_types": [],
        "ordinals": [1],
        "ineligible": [],
    },
    "dispatches": [
        {
            "ordinal": 1,
            "requested_key": "X",
            "response": {
                "entity_key": "X",
                "entity_type": "task",
                "status": "in_development",
                "action": "spawn_agent",
                "agent_type": "dev",
                "provider": "fixture",
                "model": "fixture",
                "prompt_sha256": digest64("prompt"),
                "prompt_bytes": 10,
            },
            "claim": {"session_id": "SID-095"},
            "worker": {
                "worker_id": "worker-095",
                "session_id": "SID-095",
                "kind": "final",
                "evidence": {},
            },
            "heartbeats": [],
            "outcome": "pass",
            "transition": {},
            "release": {},
            "started_at": "2026-01-01T00:00:00.000Z",
            "ended_at": "2026-01-01T00:00:01.000Z",
            "evidence_refs": {
                "prompt_sha256": digest64("prompt"),
                "prompt_bytes": 10,
                "candidate_snapshot_digest": candidate["snapshot_digest"],
            },
        }
    ],
    "stages": [
        {
            "dispatch_ordinal": 1,
            "stage": "code",
            "category": "code",
            "snapshot_digest": digest64("stage-snapshot"),
            "prompt_digest": digest64("stage-prompt"),
            "input_lineage": [],
            "replay_lineage": [],
            "output_paths": [],
            "output_digests": [],
            "usage": {"model": "fixture", "provider": "fixture"},
            "cost_usd": 0.0,
            "elapsed_seconds": 0.1,
            "errors": [],
            "rework": False,
            "intervals": [],
            "candidate": candidate,
            "artifacts": [],
            "access_events": [],
            "evidence_refs": {"candidate_snapshot_digest": candidate["snapshot_digest"]},
        }
    ],
    "workflow_policy": {
        "enabled_gates": [],
        "gate_order": [],
        "reviewer": {"provider": "fixture", "model": "fixture"},
        "prompt_digest": digest64("policy-prompt"),
        "review_bundle_digest": digest64("policy-bundle"),
        "fixes_allowed_between_gates": False,
    },
    "review_gates": [],
    "questions": [],
    "limits": {
        "max_cost_usd": 5.0,
        "max_wall_clock_seconds": 600.0,
        "max_generated_tasks": 20,
        "observed_cost_usd": 0.0,
        "observed_wall_clock_seconds": 0.1,
        "observed_generated_tasks": 0,
        "first_exceeded": None,
    },
    "outcome": {
        "terminal": "complete",
        "reason": "tc095 synthetic fixture",
        "partial_evidence": True,
        "publication_eligible": True,
    },
}

with open(record_path, "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY

"$VERIFIER" "$RECORD" --schema "$SCHEMA" \
    || fail "verify-lifecycle-run.sh rejected a candidate.identity_digest actually produced by run-lifecycle.sh's own candidate_identity()/refresh_candidate()"

echo "TC-095: pass (verify-lifecycle-run.sh accepts run-lifecycle.sh's own candidate.identity_digest)"

# Negative case: a candidate whose identity_digest disagrees with its named
# identity fields must still be rejected. Recomputing "everything but the
# digest keys" (the pre-fix formula) can never fail this way since it always
# matches whatever fields the record happens to carry — this proves the
# check compares against the fixed field list, not just itself.
TAMPERED="$WORKDIR/lifecycle-tampered.jsonl"
python3 - "$RECORD" "$TAMPERED" <<'PY'
import json
import sys

record_path, tampered_path = sys.argv[1:3]
with open(record_path, encoding="utf-8") as stream:
    record = json.loads(stream.readline())

candidate = record["stages"][0]["candidate"]
candidate["test_suite_digest"] = "0" * 64  # identity_digest is now stale

with open(tampered_path, "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY

if "$VERIFIER" "$TAMPERED" --schema "$SCHEMA" >/dev/null 2>&1; then
    fail "verify-lifecycle-run.sh accepted a candidate whose identity_digest disagrees with its identity fields"
fi

echo "TC-095: pass (verify-lifecycle-run.sh rejects a candidate whose identity_digest disagrees with its identity fields)"
