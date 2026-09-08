#!/usr/bin/env bash
# TC-114 / T-E40-F11-014: static doc-gate closing REQ-F-007/008/009/013's
# cross-cutting documentation obligation (spec.md AC-F11-22; §7.1 row 22;
# §2.1.7/§2.1.7a; test-plan.md TC-22).
#
# The gate: no LIVE documentation surface under bench/ or this epic's
# docs/plan/ tree may say "four families" (or a variant of that phrase) --
# every live surface must reflect the six admitted families (epic, feature,
# task, bug, change_card, tech_debt). Pattern frozen by spec.md's own closed
# inventory: `four famil|four deliver|all four families`.
#
# Scope: recursive over bench/ and
# docs/plan/E40-shark-bench-workflow-benchmarking-harness/, EXCLUDING this
# feature's own directory (E40-F11-six-family-lifecycle-benchmark-readiness-
# and-cover/) -- its spec.md, feature.md, research-report.md, test-plan.md,
# and task files are working documents that discuss the four->six migration
# historically/by design (spec.md §2.1.7a: "outside this feature's own
# directory"). They are never "live" assertions of the currently admitted
# set and are not gated.
#
# Allowlist (both required by spec.md §2.1.7a, keyed on path-suffix +
# matched-text, never on line number -- this file's own history already
# shows line numbers drift):
#
#   1. docs/plan/.../E40-F05-lifecycle-scenario-corpus-and-adapter-contract/
#      test-plan.md ("...four families, rejects malformed...") -- F05 is a
#      COMPLETED feature; its test plan is a historical record of what F05
#      actually admitted. Rewriting it would falsify an audit trail
#      (spec.md §2.1.7a's sole documented exclusion).
#
#   2. docs/plan/.../uat-plan.md, UAT-08 row ("E40-F05 (four families);
#      E40-F11 (widened to six)") -- accurate historical attribution of
#      WHICH feature admitted how many families, not a claim about the
#      CURRENT admitted set. uat-plan.md is one of the four parent
#      documents T-E40-F11-014 is scoped to verify, not re-author (already
#      applied by a prior commit) -- this line is unavoidable there.
#
#      NOTE FOR THE PARENT LOOP: spec.md §2.1.7a/§7.1a row 22 enumerate only
#      ONE surviving exclusion (the F05 test-plan.md line), from a grep run
#      on 2026-09-04 that predates the uat-plan.md edit adding UAT-08's
#      "E40-F05 (four families)" attribution. That edit is itself one of
#      the "already applied" parent-doc changes T-E40-F11-014 was told to
#      verify, not re-author. This gate therefore allowlists TWO entries,
#      not spec.md's documented one -- flagged here rather than silently
#      matching a stale spec claim or rewriting a file outside this task's
#      scope.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
EPIC_DOCS_DIR="$REPO_ROOT/docs/plan/E40-shark-bench-workflow-benchmarking-harness"
FEATURE_DIRNAME="E40-F11-six-family-lifecycle-benchmark-readiness-and-cover"

fail() {
	echo "TC-114 FAIL: $1" >&2
	exit 1
}

[[ -d "$BENCH_DIR" ]] || fail "bench directory missing: $BENCH_DIR"
[[ -d "$EPIC_DOCS_DIR" ]] || fail "epic docs directory missing: $EPIC_DOCS_DIR"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

PATTERN='four famil|four deliver|all four families'

# ---------------------------------------------------------------------------
# The gate itself: grep the two roots (excluding this feature's own
# directory), then partition hits into "allowlisted" (documented historical
# attribution) and "violations" (live drift). Prints violations to stdout,
# one per line as "path:lineno: content". Returns 0 iff there are zero
# violations. Also prints the allowlisted hits it found, on stderr, so a
# caller can assert the exclusion set is exactly what it expects.
#
# This script's own basename is excluded from the scan: it necessarily
# quotes the pattern and the phrase "four families" throughout its own
# comments and synthetic fixtures, and it lives under bench/scripts/tests/,
# inside the scanned bench/ root.
# ---------------------------------------------------------------------------
run_doc_gate() {
	local bench_root="$1" epic_docs_root="$2"
	local hits violation_count=0 allowed_count=0
	hits="$(grep -rn -E "$PATTERN" "$bench_root" "$epic_docs_root" \
		--exclude-dir="$FEATURE_DIRNAME" \
		--exclude="tc114_six_family_doc_gate_test.sh" 2>/dev/null || true)"

	if [[ -z "$hits" ]]; then
		return 0
	fi

	while IFS= read -r line; do
		[[ -z "$line" ]] && continue
		local path content
		path="${line%%:*}"
		content="${line#*:}"   # lineno:content
		content="${content#*:}" # content only

		if [[ "$path" == */E40-F05-lifecycle-scenario-corpus-and-adapter-contract/test-plan.md ]] \
			&& [[ "$content" == *"four families"* ]]; then
			allowed_count=$((allowed_count + 1))
			echo "ALLOWLISTED: $line" >&2
			continue
		fi
		if [[ "$path" == "$epic_docs_root/uat-plan.md" ]] \
			&& [[ "$content" == *"E40-F05 (four families)"* ]]; then
			allowed_count=$((allowed_count + 1))
			echo "ALLOWLISTED: $line" >&2
			continue
		fi

		violation_count=$((violation_count + 1))
		echo "$line"
	done <<<"$hits"

	echo "ALLOWLISTED_COUNT=$allowed_count" >&2
	[[ "$violation_count" -eq 0 ]]
}

