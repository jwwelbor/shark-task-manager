#!/usr/bin/env bash
# TC-111 / T-E40-F11-011 (spec.md REQ-F-012, AC-F11-38a/40/41/41a;
# test-plan.md TC-38a/40/41/41a): the six-family scenario-root isolation
# proof and the live-database non-mutation proof, built on
# T-E40-F11-017's per-scenario_id `scenario_roots` keying (AC-F11-38/39,
# already proven by tc111a_scenario_index_seeding_test.sh -- not repeated
# here).
#
# Scope split (task file is explicit about this):
#   - AC-F11-38a: a synthetic seventh package sharing an admitted family
#     receives its own distinct root_key, proven through the production
#     `setup --scenario-index <path>` seam (spec.md AC-F11-38a).
#   - AC-F11-40 (reinforcement): no `shark create` invocation in
#     `seed_scenario_root` ever passes `--key` -- every key is read from the
#     `--json` response. tc111a already proves the dynamic half (every
#     root_key resolves via `shark get`); this file adds the static
#     provenance guard and its counterfactual.
#   - AC-F11-41 / AC-F11-41a: no seam reads or writes the repository's live
#     `shark-tasks.db`. Per spec.md's second and third reworks (§1.2,
#     §7.7.2), this is proven by **syscall observation** (`strace -f -y`),
#     never by hashing or copying the live file -- hashing the file IS a
#     read, which directly contradicts AC-F11-41a. The live `shark-tasks.db`
#     is never opened, copied, hashed, or mutated by this test; see
#     `.claude/rules/database-critical.md`.
#
# Caller-Path Contract: real filesystem, real git, the real Shark binary
# (bin/shark), the real e40-benchmark.sh CLI, and `strace` as the tracing
# parent (spec.md §7.7.2). Nothing internal to e40_benchmark.py is imported
# or called directly, except the one clearly-labeled static source check for
# AC-F11-40's key-provenance guard (analogous to AC-F11-01a's static grep
# check -- spec.md §7.1 row 01a).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
BENCHMARK_LIB="$SCRIPTS_DIR/lib/e40_benchmark.py"
LIVE_DB="$REPO_ROOT/shark-tasks.db"

