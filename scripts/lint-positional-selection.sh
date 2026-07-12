#!/usr/bin/env bash
# Guard against the positional-selection defect class: picking element [0]
# (or [len-1]) of a status/transition slice whose order is incidental
# (alphabetical sort, map iteration) and treating that position as "the
# default/primary choice".
#
# Semantic selection belongs in the named selectors
# (internal/config/workflow/selectors.go). Where a position IS a guaranteed
# contract (e.g. AvailableTransitions[0] is pass-first by construction via
# uniqueSortedOutcomeTargets, or _start_ preserves declaration order),
# annotate the line:
#
#     x := statuses[0] //shark:ordered <why the order is a contract>
#
# Test files are exempt: tests pin ordering contracts on purpose.
set -euo pipefail
cd "$(dirname "$0")/.."

pattern='(Statuses|Transitions|Targets)(\(\))?\[(0|len\([^)]*\)-1)\]'

hits=$(grep -rnE "$pattern" --include='*.go' internal cmd 2>/dev/null \
	| grep -v '_test.go' \
	| grep -v 'internal/config/workflow/selectors.go' \
	| grep -vE '^[^:]+:[0-9]+:[[:space:]]*//' \
	| grep -v '//shark:ordered' || true)

if [[ -n "$hits" ]]; then
	echo "positional selection on an ordered status/transition slice:" >&2
	echo "$hits" >&2
	echo >&2
	echo "Use a named selector (internal/config/workflow/selectors.go), or — if the" >&2
	echo "position is a guaranteed contract at this site — annotate the line with" >&2
	echo "'//shark:ordered <reason>'." >&2
	exit 1
fi

echo "positional-selection lint: OK"
