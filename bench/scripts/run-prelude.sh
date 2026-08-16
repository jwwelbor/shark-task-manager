#!/usr/bin/env bash
# run-prelude.sh [--live-egress-file <path>]
# run-prelude.sh --verify-argv <captured_argv_json> [--live-egress-file <path>]
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
USAGE
	exit 2
}

live_egress_file="$DEFAULT_LIVE_EGRESS_FILE"
verify_argv_path=""

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
	*)
		usage
		;;
	esac
done

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

python3 - "$live_egress_file_abs" "$verify_argv_abs" <<'PYEOF'
import json
import shutil
import subprocess
import sys

import yaml

live_egress_path, verify_argv_arg = sys.argv[1:3]
verify_argv_path = None if verify_argv_arg == "__none__" else verify_argv_arg

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


class ScriptError(RuntimeError):
    """A prerequisite this script depends on could not be resolved at all --
    malformed live-egress-tools.yaml, malformed --verify-argv input, or a
    missing provider binary -- distinct from a normal argv-completeness or
    dispatch-exit-code verdict. Reported on stderr and mapped to exit 2."""


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
    else:
        dispatch_mode(members)


try:
    main()
except ScriptError as exc:
    sys.stderr.write(f"run-prelude: {exc}\n")
    sys.exit(2)
PYEOF
