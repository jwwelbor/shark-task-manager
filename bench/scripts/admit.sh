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
# Evidence-isolation guard (UAT round 3, T-E40-F01-006 rejection,
# superseding the round-2 fix below): `go test`'s package-level exit code
# is a single bit. Round 2 tried to explain a package-level "fail" by
# checking whether ANY per-test "fail" was also recorded for that
# package -- but the intentional TestStock_PermanentlyFailingRegression
# Probe (excluded from the default p2p_set) supplies exactly such a
# per-test fail on every run, so it can mask an unrelated, independent
# runtime failure (TestMain calling os.Exit after m.Run(), a panic
# outside a test body, an init() crash) riding on the SAME package's
# process exit: the probe's real failure and the injected one share one
# evidence channel, and no amount of "does a fail exist" counting can
# tell them apart from a single boolean exit code.
#
# The fix is structural, not a smarter classifier: evaluate() below never
# runs a `go test` invocation in which an excluded/expected test is
# present to hide behind. The P2P check's `go test` call always passes
# `-skip` for this item's exclude_tests plus its own F2P test names, so
# those tests do not execute at all in that invocation -- nothing in it
# is expected to fail, so its raw exit code alone is sufficient (no
# reconciliation needed, because there is nothing left to explain away).
# The F2P check's `go test` call scopes `-run` to only this item's F2P
# test name(s); since a rogue TestMain still executes for the whole
# package regardless of -run/-skip, an F2P "pass" claim is additionally
# required to come from an invocation whose own exit code was 0. See
# evaluate()'s docstring for the full reasoning.
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
import re
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


def bare_test_name(identity_or_name):
    """Strips a "<pkg>::" prefix (if present) and any "/<subtest>" suffix,
    returning the bare top-level Go test function name."""
    name = identity_or_name.split("::", 1)[-1]
    return name.split("/", 1)[0]


def anchored_alternation(names):
    """Builds a `go test -run`/`-skip` regexp that matches exactly the
    given bare top-level test names (each anchored on its own so
    "TestFoo" cannot substring-match "TestFooBar"), or None if names is
    empty. Go's -run/-skip apply this regexp against each "/"-separated
    path element of a (sub)test's full name; anchoring the top-level name
    alone is sufficient to select or exclude that whole test (and every
    subtest under it)."""
    bare = sorted({bare_test_name(n) for n in names})
    if not bare:
        return None
    return "^(" + "|".join(re.escape(n) for n in bare) + ")$"


