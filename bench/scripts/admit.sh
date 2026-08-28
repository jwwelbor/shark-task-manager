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
# Evidence-forgeability property (code review round 5, T-E40-F01-006
# rejection -- FINAL authorized rework of this defect-class lineage):
# every prior round trusted a signal self-reported by the exact process
# whose honesty was in question, and Go's `os.Exit()` lets any test or
# TestMain in the package under test forge every one of them:
#   - round 2 trusted "does at least one per-test fail exist for this
#     package" -- the intentional TestStock_PermanentlyFailingRegression
#     Probe supplies one on every run, masking an unrelated failure;
#   - round 3/round 5's first attempt trusted the raw process exit code
#     alone for the P2P check -- `TestMain{ m.Run(); os.Exit(0) }` forces
#     it to 0 regardless of what m.Run() actually returned, masking a
#     real regression (code review round 5, finding #1);
#   - that same attempt's `-skip` pattern matched test names globally
#     across the whole invocation rather than per (package, name), so a
#     patch-added test sharing an excluded test's bare name was silently
#     skipped in every package, not just the one it was meant to exempt
#     (finding #2).
#
# The property this script now satisfies: no signal used to conclude "no
# failures occurred" may be forgeable by code inside the package under
# test. Exit codes, package-level Actions, and self-reported summaries
# are ALL forgeable. Per-test JSON terminal events are NOT forgeable in
# themselves -- each is written to stdout in real time as its test
# completes, before TestMain regains control to call os.Exit (verified
# empirically) -- but reading them is only trustworthy when cross-checked
# against an INDEPENDENT count of how many tests were supposed to
# produce one. `go test -list` was evaluated and rejected for that count:
# it executes TestMain (verified empirically), making it exactly as
# forgeable as everything else tried so far. bench/scripts/testenum/ is
# the independent enumerator instead: a separate Go module (so it is
# invisible to this repository's own `go list ./...`, same mechanism
# ADR-F01-01 already uses for the fixture submodule) that statically
# parses a package's `_test.go` source with go/parser -- it never
# compiles or runs a single line of the package under test, so nothing
# that package's own code does at runtime can alter its output.
#
# check_p2p_green() below is per package (fixing finding #2's global
# -skip scope: exclusions apply strictly per (package, name), built and
# applied one package at a time) and per package requires ALL of: every
# testenum-enumerated test not deliberately excluded has a per-test
# terminal event (no missing evidence -- catches a TestMain that never
# calls m.Run() at all, exactly UAT round 2's UAT-003 shape); every
# OBSERVED per-test terminal event for that package is pass or skip,
# never fail (a real per-test fail is trustworthy signal in itself, so
# this alone catches finding #1's masked regression even though the
# process's own exit code was forged to 0); and the invocation's own
# exit code is still 0 as a corroborating check (catches a TestMain that
# forges a *nonzero* exit despite fully clean, complete per-test evidence
# -- UAT round 3's masked-behind-the-probe scenario). No single one of
# these three is sufficient alone; all three together leave no
# forgeable-signal gap. F2P already modeled part of this discipline
# before this round (explicit per-test "pass" AND an isolated exit code
# of 0) and is unchanged here.
#
# Code review round 6 (this round, the human-authorized bounded follow-up
# to round 5's escalation) found the fail-check above still had a gap of
# its own, plus a gap in the enumerator it depends on:
#   - 6a: testenum's enumerator matched a test's parameter type by AST
#     shape (`*testing.T` spelled literally) rather than by resolved
#     identity, so a dot import (`import . "testing"`) or a renamed
#     import (`import gotest "testing"`) -- both of which `go test`
#     genuinely compiles and runs -- produced ZERO output from testenum,
#     making that real, running, failing test invisible to `expected`.
#   - 6b (the more direct defect, independent of 6a): the fail-check
#     itself iterated over `expected` (testenum's enumerated set) and
#     looked up each enumerated name in `results` -- so ANY real fail
#     event whose name was not in `expected`, for ANY reason, was
#     silently discarded, even though `results` already held it as real,
#     non-forgeable evidence. A perfect enumerator would not have saved
#     this: the fail-check was scanning the wrong set.
# Both are fixed together: testenum now resolves each parameter's type
# via go/types (identity: which package actually declares it, not which
# identifier spells it -- see testenum/main.go's header for the full
# resolution and negative-case reasoning), and the fail-check below now
# scans every entry `results` holds for a package, never `expected`, so
# a residual gap in enumeration (testenum's import resolution is
# intentionally scoped to the standard library; see its header) can at
# most weaken condition 1 (missing-evidence completeness), never again
# suppress a real, already-observed failure.
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

