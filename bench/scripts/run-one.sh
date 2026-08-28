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
# T-E40-F02-003 built provisioning, seeding, and the timeout invocation.
# T-E40-F02-004 adds the pinned post-run pipeline (ADR-F02-11, REQ-F-015/
# 016/017): toolchain guard, THEN LOC, THEN quality gates + ledger diff,
# THEN F2P injection + the oracle, LAST -- never reordered. The pipeline
# runs whenever the invocation did NOT time out (completed/failed/paused/
# already_terminal/no_action all get a real, settled checkout worth
# measuring); a timed-out run skips it entirely, matching the existing
# timeout fixtures (bench/scripts/testdata/run/timeout-*), which carry no
# post/ directory at all -- collect-run.sh already treats a run directory
# with no post/toolchain-guard.json as "no post-run phase ran".
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
CANARY_BIN="${CANARY_BIN:-$SCRIPT_DIR/canary-runsurface.sh}"
BUILD_LEDGERS_BIN="${BUILD_LEDGERS_BIN:-$SCRIPT_DIR/build-ledgers.sh}"
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
command -v sha256sum >/dev/null 2>&1 || {
	echo "run-one: sha256sum not found on PATH (required for the G7 manifest identity fields)" >&2
	exit 1
}
command -v go >/dev/null 2>&1 || {
	echo "run-one: go toolchain not found on PATH (required by the post-run pipeline, T-E40-F02-004)" >&2
	exit 1
}
command -v golangci-lint >/dev/null 2>&1 || {
	echo "run-one: golangci-lint not found on PATH (required by the post-run pipeline, T-E40-F02-004)" >&2
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

# --- G7 reproducibility (architecture.md#metric-collection-and-artifact-schema,
# spec.md's manifest.shark_version/manifest.shark_binary_sha256, ADR-002 "no
# state outside the artifact directory"): identity of the shark binary that
# actually resolves and runs every SHARK_BIN invocation below. The file hash
# is captured here (no shark subprocess involved -- filesystem-only, so it
# has no cwd to police under AC-11). The `--version` string itself is a real
# SHARK_BIN invocation, deferred until after provisioning (below) so it runs
# with cwd inside the scratch project like every other invocation, never the
# live repo (REQ-N-002/AC-11) -- $shark_version is set there.
# Under the harness's own PATH-stub test seam, this pairing is inexact: the
# hash covers the resolved stub script while a stubbed `--version` forwards
# to the real binary, so the two fields describe the same binary only for
# real (non-stubbed) runs -- the pairing this field exists to pin.
shark_bin_resolved="$(command -v "$SHARK_BIN")"
shark_binary_sha256="$(sha256sum "$shark_bin_resolved" | awk '{print $1}')"
shark_version=""

# --- REQ-F-018: deterministic, self-identifying output path -----------------
run_dir="$out_root/$item_id/$variant_id/rep-$rep"
record_path="$run_dir/record.jsonl"

if [[ -e "$run_dir" ]] && { compgen -G "$run_dir/*" >/dev/null || compgen -G "$run_dir/.[!.]*" >/dev/null; }; then
	echo "run-one: run directory already contains artifacts at $run_dir (record target $record_path); refusing to mix a rerun with an incomplete prior attempt (REQ-F-018)" >&2
	exit 1
fi

# --- corpus + seed lookup (I-01: reads corpus.yaml directly, per the shape
# bench/README.md documents; never re-derives the schema) -------------------
readarray -d '' -t item_fields < <(python3 - "$corpus_yaml" "$item_id" <<'PYEOF'
import json
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
# f2p is carried as one JSON-encoded field (its paths[]/test_names[] are
# variable-length lists) -- T-E40-F02-004's post-run F2P injection step
# reads it back without a second corpus.yaml parse.
f2p_json = json.dumps(item["f2p"], sort_keys=True)
# corpus_schema_version (top-level) and p2p_set (item-level) are G7
# manifest fields (architecture.md#metric-collection-and-artifact-schema,
# spec.md's manifest.corpus_schema_version/manifest.p2p_set) -- read here
# alongside the rest of this same corpus.yaml parse rather than a second one.
for field in (item["type"], item["fixture_base_sha"], seed_path, f2p_json, data["schema_version"], item["p2p_set"]):
    sys.stdout.write(field)
    sys.stdout.write("\0")
PYEOF
)
[[ "${#item_fields[@]}" -eq 6 ]] || {
	echo "run-one: failed to resolve corpus item $item_id from $corpus_yaml" >&2
	exit 1
}
item_type="${item_fields[0]}"
fixture_base_sha="${item_fields[1]}"
seed_path="${item_fields[2]}"
item_f2p_json="${item_fields[3]}"
corpus_schema_version="${item_fields[4]}"
item_p2p_set="${item_fields[5]}"

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
		echo "run-one: X-07 canary preflight binary not found: $CANARY_BIN (pass --skip-canary to bypass, or set CANARY_BIN)" >&2
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


# EXIT traps are inherited before a newly forked child executes its first
# statement. A child can therefore be cancelled before an in-child
# `trap - EXIT` reset runs. Bind destructive cleanup to this top-level Bash
# process so no inherited trap can remove scratch while its parent still
# needs the checkout for post-run evidence.
cleanup_owner_pid="$BASHPID"
cleanup_scratch() {
	[[ "$BASHPID" == "$cleanup_owner_pid" ]] || return 0
	if [[ "$keep_scratch" != "true" && -d "$scratch_dir" ]]; then
		rm -rf "$scratch_dir"
	fi
}
trap cleanup_scratch EXIT

shark_in_scratch() {
	(cd "$scratch_dir" && "$SHARK_BIN" "$@" </dev/null)
}

# G7 reproducibility: the `--version` half of the shark binary identity
# (shark_binary_sha256 was already captured above, file-only, no cwd). Run
# through shark_in_scratch like every other invocation after this point, so
# its cwd is inside the scratch project, never the live repo (REQ-N-002/AC-11).
shark_version="$(shark_in_scratch --version 2>&1)" || {
	echo "run-one: '$SHARK_BIN --version' failed" >&2
	exit 1
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

# G7 reproducibility (spec.md's manifest.variant_bundle_sha256): a content
# hash over the just-installed workflow bundle, sorted by path, so a later
# replay can confirm which bundle content (not merely which file layout)
# produced this run without needing shark-data/ itself to survive (ADR-002:
# "no state outside the artifact directory" -- the hash is what's pinned in
# the artifact, not the bundle tree).
variant_bundle_sha256="$(python3 - "$scratch_dir/shark-data/workflow" <<'PYEOF'
import hashlib
import os
import sys

bundle_dir = sys.argv[1]
if not os.path.isdir(bundle_dir):
    sys.exit("run-one: installed workflow bundle directory not found: %s" % bundle_dir)

h = hashlib.sha256()
for root, dirs, files in os.walk(bundle_dir):
    dirs.sort()
    for name in sorted(files):
        path = os.path.join(root, name)
        rel = os.path.relpath(path, bundle_dir)
        with open(path, "rb") as f:
            content = f.read()
        h.update(rel.encode("utf-8"))
        h.update(b"\0")
        h.update(content)
        h.update(b"\0")
print(h.hexdigest())
PYEOF
)" || {
	echo "run-one: failed to hash the installed workflow bundle" >&2
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

# The generated Shark task documents live under the scratch project, but the
# agent runs in the fixture checkout passed through --workdir. Mirror only the
# planning tree before dispatch so relative entity file_path references work.
# This is harness context, not fixture code or oracle input; F2P files remain
# absent until the existing post-run injection step.
mkdir -p "$checkout_dir/docs"
cp -a "$scratch_dir/docs/plan" "$checkout_dir/docs/"

# --- phase=invoke: process-group-capped `shark run` -------------------------
echo "run-one: phase=invoke" >&2

mkdir -p "$run_dir/run"
stdout_file="$run_dir/run/stdout.json"
stderr_file="$run_dir/run/stderr.ndjson"

t0_ns="$(date +%s%N)"

(
	# This child must never own the parent-only scratch cleanup. Bash child
	# shells inherit EXIT traps; if cd/exec fails before setsid replaces this
	# shell, the inherited trap would otherwise remove scratch while the
	# parent is still collecting the failed invocation.
	trap - EXIT
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
	# The watchdog is a child shell, not a cleanup owner. Without this reset,
	# cancelling a live watchdog runs the inherited cleanup_scratch EXIT trap
	# and races the parent's post-run reads of the checkout.
	trap - EXIT
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

# --- phase=postrun: artifact copy, meta.json, pinned post-run pipeline,
# collector hand-off ----------------------------------------------------
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
	# Transcripts now live one directory level deeper, under the
	# entity key that produced them ($src/<entityKey>/*.log -- B052
	# entity-scoped path fix), so a flat "$src"/*.log glob matches
	# nothing. Discover recursively; -mindepth 2 already excludes the
	# top-level run.log without a separate basename check (LivenessRecorder
	# writes exactly one run.log per RunID, directly under $src, never
	# nested under an entity directory -- internal/runner/liveness.go's
	# resolveLogPath keys only on runID).
	#
	# Deliberate deviation from B052's code-review F1 note: the reviewer's
	# suggested full repair also nests the COPY DESTINATION by entity key
	# and re-keys collect-run.sh's reconcile_transcripts on
	# (entity_key, idx) to avoid two same-named sibling transcripts
	# colliding at a flat destination. That is not done here -- destination
	# stays flat, matching the (idx)-only keying reconcile_transcripts
	# already has. This is safe today and unreachable-by-construction, not
	# an oversight: docs/plan/bugs/B052.md's own scope note says
	# meta.json.item_type is restricted to task/bug in Phase 1, so no
	# cascade action ever fires and at most one entity directory exists per
	# run -- the same reachability argument the review's own F5 makes for
	# collect-run.sh's sibling `iteration`-keying gap. The nested-
	# destination + (entity_key, idx) rework is deferred to Phase 2
	# cascade benching, alongside F5, since both need the same keying
	# change made once rather than twice.
	readarray -d '' -t transcripts < <(find "$src" -mindepth 2 -type f -name '*.log' -print0 2>/dev/null)
	if [[ "${#transcripts[@]}" -gt 0 ]]; then
		mkdir -p "$run_dir/run/transcripts"
		for f in "${transcripts[@]}"; do
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
	"$seeded_keys_json" "$skip_canary" "$t0_ns" "$t1_ns" \
	"$fixture_base_sha" "$corpus_schema_version" "$item_p2p_set" "$variant_bundle_sha256" \
	"$shark_version" "$shark_binary_sha256" <<'PYEOF'
import json
import sys

(meta_path, item_id, item_type, variant_id, rep, timeout_s,
 harness_wall_ns, entity_key, scratch_root, scratch_db_path,
 seeded_keys_json, skip_canary, t0_ns, t1_ns,
 fixture_base_sha, corpus_schema_version, p2p_set, variant_bundle_sha256,
 shark_version, shark_binary_sha256) = sys.argv[1:21]

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
    # G7 reproducibility (architecture.md#metric-collection-and-artifact-schema,
    # E40-interaction-map.md I-02 row, uat-plan.md UAT-07, ADR-002) -- pinned
    # here so a later replay needs no state outside this artifact directory.
    "fixture_base_sha": fixture_base_sha,
    "corpus_schema_version": corpus_schema_version,
    "p2p_set": p2p_set,
    "variant_bundle_sha256": variant_bundle_sha256,
    "shark_version": shark_version,
    "shark_binary_sha256": shark_binary_sha256,
}
with open(meta_path, "w") as f:
    json.dump(meta, f, indent=2, sort_keys=True)
PYEOF

# --- pinned post-run pipeline (T-E40-F02-004, ADR-F02-11/REQ-F-015/016/017)
# Order is exactly: (1) toolchain guard, (2) LOC, (3) quality gates + ledger
# diff, (4) F2P injection + oracle -- never reordered, never parallelized.
# Runs entirely BEFORE the scratch-cleanup EXIT trap fires (it is inline in
# this same top-level script body, not a subshell or a trap handler), since
# $checkout_dir lives inside $scratch_dir. Skipped entirely on a timeout:
# see this file's header note.
if [[ "$timed_out" != "true" ]]; then
	mkdir -p "$run_dir/post"

	# Either base ledger carries the same recorded toolchain block
	# (build-ledgers.sh writes it into both); tests.json is picked
	# arbitrarily and consistently as the toolchain-guard's --base.
	base_tests_ledger="$BENCH_DIR/corpus/ledgers/$fixture_base_sha/tests.json"
	base_lint_ledger="$BENCH_DIR/corpus/ledgers/$fixture_base_sha/lint.json"

	# --- step 1: toolchain guard (REQ-F-015 point 1) --------------------
	set +e
	guard_output="$("$SCRIPT_DIR/diff-ledgers.sh" --toolchain-guard --base="$base_tests_ledger" 2>&1)"
	guard_rc=$?
	set -e

	if [[ "$guard_rc" -ne 0 ]]; then
		# REQ-F-015: the guard aborts the WHOLE post-run phase, before any
		# diff (or LOC, or F2P injection) would run -- steps 2-4 below are
		# skipped entirely. The record is still written (AC-10): this
		# branch falls through to meta.json (already written above) and
		# collect-run.sh below, never an early `exit 1`.
		python3 - "$run_dir/post/toolchain-guard.json" "$guard_output" <<'PYEOF'
import json
import sys

path, guard_output = sys.argv[1], sys.argv[2]
prefix = "diff-ledgers: toolchain mismatch: "
mismatches = [line[len(prefix):] for line in guard_output.splitlines() if line.startswith(prefix)]
if not mismatches:
    mismatches = [guard_output.strip() or "toolchain guard failed with no diagnostic output"]
with open(path, "w") as f:
    json.dump({"status": "fail", "mismatches": mismatches}, f, indent=2, sort_keys=True)
PYEOF
		echo "run-one: toolchain guard failed; skipping LOC/quality/F2P-oracle for this run (REQ-F-015)" >&2
	else
		echo '{"status": "pass"}' >"$run_dir/post/toolchain-guard.json"

		# --- step 2: LOC, BEFORE F2P injection (REQ-F-017) --------------
		# Untracked files staged as intent-to-add (contents not actually
		# staged) so a brand-new agent-authored file's lines show up as
		# "added" in the numstat below, without ever committing anything.
		(cd "$checkout_dir" && git add -A -N) >/dev/null 2>&1
		# The pre-dispatch docs/plan mirror is harness context, not an
		# agent-authored fixture change. Exclude it so the established LOC
		# measurement remains scoped to fixture code and tests.
		(cd "$checkout_dir" && git diff --numstat "$fixture_base_sha" -- . ':(exclude)docs/plan/**') >"$run_dir/post/numstat.txt"

		# --- step 3: quality gates + post-run ledger diff, BEFORE F2P
		# injection (REQ-F-016, REQ-F-015 point 3) ------------------------
		set +e
		fmt_output="$(cd "$checkout_dir" && gofmt -l . 2>&1)"
		fmt_rc=$?
		set -e
		printf '%s\n' "$fmt_output" >"$run_dir/post/fmt.txt"
		fmt_status="pass"
		[[ "$fmt_rc" -eq 0 && -z "$fmt_output" ]] || fmt_status="fail"

		set +e
		vet_output="$(cd "$checkout_dir" && go vet ./... 2>&1)"
		vet_rc=$?
		set -e
		printf '%s\n' "$vet_output" >"$run_dir/post/vet.txt"
		vet_status="pass"
		[[ "$vet_rc" -eq 0 ]] || vet_status="fail"

		set +e
		test_output="$(cd "$checkout_dir" && go test ./... 2>&1)"
		test_rc=$?
		set -e
		printf '%s\n' "$test_output" >"$run_dir/post/test.txt"
		tests_status="pass"
		[[ "$test_rc" -eq 0 ]] || tests_status="fail"

		fmt_reason=""
		vet_reason=""
		tests_reason=""
		if [[ "$fmt_rc" -eq 126 || "$fmt_rc" -eq 127 ]]; then
			fmt_status="null"
			fmt_reason="gofmt could not execute (exit code $fmt_rc)"
		fi
		if [[ "$vet_rc" -eq 126 || "$vet_rc" -eq 127 ]]; then
			vet_status="null"
			vet_reason="go vet could not execute (exit code $vet_rc)"
		fi
		if [[ "$test_rc" -eq 126 || "$test_rc" -eq 127 ]]; then
			tests_status="null"
			tests_reason="go test could not execute (exit code $test_rc)"
		fi

		python3 - "$run_dir/post/quality.json" "$fmt_status" "$fmt_reason" "$vet_status" "$vet_reason" "$tests_status" "$tests_reason" <<'PYEOF'
import json
import sys

path, fmt_status, fmt_reason, vet_status, vet_reason, tests_status, tests_reason = sys.argv[1:8]
def gate(status, reason):
    value = {"status": None if status == "null" else status}
    if reason:
        value["reason"] = reason
    return value
doc = {
    "fmt": gate(fmt_status, fmt_reason),
    "vet": gate(vet_status, vet_reason),
    "tests": gate(tests_status, tests_reason),
}
with open(path, "w") as f:
    json.dump(doc, f, indent=2, sort_keys=True)
PYEOF

		postrun_abort=""
		build_output=""
		test_diff_output=""
		lint_diff_output=""
		if ! build_output="$("$BUILD_LEDGERS_BIN" "$checkout_dir" "$run_dir/post" 2>&1)"; then
			postrun_abort="build_ledgers"
		elif ! test_diff_output="$("$SCRIPT_DIR/diff-ledgers.sh" --kind=test --base="$base_tests_ledger" --post="$run_dir/post/tests.json" 2>&1)"; then
			postrun_abort="test_diff"
		elif ! lint_diff_output="$("$SCRIPT_DIR/diff-ledgers.sh" --kind=lint --base="$base_lint_ledger" --post="$run_dir/post/lint.json" 2>&1)"; then
			postrun_abort="lint_diff"
		else
			printf '%s\n' "$test_diff_output" >"$run_dir/post/test-diff.json"
			printf '%s\n' "$lint_diff_output" >"$run_dir/post/lint-diff.json"
		fi
		if [[ -n "$postrun_abort" ]]; then
			postrun_abort_detail="${build_output:-${test_diff_output:-$lint_diff_output}}"
			postrun_abort_detail_path="$run_dir/post/postrun-abort.detail"
			printf '%s' "$postrun_abort_detail" >"$postrun_abort_detail_path"
			python3 - "$run_dir/post/postrun-abort.json" "$postrun_abort" "$postrun_abort_detail_path" <<'PYEOF'
import json
import sys
with open(sys.argv[3]) as f:
    detail = f.read()
with open(sys.argv[1], "w") as f:
    json.dump({"step": sys.argv[2], "detail": detail or "post-run command failed without diagnostic output"}, f, indent=2, sort_keys=True)
PYEOF
			rm -f "$postrun_abort_detail_path"
			echo "run-one: post-run $postrun_abort failed; recording abort and continuing to collector" >&2
		else

		# --- step 4: F2P injection + oracle, LAST (REQ-F-015 point 4,
		# ADR-F02-11). Reuses admit.sh:493 copy_f2p_files's own mechanism
		# (module-path-relative package-dir resolution) rather than
		# re-deriving it -- admit.sh exposes no standalone CLI seam to
		# invoke that function alone (it is a private helper inside
		# admit.sh's own embedded admission-gate flow, which does far more
		# than a bare copy: throwaway checkout, reference-patch apply,
		# P2P re-evaluation), so this reimplements the identical algorithm
		# rather than shelling out to admit.sh's own entrypoint. The oracle
		# run below mirrors admit.sh's own f2p_green_post_patch posture:
		# process exit code 0 AND every named F2P test individually
		# reporting "pass" (REQ-N-005 -- a subprocess exit code alone is
		# never trusted as the sole evidence).
		python3 - "$corpus_yaml" "$checkout_dir" "$item_f2p_json" "$item_type" "$run_dir/post/f2p.json" <<'PYEOF'
import json
import os
import re
import shutil
import subprocess
import sys

corpus_yaml_path, checkout_dir, f2p_json, item_type, out_path = sys.argv[1:6]
corpus_dir = os.path.dirname(os.path.abspath(corpus_yaml_path))
f2p = json.loads(f2p_json)
paths = f2p["paths"]
test_names = f2p["test_names"]
if len(paths) != len(test_names):
    sys.exit("run-one: f2p.paths and f2p.test_names length mismatch")

with open(os.path.join(checkout_dir, "go.mod")) as f:
    first_line = f.readline().strip()
if not first_line.startswith("module "):
    sys.exit("run-one: unexpected go.mod first line: %r" % first_line)
module_path = first_line[len("module "):].strip()
prefix = module_path + "/"

for path, test_name in zip(paths, test_names):
    pkg_import = test_name.split("::", 1)[0]
    if not pkg_import.startswith(prefix):
        sys.exit(
            "run-one: f2p test %r package %r is not under module %r"
            % (test_name, pkg_import, module_path)
        )
    rel_pkg_dir = pkg_import[len(prefix):]
    src = os.path.join(corpus_dir, path)
    dst_dir = os.path.join(checkout_dir, rel_pkg_dir)
    os.makedirs(dst_dir, exist_ok=True)
    shutil.copy2(src, os.path.join(dst_dir, os.path.basename(path)))


def bare_test_name(name):
    # Strips a "<pkg>::" prefix (if present) and any "/<subtest>" suffix --
    # mirrors admit.sh's own bare_test_name exactly.
    name = name.split("::", 1)[-1]
    return name.split("/", 1)[0]


bare_names = sorted({bare_test_name(n) for n in test_names})
run_pattern = "^(" + "|".join(re.escape(n) for n in bare_names) + ")$"
packages = sorted({t.split("::", 1)[0] for t in test_names})

proc = subprocess.run(
    ["go", "test", "-json", "-run", run_pattern] + packages,
    cwd=checkout_dir,
    capture_output=True,
    text=True,
)

results = {}
for line in proc.stdout.splitlines():
    line = line.strip()
    if not line:
        continue
    try:
        ev = json.loads(line)
    except ValueError:
        continue
    test = ev.get("Test")
    pkg = ev.get("Package")
    action = ev.get("Action")
    if not test or not pkg or "/" in test or action not in ("pass", "fail", "skip"):
        continue
    results["%s::%s" % (pkg, test)] = action

f2p_resolved = proc.returncode == 0 and all(results.get(name) == "pass" for name in test_names)

record = {"f2p_resolved": f2p_resolved}
if item_type == "bug":
    record["repro_confirmed"] = f2p_resolved

with open(out_path, "w") as f:
    json.dump(record, f, sort_keys=True)
PYEOF
		fi
	fi
fi

"$SCRIPT_DIR/collect-run.sh" --run-dir "$run_dir" >"$record_path.tmp" || {
	echo "run-one: collect-run.sh failed for $run_dir" >&2
	rm -f "$record_path.tmp"
	exit 1
}
mv "$record_path.tmp" "$record_path"

echo "run-one: record written to $record_path" >&2
