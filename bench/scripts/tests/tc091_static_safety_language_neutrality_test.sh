#!/usr/bin/env bash
# TC-091 / T-E40-F10-014: static safety and scope-boundary scan (spec.md
# REQ-NF-001, REQ-NF-005, REQ-NF-006; test-plan.md TC-091 full body;
# AC-014/AC-T1/AC-T2).
#
# Runs the four enumerated static scans (lib/static-scan.sh) over TC-091's
# exact Input file list: run-lifecycle-batch.sh, run-review-comparison.sh,
# pilot-ledger.sh, verify-retention-root.sh, aggregate-lifecycle.sh,
# report-lifecycle.sh, lib/spend-gate.sh, and
# bench/reports/lifecycle-baseline-schema.yaml. The shared retention and
# validation helpers are included as well: retain_pair writes retained
# artifacts, path-safety.sh owns containment checks, and
# verify_pair_retention validates the published root. Omitting those helpers
# would leave the production write/read boundary outside this static gate.
#
#   AC-T1 -- all four scans (a)-(d) report an explicit zero-match count
#            with the scanned file list enumerated.
#   AC-T2 -- a write-destination variable built by string concatenation
#            across two separate assignment statements is still traced by
#            scan (a)'s assignment-chain walk. Proven with a genuine fixture
#            script below (not documentation): a positive fixture
#            (concatenation from the declared --retention-root) must PASS,
#            and a negative fixture (concatenation from an unrelated
#            scenario field, never referencing the root) must FAIL --
#            proving the scan actually traces the chain rather than
#            pattern-matching a single-line assignment or vacuously passing
#            everything.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
STATIC_SCAN_LIB="$SCRIPT_DIR/lib/static-scan.sh"

fail() {
	echo "TC-091 FAIL: $1" >&2
	exit 1
}

[[ -f "$STATIC_SCAN_LIB" ]] || fail "lib/static-scan.sh missing"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

# shellcheck source=lib/static-scan.sh
source "$STATIC_SCAN_LIB"

# TC-091 Input: the exact enumerated file list.
FILES=(
	"$BENCH_DIR/scripts/run-lifecycle-batch.sh"
	"$BENCH_DIR/scripts/run-review-comparison.sh"
	"$BENCH_DIR/scripts/pilot-ledger.sh"
	"$BENCH_DIR/scripts/verify-retention-root.sh"
	"$BENCH_DIR/scripts/aggregate-lifecycle.sh"
	"$BENCH_DIR/scripts/report-lifecycle.sh"
	"$BENCH_DIR/scripts/lib/spend-gate.sh"
	"$BENCH_DIR/scripts/lib/retain_pair"
	"$BENCH_DIR/scripts/lib/path-safety.sh"
	"$BENCH_DIR/scripts/lib/verify_pair_retention"
	"$BENCH_DIR/reports/lifecycle-baseline-schema.yaml"
)

for f in "${FILES[@]}"; do
	[[ -f "$f" ]] || fail "enumerated scan target missing: $f"
done

# ===========================================================================
# AC-T1: all four scans, over the real committed F10 file set. Each scan
# function itself prints the scanned-file enumeration and the explicit
# zero-match/violation count (or fails loudly naming a real violation) --
# asserted here by requiring exit 0 from every one.
# ===========================================================================
static_scan_write_path "${FILES[@]}" || fail "scan (a) write-path assignment-chain trace found a real violation (see stderr above) -- see Notes: fix the violation, do not weaken the scan"
echo

static_scan_forbidden_path "${FILES[@]}" || fail "scan (b) forbidden-path scan found a real internal/, cmd/, .sharkconfig.json, or migrations/ reference"
echo

static_scan_language_branch "${FILES[@]}" || fail "scan (c) language-branch scan found a real fixture-language branch"
echo

static_scan_content_disclosure "${FILES[@]}" || fail "scan (d) content-disclosure scan found a real credential/prompt-body/transcript disclosure"
echo

# ===========================================================================
# AC-T2: a write-destination variable built by string concatenation across
# TWO separate assignment statements must still be traced correctly by
# scan (a) -- proven with a genuine fixture pair, not just documentation.
# ===========================================================================
FIXTURE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc091.XXXXXX")"
trap 'rm -rf "$FIXTURE_DIR"' EXIT

POSITIVE_FIXTURE="$FIXTURE_DIR/positive_concat.sh"
cat >"$POSITIVE_FIXTURE" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		retention_root="$2"
		shift 2
		;;
	esac
done

# The write-destination variable is built across TWO separate assignment
# statements: the first anchors it under retention_root, the second
# concatenates a literal suffix segment onto the first (AC-T2).
dest="$retention_root/scenarios"
dest="$dest/rep001"
mkdir -p "$dest"
EOF

NEGATIVE_FIXTURE="$FIXTURE_DIR/negative_concat.sh"
cat >"$NEGATIVE_FIXTURE" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		retention_root="$2"
		shift 2
		;;
	--scenario-field)
		scenario_field="$2"
		shift 2
		;;
	esac
done

# Also built across TWO separate assignment statements, but the chain
# never references retention_root at all -- an unrelated scenario field
# instead. Scan (a) must still catch this: a real violation, not a
# vacuous pass just because it "looks like" the positive fixture's shape.
base="$scenario_field/export"
base="$base/leaked.json"
mkdir -p "$base"
EOF

if ! static_scan_write_path "$POSITIVE_FIXTURE" >/tmp/tc091-positive.out 2>&1; then
	cat /tmp/tc091-positive.out >&2
	fail "AC-T2 positive fixture: a retention-root-anchored variable built by concatenation across two assignment statements was wrongly flagged as a violation"
fi
grep -q "0 violations" /tmp/tc091-positive.out || fail "AC-T2 positive fixture: scan (a) did not report 0 violations"

if static_scan_write_path "$NEGATIVE_FIXTURE" >/tmp/tc091-negative.out 2>/tmp/tc091-negative.err; then
	cat /tmp/tc091-negative.out
	fail "AC-T2 negative fixture: an unrelated-source variable built by concatenation across two assignment statements was wrongly traced as root-anchored (scan (a) is not actually walking the chain)"
fi
grep -q '\$base' /tmp/tc091-negative.err || fail "AC-T2 negative fixture: scan (a) failed for the wrong reason (expected it to name the \$base destination)"
grep -q "not the retention root" /tmp/tc091-negative.err || fail "AC-T2 negative fixture: scan (a) failed for the wrong reason (expected an 'unrelated'/not-root classification)"

rm -f /tmp/tc091-positive.out /tmp/tc091-negative.out /tmp/tc091-negative.err

echo "TC-091 PASS: all four static scans report zero violations over the complete enumerated F10 production file set (AC-T1); scan (a) correctly traces a two-statement concatenation chain both to the retention root (positive fixture) and away from it (negative fixture, AC-T2)"
