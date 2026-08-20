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
	"$SCRIPT_DIR/tc017_run_batch_test.sh"
	"$SCRIPT_DIR/tc018_aggregate_report_test.sh"
	"$SCRIPT_DIR/tc019_replay_manifest_test.sh"
	"$SCRIPT_DIR/tc020_zero_go_change_test.sh"
	"$SCRIPT_DIR/tc031_adapter_conformance_test.sh"
	"$SCRIPT_DIR/tc032_admission_full_confirmation_test.sh"
	"$SCRIPT_DIR/tc033_admission_rejection_test.sh"
	"$SCRIPT_DIR/tc034_admission_determinism_test.sh"
	"$SCRIPT_DIR/tc035_toolchain_identity_test.sh"
	"$SCRIPT_DIR/tc036_final_predicate_test.sh"
	"$SCRIPT_DIR/tc037_repo_hygiene_test.sh"
	"$SCRIPT_DIR/tc038_frozen_interface_test.sh"
	"$SCRIPT_DIR/tc039_offline_capability_test.sh"
	"$SCRIPT_DIR/tc040_fixture_base_verification_test.sh"
	"$SCRIPT_DIR/tc041_predicate_argument_trace_test.sh"
	"$SCRIPT_DIR/tc043_root_policy_isolation_test.sh"
	"$SCRIPT_DIR/tc044_time_ledger_reconciliation_test.sh"
	"$SCRIPT_DIR/tc045_candidate_identity_test.sh"
	"$SCRIPT_DIR/tc046_artifact_record_test.sh"
	"$SCRIPT_DIR/tc047_usage_mapping_canary_test.sh"
	"$SCRIPT_DIR/tc048_evaluator_access_ordering_test.sh"
	"$SCRIPT_DIR/tc049_snapshot_replay_test.sh"
	"$SCRIPT_DIR/tc050_partial_evidence_test.sh"
	"$SCRIPT_DIR/tc051_evidence_offline_determinism_test.sh"
	"$SCRIPT_DIR/tc053_live_egress_denial_test.sh"
	"$SCRIPT_DIR/tc054_replay_resolver_test.sh"
	"$SCRIPT_DIR/tc055_lineage_reconciliation_test.sh"
	"$SCRIPT_DIR/tc056_bundle_disclosure_test.sh"
	"$SCRIPT_DIR/tc057_non_applicable_record_test.sh"
	"$SCRIPT_DIR/tc058_prelude_placement_test.sh"
	"$SCRIPT_DIR/tc059_replay_offline_determinism_test.sh"
	"$SCRIPT_DIR/tc060_lifecycle_runner_contract_test.sh"
	"$SCRIPT_DIR/tc061_lifecycle_runner_loop_test.sh"
	"$SCRIPT_DIR/tc062_lifecycle_runner_limits_test.sh"
	"$SCRIPT_DIR/tc063_review_finding_capture_test.sh"
	"$SCRIPT_DIR/tc064_lifecycle_runner_offline_determinism_test.sh"
	"$SCRIPT_DIR/tc065_prelude_question_isolation_test.sh"
	"$SCRIPT_DIR/tc066_lifecycle_worker_question_handoff_test.sh"
	"$SCRIPT_DIR/tc067_lifecycle_evaluation_truth_test.sh"
	"$SCRIPT_DIR/tc068_heldback_oracle_isolation_test.sh"
	"$SCRIPT_DIR/tc069_calibration_boundary_test.sh"
	"$SCRIPT_DIR/tc070_comparison_identity_test.sh"
	"$SCRIPT_DIR/tc075_content-identity_x12_test.sh"
	"$SCRIPT_DIR/tc071_review_finding_normalization_test.sh"
	"$SCRIPT_DIR/tc072_review_comparison_modes_test.sh"
	"$SCRIPT_DIR/tc073_evaluation_replay_determinism_test.sh"
	"$SCRIPT_DIR/tc074_invalid-retention-and-aggregation_test.sh"
	"$SCRIPT_DIR/tc076_static-safety-language-neutrality_test.sh"
	"$SCRIPT_DIR/tc077_full-regression-registration_test.sh"
	"$SCRIPT_DIR/tc079_operator_preview_zero_spend_test.sh"
	"$SCRIPT_DIR/tc080_spend_gate_refusal_test.sh"
	"$SCRIPT_DIR/tc081_pilot_ledger_gate_test.sh"
	"$SCRIPT_DIR/tc082_retention_layout_test.sh"
	"$SCRIPT_DIR/tc083_failed_oracle_diagnosis_test.sh"
	"$SCRIPT_DIR/tc084_time_reconciliation_test.sh"
	"$SCRIPT_DIR/tc085_review_value_report_test.sh"
	"$SCRIPT_DIR/tc086_artifact_use_and_replay_proxy_test.sh"
	"$SCRIPT_DIR/tc087_review_comparison_operator_test.sh"
	"$SCRIPT_DIR/tc088_dimension_separation_and_noise_band_test.sh"
	"$SCRIPT_DIR/tc089_phase_separation_test.sh"
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
