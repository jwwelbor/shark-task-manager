#!/usr/bin/env bash
# TC-094: E40 top-level operator safety, identity, and comparison contract.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

fail() {
	echo "TC-094 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh is missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
grep -q 'driver_bin = SCRIPTS_DIR / "run-lifecycle-batch.sh"' \
	"$SCRIPTS_DIR/lib/e40_benchmark.py" \
	|| fail "provider-backed operator execution is not pinned to the F10 batch driver"
[[ "$(grep -c 'staging_handle = secure_mkdtemp' "$SCRIPTS_DIR/lib/e40_benchmark.py")" -eq 2 ]] \
	|| fail "setup and validate-variant are not both wired to descriptor-relative staging"

python3 - "$SCRIPTS_DIR/lib/e40_benchmark.py" "$WORKDIR" <<'PY'
import importlib.util
import os
import pathlib
import sys

module_path, workdir_raw = sys.argv[1:]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
workdir = pathlib.Path(workdir_raw)

for case_name, label in (
    ("setup", "operator setup staging directory"),
    ("validate-variant", "variant staging directory"),
):
    parent = workdir / f"{case_name}-staging-parent"
    moved = workdir / f"{case_name}-staging-moved"
    target = workdir / f"{case_name}-staging-target"
    parent.mkdir()
    target.mkdir()
    real_open = module.open_real_directory_fd
    swapped = False

    def swapping_open(path, open_label, *, create):
        global swapped
        descriptor = real_open(path, open_label, create=create)
        if not swapped and pathlib.Path(path) == parent:
            os.rename(parent, moved)
            parent.symlink_to(target, target_is_directory=True)
            swapped = True
        return descriptor

    module.open_real_directory_fd = swapping_open
    try:
        module.secure_mkdtemp(parent, f".{case_name}.", label)
    except module.OperatorError as exc:
        if "changed after validation" not in str(exc):
            raise SystemExit(f"unexpected {case_name} staging-swap refusal: {exc}")
    else:
        raise SystemExit(f"{case_name} staging accepted a parent swapped to a symlink")
    finally:
        module.open_real_directory_fd = real_open
    if any(target.iterdir()):
        raise SystemExit(f"{case_name} staging swap wrote to the external target")
    if any(moved.iterdir()):
        raise SystemExit(f"{case_name} staging refusal left a partial directory")

for case_name in ("execution", "comparison"):
    root = workdir / f"{case_name}-lock-root"
    moved = workdir / f"{case_name}-lock-moved"
    target = workdir / f"{case_name}-lock-target"
    root.mkdir()
    target.mkdir()
    real_open_lock = module.open_lock_directory_fd
    swapped = False

    def swapping_lock_open(root_descriptor, name, label):
        global swapped
        descriptor = real_open_lock(root_descriptor, name, label)
        if not swapped:
            lock_path = root / name
            os.rename(lock_path, moved)
            lock_path.symlink_to(target, target_is_directory=True)
            swapped = True
        return descriptor

    module.open_lock_directory_fd = swapping_lock_open
    try:
        module.acquire_lock(root)
    except module.OperatorError as exc:
        if "changed after validation" not in str(exc):
            raise SystemExit(f"unexpected {case_name} lock-swap refusal: {exc}")
    else:
        raise SystemExit(f"{case_name} lock accepted a directory swapped to a symlink")
    finally:
        module.open_lock_directory_fd = real_open_lock
    if any(target.iterdir()):
        raise SystemExit(f"{case_name} lock swap wrote to the external target")
    if any(moved.iterdir()):
        raise SystemExit(f"{case_name} lock refusal wrote PID state before validation")
PY

live_db="$REPO_ROOT/shark-tasks.db"
live_db_before="absent"
if [[ -f "$live_db" ]]; then
	live_db_before="$(sha256sum "$live_db" | awk '{print $1}')"
fi

provider_spy="$WORKDIR/provider-spy.sh"
cat >"$provider_spy" <<'SH'
#!/usr/bin/env bash
echo called >>"${PROVIDER_SPY_LOG:?}"
exit 97
SH
chmod +x "$provider_spy"

setup_symlink_target="$WORKDIR/setup-symlink-target"
setup_symlink_parent="$WORKDIR/setup-symlink-parent"
mkdir -p "$setup_symlink_target"
ln -s "$setup_symlink_target" "$setup_symlink_parent"
set +e
"$OPERATOR" setup --out "$setup_symlink_parent/operator" \
	>/dev/null 2>"$WORKDIR/setup-symlink.err"
setup_symlink_rc=$?
set -e
[[ "$setup_symlink_rc" -eq 2 ]] \
	|| fail "symlinked setup parent returned $setup_symlink_rc, expected 2"
[[ -z "$(find "$setup_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked setup parent redirected operator writes"

config_symlink_target="$WORKDIR/config-symlink-target"
config_symlink_parent="$WORKDIR/config-symlink-parent"
config_symlink_operator="$WORKDIR/config-symlink-operator"
mkdir -p "$config_symlink_target"
ln -s "$config_symlink_target" "$config_symlink_parent"
set +e
"$OPERATOR" setup --out "$config_symlink_operator" \
	--config-out "$config_symlink_parent/e40-demo.yaml" \
	>/dev/null 2>"$WORKDIR/config-symlink.err"
config_symlink_rc=$?
set -e
[[ "$config_symlink_rc" -eq 2 ]] \
	|| fail "symlinked config-output parent returned $config_symlink_rc, expected 2"
[[ -z "$(find "$config_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked config-output parent redirected config writes"
[[ ! -e "$config_symlink_operator" ]] \
	|| fail "setup mutated the operator root before rejecting its config-output parent"

operator_root="$WORKDIR/operator"
set +e
"$OPERATOR" setup --out "$operator_root" >"$WORKDIR/setup-a.out" 2>&1 &
setup_pid_a=$!
"$OPERATOR" setup --out "$operator_root" >"$WORKDIR/setup-b.out" 2>&1 &
setup_pid_b=$!
wait "$setup_pid_a"; setup_rc_a=$?
wait "$setup_pid_b"; setup_rc_b=$?
set -e
if [[ "$setup_rc_a" -ne 0 && "$setup_rc_b" -ne 0 ]]; then
	fail "both concurrent setup invocations failed ($setup_rc_a, $setup_rc_b)"
fi
for setup_rc in "$setup_rc_a" "$setup_rc_b"; do
	[[ "$setup_rc" -eq 0 || "$setup_rc" -eq 4 ]] \
		|| fail "unexpected concurrent setup exit $setup_rc"
done

preflight_symlink_target="$WORKDIR/preflight-symlink-target"
preflight_symlink_out="$WORKDIR/preflight-symlink-out"
mkdir -p "$preflight_symlink_target"
ln -s "$preflight_symlink_target" "$preflight_symlink_out"
set +e
"$OPERATOR" preflight --config "$operator_root/e40-demo.yaml" --reps 1 \
	--out "$preflight_symlink_out" >/dev/null 2>"$WORKDIR/preflight-symlink.err"
preflight_symlink_rc=$?
set -e
[[ "$preflight_symlink_rc" -eq 2 ]] \
	|| fail "symlinked preflight output returned $preflight_symlink_rc, expected 2"
[[ -z "$(find "$preflight_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked preflight output redirected derived writes"

set +e
PROVIDER_SPY_LOG="$WORKDIR/provider.log" LIFECYCLE_ADAPTER="$provider_spy" \
	"$OPERATOR" preflight --config "$operator_root/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/preflight-a.out" 2>&1 &
preflight_pid_a=$!
PROVIDER_SPY_LOG="$WORKDIR/provider.log" LIFECYCLE_ADAPTER="$provider_spy" \
	"$OPERATOR" preflight --config "$operator_root/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/preflight-b.out" 2>&1 &
preflight_pid_b=$!
wait "$preflight_pid_a"; preflight_rc_a=$?
wait "$preflight_pid_b"; preflight_rc_b=$?
set -e
if [[ "$preflight_rc_a" -ne 0 && "$preflight_rc_b" -ne 0 ]]; then
	fail "both concurrent preflight invocations failed ($preflight_rc_a, $preflight_rc_b)"
fi
for preflight_rc in "$preflight_rc_a" "$preflight_rc_b"; do
	[[ "$preflight_rc" -eq 0 || "$preflight_rc" -eq 4 ]] \
		|| fail "unexpected concurrent preflight exit $preflight_rc"
done
[[ ! -e "$WORKDIR/provider.log" ]] || fail "preflight invoked the provider adapter"

# Zero-spend is a structural boundary: neither an environment-selected batch
# driver nor config-selected provider/binary seams may execute in preflight.
E40_RUN_LIFECYCLE_BATCH_BIN="$provider_spy" PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	"$OPERATOR" preflight --config "$operator_root/e40-demo.yaml" --reps 1 \
	--out "$WORKDIR/preflight-driver-spy" >/dev/null
[[ ! -e "$WORKDIR/provider.log" ]] || fail "preflight invoked an environment-selected batch driver"

fd_preview_root="$WORKDIR/fd-preview-root"
mkdir -p "$fd_preview_root"
exec {fd_preview_descriptor}<"$fd_preview_root"
"$SCRIPTS_DIR/run-lifecycle-batch.sh" \
	--batch "$operator_root/preflight/baseline/batch-policy.yaml" \
	--retention-root "$fd_preview_root" \
	--retention-root-fd "$fd_preview_descriptor" --mode preview --reps 1 \
	>"$WORKDIR/fd-preview.out"
fd_preview_moved="$WORKDIR/fd-preview-moved"
mv "$fd_preview_root" "$fd_preview_moved"
mkdir "$fd_preview_root"
set +e
"$SCRIPTS_DIR/run-lifecycle-batch.sh" \
	--batch "$operator_root/preflight/baseline/batch-policy.yaml" \
	--retention-root "$fd_preview_root" \
	--retention-root-fd "$fd_preview_descriptor" --mode preview --reps 1 \
	>/dev/null 2>"$WORKDIR/fd-preview-swap.err"
fd_preview_swap_rc=$?
set -e
exec {fd_preview_descriptor}<&-
[[ "$fd_preview_swap_rc" -eq 2 ]] \
	|| fail "batch driver accepted a retention root replaced after its descriptor opened"
[[ -z "$(find "$fd_preview_root" -mindepth 1 -print -quit)" ]] \
	|| fail "batch driver wrote through a replaced retention-root path"

cp "$operator_root/e40-demo.yaml" "$WORKDIR/hostile-runtime.yaml"
python3 - "$WORKDIR/hostile-runtime.yaml" "$provider_spy" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["runtime"]["lifecycle_adapter"] = sys.argv[2]
config["runtime"]["provider_command"] = [sys.argv[2]]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	"$OPERATOR" preflight --config "$WORKDIR/hostile-runtime.yaml" --reps 1 \
	--out "$WORKDIR/preflight-runtime-spy" >/dev/null
[[ ! -e "$WORKDIR/provider.log" ]] || fail "preflight invoked a config-selected provider seam"

cp "$operator_root/e40-demo.yaml" "$WORKDIR/scalar-runtime.yaml"
python3 - "$WORKDIR/scalar-runtime.yaml" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["runtime"] = "not-a-mapping"
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
set +e
"$OPERATOR" preflight --config "$WORKDIR/scalar-runtime.yaml" --reps 1 \
	--out "$WORKDIR/preflight-scalar-runtime" \
	>/dev/null 2>"$WORKDIR/scalar-runtime.err"
scalar_runtime_rc=$?
set -e
[[ "$scalar_runtime_rc" -eq 2 ]] \
	|| fail "scalar runtime config returned $scalar_runtime_rc, expected bounded refusal 2"
grep -q 'runtime must be a YAML mapping' "$WORKDIR/scalar-runtime.err" \
	|| fail "scalar runtime refusal did not name the mapping requirement: $(cat "$WORKDIR/scalar-runtime.err")"
! grep -q 'Traceback' "$WORKDIR/scalar-runtime.err" \
	|| fail "scalar runtime refusal emitted an uncaught traceback"

cp "$operator_root/e40-demo.yaml" "$WORKDIR/hostile-binary.yaml"
python3 - "$WORKDIR/hostile-binary.yaml" "$provider_spy" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["shark_binary"] = sys.argv[2]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
set +e
PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	"$OPERATOR" preflight --config "$WORKDIR/hostile-binary.yaml" --reps 1 \
	--out "$WORKDIR/preflight-binary-spy" >/dev/null 2>&1
hostile_binary_rc=$?
set -e
[[ "$hostile_binary_rc" -eq 2 ]] \
	|| fail "preflight accepted a config-selected binary (rc=$hostile_binary_rc)"
[[ ! -e "$WORKDIR/provider.log" ]] || fail "preflight executed a config-selected binary"

live_db_after="absent"
if [[ -f "$live_db" ]]; then
	live_db_after="$(sha256sum "$live_db" | awk '{print $1}')"
fi
[[ "$live_db_before" == "$live_db_after" ]] || fail "preflight or setup mutated the live repository database"

python3 - "$operator_root/setup-result.json" "$operator_root/preflight/baseline/preflight-result.json" <<'PY'
import json
import re
import sys

setup = json.load(open(sys.argv[1], encoding="utf-8"))
preflight = json.load(open(sys.argv[2], encoding="utf-8"))
required_setup = {
    "content_bundle_digest", "prompt_bundle_digest", "workflow_bundle_digest",
    "enabled_gate_identity", "scenario_matrix", "shark_binary", "root_keys",
}
missing = sorted(required_setup - set(setup))
if missing:
    raise SystemExit(f"setup result missing identity fields: {missing}")
for field in ("content_bundle_digest", "prompt_bundle_digest", "workflow_bundle_digest"):
    if not re.fullmatch(r"[0-9a-f]{64}", setup[field]):
        raise SystemExit(f"setup {field} is not a digest")
if len(setup["scenario_matrix"]) != 4 or len(setup["root_keys"]) != 4:
    raise SystemExit("setup did not seed the four admitted lifecycle families")
if preflight["provider_calls"] != 0 or preflight["live_database_mutations"] != 0:
    raise SystemExit("preflight did not report its zero-spend boundary")
if preflight["provider_ready"] is not False or not preflight["missing_real_runtime_inputs"]:
    raise SystemExit("preflight hid missing real-runtime inputs")
if preflight["dry_run_stage_resolution_failures"] and not preflight["dry_run_limitation"]:
    raise SystemExit("preflight hid the existing F08 dry-run artifact limitation")
identity = preflight.get("planned_identity") or {}
for field in ("scenario_set_digest", "fixture_set_digest", "adapter_set_digest", "prompt_bundle_digest"):
    if not re.fullmatch(r"[0-9a-f]{64}", str(identity.get(field, ""))):
        raise SystemExit(f"preflight planned identity missing {field}")
PY

# Configure only test-local runtime placeholders. The driver below is a
# non-provider stub used solely to inspect manifests; provider behavior remains
# covered by the real F10 spend-gate suite.
mkdir -p "$WORKDIR/i05"
python3 - "$operator_root/e40-demo.yaml" "$provider_spy" "$WORKDIR/i05" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["runtime"]["lifecycle_adapter"] = sys.argv[2]
for scenario_id, entry in config["scenario_roots"].items():
    bundle = pathlib.Path(sys.argv[3]) / scenario_id
    bundle.mkdir(parents=True, exist_ok=True)
    entry["i05_bundle_dir"] = str(bundle)
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY

prompt_override="$WORKDIR/prompt-v2"
policy_override="$WORKDIR/policy-v2"
workflow_override="$WORKDIR/workflow-model-v2"
mkdir -p "$prompt_override" "$policy_override" "$workflow_override"
cat >"$prompt_override/operator.md" <<'EOF'
Use the controlled E40 prompt variant.
EOF
cat >"$policy_override/policy.yaml" <<'EOF'
policy: controlled-e40-variant
EOF
cp -R "$operator_root/scratch-template/shark-data/workflow/." "$workflow_override/"
python3 - "$workflow_override/bug.yaml" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
document = yaml.safe_load(path.read_text(encoding="utf-8"))

def change_first(value):
    if isinstance(value, dict):
        if isinstance(value.get("model"), str):
            value["model"] = value["model"] + "-e40-variant"
            return True
        return any(change_first(child) for child in value.values())
    if isinstance(value, list):
        return any(change_first(child) for child in value)
    return False

if not change_first(document):
    raise SystemExit("fixture workflow contains no model route")
path.write_text(yaml.safe_dump(document, sort_keys=False), encoding="utf-8")
PY

set +e
"$OPERATOR" validate-variant --config "$operator_root/e40-demo.yaml" \
	--variant-name prompt-v2 --prompt-root "$prompt_override" >"$WORKDIR/validate-prompt-a.out" 2>&1 &
validate_pid_a=$!
"$OPERATOR" validate-variant --config "$operator_root/e40-demo.yaml" \
	--variant-name prompt-v2 --prompt-root "$prompt_override" >"$WORKDIR/validate-prompt-b.out" 2>&1 &
validate_pid_b=$!
wait "$validate_pid_a"; validate_rc_a=$?
wait "$validate_pid_b"; validate_rc_b=$?
set -e
if [[ "$validate_rc_a" -ne 0 && "$validate_rc_b" -ne 0 ]]; then
	fail "both concurrent variant validations failed ($validate_rc_a, $validate_rc_b)"
fi
for validate_rc in "$validate_rc_a" "$validate_rc_b"; do
	[[ "$validate_rc" -eq 0 || "$validate_rc" -eq 4 ]] \
		|| fail "unexpected concurrent variant-validation exit $validate_rc"
done
"$OPERATOR" validate-variant --config "$operator_root/e40-demo.yaml" \
	--variant-name policy-v2 --policy-root "$policy_override" >"$WORKDIR/validate-policy.out"
"$OPERATOR" validate-variant --config "$operator_root/e40-demo.yaml" \
	--variant-name model-v2 --workflow-root "$workflow_override" >"$WORKDIR/validate-model.out"

driver_stub="$WORKDIR/driver-stub.sh"
cat >"$driver_stub" <<'SH'
#!/usr/bin/env bash
echo invoked >>"${DRIVER_STUB_LOG:?}"
exit 97
SH
chmod +x "$driver_stub"

fixture_harness="$WORKDIR/operator-fixture-harness.py"
cat >"$fixture_harness" <<'PY'
import hashlib
import importlib.util
import json
import os
import pathlib
import subprocess
import sys

spec = importlib.util.spec_from_file_location("e40_benchmark", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
real_run_process = module.run_process


def fixture_run_process(command, **kwargs):
    if pathlib.Path(command[0]) == module.SCRIPTS_DIR / "run-lifecycle-batch.sh":
        driver_log = os.environ.get("FIXTURE_DRIVER_LOG")
        if driver_log:
            with pathlib.Path(driver_log).open("a", encoding="utf-8") as stream:
                stream.write("invoked\n")
        forbidden_spy = os.environ.get("FIXTURE_EXPECT_NO_SPY")
        child_env = kwargs.get("env") or {}
        retained_spies = [
            key
            for key, value in child_env.items()
            if key != "FIXTURE_EXPECT_NO_SPY" and value == forbidden_spy
        ]
        if forbidden_spy and retained_spies:
            raise RuntimeError(
                f"execution environment retained hostile override spies: {retained_spies}"
            )
        values = {}
        for index, value in enumerate(command[:-1]):
            if value.startswith("--"):
                values[value] = command[index + 1]
        root = pathlib.Path(values["--retention-root"])
        policy = pathlib.Path(values["--batch"])
        write_root = pathlib.Path(
            f"/proc/self/fd/{values['--retention-root-fd']}"
            if "--retention-root-fd" in values
            else values["--retention-root"]
        )
        swap_target = os.environ.get("FIXTURE_SWAP_RUN_ROOT_TARGET")
        if swap_target:
            os.rename(root, pathlib.Path(swap_target))
            root.mkdir()
        batch = {
            "phase": "lifecycle_v2",
            "batch_id": f"batch-{root.name}-{values['--mode']}",
            "mode": values["--mode"],
            "retention_root": str(root),
            "batch_policy_digest": hashlib.sha256(policy.read_bytes()).hexdigest(),
            "min_reps": int(values["--reps"]),
            "ceilings": {
                "max_cost_usd": values["--max-cost-usd"],
                "max_wall_clock_seconds": values["--max-wall-clock-seconds"],
                "max_generated_tasks": values["--max-generated-tasks"],
            },
            "acknowledgement_ref": {
                "flag": "--acknowledge-provider-spend",
                "present": "--acknowledge-provider-spend" in command,
            },
        }
        (write_root / "batch.json").write_text(
            json.dumps(batch, indent=2, sort_keys=True) + "\n", encoding="utf-8"
        )
        authority_target = os.environ.get("FIXTURE_AUTHORITY_SYMLINK_TARGET")
        if authority_target:
            authorities = write_root / "batch-authorities"
            authorities.mkdir(exist_ok=True)
            (authorities / f"{batch['batch_id']}.json").symlink_to(
                pathlib.Path(authority_target)
            )
        return subprocess.CompletedProcess(command, 0, '{"fixture_driver":true}\n', "")
    return real_run_process(command, **kwargs)


module.run_process = fixture_run_process
sys.argv = ["e40-benchmark.sh", *sys.argv[2:]]
raise SystemExit(module.main())
PY

configured_run_target="$WORKDIR/configured-run-target"
configured_run_store="$WORKDIR/configured-run-store"
configured_run_config="$WORKDIR/configured-run-store.yaml"
mkdir -p "$configured_run_target"
ln -s "$configured_run_target" "$configured_run_store"
cp "$operator_root/e40-demo.yaml" "$configured_run_config"
python3 - "$configured_run_config" "$configured_run_store" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["run_store"] = sys.argv[2]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
set +e
python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
	baseline --config "$configured_run_config" --run-id linked-run-store \
	--reps 1 --acknowledge-provider-spend --max-cost-usd 1 \
	--max-wall-clock-seconds 30 --max-generated-tasks 1 \
	>/dev/null 2>"$WORKDIR/configured-run-store.err"
configured_run_store_rc=$?
set -e
[[ "$configured_run_store_rc" -eq 2 ]] \
	|| fail "symlinked config run_store returned $configured_run_store_rc, expected 2"
[[ -z "$(find "$configured_run_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked config run_store redirected retained writes"

assert_configured_scratch_refused() {
	local case_name="$1" config_path="$2"
	local driver_log="$WORKDIR/$case_name-driver.log"
	set +e
	FIXTURE_DRIVER_LOG="$driver_log" \
		python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
		baseline --config "$config_path" --run-id "$case_name" \
		--reps 1 --acknowledge-provider-spend --max-cost-usd 1 \
		--max-wall-clock-seconds 30 --max-generated-tasks 1 \
		>/dev/null 2>"$WORKDIR/$case_name.err"
	local scratch_rc=$?
	set -e
	[[ "$scratch_rc" -eq 2 ]] \
		|| fail "$case_name scratch-root configuration returned $scratch_rc, expected 2"
	[[ ! -e "$driver_log" ]] \
		|| fail "$case_name scratch-root configuration invoked the provider driver"
}

live_scratch_config="$WORKDIR/live-scratch.yaml"
cp "$operator_root/e40-demo.yaml" "$live_scratch_config"
python3 - "$live_scratch_config" "$REPO_ROOT" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
for entry in config["scenario_roots"].values():
    entry["scratch_root"] = sys.argv[2]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
live_db_before_scratch_refusal="$live_db_before"
if [[ -f "$live_db" ]]; then
	live_db_before_scratch_refusal="$(sha256sum "$live_db" | awk '{print $1}')"
fi
assert_configured_scratch_refused live-project-scratch "$live_scratch_config"
live_db_after_scratch_refusal="absent"
if [[ -f "$live_db" ]]; then
	live_db_after_scratch_refusal="$(sha256sum "$live_db" | awk '{print $1}')"
fi
[[ "$live_db_before_scratch_refusal" == "$live_db_after_scratch_refusal" ]] \
	|| fail "live-project scratch refusal mutated the repository database"

scratch_alias="$WORKDIR/scratch-template-alias"
symlink_scratch_config="$WORKDIR/symlink-scratch.yaml"
ln -s "$operator_root/scratch-template" "$scratch_alias"
cp "$operator_root/e40-demo.yaml" "$symlink_scratch_config"
python3 - "$symlink_scratch_config" "$scratch_alias" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
for entry in config["scenario_roots"].values():
    entry["scratch_root"] = sys.argv[2]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
scratch_digest_before="$($SCRIPTS_DIR/lib/digest_path "$operator_root/scratch-template")"
assert_configured_scratch_refused symlinked-scratch "$symlink_scratch_config"
scratch_digest_after="$($SCRIPTS_DIR/lib/digest_path "$operator_root/scratch-template")"
[[ "$scratch_digest_before" == "$scratch_digest_after" ]] \
	|| fail "symlinked scratch-root refusal mutated the setup-owned scratch tree"

set +e
DRIVER_STUB_LOG="$WORKDIR/driver.log" E40_RUN_LIFECYCLE_BATCH_BIN="$driver_stub" \
	"$OPERATOR" baseline --config "$operator_root/e40-demo.yaml" --run-id no-ack \
	--reps 1 --max-cost-usd 1 --max-wall-clock-seconds 30 --max-generated-tasks 1 \
	>"$WORKDIR/no-ack.out" 2>"$WORKDIR/no-ack.err"
no_ack_rc=$?
set -e
[[ "$no_ack_rc" -eq 3 ]] || fail "provider command without acknowledgement returned $no_ack_rc, expected 3"
[[ ! -e "$WORKDIR/driver.log" ]] || fail "driver ran before explicit spend acknowledgement"

authority_symlink_target="$WORKDIR/preplaced-authority-target.json"
printf '{}\n' >"$authority_symlink_target"
authority_symlink_target_sha="$(sha256sum "$authority_symlink_target" | awk '{print $1}')"
set +e
FIXTURE_AUTHORITY_SYMLINK_TARGET="$authority_symlink_target" \
	python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
	baseline --config "$operator_root/e40-demo.yaml" --run-id authority-symlink \
	--reps 1 --acknowledge-provider-spend --max-cost-usd 1 \
	--max-wall-clock-seconds 30 --max-generated-tasks 1 \
	>/dev/null 2>"$WORKDIR/authority-symlink.err"
authority_symlink_rc=$?
set -e
[[ "$authority_symlink_rc" -eq 4 ]] \
	|| fail "preplaced symlinked batch authority returned $authority_symlink_rc, expected 4"
[[ "$authority_symlink_target_sha" == "$(sha256sum "$authority_symlink_target" | awk '{print $1}')" ]] \
	|| fail "preplaced batch-authority symlink mutated its external target"
authority_symlink_result="$operator_root/runs/authority-symlink/operator-result.json"
if [[ -f "$authority_symlink_result" ]]; then
	python3 - "$authority_symlink_result" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
if result.get("publication_eligible") is True:
    raise SystemExit("symlinked existing authority produced publication_eligible=true")
PY
fi

run_fixture_profile() {
	local command_name="$1" run_id="$2"; shift 2
	python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
		"$command_name" --config "$operator_root/e40-demo.yaml" "$@" \
		--run-id "$run_id" --reps 1 --acknowledge-provider-spend \
		--max-cost-usd 1 --max-wall-clock-seconds 30 --max-generated-tasks 1 \
		>"$WORKDIR/$run_id.out"
}

execution_spy_driver_log="$WORKDIR/execution-spy-driver.log"
SHARK_BIN="$driver_stub" \
	LIFECYCLE_ADAPTER="$driver_stub" \
	LIFECYCLE_ADAPTER_PATH="$driver_stub" \
	LIFECYCLE_PROVIDER_COMMAND="$driver_stub" \
	RUN_LIFECYCLE_BIN="$driver_stub" \
	EVALUATE_LIFECYCLE_BIN="$driver_stub" \
	ENTITY_HISTORY_EXPORT_BIN="$driver_stub" \
	E40_RUN_LIFECYCLE_BATCH_BIN="$driver_stub" \
	E40_AGGREGATE_LIFECYCLE_BIN="$driver_stub" \
	E40_REPORT_LIFECYCLE_BIN="$driver_stub" \
	E40_VERIFY_RETENTION_ROOT_BIN="$driver_stub" \
	FIXTURE_EXPECT_NO_SPY="$driver_stub" \
	FIXTURE_DRIVER_LOG="$execution_spy_driver_log" \
	DRIVER_STUB_LOG="$WORKDIR/execution-override-spy.log" \
	PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	run_fixture_profile baseline execution-spy-scrub
[[ "$(wc -l <"$execution_spy_driver_log")" -eq 1 ]] \
	|| fail "acknowledged execution did not invoke exactly one pinned fixture driver"
[[ ! -e "$WORKDIR/provider.log" ]] \
	|| fail "acknowledged execution invoked an environment-selected executable seam"
[[ ! -e "$WORKDIR/execution-override-spy.log" ]] \
	|| fail "acknowledged execution invoked an environment-selected override spy"

detached_run_root="$WORKDIR/detached-run-root"
set +e
FIXTURE_SWAP_RUN_ROOT_TARGET="$detached_run_root" \
	python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
	baseline --config "$operator_root/e40-demo.yaml" --run-id root-swap \
	--reps 1 --acknowledge-provider-spend --max-cost-usd 1 \
	--max-wall-clock-seconds 30 --max-generated-tasks 1 \
	>/dev/null 2>"$WORKDIR/root-swap.err"
root_swap_rc=$?
set -e
[[ "$root_swap_rc" -eq 2 || "$root_swap_rc" -eq 4 ]] \
	|| fail "post-lock run-root replacement returned $root_swap_rc, expected bounded refusal"
replacement_run_root="$operator_root/runs/root-swap"
[[ -d "$replacement_run_root" ]] \
	|| fail "post-lock run-root replacement fixture did not install a real directory"
[[ -z "$(find "$replacement_run_root" -mindepth 1 -print -quit)" ]] \
	|| fail "post-lock run-root replacement received operator or driver writes"
[[ -f "$detached_run_root/batch.json" ]] \
	|| fail "descriptor-bound fixture driver did not retain evidence in the locked root"

run_fixture_profile pilot baseline-manifest
run_fixture_profile baseline baseline-manifest
run_fixture_profile pilot prompt-manifest --variant-name prompt-v2
run_fixture_profile variant prompt-manifest --variant-name prompt-v2
run_fixture_profile pilot policy-manifest --variant-name policy-v2
run_fixture_profile variant policy-manifest --variant-name policy-v2
run_fixture_profile pilot model-manifest --variant-name model-v2
run_fixture_profile variant model-manifest --variant-name model-v2

assert_resume_refused_without_driver() {
	local case_name="$1"
	local driver_log="$WORKDIR/resume-$case_name-driver.log"
	set +e
	FIXTURE_DRIVER_LOG="$driver_log" \
		python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
		baseline --config "$operator_root/e40-demo.yaml" \
		--run-id baseline-manifest --reps 1 --acknowledge-provider-spend \
		--max-cost-usd 1 --max-wall-clock-seconds 30 --max-generated-tasks 1 \
		>/dev/null 2>"$WORKDIR/resume-$case_name.err"
	local resume_rc=$?
	set -e
	[[ "$resume_rc" -eq 4 ]] \
		|| fail "$case_name resume returned $resume_rc, expected immutable refusal 4"
	[[ ! -e "$driver_log" ]] \
		|| fail "$case_name resume invoked the provider batch driver"
}

adapter_backup="$WORKDIR/provider-spy.original"
cp "$provider_spy" "$adapter_backup"
printf '\n# changed adapter bytes for immutable-resume rejection\n' >>"$provider_spy"
assert_resume_refused_without_driver changed-adapter-content
cp "$adapter_backup" "$provider_spy"
chmod +x "$provider_spy"

for i05_bundle in "$WORKDIR/i05"/*; do
	i05_scenario="$(basename "$i05_bundle")"
	i05_mutation="$i05_bundle/resume-mutation.txt"
	printf 'changed retained evidence input\n' >"$i05_mutation"
	assert_resume_refused_without_driver "changed-i05-$i05_scenario"
	unlink "$i05_mutation"
done

config_backup="$WORKDIR/e40-demo.original.yaml"
cp "$operator_root/e40-demo.yaml" "$config_backup"
python3 - "$operator_root/e40-demo.yaml" "$provider_spy" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["runtime"]["provider_command"] = [sys.argv[2], "changed-provider-route"]
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
assert_resume_refused_without_driver changed-provider-command
cp "$config_backup" "$operator_root/e40-demo.yaml"

resume_manifest="$operator_root/runs/baseline-manifest/benchmark-manifest.json"
resume_manifest_saved="$WORKDIR/resume-benchmark-manifest.json"
mv "$resume_manifest" "$resume_manifest_saved"
ln -s "$resume_manifest_saved" "$resume_manifest"
assert_resume_refused_without_driver symlinked-retained-manifest
unlink "$resume_manifest"
mv "$resume_manifest_saved" "$resume_manifest"

attempt_symlink_root="$operator_root/runs/attempt-symlink"
attempt_symlink_target="$WORKDIR/attempt-symlink-target"
mkdir -p "$attempt_symlink_root" "$attempt_symlink_target"
ln -s "$attempt_symlink_target" "$attempt_symlink_root/operator-attempts"
set +e
python3 "$fixture_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
	baseline --config "$operator_root/e40-demo.yaml" \
	--run-id attempt-symlink --reps 1 --acknowledge-provider-spend \
	--max-cost-usd 1 --max-wall-clock-seconds 30 --max-generated-tasks 1 \
	>/dev/null 2>"$WORKDIR/attempt-symlink.err"
attempt_symlink_rc=$?
set -e
[[ "$attempt_symlink_rc" -eq 2 ]] \
	|| fail "symlinked operator-attempts directory returned $attempt_symlink_rc, expected 2"
[[ -z "$(find "$attempt_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked operator-attempts directory redirected a retained write"

for run_id in baseline-manifest prompt-manifest policy-manifest model-manifest; do
	[[ -f "$operator_root/runs/$run_id/pilot-benchmark-manifest.json" ]] \
		|| fail "$run_id did not retain a separate pilot manifest"
	[[ -f "$operator_root/runs/$run_id/benchmark-manifest.json" ]] \
		|| fail "$run_id did not retain its baseline manifest"
	[[ "$(find "$operator_root/runs/$run_id/batch-authorities" -maxdepth 1 -type f -name '*.json' | wc -l)" -eq 2 ]] \
		|| fail "$run_id did not retain separate pilot and baseline batch authorities"
done

python3 - \
	"$operator_root/runs/baseline-manifest/benchmark-manifest.json" \
	"$operator_root/runs/prompt-manifest/benchmark-manifest.json" \
	"$operator_root/runs/policy-manifest/benchmark-manifest.json" \
	"$operator_root/runs/model-manifest/benchmark-manifest.json" <<'PY'
import json
import sys

baseline, prompt, policy, model = [json.load(open(path, encoding="utf-8")) for path in sys.argv[1:]]
if baseline["identity"]["prompt_bundle_digest"] == prompt["identity"]["prompt_bundle_digest"]:
    raise SystemExit("prompt variant manifest silently retained the baseline prompt digest")
if prompt["comparison_boundary"]["allowed_change_axes"] != ["prompt"]:
    raise SystemExit("prompt variant did not declare exactly the prompt axis")
if baseline["identity"]["policy_bundle_digest"] == policy["identity"]["policy_bundle_digest"]:
    raise SystemExit("policy variant manifest silently retained the baseline policy digest")
if policy["comparison_boundary"]["allowed_change_axes"] != ["policy"]:
    raise SystemExit("policy variant did not declare exactly the policy axis")
if baseline["identity"]["provider_routing"]["model_digest"] == model["identity"]["provider_routing"]["model_digest"]:
    raise SystemExit("model variant manifest did not derive a changed installed model route")
if not {"workflow", "model"}.issubset(model["comparison_boundary"]["allowed_change_axes"]):
    raise SystemExit("model route variant did not declare workflow and model axes")
for variant in (prompt, policy, model):
    if baseline["scenario_matrix"] != variant["scenario_matrix"]:
        raise SystemExit("paired manifest scenario matrices differ")
    if baseline["resource_policy"] != variant["resource_policy"]:
        raise SystemExit("paired manifest resource policies differ")
PY

cat >>"$prompt_override/operator.md" <<'EOF'
This is a different immutable source request.
EOF
set +e
"$OPERATOR" validate-variant --config "$operator_root/e40-demo.yaml" \
	--variant-name prompt-v2 --prompt-root "$prompt_override" >/dev/null 2>"$WORKDIR/revalidate.err"
revalidate_rc=$?
set -e
[[ "$revalidate_rc" -eq 4 ]] || fail "changed source reused a validated variant name (rc=$revalidate_rc)"

# Build deterministic aggregate fixtures for comparison-only behavior. These
# use real retained roots and the pinned F10 aggregator so positive publication
# cases exercise the same evidence-binding boundary as operator runs.
retention_fixture_gen="$SCRIPT_DIR/testdata/gen-100mb-retention-fixture.sh"
"$retention_fixture_gen" --seed 94 --out "$WORKDIR/retention-template" --total-bytes 0
python3 - "$WORKDIR/comparison-fixtures" "$WORKDIR/retention-template" \
	"$SCRIPTS_DIR/aggregate-lifecycle.sh" <<'PY'
import copy
import hashlib
import json
import pathlib
import shutil
import subprocess
import sys

import yaml

root = pathlib.Path(sys.argv[1])
template = pathlib.Path(sys.argv[2])
aggregator = pathlib.Path(sys.argv[3])
scenario_id = "scale-fixture-scenario"
scenario_family = "family-scale-fixture"

def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()

def token(label):
    return hashlib.sha256(label.encode()).hexdigest()

def identity():
    candidate = {
        "base_commit": "a" * 40,
        "tree_digest": token("tree"),
        "binary_diff_digest": token("binary"),
        "changed_path_digest": token("paths"),
        "dirty_untracked_manifest": token("dirty"),
        "working_tree_diff_digest": token("working"),
        "untracked_content_digest": token("untracked"),
        "test_suite_digest": token("tests"),
    }
    candidate["identity_digest"] = digest(candidate)
    execution_inputs = {
        "lifecycle_adapter": {
            "path": "/fixture/lifecycle-adapter",
            "digest": token("lifecycle-adapter"),
        },
        "provider_command_digest": token("provider-command"),
        "i05_bundles": [{
            "scenario_id": scenario_id,
            "path": "/fixture/i05/scale-fixture-scenario",
            "digest": token("i05-bundle"),
        }],
    }
    execution_inputs["i05_bundle_set_digest"] = digest(execution_inputs["i05_bundles"])
    execution_inputs["execution_input_digest"] = digest(execution_inputs)
    value = {
        "scenario_set_digest": token("scenario-set"),
        "fixture_set_digest": token("fixture-set"),
        "adapter_set_digest": token("adapter-set"),
        "shark_binary": {"path": "/fixture/shark", "sha256": token("shark"), "version": "fixture"},
        "content_bundle_digest": token("content"),
        "prompt_bundle_digest": token("prompt"),
        "workflow_bundle_digest": token("workflow"),
        "policy_bundle_digest": token("policy"),
        "enabled_gate_identity": {"source": "workflow_bundle", "digest": token("workflow")},
        "provider_routing": {
            "routes": [{"source": "bug.yaml", "pointer": "/research", "provider": "fixture", "model": "fixture-model", "effort": "fixture-effort"}],
            "provider_digest": token("provider-routes"),
            "model_digest": token("model-routes"),
            "effort_digest": token("effort-routes"),
            "routing_digest": token("routing"),
        },
        "execution_inputs": execution_inputs,
        "resource_policy_digest": token("resource"),
        "candidate": candidate,
    }
    value["identity_digest"] = digest(value)
    return value

def refresh_identity(value):
    value["candidate"]["identity_digest"] = digest({k: v for k, v in value["candidate"].items() if k != "identity_digest"})
    value["provider_routing"]["routing_digest"] = digest({k: v for k, v in value["provider_routing"].items() if k != "routing_digest"})
    execution_inputs = value["execution_inputs"]
    execution_inputs["i05_bundle_set_digest"] = digest(execution_inputs["i05_bundles"])
    execution_inputs["execution_input_digest"] = digest({k: v for k, v in execution_inputs.items() if k != "execution_input_digest"})
    value["identity_digest"] = digest({k: v for k, v in value.items() if k != "identity_digest"})

def manifest(path, role, allowed=(), mutation=None):
    ident = identity()
    matrix = [{
        "scenario_id": scenario_id, "scenario_version": "1", "family": scenario_family,
        "package_digest": token("package"), "fixture_id": "fixture-a",
        "fixture_base_sha": "b" * 40, "adapter_id": "fixture-adapter",
        "adapter_version": "1", "resource_policy_digest": token("scenario-policy"),
        "reps": 2,
    }]
    if mutation == "prompt":
        ident["prompt_bundle_digest"] = token("prompt-v2")
        ident["content_bundle_digest"] = token("content-v2")
    elif mutation == "model":
        ident["provider_routing"]["routes"][0]["model"] = "fixture-model-v2"
        ident["provider_routing"]["model_digest"] = token("model-routes-v2")
        ident["workflow_bundle_digest"] = token("workflow-v2")
        ident["content_bundle_digest"] = token("content-v2")
        ident["enabled_gate_identity"]["digest"] = token("workflow-v2")
    elif mutation == "workflow":
        ident["workflow_bundle_digest"] = token("workflow-v2")
        ident["content_bundle_digest"] = token("content-v2")
        ident["enabled_gate_identity"]["digest"] = token("workflow-v2")
    elif mutation == "fixture":
        ident["fixture_set_digest"] = token("fixture-set-v2")
        matrix[0]["fixture_base_sha"] = "c" * 40
    elif mutation == "candidate":
        ident["candidate"]["working_tree_diff_digest"] = token("working-v2")
    elif mutation == "policy":
        ident["policy_bundle_digest"] = token("policy-v2")
    refresh_identity(ident)
    config_digest = token("baseline-config")
    value = {
        "schema_version": "1.0", "phase": "lifecycle_v2", "role": role,
        "run_mode": "baseline", "run_id": path.name, "created_at": "2026-08-23T00:00:00Z",
        "profile": path.name, "identity": ident, "scenario_matrix": matrix,
        "resource_policy": {"max_cost_usd": 100.0, "max_wall_clock_seconds": 3600.0, "max_generated_tasks": 20, "repetitions": 2},
        "comparison_boundary": {
            "allowed_change_axes": list(allowed),
            "baseline_definition_digest": config_digest if role == "variant" else None,
        },
        "retention_root": str(path), "command_arguments": ["fixture"],
        "config": "/fixture/e40.yaml", "config_digest": config_digest,
    }
    value["manifest_digest"] = digest(value)
    return value

def aggregate(*, elapsed=20.0, cost=2.0, failed=False, incomplete=False, insufficient=False, bad_time=False):
    return {
        "elapsed": elapsed, "cost": cost, "failed": failed, "incomplete": incomplete,
        "insufficient": insufficient, "bad_time": bad_time,
    }


def aggregate_retention_root(path):
    completed = subprocess.run(
        [str(aggregator), "--retention-root", str(path)],
        text=True, capture_output=True, check=False,
    )
    if completed.returncode != 0:
        raise SystemExit(f"fixture aggregation failed for {path.name}: {completed.stderr}")
    return json.loads(completed.stdout)


def prepare_retention_root(path):
    shutil.copytree(template, path)
    scenario_root = path / "scenarios" / scenario_id
    shutil.copytree(scenario_root / "1", scenario_root / "2")
    for rep in (1, 2):
        pair = scenario_root / str(rep)
        lifecycle_path = pair / "lifecycle.jsonl"
        lifecycle = json.loads(lifecycle_path.read_text(encoding="utf-8"))
        lifecycle["review_gates"] = [{
            "gate_id": "code_review", "state": "findings", "round": 1,
            "candidate_ref": f"candidate-{rep}", "policy_ref": "policy-fixture",
            "findings": [{}],
        }]
        lifecycle_path.write_text(
            json.dumps(lifecycle, sort_keys=True, separators=(",", ":")) + "\n",
            encoding="utf-8",
        )
        evaluation_path = pair / "evaluation.jsonl"
        evaluation = json.loads(evaluation_path.read_text(encoding="utf-8"))
        evaluation["execution_oracle"] = {"observed_result": "pass"}
        evaluation["metrics"]["quality"] = {
            "confirmed_findings": {"value": 1, "available": True},
            "unconfirmed_findings": {"value": 0, "available": True},
            "precision": {"value": None, "available": False},
            "recall": {"value": None, "available": False},
            "truth_set_status": "unavailable",
            "review_measures": [{
                "gate": "code_review", "severity": "medium", "defect_class": "fixture",
                "measures": {
                    "emitted": {"value": 1, "available": True},
                    "normalized_unique": {"value": 1, "available": True},
                    "duplicate": {"value": 0, "available": True},
                    "recurrent": {"value": 0, "available": True},
                    "confirmed": {"value": 1, "available": True},
                    "unconfirmed": {"value": 0, "available": True},
                    "downstream_escape": {"value": 0, "available": True},
                },
            }],
        }
        evaluation_path.write_text(
            json.dumps(evaluation, sort_keys=True, separators=(",", ":")) + "\n",
            encoding="utf-8",
        )
        artifacts = {}
        for name in ("package.yaml", "lifecycle.jsonl", "evaluation.jsonl"):
            source = pair / name
            artifacts[name] = {"source_path": str(source), "sha256": hashlib.sha256(source.read_bytes()).hexdigest()}
        retained_manifest = {"scenario_id": scenario_id, "rep": rep, "artifacts": artifacts}
        (pair / "manifest.json").write_text(
            json.dumps(retained_manifest, sort_keys=True, separators=(",", ":")) + "\n",
            encoding="utf-8",
        )
    policy = {
        "schema_version": "1.0", "min_reps": 2, "scenario_index": "/fixture/scenarios.yaml",
        "scenarios": {scenario_id: {"reps": 2, "root_key": "F-fixture", "scratch_root": "/fixture/scratch"}},
    }
    policy_bytes = yaml.safe_dump(policy, sort_keys=False).encode()
    (path / "batch-policy.yaml").write_bytes(policy_bytes)
    batch = json.loads((path / "batch.json").read_text(encoding="utf-8"))
    batch.update({
        "batch_id": f"batch-{path.name}", "mode": "baseline", "min_reps": 2,
        "retention_root": str(path), "batch_policy_digest": hashlib.sha256(policy_bytes).hexdigest(),
        "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
        "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
    })
    (path / "batch.json").write_text(
        json.dumps(batch, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8"
    )
    return aggregate_retention_root(path)


def write_batch_authority(path, manifest_value):
    attempts = path / "operator-attempts"
    attempts.mkdir()
    attempt_path = attempts / "0001.json"
    attempt = {
        "started_at": "2026-08-23T00:00:00Z", "completed_at": "2026-08-23T00:00:01Z",
        "command": [str(aggregator), "--acknowledge-provider-spend"],
        "driver_exit_code": 0, "stdout": "", "stderr": "",
    }
    attempt_path.write_text(
        json.dumps(attempt, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    batch = json.loads((path / "batch.json").read_text(encoding="utf-8"))
    authority = {
        "phase": batch["phase"], "batch_id": batch["batch_id"], "mode": batch["mode"],
        "retention_root": str(path.resolve()), "batch_policy_digest": batch["batch_policy_digest"],
        "ceilings": {"max_cost_usd": 100, "max_wall_clock_seconds": 3600, "max_generated_tasks": 20},
        "acknowledgement_ref": batch["acknowledgement_ref"], "min_reps": batch["min_reps"],
    }
    record = {
        "schema_version": "1.0", "manifest_digest": manifest_value["manifest_digest"],
        "attempt_path": "operator-attempts/0001.json",
        "attempt_digest": hashlib.sha256(attempt_path.read_bytes()).hexdigest(),
        "batch_authority": authority,
    }
    record["authority_digest"] = digest(record)
    authority_dir = path / "batch-authorities"
    authority_dir.mkdir()
    (authority_dir / f"{batch['batch_id']}.json").write_text(
        json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )


def write_case(name, role, aggregate_options, allowed=(), mutation=None, batch_tamper=None):
    path = root / name
    aggregate_value = prepare_retention_root(path)
    manifest_value = manifest(path, role, allowed, mutation)
    (path / "benchmark-manifest.json").write_text(
        json.dumps(manifest_value, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    write_batch_authority(path, manifest_value)
    if batch_tamper:
        batch = json.loads((path / "batch.json").read_text(encoding="utf-8"))
        if batch_tamper == "batch_id":
            batch["batch_id"] = f"batch-{path.name}-changed"
        elif batch_tamper == "batch_policy_digest":
            batch["batch_policy_digest"] = "0" * 64
        elif batch_tamper == "acknowledgement_ref":
            batch["acknowledgement_ref"]["present"] = False
        elif batch_tamper == "mode":
            batch["mode"] = "pilot"
        elif batch_tamper == "retention_root":
            batch["retention_root"] = "/tmp/wrong-e40-retention-root"
        elif batch_tamper == "retention_root_symlink_alias":
            alias = root / f"{path.name}-retention-alias"
            alias.symlink_to(path, target_is_directory=True)
            batch["retention_root"] = str(alias)
        elif batch_tamper == "ceilings":
            batch["ceilings"]["max_cost_usd"] = "101"
        elif batch_tamper == "min_reps":
            batch["min_reps"] = 3
        else:
            raise SystemExit(f"unknown batch tamper: {batch_tamper}")
        (path / "batch.json").write_text(
            json.dumps(batch, sort_keys=True, separators=(",", ":")) + "\n",
            encoding="utf-8",
        )
        if batch_tamper == "retention_root_symlink_alias":
            authority_path = path / "batch-authorities" / f"{batch['batch_id']}.json"
            authority = json.loads(authority_path.read_text(encoding="utf-8"))
            authority["batch_authority"]["retention_root"] = batch["retention_root"]
            authority["authority_digest"] = digest({
                key: value for key, value in authority.items()
                if key != "authority_digest"
            })
            authority_path.write_text(
                json.dumps(authority, indent=2, sort_keys=True) + "\n",
                encoding="utf-8",
            )
        aggregate_value = aggregate_retention_root(path)
    if aggregate_options["failed"]:
        aggregate_value["quality"]["by_scenario"][1]["execution_oracle"] = {"observed_result": "fail"}
        aggregate_value["time"]["lifecycle_wall_seconds"] *= 0.5
        for partition in ("stage_category", "interval_category", "share_partition"):
            aggregate_value["time"][partition] = {
                key: value * 0.5 for key, value in aggregate_value["time"][partition].items()
            }
            aggregate_value["cost"][partition] = {
                key: value * 0.5 for key, value in aggregate_value["cost"][partition].items()
            }
        aggregate_value["cost"]["ceiling_consumption"]["observed_cost_usd"] *= 0.5
    if aggregate_options["incomplete"]:
        eligibility = {
            "aggregate_eligible": False, "publication_eligible": False,
            "invalidity_reasons": [{"code": "timeout", "path": "/outcome", "detail": "fixture timeout"}],
        }
        aggregate_value["scenarios"][1]["eligibility"] = eligibility
        aggregate_value["invalid"].append({
            "scenario_id": scenario_id, "rep": 2, "source": "i08_eligibility",
            "retention_path": f"scenarios/{scenario_id}/2", "eligibility": eligibility,
        })
    if aggregate_options["insufficient"]:
        for band in aggregate_value["noise_bands"]:
            band["insufficient_reps"] = True
            band["rep_count"] = 1
    if aggregate_options["bad_time"]:
        aggregate_value["time"]["stage_category"]["code"] -= 1.0
    (path / "aggregate.json").write_text(
        json.dumps(aggregate_value, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )

normal = aggregate()
write_case("baseline", "baseline", normal)
write_case("control", "variant", normal)
write_case("allowed-prompt", "variant", normal, ("prompt",), "prompt")
write_case("allowed-model", "variant", normal, ("workflow", "model"), "model")
for axis in ("prompt", "model", "workflow", "fixture", "candidate", "policy"):
    write_case(f"undeclared-{axis}", "variant", normal, (), axis)
write_case("failed-fast", "variant", aggregate(elapsed=10.0, cost=1.0, failed=True))
write_case("incomplete", "variant", aggregate(incomplete=True))
write_case("insufficient", "variant", aggregate(insufficient=True))
write_case("bad-time", "variant", aggregate(bad_time=True))
for authority_field in (
    "batch_id", "batch_policy_digest", "acknowledgement_ref", "mode",
    "retention_root", "ceilings", "min_reps",
):
    write_case(
        f"tampered-{authority_field}", "variant", normal,
        batch_tamper=authority_field,
    )
write_case(
    "tampered-retention-root-symlink-alias", "variant", normal,
    batch_tamper="retention_root_symlink_alias",
)
PY

fixtures="$WORKDIR/comparison-fixtures"
report_symlink_target="$WORKDIR/report-symlink-target"
mkdir -p "$report_symlink_target"
ln -s "$report_symlink_target" "$fixtures/control/reports"
I05_SCHEMA="$WORKDIR/hostile-i05-schema.yaml" \
	I07_SCHEMA="$WORKDIR/hostile-i07-schema.yaml" \
	E40_AGGREGATE_LIFECYCLE_BIN="$provider_spy" \
	PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	python3 - "$SCRIPTS_DIR/lib/e40_benchmark.py" "$fixtures/control" <<'PY'
import importlib.util
import pathlib
import sys

spec = importlib.util.spec_from_file_location("e40_benchmark", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
run_root = pathlib.Path(sys.argv[2])
run_descriptor = module.open_real_directory_fd(
    run_root, "operator run root", create=False
)
try:
    result = module.aggregate_and_report(run_descriptor, run_root)
except module.OperatorError as exc:
    if "not a real child directory" not in str(exc):
        raise SystemExit(f"unexpected report-symlink refusal: {exc}")
else:
    raise SystemExit(f"symlinked reports directory was accepted: {result!r}")
finally:
    import os

    os.close(run_descriptor)
PY
[[ -z "$(find "$report_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked reports directory redirected a derived write"
[[ ! -e "$WORKDIR/provider.log" ]] \
	|| fail "derived aggregation invoked an environment-selected helper"
unlink "$fixtures/control/reports"

comparison_symlink_target="$WORKDIR/comparison-symlink-target"
comparison_symlink_parent="$WORKDIR/comparison-symlink-parent"
mkdir -p "$comparison_symlink_target"
ln -s "$comparison_symlink_target" "$comparison_symlink_parent"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$comparison_symlink_parent/control" \
	>/dev/null 2>"$WORKDIR/comparison-symlink.err"
comparison_symlink_rc=$?
set -e
[[ "$comparison_symlink_rc" -eq 2 ]] \
	|| fail "symlinked comparison parent returned $comparison_symlink_rc, expected 2"
[[ -z "$(find "$comparison_symlink_target" -mindepth 1 -print -quit)" ]] \
	|| fail "symlinked comparison parent redirected publication writes"

run_reference_symlink="$WORKDIR/control-run-symlink"
ln -s "$fixtures/control" "$run_reference_symlink"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$run_reference_symlink" \
	--out "$WORKDIR/comparisons/symlinked-run-reference" \
	>/dev/null 2>"$WORKDIR/run-reference-symlink.err"
run_reference_symlink_rc=$?
set -e
[[ "$run_reference_symlink_rc" -eq 2 ]] \
	|| fail "symlinked run reference returned $run_reference_symlink_rc, expected 2"
[[ ! -e "$WORKDIR/comparisons/symlinked-run-reference" ]] \
	|| fail "symlinked run reference produced comparison output"

assert_comparison_manifest_symlink_refused() {
	local side="$1"
	local manifest="$fixtures/$side/benchmark-manifest.json"
	local saved="$WORKDIR/$side-benchmark-manifest.json"
	local output="$WORKDIR/comparisons/symlinked-$side-manifest"
	mv "$manifest" "$saved"
	ln -s "$saved" "$manifest"
	set +e
	"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
		--out "$output" >/dev/null 2>"$WORKDIR/symlinked-$side-manifest.err"
	local manifest_rc=$?
	set -e
	unlink "$manifest"
	mv "$saved" "$manifest"
	[[ "$manifest_rc" -eq 5 ]] \
		|| fail "symlinked $side manifest returned $manifest_rc, expected 5"
	[[ ! -e "$output" ]] \
		|| fail "symlinked $side manifest produced comparison output"
}

assert_comparison_manifest_symlink_refused baseline
assert_comparison_manifest_symlink_refused control

attempt_parent="$fixtures/control/operator-attempts"
attempt_parent_real="$fixtures/control/operator-attempts-real"
mv "$attempt_parent" "$attempt_parent_real"
ln -s "operator-attempts-real" "$attempt_parent"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/symlinked-attempt-parent" \
	>"$WORKDIR/symlinked-attempt-parent.out" 2>&1
attempt_parent_rc=$?
set -e
unlink "$attempt_parent"
mv "$attempt_parent_real" "$attempt_parent"
[[ "$attempt_parent_rc" -eq 5 ]] \
	|| fail "symlinked retained attempt parent returned $attempt_parent_rc, expected 5"
grep -q 'invalid_batch_authority_attempt' "$WORKDIR/symlinked-attempt-parent.out" \
	|| fail "symlinked retained attempt parent did not name its authority failure"

atomic_race_parent="$WORKDIR/atomic-race-parent"
atomic_race_moved="$WORKDIR/atomic-race-moved"
atomic_race_target="$WORKDIR/atomic-race-target"
mkdir -p "$atomic_race_parent" "$atomic_race_target"
python3 - "$SCRIPTS_DIR/lib/e40_benchmark.py" "$atomic_race_parent" \
	"$atomic_race_moved" "$atomic_race_target" <<'PY'
import importlib.util
import os
import pathlib
import sys

module_path, parent_raw, moved_raw, target_raw = sys.argv[1:]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
parent = pathlib.Path(parent_raw)
moved = pathlib.Path(moved_raw)
target = pathlib.Path(target_raw)
real_open = module.open_real_directory_fd
swapped = False


def swapping_open(path, label, *, create):
    global swapped
    descriptor = real_open(path, label, create=create)
    if not swapped and pathlib.Path(path) == parent:
        os.rename(parent, moved)
        parent.symlink_to(target, target_is_directory=True)
        swapped = True
    return descriptor


module.open_real_directory_fd = swapping_open
try:
    module.atomic_write(parent / "artifact.json", b"{}\n", overwrite=False)
except module.OperatorError as exc:
    if "changed after validation" not in str(exc):
        raise SystemExit(f"unexpected atomic parent-swap refusal: {exc}")
else:
    raise SystemExit("atomic write accepted a parent swapped to a symlink")
PY
[[ -z "$(find "$atomic_race_target" -mindepth 1 -print -quit)" ]] \
	|| fail "atomic parent-swap race redirected a write to the external target"
[[ -z "$(find "$atomic_race_moved" -mindepth 1 -print -quit)" ]] \
	|| fail "atomic parent-swap refusal left a partial artifact behind"

if ! "$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/control" >"$WORKDIR/control.out"; then
	fail "retention-bound control comparison failed: $(cat "$WORKDIR/control.out")"
fi
control_sha="$(sha256sum "$WORKDIR/comparisons/control/comparison.json" | awk '{print $1}')"
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/control" >"$WORKDIR/control-rerun.out"
[[ "$control_sha" == "$(sha256sum "$WORKDIR/comparisons/control/comparison.json" | awk '{print $1}')" ]] \
	|| fail "completed comparison was overwritten on rerun"

python3 - "$SCRIPTS_DIR/lib/e40_benchmark.py" "$fixtures/baseline" \
	"$fixtures/control" "$WORKDIR/comparisons/control" "$WORKDIR/comparisons" <<'PY'
import argparse
import hashlib
import importlib.util
import json
import os
import pathlib
import sys

module_path, baseline_raw, variant_raw, cache_raw, outputs_raw = sys.argv[1:]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
baseline = pathlib.Path(baseline_raw)
variant = pathlib.Path(variant_raw)
cache = pathlib.Path(cache_raw)
outputs = pathlib.Path(outputs_raw)
batch = json.loads((baseline / "batch.json").read_text(encoding="utf-8"))
authority = baseline / "batch-authorities" / f"{batch['batch_id']}.json"


def tree_digest(root):
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        digest.update(path.relative_to(root).as_posix().encode())
        if path.is_file():
            digest.update(path.read_bytes())
    return digest.hexdigest()


cases = (
    ("manifest", baseline / "benchmark-manifest.json", "baseline manifest", None),
    ("batch", baseline / "batch.json", "baseline retained batch identity", None),
    ("authority", authority, "baseline batch authority", None),
    ("aggregate", baseline / "aggregate.json", "baseline aggregate", None),
    ("cache-json", cache / "comparison.json", "existing comparison", cache),
    ("cache-report", cache / "comparison.md", "existing comparison report", cache),
    (
        "cache-completion",
        cache / "comparison.complete.json",
        "comparison completion marker",
        cache,
    ),
)
for name, target, read_label, existing_output in cases:
    saved = target.with_name(f"{target.name}.race-original")
    output = existing_output or outputs / f"authority-read-race-{name}"
    before = tree_digest(output) if existing_output else None
    real_open = module.open_real_directory_fd
    swapped = False

    def swapping_open(path, open_label, *, create):
        global swapped
        descriptor = real_open(path, open_label, create=create)
        if not swapped and open_label == f"parent directory for {read_label}":
            os.rename(target, saved)
            target.symlink_to(saved.name)
            swapped = True
        return descriptor

    module.open_real_directory_fd = swapping_open
    try:
        args = argparse.Namespace(
            baseline=str(baseline),
            variant=str(variant),
            out=str(output),
            config=None,
        )
        try:
            status = module.cmd_compare(args)
        except module.OperatorError as exc:
            status = exc.exit_code
    finally:
        module.open_real_directory_fd = real_open
        if target.is_symlink():
            target.unlink()
        if saved.exists():
            os.rename(saved, target)
    if not swapped:
        raise SystemExit(f"{name}: deterministic final-file swap did not trigger")
    if status != 5:
        raise SystemExit(f"{name}: final-file symlink race returned {status}, expected 5")
    if existing_output:
        if tree_digest(output) != before:
            raise SystemExit(f"{name}: refused cache race changed publication bytes")
    elif output.exists():
        raise SystemExit(f"{name}: authority read race created a publication root")
PY

partial_out="$WORKDIR/comparisons/partial-publication"
mkdir -p "$partial_out"
cp "$WORKDIR/comparisons/control/comparison.json" "$partial_out/comparison.json"
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$partial_out" >"$WORKDIR/partial-publication.out"
grep -q '"status": "recovered"' "$WORKDIR/partial-publication.out" \
	|| fail "JSON-only partial comparison publication was not recovered"
[[ -f "$partial_out/comparison.md" && -f "$partial_out/comparison.complete.json" ]] \
	|| fail "partial comparison recovery did not publish report and completion marker"

I05_SCHEMA="$WORKDIR/hostile-i05-schema.yaml" \
	I07_SCHEMA="$WORKDIR/hostile-i07-schema.yaml" \
	E40_AGGREGATE_LIFECYCLE_BIN="$provider_spy" \
	PROVIDER_SPY_LOG="$WORKDIR/provider.log" \
	"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/hostile-schema-overrides" >/dev/null
[[ ! -e "$WORKDIR/provider.log" ]] \
	|| fail "comparison invoked an environment-selected aggregation helper"

cp "$fixtures/control/aggregate.json" "$WORKDIR/control-aggregate.json"
cp "$fixtures/baseline/aggregate.json" "$fixtures/control/aggregate.json"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/swapped-aggregate" >"$WORKDIR/swapped-aggregate.out" 2>&1
swapped_aggregate_rc=$?
set -e
cp "$WORKDIR/control-aggregate.json" "$fixtures/control/aggregate.json"
[[ "$swapped_aggregate_rc" -eq 5 ]] \
	|| fail "aggregate swapped from another retention root returned $swapped_aggregate_rc, expected 5"
grep -q 'aggregate_detached_from_retention_root' "$WORKDIR/swapped-aggregate.out" \
	|| fail "swapped aggregate did not name its retention-root binding failure"

"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/allowed-prompt" \
	--out "$WORKDIR/comparisons/allowed-prompt" >/dev/null
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/allowed-model" \
	--out "$WORKDIR/comparisons/allowed-model" >/dev/null

for axis in prompt model workflow fixture candidate policy; do
	set +e
	"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/undeclared-$axis" \
		--out "$WORKDIR/comparisons/undeclared-$axis" >/dev/null 2>&1
	rc=$?
	set -e
	[[ "$rc" -eq 5 ]] || fail "undeclared $axis identity change returned $rc, expected 5"
done

for authority_field in batch_id batch_policy_digest acknowledgement_ref mode retention_root ceilings min_reps; do
	set +e
	"$OPERATOR" compare --baseline "$fixtures/baseline" \
		--variant "$fixtures/tampered-$authority_field" \
		--out "$WORKDIR/comparisons/tampered-$authority_field" >/dev/null 2>&1
	rc=$?
	set -e
	[[ "$rc" -eq 5 ]] \
		|| fail "tampered batch authority field $authority_field returned $rc, expected 5"
done

set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" \
	--variant "$fixtures/tampered-retention-root-symlink-alias" \
	--out "$WORKDIR/comparisons/tampered-retention-root-symlink-alias" \
	>"$WORKDIR/tampered-retention-root-symlink-alias.out" 2>&1
retention_alias_rc=$?
set -e
[[ "$retention_alias_rc" -eq 5 ]] \
	|| fail "self-digested retention-root symlink alias returned $retention_alias_rc, expected 5"
grep -q 'batch_retention_root_not_real' \
	"$WORKDIR/tampered-retention-root-symlink-alias.out" \
	|| fail "retention-root symlink alias did not name its real-directory failure"

for case_name in failed-fast incomplete insufficient bad-time; do
	set +e
	"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/$case_name" \
		--out "$WORKDIR/comparisons/$case_name" >/dev/null 2>&1
	rc=$?
	set -e
	[[ "$rc" -eq 5 ]] || fail "$case_name comparison returned $rc, expected 5"
done

cp "$WORKDIR/comparisons/failed-fast/comparison.json" "$WORKDIR/failed-fast-comparison.json"
python3 - "$WORKDIR/comparisons/failed-fast/comparison.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
document = json.loads(path.read_text(encoding="utf-8"))
document["publication_eligible"] = True
path.write_text(json.dumps(document, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/failed-fast" \
	--out "$WORKDIR/comparisons/failed-fast" >"$WORKDIR/tampered-cache.out" 2>&1
tampered_cache_rc=$?
set -e
[[ "$tampered_cache_rc" -eq 5 ]] \
	|| fail "hand-authored cached comparison returned $tampered_cache_rc, expected 5"
grep -q 'cached_comparison_integrity_mismatch' "$WORKDIR/tampered-cache.out" \
	|| fail "hand-authored cached comparison did not name its integrity failure"
cp "$WORKDIR/failed-fast-comparison.json" "$WORKDIR/comparisons/failed-fast/comparison.json"

mkdir -p "$WORKDIR/comparisons/symlink-cache"
ln -s "$WORKDIR/comparisons/control/comparison.json" \
	"$WORKDIR/comparisons/symlink-cache/comparison.json"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$WORKDIR/comparisons/symlink-cache" >/dev/null 2>&1
symlink_cache_rc=$?
set -e
[[ "$symlink_cache_rc" -eq 5 ]] \
	|| fail "symlinked cached comparison returned $symlink_cache_rc, expected 5"

lock_harness="$WORKDIR/comparison-lock-harness.py"
cat >"$lock_harness" <<'PY'
import argparse
import importlib.util
import pathlib
import sys
import time

module_path, baseline, variant, output, signal, release = sys.argv[1:]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
real_atomic_write = module.atomic_write
output_path = pathlib.Path(output)
signal_path = pathlib.Path(signal)
release_path = pathlib.Path(release)


def paused_atomic_write(path, data, *, overwrite=True):
    real_atomic_write(path, data, overwrite=overwrite)
    if path == output_path / "comparison.json":
        signal_path.write_text("json-published-lock-held\n", encoding="utf-8")
        for _ in range(200):
            if release_path.exists():
                return
            time.sleep(0.05)
        raise RuntimeError("timed out waiting to release deterministic comparison race")


module.atomic_write = paused_atomic_write
args = argparse.Namespace(baseline=baseline, variant=variant, out=output, config=None)
raise SystemExit(module.cmd_compare(args))
PY
locked_out="$WORKDIR/comparisons/deterministic-lock"
lock_signal="$WORKDIR/comparison-lock.signal"
lock_release="$WORKDIR/comparison-lock.release"
python3 "$lock_harness" "$SCRIPTS_DIR/lib/e40_benchmark.py" \
	"$fixtures/baseline" "$fixtures/control" "$locked_out" \
	"$lock_signal" "$lock_release" >"$WORKDIR/lock-first.out" 2>&1 &
lock_first_pid=$!
for _ in $(seq 1 200); do
	[[ -f "$lock_signal" ]] && break
	sleep 0.05
done
[[ -f "$lock_signal" ]] || fail "deterministic comparison writer never reached partial publication"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" \
	--out "$locked_out" >"$WORKDIR/lock-second.out" 2>&1
lock_second_rc=$?
set -e
touch "$lock_release"
set +e
wait "$lock_first_pid"
lock_first_rc=$?
set -e
[[ "$lock_first_rc" -eq 0 ]] || fail "locked comparison writer returned $lock_first_rc"
[[ "$lock_second_rc" -eq 4 ]] \
	|| fail "concurrent cache reader bypassed the output lock (rc=$lock_second_rc, expected 4)"
[[ -f "$locked_out/comparison.complete.json" ]] \
	|| fail "locked comparison writer did not publish its completion marker"

concurrent_out="$WORKDIR/comparisons/concurrent"
set +e
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" --out "$concurrent_out" >"$WORKDIR/concurrent-a.out" 2>&1 &
pid_a=$!
"$OPERATOR" compare --baseline "$fixtures/baseline" --variant "$fixtures/control" --out "$concurrent_out" >"$WORKDIR/concurrent-b.out" 2>&1 &
pid_b=$!
wait "$pid_a"; rc_a=$?
wait "$pid_b"; rc_b=$?
set -e
if [[ "$rc_a" -ne 0 && "$rc_b" -ne 0 ]]; then
	fail "both concurrent comparison invocations failed ($rc_a, $rc_b)"
fi
if [[ "$rc_a" -ne 0 && "$rc_a" -ne 4 ]]; then
	fail "unexpected first concurrent exit $rc_a"
fi
if [[ "$rc_b" -ne 0 && "$rc_b" -ne 4 ]]; then
	fail "unexpected second concurrent exit $rc_b"
fi

python3 - "$WORKDIR/comparisons" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])

def load(name):
    return json.loads((root / name / "comparison.json").read_text(encoding="utf-8"))

control = load("control")
if not control["publication_eligible"] or control["dimensions"]["correctness"]["delta"] != 0.0:
    raise SystemExit("identical comparison was not a valid zero-delta control")
if control["dimensions"]["tokens"]["input"]["status"] != "unavailable" or control["dimensions"]["tokens"]["output"]["status"] != "unavailable":
    raise SystemExit("missing token rollups were not reported separately as unavailable")
review = control["dimensions"]["review_findings"]
if review["truth_set_available"] is not False or review["precision"] != "unavailable" or review["recall"] != "unavailable":
    raise SystemExit("missing truth set invented precision or recall")
if not control["retained_raw_evidence"]["baseline"] or not control["retained_raw_evidence"]["variant"]:
    raise SystemExit("comparison did not point to retained raw evidence")

allowed = load("allowed-prompt")
if not allowed["publication_eligible"] or allowed["boundary"]["authorized_change_axes"] != ["prompt"]:
    raise SystemExit("declared prompt-only variant did not pass its fair boundary")
allowed_model = load("allowed-model")
if not allowed_model["publication_eligible"] or allowed_model["boundary"]["authorized_change_axes"] != ["workflow", "model"]:
    raise SystemExit("declared workflow model-route variant did not pass its fair boundary")

failed = load("failed-fast")
if not failed["quality_regression"] or failed["interpretation"] != "variant_not_better_due_to_quality_regression":
    raise SystemExit("faster/cheaper failed-oracle variant hid its quality regression")
if failed["dimensions"]["lifecycle_time"]["delta_seconds"] >= 0 or failed["dimensions"]["provider_cost"]["delta_usd"] >= 0:
    raise SystemExit("failed-fast fixture did not preserve its faster/cheaper dimensions")
if not any(reason["reason"] == "failed_or_missing_execution_oracle" for reason in failed["exclusions"]):
    raise SystemExit("failed oracle reason is not visible")

incomplete = load("incomplete")
reason_names = {reason["reason"] for reason in incomplete["exclusions"]}
if not {"retained_invalid_run", "scenario_ineligible"}.issubset(reason_names):
    raise SystemExit("incomplete/timeout evidence was not retained as an exclusion")
if not any(reason["reason"] == "insufficient_reps" for reason in load("insufficient")["exclusions"]):
    raise SystemExit("insufficient repetitions silently entered the comparison")
if not any(reason["reason"] == "time_reconciliation_failed" for reason in load("bad-time")["exclusions"]):
    raise SystemExit("stage intervals did not reconcile to lifecycle wall time")

for axis in ("prompt", "model", "workflow", "fixture", "candidate", "policy"):
    if load(f"undeclared-{axis}")["publication_eligible"]:
        raise SystemExit(f"undeclared {axis} change was published")

json.loads((root / "concurrent" / "comparison.json").read_text(encoding="utf-8"))
PY

echo "TC-094: top-level E40 operator safety, immutable identity, and fair comparison contract pass"
