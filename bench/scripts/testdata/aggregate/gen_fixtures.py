#!/usr/bin/env python3
"""gen_fixtures.py -- golden-derived aggregate/replay fixture generator.

T-E40-F03-002 (test-plan.md "Fixture derivation: a committed generator, not
a hand-copy", REQ-N-006, ADR-F03-08). Reads ONE of the two committed I-02
goldens --

  tests/contracts/testdata/e40_i02_golden_record.jsonl       (outcome=completed)
  tests/contracts/testdata/e40_i02_golden_record_timeout.jsonl (outcome=timeout)

-- FRESH from disk on every invocation (no cached/pre-committed snapshot of
their content anywhere in this file or elsewhere in the repo), applies a
small, explicit set of edits, and prints exactly one line of sorted-key JSON
to stdout: a fixture record for the `tc018`/`tc019` self-tests (T-E40-F03-
003/004/006/007, not yet built -- this generator is their shared primitive).

Neither golden's own `manifest.item_id` exists in `bench/corpus/corpus.yaml`
(verified directly), so every invocation MUST supply its own `--item-id`:
a synthetic id (`f03-fixture-<scenario>`) for `aggregate-runs.sh` fixtures
(which never reads corpus.yaml, REQ-F-007), or a real corpus id (e.g.
`cart-remove-item-last-match`) for `replay-manifest.sh` fixtures (REQ-F-027's
item-resolution precondition).

Rewrite/preserve contract (test-plan.md "Fixture derivation" #2/#3, verbatim):
  MAY rewrite:  manifest.item_id   (--item-id, always required)
                manifest.item_type (--item-type, only if the scenario needs
                                     the other item type)
                manifest.rep       (--rep)
                manifest.run_key   (never set directly -- ALWAYS recomputed
                                     as <item_id>::<variant_id>::rep<rep>
                                     from the two fields above and the
                                     golden's own, unrewritable variant_id)
                schema_version     (--schema-version, only for a scenario
                                     that explicitly mutates it, e.g.
                                     TC-018b/TC-018u)
                the caller-named metric field(s) under test (--set/--unset,
                     generic dotted-path primitives -- see below)
                stages[]'s LENGTH only (--duplicate-stage, T-E40-F03-004's
                     addition for TC-018k's rework-loop fixture -- appends
                     a deep copy of an EXISTING stage, never invents one;
                     see _apply_duplicate_stage's docstring)
  MUST preserve, unedited: every other key name/type, the `sources` block's
                shape and closed value set, the family-presence pattern
                implied by `outcome` (a completed-outcome fixture never
                fabricates timeout_detail; a timeout-outcome fixture never
                fabricates runresult), and everything --set/--unset were not
                pointed at.

Because this file starts every fixture from `copy.deepcopy(golden)` and
never reconstructs the record field-by-field, a future producer change that
renames, removes, or adds a field this generator has never heard of still
propagates through untouched (added fields) or breaks loudly at whatever
call site depended on the old name (renamed/removed fields) -- exactly
ADR-F03-08's guarantee: "a future producer change that breaks the goldens
breaks F03's tests too", restated at the mechanism level.

--set/--unset are generic, but deliberately narrow: they OVERWRITE or
REMOVE a field that already exists somewhere in the selected golden. They
never invent a new field name (that would silently violate "preserve
everything else, unedited"), and they refuse the five reserved paths above
(manifest.item_id/.item_type/.rep/.run_key, schema_version) -- those have
dedicated flags with derivation rules (run_key's recompute) that a generic
setter must not bypass.

Usage:
  gen_fixtures.py --golden completed --item-id f03-fixture-x
  gen_fixtures.py --golden timeout   --item-id cart-remove-item-last-match --rep 3
  gen_fixtures.py --golden completed --item-id f03-fixture-y \\
      --set loc.prod_added=10 --set loc.prod_deleted=2
  gen_fixtures.py --golden completed --item-id f03-fixture-z \\
      --unset oracle --unset loc --unset sources.oracle --unset sources.loc
  gen_fixtures.py --golden completed --item-id f03-fixture-w --schema-version 99.0
  gen_fixtures.py --golden completed --item-id f03-fixture-v --duplicate-stage 0 \\
      --set 'stages[-1].usage.input_tokens=999' --set 'stages[-1].usage.output_tokens=888' \\
      --set 'stages[-1].usage.total_cost_usd=0.5' --set 'stages[-1].duration_ns=1000000000'
  gen_fixtures.py --self-check

Golden source override (testability seam, mirrors this codebase's
RUN_ONE_BIN/SHARK_BIN pattern -- used by --self-check to prove a golden
schema change propagates; a real caller normally never needs it):
  GEN_FIXTURES_GOLDEN_COMPLETED_PATH
  GEN_FIXTURES_GOLDEN_TIMEOUT_PATH
"""
import argparse
import copy
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

