#!/usr/bin/env bash
# replay-stage-evidence.sh <bundle_dir> [--checkout <fixture_checkout>] [--adapter <adapter_path>]
#
# The REQ-F-013 replay guard (spec.md "Component changes" row for
# replay-stage-evidence.sh; T-E40-F06-010). Re-evaluates a STORED bundle
# against its named roots with no worker rerun and no provider call: every
# field this script needs is resolvable from the bundle plus the caller-
# supplied roots (REQ-F-013's own wording), never by re-dispatching the
# worker "to be sure" (test-plan.md tc049 Caller-Path Contract's Negative
# case). This script never execs a provider binary itself -- tc049 proves
# that empirically with a PATH-stubbed provider that records zero
# invocations across every replay, rather than assuming it from this
# script's own source.
#
# A wholly separate script from verify-stage-evidence.sh (T-E40-F06-004
# through -008): that script validates a bundle's FIELD SHAPE against
# i05-schema.yaml; this one validates that a bundle's RECORDED CONTENT still
# matches (a) its own declared digest (REQ-F-015 immutability) and (b) the
# live state of the roots and I-04 adapter it was captured against
# (REQ-F-013 drift). Per this task's Notes for Agent, it does not implement
# any of verify-stage-evidence.sh's live-bundle portions, and it is not
# modified by, nor does it call into, that script.
#
# For every stage snapshot indexed in <bundle_dir>/bundle.json's `stages[]`:
#
#   1. REQ-F-015 immutability: recompute `snapshot_digest` over the
#      snapshot's own canonical serialization -- `json.dumps(sort_keys=True)`
#      over every field EXCLUDING `snapshot_digest` itself (the exact
#      "computed over the canonical serialization excluding this field"
#      contract spec.md's Stage-snapshot table states, and the exact bug
#      this task's Notes for Agent calls out: recomputing WITH the digest
#      field folded in would make an unmodified snapshot spuriously
#      mismatch its own recorded value). A mismatch means the stored
#      snapshot file was edited after it was recorded.
#
#      A mismatch is reported with the most specific diagnosis this script
#      can make from the CURRENT content alone (it holds no separate
#      "before" copy to diff against): if any `artifacts[]` entry in the
#      mismatched snapshot is missing its `consumers` key, that is reported
#      as `artifact_consumption_record_missing` naming the artifact --
#      because an absent `consumers` key is independently a LEGITIMATE,
#      digest-matching state per REQ-F-008/ADR-F06-07 ("consumption
#      evidence was not collected"), so a mismatched digest plus a missing
#      key together is exactly what "the key was deleted after recording"
#      looks like. Every other kind of edit falls back to the generic
#      `snapshot_mutated` naming the stage (REQ-F-015, AC-013 case (ii)).
#
#   2. Only when the digest matches (an untampered snapshot) and its
#      `stage_category` is `code` or `review`: two independent REQ-F-013
#      drift checks against `--checkout`, sourced from the snapshot's own
#      `candidate` block, never re-derived or assumed:
#
#        a. File drift, from `candidate.dirty_untracked_manifest` (REQ-F-006
#           already defines this as an ordered `{path, digest, tracked}`
#           list -- the recorded per-file inventory this script treats as
#           ground truth for replay, run BEFORE any adapter invocation so a
#           `python3 -m pytest` cache directory the test-suite check below
#           creates can never masquerade as a drifted file):
#             - a manifest entry whose CURRENT digest under `--checkout`
#               differs from its recorded digest is `tracked_file_changed`
#               (`tracked: true`) or `untracked_file_changed`
#               (`tracked: false`), naming the path.
#             - a file that exists under `--checkout` NOW but is absent from
#               the manifest entirely (outside `candidate.test_suite_dir`,
#               the test-suite check's own domain below) is a newly ADDED
#               untracked file -- `untracked_file_changed`, naming the path.
#        b. Test-suite drift, from `candidate.test_suite_ids` (this script's
#           own recorded-id-set field, alongside the opaque
#           `test_suite_digest` REQ-F-006 already requires -- REQ-F-013 asks
#           for the DIFFERING TEST ID BY NAME, which an opaque digest alone
#           cannot name): invokes `<adapter> test --checkout <checkout>`
#           (I-04's real, already-covered `test` capability, TC-030 --
#           per this task's Brownfield Context, never re-implemented here)
#           and diffs its live, normalized `entries[].id` set against the
#           recorded set. Any id present in exactly one of the two sets is
#           `test_suite_changed`, naming that id.
#
# Requires: python3 on PATH; `--adapter` requires the named adapter
# executable (I-04's contract, REQ-F-007) to be runnable.
#
# Exit status: 0 = replay clean (JSON summary on stdout). 1 = one or more
# named drift/mutation verdicts (test-plan.md tc049's Caller-Path Contract:
# nothing meaningful on stdout, one named line per verdict on stderr). 2 = a
# script/usage/authoring error (bad args, missing bundle files, a
# code/review stage with no `--checkout`, an adapter invocation that itself
# could not run).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	echo "usage: replay-stage-evidence.sh <bundle_dir> [--checkout <fixture_checkout>] [--adapter <adapter_path>]" >&2
	exit 2
}

