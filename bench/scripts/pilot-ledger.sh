#!/usr/bin/env bash
# pilot-ledger.sh --retention-root <retention_root> --record --scenario <id>
#                  --rep <n> --operator <identity> --checklist <checklist.json>
# pilot-ledger.sh --retention-root <retention_root> --verify [--family <f>]
#
# T-E40-F10-006 (spec.md REQ-F-005, ADR-F10-09): the offline pilot-inspection
# attestation ledger. `--record` appends one operator attestation per
# scenario family into <retention_root>/pilot-ledger.jsonl, capturing the
# inspected run reference, the checklist item results, the inspecting
# operator identity, and the digests of the retained artifacts inspected.
# `--verify` re-computes those same digests from the artifacts CURRENTLY
# retained on disk and reports per-family pass/fail.
#
# ADR-F10-09: the gate is an artifact-digest attestation, not a boolean
# flag or a bare run-id reference -- either of those would be unable to
# detect a later change to the retained evidence. `--verify` therefore
# never trusts a stored digest without recomputing it fresh.
#
# Digest computation (bench/reports/lifecycle-baseline-schema.yaml
# digest_rules, matching I-05/I-07/I-08's own digest_rules blocks):
# algorithm sha256, lowercase hex encoding. A single file is digested as
# `file_encoding: sha256_raw_bytes` (raw bytes, no re-serialization). A
# directory (evidence/, transcripts/) is digested as a `canonicalization:
# compact_json_sorted_keys_utf8` document: the sorted list of
# {relative_path, sha256_raw_bytes} for every file it contains, sha256'd.
#
# The eight artifacts digested per (scenario, rep) are exactly
# bench/reports/lifecycle-baseline-schema.yaml's `retention_required_
# artifacts` list (spec.md Data model changes): package.yaml, evidence,
# transcripts, entity-history.json, lifecycle.jsonl, evaluation.jsonl,
# oracle.json, manifest.json.
#
# Per the schema's `digest_rules.empty_artifact_semantics` (code-review-
# 2026-08-20T2138-E40-F10.md findings 1 and 7; corrected by UAT-R3-01 round
# 3): an artifact that exists on disk but is empty (a zero-file directory, a
# zero-byte file) still digests to a real value -- digest_of_path never
# returns None for it. That alone is NOT sufficient to attest or verify it,
# though: both `--record` and `--verify` additionally cross-check
# manifest.json's per-artifact `source_path` for every required artifact
# (all but manifest.json itself). An empty/missing `source_path` is NEVER an
# accepted "not yet wired" gap (UAT-R3-01: retain_pair itself now refuses to
# write such an artifact at all) -- `--record` refuses to attest over it
# (`required_artifact_source_unavailable`) and `--verify` fails a family
# whose retained manifest still shows one, distinctly, as
# `missing_source_provenance`, regardless of digest agreement. A real,
# non-empty `source_path` whose artifact content is nonetheless empty (a
# real source WAS checked and found empty) remains a distinct, separately
# reported defect: `empty_source_artifact`.
#
# UAT round 6 (2026-08-21T233606Z, defect class: "treating a present file,
# digest field, or non-empty provenance string as proof of verified source
# derivation"): a non-empty `source_path` string, by itself, is not proof the
# retained bytes were legitimately derived from anything -- a forged or
# hand-planted manifest.json can carry a fabricated `source_path` and still
# pass every check described above. Both `--record` and `--verify`
# additionally delegate to `lib/verify_pair_retention` (T-E40-F10-004's
# shared completeness/lineage/digest authority), which independently
# recomputes each required artifact's digest and compares it against
# manifest.json's OWN recorded `sha256` claim -- the retention producer's
# own recorded provenance, not just pilot-ledger's own freshly-computed
# digest. Literal `source_path` *reachability* (the path still resolving on
# disk) is deliberately NOT re-checked: several source_path values point
# into run-lifecycle-batch.sh dispatch_pair's ephemeral `pair_work`
# (mktemp -d), which is `rm -rf`'d immediately after retain_pair returns --
# by the time an operator runs --record, that path is routinely already gone
# even for a fully legitimate retention, so requiring it to exist would
# refuse every real attestation.
#
# Ledger row fields match the schema's `pilot_attestation_required_fields`:
# /run_reference, /checklist_results, /operator_identity,
# /inspected_artifact_digests (plus this script's own `family` and
# `recorded_at` bookkeeping fields, which the schema does not forbid).
#
# `--verify` groups the (append-only) ledger by family and uses the LATEST
# row recorded for each family -- a re-attestation supersedes an earlier
# one, it does not need to be deleted. Verifying a family absent from the
# ledger fails with `no_attestation`; verifying a family whose recorded
# digests no longer match the current retained bytes fails with
# `stale_digest` (naming the artifact(s) that changed) -- this is also
# what catches a `--reclaim-incomplete`-quarantined-and-rerun pair: the
# regenerated bytes at the same retention path simply recompute to a
# different digest, with no special-case code required. A family whose
# digests all match but whose manifest.json shows a required artifact with
# no source provenance fails with `missing_source_provenance`; one whose
# manifest.json shows a real-source-checked empty artifact fails with
# `empty_source_artifact` (naming the artifact(s) in either case) instead.
#
# Exit status: 0 record success / verify all-requested-families pass,
# 1 verify found at least one failing family, 2 usage error.
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage: pilot-ledger.sh --retention-root <retention_root> --record \
         --scenario <id> --rep <n> --operator <identity> \
         --checklist <checklist.json>
       pilot-ledger.sh --retention-root <retention_root> --verify [--family <f>]
