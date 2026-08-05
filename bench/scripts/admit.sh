#!/usr/bin/env bash
# admit.sh <corpus_yaml_path> [--item <id>] [--patch <path>]
#
# Execution-based admission gate (T-E40-F01-006, REQ-F-005/006/007). For
# each candidate, runs the five checks pinned by
# architecture.md#corpus-and-oracle-contract against a fresh
# checkout-fixture.sh checkout:
#   (a) F2P red at base
#   (b) P2P green at base
#   (c) reference patch applies (git apply's own exit status)
#   (d) F2P green after the patch
#   (e) P2P still green after the patch
#
# A candidate failing any check is rejected and the verdict names exactly
# that check; the same evaluation code path runs for every candidate, so no
# item, negative_item, or transient candidate is ever special-cased into a
# forced verdict.
#
# No --item: evaluates every entry in corpus.yaml's `items:` (the admitted
# set) -- never `negative_items:` (AC-T3, REQ-F-007) -- printing one JSON
# verdict line per item to stdout, sorted by item id (Determinism boundary,
# test-plan.md). The script's own exit code is the gate's summary
# assertion: 0 only if every item was admitted, at least 10 were admitted,
# and at least one admitted `bug` item confirms its repro oracle; 1
# otherwise, with the specific problem(s) named on stderr.
#
# --item <id>: evaluates exactly one candidate looked up by id in `items:`
# or `negative_items:`, printing one JSON verdict line to stdout. Exit 0 if
# admitted, 1 if rejected.
#
# --patch <path>: only valid together with --item. Overrides the looked-up
# candidate's reference_patch_path with <path> while keeping its f2p/p2p_set
# from the manifest unchanged. This is how
# tests/tc005_admit_rejection_branches_test.sh drives the two transient
# candidates for rejection branches (c) and (e), which REQ-F-007 excuses
# from needing a committed corpus.yaml entry.
#
# Scope boundary (task spec): this script does not implement the base-SHA
# ledgers (build-ledgers.sh / diff-ledgers.sh, T-E40-F01-008), and it never
# invokes golangci-lint itself -- running lint checks belongs to the ledger
# scripts, not REQ-F-005.
#
# Reproducibility (T-E40-F01-007, REQ-F-012): every check above is a fresh
# subprocess against a fresh checkout-fixture.sh checkout, and verdict lines
# are always emitted sorted by item id with sort_keys=True JSON -- two
# independent runs against two independently provisioned checkouts of the
# same base SHA produce byte-identical output (test-plan.md Determinism
# boundary; tests/tc006_admit_reproducibility_test.sh).
#
# Offline hardening (T-E40-F01-007, REQ-NF-005): before evaluating any
# candidate (--item or full-set), this script verifies the pinned
# `golangci-lint` binary named by corpus.yaml's
# fixture.toolchain.golangci_lint_version is present on PATH and fails
# loudly, naming the missing tool, if it is not. This script never shells
# out to `curl` or `make`, so a missing pinned tool can never silently
# trigger the root Makefile:108 auto-installer and reintroduce a network
# dependency (tests/tc013_admit_offline_test.sh).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKOUT_SCRIPT="$SCRIPT_DIR/checkout-fixture.sh"

usage() {
	echo "usage: admit.sh <corpus_yaml_path> [--item <id>] [--patch <path>]" >&2
	exit 2
}