# ===========================================================================
# Case 1 (positive): the real, live repository tree must pass, with exactly
# the two documented allowlisted hits and zero violations.
# ===========================================================================
REAL_STDERR="$WORKDIR/real-stderr.txt"
VIOLATIONS="$(run_doc_gate "$BENCH_DIR" "$EPIC_DOCS_DIR" 2>"$REAL_STDERR")" || {
	echo "$VIOLATIONS"
	fail "live repository still has a 'four famil'-pattern hit outside the documented allowlist (named above)"
}
ALLOWED_LINE_COUNT="$(grep -c '^ALLOWLISTED:' "$REAL_STDERR" || true)"
[[ "$ALLOWED_LINE_COUNT" -eq 2 ]] || fail "expected exactly 2 allowlisted historical hits (F05 test-plan.md + uat-plan.md UAT-08), found $ALLOWED_LINE_COUNT -- the doc-gate's allowlist and the live tree have drifted apart"

echo "TC-114: live repository passes (0 violations, 2 documented historical exclusions)"

# ===========================================================================
# Negative-case scaffolding: a tiny synthetic tree, not the real repository,
# so the mutation cases below never touch real files and run instantly.
# (WORKDIR was created above, before the real-tree check.)
# ===========================================================================
build_synthetic_tree() {
	local root="$1"
	rm -rf "$root"
	mkdir -p "$root/bench" \
		"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/$FEATURE_DIRNAME" \
		"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract"

	cat >"$root/bench/README.md" <<'EOF'
# bench README (synthetic)
The harness admits all six lifecycle families.
EOF

	cat >"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md" <<'EOF'
# architecture (synthetic)
All six families are covered by the delivery boundary.
EOF

	cat >"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md" <<'EOF'
# uat-plan (synthetic)
| UAT-08 | All six lifecycle families are admitted correctly | ... | E40-F05 (four families); E40-F11 (widened to six); G8; I-04 |
EOF

	cat >"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/test-plan.md" <<'EOF'
# F05 test-plan (synthetic, historical record)
  four families, rejects malformed with field named -> AC-001/AC-005;
EOF

	cat >"$root/docs/plan/E40-shark-bench-workflow-benchmarking-harness/$FEATURE_DIRNAME/spec.md" <<'EOF'
# F11 spec (synthetic, self-referential prose)
This revision widens the admitted vocabulary from four families to six.
No live surface may say "four families".
EOF
}

# ===========================================================================
# Case 2 (negative, per-surface): revert a live surface to "four families"
# and require the gate to fail, naming that file:line.
# ===========================================================================
build_synthetic_tree "$WORKDIR/case2"
sed -i 's/all six lifecycle families/all four families/' "$WORKDIR/case2/bench/README.md"

if VIOLATIONS="$(run_doc_gate "$WORKDIR/case2/bench" "$WORKDIR/case2/docs/plan/E40-shark-bench-workflow-benchmarking-harness" 2>/dev/null)"; then
	fail "case 2 (bench/README.md reverted to 'four families'): gate should have failed but passed"
fi
[[ "$VIOLATIONS" == *"README.md"* ]] || fail "case 2: gate failed but did not name bench/README.md; got: $VIOLATIONS"
echo "TC-114: case 2 passes (reverted bench/README.md correctly fails, naming the file)"

# Same case, second surface: architecture.md.
build_synthetic_tree "$WORKDIR/case2b"
sed -i 's/All six families are covered/All four families are covered/' "$WORKDIR/case2b/docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md"

if VIOLATIONS="$(run_doc_gate "$WORKDIR/case2b/bench" "$WORKDIR/case2b/docs/plan/E40-shark-bench-workflow-benchmarking-harness" 2>/dev/null)"; then
	fail "case 2b (architecture.md reverted to 'four families'): gate should have failed but passed"
fi
[[ "$VIOLATIONS" == *"architecture.md"* ]] || fail "case 2b: gate failed but did not name architecture.md; got: $VIOLATIONS"
echo "TC-114: case 2b passes (reverted architecture.md correctly fails, naming the file)"

# ===========================================================================
# Case 3 (negative, exclusion-intentionality): the F05 allowlist entry is
# load-bearing, not an artifact of a pattern that happens never to match
# F05's real text. Removing the underlying "four families" text from F05's
# (synthetic) historical record -- WITHOUT touching the gate's allowlist
# logic at all -- must make that allowlist branch stop firing (1 allowed
# hit instead of 2) while the gate still passes overall. This proves the
# branch reacts to real matched text rather than being a dead/decorative
# exception.
# ===========================================================================
build_synthetic_tree "$WORKDIR/case3"
cat >"$WORKDIR/case3/docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/test-plan.md" <<'EOF'
# F05 test-plan (synthetic, historical record -- rewritten to name the
# admitted count without matching the gate's pattern)
  all six families are exercised by AC-001/AC-005;
EOF

STDERR_FILE="$WORKDIR/case3-stderr.txt"
if ! VIOLATIONS="$(run_doc_gate "$WORKDIR/case3/bench" "$WORKDIR/case3/docs/plan/E40-shark-bench-workflow-benchmarking-harness" 2>"$STDERR_FILE")"; then
	fail "case 3: gate should still pass once F05's text no longer matches the pattern; violations: $VIOLATIONS"
fi
CASE3_ALLOWED="$(grep -c '^ALLOWLISTED:' "$STDERR_FILE" || true)"
[[ "$CASE3_ALLOWED" -eq 1 ]] || fail "case 3: expected exactly 1 allowlisted hit (uat-plan.md only) once F05's text was rewritten, got $CASE3_ALLOWED -- the F05 allowlist branch is not reacting to real matched text"
echo "TC-114: case 3 passes (F05 exclusion is proven load-bearing: it stops firing when the underlying text does, gate still passes overall)"

echo "TC-114 PASS: six-family doc gate holds on the live tree and its mutation/removal negative cases behave as specified"
