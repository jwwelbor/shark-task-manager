#!/usr/bin/env bash
# run-one.sh --item <id> --variant <id> --rep <n> --timeout <seconds> \
#            --out <artifact_root> [--corpus <corpus.yaml>] [--keep-scratch] \
#            [--skip-canary]
#
# The E40-F02 driver (ADR-F02-01): provisions a throwaway scratch shark
# project, seeds one corpus item's entity into it, invokes `shark run` under
# a process-group timeout cap, captures the raw run artifacts, and hands off
# to collect-run.sh (T-E40-F02-001/002) to produce record.jsonl. Never
# prompts; a single invocation drives exactly one (item, variant, rep).
#
# THIS TASK'S SLICE (T-E40-F02-003): provisioning, seeding, timeout
# invocation, and the collector hand-off. The pinned post-run pipeline
# (toolchain guard, LOC, quality gates, F2P injection -- REQ-F-015/016/017,
# ADR-F02-11) is T-E40-F02-004's extension; until that lands, no post/ files
# are written, so a record's oracle/quality/loc blocks stay absent (never
# fabricated) -- collect-run.sh's own contract already treats all of post/
# as optional for exactly this reason.
#
# Phase markers (REQ-N-006 observability; asserted by TC-014a): exactly one
# "run-one: phase=<name>" line to stderr, in order, per phase --
# provision (scratch-env + workflow_config repoint + transcript capture +
# fixture checkout), seed (entity creation), invoke (the timeout-capped
# `shark run`), postrun (artifact copy + meta.json + collect-run.sh).
#
# Testability (task spec AC bullet): `shark` is resolved via
# ${SHARK_BIN:-shark} -- PATH by default, or a single overridable variable
# -- NEVER a hardcoded path to the shark-scratch-env.sh-copied binary, so
# TC-014's PATH-stub can substitute it for every create/run invocation made
# AFTER provisioning. Provisioning itself (scripts/shark-scratch-env.sh)
# always uses the real repo-root binary internally and is not stubbable by
# design (test-plan.md's Caller-Path Contract) -- that's what proves
# run-one.sh never invokes shark project-initialisation commands itself
# (REQ-F-002). The X-07 canary (REQ-F-020) is resolved the same way:
# ${CANARY_BIN:-canary-runsurface.sh} via PATH, so TC-014g can substitute it
# too, ahead of T-E40-F02-005 actually building the real script.
#
# REQ-N-002/Safety: never invokes shark project-initialisation commands
# directly (only scripts/shark-scratch-env.sh does that, with its own
# always-/tmp scratch directory), and never writes to the live repository,
# its .sharkconfig.json, or the live database. Every SHARK_BIN invocation
# after provisioning runs with cwd inside the scratch project directory
# (never the live repo), because shark's project-root auto-detection
# (FindProjectRoot) walks up from cwd and ignores --config/--db for that
# walk.
#
# ADR-F02-03: the `shark run` invocation runs in its own session/process
# group (setsid); cap expiry signals the WHOLE group (SIGTERM, then SIGKILL
# after a grace period) so no agent subprocess (`claude`, a child of
# `shark`) survives the cap. On that path shark's own run_end summary line
# never exists (run.go's deferred Finish() never executes under SIGKILL);
# collect-run.sh already reads the per-event-appended stderr/run.log
# instead of requiring it.
#
# REQ-F-018: the output path is deterministic
# (<out>/<item>/<variant>/rep-<rep>/record.jsonl). A second invocation into
# an already-completed path refuses (non-zero, naming the path) rather than
# silently overwriting.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

SHARK_BIN="${SHARK_BIN:-shark}"
CANARY_BIN="${CANARY_BIN:-canary-runsurface.sh}"
KILL_GRACE_S="${RUN_ONE_KILL_GRACE_S:-10}"

usage() {
	cat >&2 <<'EOF'
usage: run-one.sh --item <id> --variant <id> --rep <n> --timeout <seconds> \
                   --out <artifact_root> [--corpus <corpus.yaml>] \
                   [--keep-scratch] [--skip-canary]
EOF
	exit 2
}

