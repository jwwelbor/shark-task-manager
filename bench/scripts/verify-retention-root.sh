#!/usr/bin/env bash
# verify-retention-root.sh --retention-root <retention_root> \
#                           --schema bench/reports/lifecycle-baseline-schema.yaml
#
# T-E40-F10-007 (spec.md REQ-F-004, REQ-NF-007; component-changes row for
# this file): the offline retention validator. It NEVER repairs a root --
# read-only, fail-closed, and bounded. For every retained (scenario, rep)
# pair under <retention_root>/scenarios/<scenario_id>/<rep>/ it runs, in
# order:
#
#   1. Layout completeness -- every artifact in the schema's
#      `retention_required_artifacts` list must be present. AC-T2: if this
#      phase fails (including `manifest.json` itself being absent), NO
#      digest check is attempted for that pair -- phase 2 requires reading
#      manifest.json, so a missing/incomplete layout short-circuits before
#      any digest work.
#   2. Pair-level lineage -- manifest.json's own `/scenario_id` and `/rep`
#      fields must agree with the `scenarios/<scenario_id>/<rep>/` directory
#      the manifest was found in (spec.md Architecture component-changes row:
#      "pair" is this feature's (scenario, rep) unit throughout --
#      retain_pair/classify_pair/quarantine_pair in run-lifecycle-batch.sh).
#      This is a manifest-vs-directory check only: per this task's
#      Integration Contracts ("does not itself read I-05/I-07/I-08 semantic
#      fields"), it never inspects lifecycle.jsonl/evaluation.jsonl content.
#   3. Per-artifact digest equality against manifest.json's recorded source
#      digests, for the seven non-manifest retained artifacts. Digest
#      computation reuses T-E40-F10-006's (pilot-ledger.sh) exact pattern:
#      a file digests as raw bytes (sha256_raw_bytes); a directory
#      (evidence/, transcripts/) digests as the sha256 of the compact
#      sorted-key JSON list of {path, sha256} for every file it contains
#      (compact_json_sorted_keys_utf8) -- both per
#      bench/reports/lifecycle-baseline-schema.yaml `digest_rules`.
#        - Digest equality is the authoritative byte-preservation proof; a
#          digest MATCH always passes regardless of source_path, because a
#          recorded source_path is provenance metadata that is EXPECTED to
#          go stale once byte preservation is already confirmed (a real
#          producer's ephemeral pair_work path is deleted right after
#          retention -- run-lifecycle-batch.sh retain_pair). Only when a
#          digest MISMATCHES is source_path consulted, to give that
#          mismatch a more specific reason: a non-empty recorded
#          `source_path` that no longer exists on disk is reported as
#          `source_path_missing` (TC-082 Edge Case) rather than
#          `digest_mismatch`/`re_serialized`, since the mismatch can no
#          longer be re-verified against its source at all. An empty
#          `source_path` is this codebase's existing placeholder
#          convention for an artifact not yet wired by its producer and is
#          never treated as "missing" either way.
#        - On a raw-byte mismatch for a JSON/JSONL artifact
#          (entity-history.json, lifecycle.jsonl, evaluation.jsonl,
#          oracle.json), a second check is attempted: parse the CURRENT
#          file's line(s) as JSON and re-serialize each canonically
#          (compact, sorted keys, UTF-8, matching this repository's
#          existing `json.dumps(..., sort_keys=True, separators=(",",
#          ":"))` producer convention), rejoined with a trailing newline. If
#          that recomputed canonical digest equals the recorded digest, the
#          artifact's *content* is unchanged and only its *encoding*
#          differs -- ADR-F10-05 forbids this ("re-serializing would
#          silently create a second, divergent copy") just as much as an
#          outright mutation, so it is still a failure, but named
#          `re_serialized` rather than `digest_mismatch` so an operator can
#          tell the two apart. `package.yaml` (YAML, not JSON) and the two
#          directory artifacts have no such second check and always report
#          `digest_mismatch`.
#   4. Upstream schema validity, delegated (never duplicated) to the
#      existing `verify-lifecycle-run.sh` (lifecycle.jsonl, i07-schema.yaml)
#      and `verify-lifecycle-evaluation.sh` (evaluation.jsonl,
#      i08-schema.yaml). A well-formed-but-ineligible I-08 record (exit 1
#      with a verdict on stdout, per that script's own contract: it exits
#      `0 if aggregate_eligible else 1` even for a structurally valid
#      record) is NOT a retention-root defect -- it is a normal outcome
#      `aggregate-lifecycle.sh`'s `invalid/index.jsonl` is designed to
#      carry -- and is reported only informationally. Only a malformed
#      record (empty stdout, or delegate exit 2) is a validator failure
#      here, named `upstream_lifecycle_invalid` / `upstream_evaluation_invalid`.
#
# Reason vocabulary (local to this script; the eight-artifact retention
# LAYOUT itself is schema-owned per `retention_required_artifacts` /
# `retention_optional_artifacts`, read below, but this validator's own
# pass/fail reason codes are not added to
# bench/reports/lifecycle-baseline-schema.yaml -- TC-078's contract test
# owns that schema's exact required-key shape and this task's exit gate
# forbids unrelated changes, so the codes are documented here instead):
#   missing                    -- required artifact absent (phase 1)
#   lineage_mismatch           -- manifest.json /scenario_id or /rep
#                                 disagrees with its own directory location
#   source_path_missing        -- digest already mismatched AND the
#                                 recorded non-empty source_path no longer
#                                 exists (so the mismatch cannot be
#                                 re-verified against source); never fires
#                                 on a digest that matches
#   digest_mismatch            -- recomputed digest disagrees with the
#                                 recorded one and no canonical-content
#                                 match rescues it
#   re_serialized               -- recomputed raw digest disagrees, but the
#                                 artifact's re-canonicalized content digest
#                                 equals the recorded one
#   upstream_lifecycle_invalid  -- verify-lifecycle-run.sh rejected
#                                 lifecycle.jsonl (malformed, not merely
#                                 ineligible)
#   upstream_evaluation_invalid -- verify-lifecycle-evaluation.sh rejected
#                                 evaluation.jsonl (malformed, not merely
#                                 ineligible)
#
# Emits a bounded verdict: one JSON line per (scenario, rep) pair on
# stdout (`{"scenario_id":...,"rep":...,"verdict":"pass"|"fail",
# "failures":[{"artifact":...,"reason":...,"detail":...}, ...]}`), plus one
# human-readable "<scenario_id>/<rep>: <artifact>: <reason>: <detail>" line
# per failure on stderr. Exit status: 0 every pair passes, 1 at least one
# pair fails, 2 usage error.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# TD-077 defect-class precedent (sibling path, never bare PATH resolution;
# matches run-lifecycle-batch.sh's RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN
# and lib/spend-gate.sh's PILOT_LEDGER_BIN).
VERIFY_LIFECYCLE_RUN_BIN="${VERIFY_LIFECYCLE_RUN_BIN:-$SCRIPT_DIR/verify-lifecycle-run.sh}"
VERIFY_LIFECYCLE_EVALUATION_BIN="${VERIFY_LIFECYCLE_EVALUATION_BIN:-$SCRIPT_DIR/verify-lifecycle-evaluation.sh}"
I07_SCHEMA="${I07_SCHEMA:-$BENCH_DIR/runs/i07-schema.yaml}"
I08_SCHEMA="${I08_SCHEMA:-$BENCH_DIR/evaluation/i08-schema.yaml}"