SCRIPT_PATH = Path(__file__).resolve()
SCRIPT_DIR = SCRIPT_PATH.parent  # .../bench/scripts/testdata/aggregate
REPO_ROOT = SCRIPT_DIR.parent.parent.parent.parent  # aggregate -> testdata -> scripts -> bench -> repo root

DEFAULT_GOLDEN_PATHS = {
    "completed": REPO_ROOT / "tests/contracts/testdata/e40_i02_golden_record.jsonl",
    "timeout": REPO_ROOT / "tests/contracts/testdata/e40_i02_golden_record_timeout.jsonl",
}

GOLDEN_PATH_ENV_OVERRIDE = {
    "completed": "GEN_FIXTURES_GOLDEN_COMPLETED_PATH",
    "timeout": "GEN_FIXTURES_GOLDEN_TIMEOUT_PATH",
}

# The five fields with their own dedicated flag and derivation rule -- a
# generic --set/--unset must never bypass these (see module docstring).
RESERVED_PATHS = {
    "manifest.item_id": "--item-id",
    "manifest.item_type": "--item-type",
    "manifest.rep": "--rep",
    "manifest.run_key": "--rep/--item-id (run_key is always recomputed, never independently settable)",
    "schema_version": "--schema-version",
}


def golden_path(kind):
    override = os.environ.get(GOLDEN_PATH_ENV_OVERRIDE[kind])
    return Path(override) if override else DEFAULT_GOLDEN_PATHS[kind]


def load_golden(kind):
    """Reads the selected golden fresh from disk -- no caching, no module-
    level snapshot -- so repeated calls within one process (the self-check
    calls this several times) always see the file's current content."""
    path = golden_path(kind)
    with open(path) as f:
        lines = [ln for ln in (raw.strip() for raw in f) if ln]
    if len(lines) != 1:
        sys.exit(
            "gen_fixtures: golden %s must contain exactly one JSON record, found %d"
            % (path, len(lines))
        )
    return json.loads(lines[0])


def _parse_set_value(raw_value):
    """--set's RHS is JSON-parsed when possible (so numbers/bools/arrays/
    objects work unquoted), falling back to the literal string otherwise
    (so `--set quality.toolchain_guard=go_version_mismatch` needs no
    quoting)."""
    try:
        return json.loads(raw_value)
    except json.JSONDecodeError:
        return raw_value


def _check_not_reserved(path_str, flag):
    dedicated = RESERVED_PATHS.get(path_str)
    if dedicated is not None:
        sys.exit(
            "gen_fixtures: %s %s: use %s instead -- this field has a derivation "
            "rule %s must not bypass" % (flag, path_str, dedicated, flag)
        )


# A path segment is either a plain object key ("usage") or an object key
# followed by a list index ("stages[1]", "stages[-1]") -- the ONLY list-
# indexing form this generator supports, added for TC-018k (T-E40-F03-004),
# which needs to target the second of two `stages[]` entries produced by
# --duplicate-stage. Every other path segment in this file remains
# dict-only, unchanged from T-E40-F03-002.
_SEGMENT_RE = re.compile(r"^([A-Za-z_][A-Za-z0-9_]*)(\[(-?\d+)\])?$")


def _split_segment(part, path_str):
    m = _SEGMENT_RE.match(part)
    if not m:
        sys.exit("gen_fixtures: invalid path segment '%s' in '%s'" % (part, path_str))
    name, _bracket, idx = m.groups()
    return name, (int(idx) if idx is not None else None)


def _step_into(node, part, path_str):
    """Resolves one non-final path segment, indexing into a list when the
    segment carries a bracketed index."""
    name, idx = _split_segment(part, path_str)
    if not isinstance(node, dict) or name not in node:
        sys.exit("gen_fixtures: path '%s' does not exist in the base record" % path_str)
    node = node[name]
    if idx is None:
        return node
    if not isinstance(node, list):
        sys.exit("gen_fixtures: path '%s': '%s' is not a list" % (path_str, name))
    try:
        return node[idx]
    except IndexError:
        sys.exit("gen_fixtures: path '%s': index %d out of range for '%s'" % (path_str, idx, name))