def run_go_tests(checkout_dir, packages, run_pattern=None, skip_pattern=None):
    """Runs `go test -json` for packages in checkout_dir, optionally
    scoped by a -run and/or -skip regexp (see anchored_alternation), and
    returns (results, problem_pkgs, returncode).

    results maps "<pkg>::<test>" to its terminal action (pass/fail/skip)
    for every per-test event this specific invocation observed.
    problem_pkgs is the set of package import paths this invocation
    reported as failed at the package level -- compile failures
    (FailedBuild) and package-level runtime/process failures alike --
    named for caller diagnostics only. returncode is the raw `go test`
    process exit code: the single source of truth this script's callers
    use to decide pass/fail (see evaluate())."""
    cmd = ["go", "test", "-json"]
    if run_pattern:
        cmd += ["-run", run_pattern]
    if skip_pattern:
        cmd += ["-skip", skip_pattern]
    cmd += list(packages)
    proc = subprocess.run(cmd, cwd=checkout_dir, capture_output=True, text=True)

    results = {}
    problem_pkgs = set()
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
            continue
        if not test and pkg and action == "fail":
            problem_pkgs.add(pkg)

    return results, problem_pkgs, proc.returncode


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
    """UAT round 3 (UAT-005): the round-2 "explained by any per-test fail"
    reconciliation is itself the defect class -- an expected/excluded
    failure (the intentional TestStock_PermanentlyFailingRegressionProbe)
    shares its package's evidence channel with an unexpected one, so a
    single genuinely-failing test can mask an independent runtime failure
    (a panic outside a test body, TestMain calling os.Exit, an init()
    crash) riding on the same package's process exit. Counting failures
    cannot close this: `go test`'s package-level exit code is a single
    bit, and no amount of "does at least one recorded fail exist" logic
    can tell "exactly the known failures happened" apart from "the known
    failures happened AND something else broke too".

    The structural fix is to never let an admission run's evidence
    channel contain an expected failure in the first place:

    - The P2P-isolated run below always adds `-skip` for exactly this
      item's exclude_tests union its own F2P test names (both are
      supposed to be absent from the P2P universe already, per
      REQ-F-003). With every test this item is allowed to see fail
      actually skipped -- not merely subtracted from a results dict
      after the fact -- NOTHING in that specific invocation is expected
      to fail. Its raw process exit code is therefore sufficient by
      itself: p2p_green == (that invocation's returncode == 0), full
      stop, no per-test reconciliation needed, because there is no
      excluded test left in the run to hide behind.
    - The F2P-isolated run below scopes `-run` to exactly this item's own
      F2P test name(s) and nothing else. A rogue TestMain still executes
      unconditionally for the package and can still force a non-zero
      process exit regardless of the isolated test's own outcome, so
      f2p_green additionally requires that invocation's returncode == 0
      -- a "pass" claim is trusted only when nothing else in the process
      misbehaved either. A "fail" claim never needs this extra scrutiny:
      it is real evidence either way.

    This closes the masking channel by construction (the excluded/F2P
    test physically does not execute in the run being judged), not by a
    smarter classifier over the same shared-channel evidence."""
    item_id_ = item["id"]
    item_type = item.get("type")
    p2p_set_name = item["p2p_set"]
    p2p_set = p2p_sets.get(p2p_set_name)
    if p2p_set is None:
        raise RuntimeError(f"item {item_id_}: unknown p2p_set {p2p_set_name!r}")
    packages = p2p_set.get("packages") or ["./..."]
    run_selector = p2p_set.get("run_selector") or None
    exclude_tests = set(p2p_set.get("exclude_tests") or [])
    f2p_ids = list(item["f2p"]["test_names"])
    if not f2p_ids:
        raise RuntimeError(f"item {item_id_}: f2p.test_names is empty")
    exclude_from_p2p = exclude_tests | set(f2p_ids)

    p2p_skip_pattern = anchored_alternation(exclude_from_p2p)
    f2p_run_pattern = anchored_alternation(f2p_ids)
    # The F2P run is scoped to exactly the F2P test(s)' own package(s) --
    # NOT p2p_set.packages ("./..." for every defined set). `go test
    # ./...` runs every matched package as its own subprocess and the
    # overall command exits non-zero if ANY of them does, regardless of
    # -run selection within each -- so reusing the P2P-wide package glob
    # here would make the F2P check spuriously sensitive to a completely
    # unrelated package's runtime failure (e.g. the very
    # pkg/inventory poisoning this fix exists to catch, when the item's
    # own F2P test lives elsewhere), reporting F2P-still-red-post-patch
    # for what is actually a P2P problem. Scoping to the F2P test's own
    # package(s) keeps the two checks properly independent.
    f2p_packages = sorted({t.split("::", 1)[0] for t in f2p_ids})

    checks = {
        "f2p_red_at_base": None,
        "p2p_green_at_base": None,
        "patch_applies": None,
        "f2p_green_post_patch": None,
        "p2p_green_post_patch": None,
    }
    failing_check = None
    unexplained_failed_packages = []

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

        # F2P-isolated run: -run scopes execution to exactly this item's
        # own F2P test name(s), so nothing this item does not name can
        # produce a per-test result here.
        base_f2p_results, _base_f2p_problem_pkgs, _base_f2p_rc = run_go_tests(
            checkout_dir, f2p_packages, run_pattern=f2p_run_pattern
        )
        f2p_red_at_base = all(base_f2p_results.get(t) == "fail" for t in f2p_ids)
        checks["f2p_red_at_base"] = f2p_red_at_base
        if not f2p_red_at_base:
            failing_check = FAIL_F2P_GREEN_AT_BASE

        if failing_check is None:
            # P2P-isolated run: -skip removes exactly the tests this item
            # is allowed to see fail (exclude_tests + its own F2P names).
            # Nothing left running is expected to fail, so the raw exit
            # code alone is authoritative -- see evaluate()'s docstring.
            _base_p2p_results, base_p2p_problem_pkgs, base_p2p_rc = run_go_tests(
                checkout_dir, packages, run_pattern=run_selector, skip_pattern=p2p_skip_pattern
            )
            p2p_green_at_base = base_p2p_rc == 0
            checks["p2p_green_at_base"] = p2p_green_at_base
            if not p2p_green_at_base:
                failing_check = FAIL_P2P_RED_AT_BASE
                unexplained_failed_packages = sorted(base_p2p_problem_pkgs)

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
            post_f2p_results, _post_f2p_problem_pkgs, post_f2p_rc = run_go_tests(
                checkout_dir, f2p_packages, run_pattern=f2p_run_pattern
            )

            # A "pass" claim is trusted only when the isolated F2P run's
            # own process exit also agrees nothing else misbehaved --
            # otherwise a rogue TestMain forcing a non-zero exit
            # regardless of this test's real outcome would read as a
            # false "green" (see evaluate()'s docstring).
            f2p_green_post_patch = (
                post_f2p_rc == 0
                and all(post_f2p_results.get(t) == "pass" for t in f2p_ids)
            )
            checks["f2p_green_post_patch"] = f2p_green_post_patch
            if not f2p_green_post_patch:
                failing_check = FAIL_F2P_STILL_RED

            if failing_check is None:
                _post_p2p_results, post_p2p_problem_pkgs, post_p2p_rc = run_go_tests(
                    checkout_dir, packages, run_pattern=run_selector, skip_pattern=p2p_skip_pattern
                )
                p2p_green_post_patch = post_p2p_rc == 0
                checks["p2p_green_post_patch"] = p2p_green_post_patch
                if not p2p_green_post_patch:
                    failing_check = FAIL_P2P_RED_POST_PATCH
                    unexplained_failed_packages = sorted(post_p2p_problem_pkgs)
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
        "unexplained_failed_packages": unexplained_failed_packages,
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