EOF
	exit 2
}

retention_root=""
action=""
scenario=""
rep=""
operator=""
checklist=""
family=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		[[ $# -ge 2 ]] || usage
		retention_root="$2"
		shift 2
		;;
	--record)
		[[ -z "$action" ]] || usage
		action="record"
		shift
		;;
	--verify)
		[[ -z "$action" ]] || usage
		action="verify"
		shift
		;;
	--scenario)
		[[ $# -ge 2 ]] || usage
		scenario="$2"
		shift 2
		;;
	--rep)
		[[ $# -ge 2 ]] || usage
		rep="$2"
		shift 2
		;;
	--operator)
		[[ $# -ge 2 ]] || usage
		operator="$2"
		shift 2
		;;
	--checklist)
		[[ $# -ge 2 ]] || usage
		checklist="$2"
		shift 2
		;;
	--family)
		[[ $# -ge 2 ]] || usage
		family="$2"
		shift 2
		;;
	--help)
		usage
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$retention_root" ]] || usage
[[ -n "$action" ]] || usage

command -v python3 >/dev/null 2>&1 || {
	echo "pilot-ledger: python3 not found on PATH" >&2
	exit 2
}

[[ -d "$retention_root" ]] || {
	echo "pilot-ledger: retention root not found: $retention_root" >&2
	exit 2
}
RETENTION_ROOT_CANON="$(cd "$retention_root" && pwd)"
LEDGER_PATH="$RETENTION_ROOT_CANON/pilot-ledger.jsonl"

# T-E40-F10-006 rework (code-review-2026-08-20T1731-E40-F10.md finding 13):
# assert_within_out_root is the SAME containment guard run-lifecycle-batch.sh
# and run-review-comparison.sh already use for their own retention paths
# (bench/scripts/lib/path-safety.sh). Its caller contract requires
# out_root_canon to be set via `realpath -m` (not the `cd && pwd` form used
# for RETENTION_ROOT_CANON above, which can preserve a symlink component) --
# followed here exactly so the two canonicalizations can never disagree.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export F10_DIGEST_HELPER="$SCRIPT_DIR/lib/digest_path"
out_root_canon="$(realpath -m -- "$RETENTION_ROOT_CANON")"
# shellcheck source=lib/path-safety.sh
source "$SCRIPT_DIR/lib/path-safety.sh"

# T-E40-F10-006 UAT round 6 (2026-08-21T233606Z, defect class: "treating a
# present file, digest field, or non-empty provenance string as proof of
# verified source derivation"): BENCH_DIR/LIFECYCLE_SCHEMA follow the exact
# convention run-lifecycle-batch.sh/run-review-comparison.sh already use
# (never a private hardcoded copy), so this script's own
# lib/verify_pair_retention invocation below resolves the SAME schema file
# those two drivers already validate retention pairs against.
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
LIFECYCLE_SCHEMA="${LIFECYCLE_SCHEMA:-$BENCH_DIR/reports/lifecycle-baseline-schema.yaml}"
VERIFY_PAIR_RETENTION="$SCRIPT_DIR/lib/verify_pair_retention"
[[ -f "$VERIFY_PAIR_RETENTION" ]] || {
	echo "pilot-ledger: lib/verify_pair_retention not found at $VERIFY_PAIR_RETENTION" >&2
	exit 2
}
[[ -f "$LIFECYCLE_SCHEMA" ]] || {
	echo "pilot-ledger: lifecycle-baseline-schema.yaml not found at $LIFECYCLE_SCHEMA" >&2
	exit 2
}

# The ledger reads and appends the same retained pair namespace as the batch
# and comparison drivers. Serialize all ledger actions with that namespace's
# lock so an attestation cannot observe a pair while another producer replaces
# its artifacts.
LOCK_DIR="$out_root_canon/.lifecycle-batch.lock"
if ! mkdir "$LOCK_DIR"; then
	owner=""
	[[ -f "$LOCK_DIR/pid" ]] && owner="$(<"$LOCK_DIR/pid")"
	if [[ "$owner" =~ ^[0-9]+$ ]] && kill -0 "$owner"; then
		echo "pilot-ledger: retention root is already locked by process $owner: $out_root_canon" >&2
		exit 4
	fi
	if [[ ! "$owner" =~ ^[0-9]+$ ]]; then
		echo "pilot-ledger: retention root lock has no valid owner: $out_root_canon" >&2
		exit 4
	fi
	rm -f "$LOCK_DIR/pid"
	if ! rmdir "$LOCK_DIR" || ! mkdir "$LOCK_DIR"; then
		echo "pilot-ledger: retention root lock could not be recovered: $out_root_canon" >&2
		exit 4
	fi
fi
printf '%s\n' "$$" >"$LOCK_DIR/pid"
trap 'rm -f "$LOCK_DIR/pid"; rmdir "$LOCK_DIR" 2>/dev/null || true' EXIT

# REQ-F-002's scenario_id grammar (bench/scenarios/packages/<scenario_id>/
# package.yaml identity block: "unique lowercase-kebab identity"), the same
# closed grammar the I-04 scenario contract test (tests/contracts,
# e40I04ScenarioIDPattern) enforces for I-04 admission. --scenario is interpolated
# into a retention-relative path below; validating it against this grammar
# up front means it can never contain '/', '.', or an absolute-path prefix,
# so a traversal escape is structurally impossible before any file is
# touched.
SCENARIO_ID_GRAMMAR='^[a-z0-9]+(-[a-z0-9]+)*$'

case "$action" in
record)
	[[ -n "$scenario" ]] || usage
	[[ "$scenario" =~ $SCENARIO_ID_GRAMMAR ]] || {
		echo "pilot-ledger: --scenario must be a lowercase-kebab scenario_id (REQ-F-002 grammar: ^[a-z0-9]+(-[a-z0-9]+)*\$), got '$scenario'" >&2
		exit 2
	}
	[[ -n "$rep" ]] || usage
	[[ "$rep" =~ ^[0-9]+$ ]] || {
		echo "pilot-ledger: --rep must be a non-negative integer, got '$rep'" >&2
		exit 2
	}
	[[ -n "$operator" ]] || usage
	[[ -n "$checklist" ]] || usage
	[[ -f "$checklist" ]] || {
		echo "pilot-ledger: checklist file not found: $checklist" >&2
		exit 2
	}

	# Containment check on the constructed retention-relative path, BEFORE
	# any read or write against it -- defense in depth alongside the
	# grammar check above (finding 13's fix instruction: validate the
	# grammar AND containment-check the constructed path, exactly as
	# run-lifecycle-batch.sh/run-review-comparison.sh already do for their
	# own retention paths).
	scenario_dir_precheck="$RETENTION_ROOT_CANON/scenarios/$scenario/$rep"
	assert_within_out_root "$scenario_dir_precheck" || exit 2

	python3 - "$RETENTION_ROOT_CANON" "$scenario" "$rep" "$operator" "$checklist" "$LEDGER_PATH" \
		"$VERIFY_PAIR_RETENTION" "$LIFECYCLE_SCHEMA" <<'PYEOF'
import hashlib
import json
import os
import subprocess
import sys
import time

try:
    import yaml
except ImportError:
    print("pilot-ledger: PyYAML not available", file=sys.stderr)
    raise SystemExit(2)

(
    retention_root, scenario_id, rep_str, operator, checklist_path, ledger_path,
    verify_pair_retention_bin, lifecycle_schema,
) = sys.argv[1:9]
rep = int(rep_str)

# The eight canonical retained artifacts (bench/reports/
# lifecycle-baseline-schema.yaml retention_required_artifacts).
RETAINED_ARTIFACTS = [
    "package.yaml", "evidence", "transcripts", "entity-history.json",
    "lifecycle.jsonl", "evaluation.jsonl", "oracle.json", "manifest.json",
]


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def digest_of_path(path):
    result = subprocess.run(
        [sys.executable, os.environ["F10_DIGEST_HELPER"], path],
        capture_output=True, text=True, check=False,
    )
    digest = result.stdout.strip()
    return digest if result.returncode == 0 and digest else None


def symlink_in_path(path):
    current = os.path.abspath(path)
    while True:
        if os.path.islink(current):
            return True
        parent = os.path.dirname(current)
        if parent == current:
            break
        current = parent
    if os.path.isdir(path):
        for root, dirs, files in os.walk(path, followlinks=False):
            if any(os.path.islink(os.path.join(root, name)) for name in dirs + files):
                return True
    return False


# T-E40-F10-006 rework finding 13: --scenario is already validated against
# REQ-F-002's lowercase-kebab grammar and containment-checked by the caller
# (assert_within_out_root) before this subprocess is even started -- both
# BEFORE any read or write, per the finding's fix instruction. This
# realpath-based re-check is belt-and-suspenders against this exact
# subprocess someday being invoked directly with an unvalidated argv,
# independent of the bash-side guard.
RETENTION_ROOT_REAL = os.path.realpath(retention_root)


def within_retention_root(path):
    real = os.path.realpath(path)
    return real == RETENTION_ROOT_REAL or real.startswith(RETENTION_ROOT_REAL + os.sep)


scenario_dir = os.path.join(retention_root, "scenarios", scenario_id, str(rep))
if not within_retention_root(scenario_dir):
    print(f"pilot-ledger: refusing --scenario '{scenario_id}': resolves outside retention root", file=sys.stderr)
    raise SystemExit(2)
if not os.path.isdir(scenario_dir):
    print(f"pilot-ledger: no retained scenario directory found: {scenario_dir}", file=sys.stderr)
    raise SystemExit(2)
for name in RETAINED_ARTIFACTS:
    if symlink_in_path(os.path.join(scenario_dir, name)):
        print(f"pilot-ledger: retained artifact path contains a symlink: {name}", file=sys.stderr)
        raise SystemExit(2)

package_path = os.path.join(scenario_dir, "package.yaml")
if not os.path.isfile(package_path):
    print(f"pilot-ledger: package.yaml missing under {scenario_dir}", file=sys.stderr)
    raise SystemExit(2)
with open(package_path) as f:
    pkg = yaml.safe_load(f) or {}
family = str(pkg.get("entity_family") or "")
if not family:
    print(f"pilot-ledger: {package_path} has no entity_family; cannot determine scenario family", file=sys.stderr)
    raise SystemExit(2)

digests = {}
missing = []
for name in RETAINED_ARTIFACTS:
    digest = digest_of_path(os.path.join(scenario_dir, name))
    if digest is None:
        missing.append(name)
    digests[name] = digest
if missing:
    print(
        f"pilot-ledger: cannot record attestation, missing retained artifact(s) under {scenario_dir}: {', '.join(missing)}",
        file=sys.stderr,
    )
    raise SystemExit(2)

# UAT-R3-01 (round 3), T-E40-F10-006 fix requirement 1 ("sweep every ledger
# completeness AND digest-verification branch"): a digest existing on disk
# is not proof the artifact has a real source -- retain_pair's pre-fix
# fabrication wrote a real, present, digestible zero-byte/empty placeholder
# with source_path == "" for a required artifact and still returned success.
# --record MUST refuse to attest over that placeholder, not just over an
# artifact literally absent from disk. manifest.json (itself one of
# RETAINED_ARTIFACTS) carries the per-artifact source_path provenance
# retain_pair wrote; read it here before recording anything.
manifest_path = os.path.join(scenario_dir, "manifest.json")
try:
    with open(manifest_path, encoding="utf-8") as f:
        manifest = json.load(f)
except (OSError, json.JSONDecodeError) as exc:
    print(f"pilot-ledger: cannot read manifest.json: {exc}", file=sys.stderr)
    raise SystemExit(2)
if not isinstance(manifest, dict):
    print("pilot-ledger: manifest.json must contain a JSON object", file=sys.stderr)
    raise SystemExit(2)
manifest_artifacts = manifest.get("artifacts") or {}
if not isinstance(manifest_artifacts, dict):
    print("pilot-ledger: manifest.json artifacts must be a JSON object", file=sys.stderr)
    raise SystemExit(2)
missing_source_provenance = []
for name in RETAINED_ARTIFACTS:
    if name == "manifest.json":
        continue
    source_path = (manifest_artifacts.get(name) or {}).get("source_path")
    if not isinstance(source_path, str) or not source_path:
        missing_source_provenance.append(name)
if missing_source_provenance:
    print(
        f"pilot-ledger: cannot record attestation, required retained artifact(s) under {scenario_dir} "
        f"have no real source provenance (required_artifact_source_unavailable): {', '.join(sorted(set(missing_source_provenance)))}",
        file=sys.stderr,
    )
    raise SystemExit(2)

# UAT round 6 (2026-08-21T233606Z, defect class: "treating a present file,
# digest field, or non-empty provenance string as proof of verified source
# derivation"): the checks above establish that every required artifact's
# manifest.json entry NAMES a non-empty source_path -- they never confirm
# that claim is honest. A forged manifest.json can carry a fabricated
# source_path alongside a `sha256` value that happens to agree with the
# retained bytes; nothing above would ever catch that, because nothing
# above ever reads manifest.json's own per-artifact `sha256` field. Delegate
# to lib/verify_pair_retention (T-E40-F10-004's single shared authority for
# "is this (scenario, rep) pair really complete, lineage-correct, and
# byte-preserved") to independently recompute each artifact's digest and
# compare it against manifest.json's OWN recorded claim, rather than
# re-implementing that check a second time here.
#
# Deliberately NOT re-checked here: whether source_path itself still
# resolves on disk. lifecycle.jsonl/evaluation.jsonl/entity-history.json's
# source_path values point into run-lifecycle-batch.sh dispatch_pair's
# ephemeral `pair_work` (mktemp -d), which that driver rm -rf's
# (run-lifecycle-batch.sh:823) immediately after retain_pair returns --by
# the time an operator runs --record (a separate, later, offline pilot-
# inspection step), that path is routinely already gone even for a fully
# legitimate retention. Requiring it to still exist would refuse every real
# attestation, not just forged ones. Cross-checking manifest.json's own
# recorded digest against the artifact's CURRENT retained bytes (below) is
# the strongest independent verification achievable without depending on an
# ephemeral source surviving to inspection time.
verify_result = subprocess.run(
    [sys.executable, verify_pair_retention_bin, scenario_id, str(rep), scenario_dir, lifecycle_schema],
    capture_output=True,
    text=True,
)
if verify_result.returncode != 0:
    token = (verify_result.stdout or "").strip().splitlines()[-1] if verify_result.stdout.strip() else "verification_failed:unknown"
    print(
        f"pilot-ledger: cannot record attestation, retained pair at {scenario_dir} failed independent "
        f"provenance verification ({token}) -- manifest.json's own recorded per-artifact digest could not "
        f"be confirmed against the currently retained bytes",
        file=sys.stderr,
    )
    if verify_result.stderr:
        print(verify_result.stderr, file=sys.stderr, end="")
    raise SystemExit(2)

with open(checklist_path) as f:
    checklist_results = json.load(f)

record = {
    "family": family,
    "run_reference": {
        "scenario_id": scenario_id,
        "rep": rep,
        "retention_path": f"scenarios/{scenario_id}/{rep}",
    },
    "operator_identity": operator,
    "checklist_results": checklist_results,
    "inspected_artifact_digests": digests,
    "recorded_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
}
# Open the final ledger entry with O_NOFOLLOW.  A pre-existing symlink at the
# ledger path must never turn an attestation append into a write to an
# operator-unrelated file.  The flag closes the final-component race between
# an lstat-style check and the append itself; the canonical retention root
# above remains the trusted parent anchor.
if os.path.islink(ledger_path):
    print(f"pilot-ledger: refusing to append through symlink: {ledger_path}", file=sys.stderr)
    raise SystemExit(1)
ledger_flags = os.O_WRONLY | os.O_APPEND | os.O_CREAT
if hasattr(os, "O_NOFOLLOW"):
    ledger_flags |= os.O_NOFOLLOW
try:
    ledger_fd = os.open(ledger_path, ledger_flags, 0o600)
except OSError as exc:
    print(f"pilot-ledger: refusing to open ledger safely: {exc}", file=sys.stderr)
    raise SystemExit(1) from exc
with os.fdopen(ledger_fd, "a", encoding="utf-8") as f:
    f.write(json.dumps(record, sort_keys=True) + "\n")

print(f"pilot-ledger: recorded attestation for family '{family}' (scenario_id={scenario_id}, rep={rep})")
PYEOF
	exit $?
	;;
verify)
	python3 - "$RETENTION_ROOT_CANON" "$LEDGER_PATH" "$family" \
		"$VERIFY_PAIR_RETENTION" "$LIFECYCLE_SCHEMA" <<'PYEOF'
import hashlib
import json
import os
import re
import subprocess
import sys

retention_root, ledger_path, family_filter, verify_pair_retention_bin, lifecycle_schema = sys.argv[1:6]

RETAINED_ARTIFACTS = [
    "package.yaml", "evidence", "transcripts", "entity-history.json",
    "lifecycle.jsonl", "evaluation.jsonl", "oracle.json", "manifest.json",
]

# REQ-F-002 lowercase-kebab scenario_id grammar -- the same closed grammar
# --record validates argv against and the I-04 scenario contract test
# (tests/contracts, e40I04ScenarioIDPattern) enforces for I-04 admission.
# Applied here too because
# `ref.get("scenario_id")` below comes from the (append-only, hand-editable)
# ledger file, not a freshly-validated argv -- a row written before this fix
# landed, or a hand-edited ledger, could still carry an unsafe value.
SCENARIO_ID_PATTERN = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")

RETENTION_ROOT_REAL = os.path.realpath(retention_root)


def within_retention_root(path):
    real = os.path.realpath(path)
    return real == RETENTION_ROOT_REAL or real.startswith(RETENTION_ROOT_REAL + os.sep)


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def digest_of_path(path):
    result = subprocess.run(
        [sys.executable, os.environ["F10_DIGEST_HELPER"], path],
        capture_output=True, text=True, check=False,
    )
    digest = result.stdout.strip()
    return digest if result.returncode == 0 and digest else None


def symlink_in_path(path):
    current = os.path.abspath(path)
    while True:
        if os.path.islink(current):
            return True
        parent = os.path.dirname(current)
        if parent == current:
            break
        current = parent
    if os.path.isdir(path):
        for root, dirs, files in os.walk(path, followlinks=False):
            if any(os.path.islink(os.path.join(root, name)) for name in dirs + files):
                return True
    return False


# The two canonical "empty" digests per digest_rules.empty_artifact_semantics:
# a directory with zero files (canonicalization applied to the empty file
# list `[]`), and a zero-byte file (file_encoding of the empty byte
# string). Used only by the source_path cross-check below -- digest_of_path
# itself does not need to know which "empty" shape it computed.
EMPTY_DIR_DIGEST = sha256_bytes(json.dumps([], sort_keys=True, separators=(",", ":")).encode("utf-8"))
EMPTY_FILE_DIGEST = sha256_bytes(b"")


# Group by family; an append-only ledger's LATEST row per family is the
# current attestation -- a re-record supersedes an earlier one.
latest_by_family = {}
if os.path.isfile(ledger_path):
    with open(ledger_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue
            if not isinstance(entry, dict):
                continue
            fam = entry.get("family")
            if fam:
                latest_by_family[fam] = entry

targets = [family_filter] if family_filter else sorted(latest_by_family.keys())

any_fail = False
for fam in targets:
    entry = latest_by_family.get(fam)
    if entry is None:
        print(f"family={fam}: FAILED (no_attestation)")
        any_fail = True
        continue
    ref = entry.get("run_reference") or {}
    scenario_id = ref.get("scenario_id")
    rep = ref.get("rep")
    # Finding 13's fix, applied on the read side: validate the grammar AND
    # containment-check the constructed path BEFORE any read, exactly as
    # --record now does -- a ledger row is data, not a trusted argv, so a
    # malformed/unsafe scenario_id here must fail the family's verification
    # rather than being dereferenced.
    if (
        not isinstance(scenario_id, str)
        or not SCENARIO_ID_PATTERN.match(scenario_id)
        or not isinstance(rep, int)
        or rep < 0
    ):
        print(f"family={fam}: FAILED (unsafe_scenario_reference)")
        any_fail = True
        continue
    scenario_dir = os.path.join(retention_root, "scenarios", scenario_id, str(rep))
    if not within_retention_root(scenario_dir):
        print(f"family={fam}: FAILED (unsafe_scenario_reference)")
        any_fail = True
        continue
    if any(symlink_in_path(os.path.join(scenario_dir, name)) for name in RETAINED_ARTIFACTS):
        print(f"family={fam}: FAILED (schema_invalid: retained artifact path contains a symlink)")
        any_fail = True
        continue
    recorded_digests = entry.get("inspected_artifact_digests") or {}
    # Advisor review of T-E40-F10-006's rework (this file's own comment on
    # digest_of_path already names the failure mode): a recorded digest that
    # is missing or not a non-empty string -- a forged/hand-edited row, or a
    # pre-fix --record that never wrote a key for an artifact -- makes
    # `current != recorded_digests.get(name)` compare None == None when the
    # artifact is ALSO currently missing on disk, which is not a staleness
    # mismatch and would wrongly report "verified" with zero digests
    # actually inspected. Every recorded digest must be a real non-empty
    # string BEFORE the staleness comparison runs, so None can never match
    # None.
    incomplete = [
        name
        for name in RETAINED_ARTIFACTS
        if not isinstance(recorded_digests.get(name), str) or not recorded_digests[name]
    ]
    if incomplete:
        print(f"family={fam}: FAILED (incomplete_attestation: {', '.join(incomplete)})")
        any_fail = True
        continue
    # bench/reports/lifecycle-baseline-schema.yaml digest_rules.
    # empty_artifact_semantics' second axis (code-review-2026-08-20T2138-
    # E40-F10.md findings 1 and 7, corrected by UAT-R3-01 round 3):
    # digest_of_path alone cannot tell "no source was ever available to
    # check" (source_path=="" on a REQUIRED artifact -- UAT-R3-01: never an
    # accepted gap, always a hard failure) apart from "a real source was
    # checked and found empty" (source_path!="", a genuine producer-defect
    # signal, reported distinctly) -- both digest identically. manifest.json
    # (itself one of the eight retained artifacts, written by
    # retain_pair/retain_gate) is where that provenance lives; read it here.
    # A missing/malformed manifest.json is ALSO caught by the staleness loop
    # below (its own digest_of_path/recorded-digest comparison) whenever the
    # corruption happened AFTER --record ran. UAT round 6 sweep site: when it
    # happened BEFORE --record (the ledger's own recorded digest already
    # reflects the broken file, so it is not "stale"), silently degrading to
    # an empty dict here used to let every non-manifest artifact's source_path
    # lookup return None and fall through the missing_source_provenance branch
    # below with no explanation of WHY -- correct in effect, but it buries an
    # unreadable-manifest condition inside a differently-named reason instead
    # of reporting it directly. Track the read failure explicitly so it is
    # named, not just implied.
    manifest_artifacts = {}
    manifest_read_error = None
    try:
        with open(os.path.join(scenario_dir, "manifest.json"), encoding="utf-8") as f:
            manifest_artifacts = (json.load(f) or {}).get("artifacts") or {}
    except (OSError, ValueError) as exc:
        manifest_read_error = str(exc)

    stale = []
    missing_source_provenance = []
    empty_source_defect = []
    for name in RETAINED_ARTIFACTS:
        current = digest_of_path(os.path.join(scenario_dir, name))
        if current != recorded_digests.get(name):
            stale.append(name)
            continue
        if name == "manifest.json":
            continue
        source_path = (manifest_artifacts.get(name) or {}).get("source_path")
        if not isinstance(source_path, str) or not source_path:
            # UAT-R3-01 (round 3): a required artifact with no real source
            # provenance is NEVER an accepted "not yet wired" state,
            # regardless of digest agreement -- retain_pair's pre-fix
            # fabrication is exactly a digest that matches its own recorded
            # (empty) value while source_path stays "".
            missing_source_provenance.append(name)
            continue
        if current in (EMPTY_DIR_DIGEST, EMPTY_FILE_DIGEST):
            empty_source_defect.append(name)

    failure_reasons = []
    if stale:
        failure_reasons.append(f"stale_digest: {', '.join(stale)}")
    if missing_source_provenance:
        failure_reasons.append(f"missing_source_provenance: {', '.join(missing_source_provenance)}")
    if empty_source_defect:
        failure_reasons.append(f"empty_source_artifact: {', '.join(empty_source_defect)}")
    if manifest_read_error is not None:
        failure_reasons.append(f"unreadable_manifest: manifest.json ({manifest_read_error})")

    # UAT round 6 (2026-08-21T233606Z, defect class: "treating a present
    # file, digest field, or non-empty provenance string as proof of
    # verified source derivation"): everything above checks pilot-ledger's
    # OWN recorded digest against the current bytes (staleness) and whether
    # manifest.json NAMES a source_path -- neither ever reads manifest.json's
    # own per-artifact `sha256` claim (the retention producer's own recorded
    # provenance) and compares it against the CURRENT retained bytes. A
    # forged manifest.json with a fabricated-but-non-empty source_path and a
    # `sha256` value crafted to agree with the retained bytes sails through
    # every check above undetected. Delegate to lib/verify_pair_retention
    # (T-E40-F10-004's shared completeness/lineage/digest authority) rather
    # than re-implementing that comparison a second time here -- it is run
    # unconditionally, independent of whatever the checks above already
    # found, so a manifest that is internally self-consistent with the
    # retained bytes but was never really produced by retain_pair (e.g. a
    # lineage mismatch, or a missing artifact this loop's own None-vs-None
    # guard already defends against) is caught here too.
    verify_result = subprocess.run(
        [sys.executable, verify_pair_retention_bin, scenario_id, str(rep), scenario_dir, lifecycle_schema],
        capture_output=True,
        text=True,
    )
    if verify_result.returncode != 0:
        stdout = (verify_result.stdout or "").strip()
        token = stdout.splitlines()[-1] if stdout else "verification_failed:unknown"
        vr_reason, _, vr_artifact = token.partition(":")
        failure_reasons.append(f"{vr_reason or 'verification_failed'}: {vr_artifact or 'unknown'}")

    if failure_reasons:
        print(f"family={fam}: FAILED ({'; '.join(failure_reasons)})")
        any_fail = True
    else:
        print(f"family={fam}: verified")

if not targets:
    print("pilot-ledger: no attestations recorded", file=sys.stderr)

raise SystemExit(1 if any_fail else 0)
PYEOF
	exit $?
	;;
esac