def _walk_to_container(record, path_str):
    """Returns (container, leaf_key) for the final path segment -- leaf_key
    is a dict string key, or a list integer index when the final segment
    carries a bracketed index (e.g. "stages[-1]")."""
    parts = path_str.split(".")
    node = record
    for part in parts[:-1]:
        node = _step_into(node, part, path_str)

    name, idx = _split_segment(parts[-1], path_str)
    if not isinstance(node, dict) or name not in node:
        sys.exit("gen_fixtures: path '%s' does not exist in the base record" % path_str)
    if idx is None:
        if not isinstance(node, dict):
            sys.exit("gen_fixtures: path '%s' does not resolve to an object" % path_str)
        return node, name
    target = node[name]
    if not isinstance(target, list):
        sys.exit("gen_fixtures: path '%s': '%s' is not a list" % (path_str, name))
    if not (-len(target) <= idx < len(target)):
        sys.exit("gen_fixtures: path '%s': index %d out of range for '%s'" % (path_str, idx, name))
    return target, idx


def _apply_set(record, path_str, raw_value):
    _check_not_reserved(path_str, "--set")
    container, leaf = _walk_to_container(record, path_str)
    has_leaf = leaf in container if isinstance(container, dict) else True  # list index already range-checked
    if not has_leaf:
        sys.exit(
            "gen_fixtures: --set %s: field does not exist in the base record -- "
            "--set only overwrites an EXISTING field, it never invents a new one "
            "(test-plan.md's preserve-structure rule)" % path_str
        )
    container[leaf] = _parse_set_value(raw_value)


def _apply_unset(record, path_str):
    _check_not_reserved(path_str, "--unset")
    container, leaf = _walk_to_container(record, path_str)
    if isinstance(container, dict):
        if leaf not in container:
            sys.exit(
                "gen_fixtures: --unset %s: field does not exist in the base record "
                "(nothing to remove)" % path_str
            )
    del container[leaf]


def _apply_duplicate_stage(record, source_index):
    """Appends a deep copy of stages[source_index] to record['stages']
    (TC-018k, T-E40-F03-004: a rework loop re-entering the same status).
    `index` on the copy is recomputed to stay monotonically after every
    existing stage; every other field is copied verbatim -- a subsequent
    --set 'stages[-1].usage.<field>=<value>' is what makes the two
    occurrences distinct, keeping this generator's derivation guarantee
    (ADR-F03-08): the duplicate is whatever the golden actually shipped,
    never a hand-authored stage."""
    stages = record.get("stages")
    if not isinstance(stages, list):
        sys.exit(
            "gen_fixtures: --duplicate-stage %d: record has no stages[] list "
            "(a timeout-outcome fixture never carries one)" % source_index
        )
    try:
        source = stages[source_index]
    except IndexError:
        sys.exit(
            "gen_fixtures: --duplicate-stage %d: index out of range (stages has %d entries)"
            % (source_index, len(stages))
        )
    new_stage = copy.deepcopy(source)
    existing_indices = [s.get("index") for s in stages if isinstance(s.get("index"), int)]
    if existing_indices:
        new_stage["index"] = max(existing_indices) + 1
    stages.append(new_stage)


def generate(golden_kind, item_id, item_type=None, rep=None, schema_version=None, sets=(), unsets=(), duplicate_stages=()):
    """Builds one derived fixture record from the selected golden.

    `sets` is an iterable of (dotted_path, raw_string_value) pairs;
    `unsets` is an iterable of dotted paths to remove. `duplicate_stages`
    is an iterable of 0-based source indices into `stages[]`, applied (in
    order) before `sets`/`unsets` so a --set can target the newly
    appended copy via `stages[-1]...`.
    """
    golden = load_golden(golden_kind)
    record = copy.deepcopy(golden)

    # manifest.item_id -- always rewritten. Neither golden's own item_id
    # exists in corpus.yaml, so a caller MUST supply a real one (checked by
    # main(); generate() itself just requires a non-empty string here so the
    # self-check can call it directly without going through argparse).
    record["manifest"]["item_id"] = item_id

    if item_type is not None:
        record["manifest"]["item_type"] = item_type

    if rep is not None:
        record["manifest"]["rep"] = rep

    # manifest.run_key: ALWAYS recomputed (test-plan.md "Fixture derivation"
    # #2), from the item_id/rep just resolved above and the golden's own
    # variant_id, which no flag ever rewrites.
    variant_id = record["manifest"]["variant_id"]
    record["manifest"]["run_key"] = "%s::%s::rep%s" % (
        record["manifest"]["item_id"],
        variant_id,
        record["manifest"]["rep"],
    )

    if schema_version is not None:
        record["schema_version"] = schema_version

    for source_index in duplicate_stages:
        _apply_duplicate_stage(record, source_index)

    for path_str, raw_value in sets:
        _apply_set(record, path_str, raw_value)
    for path_str in unsets:
        _apply_unset(record, path_str)

    return record


