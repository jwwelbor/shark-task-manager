#!/usr/bin/env bash
# Tier 1b/Tier 2 bench self-test wrapper (test-plan.md "Test tiers" table).
# Invokes every tc0NN_<slug>_test.sh in this directory in sequence. Each
# later script task (T-E40-F01-006 through -009) registers its own
# tc0NN_*_test.sh into the list below.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

tests=(
	"$SCRIPT_DIR/tc003_clean_checkout_test.sh"
	"$SCRIPT_DIR/tc004_admit_full_set_test.sh"
	"$SCRIPT_DIR/tc005_admit_rejection_branches_test.sh"
	"$SCRIPT_DIR/tc006_admit_reproducibility_test.sh"
	"$SCRIPT_DIR/tc007_ledger_reproducibility_test.sh"
	"$SCRIPT_DIR/tc008_lint_position_shift_test.sh"
	"$SCRIPT_DIR/tc009_lint_diff_boundary_test.sh"
	"$SCRIPT_DIR/tc010_test_diff_transitions_test.sh"
	"$SCRIPT_DIR/tc011_toolchain_guard_test.sh"
	"$SCRIPT_DIR/tc013_admit_offline_test.sh"
	"$SCRIPT_DIR/tc014_run_one_smoke_test.sh"
	"$SCRIPT_DIR/tc015_collect_run_record_test.sh"
	"$SCRIPT_DIR/tc016_canary_runsurface_test.sh"
)

status=0
for t in "${tests[@]}"; do
	name="$(basename "$t")"
	echo "==> $name"
	if "$t"; then
		echo "PASS: $name"
	else
		echo "FAIL: $name" >&2
		status=1
	fi
	echo
done

exit "$status"
