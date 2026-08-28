#!/usr/bin/env bash
# canary-runsurface.sh [--corpus <corpus.yaml>]
#
# X-07 canary preflight (REQ-F-020, ADR-F02-10, AC-13): asserts the
# RunResult/StageLog JSON field set (internal/runner/controller.go:82-135)
# and the transcript byte format (internal/runner/transcript.go:10-25,51)
# this feature's collector depends on are unchanged -- via a REAL `shark
# run` invocation, not a re-derivation of the shape from memory. Aborts and
# names exactly the changed field on drift; never a generic "shape
# mismatch" message.
#
# Provisions its own throwaway scratch project (scripts/shark-scratch-env.sh
# -- the same real provisioning path run-one.sh itself uses, never a second
# implementation of it) because AC-21 runs this preflight BEFORE
# run-one.sh's own provisioning, so there is no run-one.sh-created project to
# reuse yet. Tears the scratch project down on exit regardless of outcome.
#
# The dispatched agent is a PATH-stubbed `claude`
# (bench/scripts/testdata/stubs/claude) only -- `shark run` itself is the
# real repo binary, never stubbed, because the whole point of this canary is
# asserting the REAL RunController's output shape. Per test-plan.md's own
# framing, "a real invocation" here means the real
# RunController/LivenessRecorder/transcript-writer code path actually
# executing, not a live, metered Anthropic API call -- a `claude` call would
# tell this canary nothing about the shark surface it exists to pin, and
# would burn API budget on every single batch's preflight for zero benefit.
#
# Expected field lists below are pinned literals, authored by cross-checking
# a live invocation's actual JSON keys against controller.go's own json
# struct tags -- never re-derived from controller.go at runtime (a rename
# there would just move the expectation with it, and the canary could never
# catch it) and never guessed from memory.
#
# Output contract: every diagnostic this script prints, including the final
# result sentinel, goes to STDERR -- exactly one line reading "PASS" or
# "FAIL: <field>" as the LAST line, matching testdata/stubs/shark's
# established "failure text lands on stderr" convention that run-one.sh's
# own canary-failure error-surfacing (REQ-F-020/AC-21) depends on.
#
# Env var override (mirrors TC-011's DIFF_LEDGERS_GOLANGCI_CONFIG
# technique): when CANARY_RUNSURFACE_RUNRESULT_FIXTURE is set, this script
# skips provisioning and dispatch entirely and treats that path as the
# "observed" RunResult JSON -- only the RunResult-level field-set check
# runs (StageLog/transcript checks need a real run directory, which does
# not exist on this path). This is TC-016b's technique, and it is also how
# TC-016a proves the "reordered JSON keys still pass" negative case.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

CLAUDE_STUB="$SCRIPT_DIR/testdata/stubs/claude"

usage() {
	echo "usage: canary-runsurface.sh [--corpus <corpus.yaml>]" >&2
	exit 2
}