item_id=""
variant_id=""
rep=""
timeout_s=""
out_root=""
corpus_yaml=""
keep_scratch="false"
skip_canary="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--item)
		[[ $# -ge 2 ]] || usage
		item_id="$2"
		shift 2
		;;
	--variant)
		[[ $# -ge 2 ]] || usage
		variant_id="$2"
		shift 2
		;;
	--rep)
		[[ $# -ge 2 ]] || usage
		rep="$2"
		shift 2
		;;
	--timeout)
		[[ $# -ge 2 ]] || usage
		timeout_s="$2"
		shift 2
		;;
	--out)
		[[ $# -ge 2 ]] || usage
		out_root="$2"
		shift 2
		;;
	--corpus)
		[[ $# -ge 2 ]] || usage
		corpus_yaml="$2"
		shift 2
		;;
	--keep-scratch)
		keep_scratch="true"
		shift
		;;
	--skip-canary)
		skip_canary="true"
		shift
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$item_id" && -n "$variant_id" && -n "$rep" && -n "$timeout_s" && -n "$out_root" ]] || usage
corpus_yaml="${corpus_yaml:-$BENCH_DIR/corpus/corpus.yaml}"

command -v python3 >/dev/null 2>&1 || {
	echo "run-one: python3 not found on PATH" >&2
	exit 1
}
command -v setsid >/dev/null 2>&1 || {
	echo "run-one: setsid not found on PATH (required for process-group teardown, ADR-F02-03)" >&2
	exit 1
}
command -v "$SHARK_BIN" >/dev/null 2>&1 || {
	echo "run-one: shark binary not found on PATH (resolved as '$SHARK_BIN'); build/install it or set SHARK_BIN" >&2
	exit 1
}
[[ -f "$corpus_yaml" ]] || {
	echo "run-one: corpus yaml not found: $corpus_yaml" >&2
	exit 1
}
[[ -x "$SCRIPT_DIR/checkout-fixture.sh" ]] || {
	echo "run-one: checkout-fixture.sh missing or not executable" >&2
	exit 1
}
[[ -x "$SCRIPT_DIR/collect-run.sh" ]] || {
	echo "run-one: collect-run.sh missing or not executable" >&2
	exit 1
}

# --- REQ-F-018: deterministic, self-identifying output path -----------------
run_dir="$out_root/$item_id/$variant_id/rep-$rep"
record_path="$run_dir/record.jsonl"

if [[ -f "$record_path" ]]; then
	echo "run-one: record already exists at $record_path; refusing to overwrite (REQ-F-018)" >&2
	exit 1
fi

# --- corpus + seed lookup (I-01: reads corpus.yaml directly, per the shape
# bench/README.md documents; never re-derives the schema) -------------------
readarray -d '' -t item_fields < <(python3 - "$corpus_yaml" "$item_id" <<'PYEOF'
import os
import sys

import yaml

corpus_path, item_id = sys.argv[1], sys.argv[2]
with open(corpus_path) as f:
    data = yaml.safe_load(f)

item = None
for it in data.get("items") or []:
    if it["id"] == item_id:
        item = it
        break
if item is None:
    sys.stderr.write("run-one: item not found in corpus.yaml items: %s\n" % item_id)
    sys.exit(1)

corpus_dir = os.path.dirname(os.path.abspath(corpus_path))
seed_path = os.path.join(corpus_dir, item["seed_path"])
for field in (item["type"], item["fixture_base_sha"], seed_path):
    sys.stdout.write(field)
    sys.stdout.write("\0")
PYEOF
)
[[ "${#item_fields[@]}" -eq 3 ]] || {
	echo "run-one: failed to resolve corpus item $item_id from $corpus_yaml" >&2
	exit 1
}
item_type="${item_fields[0]}"
fixture_base_sha="${item_fields[1]}"
seed_path="${item_fields[2]}"

readarray -d '' -t seed_fields < <(python3 - "$seed_path" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    seed = yaml.safe_load(f)
for field in (seed.get("title", ""), seed.get("description", ""), seed.get("severity", "")):
    sys.stdout.write(field)
    sys.stdout.write("\0")
PYEOF
)
[[ "${#seed_fields[@]}" -eq 3 ]] || {
	echo "run-one: failed to parse seed file: $seed_path" >&2
	exit 1
}
seed_title="${seed_fields[0]}"
seed_description="${seed_fields[1]}"
seed_severity="${seed_fields[2]}"

if [[ "$item_type" != "task" && "$item_type" != "bug" ]]; then
	echo "run-one: unsupported item type '$item_type' for $item_id (only task/bug are benched, ADR-005)" >&2
	exit 1
fi

# --- REQ-F-020/AC-21: X-07 canary preflight, before any provisioning -------
if [[ "$skip_canary" != "true" ]]; then
	command -v "$CANARY_BIN" >/dev/null 2>&1 || {
		echo "run-one: X-07 canary preflight binary not found on PATH: $CANARY_BIN (pass --skip-canary to bypass, or set CANARY_BIN)" >&2
		exit 1
	}
	canary_args=("$CANARY_BIN" --corpus "$corpus_yaml")
	echo "run-one: running X-07 canary preflight ($CANARY_BIN)" >&2
	if ! "${canary_args[@]}"; then
		echo "run-one: X-07 canary preflight failed; aborting before provisioning (REQ-F-020)" >&2
		exit 1
	fi
fi

# --- phase=provision: scratch project, workflow repoint, transcripts,
# fixture checkout ------------------------------------------------------------
echo "run-one: phase=provision" >&2

scratch_dir="$("$REPO_ROOT/scripts/shark-scratch-env.sh" "bench-$item_id")"
[[ -n "$scratch_dir" && -d "$scratch_dir" ]] || {
	echo "run-one: scripts/shark-scratch-env.sh did not print a usable scratch directory" >&2
	exit 1
}

cleanup_scratch() {
	if [[ "$keep_scratch" != "true" && -d "$scratch_dir" ]]; then
		rm -rf "$scratch_dir"
	fi
}
trap cleanup_scratch EXIT

shark_in_scratch() {
	(cd "$scratch_dir" && "$SHARK_BIN" "$@" </dev/null)
}

# REQ-F-003: admin init defaults to the embedded bundle (workflow_config
# unset) and transcripts off -- neither is acceptable, neither is assumed.
# install-shark-data materializes shark-data/workflow/ and writes
# workflow_config into .sharkconfig.json; Phase 1's only variant is
# "default" (spec.md "Out of scope"), so every variant currently repoints
# at the same embedded bundle -- this is the repoint mechanism, not variant
# bundle authoring.
shark_in_scratch admin install-shark-data --json >/dev/null || {
	echo "run-one: 'shark admin install-shark-data' failed in the scratch project" >&2
	exit 1
}

python3 - "$scratch_dir/.sharkconfig.json" <<'PYEOF'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    cfg = json.load(f)
obs = cfg.setdefault("observability", {})
obs["capture_agent_transcripts"] = True
with open(path, "w") as f:
    json.dump(cfg, f, indent=2, sort_keys=True)
PYEOF

checkout_dir="$scratch_dir/checkout"
"$SCRIPT_DIR/checkout-fixture.sh" "$fixture_base_sha" "$checkout_dir" || {
	echo "run-one: checkout-fixture.sh failed for base_sha=$fixture_base_sha" >&2
	exit 1
}

# --- phase=seed: ADR-F02-08 -- create the host epic+feature for a task item
# (bug items are standalone); every key captured from --json, never chosen
# by the driver -------------------------------------------------------------
echo "run-one: phase=seed" >&2

run_create_key() {
	# $1 = entity type ("epic"/"feature"/"task"/"bug"), remaining args are
	# passed through to `shark create <type> ...`. Returns just the
	# assigned .key from the --json response -- REQ-N-005: a missing key is
	# never silently invented from the driver's own input.
	local out
	out="$(shark_in_scratch create "$@" --json)" || {
		echo "run-one: 'shark create $*' failed in the scratch project" >&2
		exit 1
	}
	python3 -c 'import json,sys; print(json.loads(sys.argv[1])["key"])' "$out"
}

seeded_entity_key=""
seeded_keys_json=""

if [[ "$item_type" == "task" ]]; then
	seeded_epic_key="$(run_create_key epic "shark-bench host epic: $item_id")"
	seeded_feature_key="$(run_create_key feature "$seeded_epic_key" "shark-bench host feature: $item_id")"
	seeded_entity_key="$(run_create_key task "$seeded_epic_key" "$seeded_feature_key" "$seed_title" --description="$seed_description")"
	seeded_keys_json="$(python3 -c 'import json,sys; print(json.dumps({"epic": sys.argv[1], "feature": sys.argv[2], "task": sys.argv[3]}))' \
		"$seeded_epic_key" "$seeded_feature_key" "$seeded_entity_key")"
else
	seeded_entity_key="$(run_create_key bug "$seed_title" --description="$seed_description" --severity="$seed_severity")"
	seeded_keys_json="$(python3 -c 'import json,sys; print(json.dumps({"bug": sys.argv[1]}))' "$seeded_entity_key")"
fi

# --- phase=invoke: process-group-capped `shark run` -------------------------
echo "run-one: phase=invoke" >&2

mkdir -p "$run_dir/run"
stdout_file="$run_dir/run/stdout.json"
stderr_file="$run_dir/run/stderr.ndjson"

t0_ns="$(date +%s%N)"

(
	cd "$scratch_dir"
	exec setsid "$SHARK_BIN" run "$seeded_entity_key" --json --workdir "$checkout_dir"
) >"$stdout_file" 2>"$stderr_file" </dev/null &
pid=$!
# setsid succeeds without forking when the calling process is not already a
# process group leader (true here: a background subshell of a
# non-interactive script), so pid IS normally the setsid'd process's final
# pid, which becomes its own new session/process group leader (pgid ==
# pid). Verified rather than trusted blindly: a silent mismatch would send
# kill -TERM/-KILL -$pgid at a stale or wrong group, leaving a live agent
# process consuming API budget after the cap (REQ-N-003). An empty lookup
# means the process already exited (a very fast completion) -- nothing left
# to kill, so pid is a safe fallback.
pgid="$pid"
observed_pgid="$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ')" || true
if [[ -n "$observed_pgid" ]]; then
	pgid="$observed_pgid"
fi

timed_out_marker="$(mktemp -u "${TMPDIR:-/tmp}/run-one-timedout-XXXXXX")"
(
	sleep "$timeout_s"
	: >"$timed_out_marker"
	kill -TERM -"$pgid" 2>/dev/null || true
	sleep "$KILL_GRACE_S"
	kill -KILL -"$pgid" 2>/dev/null || true
) &
watchdog_pid=$!

set +e
wait "$pid"
wait_status=$?
set -e

# Cancel the watchdog if the invocation finished before the cap fired.
kill "$watchdog_pid" 2>/dev/null || true
wait "$watchdog_pid" 2>/dev/null || true

t1_ns="$(date +%s%N)"
harness_wall_ns=$((t1_ns - t0_ns))

timed_out="false"
if [[ -f "$timed_out_marker" ]]; then
	timed_out="true"
fi
rm -f "$timed_out_marker"

exit_code=""
signaled="false"
if [[ "$wait_status" -ge 128 ]]; then
	signaled="true"
else
	exit_code="$wait_status"
fi

# --- phase=postrun: artifact copy, meta.json, collector hand-off -----------
# (T-E40-F02-004 extends this phase with the pinned toolchain-guard/LOC/
# quality-gate/F2P-injection pipeline, ADR-F02-11; none of that runs yet.)
echo "run-one: phase=postrun" >&2

resolve_run_id() {
	# Mirrors collect-run.sh's own run_id resolution (REQ-F-008/ADR-F02-02)
	# closely enough for artifact copying: the liveness announce line first,
	# falling back to the newest directory under scratch_root/.shark/runs/.
	# The record's OWN run_id is resolved independently by collect-run.sh
	# from this same captured stderr file, so this copy never becomes the
	# record's source of truth.
	local stderr_path="$1" scratch_root="$2"
	local announce
	announce="$(grep -m1 '^run\.log: ' "$stderr_path" 2>/dev/null || true)"
	if [[ -n "$announce" ]]; then
		local p="${announce#run.log: }"
		basename "$(dirname "$p")"
		return 0
	fi
	local runs_dir="$scratch_root/.shark/runs"
	if [[ -d "$runs_dir" ]]; then
		ls -1t "$runs_dir" 2>/dev/null | head -n1
	fi
}

run_id="$(resolve_run_id "$stderr_file" "$scratch_dir")"
if [[ -n "$run_id" && -d "$scratch_dir/.shark/runs/$run_id" ]]; then
	src="$scratch_dir/.shark/runs/$run_id"
	if [[ -f "$src/run.log" ]]; then
		cp "$src/run.log" "$run_dir/run/run.log"
	fi
	shopt -s nullglob
	transcripts=("$src"/*.log)
	shopt -u nullglob
	if [[ "${#transcripts[@]}" -gt 0 ]]; then
		mkdir -p "$run_dir/run/transcripts"
		for f in "${transcripts[@]}"; do
			[[ "$(basename "$f")" == "run.log" ]] && continue
			cp "$f" "$run_dir/run/transcripts/"
		done
	fi
fi

python3 - "$run_dir/run/exit_status" "$exit_code" "$signaled" "$timed_out" <<'PYEOF'
import json
import sys

path, exit_code, signaled, timed_out = sys.argv[1:5]
obj = {
    "exit_code": (int(exit_code) if exit_code != "" else None),
    "signaled": signaled == "true",
    "timed_out": timed_out == "true",
}
with open(path, "w") as f:
    json.dump(obj, f, indent=2, sort_keys=True)
PYEOF

python3 - "$run_dir/meta.json" "$item_id" "$item_type" "$variant_id" "$rep" "$timeout_s" \
	"$harness_wall_ns" "$seeded_entity_key" "$scratch_dir" "$scratch_dir/shark-tasks.db" \
	"$seeded_keys_json" "$skip_canary" "$t0_ns" "$t1_ns" <<'PYEOF'
import json
import sys

(meta_path, item_id, item_type, variant_id, rep, timeout_s,
 harness_wall_ns, entity_key, scratch_root, scratch_db_path,
 seeded_keys_json, skip_canary, t0_ns, t1_ns) = sys.argv[1:15]

meta = {
    "item_id": item_id,
    "item_type": item_type,
    "variant_id": variant_id,
    "rep": int(rep),
    "timeout_cap_s": int(timeout_s),
    "harness_wall_ns": int(harness_wall_ns),
    "entity_key": entity_key,
    "scratch_root": scratch_root,
    "scratch_db_path": scratch_db_path,
    "seeded_keys": json.loads(seeded_keys_json),
    "skip_canary": skip_canary == "true",
    "t0_ns": int(t0_ns),
    "t1_ns": int(t1_ns),
}
with open(meta_path, "w") as f:
    json.dump(meta, f, indent=2, sort_keys=True)
PYEOF

"$SCRIPT_DIR/collect-run.sh" --run-dir "$run_dir" >"$record_path.tmp" || {
	echo "run-one: collect-run.sh failed for $run_dir" >&2
	rm -f "$record_path.tmp"
	exit 1
}
mv "$record_path.tmp" "$record_path"

echo "run-one: record written to $record_path" >&2