usage() {
	echo "usage: verify-retention-root.sh --retention-root <retention_root> --schema <lifecycle-baseline-schema.yaml>" >&2
	exit 2
}

retention_root=""
schema_path=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		[[ $# -ge 2 ]] || usage
		retention_root="$2"
		shift 2
		;;
	--schema)
		[[ $# -ge 2 ]] || usage
		schema_path="$2"
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

[[ -n "$retention_root" && -n "$schema_path" ]] || usage
[[ -d "$retention_root" ]] || {
	echo "verify-retention-root: retention root not found: $retention_root" >&2
	exit 2
}
[[ -f "$schema_path" ]] || {
	echo "verify-retention-root: schema not found: $schema_path" >&2
	exit 2
}
[[ -x "$VERIFY_LIFECYCLE_RUN_BIN" ]] || {
	echo "verify-retention-root: verify-lifecycle-run.sh not found or not executable: $VERIFY_LIFECYCLE_RUN_BIN" >&2
	exit 2
}
[[ -x "$VERIFY_LIFECYCLE_EVALUATION_BIN" ]] || {
	echo "verify-retention-root: verify-lifecycle-evaluation.sh not found or not executable: $VERIFY_LIFECYCLE_EVALUATION_BIN" >&2
	exit 2
}
[[ -f "$I07_SCHEMA" ]] || {
	echo "verify-retention-root: i07 schema not found: $I07_SCHEMA" >&2
	exit 2
}
[[ -f "$I08_SCHEMA" ]] || {
	echo "verify-retention-root: i08 schema not found: $I08_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-retention-root: python3 not found on PATH" >&2
	exit 2
}

RETENTION_ROOT_CANON="$(cd "$retention_root" && pwd)"
SCENARIOS_DIR="$RETENTION_ROOT_CANON/scenarios"
[[ -d "$SCENARIOS_DIR" ]] || {
	echo "verify-retention-root: no scenarios/ directory under retention root: $RETENTION_ROOT_CANON" >&2
	exit 2
}

# Phase 1+2: layout completeness and manifest presence, driven by the
# schema's own retention_required_artifacts list (REQ-F-018). Emits one
# "<scenario_id> <rep>" pending-pair line per discovered (scenario, rep)
# directory to a temp file for the phase-2/3 python step below, and prints
# any layout failures immediately (so AC-T2's ordering is structural: phase
# 2/3 python is never invoked for a pair whose layout failed).
overall_status=0
PENDING="$(mktemp)"
trap 'rm -f "$PENDING"' EXIT

for scenario_dir in "$SCENARIOS_DIR"/*/; do
	[[ -d "$scenario_dir" ]] || continue
	scenario_id="$(basename "$scenario_dir")"
	for rep_dir in "$scenario_dir"*/; do
		[[ -d "$rep_dir" ]] || continue
		rep="$(basename "$rep_dir")"
		rep_dir_canon="$(cd "$rep_dir" && pwd)"

		layout_failures="$(python3 - "$rep_dir_canon" "$schema_path" <<'PYEOF'
import os
import sys
import yaml

rep_dir, schema_path = sys.argv[1:3]
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}
required = schema.get("retention_required_artifacts") or []
for name in required:
    if not os.path.exists(os.path.join(rep_dir, name)):
        print(name)
PYEOF
		)"

		if [[ -n "$layout_failures" ]]; then
			overall_status=1
			failures_json="[]"
			while IFS= read -r artifact; do
				[[ -n "$artifact" ]] || continue
				echo "verify-retention-root: $scenario_id/$rep: $artifact: missing: required retained artifact is absent" >&2
				failures_json="$(python3 -c 'import json,sys; arr=json.loads(sys.argv[1]); arr.append({"artifact": sys.argv[2], "reason": "missing", "detail": "required retained artifact is absent"}); print(json.dumps(arr, sort_keys=True, separators=(",", ":")))' "$failures_json" "$artifact")"
			done <<<"$layout_failures"
			python3 -c 'import json,sys; print(json.dumps({"scenario_id": sys.argv[1], "rep": sys.argv[2], "verdict": "fail", "failures": json.loads(sys.argv[3])}, sort_keys=True, separators=(",", ":")))' "$scenario_id" "$rep" "$failures_json"
			continue
		fi

		echo "$scenario_id|$rep|$rep_dir_canon" >>"$PENDING"
	done
done

# Phase 3+4 (only for pairs whose layout was complete): digest equality,
# pair-level lineage, then delegated upstream schema validity.
if [[ -s "$PENDING" ]]; then
	set +e
	python3 - "$PENDING" "$I07_SCHEMA" "$I08_SCHEMA" "$VERIFY_LIFECYCLE_RUN_BIN" "$VERIFY_LIFECYCLE_EVALUATION_BIN" <<'PYEOF'
import hashlib
import json
import os
import subprocess
import sys

pending_path, i07_schema, i08_schema, verify_run_bin, verify_eval_bin = sys.argv[1:6]

JSON_ARTIFACTS = {"entity-history.json", "lifecycle.jsonl", "evaluation.jsonl", "oracle.json"}


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def digest_of_path(path):
    # Mirrors pilot-ledger.sh's digest_of_path exactly (T-E40-F10-006
    # pattern reuse): file_encoding: sha256_raw_bytes for a single file;
    # canonicalization: compact_json_sorted_keys_utf8 over the sorted
    # {path, sha256} list of every file for a directory.
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": sha256_bytes(fh.read())})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return sha256_bytes(canonical)
    if os.path.isfile(path):
        with open(path, "rb") as fh:
            return sha256_bytes(fh.read())
    return None


def canonical_json_digest(path):
    # Secondary check for JSON/JSONL artifacts only: re-serialize the
    # parsed content compactly with sorted keys, matching this
    # repository's existing producer convention (e.g.
    # run-lifecycle-batch.sh retain_pair / pilot-ledger.sh:
    # `json.dumps(..., sort_keys=True, separators=(",", ":")) + "\n"`).
    # Returns None if the content cannot be parsed as JSON at all.
    #
    # Two shapes are tried, in order:
    #   1. the whole file as ONE JSON document (a pretty-printed,
    #      re-indented, or key-reordered re-serialization of a single JSON
    #      object spans multiple lines and is not itself valid line-by-line
    #      JSONL);
    #   2. one JSON object per non-empty line (JSONL: lifecycle.jsonl /
    #      evaluation.jsonl), each re-serialized and rejoined with "\n".
    try:
        with open(path, "r", encoding="utf-8") as fh:
            raw = fh.read()
    except OSError:
        return None
    if not raw.strip():
        return None
    try:
        obj = json.loads(raw)
        canonical = (json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n").encode("utf-8")
        return sha256_bytes(canonical)
    except json.JSONDecodeError:
        pass
    lines = [ln for ln in raw.split("\n") if ln.strip() != ""]
    canon_lines = []
    for ln in lines:
        try:
            obj = json.loads(ln)
        except json.JSONDecodeError:
            return None
        canon_lines.append(json.dumps(obj, sort_keys=True, separators=(",", ":")))
    canonical = ("\n".join(canon_lines) + "\n").encode("utf-8")
    return sha256_bytes(canonical)


def run_delegate(binary, artifact_path, schema_path):
    proc = subprocess.run(
        [binary, artifact_path, "--schema", schema_path],
        capture_output=True, text=True, check=False,
    )
    return proc.returncode, proc.stdout, proc.stderr


overall_fail = False

with open(pending_path, encoding="utf-8") as f:
    pairs = [line.rstrip("\n").split("|", 2) for line in f if line.strip()]

for scenario_id, rep, rep_dir in pairs:
    failures = []
    manifest_path = os.path.join(rep_dir, "manifest.json")
    try:
        with open(manifest_path, encoding="utf-8") as f:
            manifest = json.load(f)
    except (OSError, json.JSONDecodeError) as exc:
        failures.append({"artifact": "manifest.json", "reason": "digest_mismatch", "detail": f"manifest.json is not valid JSON: {exc}"})
        manifest = None

    if manifest is not None:
        # Phase 2: pair-level lineage -- manifest's own declared identity
        # must agree with the directory it was found in.
        if str(manifest.get("scenario_id")) != scenario_id or str(manifest.get("rep")) != str(int(rep)):
            failures.append({
                "artifact": "manifest.json",
                "reason": "lineage_mismatch",
                "detail": f"manifest declares scenario_id={manifest.get('scenario_id')!r} rep={manifest.get('rep')!r}, directory is {scenario_id}/{rep}",
            })

        artifacts = manifest.get("artifacts") or {}
        for name, entry in sorted(artifacts.items()):
            if not isinstance(entry, dict):
                failures.append({"artifact": name, "reason": "digest_mismatch", "detail": "manifest artifact entry is not an object"})
                continue
            recorded_digest = entry.get("sha256")
            source_path = entry.get("source_path") or ""

            current_digest = digest_of_path(os.path.join(rep_dir, name))
            if current_digest is None:
                failures.append({"artifact": name, "reason": "missing", "detail": "artifact listed in manifest is absent from the retained directory"})
                continue
            if current_digest == recorded_digest:
                # Digest equality is the authoritative byte-preservation
                # proof (REQ-F-004). A recorded source_path that no longer
                # resolves is expected provenance staleness for an already
                # digest-confirmed artifact (e.g. an ephemeral pair_work
                # path a real producer deletes right after retention) --
                # NOT a preservation defect -- so source_path is only
                # consulted below, once a digest mismatch has already been
                # found, to give that mismatch a more specific reason.
                continue

            if source_path and not os.path.exists(source_path):
                failures.append({
                    "artifact": name,
                    "reason": "source_path_missing",
                    "detail": f"recomputed digest {current_digest} does not match manifest-recorded digest {recorded_digest}, and the recorded source_path no longer exists ({source_path}) so it cannot be re-verified against source",
                })
                continue

            reason = "digest_mismatch"
            if name in JSON_ARTIFACTS:
                canon_digest = canonical_json_digest(os.path.join(rep_dir, name))
                if canon_digest is not None and canon_digest == recorded_digest:
                    reason = "re_serialized"
            failures.append({
                "artifact": name,
                "reason": reason,
                "detail": f"recomputed digest {current_digest} does not match manifest-recorded digest {recorded_digest}",
            })

        # Phase 4: delegate upstream schema validity (never duplicate it).
        lifecycle_path = os.path.join(rep_dir, "lifecycle.jsonl")
        if os.path.isfile(lifecycle_path) and not any(f["artifact"] == "lifecycle.jsonl" for f in failures):
            rc, out, err = run_delegate(verify_run_bin, lifecycle_path, i07_schema)
            if rc == 2 or (rc != 0 and not out.strip()):
                failures.append({"artifact": "lifecycle.jsonl", "reason": "upstream_lifecycle_invalid", "detail": (err or out).strip()[:400]})
            # rc == 0 (accepted) needs no further action; verify-lifecycle-run.sh
            # has no "malformed-but-informational" middle state the way
            # verify-lifecycle-evaluation.sh does.

        evaluation_path = os.path.join(rep_dir, "evaluation.jsonl")
        if os.path.isfile(evaluation_path) and not any(f["artifact"] == "evaluation.jsonl" for f in failures):
            rc, out, err = run_delegate(verify_eval_bin, evaluation_path, i08_schema)
            if rc == 2 or (rc != 0 and not out.strip()):
                # rc==2 is a real usage/malformed failure. rc==1 with empty
                # stdout means the errors-path fired before any verdict was
                # printed (malformed record) -- also a real failure. rc==1
                # WITH a verdict on stdout is a well-formed-but-ineligible
                # record (verify-lifecycle-evaluation.sh's own contract:
                # `raise SystemExit(0 if aggregate_eligible else 1)`), which
                # is a normal retained outcome, not a retention defect.
                failures.append({"artifact": "evaluation.jsonl", "reason": "upstream_evaluation_invalid", "detail": (err or out).strip()[:400]})

    verdict = "fail" if failures else "pass"
    if failures:
        overall_fail = True
        for item in failures:
            print(f"verify-retention-root: {scenario_id}/{rep}: {item['artifact']}: {item['reason']}: {item['detail']}", file=sys.stderr)
    print(json.dumps({"scenario_id": scenario_id, "rep": rep, "verdict": verdict, "failures": failures}, sort_keys=True, separators=(",", ":")))

sys.exit(1 if overall_fail else 0)
PYEOF
	phase34_rc=$?
	set -e
	if [[ "$phase34_rc" -ne 0 ]]; then
		overall_status=1
	fi
fi

exit "$overall_status"
