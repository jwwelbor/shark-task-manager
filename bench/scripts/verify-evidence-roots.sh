#!/usr/bin/env bash
# verify-evidence-roots.sh <package_yaml> <fixture_checkout> <scratch_project> <evaluator_root>
#
# The REQ-F-010 (admission-time) / REQ-F-011 (dispatch-boundary) isolation
# guard (ADR-F06-03). Both requirements share one underlying property --
# "no evaluator-only material is reachable from an agent-visible root" -- so
# one script proves it, invoked either once per candidate package (admission
# time, against a fresh checkout) or immediately before every worker
# dispatch (against the live, in-flight roots). The caller decides *when*
# to invoke this script; the script itself does not know or care which of
# the two callers it is serving.
#
# Derives, from <package_yaml>'s own evaluator_only block (reference_solution,
# oracle_tests[], answer_keys[]) resolved against <evaluator_root>, three
# independent signals per declared entry (REQ-F-011: "any evaluator-only
# file, its content digest, or a test identity it defines"):
#   (1) filename  -- a file bearing the same basename exists anywhere in an
#                     agent-visible root.
#   (2) content_digest -- any file in an agent-visible root is byte-
#                     identical to the declared entry (catches a rename that
#                     preserves content).
#   (3) test_identity  -- oracle_tests[] entries only (the files that
#                     actually define tests; reference_solution/answer_keys
#                     are not test-defining files and are covered by (1)/(2)
#                     alone). The entry's own basename stem (the `<module>`
#                     half of I-04's `<module>::<test>` normalized id, e.g.
#                     "test_recurring" for "test_recurring.py") appears as a
#                     whole token (no adjoining identifier character on
#                     either side -- a real match, e.g. an `import
#                     test_recurring`, not merely a shared prefix with an
#                     unrelated identifier such as the base fixture's own
#                     skip-marked `test_recurring_task_...` placeholder)
#                     inside any file's content in an agent-visible root
#                     (catches a copy-paste that keeps the defining name but
#                     drops the original file name).
#
# Names are ALWAYS derived from <package_yaml> at call time (REQ-F-010) --
# nothing here is hardcoded per scenario, so renaming an oracle_tests[]
# entry in a scratch copy of a package changes the search target with no
# edit to this script (AC-010).
#
# Walks BOTH agent-visible roots -- <fixture_checkout> AND <scratch_project>
# -- independently (ADR-F06-03: "a guard that walks only --workdir misses
# everything Shark writes into the scratch project"). <evaluator_root> is
# never walked for leaks; it is only the root the declared evaluator_only
# paths are read FROM.
#
# Root names in messages are read from bench/evidence/i05-schema.yaml's
# `roots:` map (REQ-F-017: one machine-readable vocabulary owner) rather
# than embedded here as a private copy.
#
# A REQ-NF-007 "bounded filesystem walk, no network call": this script
# never checks out a fixture, never invokes an adapter, and never spawns any
# process other than python3 for YAML/digest work -- both <fixture_checkout>
# and <scratch_project> are caller-supplied, already-materialized
# directories. It never invokes a dispatcher of any kind, which is exactly
# the property AC-009 case (d) observes via a PATH-stubbed dispatcher's
# empty invocation log.
#
# Exit status: 0 = both roots clean ("CLEAN" on stdout). 1 = an isolation
# violation was found -- a normal, informative verdict; the message on
# stderr names the offending root, the offending path, the evaluator-only
# source it matched, and the match kind. 2 = a script/usage/authoring error
# (bad args, missing files, malformed package.yaml, a declared evaluator_only
# path that escapes <evaluator_root> or does not exist on disk).
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
I05_SCHEMA="$BENCH_DIR/evidence/i05-schema.yaml"

usage() {
	echo "usage: verify-evidence-roots.sh <package_yaml> <fixture_checkout> <scratch_project> <evaluator_root>" >&2
	exit 2
}