# ---------------------------------------------------------------------------
# --self-check: the generator's own verification mode (no tc018/tc019 harness
# needed -- those don't exist yet, T-E40-F03-003/004/006/007's scope). Proves
# (a) a plain derive preserves everything but the identity fields, structurally,
# against whichever golden it came from; (b) --set/--unset touch only their
# named target; (c) the five reserved paths reject --set/--unset; (d) a
# schema change in the golden's own content (a field this script has never
# heard of) propagates into the derived fixture -- the direct evidence that
# nothing here is a cached/hand-copied snapshot.
# ---------------------------------------------------------------------------


def _diff_untouched(golden, derived, ignore):
    """Recursively compares `golden` and `derived` (both starting as dicts),
    requiring every key/value/type to match except at dotted paths listed in
    `ignore`. Lists are compared by value (not descended into) -- sufficient
    here since no path this generator supports ever indexes into a list.
    Returns (ok, first_mismatch_description)."""

    def walk(g, d, prefix):
        if isinstance(g, dict):
            if not isinstance(d, dict):
                return "%s: golden is an object, derived is not" % prefix
            g_keys, d_keys = set(g.keys()), set(d.keys())
            for k in sorted(g_keys - d_keys):
                path = "%s.%s" % (prefix, k) if prefix else k
                if path not in ignore:
                    return "%s: present in golden, absent in derived" % path
            for k in sorted(d_keys - g_keys):
                path = "%s.%s" % (prefix, k) if prefix else k
                if path not in ignore:
                    return "%s: present in derived, absent from golden (fabricated field)" % path
            for k in sorted(g_keys & d_keys):
                path = "%s.%s" % (prefix, k) if prefix else k
                if path in ignore:
                    continue
                err = walk(g[k], d[k], path)
                if err:
                    return err
            return None
        if type(g) is not type(d) and not (isinstance(g, (int, float)) and isinstance(d, (int, float))):
            return "%s: type changed (%s -> %s)" % (prefix, type(g).__name__, type(d).__name__)
        if g != d:
            return "%s: value changed (%r -> %r)" % (prefix, g, d)
        return None

    err = walk(golden, derived, "")
    return (err is None, err or "")


