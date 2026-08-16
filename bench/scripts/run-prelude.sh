#!/usr/bin/env bash
# run-prelude.sh [--live-egress-file <path>]
# run-prelude.sh --verify-argv <captured_argv_json> [--live-egress-file <path>]
# run-prelude.sh --package <package.yaml> --result-out <path> [--live-egress-file <path>]
#
# T-E40-F07-004 slice: the PATH-resolved dispatch SCAFFOLD and its REQ-F-004
# structural denial-argument construction and fail-before-dispatch
# self-check, plus a --verify-argv test mode that exercises the same
# completeness check against an externally supplied, already-constructed
# argument vector without ever touching the dispatcher (AC-T1's third case:
# "a stubbed argv missing a member fails before dispatch").
#
# This is a deliberately PARTIAL scaffold. Out of scope here, per this
# task's Brownfield Context, and added by later tasks that MODIFY this same
# file, keeping this task's argv/denial portion untouched:
#   - REQ-F-014's replay_reference consistency check, REQ-F-012's bulk-
#     disclosure check, and REQ-F-013's non-applicable short-circuit
#     (T-E40-F07-010).
#   - REQ-F-015's scratch-project working-directory pin, the REAL
#     bench/replay/preamble.md read from disk (this task uses an in-memory
#     placeholder -- bench/replay/preamble.md does not exist yet, see
#     PLACEHOLDER_PREAMBLE below), preamble_digest recording, and
#     REQ-F-016's Output Standards filename check (T-E40-F07-011).
#   - Writing the I-06 replay result document at all (T-E40-F07-009/011).
#
# T-E40-F07-010 update: lands the first two of the bullets above --
# REQ-F-014's read-only replay_reference consistency assertion (checked
# before ANY dispatch, package or --package-less) and REQ-F-013's
# non-applicable short-circuit (an all-non-applicable I-04 package never
# reaches build_disallowed_args/dispatch_mode at all -- it returns after
# writing an explicit `not_applicable` result). Both are gated behind the
# new, OPTIONAL `--package`/`--result-out` flags: every existing
# --live-egress-file / --verify-argv invocation (T-E40-F07-004's own tc053)
# is byte-for-byte unaffected because this task's own Brownfield Context
# forbids touching the argv/denial portion below. REQ-F-012's bulk-
# disclosure check and REQ-F-015/016's real placement/preamble remain
# T-E40-F07-011's, unchanged here.
#
# Dispatch shape (spec.md "Component changes" row for run-prelude.sh, fixed
# there so no task author invents it): a subprocess invocation of a named
# provider CLI binary resolved from PATH, in non-interactive print mode
# (`-p <prompt>`), with one `--disallowedTools <tool>` argument per
# live-egress-set member, and the prompt formed as the preamble followed by
# the Rider product-design action invocation. Resolving the binary from
# PATH (never a hardcoded absolute path) is what makes a PATH-stubbed
# dispatcher able to record the constructed argv (AC-T1) or record zero
# invocations on a failing/non-dispatch path, mirroring
# internal/runner/claude_dispatcher.go's own exec.LookPath("claude")
# resolution and its `-p`/`--disallowedTools` flag shape -- read there for
# POSTURE only; nothing under internal/ is imported (REQ-NF-001).
#
# Exit status: 0 = dispatch (or --verify-argv check) succeeded, printing
# "ARGV_COMPLETE" for --verify-argv mode. 1 = a normal, informative
# argv-completeness violation ("argv_incomplete", naming the missing
# member(s)) or a non-zero exit from the dispatched provider. 2 = a
# script/usage/authoring error (bad args, missing files, malformed
# live-egress-tools.yaml or --verify-argv input, provider binary not on
# PATH).
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_LIVE_EGRESS_FILE="$BENCH_DIR/replay/live-egress-tools.yaml"

usage() {
	cat >&2 <<'USAGE'
usage:
  run-prelude.sh [--live-egress-file <path>]
  run-prelude.sh --verify-argv <captured_argv_json> [--live-egress-file <path>]
  run-prelude.sh --package <package.yaml> --result-out <path> [--live-egress-file <path>]
USAGE
	exit 2
}