[[ $# -ge 1 ]] || usage
corpus_yaml="$1"
shift

item_id=""
patch_override=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--item)
		[[ $# -ge 2 ]] || usage
		item_id="$2"
		shift 2
		;;
	--patch)
		[[ $# -ge 2 ]] || usage
		patch_override="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

if [[ -n "$patch_override" && -z "$item_id" ]]; then
	echo "admit: --patch requires --item" >&2
	usage
fi

[[ -f "$corpus_yaml" ]] || {
	echo "admit: corpus yaml not found: $corpus_yaml" >&2
	exit 1
}
[[ -x "$CHECKOUT_SCRIPT" ]] || {
	echo "admit: checkout-fixture.sh missing or not executable: $CHECKOUT_SCRIPT" >&2
	exit 1
}
command -v python3 >/dev/null 2>&1 || {
	echo "admit: python3 not found on PATH" >&2
	exit 1
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "admit: python3 module 'yaml' (PyYAML) not available (required to parse corpus.yaml)" >&2
	exit 1
}
command -v go >/dev/null 2>&1 || {
	echo "admit: go toolchain not found on PATH" >&2
	exit 1
}
command -v git >/dev/null 2>&1 || {
	echo "admit: git not found on PATH" >&2
	exit 1
}

corpus_yaml_abs="$(cd "$(dirname "$corpus_yaml")" && pwd)/$(basename "$corpus_yaml")"

if [[ -n "$patch_override" ]]; then
	[[ -f "$patch_override" ]] || {
		echo "admit: --patch file not found: $patch_override" >&2
		exit 1
	}
	patch_override="$(cd "$(dirname "$patch_override")" && pwd)/$(basename "$patch_override")"
fi

exec python3 - "$corpus_yaml_abs" "$CHECKOUT_SCRIPT" "$item_id" "$patch_override" <<'PYEOF'
import json
import os
import shutil
import subprocess
import sys
import tempfile

import yaml

corpus_yaml_path, checkout_script, item_id, patch_override = sys.argv[1:5]
item_id = item_id or None
patch_override = patch_override or None

corpus_dir = os.path.dirname(corpus_yaml_path)

with open(corpus_yaml_path) as f:
    data = yaml.safe_load(f)

base_sha = data["fixture"]["base_sha"]
p2p_sets = data.get("p2p_sets") or {}

FAIL_F2P_GREEN_AT_BASE = "F2P-green-at-base"
FAIL_P2P_RED_AT_BASE = "P2P-red-at-base"
FAIL_PATCH_APPLY = "patch-apply-failure"
FAIL_F2P_STILL_RED = "F2P-still-red-post-patch"
FAIL_P2P_RED_POST_PATCH = "P2P-red-post-patch"


def find_item(candidate_id):
    for section in ("items", "negative_items"):
        for it in data.get(section) or []:
            if it["id"] == candidate_id:
                return it
    return None


def check_golangci_lint_present():
    """REQ-NF-005 / T-E40-F01-007 offline hardening: fail loudly, before any
    admission subprocess runs, if the pinned golangci-lint binary is not on
    PATH. admit.sh never runs golangci-lint itself (that belongs to the
    ledger scripts), but its presence is part of the pinned toolchain the
    admitted verdicts are claimed against, so a missing pinned tool must
    stop the gate rather than silently pass. This check never shells out to
    an installer -- it only calls shutil.which, so a missing binary can
    never trigger the root Makefile:108 curl-based auto-install."""
    pinned_version = data["fixture"]["toolchain"]["golangci_lint_version"]
    if shutil.which("golangci-lint") is None:
        print(
            f"admit: golangci-lint not found on PATH (pinned {pinned_version} "
            "required by corpus.yaml fixture.toolchain.golangci_lint_version); "
            "admission does not fall back to a network installer -- install "
            "the pinned binary and re-run",
            file=sys.stderr,
        )
        sys.exit(1)


def read_module_path(checkout_dir):
    with open(os.path.join(checkout_dir, "go.mod")) as f:
        first_line = f.readline().strip()
    if not first_line.startswith("module "):
        raise RuntimeError(f"unexpected go.mod first line: {first_line!r}")
    return first_line[len("module "):].strip()


def run_go_tests(checkout_dir, packages, run_selector):
    """Runs `go test -json` for packages in checkout_dir and returns
    (results, build_failed_pkgs) where results maps "<pkg>::<test>" to its
    terminal action (pass/fail/skip) and build_failed_pkgs is the set of
    package import paths that failed to compile."""
    cmd = ["go", "test", "-json"]
    if run_selector:
        cmd += ["-run", run_selector]
    cmd += list(packages)
    proc = subprocess.run(cmd, cwd=checkout_dir, capture_output=True, text=True)

    results = {}
    build_failed_pkgs = set()
    for line in proc.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue
        action = ev.get("Action")
        test = ev.get("Test")
        pkg = ev.get("Package")
        if test and action in ("pass", "fail", "skip"):
            results[f"{pkg}::{test}"] = action
        elif action == "fail" and not test and ev.get("FailedBuild") and pkg:
            build_failed_pkgs.add(pkg)
    return results, build_failed_pkgs


def test_status(results, build_failed_pkgs, test_id):
    pkg = test_id.split("::", 1)[0]
    if pkg in build_failed_pkgs:
        return "fail"
    return results.get(test_id, "missing")


def copy_f2p_files(item, checkout_dir):
    module_path = read_module_path(checkout_dir)
    prefix = module_path + "/"
    paths = item["f2p"]["paths"]
    test_names = item["f2p"]["test_names"]
    if len(paths) != len(test_names):
        raise RuntimeError(
            f"item {item['id']}: f2p.paths and f2p.test_names length mismatch"
        )
    for path, test_name in zip(paths, test_names):
        pkg_import = test_name.split("::", 1)[0]
        if not pkg_import.startswith(prefix):
            raise RuntimeError(
                f"item {item['id']}: f2p test {test_name!r} package "
                f"{pkg_import!r} is not under module {module_path!r}"
            )
        rel_pkg_dir = pkg_import[len(prefix):]
        src = os.path.join(corpus_dir, path)
        dst_dir = os.path.join(checkout_dir, rel_pkg_dir)
        os.makedirs(dst_dir, exist_ok=True)
        shutil.copy2(src, os.path.join(dst_dir, os.path.basename(path)))


def evaluate(item, patch_path):
    item_id_ = item["id"]
    item_type = item.get("type")
    p2p_set_name = item["p2p_set"]
    p2p_set = p2p_sets.get(p2p_set_name)
    if p2p_set is None:
        raise RuntimeError(f"item {item_id_}: unknown p2p_set {p2p_set_name!r}")
    packages = p2p_set.get("packages") or ["./..."]
    run_selector = p2p_set.get("run_selector") or ""
    exclude_tests = set(p2p_set.get("exclude_tests") or [])
    f2p_ids = list(item["f2p"]["test_names"])
    exclude_from_p2p = exclude_tests | set(f2p_ids)

    checks = {
        "f2p_red_at_base": None,
        "p2p_green_at_base": None,
        "patch_applies": None,
        "f2p_green_post_patch": None,
        "p2p_green_post_patch": None,
    }
    failing_check = None

    parent = tempfile.mkdtemp(prefix="admit-")
    try:
        checkout_dir = os.path.join(parent, "checkout")
        try:
            subprocess.run(
                [checkout_script, base_sha, checkout_dir],
                check=True,
                capture_output=True,
                text=True,
            )
        except subprocess.CalledProcessError as exc:
            raise RuntimeError(
                f"item {item_id_}: checkout-fixture.sh failed: {exc.stderr}"
            ) from exc

        copy_f2p_files(item, checkout_dir)

        base_results, base_build_failed = run_go_tests(checkout_dir, packages, run_selector)

        f2p_red_at_base = all(
            test_status(base_results, base_build_failed, t) == "fail" for t in f2p_ids
        )
        checks["f2p_red_at_base"] = f2p_red_at_base
        if not f2p_red_at_base:
            failing_check = FAIL_F2P_GREEN_AT_BASE

        if failing_check is None:
            p2p_ids = [t for t in base_results if t not in exclude_from_p2p]
            p2p_green_at_base = not base_build_failed and all(
                test_status(base_results, base_build_failed, t) != "fail" for t in p2p_ids
            )
            checks["p2p_green_at_base"] = p2p_green_at_base
            if not p2p_green_at_base:
                failing_check = FAIL_P2P_RED_AT_BASE

        if failing_check is None:
            apply_proc = subprocess.run(
                ["git", "apply", patch_path],
                cwd=checkout_dir,
                capture_output=True,
                text=True,
            )
            patch_applies = apply_proc.returncode == 0
            checks["patch_applies"] = patch_applies
            if not patch_applies:
                failing_check = FAIL_PATCH_APPLY

        if failing_check is None:
            post_results, post_build_failed = run_go_tests(checkout_dir, packages, run_selector)

            f2p_green_post_patch = all(
                test_status(post_results, post_build_failed, t) == "pass" for t in f2p_ids
            )
            checks["f2p_green_post_patch"] = f2p_green_post_patch
            if not f2p_green_post_patch:
                failing_check = FAIL_F2P_STILL_RED

            if failing_check is None:
                p2p_ids_post = [t for t in post_results if t not in exclude_from_p2p]
                p2p_green_post_patch = not post_build_failed and all(
                    test_status(post_results, post_build_failed, t) != "fail"
                    for t in p2p_ids_post
                )
                checks["p2p_green_post_patch"] = p2p_green_post_patch
                if not p2p_green_post_patch:
                    failing_check = FAIL_P2P_RED_POST_PATCH
    finally:
        shutil.rmtree(parent, ignore_errors=True)

    status = "rejected" if failing_check else "admitted"
    repro_confirmed = True if (status == "admitted" and item_type == "bug") else None

    return {
        "item_id": item_id_,
        "type": item_type,
        "status": status,
        "failing_check": failing_check,
        "checks": checks,
        "repro_confirmed": repro_confirmed,
    }


def main():
    check_golangci_lint_present()

    if item_id:
        item = find_item(item_id)
        if item is None:
            print(
                f"admit: no candidate found with id {item_id!r} in items or negative_items",
                file=sys.stderr,
            )
            sys.exit(2)
        patch_path = patch_override or os.path.join(corpus_dir, item["reference_patch_path"])
        verdict = evaluate(item, patch_path)
        print(json.dumps(verdict, sort_keys=True))
        sys.exit(0 if verdict["status"] == "admitted" else 1)

    candidates = sorted(data.get("items") or [], key=lambda it: it["id"])
    if not candidates:
        print("admit: corpus.yaml has no items", file=sys.stderr)
        sys.exit(1)

    verdicts = []
    for item in candidates:
        patch_path = os.path.join(corpus_dir, item["reference_patch_path"])
        verdict = evaluate(item, patch_path)
        print(json.dumps(verdict, sort_keys=True))
        sys.stdout.flush()
        verdicts.append(verdict)

    admitted = [v for v in verdicts if v["status"] == "admitted"]
    rejected = [v for v in verdicts if v["status"] == "rejected"]
    bug_confirmed = any(v["type"] == "bug" and v.get("repro_confirmed") for v in admitted)

    problems = []
    if rejected:
        problems.append(
            "candidates rejected from the admitted set: "
            + ", ".join(f"{v['item_id']} ({v['failing_check']})" for v in rejected)
        )
    if len(admitted) < 10:
        problems.append(f"only {len(admitted)} admitted items, need >= 10")
    if not bug_confirmed:
        problems.append("no admitted bug-type item has a confirmed repro oracle")

    if problems:
        print("admit: gate summary assertion failed: " + "; ".join(problems), file=sys.stderr)
        sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except SystemExit:
        raise
    except Exception as exc:  # top-level CLI error boundary: turn an
        # unexpected internal failure into a named stderr message and a
        # non-zero exit instead of a raw traceback.
        print(f"admit: {exc}", file=sys.stderr)
        sys.exit(2)
PYEOF