def _run_self_check():
    failures = []

    def check(name, condition, detail=""):
        status = "PASS" if condition else "FAIL"
        line = "gen_fixtures self-check: %s: %s" % (name, status)
        if not condition and detail:
            line += " -- %s" % detail
        print(line, file=sys.stderr)
        if not condition:
            failures.append(name)

    # (a) Plain derive, per golden: identity fields rewritten correctly,
    # schema_version preserved, family-presence pattern intact, and every
    # untouched field matches the golden verbatim.
    for kind in ("completed", "timeout"):
        golden = load_golden(kind)
        derived = generate(kind, item_id="self-check-item", rep=7)

        check(
            "%s: manifest.run_key recomputed correctly" % kind,
            derived["manifest"]["run_key"] == "self-check-item::%s::rep7" % golden["manifest"]["variant_id"],
        )
        check("%s: manifest.item_id rewritten" % kind, derived["manifest"]["item_id"] == "self-check-item")
        check("%s: manifest.rep rewritten" % kind, derived["manifest"]["rep"] == 7)
        check(
            "%s: schema_version preserved (no --schema-version given)" % kind,
            derived["schema_version"] == golden["schema_version"],
        )
        ok, mismatch = _diff_untouched(
            golden, derived, ignore={"manifest.item_id", "manifest.rep", "manifest.run_key"}
        )
        check("%s: every other field preserved verbatim" % kind, ok, mismatch)

    completed_derived = generate("completed", item_id="x")
    timeout_derived = generate("timeout", item_id="y")
    check(
        "completed-derived carries runresult, never timeout_detail",
        "runresult" in completed_derived and "timeout_detail" not in completed_derived,
    )
    check(
        "timeout-derived carries timeout_detail, never runresult",
        "timeout_detail" in timeout_derived and "runresult" not in timeout_derived,
    )

    # (b) --set overwrites exactly its target; --unset removes exactly its
    # target; neither touches anything else.
    derived_set = generate("completed", item_id="x", sets=[("loc.prod_added", "999")])
    check("--set overwrites the targeted field", derived_set["loc"]["prod_added"] == 999)
    ok, mismatch = _diff_untouched(
        load_golden("completed"),
        derived_set,
        ignore={"manifest.item_id", "manifest.rep", "manifest.run_key", "loc.prod_added"},
    )
    check("--set touches no other field", ok, mismatch)

    derived_unset = generate("completed", item_id="x", unsets=["oracle"])
    check("--unset removes the targeted block", "oracle" not in derived_unset)
    ok, mismatch = _diff_untouched(
        load_golden("completed"),
        derived_unset,
        ignore={"manifest.item_id", "manifest.rep", "manifest.run_key", "oracle"},
    )
    check("--unset touches no other field", ok, mismatch)

    # (b2) --duplicate-stage appends a deep copy of an EXISTING stage
    # (never invents one), recomputes only its `index`, and a subsequent
    # list-indexed --set targets exclusively the new copy -- T-E40-F03-004's
    # addition, exercised by TC-018k (rework-loop fixture).
    golden_completed = load_golden("completed")
    derived_dup = generate("completed", item_id="x", duplicate_stages=[0])
    check(
        "--duplicate-stage appends one entry",
        len(derived_dup["stages"]) == len(golden_completed["stages"]) + 1,
    )
    check(
        "--duplicate-stage's copy matches its source except for a recomputed index",
        derived_dup["stages"][-1]["status"] == golden_completed["stages"][0]["status"]
        and derived_dup["stages"][-1]["usage"] == golden_completed["stages"][0]["usage"]
        and derived_dup["stages"][-1]["index"] == max(s["index"] for s in golden_completed["stages"]) + 1,
    )
    check(
        "--duplicate-stage leaves the original stages[] entries untouched",
        derived_dup["stages"][: len(golden_completed["stages"])] == golden_completed["stages"],
    )

    derived_dup_set = generate(
        "completed",
        item_id="x",
        duplicate_stages=[0],
        sets=[("stages[-1].usage.input_tokens", "999")],
    )
    check(
        "list-indexed --set targets only the duplicated stage",
        derived_dup_set["stages"][-1]["usage"]["input_tokens"] == 999
        and derived_dup_set["stages"][0]["usage"]["input_tokens"] == golden_completed["stages"][0]["usage"]["input_tokens"],
    )

    dup_out_of_range_rejected = True
    try:
        generate("completed", item_id="x", duplicate_stages=[999])
        dup_out_of_range_rejected = False
    except SystemExit:
        pass
    check("--duplicate-stage with an out-of-range index is rejected", dup_out_of_range_rejected)

    dup_on_timeout_rejected = True
    try:
        generate("timeout", item_id="x", duplicate_stages=[0])
        dup_on_timeout_rejected = False
    except SystemExit:
        pass
    check("--duplicate-stage on a stages-less (timeout) golden is rejected", dup_on_timeout_rejected)

    # (c) The five reserved paths reject --set/--unset (must use the
    # dedicated flag instead).
    for reserved_path in RESERVED_PATHS:
        rejected = True
        try:
            generate("completed", item_id="x", sets=[(reserved_path, "bypass")])
            rejected = False
        except SystemExit:
            pass
        check("--set %s is rejected (use %s)" % (reserved_path, RESERVED_PATHS[reserved_path]), rejected)

    unset_rejected = True
    try:
        generate("completed", item_id="x", unsets=["manifest.item_id"])
        unset_rejected = False
    except SystemExit:
        pass
    check("--unset manifest.item_id is rejected (use --item-id)", unset_rejected)

    # (d) A golden schema change propagates: copy the completed golden to a
    # temp file, add a nested field AND a top-level field this script has
    # never heard of, point the generator at the temp copy via the
    # documented env-var override, and confirm the derived fixture carries
    # both new fields verbatim -- direct evidence of "reads fresh", not "a
    # hand-copied/hardcoded snapshot".
    probe_source = load_golden("completed")
    probe_source["quality"]["self_check_probe_field"] = "self-check-probe-value"
    probe_source["self_check_probe_top_level"] = {"nested": True}
    tmp_fd, tmp_path = tempfile.mkstemp(suffix=".jsonl")
    try:
        with os.fdopen(tmp_fd, "w") as tmp:
            tmp.write(json.dumps(probe_source, sort_keys=True) + "\n")

        env = dict(os.environ)
        env["GEN_FIXTURES_GOLDEN_COMPLETED_PATH"] = tmp_path
        proc = subprocess.run(
            [sys.executable, str(SCRIPT_PATH), "--golden", "completed", "--item-id", "self-check-probe"],
            capture_output=True,
            text=True,
            env=env,
            check=False,
        )
        propagate_ok = False
        detail = proc.stderr.strip()
        if proc.returncode == 0:
            try:
                out_record = json.loads(proc.stdout)
                propagate_ok = (
                    out_record.get("quality", {}).get("self_check_probe_field") == "self-check-probe-value"
                    and out_record.get("self_check_probe_top_level") == {"nested": True}
                )
                if not propagate_ok:
                    detail = "derived record did not carry the probe fields: %r" % out_record
            except json.JSONDecodeError:
                detail = "stdout was not valid JSON: %r" % proc.stdout
        check(
            "a golden schema change (new field, via GEN_FIXTURES_GOLDEN_COMPLETED_PATH) "
            "propagates to the derived fixture unmodified",
            propagate_ok,
            detail,
        )
    finally:
        os.unlink(tmp_path)

    if failures:
        print(
            "gen_fixtures self-check: FAIL (%d check(s) failed: %s)" % (len(failures), ", ".join(failures)),
            file=sys.stderr,
        )
        return 1
    print("gen_fixtures self-check: PASS (all checks green)", file=sys.stderr)
    return 0