[[ $# -eq 4 ]] || usage
package_yaml="$1"
fixture_checkout="$2"
scratch_project="$3"
evaluator_root="$4"

[[ -f "$package_yaml" ]] || {
	echo "verify-evidence-roots: package.yaml not found: $package_yaml" >&2
	exit 2
}
[[ -d "$fixture_checkout" ]] || {
	echo "verify-evidence-roots: fixture checkout dir not found: $fixture_checkout" >&2
	exit 2
}
[[ -d "$scratch_project" ]] || {
	echo "verify-evidence-roots: scratch project dir not found: $scratch_project" >&2
	exit 2
}
[[ -d "$evaluator_root" ]] || {
	echo "verify-evidence-roots: evaluator root dir not found: $evaluator_root" >&2
	exit 2
}
[[ -f "$I05_SCHEMA" ]] || {
	echo "verify-evidence-roots: i05-schema.yaml not found: $I05_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-evidence-roots: python3 not found on PATH" >&2
	exit 2
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "verify-evidence-roots: python3 module 'yaml' (PyYAML) not available" >&2
	exit 2
}

package_yaml_abs="$(cd "$(dirname "$package_yaml")" && pwd)/$(basename "$package_yaml")"
fixture_checkout_abs="$(cd "$fixture_checkout" && pwd)"
scratch_project_abs="$(cd "$scratch_project" && pwd)"
evaluator_root_abs="$(cd "$evaluator_root" && pwd)"

python3 - "$package_yaml_abs" "$fixture_checkout_abs" "$scratch_project_abs" "$evaluator_root_abs" "$I05_SCHEMA" <<'PYEOF'
import hashlib
import os
import re
import sys

import yaml

package_yaml_path, fixture_checkout, scratch_project, evaluator_root, i05_schema_path = sys.argv[1:6]


class ScriptError(RuntimeError):
    """A prerequisite this guard depends on could not be resolved at all --
    distinct from a normal isolation-violation verdict. Reported on stderr
    and mapped to exit 2."""


def load_yaml(path, label):
    with open(path) as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        raise ScriptError(f"{label} is not a YAML mapping: {path}")
    return data


def resolve_in_evaluator_root(evaluator_root, rel_path, label):
    """Resolves a package-declared relative path (untrusted input -- it
    comes from the candidate's own package.yaml) against evaluator_root and
    enforces containment before anything is read from it, mirroring
    admit-scenario.sh's resolve_scoped (REQ-NF-005)."""
    if os.path.isabs(rel_path):
        raise ScriptError(f"{label}: absolute path not allowed: {rel_path!r}")
    root_real = os.path.realpath(evaluator_root)
    candidate = os.path.realpath(os.path.join(evaluator_root, rel_path))
    if candidate != root_real and not candidate.startswith(root_real + os.sep):
        raise ScriptError(f"{label}: resolved path escapes evaluator_root: {rel_path!r} -> {candidate!r}")
    if not os.path.isfile(candidate):
        raise ScriptError(
            f"{label}: declared evaluator_only path does not exist as a file under evaluator_root: {rel_path!r} -> {candidate!r}"
        )
    return candidate


def build_targets(package_yaml_path, evaluator_root):
    """Builds the ordered, package-derived list of search targets
    (REQ-F-010: "names MUST be derived from the package at call time, never
    from a hardcoded list")."""
    package = load_yaml(package_yaml_path, "package.yaml")
    evaluator_only = package.get("evaluator_only")
    if not isinstance(evaluator_only, dict):
        raise ScriptError(f"package.yaml has no evaluator_only block: {package_yaml_path}")

    declared_entries = []  # [(field, index_or_None, declared_rel_path)]

    reference_solution = evaluator_only.get("reference_solution")
    if reference_solution:
        declared_entries.append(("reference_solution", None, reference_solution))

    for idx, entry in enumerate(evaluator_only.get("oracle_tests") or []):
        declared_entries.append(("oracle_tests", idx, entry))

    for idx, entry in enumerate(evaluator_only.get("answer_keys") or []):
        declared_entries.append(("answer_keys", idx, entry))

    if not declared_entries:
        raise ScriptError(
            f"package.yaml evaluator_only block declares no reference_solution, oracle_tests[], or answer_keys[] entries: {package_yaml_path}"
        )

    targets = []
    for field, idx, declared_rel_path in declared_entries:
        label = f"evaluator_only.{field}" + (f"[{idx}]" if idx is not None else "")
        resolved = resolve_in_evaluator_root(evaluator_root, declared_rel_path, label)
        basename = os.path.basename(resolved)
        stem = os.path.splitext(basename)[0]
        with open(resolved, "rb") as f:
            content = f.read()
        digest = hashlib.sha256(content).hexdigest()
        # The test_identity signal applies only to oracle_tests[] entries --
        # REQ-F-010's "a test identity those files define" grammatically
        # scopes to the files that define tests. reference_solution (a
        # patch) and answer_keys (data) are not test-defining files, and
        # their stems can coincide with ordinary English words ("reference",
        # "answer") that would otherwise produce false isolation violations
        # on legitimate, unrelated content. Word-boundary pattern: matches
        # the stem only as a whole identifier (no adjoining [A-Za-z0-9_.] on
        # either side), so a shared *prefix* with an unrelated identifier
        # (e.g. the base fixture's own test_recurring_task_... placeholder
        # sharing "test_recurring" with an oracle file named
        # test_recurring.py) is never a false positive. "." is included in
        # the disqualifying class alongside identifier characters: without
        # it, an ordinary base-fixture comment or docstring mentioning the
        # oracle file's own name in prose (e.g. "see
        # test_due_date_boundary.py") would match, because "." is not a
        # word character and would otherwise satisfy the lookahead --
        # verified against the real py-bug-due-date-boundary package this
        # would have been a live false positive for, not a hypothetical one.
        stem_pattern = None
        if field == "oracle_tests":
            stem_pattern = re.compile(
                rb"(?<![A-Za-z0-9_.])" + re.escape(stem.encode()) + rb"(?![A-Za-z0-9_.])"
            )
        targets.append(
            {
                "source": label,
                "declared_path": declared_rel_path,
                "basename": basename,
                "stem": stem,
                "stem_pattern": stem_pattern,
                "digest": digest,
            }
        )
    return targets


def walk_files(root):
    """Deterministic, sorted, .git-excluding file walk (REQ-NF-004: byte-
    identical verdicts across repeated runs over an unchanged root)."""
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = sorted(d for d in dirnames if d != ".git")
        for name in sorted(filenames):
            yield os.path.join(dirpath, name)


def check_root(root_name, root_path, targets):
    """Returns the first violation found walking root_path, checking
    targets in their package-declared order and match kinds in a fixed
    priority (filename, content_digest, test_identity), or None if clean."""
    for path in walk_files(root_path):
        try:
            with open(path, "rb") as f:
                file_bytes = f.read()
        except OSError:
            continue
        file_basename = os.path.basename(path)
        file_digest = hashlib.sha256(file_bytes).hexdigest()
        for target in targets:
            if file_basename == target["basename"]:
                return (root_name, path, target, "filename")
            if file_digest == target["digest"]:
                return (root_name, path, target, "content_digest")
            if target["stem_pattern"] is not None and target["stem_pattern"].search(file_bytes):
                return (root_name, path, target, "test_identity")
    return None


def main():
    # REQ-F-017: root names come from the single vocabulary owner, not a
    # private copy embedded in this script.
    schema = load_yaml(i05_schema_path, "i05-schema.yaml")
    schema_roots = schema.get("roots") or {}
    for required in ("agent_fixture_checkout", "scratch_shark_project"):
        if required not in schema_roots:
            raise ScriptError(f"i05-schema.yaml roots: is missing required key {required!r}")

    root_fixture_checkout = "agent_fixture_checkout"
    root_scratch_project = "scratch_shark_project"

    targets = build_targets(package_yaml_path, evaluator_root)

    violation = check_root(root_fixture_checkout, fixture_checkout, targets) or check_root(
        root_scratch_project, scratch_project, targets
    )

    if violation is not None:
        root_name, matched_path, target, match_kind = violation
        sys.stderr.write(
            "verify-evidence-roots: isolation_violation: "
            f"root={root_name} path={matched_path} "
            f"source={target['source']}={target['declared_path']!r} "
            f"match_kind={match_kind}\n"
        )
        sys.exit(1)

    print("CLEAN")


try:
    main()
except ScriptError as exc:
    sys.stderr.write(f"verify-evidence-roots: {exc}\n")
    sys.exit(2)
PYEOF