fail() {
	echo "TC-111 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh is missing or not executable"
[[ -x "$REPO_ROOT/bin/shark" ]] || fail "bin/shark is missing; run 'make shark' first"
[[ -f "$LIVE_DB" ]] || fail "repository live database is missing: $LIVE_DB"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
command -v strace >/dev/null 2>&1 || fail "strace is required for the AC-F11-41/41a syscall proof"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# AC-F11-38a: build a test-only scenario index with the six admitted
# packages plus the synthetic seventh (entity_family: bug, distinct
# scenario_id), reachable only through `setup --scenario-index <path>` --
# never by editing bench/scenarios/scenarios.yaml or harness internals.
# ===========================================================================
DUPLICATE_PACKAGE_DIR="$SCRIPT_DIR/testdata/duplicate-family/py-bug-second-scenario"
[[ -f "$DUPLICATE_PACKAGE_DIR/package.yaml" ]] \
	|| fail "synthetic duplicate-family package.yaml is missing: $DUPLICATE_PACKAGE_DIR"

SEVEN_PACKAGE_INDEX="$WORKDIR/seven-package-scenario-index.yaml"
cat >"$SEVEN_PACKAGE_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $BENCH_DIR/scenarios/packages/py-bug-due-date-boundary
  - $BENCH_DIR/scenarios/packages/py-change-priority-scale
  - $BENCH_DIR/scenarios/packages/py-feature-recurring-tasks
  - $BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation
  - $BENCH_DIR/scenarios/packages/py-task-delete-task
  - $BENCH_DIR/scenarios/packages/py-epic-task-organization
  - $DUPLICATE_PACKAGE_DIR
EOF

ISO_ROOT="$WORKDIR/iso-setup"
DUMMY_DB="$WORKDIR/positive-control-dummy.db"
TRACE_LOG="$WORKDIR/setup.strace.log"
TRACED_RUN_SCRIPT="$WORKDIR/traced-run.sh"

[[ ! -e "$ISO_ROOT" ]] || fail "operator root must not exist before setup (non-existent-temp-path counterfactual)"
[[ ! -e "$DUMMY_DB" ]] || fail "positive-control dummy database must not exist before the traced run"

# The 17 names isolated_environment() strips (spec.md AC-F11-41a, verified
# against bench/scripts/lib/e40_benchmark.py:isolated_environment 2026-09-06).
# Poisoned here with a value naming an unreachable root, so any leak through
# to a child process (and hence into an opened path) is unmistakable.
SCRUBBED_NAMES=(
	SHARK_DB_URL SHARK_AUTH_TOKEN SHARK_AUTH_TOKEN_FILE SHARK_BIN
	LIFECYCLE_ADAPTER LIFECYCLE_ADAPTER_PATH LIFECYCLE_PROVIDER_COMMAND
	RUN_LIFECYCLE_BIN EVALUATE_LIFECYCLE_BIN ENTITY_HISTORY_EXPORT_BIN
	LIFECYCLE_SCHEMA I05_SCHEMA I07_SCHEMA E40_RUN_LIFECYCLE_BATCH_BIN
	E40_AGGREGATE_LIFECYCLE_BIN E40_REPORT_LIFECYCLE_BIN E40_VERIFY_RETENTION_ROOT_BIN
)
POISON_ENV_ARGS=()
for name in "${SCRUBBED_NAMES[@]}"; do
	POISON_ENV_ARGS+=("${name}=/DENIED/root/must-not-leak-into-any-resolved-path")
done

# The traced run does two things inside the SAME strace session
# (spec.md §7.7.2's "positive control ... inside the same traced run"):
#   1. the real `setup --scenario-index <7-package index>` invocation --
#      seeds all six families plus the duplicate-family root; and
#   2. a helper that opens an INDEPENDENT dummy database the test itself
#      creates (never a copy of the live file) -- the positive control that
#      proves the observer is actually working.
cat >"$TRACED_RUN_SCRIPT" <<EOF
#!/usr/bin/env bash
set -euo pipefail
"$OPERATOR" setup --out "$ISO_ROOT" --scenario-index "$SEVEN_PACKAGE_INDEX"
python3 -c "
with open(r'''$DUMMY_DB''', 'wb') as handle:
    handle.write(b'positive control dummy database -- never the live shark-tasks.db')
with open(r'''$DUMMY_DB''', 'rb') as handle:
    handle.read()
"
EOF
chmod +x "$TRACED_RUN_SCRIPT"

setup_rc=0
env "${POISON_ENV_ARGS[@]}" strace -f -qq -y \
	-e trace=openat,open,unlink,unlinkat,rename,renameat,renameat2 \
	-o "$TRACE_LOG" \
	bash "$TRACED_RUN_SCRIPT" >"$WORKDIR/traced-run.out" 2>&1 || setup_rc=$?
[[ "$setup_rc" -eq 0 ]] || fail "traced setup run failed (rc=$setup_rc): $(cat "$WORKDIR/traced-run.out")"
[[ -s "$TRACE_LOG" ]] || fail "strace produced an empty log -- the observer did not run"

# The write-detection arms (rename*/unlink*) are dead weight unless the trace
# actually records one: `setup` always does a staging-to-final publish
# rename (publish_staging_directory -> os.rename(..., dir_fd=...)), so a
# working rename arm MUST have fired at least once in this very run. This
# guards against glibc/Python resolving to a syscall name (e.g. renameat2,
# unlinkat) that a narrower -e trace= filter would silently miss, leaving
# the no-write proof resting on `openat` alone.
rename_records="$(grep -cE 'renameat2?\(|(^|[^_a-z])rename\(' "$TRACE_LOG" || true)"
[[ "$rename_records" -gt 0 ]] \
	|| fail "no rename/renameat/renameat2 record observed even though setup performs a staging-directory publish rename -- the rename write-detection arm is dead"
echo "TC-111(AC-F11-41 rename-arm liveness): $rename_records rename record(s) observed -- the rename/unlink write-detection arm is live, not silently dead -- PASS"

echo "TC-111: traced 7-scenario setup run (6 admitted families + AC-F11-38a synthetic duplicate) completed -- proceeding to isolation assertions"

# ===========================================================================
# AC-F11-38a: the synthetic seventh package (entity_family: bug) receives a
# root_key distinct from py-bug-due-date-boundary's.
# ===========================================================================
python3 - "$ISO_ROOT/setup-result.json" "$ISO_ROOT/e40-demo.yaml" "$REPO_ROOT" "$SEVEN_PACKAGE_INDEX" <<'PY'
import json
import sys

import yaml

result_path, config_path, repo_root, index_path = sys.argv[1:5]
result = json.load(open(result_path, encoding="utf-8"))
config = yaml.safe_load(open(config_path, encoding="utf-8"))

# entity_family is read from each package.yaml itself -- never hardcoded in
# this test -- so a fixture drift (someone editing the synthetic package's
# family, or a real package's) makes this check fail loudly instead of the
# assertions below quietly testing nothing.
index = yaml.safe_load(open(index_path, encoding="utf-8"))
family_by_scenario: dict[str, str] = {}
for package_dir in index["scenarios"]:
    package = yaml.safe_load(open(f"{package_dir}/package.yaml", encoding="utf-8"))
    family_by_scenario[package["scenario_id"]] = package["entity_family"]

expected_scenario_ids = set(family_by_scenario)
if len(expected_scenario_ids) != 7:
    raise SystemExit(f"expected 7 packages in the index, got {len(expected_scenario_ids)}")

root_keys = result["root_keys"]
if set(root_keys) != expected_scenario_ids:
    raise SystemExit(
        f"expected exactly the 6 admitted scenario_ids plus the synthetic "
        f"duplicate, got: {sorted(root_keys)}"
    )

# Precondition: the "duplicate family" premise must hold for real, read from
# the package files, not assumed. Exactly one family must be shared by two
# scenario_ids (the deliberate duplicate) and the other five must be
# pairwise distinct -- otherwise this test is not exercising AC-F11-38a at
# all, regardless of what the assertions below conclude.
family_counts: dict[str, list[str]] = {}
for scenario_id, family in family_by_scenario.items():
    family_counts.setdefault(family, []).append(scenario_id)
duplicated = {family: ids for family, ids in family_counts.items() if len(ids) > 1}
if len(duplicated) != 1:
    raise SystemExit(
        f"expected exactly one duplicated entity_family among the 7 "
        f"packages, found: {duplicated}"
    )
(dup_family, dup_scenario_ids), = duplicated.items()
if set(dup_scenario_ids) != {"py-bug-due-date-boundary", "py-bug-second-scenario"}:
    raise SystemExit(
        f"the duplicated family {dup_family!r} is shared by an unexpected "
        f"scenario_id pair: {sorted(dup_scenario_ids)}"
    )
print(
    f"TC-111(AC-F11-38a precondition): py-bug-due-date-boundary and "
    f"py-bug-second-scenario genuinely share entity_family={dup_family!r} "
    f"(read from their own package.yaml files, not assumed) -- PASS"
)

bug_a = root_keys["py-bug-due-date-boundary"]
bug_b = root_keys["py-bug-second-scenario"]
if bug_a == bug_b:
    raise SystemExit(
        f"AC-F11-38a violated: two packages sharing entity_family="
        f"{dup_family!r} collided on one root_key ({bug_a!r})"
    )
if not (bug_a and bug_b):
    raise SystemExit(f"one of the two duplicated-family root_keys is empty: {bug_a!r}, {bug_b!r}")
print(
    f"TC-111(AC-F11-38a positive): duplicate-family packages received "
    f"distinct root_keys ({bug_a!r} != {bug_b!r}) -- PASS"
)

# Counterfactual: fold the SAME captured root_keys into a family-keyed dict
# the way the pre-rework `roots[family]` lookup did, using entity_family
# values read from the packages themselves (not a test-authored literal).
# Iterating and keeping only the last write per family is exactly what a
# `dict[family] = root_key` assignment loop does -- this is not a
# simulation of different code, it is the documented failure mode of the
# lookup this feature replaced, applied to real captured data.
scenario_roots = config["scenario_roots"]
if set(scenario_roots) != expected_scenario_ids:
    raise SystemExit(f"config scenario_roots key set mismatch: {sorted(scenario_roots)}")

family_keyed_roots: dict[str, str] = {}
for scenario_id in sorted(root_keys):
    family_keyed_roots[family_by_scenario[scenario_id]] = root_keys[scenario_id]

if len(family_keyed_roots) >= len(root_keys):
    raise SystemExit(
        "counterfactual is not meaningful: re-keying by family did not "
        "collapse any entries, so it fails to demonstrate the pre-rework "
        "collision this AC guards against"
    )
if family_keyed_roots[dup_family] not in (bug_a, bug_b):
    raise SystemExit("counterfactual re-keying produced an unexpected value")
print(
    "TC-111(AC-F11-38a counterfactual): re-keying the SAME captured roots by "
    f"entity_family (the pre-rework roots[family] lookup) collapses the two "
    f"{dup_family} scenarios onto one root_key ({family_keyed_roots[dup_family]!r}) "
    f"-- {len(root_keys)} scenario_id-keyed roots became "
    f"{len(family_keyed_roots)} family-keyed roots -- PASS (this is the "
    "defect AC-F11-38/38a close)"
)

scratch_root_value = config.get("scratch_root", "")
if scratch_root_value.startswith(repo_root):
    raise SystemExit(f"scratch_root resolves inside REPO_ROOT: {scratch_root_value}")
print("TC-111(AC-F11-41a config-level): scratch_root lies outside REPO_ROOT -- PASS")
PY

# ===========================================================================
# AC-F11-40 (static reinforcement): `seed_scenario_root`'s `create()` calls
# never pass `--key` -- every key comes from the `--json` response
# (dynamic proof that keys resolve to real entities is tc111a's). This
# repository's cloud database assigns keys (feedback_cloud_db_no_key_
# assignment); a harness-supplied key is a defect.
# ===========================================================================
python3 - "$BENCHMARK_LIB" <<'PY'
import re
import sys

source_path = sys.argv[1]
source = open(source_path, encoding="utf-8").read()

match = re.search(
    r"^def seed_scenario_root\(.*?\n(?:.*\n)*?(?=^def )",
    source,
    re.MULTILINE,
)
if not match:
    raise SystemExit("could not locate seed_scenario_root in e40_benchmark.py")
body = match.group(0)


def has_key_flag(text: str) -> bool:
    return bool(re.search(r'["\']--key["\']', text))


if has_key_flag(body):
    raise SystemExit(
        "AC-F11-40 violated: seed_scenario_root passes --key to a `shark "
        "create` invocation instead of reading the key from --json"
    )
print(
    "TC-111(AC-F11-40 positive): seed_scenario_root never passes --key to "
    "any `shark create` invocation -- PASS"
)

# Counterfactual: prove the same detector actually fires on the defect it
# guards against, using a synthetic snippet -- never by mutating the real
# harness source.
mutated = body.replace(
    'f"{label} target feature"],',
    'f"{label} target feature"], ["--key", "E99-F99"],',
    1,
)
if mutated == body:
    raise SystemExit("counterfactual mutation did not apply -- fixture drifted from source")
if not has_key_flag(mutated):
    raise SystemExit("counterfactual is not meaningful: mutated snippet still has no --key flag")
print(
    "TC-111(AC-F11-40 counterfactual): the same --key detector fires on a "
    "synthetically mutated copy of the function body -- PASS (detector is "
    "capable of failing, not vacuously green)"
)
PY

# ===========================================================================
# AC-F11-41 / AC-F11-41a: parse the strace log from the traced run above.
# No openat/open/rename(at[2])/unlink(at) record may resolve to the live
# shark-tasks.db (or its -wal/-shm siblings); every resolved database path
# must lie under the isolated scratch project; the positive control (an
# independently-created dummy database, never a copy of the live file) must
# appear, proving the observer actually works.
# ===========================================================================
python3 - "$TRACE_LOG" "$LIVE_DB" "$WORKDIR" "$DUMMY_DB" "$REPO_ROOT" <<'PY'
import re
import sys

log_path, live_db, scratch_root, dummy_db, repo_root = sys.argv[1:6]

# One reusable checker so the "must be capable of failing" requirement
# (AC-F11-41/41a) is proven against a synthetic bad log below, not asserted
# by construction.
LIVE_DB_BASENAMES = ("shark-tasks.db", "shark-tasks.db-wal", "shark-tasks.db-shm")
RESOLVED_SUFFIX = re.compile(r"<([^<>]+)>\s*$")


def check_log(log_text: str, live_db_path: str, scratch_root_path: str, repo_root_path: str):
    """Returns (db_paths_seen, dummy_seen). Raises SystemExit naming the
    offending line if the live database (or a sibling) was ever resolved
    outside the scratch project, or if a live-path literal appears at all.
    """
    db_paths_seen: set[str] = set()
    dummy_seen = False
    live_db_dir = "/".join(live_db_path.split("/")[:-1])
    for lineno, line in enumerate(log_text.splitlines(), start=1):
        if not any(
            token in line
            for token in ("openat(", "open(", "rename(", "renameat(", "renameat2(", "unlink(", "unlinkat(")
        ):
            continue
        if dummy_db_marker in line:
            dummy_seen = True
        resolved_match = RESOLVED_SUFFIX.search(line)
        resolved_path = resolved_match.group(1) if resolved_match else None
        for basename in LIVE_DB_BASENAMES:
            if basename not in line:
                continue
            # A record naming a live-db basename must resolve strictly
            # under the scratch project -- never under REPO_ROOT (and in
            # particular never live_db_dir, the repository root itself).
            if resolved_path is not None:
                if not resolved_path.startswith(scratch_root_path + "/") and resolved_path != scratch_root_path:
                    raise SystemExit(
                        f"AC-F11-41/41a VIOLATION at line {lineno}: {basename} "
                        f"resolved outside the scratch project: {resolved_path!r}\n"
                        f"  raw record: {line.strip()}"
                    )
                if resolved_path.startswith(repo_root_path):
                    raise SystemExit(
                        f"AC-F11-41/41a VIOLATION at line {lineno}: {basename} "
                        f"resolved inside REPO_ROOT: {resolved_path!r}\n"
                        f"  raw record: {line.strip()}"
                    )
                db_paths_seen.add(resolved_path)
            else:
                # Call failed to resolve (e.g. ENOENT) -- still must not
                # literally name the live directory.
                if live_db_dir in line:
                    raise SystemExit(
                        f"AC-F11-41/41a VIOLATION at line {lineno}: unresolved "
                        f"record names the live repository directory: {line.strip()}"
                    )
    return db_paths_seen, dummy_seen


dummy_db_marker = dummy_db

log_text = open(log_path, encoding="utf-8", errors="replace").read()
db_paths_seen, dummy_seen = check_log(log_text, live_db, scratch_root, repo_root)

if not dummy_seen:
    raise SystemExit(
        "positive control FAILED: the independently-created dummy database "
        "was never observed in the trace -- the observer is not working"
    )
print(
    "TC-111(AC-F11-41 positive control): the syscall observer recorded the "
    "independent dummy database opened inside the same traced run -- PASS"
)

if not db_paths_seen:
    raise SystemExit(
        "no shark-tasks.db (or sibling) open was observed at all -- setup "
        "should have created one under the scratch project; the trace may "
        "be incomplete"
    )
for path in db_paths_seen:
    if not (path.startswith(scratch_root + "/") or path == scratch_root):
        raise SystemExit(f"resolved database path is not under the scratch project: {path}")
print(
    f"TC-111(AC-F11-41 positive): every resolved shark-tasks.db path "
    f"({sorted(db_paths_seen)}) lies under the scratch project and none "
    f"names the live repository database -- PASS"
)

# The synthetic "bad log" proves check_log() is capable of failing --
# required by AC-F11-41/41a's "must fail rather than pass" language. This
# is a fabricated log line, not a real access to the live file.
fabricated_bad_line = (
    f'99999 openat(AT_FDCWD</somewhere>, "shark-tasks.db", O_RDONLY) = '
    f'3<{live_db}>'
)
try:
    check_log(fabricated_bad_line, live_db, scratch_root, repo_root)
except SystemExit:
    print(
        "TC-111(AC-F11-41 negative control): the same checker correctly "
        "rejects a fabricated log line naming the live database -- PASS "
        "(detector is capable of failing, not vacuously green)"
    )
else:
    raise SystemExit(
        "negative control FAILED: the checker accepted a fabricated log "
        "line naming the live shark-tasks.db path"
    )
PY

# ===========================================================================
# AC-F11-41a env-scrub proof. The load-bearing half is the static check
# below (isolated_environment() strips exactly the 17 named vars, read
# straight from source). This grep is a cheap belt-and-braces companion,
# not independent proof on its own: `setup` doesn't consume most of these
# 17 vars, and the -e trace= filter only records file syscalls, so a
# poisoned value would rarely have a syscall path to leak through even if
# scrubbing were broken. It stays because it is free and would catch the
# case where a poisoned value ends up embedded in a file path.
# ===========================================================================
if grep -qF "DENIED" "$TRACE_LOG"; then
	fail "a poisoned scrubbed-env value leaked into a resolved path -- see: $(grep -F 'DENIED' "$TRACE_LOG" | head -1)"
fi
echo "TC-111(AC-F11-41a env scrub, belt-and-braces): none of the 17 poisoned scrubbed-env values leaked into any resolved path -- PASS"

python3 - "$BENCHMARK_LIB" <<'PY'
import re
import sys

source_path = sys.argv[1]
source = open(source_path, encoding="utf-8").read()

match = re.search(
    r"^def isolated_environment\(.*?\n(?:.*\n)*?(?=^def )",
    source,
    re.MULTILINE,
)
if not match:
    raise SystemExit("could not locate isolated_environment in e40_benchmark.py")
body = match.group(0)

expected = {
    "SHARK_DB_URL", "SHARK_AUTH_TOKEN", "SHARK_AUTH_TOKEN_FILE", "SHARK_BIN",
    "LIFECYCLE_ADAPTER", "LIFECYCLE_ADAPTER_PATH", "LIFECYCLE_PROVIDER_COMMAND",
    "RUN_LIFECYCLE_BIN", "EVALUATE_LIFECYCLE_BIN", "ENTITY_HISTORY_EXPORT_BIN",
    "LIFECYCLE_SCHEMA", "I05_SCHEMA", "I07_SCHEMA", "E40_RUN_LIFECYCLE_BATCH_BIN",
    "E40_AGGREGATE_LIFECYCLE_BIN", "E40_REPORT_LIFECYCLE_BIN",
    "E40_VERIFY_RETENTION_ROOT_BIN",
}

# Extract ONLY the names passed to environment.pop(name, None) inside the
# `for name in (...):` block -- not every quoted string in the function
# body (which would also pick up the unrelated "SHARK_DB_BACKEND"/"sqlite"
# literals a few lines above and silently mask a real drift in either
# direction).
tuple_match = re.search(r"for name in \(\s*(.*?)\s*\):", body, re.DOTALL)
if not tuple_match:
    raise SystemExit("could not locate the scrubbed-name tuple in isolated_environment()")
found = set(re.findall(r'"([A-Z0-9_]+)"', tuple_match.group(1)))

if found != expected:
    missing = expected - found
    extra = found - expected
    raise SystemExit(
        f"isolated_environment() scrub list drifted from spec.md's 17 names "
        f"-- missing: {sorted(missing)}, extra: {sorted(extra)}"
    )
if len(expected) != 17:
    raise SystemExit(f"expected list itself is not 17 names: {len(expected)}")
print(
    "TC-111(AC-F11-41a positive): isolated_environment() strips all 17 "
    "required names, and no others -- PASS"
)

# Counterfactual: dropping one name from the expected set must be detected
# by the same equality check.
mutated_expected = expected - {"SHARK_BIN"}
if found == mutated_expected:
    raise SystemExit("counterfactual is not meaningful: mutated set still matches")
print(
    "TC-111(AC-F11-41a counterfactual): removing one name from the expected "
    "set is detected as a mismatch by the same check -- PASS"
)
PY

# ===========================================================================
# AC-F11-41a counterfactual (non-existent temp path): this entire test ran
# against a brand-new operator root created by mktemp moments before use
# (asserted above to not pre-exist) and every step still passed -- proving
# no seam depends on, or falls back to, any fixed pre-existing path. A
# second, independently-rooted run corroborates this is not incidental to
# one lucky path.
# ===========================================================================
ISO_ROOT_2="$WORKDIR/iso-setup-second-independent-root"
[[ ! -e "$ISO_ROOT_2" ]] || fail "second operator root must not pre-exist"
second_rc=0
"$OPERATOR" setup --out "$ISO_ROOT_2" --scenario-index "$SEVEN_PACKAGE_INDEX" \
	>"$WORKDIR/second-setup.out" 2>&1 || second_rc=$?
[[ "$second_rc" -eq 0 ]] || fail "second independently-rooted setup failed: $(cat "$WORKDIR/second-setup.out")"
[[ -f "$ISO_ROOT_2/scratch-template/shark-tasks.db" ]] \
	|| fail "second independently-rooted setup did not create its own scratch database"
echo "TC-111(AC-F11-41a counterfactual): two independently-rooted, previously non-existent temp paths both resolve correctly with no fixed fallback -- PASS"

echo "TC-111: six-family scenario-root isolation + live-database non-mutation proof -- ALL PASS"