def build_arg_parser():
    parser = argparse.ArgumentParser(
        description="Golden-derived aggregate/replay fixture generator (T-E40-F03-002).",
    )
    parser.add_argument(
        "--self-check", action="store_true", help="run the generator's own verification mode and exit"
    )
    parser.add_argument("--golden", choices=sorted(DEFAULT_GOLDEN_PATHS), help="which committed golden to derive from")
    parser.add_argument("--item-id", help="synthetic id for tc018, real corpus id for tc019 (required)")
    parser.add_argument("--item-type", help="overrides manifest.item_type (default: the golden's own value)")
    parser.add_argument("--rep", type=int, help="overrides manifest.rep (default: the golden's own value)")
    parser.add_argument("--schema-version", help="overrides schema_version (default: the golden's own value)")
    parser.add_argument(
        "--set",
        dest="sets",
        action="append",
        default=[],
        metavar="PATH=VALUE",
        help="overwrite an EXISTING dotted-path field; VALUE is JSON-parsed when possible, else a literal string",
    )
    parser.add_argument(
        "--unset",
        dest="unsets",
        action="append",
        default=[],
        metavar="PATH",
        help="remove an EXISTING dotted-path field",
    )
    parser.add_argument(
        "--duplicate-stage",
        dest="duplicate_stages",
        action="append",
        default=[],
        type=int,
        metavar="SOURCE_INDEX",
        help="append a deep copy of stages[SOURCE_INDEX] (0-based); target it afterwards via --set 'stages[-1].usage.<field>=<value>'",
    )
    return parser


def main(argv):
    args = build_arg_parser().parse_args(argv)

    if args.self_check:
        return _run_self_check()

    if not args.golden:
        sys.exit("gen_fixtures: --golden {completed,timeout} is required")
    if not args.item_id:
        sys.exit(
            "gen_fixtures: --item-id is required -- neither golden's own item_id exists in "
            "bench/corpus/corpus.yaml (test-plan.md 'Fixture derivation' #5)"
        )

    sets = []
    for raw in args.sets:
        if "=" not in raw:
            sys.exit("gen_fixtures: --set %r must be PATH=VALUE" % raw)
        path_str, _, value = raw.partition("=")
        sets.append((path_str, value))

    record = generate(
        args.golden,
        item_id=args.item_id,
        item_type=args.item_type,
        rep=args.rep,
        schema_version=args.schema_version,
        sets=sets,
        unsets=args.unsets,
        duplicate_stages=args.duplicate_stages,
    )
    print(json.dumps(record, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