live_egress_file="$DEFAULT_LIVE_EGRESS_FILE"
verify_argv_path=""
package_path=""
result_out_path=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--live-egress-file)
		[[ $# -ge 2 ]] || usage
		live_egress_file="$2"
		shift 2
		;;
	--verify-argv)
		[[ $# -ge 2 ]] || usage
		verify_argv_path="$2"
		shift 2
		;;
	--package)
		[[ $# -ge 2 ]] || usage
		package_path="$2"
		shift 2
		;;
	--result-out)
		[[ $# -ge 2 ]] || usage
		result_out_path="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

# --package and --verify-argv are two independent modes (T-E40-F07-010 vs
# T-E40-F07-004) that never combine -- REQ-F-014's consistency check reads a
# real I-04 package, --verify-argv checks an externally supplied argv, and
# no test case or caller needs both at once.
[[ -z "$package_path" || -z "$verify_argv_path" ]] || usage
# --result-out is required alongside --package: REQ-F-013's non-applicable
# short-circuit must know where to write the explicit not_applicable
# record, and T-E40-F07-011 is expected to reuse the same flag for the
# scored-dispatch result it adds.
if [[ -n "$package_path" ]]; then
	[[ -n "$result_out_path" ]] || usage
fi

[[ -f "$live_egress_file" ]] || {
	echo "run-prelude: live-egress-tools.yaml not found: $live_egress_file" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "run-prelude: python3 not found on PATH" >&2
	exit 2
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "run-prelude: python3 module 'yaml' (PyYAML) not available" >&2
	exit 2
}

live_egress_file_abs="$(cd "$(dirname "$live_egress_file")" && pwd)/$(basename "$live_egress_file")"

verify_argv_abs="__none__"
if [[ -n "$verify_argv_path" ]]; then
	[[ -f "$verify_argv_path" ]] || {
		echo "run-prelude: --verify-argv file not found: $verify_argv_path" >&2
		exit 2
	}
	verify_argv_abs="$(cd "$(dirname "$verify_argv_path")" && pwd)/$(basename "$verify_argv_path")"
fi

# REQ-F-014/REQ-F-013 (T-E40-F07-010): the I-04 package to check/short-circuit
# and where to write REQ-F-013's explicit not_applicable record. package_path
# resolves the package.yaml itself so its containing directory can serve as
# the base replay_reference resolves against (mirrors TC-030's own
# filepath.Join(packageDir, rawPath) rule). result_out_path is deliberately
# NOT resolved/validated here: REQ-F-014's own "MUST be checked before any
# prelude dispatch" wording ranks the consistency assertion ahead of any
# output-path hygiene, so a bad --result-out must never mask a genuine
# consistency violation by failing first. It is validated (and resolved)
# only inside the Python short-circuit branch, after check_consistency has
# already passed -- see write_not_applicable_result.
package_abs="__none__"
result_out_arg="__none__"
if [[ -n "$package_path" ]]; then
	[[ -f "$package_path" ]] || {
		echo "run-prelude: --package file not found: $package_path" >&2
		exit 2
	}
	package_abs="$(cd "$(dirname "$package_path")" && pwd)/$(basename "$package_path")"
	result_out_arg="$result_out_path"
fi

# The single-owner I-06 schema file (REQ-F-018): read at call time for
# schema_version rather than duplicating it as a private constant here.
# Always the real, committed file -- unlike --live-egress-file, no test case
# needs a scratch override of this.
i06_schema_file="$BENCH_DIR/replay/i06-schema.yaml"

python3 - "$live_egress_file_abs" "$verify_argv_abs" "$package_abs" "$result_out_arg" "$i06_schema_file" <<'PYEOF'
import json
import os
import shutil
import subprocess
import sys

import yaml

live_egress_path, verify_argv_arg, package_arg, result_out_arg, i06_schema_path = sys.argv[1:6]
verify_argv_path = None if verify_argv_arg == "__none__" else verify_argv_arg
package_path = None if package_arg == "__none__" else package_arg
result_out_path = None if result_out_arg == "__none__" else result_out_arg

# T-E40-F07-011 replaces this in-memory placeholder with bench/replay/preamble.md's
# real, digest-pinned content, read from disk at dispatch time (never an
# in-memory constant, per that task's own AC-T3) -- this task's Scope
# deliberately does not create bench/replay/preamble.md (reserved for
# T-E40-F07-011; T-E40-F07-001 explicitly excludes it too).
PLACEHOLDER_PREAMBLE = (
    "[T-E40-F07-004 placeholder preamble -- bench/replay/preamble.md does not "
    "exist yet. T-E40-F07-011 replaces this in-memory placeholder with the "
    "real, digest-pinned preamble content read from disk at dispatch time.]"
)

# research-report.md / spec.md: F07 wraps the existing
# `/shark-rider project product-design` action through X-10, unmodified.
RIDER_ACTION_INVOCATION = "/shark-rider project product-design"

PROVIDER_BIN_NAME = "claude"

# REQ-F-013/REQ-F-014 (T-E40-F07-010): I-04's fixed stage_matrix.prelude key
# set, in the fixed D01-D05 order both this script's not_applicable result
# and TC-030's own package validation rely on.
PRELUDE_STAGES = ["D01", "D02", "D03", "D04", "D05"]


class ScriptError(RuntimeError):
    """A prerequisite this script depends on could not be resolved at all --
    malformed live-egress-tools.yaml, malformed --verify-argv input, or a
    missing provider binary -- distinct from a normal argv-completeness or
    dispatch-exit-code verdict. Reported on stderr and mapped to exit 2."""


class Violation(RuntimeError):
    """A real, informative REQ-F-014 rejection verdict -- mapped to exit 1,
    never conflated with a script/usage error, mirroring
    replay-answer.sh's/verify-replay-result.sh's own Violation/ScriptError
    split. kind is one of package_replay_reference_missing,
    package_replay_reference_unexpected, or package_scenario_id_mismatch."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


def load_live_egress_members(path):
    """REQ-F-004/REQ-F-018: the live-egress set is read from its single
    owner file at call time, never hardcoded here -- so removing or adding a
    tools[] entry changes both this script's denial construction and
    verify-replay-isolation.sh's transcript scan with no script edit."""
    with open(path) as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        raise ScriptError(f"live-egress-tools.yaml is not a YAML mapping: {path}")
    tools = data.get("tools")
    if not isinstance(tools, list) or not tools:
        raise ScriptError(f"live-egress-tools.yaml declares no tools[] entries: {path}")
    members = []
    seen = set()
    for idx, entry in enumerate(tools):
        if not isinstance(entry, dict) or not entry.get("name") or not entry.get("reason"):
            raise ScriptError(
                f"live-egress-tools.yaml tools[{idx}] is missing a required name or reason: {entry!r}"
            )
        name = entry["name"]
        if not isinstance(name, str):
            raise ScriptError(f"live-egress-tools.yaml tools[{idx}].name is not a string: {name!r}")
        if name in seen:
            raise ScriptError(f"live-egress-tools.yaml declares duplicate tool name: {name!r}")
        seen.add(name)
        members.append(name)
    return members


def build_disallowed_args(members):
    """REQ-F-004: one --disallowedTools <tool> argument per live-egress-set
    member, in the set's declared order -- mirrors
    internal/runner/claude_dispatcher.go's buildClaudeArgs flag shape
    (posture only; not imported, REQ-NF-001)."""
    args = []
    for name in members:
        args += ["--disallowedTools", name]
    return args


def missing_members(argv, members):
    """REQ-F-004's single-owner completeness arithmetic: which live-egress-
    set members have no matching --disallowedTools value anywhere in argv.
    Called identically by --verify-argv mode (an externally supplied,
    already-constructed argv) and by the real dispatch path immediately
    before invoking the provider binary, so the gate is exercised by both a
    direct fixture and the real construction path -- never dead code only
    the happy path visits."""
    present = set()
    for i, tok in enumerate(argv):
        if tok == "--disallowedTools" and i + 1 < len(argv):
            present.add(argv[i + 1])
    return [name for name in members if name not in present]


def resolve_within(root, rel_path, label):
    """Resolves a package-relative path (here: I-04's replay_reference) and
    enforces containment, mirroring replay-answer.sh's own resolve_within /
    TC-030's e40I04CheckPathField (filepath.Join(packageDir, rawPath) +
    containment) -- the same rule applied to the same field, read here
    read-only rather than validated for admission."""
    if os.path.isabs(rel_path):
        raise ScriptError(f"{label} must not be absolute: {rel_path!r}")
    root_real = os.path.realpath(root)
    candidate = os.path.realpath(os.path.join(root, rel_path))
    if candidate != root_real and not candidate.startswith(root_real + os.sep):
        raise ScriptError(f"{label} escapes package directory: {rel_path!r} -> {candidate!r}")
    return candidate


def load_i06_schema_version(schema_path):
    """REQ-F-018 single-owner discipline: schema_version is read from
    i06-schema.yaml at call time rather than duplicated as a private
    constant here."""
    with open(schema_path) as f:
        data = yaml.safe_load(f)
    version = data.get("schema_version") if isinstance(data, dict) else None
    if not version:
        raise ScriptError(f"i06-schema.yaml has no schema_version: {schema_path}")
    return version


def load_package(path):
    """Reads the I-04 package this run operates on and extracts exactly the
    slice spec.md's own "Consumes: I-04" row assigns to F07: entity_family,
    stage_matrix.prelude, and replay_reference (plus scenario_id/version,
    needed to name and cross-check both). Never re-validates the rest of
    I-04's shape -- TC-030 remains the single shared proof of that contract
    (test-plan.md "Consumes: I-04")."""
    package_dir = os.path.dirname(path)
    try:
        with open(path) as f:
            raw = f.read()
    except OSError as exc:
        raise ScriptError(f"could not read package: {path}: {exc}") from exc
    try:
        data = yaml.safe_load(raw)
    except yaml.YAMLError as exc:
        raise ScriptError(f"package is not valid YAML: {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise ScriptError(f"package is not a YAML mapping: {path}")

    scenario_id = data.get("scenario_id")
    if not scenario_id:
        raise ScriptError(f"package missing scenario_id: {path}")

    stage_matrix = data.get("stage_matrix")
    prelude = stage_matrix.get("prelude") if isinstance(stage_matrix, dict) else None
    if not isinstance(prelude, dict):
        raise ScriptError(f"package missing stage_matrix.prelude: {path}")

    stages = {}
    for stage in PRELUDE_STAGES:
        entry = prelude.get(stage)
        if not isinstance(entry, dict) or "applicable" not in entry:
            raise ScriptError(f"package stage_matrix.prelude.{stage} missing or malformed: {path}")
        applicable = bool(entry.get("applicable"))
        reason = entry.get("reason")
        if not applicable and not reason:
            raise ScriptError(f"package stage_matrix.prelude.{stage} is applicable: false but has no reason: {path}")
        stages[stage] = {"applicable": applicable, "reason": reason}

    return {
        "path": path,
        "dir": package_dir,
        "scenario_id": scenario_id,
        "scenario_version": data.get("scenario_version"),
        "entity_family": data.get("entity_family"),
        "replay_reference": data.get("replay_reference"),
        "prelude": stages,
    }


def check_consistency(package):
    """REQ-F-014: a READ-ONLY consistency assertion over I-04, checked
    before any prelude dispatch (and before REQ-F-013's own short-circuit
    decision, below, so a mismatched/missing/unexpected replay_reference is
    rejected even on an all-non-applicable package). No I-04 file is ever
    written here (REQ-NF-006) -- this only reads the package and, when at
    least one stage is applicable, the bundle its own replay_reference
    names."""
    pkg_id = package["scenario_id"]
    pkg_path = package["path"]
    replay_reference = package.get("replay_reference")
    any_applicable = any(stage["applicable"] for stage in package["prelude"].values())

    if any_applicable:
        if not replay_reference or not str(replay_reference).strip():
            raise Violation(
                "package_replay_reference_missing",
                f"package {pkg_id} ({pkg_path}) has at least one applicable prelude.D0X stage but "
                f"carries no replay_reference field",
            )
        bundle_path = resolve_within(package["dir"], replay_reference, "package replay_reference")
        if not os.path.isfile(bundle_path):
            raise Violation(
                "package_replay_reference_missing",
                f"package {pkg_id} ({pkg_path}) replay_reference {replay_reference!r} does not "
                f"resolve to an existing file: {bundle_path}",
            )
        try:
            with open(bundle_path) as f:
                bundle = json.load(f)
        except (OSError, json.JSONDecodeError) as exc:
            raise ScriptError(
                f"package {pkg_id} replay_reference bundle is not readable/valid JSON: {bundle_path}: {exc}"
            ) from exc
        binding = bundle.get("scenario_binding") if isinstance(bundle, dict) else None
        bundle_scenario_id = binding.get("scenario_id") if isinstance(binding, dict) else None
        if bundle_scenario_id != pkg_id:
            raise Violation(
                "package_scenario_id_mismatch",
                f"package {pkg_id} ({pkg_path}) replay_reference bundle {bundle_path} carries "
                f"scenario_binding.scenario_id={bundle_scenario_id!r}, which disagrees with the "
                f"package's own scenario_id={pkg_id!r}",
            )
    else:
        if replay_reference:
            raise Violation(
                "package_replay_reference_unexpected",
                f"package {pkg_id} ({pkg_path}) has no applicable prelude.D0X stage but carries a "
                f"replay_reference field ({replay_reference!r}); REQ-F-014 forbids replay_reference "
                f"on an all-non-applicable package",
            )


def write_not_applicable_result(package, schema_version, result_out_path):
    """REQ-F-013: writes the explicit not_applicable replay result -- the
    deliverable itself, not merely "nothing to do". Each D01-D05 record
    carries {applicable: false, reason} copied VERBATIM from the package
    (never regenerated or paraphrased -- tc057 byte-diffs it).

    result_out_path is resolved and its parent directory validated HERE,
    deliberately deferred from argument parsing -- this function is only
    ever reached after check_consistency has already passed (see main()),
    so a bad --result-out can never mask a real REQ-F-014 rejection by
    failing before the consistency assertion runs."""
    result_out_path = os.path.abspath(result_out_path)
    result_out_dir = os.path.dirname(result_out_path)
    if not os.path.isdir(result_out_dir):
        raise ScriptError(f"--result-out parent directory does not exist: {result_out_dir}")

    stages = [
        {
            "stage": stage,
            "applicable": False,
            "reason": package["prelude"][stage]["reason"],
        }
        for stage in PRELUDE_STAGES
    ]
    result = {
        "schema_version": schema_version,
        "scenario": {
            "scenario_id": package["scenario_id"],
            "scenario_version": package.get("scenario_version"),
            "entity_family": package.get("entity_family"),
        },
        # run_id is opaque to F07 (spec.md Document B); E40-F08 assigns it.
        "run_id": "",
        # No bundle/preamble/artifact_root/proxy/edge data exists for a run
        # that never dispatched -- REQ-F-014 already forbids this package
        # from carrying replay_reference at all, so these are the vacuous
        # values for a genuinely not_applicable run, not omissions.
        "replay_bundle": {},
        "preamble_digest": "",
        "artifact_root": {},
        "stages": stages,
        "replayed_interaction_proxies": {},
        "artifact_consumption_edges": [],
        "terminal_outcome": "not_applicable",
        # REQ-F-017: publication_eligible is false for every outcome other
        # than complete/not_applicable -- not_applicable is eligible.
        "publication_eligible": True,
        "ineligibility_reasons": [],
    }
    with open(result_out_path, "w") as f:
        json.dump(result, f, indent=2, sort_keys=True)
        f.write("\n")


def verify_argv_mode(members, argv_path):
    with open(argv_path) as f:
        try:
            payload = json.load(f)
        except ValueError as exc:
            raise ScriptError(f"--verify-argv file is not valid JSON: {argv_path}: {exc}") from exc
    argv = payload.get("argv") if isinstance(payload, dict) else payload
    if not isinstance(argv, list) or not all(isinstance(a, str) for a in argv):
        raise ScriptError(
            f"--verify-argv file is not a captured {{\"argv\": [...]}} object or a plain string array: {argv_path}"
        )
    missing = missing_members(argv, members)
    if missing:
        sys.stderr.write(
            "run-prelude: argv_incomplete: the supplied argument vector is missing a "
            f"--disallowedTools entry for live-egress-set member(s): {', '.join(missing)}\n"
        )
        sys.exit(1)
    print("ARGV_COMPLETE")
    sys.exit(0)


def dispatch_mode(members):
    provider_bin = shutil.which(PROVIDER_BIN_NAME)
    if provider_bin is None:
        raise ScriptError(f"provider binary {PROVIDER_BIN_NAME!r} not found on PATH")

    prompt = PLACEHOLDER_PREAMBLE + "\n\n" + RIDER_ACTION_INVOCATION
    disallowed_args = build_disallowed_args(members)
    argv = [provider_bin, "-p", prompt] + disallowed_args

    # REQ-F-004: fail closed BEFORE dispatch if the argument vector actually
    # about to be handed to subprocess.run is missing a member. Checked
    # against the FINAL argv (not an intermediate disallowed_args slice
    # computed earlier): T-E40-F07-010 and -011 both insert logic between
    # construction and dispatch (the non-applicable short-circuit, the
    # disclosure check, the real preamble read), so a gate scanning a
    # pre-insertion slice would not see a post-insertion mutation of argv
    # itself. missing_members scans whole argv elements
    # (`tok == "--disallowedTools"`), so a prompt string that happens to
    # contain that literal substring cannot cause a false pass -- and even a
    # false positive here is a loud refusal to dispatch, the safe direction.
    missing = missing_members(argv, members)
    if missing:
        sys.stderr.write(
            "run-prelude: argv_incomplete: refusing to dispatch -- constructed argument vector is "
            f"missing a --disallowedTools entry for: {', '.join(missing)}\n"
        )
        sys.exit(1)

    proc = subprocess.run(argv, capture_output=True, text=True)
    sys.stdout.write(proc.stdout)
    sys.stderr.write(proc.stderr)
    sys.exit(proc.returncode)


def main():
    members = load_live_egress_members(live_egress_path)
    if verify_argv_path is not None:
        verify_argv_mode(members, verify_argv_path)
        return

    if package_path is not None:
        package = load_package(package_path)
        # REQ-F-014: checked before ANY prelude dispatch, whether this
        # package turns out applicable or not.
        check_consistency(package)
        if not any(stage["applicable"] for stage in package["prelude"].values()):
            # REQ-F-013: the Rider action is NEVER invoked for an
            # all-non-applicable package -- return here, before
            # build_disallowed_args/dispatch_mode is ever reached, so the
            # dispatcher records zero invocations.
            schema_version = load_i06_schema_version(i06_schema_path)
            write_not_applicable_result(package, schema_version, result_out_path)
            print("NOT_APPLICABLE")
            return
        # An applicable package passed its consistency check -- fall
        # through to the unmodified dispatch path below (T-E40-F07-011
        # extends this branch with the real placement/preamble/result
        # write; out of this task's scope per its own Brownfield Context).

    dispatch_mode(members)


try:
    main()
except Violation as violation:
    sys.stderr.write(f"run-prelude: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"run-prelude: {exc}\n")
    sys.exit(2)
PYEOF