# testenum: independent, non-executing enumeration of a package's
# top-level Go test functions (see testenum/main.go's own header for the
# full trust-model rationale). It lives in its own Go module (a separate
# go.mod under this directory, same mechanism ADR-F01-01 uses for the
# fixture submodule) so it is never part of this repository's own `go
# list ./...`. Built once here and reused for every package/item this
# run evaluates.
TESTENUM_DIR="$SCRIPT_DIR/testenum"
[[ -f "$TESTENUM_DIR/go.mod" ]] || {
	echo "admit: testenum module not found: $TESTENUM_DIR" >&2
	exit 1
}
testenum_build_dir="$(mktemp -d)"
trap 'rm -rf "$testenum_build_dir"' EXIT
testenum_bin="$testenum_build_dir/testenum"
(cd "$TESTENUM_DIR" && go build -o "$testenum_bin" .) || {
	echo "admit: failed to build testenum (independent test enumerator)" >&2
	exit 1
}

python3 - "$corpus_yaml_abs" "$CHECKOUT_SCRIPT" "$item_id" "$patch_override" "$testenum_bin" <<'PYEOF'
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

import yaml

corpus_yaml_path, checkout_script, item_id, patch_override, testenum_bin = sys.argv[1:6]
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


def go_list_packages(checkout_dir, packages):
    """Resolves package glob(s)/patterns (e.g. ["./..."]) to concrete
    (import_path, absolute_dir) pairs via `go list`. Pure build-graph
    metadata resolution -- verified empirically that `go list` executes
    no code belonging to any resolved package (unlike `go test -list`,
    which does run TestMain -- see this file's header). Raises if `go
    list` itself fails (e.g. the checkout does not build at all)."""
    cmd = ["go", "list", "-buildvcs=false", "-f", "{{.ImportPath}}|{{.Dir}}"] + list(packages)
    proc = subprocess.run(cmd, cwd=checkout_dir, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(f"'go list' failed for packages {packages!r}: {proc.stderr.strip()}")
    result = []
    for line in proc.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        import_path, _, directory = line.partition("|")
        result.append((import_path, directory))
    return result


_UNSAFE_RUN_SELECTOR_CHARS = frozenset("|[(\\")


def validate_run_selector_grammar(run_selector):
    """Rejects a `run_selector` unless it stays inside a conservative
    grammar: no `|` (top-level alternation), `[` (character class), `(`
    (group), or `\\` (escape) -- anchors (`^`/`$`), wildcards (`.`/`*`),
    and `/` (subtest separator) are all still allowed.

    Why the restriction: Go's real `-run` semantics (testing.splitRegexp)
    split a selector on top-level `|` INTO AN ALTERNATION FIRST, then split
    EACH ALTERNATIVE on `/` independently. A naive `split("/", 1)[0]` (used
    by run_selector_top_level_pattern below) does not replicate that --
    e.g. "^TestA$/^NoSuchSub$|^TestZ$" makes real `go test -run` select
    and run BOTH TestA and TestZ, but a naive split yields only {TestA},
    silently dropping TestZ from `expected` (B053 round 2: reimplementing
    Go's full alternation-then-split algorithm in Python was considered
    and rejected as unnecessary complexity for a bounded fix). Within this
    restricted grammar there is no `|` to worry about splitting
    independently, and no `[`/`(`/`\\` to confuse a naive split, so
    `split("/", 1)[0]` is provably correct for every selector this
    function accepts.

    Incidentally also closes two related findings: RE2 (Go) and Python's
    `re` disagree on POSIX character classes like `[[:alpha:]]` (banning
    `[` closes this), and Python's backtracking `re` engine is vulnerable
    to catastrophic backtracking (ReDoS) on constructs RE2's linear-time
    matching accepts safely (banning `\\` and `(` is defense-in-depth here
    -- RE2 itself can't express the nested-quantifier/backreference
    patterns that cause ReDoS, but Python's `re` can).

    Raises RuntimeError if run_selector uses any of these constructs, so a
    filter result that can't be trusted is never silently produced."""
    bad = sorted(_UNSAFE_RUN_SELECTOR_CHARS.intersection(run_selector))
    if bad:
        raise RuntimeError(
            f"p2p_set run_selector {run_selector!r} uses unsupported regexp "
            f"syntax {bad!r} -- admit.sh's run_selector filtering only "
            "supports a restricted grammar (no |, [, (, or \\) because "
            "naive per-\"/\"-element splitting cannot safely replicate "
            "Go's full -run alternation semantics for those constructs"
        )


def run_selector_top_level_pattern(run_selector):
    """Extracts the element of a `-run` pattern that Go matches against a
    top-level test's own name -- the part before the first "/" (see -run's
    documented per-"/"-element matching, also relied on by
    anchored_alternation above). `expected` only ever holds bare top-level
    names (testenum never enumerates subtests), so this is the only part
    of run_selector relevant to filtering it: a selector targeting a
    specific subtest (e.g. "TestFoo/bar") still requires TestFoo itself to
    match its own element for that subtest to run at all, and Go still
    forwards a terminal event for the top-level test regardless of which
    of its subtests matched. Returns None if run_selector is falsy.

    Callers MUST run run_selector through validate_run_selector_grammar()
    first -- outside that restricted grammar, a plain split("/", 1)[0]
    does not correctly replicate Go's real -run semantics (see that
    function's docstring)."""
    if not run_selector:
        return None
    return run_selector.split("/", 1)[0]


def filter_expected_by_run_selector(expected, run_selector):
    """Filters an `expected` set of bare top-level test names down to only
    those a non-empty `run_selector` would actually select, so `expected`
    stays in sync with what run_go_tests()'s own `-run run_selector` just
    executed (B053: previously `expected` ignored run_selector entirely,
    so every test the selector legitimately excluded was reported as a
    spurious missing terminal event).

    Deliberately does NOT shell out to `go test -list` for this: this
    file's header established that `-list` executes TestMain and is
    therefore exactly as forgeable as every other signal this file
    refuses to trust (see "Evidence-forgeability property" above). Plain
    regex matching against testenum's own non-executing enumeration keeps
    the same non-forgeable trust model.

    run_selector is first validated against a restricted grammar (see
    validate_run_selector_grammar) and, within that grammar, matched with
    Python's re against each bare name using only the top-level (pre-"/")
    element (see run_selector_top_level_pattern) -- NOT a full replication
    of Go's per-"/"-element alternation matching, which the restricted
    grammar makes unnecessary (B053 round 2)."""
    if not run_selector:
        return expected
    validate_run_selector_grammar(run_selector)
    top_level_pattern = run_selector_top_level_pattern(run_selector)
    try:
        pattern = re.compile(top_level_pattern)
    except re.error as exc:
        raise RuntimeError(
            f"p2p_set run_selector {run_selector!r} is not a usable regexp "
            f"(top-level element {top_level_pattern!r}): {exc}"
        )
    return {name for name in expected if pattern.search(name)}


def enumerate_tests(pkg_dir):
    """Independent, non-executing enumeration of a package directory's
    top-level Go test function names via testenum (see
    bench/scripts/testenum/main.go) -- never forgeable by code in the
    target package, because that package is never built or run to
    produce this list; only its *_test.go source text is parsed."""
    proc = subprocess.run([testenum_bin, pkg_dir], capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(
            f"testenum failed to enumerate {pkg_dir!r}: {proc.stderr.strip()}"
        )
    return {line.strip() for line in proc.stdout.splitlines() if line.strip()}


def check_p2p_green(checkout_dir, packages, run_selector, exclude_from_p2p):
    """Evaluates P2P-green per the evidence-forgeability property stated
    at the top of this file. Runs `go test` ONE PACKAGE AT A TIME (never
    a shared "./..." invocation) so each package's -skip pattern is built
    from ONLY the exclude_from_p2p names whose own "<pkg>::" prefix
    matches that exact package -- fixing finding #2 (a global -skip
    pattern exempted a same-named test in every package, not just the
    one it was meant to excuse).

    For each package, "clean" requires ALL three of:
      1. every testenum-enumerated test not in this package's own skip
         set, and selected by run_selector when one is given (B053: see
         filter_expected_by_run_selector -- a test the selector itself
         excluded is never expected to produce a terminal event), has a
         per-test terminal event (no missing evidence -- this alone
         catches a TestMain that never calls m.Run(), so nothing it
         "does" can hide a package that produced zero results);
      2. none of the OBSERVED per-test terminal events for this package
         is "fail" (code review round 6, finding 6b: this scans every
         entry `results` actually holds for this package, not just
         entries whose name also appears in the enumerated set -- a real
         per-test fail is trustworthy in itself regardless of whether
         testenum's enumeration happens to know its name, which closes
         finding #1's masked regression two ways at once: m.Run()
         genuinely recorded the failing test before TestMain forged the
         process exit to 0, AND the test's name never has to match
         anything testenum enumerated for its failure to count);
      3. the invocation's own raw exit code is 0 (a corroborating check
         -- this alone catches a TestMain that forges a *non-zero* exit
         despite fully clean, complete per-test evidence, i.e. UAT round
         3's probe-masked scenario).
    No single one of the three is sufficient; together they leave no
    forgeable-signal gap -- and critically, (2) no longer depends on (1)
    at all, so a gap in testenum's own recognition can at most weaken
    completeness checking (1), never suppress a real, already-observed
    failure (2). A package with zero enumerated tests still must exit 0
    (it must still build), even though there is nothing to
    cross-reference.

    A fourth, SET-WIDE (not per-package) precondition guards (1) itself
    (B053 finding 1, code review round on PR #203): when run_selector is
    given, it must select at least one enumerated test SOMEWHERE across
    this p2p_set's packages, or the whole call raises RuntimeError before
    returning any verdict. This is deliberately set-wide rather than
    per-package -- a selector legitimately scoped to only one package's
    tests would otherwise trip a per-package version of this guard on
    every OTHER package in a multi-package (e.g. "./...") set, which is
    not a misconfiguration. Without this precondition, a selector matching
    nothing (typo, overly narrow pattern) would filter `expected` down to
    the empty set in every package -- (1)'s missing-evidence check becomes
    vacuously true (nothing expected, nothing missing) and `go test -run
    <no-match>` exits 0 with nothing to run -- so a package that ran zero
    tests would read as P2P-green.

    Returns (all_clean: bool, problem_packages: list[str]) where each
    problem_packages entry names the package and exactly which
    condition(s) failed, for verdict diagnostics."""
    all_clean = True
    problem_packages = []
    # Zero-match run_selector guard (B053 finding 1, see docstring above):
    # track how many enumerated tests existed before filtering vs. how many
    # survived filtering, across every package in this p2p_set (not
    # per-package -- see rationale above).
    total_pre_filter_candidates = 0
    total_post_filter_selected = 0

    for import_path, pkg_dir in go_list_packages(checkout_dir, packages):
        pkg_skip_names = {
            bare_test_name(name)
            for name in exclude_from_p2p
            if name.split("::", 1)[0] == import_path
        }
        skip_pattern = anchored_alternation(pkg_skip_names) if pkg_skip_names else None

        results, _problem_pkgs, returncode = run_go_tests(
            checkout_dir, [import_path], run_pattern=run_selector, skip_pattern=skip_pattern
        )
        pre_filter_expected = enumerate_tests(pkg_dir) - pkg_skip_names
        expected = filter_expected_by_run_selector(pre_filter_expected, run_selector)
        total_pre_filter_candidates += len(pre_filter_expected)
        total_post_filter_selected += len(expected)
        observed = {
            bare_test_name(identity)
            for identity in results
            if identity.split("::", 1)[0] == import_path
        }
        missing = sorted(expected - observed)
        # Iterate over every OBSERVED terminal event for this package,
        # not over `expected` (code review round 6, finding 6b). The
        # earlier form looked up each *enumerated* name in `results`, so
        # any real per-test "fail" event whose name was not in
        # `expected` -- for ANY reason, including a genuine gap in
        # testenum's own recognition (6a) -- was silently discarded even
        # though `results` already held it as real, non-forgeable
        # evidence (`go test -json`'s own per-test terminal events,
        # which this file's header establishes are trustworthy in
        # themselves). A name this package's own skip pattern actually
        # excluded cannot appear in `results` at all -- it never ran --
        # so no separate exclusion is needed here; every entry in
        # `results` for this package is real, observed, in-scope
        # evidence.
        failed = sorted({
            bare_test_name(identity)
            for identity, outcome in results.items()
            if identity.split("::", 1)[0] == import_path and outcome == "fail"
        })
        package_clean = not missing and not failed and returncode == 0

        if not package_clean:
            all_clean = False
            detail = []
            if missing:
                detail.append(f"missing terminal event for {missing}")
            if failed:
                detail.append(f"failing {failed}")
            if returncode != 0 and not missing and not failed:
                detail.append(f"exit code {returncode} despite clean, complete per-test evidence")
            problem_packages.append(f"{import_path} ({'; '.join(detail)})")

    if run_selector and total_pre_filter_candidates > 0 and total_post_filter_selected == 0:
        raise RuntimeError(
            f"p2p_set run_selector {run_selector!r} matched none of the "
            f"{total_pre_filter_candidates} testenum-enumerated test(s) across "
            f"{packages!r} -- refusing to treat 0 tests actually run as "
            "P2P-green; check the selector for a typo or an overly narrow pattern"
        )

    return all_clean, problem_packages


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


def prepare_checkout(item, parent):
    """Provisions a fresh checkout-fixture.sh checkout under `parent` (an
    already-created temp directory owned by the caller) and stages this
    item's F2P harness file(s) into it. Returns the checkout_dir path.
    Isolated from evaluate()'s try/finally so the temp-directory lifecycle
    (one mkdtemp per evaluate() call, one finally rmtree) stays owned by
    the caller, as required by the evidence-forgeability property."""
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
            f"item {item['id']}: checkout-fixture.sh failed: {exc.stderr}"
        ) from exc
    copy_f2p_files(item, checkout_dir)
    return checkout_dir


def run_check(checks, check_name, result, failure_code):
    """Records an already-run check's `result` in `checks[check_name]` and
    returns (failing_check, extra): `failing_check` is `failure_code` if
    the check failed, else None; `extra` is diagnostic detail (e.g. a
    problem-packages list) when `result` is a (passed, extra) tuple, else
    None. Callers gate each call on `if failing_check is None`, so the
    check itself has already run by the time this is called."""
    passed, extra = result if isinstance(result, tuple) else (result, None)
    checks[check_name] = passed
    return (None if passed else failure_code), extra


def f2p_red_or_green(checkout_dir, f2p_packages, f2p_run_pattern, f2p_ids, expect):
    """F2P-isolated run shared by the f2p_red_at_base and
    f2p_green_post_patch checks: -run scopes execution to exactly this
    item's own F2P test name(s), so nothing this item does not name can
    produce a per-test result here. `expect` is "fail" (base check) or
    "pass" (post-patch check). For "pass", a claim is trusted only when
    the isolated run's own process exit also agrees nothing else
    misbehaved -- otherwise a rogue TestMain forcing a non-zero exit
    regardless of this test's real outcome would read as a false "green"
    (see evaluate()'s docstring)."""
    results, _problem_pkgs, rc = run_go_tests(
        checkout_dir, f2p_packages, run_pattern=f2p_run_pattern
    )
    if expect == "pass" and rc != 0:
        return False
    return all(results.get(t) == expect for t in f2p_ids)


def patch_applies(checkout_dir, patch_path):
    """Applies the item's reference patch to the checkout in-place via
    `git apply` and reports whether it applied cleanly."""
    apply_proc = subprocess.run(
        ["git", "apply", patch_path],
        cwd=checkout_dir,
        capture_output=True,
        text=True,
    )
    return apply_proc.returncode == 0


def evaluate(item, patch_path):
    """See this file's header for the full evidence-forgeability property
    and its history across rounds 2, 3, and this round's fix. In short:
    P2P-green is decided by check_p2p_green() (per-package, testenum-
    cross-referenced, exit-code-corroborated); F2P-green is decided by an
    explicit per-test "pass" match AND an isolated invocation exit code
    of 0 (unchanged from round 3, already sound per code review round
    5's own confirmation)."""
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
        checkout_dir = prepare_checkout(item, parent)

        failing_check, _extra = run_check(
            checks,
            "f2p_red_at_base",
            f2p_red_or_green(checkout_dir, f2p_packages, f2p_run_pattern, f2p_ids, "fail"),
            FAIL_F2P_GREEN_AT_BASE,
        )

        if failing_check is None:
            failing_check, unexplained_failed_packages = run_check(
                checks,
                "p2p_green_at_base",
                check_p2p_green(checkout_dir, packages, run_selector, exclude_from_p2p),
                FAIL_P2P_RED_AT_BASE,
            )

        if failing_check is None:
            failing_check, _extra = run_check(
                checks,
                "patch_applies",
                patch_applies(checkout_dir, patch_path),
                FAIL_PATCH_APPLY,
            )

        if failing_check is None:
            failing_check, _extra = run_check(
                checks,
                "f2p_green_post_patch",
                f2p_red_or_green(checkout_dir, f2p_packages, f2p_run_pattern, f2p_ids, "pass"),
                FAIL_F2P_STILL_RED,
            )

            if failing_check is None:
                failing_check, unexplained_failed_packages = run_check(
                    checks,
                    "p2p_green_post_patch",
                    check_p2p_green(checkout_dir, packages, run_selector, exclude_from_p2p),
                    FAIL_P2P_RED_POST_PATCH,
                )
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
