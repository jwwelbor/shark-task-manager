#!/usr/bin/env bash
# TC-041 (test-plan.md AC test matrix / Caller-Path Contracts; T-E40-F05-007
# task spec Test Cases; AC-021, AC-T3).
#
# Static argument-trace check, not a runtime test: `eval-predicate.sh` must
# never read anything beyond its own three positional arguments, and no
# comparison-file naming convention this feature's ADR-F05-10 forbids may
# exist anywhere the scenario tree is committed. This is the discriminator
# the plan's "Acceptance-criteria review" names between an absolute
# implementation and a base-relative one (TC-036's mutations alone cannot
# tell the two apart -- see test-plan.md's own discussion). No stubbed
# JSON, no hand-authored happy-path smoke test: TC-036's boundary/negative
# matrix over real adapter `test`/`lint` output is deferred to the seed
# tasks (T-E40-F05-008/010/011/013) per this task's own spec, and this
# file's forbidden mocks are exactly that kind of artifact -- red here means
# "the script does not exist yet", nothing more.
#
# Four independent parts, all against the real committed source text and a
# real `find`, per the Caller-Path Contract's "do not restrict the grep to a
# hand-picked line range or a single naming convention":
#   1. Every `open(...)` call inside the script resolves to one of the
#      three variables the script itself binds from its own positional
#      arguments (sys.argv[1:4]) -- not a literal path, not a derived path.
#   2. No other file-read construct (`cat`, `source`, `jq -f`, or a stray
#      shell `<` redirection outside the one `<<'PYEOF'` heredoc marker)
#      appears in the script's bash preamble (the heredoc body is inert
#      text to bash -- its own reads are covered by part 1).
#   3. `grep -riE 'ledger|baseline|snapshot'` over the whole script -- zero
#      hits, broader than the literal string "ledger" (task Notes for
#      Agent; AC-T3).
#   4. `find bench/scenarios -iname '*ledger*' -o -iname '*baseline*' -o
#      -iname '*snapshot*'` -- zero results; no such file is committed
#      anywhere under bench/scenarios/.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

TARGET="$SCRIPTS_DIR/eval-predicate.sh"

fail() {
	echo "TC-041 FAIL: $1" >&2
	exit 1
}

[[ -f "$TARGET" ]] || fail "eval-predicate.sh missing: $TARGET (T-E40-F05-007 not yet implemented)"

# ---------------------------------------------------------------------------
# Part 1 -- every open(...) call traces to one of the script's own three
# positional-argument variables.
# ---------------------------------------------------------------------------
echo "TC-041: part 1 - every open(...) call traces to \$1/\$2/\$3"

argv_unpack_line="$(grep -nE '=[[:space:]]*sys\.argv\[1:4\]' "$TARGET" || true)"
[[ -n "$argv_unpack_line" ]] || fail "no 'VAR1, VAR2, VAR3 = sys.argv[1:4]' unpack line found -- cannot trace open() calls to the three positional arguments"
[[ "$(echo "$argv_unpack_line" | wc -l)" -eq 1 ]] || fail "expected exactly one sys.argv[1:4] unpack line, found: $argv_unpack_line"

lhs="$(echo "$argv_unpack_line" | sed -E 's/^[0-9]+:[[:space:]]*//; s/[[:space:]]*=.*$//')"
allowed_vars="$(echo "$lhs" | tr -d ' ' | tr ',' '\n' | sort -u)"
[[ "$(echo "$allowed_vars" | grep -c .)" -eq 3 ]] || fail "expected exactly 3 variables bound from sys.argv[1:4], got: $lhs"

open_lines="$(grep -nE '\bopen\(' "$TARGET" || true)"
open_count="$(echo "$open_lines" | grep -c . || true)"
[[ "$open_count" -eq 3 ]] || fail "expected exactly 3 open(...) calls in eval-predicate.sh (one per positional argument), found $open_count:
$open_lines"

while IFS= read -r line; do
	[[ -n "$line" ]] || continue
	var="$(echo "$line" | sed -E 's/^[0-9]+:.*\bopen\(([A-Za-z_][A-Za-z0-9_]*).*/\1/')"
	[[ -n "$var" ]] || fail "could not parse the argument of an open(...) call: $line"
	echo "$allowed_vars" | grep -qx "$var" || fail "open($var...) does not trace to a positional-argument variable ($lhs): $line"
done <<<"$open_lines"

echo "TC-041(part 1: all 3 open() calls trace to sys.argv[1:4] via {$lhs}) PASS"

# ---------------------------------------------------------------------------
# Part 2 -- no other file-read construct in the bash preamble (everything
# before the python3 heredoc starts; the heredoc body itself is inert text
# to bash and is covered by part 1's open() trace).
# ---------------------------------------------------------------------------
echo "TC-041: part 2 - no cat/source/jq -f/stray redirection in the bash preamble"

heredoc_start_line="$(grep -nE "<<'PYEOF'" "$TARGET" | head -n1 | cut -d: -f1)"
[[ -n "$heredoc_start_line" ]] || fail "no <<'PYEOF' heredoc marker found -- cannot isolate the bash preamble from the python body"

preamble="$(head -n "$((heredoc_start_line - 1))" "$TARGET")"

if hits="$(echo "$preamble" | grep -nE '\bcat\b|\bsource\b|jq[[:space:]]+-f')"; then
	fail "forbidden file-read construct (cat/source/jq -f) found in the bash preamble:
$hits"
fi

# Any '<' in the preamble must appear only inside a comment line (first
# non-whitespace character '#') or inside the quoted usage message (a
# printed string, not a redirection token) -- a real shell redirection
# outside those two would be a hidden read this check must catch.
redirect_hits="$(echo "$preamble" | grep -n '<' | grep -vE '^[0-9]+:[[:space:]]*#' | grep -vE 'echo "usage' || true)"
[[ -z "$redirect_hits" ]] || fail "possible shell redirection ('<') outside a comment line in the bash preamble:
$redirect_hits"

echo "TC-041(part 2: bash preamble has no cat/source/jq -f and no stray '<' redirection) PASS"

# ---------------------------------------------------------------------------
# Part 3 -- no comparison-file naming convention anywhere in the script,
# broader than the literal string "ledger" (AC-T3).
# ---------------------------------------------------------------------------
echo "TC-041: part 3 - no ledger/baseline/snapshot naming convention in eval-predicate.sh"

if hits="$(grep -inE 'ledger|baseline|snapshot' "$TARGET")"; then
	fail "forbidden comparison-file naming convention found in eval-predicate.sh:
$hits"
fi

echo "TC-041(part 3: no ledger/baseline/snapshot token anywhere in eval-predicate.sh) PASS"

# ---------------------------------------------------------------------------
# Part 4 -- no such file is committed anywhere under bench/scenarios/.
# ---------------------------------------------------------------------------
echo "TC-041: part 4 - no ledger/baseline/snapshot-named file under bench/scenarios/"

scenarios_dir="$BENCH_DIR/scenarios"
if [[ -d "$scenarios_dir" ]]; then
	stray_files="$(find "$scenarios_dir" -iname '*ledger*' -o -iname '*baseline*' -o -iname '*snapshot*')"
	[[ -z "$stray_files" ]] || fail "found ledger/baseline/snapshot-named file(s) under bench/scenarios/:
$stray_files"
else
	echo "TC-041(part 4: bench/scenarios/ does not exist -- vacuously zero matches) SKIP" >&2
fi

echo "TC-041(part 4: no ledger/baseline/snapshot-named file under bench/scenarios/) PASS"

echo "TC-041: PASS"