# --corpus is accepted (run-one.sh always passes it, AC-21) but intentionally
# unused: this canary checks shark run's OUTPUT SHAPE, which has nothing to
# do with which corpus item would eventually be seeded -- it seeds its own
# trivial ad-hoc task, never a real corpus item.
while [[ $# -gt 0 ]]; do
	case "$1" in
	--corpus)
		[[ $# -ge 2 ]] || usage
		shift 2
		;;
	*)
		usage
		;;
	esac
done

# --- pinned expected field sets -- see header note above. ---
RUNRESULT_REQUIRED="entity_key,final_status,stages_completed,stages,outcome,total_duration_ns"
RUNRESULT_OPTIONAL="error,question_block"
STAGELOG_REQUIRED="status,action,duration_ns,exit_code,entity_key"
STAGELOG_OPTIONAL="agent_type,provider,output_summary"

fail_field() {
	echo "FAIL: $1" >&2
	exit 1
}

# check_runresult_fields <json-file> -- prints the sorted observed field CSV
# on stdout and returns 0 when every required RunResult field is present and
# no field outside required+optional appears; otherwise prints the single
# offending field name on stdout and returns 1. REQ-F-020: never a generic
# "shape mismatch" -- always exactly one named field.
check_runresult_fields() {
	local json_file="$1"
	python3 - "$json_file" "$RUNRESULT_REQUIRED" "$RUNRESULT_OPTIONAL" <<'PYEOF'
import json
import sys

path, required_csv, optional_csv = sys.argv[1:4]
required = [f for f in required_csv.split(",") if f]
optional = [f for f in optional_csv.split(",") if f]
known = set(required) | set(optional)

with open(path) as f:
    obj = json.load(f)

if not isinstance(obj, dict):
    sys.stderr.write("canary-runsurface: RunResult JSON is not an object\n")
    sys.exit(2)

observed = set(obj.keys())

missing = sorted(f for f in required if f not in observed)
if missing:
    print(missing[0])
    sys.exit(1)

unexpected = sorted(observed - known)
if unexpected:
    print(unexpected[0])
    sys.exit(1)

print(",".join(sorted(observed)))
PYEOF
}

# verify_spawn_agent_and_transcript <observed-json-file> <run-dir> -- finds
# the first RunResult.stages[] entry with action=="spawn_agent", checks its
# StageLog field set against the pinned lists, then locates and verifies the
# transcript file's byte format (COMMAND:/EXIT:/DURATION:<ms>ms/
# ---STDOUT---/---STDERR---) using the stage's own exit_code/output_summary
# as an exact anchor for the STDOUT block -- this is what makes the check
# robust against the COMMAND value itself spanning many lines (the
# assembled dispatch prompt), which rules out any line-number-based marker
# search.
#
# Prints the sorted StageLog field CSV then the transcript path on success
# (exit 0); prints the single offending field name on stdout on a field-set
# drift (exit 1); prints a precondition failure message on stderr for
# anything else (no spawn_agent stage, non-zero exit, missing/ambiguous
# transcript) (exit 2) -- this last class is not a "field rename", so it is
# reported distinctly rather than reusing the field-drift shape.
verify_spawn_agent_and_transcript() {
	local json_file="$1" run_dir="$2"
	python3 - "$json_file" "$run_dir" "$STAGELOG_REQUIRED" "$STAGELOG_OPTIONAL" <<'PYEOF'
import glob
import json
import os
import re
import sys

json_path, run_dir, required_csv, optional_csv = sys.argv[1:5]
required = [f for f in required_csv.split(",") if f]
optional = [f for f in optional_csv.split(",") if f]
known = set(required) | set(optional)

with open(json_path) as f:
    obj = json.load(f)

stages = obj.get("stages")
if not isinstance(stages, list):
    sys.stderr.write("canary-runsurface: no stages[] array in RunResult\n")
    sys.exit(2)

spawn_agent_stages = [s for s in stages if isinstance(s, dict) and s.get("action") == "spawn_agent"]
if not spawn_agent_stages:
    sys.stderr.write("canary-runsurface: no spawn_agent stage in RunResult.stages -- cannot verify StageLog/transcript shape\n")
    sys.exit(2)

stage = spawn_agent_stages[0]
observed = set(stage.keys())

missing = sorted(f for f in required if f not in observed)
if missing:
    print(missing[0])
    sys.exit(1)

unexpected = sorted(observed - known)
if unexpected:
    print(unexpected[0])
    sys.exit(1)

if stage.get("exit_code") != 0:
    sys.stderr.write("canary-runsurface: spawn_agent stage exit_code=%r, want 0\n" % stage.get("exit_code"))
    sys.exit(2)

stdout = stage.get("output_summary")
if not stdout:
    sys.stderr.write("canary-runsurface: spawn_agent stage has no output_summary -- cannot anchor the transcript check\n")
    sys.exit(2)

status = stage.get("status", "")
provider = stage.get("provider", "")
# Transcripts now live one directory level deeper, under the entity
# key that produced them (run_dir/<entityKey>/*.log -- B052
# entity-scoped path fix). This canary seeds exactly one task, so
# exactly one entity directory exists; search recursively so this
# check does not care how deep that directory is nested.
candidates = sorted(
    glob.glob(os.path.join(run_dir, "**", "*-%s-%s.log" % (status, provider)), recursive=True)
)
if len(candidates) != 1:
    sys.stderr.write(
        "canary-runsurface: expected exactly one transcript matching *-%s-%s.log under %s, found %d\n"
        % (status, provider, run_dir, len(candidates))
    )
    sys.exit(2)

transcript_path = candidates[0]
with open(transcript_path) as f:
    content = f.read()

if not content.startswith("COMMAND: "):
    sys.stderr.write("canary-runsurface: transcript does not start with 'COMMAND: ': %s\n" % transcript_path)
    sys.exit(2)

exit_code = stage.get("exit_code")
pattern = (
    r"\nEXIT: " + re.escape(str(exit_code))
    + r"\nDURATION: \d+ms\n---STDOUT---\n"
    + re.escape(stdout)
    + r"\n---STDERR---\n\Z"
)
if not re.search(pattern, content):
    sys.stderr.write(
        "canary-runsurface: transcript byte format drifted (COMMAND:/EXIT:/DURATION:<ms>ms/---STDOUT---/---STDERR---): %s\n"
        % transcript_path
    )
    sys.exit(2)

print(",".join(sorted(observed)))
print(transcript_path)
PYEOF
}

# --- Env var override path (TC-016b, and TC-016a's reordered-keys negative
# case): no provisioning, no dispatch -- just the RunResult-level field-set
# check against the given fixture file. ---
if [[ -n "${CANARY_RUNSURFACE_RUNRESULT_FIXTURE:-}" ]]; then
	[[ -f "$CANARY_RUNSURFACE_RUNRESULT_FIXTURE" ]] || {
		echo "canary-runsurface: CANARY_RUNSURFACE_RUNRESULT_FIXTURE not found: $CANARY_RUNSURFACE_RUNRESULT_FIXTURE" >&2
		exit 2
	}
	set +e
	fields="$(check_runresult_fields "$CANARY_RUNSURFACE_RUNRESULT_FIXTURE")"
	rc=$?
	set -e
	if [[ "$rc" -ne 0 ]]; then
		fail_field "$fields"
	fi
	echo "canary-runsurface: RunResult fields: $fields" >&2
	echo "PASS" >&2
	exit 0
fi

# --- Real invocation path (TC-016a) ---------------------------------------
[[ -x "$CLAUDE_STUB" ]] || {
	echo "canary-runsurface: PATH-stub claude missing or not executable: $CLAUDE_STUB" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "canary-runsurface: python3 not found on PATH" >&2
	exit 2
}
command -v timeout >/dev/null 2>&1 || {
	echo "canary-runsurface: timeout not found on PATH" >&2
	exit 2
}

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/canary-runsurface-work-XXXXXX")"
scratch_dir=""
cleanup() {
	rm -rf "$work_dir"
	[[ -n "$scratch_dir" && -d "$scratch_dir" ]] && rm -rf "$scratch_dir"
	return 0
}
trap cleanup EXIT

scratch_dir="$("$REPO_ROOT/scripts/shark-scratch-env.sh" "canary-runsurface")"
[[ -n "$scratch_dir" && -d "$scratch_dir" ]] || {
	echo "canary-runsurface: scripts/shark-scratch-env.sh did not print a usable scratch directory" >&2
	exit 2
}
echo "canary-runsurface: scratch=$scratch_dir" >&2

real_shark="$scratch_dir/shark"
[[ -x "$real_shark" ]] || {
	echo "canary-runsurface: scratch project has no shark binary at $real_shark" >&2
	exit 2
}

shark_in_scratch() {
	(cd "$scratch_dir" && "$real_shark" "$@" </dev/null)
}

create_key() {
	# $1 = entity_type, remaining args passed through -- captures the
	# assigned .key from --json, never invents one (REQ-N-005).
	local entity_type="$1"
	shift
	local out
	out="$(shark_in_scratch create "$entity_type" "$@" --json)" || {
		echo "canary-runsurface: 'shark create $entity_type $*' failed in the scratch project" >&2
		exit 2
	}
	python3 -c 'import json,sys; print(json.loads(sys.argv[1])["key"])' "$out"
}

# --- seed a trivial, ad-hoc task (not a real corpus item -- this canary
# checks shark run's output SHAPE, which is independent of the entity's own
# content) -----------------------------------------------------------------
epic_key="$(create_key epic "shark-bench X-07 canary probe")"
feature_key="$(create_key feature "$epic_key" "shark-bench X-07 canary probe")"
task_key="$(create_key task "$epic_key" "$feature_key" "shark-bench X-07 canary probe")"

# --- capture_agent_transcripts=true -- required for a real transcript file
# to be written (in addition to a non-empty ProjectRoot, which the scratch
# project's own .sharkconfig.json supplies via shark's project-root
# auto-detection). ---
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

# --- dispatch: real shark run, PATH-stubbed claude only. A bounded timeout
# guards against a misconfigured PATH accidentally reaching a real claude
# binary (which could hang on an interactive prompt or spend real budget) --
# not a substitute for run-one.sh's own process-group cap, which exists for
# a genuinely long-running real agent dispatch; this canary's own dispatch
# always completes in well under a second against the stub. ---
claude_stub_dir="$work_dir/claude-stub-bin"
mkdir -p "$claude_stub_dir"
cp "$CLAUDE_STUB" "$claude_stub_dir/claude"
chmod +x "$claude_stub_dir/claude"

observed_file="$work_dir/observed-runresult.json"
run_stderr="$work_dir/run.stderr"

# REQ-N-002: cd into the scratch project FIRST -- shark's project-root
# auto-detection walks up from the current directory and ignores --config/
# --db for that walk, so running this from the canary's own (repo-tree) cwd
# would resolve back to the live repo/live database instead of the scratch
# project.
set +e
(
	cd "$scratch_dir"
	PATH="$claude_stub_dir:$PATH" timeout 60 "$real_shark" run "$task_key" --json </dev/null
) >"$observed_file" 2>"$run_stderr"
run_rc=$?
set -e
if [[ "$run_rc" -ne 0 ]]; then
	echo "canary-runsurface: 'shark run $task_key --json' exited $run_rc: $(cat "$run_stderr")" >&2
	exit 2
fi

set +e
runresult_fields="$(check_runresult_fields "$observed_file")"
rc=$?
set -e
if [[ "$rc" -ne 0 ]]; then
	fail_field "$runresult_fields"
fi

run_dirs=("$scratch_dir"/.shark/runs/*/)
[[ "${#run_dirs[@]}" -eq 1 && -d "${run_dirs[0]}" ]] || {
	echo "canary-runsurface: expected exactly one directory under $scratch_dir/.shark/runs, found ${#run_dirs[@]}" >&2
	exit 2
}
run_dir="${run_dirs[0]%/}"

set +e
verify_out="$(verify_spawn_agent_and_transcript "$observed_file" "$run_dir")"
rc=$?
set -e
if [[ "$rc" -eq 1 ]]; then
	fail_field "$verify_out"
elif [[ "$rc" -ne 0 ]]; then
	exit 2
fi
stagelog_fields="$(echo "$verify_out" | head -n1)"
transcript_path="$(echo "$verify_out" | tail -n1)"

echo "canary-runsurface: RunResult fields: $runresult_fields" >&2
echo "canary-runsurface: StageLog fields: $stagelog_fields" >&2
echo "canary-runsurface: transcript format confirmed: $transcript_path" >&2
echo "PASS" >&2
exit 0