[[ $# -ge 1 ]] || usage
BUNDLE_DIR="$1"
shift

CHECKOUT=""
ADAPTER=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--checkout)
		[[ $# -ge 2 ]] || usage
		CHECKOUT="$2"
		shift 2
		;;
	--adapter)
		[[ $# -ge 2 ]] || usage
		ADAPTER="$2"
		shift 2
		;;
	*)
		echo "replay-stage-evidence: unknown argument: $1" >&2
		usage
		;;
	esac
done

[[ -d "$BUNDLE_DIR" ]] || {
	echo "replay-stage-evidence: bundle dir not found: $BUNDLE_DIR" >&2
	exit 2
}
[[ -f "$BUNDLE_DIR/bundle.json" ]] || {
	echo "replay-stage-evidence: bundle.json not found under $BUNDLE_DIR" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "replay-stage-evidence: python3 not found on PATH" >&2
	exit 2
}

bundle_dir_abs="$(cd "$BUNDLE_DIR" && pwd)"
checkout_abs=""
if [[ -n "$CHECKOUT" ]]; then
	[[ -d "$CHECKOUT" ]] || {
		echo "replay-stage-evidence: --checkout dir not found: $CHECKOUT" >&2
		exit 2
	}
	checkout_abs="$(cd "$CHECKOUT" && pwd)"
fi
adapter_abs=""
if [[ -n "$ADAPTER" ]]; then
	[[ -x "$ADAPTER" ]] || {
		echo "replay-stage-evidence: --adapter not found or not executable: $ADAPTER" >&2
		exit 2
	}
	adapter_abs="$(cd "$(dirname "$ADAPTER")" && pwd)/$(basename "$ADAPTER")"
fi

python3 - "$bundle_dir_abs" "$checkout_abs" "$adapter_abs" <<'PYEOF'
import hashlib
import json
import os
import subprocess
import sys

bundle_dir, checkout, adapter = sys.argv[1:4]


class ScriptError(RuntimeError):
    """A usage/authoring problem this script cannot evaluate -- distinct
    from a named drift/mutation verdict. Caught once at the bottom and
    reported with exit 2, mirroring verify-evidence-roots.sh's own
    ScriptError-vs-verdict split."""


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    with open(path, "rb") as f:
        return sha256_bytes(f.read())


def load_json(path):
    with open(path) as f:
        return json.load(f)


def recompute_snapshot_digest(snapshot):
    # REQ-F-015: canonical serialization EXCLUDING snapshot_digest itself
    # -- folding the digest field into its own input is the exact bug
    # this task's Notes for Agent names, which would make even an
    # untouched snapshot spuriously fail to reproduce its recorded value.
    payload = {k: v for k, v in snapshot.items() if k != "snapshot_digest"}
    return "sha256:" + sha256_bytes(json.dumps(payload, sort_keys=True).encode("utf-8"))


def check_immutability(stage_key, stage_entry, snapshot, verdicts):
    """REQ-F-015. Returns True if the snapshot's digest verifies (safe to
    run the REQ-F-013 drift checks below against its content), False if a
    mutation verdict was recorded (drift checks are skipped -- an already
    self-mismatching snapshot is not a trustworthy source of "what was
    recorded")."""
    recomputed = recompute_snapshot_digest(snapshot)
    recorded_in_snapshot = snapshot.get("snapshot_digest")
    recorded_in_index = stage_entry.get("snapshot_digest")
    if recomputed == recorded_in_snapshot == recorded_in_index:
        return True

    # Most specific diagnosis available from the CURRENT content alone
    # (ADR-F06-07: an absent `consumers` key is independently legitimate,
    # so only a DIGEST MISMATCH plus a missing key together identifies a
    # post-recording deletion).
    named = False
    for artifact in snapshot.get("artifacts") or []:
        if "consumers" not in artifact:
            named = True
            verdicts.append(
                (
                    "artifact_consumption_record_missing",
                    f"stage={stage_key} artifact={artifact.get('path', '<unknown>')}",
                )
            )
    if not named:
        verdicts.append(("snapshot_mutated", f"stage={stage_key}"))
    return False


def check_file_drift(stage_key, candidate, checkout, verdicts):
    manifest = candidate.get("dirty_untracked_manifest") or []
    recorded_paths = set()
    for entry in manifest:
        rel_path = entry["path"]
        recorded_paths.add(rel_path)
        kind = "tracked_file_changed" if entry.get("tracked") else "untracked_file_changed"
        abs_path = os.path.join(checkout, rel_path)
        if not os.path.isfile(abs_path):
            verdicts.append((kind, f"stage={stage_key} path={rel_path}"))
            continue
        current_digest = "sha256:" + sha256_file(abs_path)
        if current_digest != entry["digest"]:
            verdicts.append((kind, f"stage={stage_key} path={rel_path}"))

    # Newly ADDED untracked files: present under --checkout now, absent from
    # the recorded manifest entirely. Excludes candidate.test_suite_dir --
    # that subtree is the test-suite-drift check's own domain (below), run
    # AFTER this scan specifically so its own cache side effects never leak
    # into this one.
    test_suite_dir = candidate.get("test_suite_dir")
    for root, _dirs, files in os.walk(checkout):
        rel_root = os.path.relpath(root, checkout)
        for fname in files:
            rel_path = fname if rel_root == "." else os.path.join(rel_root, fname)
            if test_suite_dir and (
                rel_path == test_suite_dir or rel_path.startswith(test_suite_dir + os.sep)
            ):
                continue
            if rel_path in recorded_paths:
                continue
            verdicts.append(("untracked_file_changed", f"stage={stage_key} path={rel_path}"))


def check_test_suite_drift(stage_key, candidate, checkout, adapter, verdicts):
    recorded_ids = candidate.get("test_suite_ids")
    if recorded_ids is None:
        return
    if not adapter:
        raise ScriptError(
            f"stage={stage_key}: candidate.test_suite_ids present but --adapter not supplied"
        )
    result = subprocess.run(
        [adapter, "test", "--checkout", checkout],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise ScriptError(
            f"stage={stage_key}: adapter test invocation failed (exit {result.returncode}): "
            f"{result.stderr.strip()}"
        )
    try:
        live = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise ScriptError(f"stage={stage_key}: adapter test produced non-JSON output: {exc}") from exc

    live_ids = {entry["id"] for entry in live.get("entries", [])}
    recorded_set = set(recorded_ids)
    for test_id in sorted(live_ids ^ recorded_set):
        verdicts.append(("test_suite_changed", f"stage={stage_key} test_id={test_id}"))


def main():
    bundle = load_json(os.path.join(bundle_dir, "bundle.json"))
    stages = bundle.get("stages") or []
    if not stages:
        raise ScriptError("bundle.json declares no stages[]")

    verdicts = []
    stages_checked = 0

    for stage_entry in stages:
        stage_key = stage_entry.get("stage_key", "<unknown>")
        snapshot_path = stage_entry.get("snapshot_path")
        if not snapshot_path:
            raise ScriptError(f"stage={stage_key}: stage index entry missing snapshot_path")
        snapshot_full_path = os.path.join(bundle_dir, snapshot_path)
        if not os.path.isfile(snapshot_full_path):
            raise ScriptError(f"stage={stage_key}: snapshot file not found: {snapshot_full_path}")
        snapshot = load_json(snapshot_full_path)

        stages_checked += 1
        digest_ok = check_immutability(stage_key, stage_entry, snapshot, verdicts)
        if not digest_ok:
            continue

        stage_category = snapshot.get("stage_category")
        candidate = snapshot.get("candidate")
        if stage_category in ("code", "review") and candidate:
            if not checkout:
                raise ScriptError(
                    f"stage={stage_key}: stage_category={stage_category} requires --checkout"
                )
            check_file_drift(stage_key, candidate, checkout, verdicts)
            check_test_suite_drift(stage_key, candidate, checkout, adapter, verdicts)

    if verdicts:
        for kind, detail in verdicts:
            sys.stderr.write(f"{kind} {detail}\n")
        sys.exit(1)

    print(json.dumps({"result": "clean", "stages_checked": stages_checked}))


try:
    main()
except ScriptError as exc:
    sys.stderr.write(f"replay-stage-evidence: {exc}\n")
    sys.exit(2)
PYEOF
