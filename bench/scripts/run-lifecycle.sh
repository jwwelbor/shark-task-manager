#!/usr/bin/env bash
# run-lifecycle.sh --scenario <package.yaml> --run-id <id> --root <key>
#                    --scratch-root <dir> [--fixture-root <dir>]
#                    [--evidence-root <dir>] [--output <lifecycle.jsonl>]
#                    [--limits <policy.yaml>] [--replay <i06-result.json>]
#                    [--prelude <prelude.jsonl>]
#                    [--mode contract|dry-run]
#
# Host-side F08 controller. Shark remains the owner of prompt assembly,
# claims, leases, workflow routing, and Question state; this script only
# drives the public keyed command sequence and records bounded evidence.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

command -v python3 >/dev/null 2>&1 || {
	echo "run-lifecycle: python3 not found on PATH" >&2
	exit 2
}

LIFECYCLE_BENCH_DIR="$BENCH_DIR" exec python3 - "$@" <<'PY'
import hashlib
import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

try:
    import yaml
except ImportError as exc:
    print(f"run-lifecycle: PyYAML is required: {exc}", file=sys.stderr)
    raise SystemExit(2)


STOP_OUTCOMES = {
    "resource_limit", "lease_loss", "missing_outcome", "unresolved_gate",
    "pause", "archive", "error", "cancellation", "worker_failure", "timeout",
}


class LeaseLoss(RuntimeError):
    """Raised when the parent cannot renew its returned Shark session."""


class ResourceLimit(RuntimeError):
    """Raised when an in-flight provider exceeds the scenario wall deadline."""


class Cancellation(RuntimeError):
    """Raised by signal handling after the active provider tree is stopped."""


def usage():
    print(
        "usage: run-lifecycle.sh --scenario <package.yaml> --run-id <id> "
        "--root <key> --scratch-root <dir> [--fixture-root <dir>] "
        "[--evidence-root <dir>] [--output <path>] "
        "[--limits <policy.yaml>] [--replay <i06-result.json>] "
        "[--prelude <prelude.jsonl>] [--mode contract|dry-run]",
        file=sys.stderr,
    )
    raise SystemExit(2)


def parse_args(argv):
    values = {"mode": "live", "output": "", "limits": "", "fixture_root": "", "evidence_root": "", "prelude": "", "replay": ""}
    required = {"--scenario": "scenario", "--run-id": "run_id", "--root": "root", "--scratch-root": "scratch_root"}
    index = 0
    while index < len(argv):
        option = argv[index]
        if option in required or option in {"--output", "--limits", "--mode", "--fixture-root", "--evidence-root", "--prelude", "--replay"}:
            if index + 1 >= len(argv):
                usage()
            values[required.get(option, option[2:].replace("-", "_") or "mode")] = argv[index + 1]
            index += 2
            continue
        usage()
    if any(not values.get(name) for name in required.values()):
        usage()
    if values["mode"] not in {"live", "contract", "dry-run"}:
        print(f"run-lifecycle: unsupported mode: {values['mode']}", file=sys.stderr)
        raise SystemExit(2)
    return values


def sha256_bytes(value):
    return hashlib.sha256(value).hexdigest()


def sha256_file(path):
    return sha256_bytes(path.read_bytes())


def canonical_digest(value):
    encoded = json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    return sha256_bytes(encoded)


def tree_digest(root):
    material = bytearray()
    for path in sorted(Path(root).rglob("*"), key=lambda item: item.relative_to(root).as_posix()):
        if ".git" in path.relative_to(root).parts:
            continue
        relative = path.relative_to(root).as_posix()
        if path.is_symlink():
            kind, payload = "symlink", os.readlink(path).encode("utf-8")
        elif path.is_file():
            kind, payload = "file", path.read_bytes()
        elif path.is_dir():
            kind, payload = "directory", b""
        else:
            continue
        material.extend(relative.encode("utf-8") + b"\0" + kind.encode("ascii") + b"\0" + payload + b"\0")
    return sha256_bytes(bytes(material))


DEEP_REVIEW_FILES = (
    "skills/shark-rider/skills/deep-review/SKILL.md",
    "skills/shark-rider/skills/deep-review/references/angle-a-bugs.md",
    "skills/shark-rider/skills/deep-review/references/angle-b-behavior.md",
    "skills/shark-rider/skills/deep-review/references/angle-c-sibling.md",
    "skills/shark-rider/skills/deep-review/references/angle-d-cleanup.md",
    "skills/shark-rider/skills/deep-review/references/angle-e-tests.md",
    "skills/shark-rider/skills/deep-review/references/angle-f-standards.md",
    "skills/shark-rider/skills/deep-review/references/consolidator.md",
    "skills/shark-rider/skills/deep-review/scripts/get_diff.sh",
)


def deep_review_digest(repo_root):
    material = bytearray()
    for relative in DEEP_REVIEW_FILES:
        path = Path(repo_root) / relative
        if not path.is_file():
            raise RuntimeError(f"deep-review identity source is missing: {path}")
        material.extend(relative.encode("utf-8") + b"\0" + path.read_bytes() + b"\0")
    return sha256_bytes(bytes(material))


def git_bytes(repo_root, args):
    try:
        completed = subprocess.run(
            ["git", *args], cwd=repo_root, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        raise RuntimeError(f"cannot derive candidate identity with git {' '.join(args)}: {exc}") from exc
    return completed.stdout


def adapter_json(adapter, capability, fixture_root, deadline=None, active_state=None):
    process = subprocess.Popen(
        [str(adapter), capability, "--checkout", str(fixture_root)],
        text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        start_new_session=True,
    )
    if active_state is not None:
        active_state["adapter_process"] = process
    timeout = None if deadline is None else max(0.001, deadline - time.monotonic())
    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except subprocess.TimeoutExpired as exc:
        stop_process_group(process)
        raise ResourceLimit(
            f"resource ceiling exceeded during execution-adapter {capability}: "
            "max_wall_clock_seconds"
        ) from exc
    finally:
        if active_state is not None and active_state.get("adapter_process") is process:
            active_state["adapter_process"] = None
    if process.returncode != 0:
        detail = stderr.strip() or stdout.strip()
        raise RuntimeError(f"execution adapter {capability} failed ({process.returncode}): {detail}")
    return load_json(stdout, f"execution adapter {capability}")


def candidate_identity(
    repo_root, execution_adapter, adapter_name, adapter_version,
    toolchain_identity, deadline=None, active_state=None,
):
    """Derive comparison identity from the post-dispatch checkout and adapter test inventory."""
    base_commit = git_bytes(repo_root, ["rev-parse", "HEAD"]).decode().strip()
    if not base_commit:
        raise RuntimeError("git merge-base returned an empty base commit")
    tracked_paths = git_bytes(repo_root, ["diff", "--name-only", "-z", base_commit])
    binary_diff = git_bytes(repo_root, ["diff", "--binary", base_commit])
    tracked = {
        path for path in git_bytes(repo_root, ["ls-files", "-z"]).decode().split("\0") if path
    }
    # Include ignored files as well as ordinary untracked files. Replay scans
    # the complete checkout, so omitting provider-created caches would make a
    # freshly captured snapshot report drift immediately.
    untracked = {
        path for path in git_bytes(repo_root, ["ls-files", "--others", "-z"]).decode().split("\0") if path
    }
    untracked.update({
        path for path in git_bytes(
            repo_root, ["ls-files", "--others", "--ignored", "--exclude-standard", "-z"]
        ).decode().split("\0") if path
    })
    manifest = []
    for path in sorted(tracked | untracked):
        candidate_path = Path(repo_root) / path
        if candidate_path.is_symlink():
            content = os.readlink(candidate_path).encode("utf-8")
        elif candidate_path.is_file():
            content = candidate_path.read_bytes()
        else:
            # Deleted tracked paths are represented by binary_diff_digest and
            # changed_path_digest. A file manifest cannot truthfully invent
            # bytes for an absent path.
            continue
        manifest.append({
            "path": path,
            "digest": f"sha256:{sha256_bytes(content)}",
            "tracked": path in tracked,
        })

    # Test discovery uses the adapter's real test capability, but it runs in
    # an isolated copy so cache and bytecode side effects do not mutate the
    # candidate whose identity is being recorded.
    test_identity_error = None
    try:
        with tempfile.TemporaryDirectory(prefix="e40-test-identity-") as temporary:
            isolated_checkout = Path(temporary) / "checkout"
            shutil.copytree(repo_root, isolated_checkout, symlinks=True, ignore=shutil.ignore_patterns(".git"))
            test_document = adapter_json(
                execution_adapter, "test", isolated_checkout, deadline, active_state,
            )
        entries = test_document.get("entries")
        if not isinstance(entries, list):
            raise RuntimeError("execution adapter test result omitted entries array")
        test_ids = sorted({
            str(entry["id"])
            for entry in entries
            if isinstance(entry, dict) and isinstance(entry.get("id"), str) and entry["id"]
        })
        if len(test_ids) != len(entries):
            raise RuntimeError("execution adapter test result contains malformed or duplicate test ids")
    except Cancellation:
        raise
    except (ResourceLimit, RuntimeError) as exc:
        test_ids = []
        test_identity_error = str(exc)
    test_identity = {
        "adapter": {"name": adapter_name, "version": adapter_version},
        "toolchain_identity": toolchain_identity,
        "test_ids": test_ids,
    }
    if test_identity_error:
        test_identity["error"] = test_identity_error
    top_levels = {
        test_id.split("::", 1)[0].replace("\\", "/").split("/", 1)[0].split(".", 1)[0]
        for test_id in test_ids
    }
    components = {
        "base_commit": base_commit,
        "tree_digest": sha256_bytes(git_bytes(repo_root, ["rev-parse", "HEAD^{tree}"])),
        "binary_diff_digest": sha256_bytes(binary_diff),
        "changed_path_digest": sha256_bytes(tracked_paths),
        "dirty_untracked_manifest": manifest,
        "test_suite_digest": canonical_digest(test_identity),
    }
    candidate = dict(components)
    candidate["test_suite_ids"] = test_ids
    if test_identity_error:
        candidate["test_identity_error"] = test_identity_error
    if len(top_levels) == 1:
        candidate["test_suite_dir"] = next(iter(top_levels))
    candidate["identity_digest"] = canonical_digest(components)
    candidate["snapshot_digest"] = canonical_digest(candidate)
    return candidate


def scratch_content_digest(root):
    """Digest the worker's actual scratch-project contents at stage end."""
    root = Path(root).resolve()
    material = bytearray()
    if not root.exists():
        return sha256_bytes(b"")
    for path in sorted(root.rglob("*"), key=lambda item: item.relative_to(root).as_posix()):
        relative = path.relative_to(root).as_posix().encode("utf-8")
        material.extend(relative)
        material.append(0)
        if path.is_symlink():
            material.extend(b"symlink\0")
            material.extend(os.readlink(path).encode("utf-8"))
        elif path.is_file():
            material.extend(b"file\0")
            material.extend(path.read_bytes())
        elif path.is_dir():
            material.extend(b"directory\0")
        material.append(0)
    return sha256_bytes(bytes(material))


def refresh_candidate(
    candidate, fixture_root, scratch, execution_adapter, adapter_name,
    adapter_version, toolchain_identity, deadline=None, active_state=None,
):
    # The stage snapshot is a post-dispatch observation. Re-derive every
    # fixture identity field after the worker returns so an edit made by this
    # stage cannot first appear in the following stage's evidence.
    current = candidate_identity(
        fixture_root, execution_adapter, adapter_name, adapter_version,
        toolchain_identity, deadline, active_state,
    )
    candidate.clear()
    candidate.update(current)
    candidate["scratch_content_digest"] = scratch_content_digest(scratch)
    candidate["snapshot_digest"] = canonical_digest(candidate)


def timestamp():
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")


def bounded(value):
    if isinstance(value, dict):
        return {str(key): bounded(item) for key, item in list(value.items())[:32]}
    if isinstance(value, list):
        return [bounded(item) for item in value[:32]]
    if isinstance(value, (str, int, float, bool)) or value is None:
        if isinstance(value, str):
            return value[:512]
        return value
    return str(value)[:512]


def load_json(stdout, label):
    try:
        return json.loads(stdout.strip())
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"{label} returned non-JSON output: {exc}") from exc


def run_command(shark, args, cwd, *, expect_json=True):
    command_args = list(args)
    if "--json" not in command_args:
        command_args.append("--json")
    try:
        completed = subprocess.run([shark, *command_args], cwd=cwd, text=True, capture_output=True, check=False)
    except OSError as exc:
        raise RuntimeError(f"unable to execute shark {' '.join(command_args)}: {exc}") from exc
    if completed.returncode != 0:
        detail = completed.stderr.strip() or completed.stdout.strip()
        raise RuntimeError(f"shark {' '.join(command_args)} failed ({completed.returncode}): {detail}")
    if expect_json:
        return load_json(completed.stdout, f"shark {' '.join(command_args)}")
    output = completed.stdout.strip()
    if not output:
        return {}
    try:
        return load_json(output, f"shark {' '.join(command_args)}")
    except RuntimeError:
        return {"output": output[:512]}


def pre_dispatch_gates(scenario_path, scenario, fixture_root, scratch):
    evaluator_root = scenario_path.parent.resolve()
    replay_reference = scenario.get("replay_reference")
    if scenario.get("entity_family") == "feature":
        if not isinstance(replay_reference, str) or not replay_reference.strip():
            raise RuntimeError("feature scenario is missing replay_reference")
        package_root = scenario_path.parent.resolve()
        bundle_path = (package_root / replay_reference).resolve()
        try:
            bundle_path.relative_to(package_root)
        except ValueError as exc:
            raise RuntimeError("replay_reference escapes the scenario package") from exc
        if not bundle_path.is_file():
            raise RuntimeError(f"replay bundle is missing: {bundle_path}")
        bundle = load_json(bundle_path.read_text(encoding="utf-8"), "replay bundle")
        binding = bundle.get("scenario_binding")
        if not isinstance(binding, dict) or binding.get("scenario_id") != scenario.get("scenario_id"):
            raise RuntimeError("replay bundle scenario_binding.scenario_id does not match the scenario")
        if binding.get("scenario_version") != scenario.get("scenario_version"):
            raise RuntimeError("replay bundle scenario_binding.scenario_version does not match the scenario")
        bundle_version = bundle.get("bundle_version")
        if not isinstance(bundle_version, str) or not bundle_version.strip():
            raise RuntimeError("replay bundle is missing bundle_version")
        guard = Path(os.environ["LIFECYCLE_BENCH_DIR"]) / "scripts" / "verify-replay-isolation.sh"
        process = subprocess.run([str(guard), str(bundle_path), str(fixture_root), str(scratch)], text=True, capture_output=True, check=False)
        if process.returncode != 0:
            detail = process.stderr.strip() or process.stdout.strip()
            raise RuntimeError(f"replay isolation gate rejected the roots: {detail}")

    guard = Path(os.environ["LIFECYCLE_BENCH_DIR"]) / "scripts" / "verify-evidence-roots.sh"
    process = subprocess.run(
        [str(guard), str(scenario_path), str(fixture_root), str(scratch), str(evaluator_root)],
        text=True, capture_output=True, check=False,
    )
    if process.returncode != 0:
        detail = process.stderr.strip() or process.stdout.strip()
        raise RuntimeError(f"evaluator disclosure gate rejected the roots: {detail}")


def write_partial(path, record):
    partial = path.with_suffix(path.suffix + ".partial")
    partial.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")


def scenario_identity(scenario_path, scenario, fixture_root, limits):
    fixture = scenario.get("fixture") or {}
    adapter = scenario.get("adapter") or {}
    version = scenario.get("scenario_version", "1")
    return {
        "schema_version": "1.0",
        "run_id": "",
        "scenario_id": str(scenario.get("scenario_id", scenario_path.stem)),
        "scenario_version": str(version),
        "fixture_id": str(fixture.get("fixture_id", "unknown")),
        "fixture_digest": tree_digest(fixture_root),
        "adapter_id": str(adapter.get("name", "unknown")),
        "adapter_version": str(adapter.get("version", "unknown")),
        "shark_binary_digest": "0" * 64,
        "shark_content_digest": "0" * 64,
        "toolchain_identity": scenario.get("toolchain_identity") or [],
        "rendered_prompt_digests": [],
        "provider_identity": [],
        "judge_identity": {"model": "not_applicable", "configuration": "not_applicable"},
        "reference_digests": [sha256_file(scenario_path)],
        "resource_policy_digest": canonical_digest(limits),
        "roots": {
            "agent_fixture_checkout": str(fixture_root),
            "scratch_shark_project": "",
            "evaluator_only": str(scenario_path.parent.resolve()),
        },
    }


def candidate_template():
    raise RuntimeError("candidate_template requires a derived candidate identity")


def make_record(identity, root, scratch, limits, repo_root):
    return {
        "identity": identity,
        "entity_graph": {
            "root_key": root, "root_type": "unknown", "resolved_via": "scenario_root",
            "fork_candidates": [], "selected_keys": [], "selected_types": [],
            "ordinals": [], "ineligible": [],
        },
        "dispatches": [], "stages": [],
        "workflow_policy": {
            "enabled_gates": [],
            "gate_order": [],
            "reviewer": {"provider": "unknown", "model": "unknown", "effort": "unknown"},
            "prompt_digest": "0" * 64, "rendered_prompt_digest": "0" * 64,
            "review_bundle_digest": deep_review_digest(repo_root),
            "deep_review_bundle_digest": deep_review_digest(repo_root),
            "fixes_allowed_between_gates": False,
        },
        "review_gates": [], "questions": [],
        "limits": {
            "max_cost_usd": limits["max_cost_usd"],
            "max_wall_clock_seconds": limits["max_wall_clock_seconds"],
            "max_generated_tasks": limits["max_generated_tasks"],
            "observed_cost_usd": 0.0, "observed_wall_clock_seconds": 0.0,
            "observed_generated_tasks": 0, "first_exceeded": None,
        },
        "outcome": {"terminal": "error", "reason": "run not started", "partial_evidence": False, "publication_eligible": False},
    }


def stage_record(dispatch, candidate):
    usage = dict((dispatch.get("worker") or {}).get("usage") or {})
    usage.update({
        "provider": dispatch["response"].get("provider", "unknown"),
        "model": dispatch["response"].get("model", "unknown"),
    })
    return {
        "dispatch_ordinal": dispatch["ordinal"], "stage": dispatch["response"].get("status", "development"),
        "category": stage_category(dispatch["response"].get("status", "development")), "snapshot_digest": candidate["snapshot_digest"],
        "prompt_digest": dispatch["response"].get("prompt_sha256", "0" * 64),
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": usage,
        "cost_usd": dispatch.get("cost_usd", 0.0), "elapsed_seconds": dispatch.get("elapsed_seconds", 0.0),
        "errors": [], "rework": False, "intervals": [], "candidate": candidate,
        "artifacts": [], "access_events": [],
        "evidence_refs": {"candidate_snapshot_digest": candidate["snapshot_digest"]},
    }


def prelude_lineage(record):
    replay = (record.get("prelude") or {}).get("replay") or {}
    replay_bundle = replay.get("replay_bundle") or {}
    reference = replay_bundle.get("bundle_path")
    lineage = []
    for stage in replay.get("stages") or []:
        if not isinstance(stage, dict):
            continue
        for artifact in stage.get("artifacts") or []:
            if not isinstance(artifact, dict):
                continue
            for entry in artifact.get("consumed_entries") or []:
                if isinstance(entry, dict) and entry.get("entry_digest"):
                    lineage.append({
                        "replay_reference": reference,
                        "entry_digest": entry["entry_digest"],
                    })
    return sorted(lineage, key=lambda item: (str(item["replay_reference"]), str(item["entry_digest"])))


def stage_input_lineage(record, dispatch, fixture_root, fixture_digest, execution_adapter, lifecycle_adapter):
    """Return the typed identities for every external input consumed by a stage."""
    identity = record["identity"]
    scratch_root = Path(identity["roots"]["scratch_shark_project"])
    prompt_path = scratch_root / "prompts" / f"{dispatch['ordinal']:04d}"
    scenario_path = Path(identity["scenario_path"])
    scenario = yaml.safe_load(scenario_path.read_text(encoding="utf-8")) or {}
    agent_visible_relative = str(((scenario.get("input") or {}).get("agent_visible")) or "")
    if not agent_visible_relative:
        raise RuntimeError("scenario package omitted input.agent_visible")
    scenario_root = scenario_path.parent.resolve()
    agent_visible_path = (scenario_root / agent_visible_relative).resolve()
    try:
        agent_visible_path.relative_to(scenario_root)
    except ValueError as exc:
        raise RuntimeError(
            f"scenario input.agent_visible escapes package root: {agent_visible_relative}"
        ) from exc
    if not agent_visible_path.is_file():
        raise RuntimeError(f"scenario input.agent_visible is missing: {agent_visible_path}")
    inputs = [
        {
            "source_kind": "scenario_package",
            "path": identity["scenario_path"],
            "digest": sha256_file(scenario_path),
        },
        {
            "source_kind": "agent_visible_input",
            "path": str(agent_visible_path),
            "digest": sha256_file(agent_visible_path),
        },
        {
            "source_kind": "rendered_prompt",
            "path": str(prompt_path),
            "digest": str(dispatch["response"]["prompt_sha256"]),
        },
        {
            "source_kind": "fixture_checkout",
            "path": str(fixture_root),
            "digest": fixture_digest,
        },
        {
            "source_kind": "shark_content",
            "path": str(scratch_root / "shark-data"),
            "digest": identity["shark_content_digest"],
        },
        {
            "source_kind": "execution_adapter",
            "path": str(execution_adapter),
            "digest": sha256_file(Path(execution_adapter)),
        },
    ]
    if lifecycle_adapter:
        inputs.append({
            "source_kind": "lifecycle_adapter",
            "path": str(Path(lifecycle_adapter).resolve()),
            "digest": sha256_file(Path(lifecycle_adapter).resolve()),
        })
    prelude = record.get("prelude") or {}
    if prelude.get("path") and prelude.get("digest"):
        inputs.append({
            "source_kind": "prelude_result",
            "path": str(prelude["path"]),
            "digest": str(prelude["digest"]),
        })
    for prior_stage in record.get("stages") or []:
        for artifact in prior_stage.get("artifacts") or []:
            if not isinstance(artifact, dict) or not artifact.get("path") or not artifact.get("digest"):
                raise RuntimeError("prior stage artifact omitted path or digest")
            inputs.append({
                "source_kind": "prior_stage_artifact",
                "path": str(artifact["path"]),
                "digest": str(artifact["digest"]),
            })
    return sorted(inputs, key=lambda item: (item["source_kind"], item["path"]))


def stage_category(status):
    normalized = str(status).lower()
    if "research" in normalized or "discover" in normalized:
        return "discovery"
    if "spec" in normalized or "refin" in normalized:
        return "specification"
    if "plan" in normalized or "ready" in normalized:
        return "planning"
    if "review" in normalized:
        return "review"
    if normalized == "qa" or "quality" in normalized:
        return "qa"
    if "uat" in normalized or "accept" in normalized or "approval" in normalized:
        return "uat"
    if "ship" in normalized or "complete" in normalized:
        return "shipping"
    return "code"


def evidence_snapshot_digest(stage):
    payload = {key: value for key, value in stage.items() if key != "snapshot_digest"}
    return "sha256:" + sha256_bytes(json.dumps(payload, sort_keys=True).encode("utf-8"))


def record_prior_artifact_consumption(
    record, evidence_root, stage_index, consuming_stage, observed_at,
):
    """Join a new stage to every prior artifact in both I-05 and I-07."""
    edge = {
        "consuming_stage": consuming_stage,
        "edge_kind": "read",
        "observed_at": observed_at,
    }
    index_by_ordinal = {
        item["dispatch_ordinal"]: item for item in stage_index
    }
    for lifecycle_stage in record.get("stages") or []:
        artifacts = lifecycle_stage.get("artifacts") or []
        if not artifacts:
            continue
        for artifact in artifacts:
            consumers = artifact.get("consumers")
            if not isinstance(consumers, list):
                raise RuntimeError("prior stage artifact consumers is not an array")
            consumers.append(dict(edge))

        ordinal = lifecycle_stage["dispatch_ordinal"]
        index_entry = index_by_ordinal.get(ordinal)
        if index_entry is None:
            raise RuntimeError(f"prior stage {ordinal} omitted I-05 index entry")
        snapshot_path = evidence_root / index_entry["snapshot_path"]
        snapshot = load_json(snapshot_path.read_text(encoding="utf-8"), "prior stage snapshot")
        snapshot_artifacts = snapshot.get("artifacts")
        if not isinstance(snapshot_artifacts, list) or len(snapshot_artifacts) != len(artifacts):
            raise RuntimeError(f"prior stage {ordinal} artifact graph disagrees between I-05 and I-07")
        lifecycle_by_identity = {
            (artifact.get("path"), artifact.get("digest")): artifact
            for artifact in artifacts
        }
        for artifact in snapshot_artifacts:
            identity = (artifact.get("path"), artifact.get("digest"))
            lifecycle_artifact = lifecycle_by_identity.get(identity)
            if lifecycle_artifact is None:
                raise RuntimeError(f"prior stage {ordinal} artifact identity disagrees between I-05 and I-07")
            artifact["consumers"] = [dict(item) for item in lifecycle_artifact["consumers"]]
        snapshot["snapshot_digest"] = evidence_snapshot_digest(snapshot)
        snapshot_path.write_text(
            json.dumps(snapshot, sort_keys=True, separators=(",", ":")) + "\n",
            encoding="utf-8",
        )
        index_entry["snapshot_digest"] = snapshot["snapshot_digest"]


def mapped_provider(response):
    provider = str(response.get("provider", "")).lower()
    return "anthropic_claude_cli" if "anthropic" in provider or "claude" in provider else provider


USAGE_BINDINGS = {
    "total_cost": ("cost_usd", "total_cost_usd", (int, float)),
    "input_tokens": ("input_tokens", "usage.input_tokens", int),
    "output_tokens": ("output_tokens", "usage.output_tokens", int),
    "cache_read_input_tokens": ("cache_read_input_tokens", "usage.cache_read_input_tokens", int),
    "cache_creation_input_tokens": ("cache_creation_input_tokens", "usage.cache_creation_input_tokens", int),
    "model_ids": ("model_ids", "sorted(modelUsage keys)", list),
    "api_active_duration_ms": ("api_active_duration_ms", "duration_api_ms", int),
    "turn_count": ("turn_count", "num_turns", int),
    "provider_session_id": ("provider_session_id", "session_id", str),
}


def semantic_usage(provider_name, raw_usage):
    if provider_name != "anthropic_claude_cli":
        return {}, [{
            "kind": "unmapped_provider",
            "detail": f"provider={provider_name or 'unknown'}",
        }]

    usage = {}
    errors = []
    for slot, (source, envelope_path, expected_type) in USAGE_BINDINGS.items():
        value = raw_usage.get(source)
        valid = isinstance(value, expected_type) and not isinstance(value, bool)
        if slot == "model_ids":
            valid = valid and all(isinstance(item, str) and item for item in value)
            if valid:
                value = sorted(value)
        elif slot == "provider_session_id":
            valid = valid and bool(value)
        else:
            valid = valid and value >= 0
        if valid:
            usage[slot] = float(value) if slot == "total_cost" else value
        else:
            errors.append({
                "kind": "usage_slot_unavailable",
                "detail": f"slot={slot} envelope_path={envelope_path}",
            })
    return usage, errors


def capture_review_gate(
    shark, scratch, evidence_root, entity, response, candidate, round_number,
    seen_note_ids, repo_root, policy,
):
    gate_id = str(response.get("status", "review"))
    policy = dict(policy)
    policy.update({
        "gate_id": gate_id,
        "provider": str(response.get("provider", "unknown")),
        "model": str(response.get("model", "unknown")),
        "effort": str(response.get("effort") or "default"),
        "reached": True,
        "rendered_prompt_digest": str(response.get("prompt_sha256", "")),
        "deep_review_bundle_digest": deep_review_digest(repo_root),
    })
    policy["policy_digest"] = canonical_digest({
        key: value for key, value in policy.items() if key != "policy_digest"
    })
    gate = {
        "gate_id": gate_id,
        "reached": True,
        "round": round_number,
        "collector_status": "complete",
        "candidate_ref": {"snapshot_digest": candidate["snapshot_digest"]},
        "policy_ref": {"policy_digest": policy["policy_digest"]},
        "findings": [],
    }
    try:
        entity_record = run_command(shark, ["get", entity, "--json"], scratch)
        notes = entity_record.get("notes") or []
        if not isinstance(notes, list):
            raise RuntimeError("entity notes are not an array")
        fresh = []
        for note in notes:
            if not isinstance(note, dict):
                continue
            note_id = note.get("id")
            if note_id in seen_note_ids:
                continue
            seen_note_ids.add(note_id)
            if note.get("note_type") == "review-finding":
                fresh.append(note)
        for note in fresh:
            metadata = note.get("metadata") or {}
            if isinstance(metadata, str):
                metadata = json.loads(metadata)
            if not isinstance(metadata, dict):
                raise RuntimeError("review-finding metadata is not an object")
            missing = [
                field for field in ("severity", "defect_class", "fingerprint")
                if not isinstance(metadata.get(field), str) or not metadata[field].strip()
            ]
            if missing:
                raise RuntimeError(f"review-finding metadata omitted {', '.join(missing)}")
            raw_bytes = json.dumps(note, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
            gate["findings"].append({
                "gate": str(metadata.get("gate") or gate_id),
                "round": int(metadata.get("round") or round_number),
                "severity": metadata["severity"],
                "defect_class": metadata["defect_class"],
                "fingerprint": metadata["fingerprint"],
                "criterion": str(metadata.get("criterion") or "not_reported"),
                "test": str(metadata.get("test") or "not_reported"),
                "disposition": str(metadata.get("disposition") or "open"),
                "metadata": bounded(metadata),
                "raw_bytes": raw_bytes,
            })
    except (RuntimeError, ValueError, TypeError, json.JSONDecodeError) as exc:
        gate["collector_status"] = "failure"
        gate["collector_error"] = str(exc)
        gate["findings"] = []

    capture_script = repo_root / "bench" / "scripts" / "review-capture.sh"
    with tempfile.TemporaryDirectory(dir=evidence_root) as temporary:
        source = Path(temporary) / "review.json"
        destination = Path(temporary) / "capture.json"
        source.write_text(json.dumps({"gates": [gate]}, sort_keys=True) + "\n", encoding="utf-8")
        process = subprocess.run(
            [str(capture_script), "--input", str(source), "--output", str(destination)],
            text=True, capture_output=True, check=False,
        )
        if process.returncode != 0:
            detail = process.stderr.strip() or process.stdout.strip()
            raise RuntimeError(f"review capture failed ({process.returncode}): {detail}")
        captured = load_json(destination.read_text(encoding="utf-8"), "review capture")
    return captured["review_gates"][0], policy


def write_stage_evidence(
    evidence_root, record, dispatch, candidate, fixture_root,
    fixture_input_digest, execution_adapter, lifecycle_adapter,
    started_ns, provider_started_ns, provider_ended_ns, ended_ns, rework_count,
):
    ordinal = dispatch["ordinal"]
    stage_key = str(dispatch["response"].get("status", "development"))
    category = stage_category(stage_key)
    artifacts_dir = evidence_root / "artifacts"
    stages_dir = evidence_root / "stages"
    transcripts_dir = evidence_root / "transcripts"
    artifacts_dir.mkdir(parents=True, exist_ok=True)
    stages_dir.mkdir(parents=True, exist_ok=True)
    transcripts_dir.mkdir(parents=True, exist_ok=True)
    patch_relative = f"artifacts/{ordinal:04d}-{stage_key}.patch"
    patch_path = evidence_root / patch_relative
    patch_bytes = git_bytes(fixture_root, ["diff", "--binary", candidate["base_commit"]])
    patch_path.write_bytes(patch_bytes)
    artifact = {
        "artifact_type": "code_diff", "path": patch_relative,
        "digest": sha256_bytes(patch_bytes), "size_bytes": len(patch_bytes),
        "producer_stage": stage_key, "consumers": [],
    }
    raw_usage = dict((dispatch.get("worker") or {}).get("usage") or {})
    provider_name = mapped_provider(dispatch["response"])
    usage, usage_errors = semantic_usage(provider_name, raw_usage)
    if candidate.get("test_identity_error"):
        usage_errors.append({
            "kind": "test_suite_unavailable",
            "detail": str(candidate["test_identity_error"]),
        })
    elapsed_ns = max(1, ended_ns - started_ns)
    provider_start = min(elapsed_ns, max(0, (provider_started_ns or ended_ns) - started_ns))
    intervals = {
        # The provider envelope supplies aggregate API-active duration but no
        # event boundaries. A duration alone cannot establish a genuine
        # half-open wall interval, so it remains a usage measurement and the
        # wall span stays explicitly unclassified.
        "provider_active": [],
        "tool_and_test": [],
        "queue_or_claim_wait": [[0, provider_start]] if provider_start > 0 else [],
        "replay_or_human_gate_wait": [],
        "retry_or_backoff": [],
        # Claude reports aggregate API-active duration, not event boundaries.
        # Keep every unmeasured part of the adapter and controller lifecycle
        # explicit instead of inventing provider or tool attribution.
        "unclassified": [[provider_start, elapsed_ns]] if provider_start < elapsed_ns else [],
    }
    stage_candidate = {
        key: candidate[key]
        for key in (
            "base_commit", "tree_digest", "binary_diff_digest",
            "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest",
        )
    }
    for key in ("test_suite_ids", "test_suite_dir", "test_identity_error"):
        if key in candidate:
            stage_candidate[key] = candidate[key]
    stage = {
        "dispatch_ordinal": ordinal,
        "entity": {"entity_key": dispatch["response"].get("entity_key"), "entity_type": dispatch["response"].get("entity_type")},
        "stage_key": stage_key, "stage_category": category,
        "provider": provider_name,
        "prompt_digest": dispatch["response"].get("prompt_sha256"),
        "input_lineage": stage_input_lineage(
            record, dispatch, fixture_root, fixture_input_digest,
            execution_adapter, lifecycle_adapter,
        ),
        "replay_lineage": prelude_lineage(record), "artifacts": [artifact], "usage": usage,
        "time_ledger": {
            "stage_start": 0, "stage_end": elapsed_ns, "reconciliation_epsilon_ns": 0,
            "intervals": intervals,
        },
        "candidate": stage_candidate,
        "errors": usage_errors, "rework_count": rework_count, "evaluator_access": [],
    }
    # I-05's established replay contract hashes the default, sorted JSON
    # serialization and carries an explicit algorithm prefix. I-07 keeps its
    # own unprefixed digest vocabulary, so the caller strips the prefix at
    # that interface boundary.
    stage["snapshot_digest"] = evidence_snapshot_digest(stage)
    snapshot_relative = f"stages/{ordinal:04d}-{stage_key}.json"
    (evidence_root / snapshot_relative).write_text(
        json.dumps(stage, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8"
    )
    transcript = {
        "schema_version": "1.0", "source": "bounded_worker_result",
        "dispatch_ordinal": ordinal, "stage": stage_key,
        "started_at": dispatch.get("started_at"), "ended_at": dispatch.get("ended_at"),
        "worker": dispatch.get("worker"), "outcome": dispatch.get("outcome"),
    }
    (transcripts_dir / f"{ordinal:04d}-{stage_key}.json").write_text(
        json.dumps(transcript, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8"
    )
    return stage, snapshot_relative, artifact


def write_i05_bundle(evidence_root, record, scenario, initial_roots, stage_index, terminal, ended_at):
    identity = record["identity"]
    bundle = {
        "schema_version": "1.0",
        "identity": {
            "run_id": identity["run_id"], "scenario_id": identity["scenario_id"],
            "scenario_version": identity["scenario_version"],
            "dispatch_id": identity["dispatch_id"], "dispatch_ordinal": identity["dispatch_ordinal"],
        },
        "scenario": {
            "scenario_id": identity["scenario_id"], "scenario_version": int(identity["scenario_version"]),
            "entity_family": scenario.get("entity_family"),
        },
        "run_id": identity["run_id"], "dispatch_id": identity["dispatch_id"],
        "dispatch_ordinal": identity["dispatch_ordinal"],
        "roots": initial_roots,
        "stage_matrix_source": {
            "package_path": identity["scenario_path"], "package_digest": sha256_file(Path(identity["scenario_path"])),
            "prelude": (scenario.get("stage_matrix") or {}).get("prelude") or {},
            "lifecycle": (scenario.get("stage_matrix") or {}).get("lifecycle") or {},
        },
        "dispatches": record["dispatches"], "stages": stage_index,
        "prelude": record.get("prelude", {}),
        # This bundle is written only after the controller has selected its
        # terminal outcome. A stop outcome is terminal but ineligible; it is
        # not the same thing as an in-progress bundle whose evaluator access
        # must still be refused.
        "terminal_status": {"reached": True, "reached_at": ended_at},
        "publication_eligible": terminal == "complete", "ineligibility_reasons": [] if terminal == "complete" else [record["outcome"]["reason"]],
    }
    if terminal != "complete":
        bundle["stop_outcome"] = terminal
    (evidence_root / "bundle.json").write_text(
        json.dumps(bundle, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8"
    )


def limits_from(scenario, path):
    policy = dict(scenario.get("resource_policy") or {})
    if path:
        try:
            loaded = yaml.safe_load(Path(path).read_text(encoding="utf-8")) or {}
        except (OSError, yaml.YAMLError) as exc:
            raise RuntimeError(f"cannot read limits policy {path}: {exc}") from exc
        policy.update(loaded)
    names = ("max_cost_usd", "max_wall_clock_seconds", "max_generated_tasks")
    try:
        values = {"max_cost_usd": float(policy["max_cost_usd"]), "max_wall_clock_seconds": float(policy["max_wall_clock_seconds"]), "max_generated_tasks": int(policy["max_generated_tasks"])}
    except (KeyError, TypeError, ValueError) as exc:
        raise RuntimeError(f"limits policy must declare {', '.join(names)}: {exc}") from exc
    if any(values[name] <= 0 for name in names):
        raise RuntimeError("resource ceilings must be positive")
    return values


def fork_candidates(response):
    entities = response.get("entities")
    if not isinstance(entities, list):
        entities = response.get("candidates")
    if not isinstance(entities, list):
        raise RuntimeError("parallel_candidates response must contain an entities array")
    seen = set()
    normalized = []
    for entity in entities:
        if not isinstance(entity, dict) or not entity.get("entity_key"):
            raise RuntimeError("parallel_candidates contains a malformed entity")
        key = str(entity["entity_key"])
        if key in seen:
            raise RuntimeError(f"parallel_candidates contains duplicate entity {key}")
        seen.add(key)
        normalized.append({"entity_key": key, "entity_type": str(entity.get("entity_type", "unknown"))})
    return sorted(normalized, key=lambda item: item["entity_key"])


def allowed_outcomes(scratch, response):
    entity_type = str(response.get("entity_type", "")).replace("_", "-")
    candidates = [entity_type, entity_type.replace("-card", "")]
    for name in candidates:
        path = scratch / "shark-data" / "workflow" / f"{name}.yaml"
        if not path.is_file():
            continue
        workflow = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        step = ((workflow.get("steps") or {}).get(str(response.get("status"))) or {})
        outcomes = step.get("outcomes") or {}
        if isinstance(outcomes, dict) and outcomes:
            return sorted(str(value) for value in outcomes)
    return ["blocked", "fail", "on_hold", "pass"]


def configured_gate_policies(scratch, entity_family, repo_root):
    """Retain every review-like gate reachable through any workflow outcome."""
    family_name = str(entity_family).replace("_", "-")
    candidates = [family_name, family_name.replace("-card", "")]
    workflow_path = next(
        (scratch / "shark-data" / "workflow" / f"{name}.yaml" for name in candidates
         if (scratch / "shark-data" / "workflow" / f"{name}.yaml").is_file()),
        None,
    )
    if workflow_path is None:
        return []
    workflow = yaml.safe_load(workflow_path.read_text(encoding="utf-8")) or {}
    steps = workflow.get("steps") or {}
    visited = set()
    pending = [workflow.get("start")]
    policies = []
    gate_phases = {"code_review", "review", "qa", "uat", "approval"}
    while pending:
        current = pending.pop(0)
        if not isinstance(current, str) or not current or current in visited:
            continue
        visited.add(current)
        step = steps.get(current) or {}
        phase = str(step.get("phase") or "")
        if step.get("action") == "spawn_agent" and phase in gate_phases:
            prompt_source = str(step.get("prompt") or "")
            prompt_path = scratch / "shark-data" / "prompts" / prompt_source
            prompt_source_digest = sha256_file(prompt_path) if prompt_path.is_file() else "0" * 64
            policy = {
                "gate_id": current,
                "phase": phase,
                "provider": str(step.get("provider") or "unknown"),
                "model": str(step.get("model") or "unknown"),
                "effort": str(step.get("effort") or "default"),
                "prompt_source": prompt_source,
                "prompt_source_digest": prompt_source_digest,
                "rendered_prompt_digest": None,
                "deep_review_bundle_digest": deep_review_digest(repo_root),
                "fixes_allowed": "fail" in (step.get("outcomes") or {}),
                "reached": False,
            }
            policy["policy_digest"] = canonical_digest(policy)
            policies.append(policy)
        outcomes = step.get("outcomes") or {}
        if isinstance(outcomes, dict):
            for target in outcomes.values():
                if (
                    isinstance(target, str) and target in steps
                    and target not in visited and target not in pending
                ):
                    pending.append(target)
    return policies


def refresh_workflow_policy(record):
    policy = record["workflow_policy"]
    gate_policies = policy.get("gate_policies") or []
    policy["enabled_gates"] = []
    for item in gate_policies:
        if item["gate_id"] not in policy["enabled_gates"]:
            policy["enabled_gates"].append(item["gate_id"])
    policy["gate_order"] = list(policy["enabled_gates"])
    if len(policy["enabled_gates"]) == 1:
        item = next(
            candidate for candidate in reversed(gate_policies)
            if candidate["gate_id"] == policy["enabled_gates"][0]
        )
        policy["reviewer"] = {
            "provider": item["provider"], "model": item["model"], "effort": item["effort"],
        }
    elif gate_policies:
        policy["reviewer"] = {
            "provider": "per_gate", "model": "per_gate", "effort": "per_gate",
        }
    policy["prompt_digest"] = canonical_digest([
        item["prompt_source_digest"] for item in gate_policies
    ]) if gate_policies else "0" * 64
    policy["rendered_prompt_digest"] = canonical_digest([
        item.get("rendered_prompt_digest") for item in gate_policies
    ]) if gate_policies else "0" * 64
    policy["fixes_allowed_between_gates"] = any(
        bool(item.get("fixes_allowed")) for item in gate_policies
    )


def gate_policy_for(record, response, repo_root):
    gate_id = str(response.get("status", "review"))
    for policy in record["workflow_policy"].get("gate_policies") or []:
        if policy.get("gate_id") == gate_id:
            return policy
    fallback = {
        "gate_id": gate_id,
        "phase": stage_category(gate_id),
        "provider": str(response.get("provider") or "unknown"),
        "model": str(response.get("model") or "unknown"),
        "effort": str(response.get("effort") or "default"),
        "prompt_source": "unavailable",
        "prompt_source_digest": "0" * 64,
        "rendered_prompt_digest": None,
        "deep_review_bundle_digest": deep_review_digest(repo_root),
        "fixes_allowed": False,
        "reached": False,
    }
    fallback["policy_digest"] = canonical_digest(fallback)
    record["workflow_policy"].setdefault("gate_policies", []).append(fallback)
    return fallback


def retain_gate_policy(record, replacement):
    policies = record["workflow_policy"].get("gate_policies") or []
    for policy in policies:
        if policy.get("policy_digest") == replacement.get("policy_digest"):
            return
    policies.append(replacement)


def stop_process_group(process):
    if process is None or process.poll() is not None:
        return
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait(timeout=2)


def adapter_result(adapter, request, cwd, mode, shark, fixture_root, deadline, active_state):
    if mode in {"contract", "dry-run"}:
        return ({"worker_id": "offline-worker", "session_id": request["session_id"], "kind": "final", "recommended_outcome": "pass", "evidence": {"mode": mode}}, [])
    try:
        adapter_env = dict(os.environ)
        adapter_env["E40_AGENT_FIXTURE_CHECKOUT"] = str(fixture_root)
        process = subprocess.Popen(
            [adapter], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            cwd=cwd, env=adapter_env, text=True, start_new_session=True,
        )
        active_state["adapter_process"] = process
        process.stdin.write(json.dumps(request, separators=(",", ":")))
        process.stdin.close()
    except OSError as exc:
        stop_process_group(locals().get("process"))
        active_state["adapter_process"] = None
        raise RuntimeError(f"unable to execute lifecycle adapter: {exc}") from exc
    explicit_interval = os.environ.get("LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS")
    if explicit_interval:
        try:
            heartbeat_interval = max(0.001, float(explicit_interval))
        except ValueError as exc:
            stop_process_group(process)
            active_state["adapter_process"] = None
            raise RuntimeError(f"invalid LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS: {explicit_interval}") from exc
    else:
        try:
            ttl = float(os.environ.get("SHARK_CLAIM_TTL_SECONDS", "180"))
        except ValueError as exc:
            stop_process_group(process)
            active_state["adapter_process"] = None
            raise RuntimeError("invalid SHARK_CLAIM_TTL_SECONDS") from exc
        heartbeat_interval = max(1.0, min(60.0, ttl / 3.0))
    last_heartbeat = time.monotonic()
    heartbeat_events = []
    try:
        while process.poll() is None:
            now = time.monotonic()
            if now >= deadline:
                stop_process_group(process)
                active_state["adapter_process"] = None
                raise ResourceLimit("resource ceiling exceeded during provider dispatch: max_wall_clock_seconds")
            if now - last_heartbeat >= heartbeat_interval:
                try:
                    heartbeat = run_command(shark, ["heartbeat", request["entity_key"], "--session", request["session_id"], "--progress", "0.5", "--note", str(request.get("runner_id", "lifecycle"))], cwd, expect_json=False)
                    heartbeat_events.append({"session_id": request["session_id"], "at": timestamp(), "response": bounded(heartbeat)})
                except RuntimeError as exc:
                    stop_process_group(process)
                    active_state["adapter_process"] = None
                    raise LeaseLoss(f"heartbeat failed for {request['entity_key']}: {exc}") from exc
                last_heartbeat = now
            time.sleep(min(0.05, heartbeat_interval / 4.0))
        stdout = process.stdout.read()
        stderr = process.stderr.read()
        process.wait()
    except (BrokenPipeError, OSError, subprocess.TimeoutExpired) as exc:
        if process.poll() is None:
            stop_process_group(process)
        active_state["adapter_process"] = None
        raise RuntimeError(f"lifecycle adapter process failed: {exc}") from exc
    active_state["adapter_process"] = None
    if process.returncode != 0:
        raise RuntimeError(f"lifecycle adapter failed ({process.returncode}): {stderr.strip()}")
    return load_json(stdout, "lifecycle adapter"), heartbeat_events


def route_worker_question(worker_result, entity, session, runner_id, cwd, shark):
    required = ("category", "question", "why_blocking", "recommendation")
    if any(not isinstance(worker_result.get(field), str) or not worker_result[field].strip() for field in required):
        raise RuntimeError(f"question worker result for {entity} omitted required handoff fields")
    title = worker_result["question"][:200]
    summary = f"{worker_result['why_blocking']} Recommendation: {worker_result['recommendation']}"
    created = run_command(
        shark,
        ["question", "create", title, "--summary", summary, "--requester", runner_id, "--blocking"],
        cwd,
    )
    question_key = str(created.get("key", ""))
    if not question_key:
        raise RuntimeError(f"question worker handoff for {entity} returned no question key")
    run_command(
        shark,
        ["question", "configure-workflow", question_key, "--resolution-owner", runner_id, "--responder", runner_id],
        cwd,
        expect_json=False,
    )
    run_command(shark, ["link", question_key, entity, "--type", "question_blocks"], cwd, expect_json=False)
    return question_key


def main(argv):
    args = parse_args(argv)
    scenario_path = Path(args["scenario"]).resolve()
    if not scenario_path.is_file():
        raise RuntimeError(f"scenario package not found: {scenario_path}")
    try:
        scenario = yaml.safe_load(scenario_path.read_text(encoding="utf-8")) or {}
    except (OSError, yaml.YAMLError) as exc:
        raise RuntimeError(f"cannot read scenario package {scenario_path}: {exc}") from exc
    if (scenario.get("admission") or {}).get("status") != "admitted":
        raise RuntimeError("scenario package is not admitted")

    scratch = Path(args["scratch_root"]).resolve()
    scratch.mkdir(parents=True, exist_ok=True)
    fixture_decl = (scenario.get("fixture") or {}).get("submodule_path", scenario_path.parent)
    fixture_root = Path(args["fixture_root"] or fixture_decl).resolve()
    if not fixture_root.is_dir() or not (fixture_root / ".git").exists():
        raise RuntimeError(f"agent fixture checkout is not a git checkout: {fixture_root}")
    default_output = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "runs" / args["run_id"] / "lifecycle.jsonl"
    output = Path(args["output"]).resolve() if args["output"] else Path(os.environ.get("LIFECYCLE_OUTPUT", str(default_output))).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    evidence_root = Path(args["evidence_root"] or (output.parent / "evidence")).resolve()
    evidence_root.mkdir(parents=True, exist_ok=True)
    limits = limits_from(scenario, args["limits"])
    shark = os.environ.get("SHARK_BIN", "shark")
    resolved_shark = shutil.which(shark)
    if not resolved_shark:
        raise RuntimeError(f"shark executable not found on PATH: {shark}")
    adapter = os.environ.get("LIFECYCLE_ADAPTER") or os.environ.get("LIFECYCLE_ADAPTER_PATH", "")
    if args["mode"] not in {"contract", "dry-run"} and not adapter:
        raise RuntimeError("LIFECYCLE_ADAPTER is required for live lifecycle runs")

    repo_root = Path(os.environ["LIFECYCLE_BENCH_DIR"]).parent.resolve()
    adapter_decl = scenario.get("adapter") or {}
    adapter_name = str(adapter_decl.get("name", "unknown"))
    adapter_version = str(adapter_decl.get("version", "unknown"))
    execution_adapter = repo_root / "bench" / "adapters" / adapter_name / "adapter.sh"
    if not execution_adapter.is_file() or not os.access(execution_adapter, os.X_OK):
        raise RuntimeError(f"registered execution adapter is unavailable: {execution_adapter}")
    toolchain_identity = scenario.get("toolchain_identity") or []
    identity = scenario_identity(scenario_path, scenario, fixture_root, limits)
    identity["run_id"] = args["run_id"]
    identity["dispatch_id"] = f"{args['run_id']}:1"
    identity["dispatch_ordinal"] = 1
    identity["scenario_path"] = str(scenario_path)
    identity["roots"]["scratch_shark_project"] = str(scratch)
    identity["shark_binary_digest"] = sha256_file(Path(resolved_shark))
    identity["shark_content_digest"] = tree_digest(scratch / "shark-data")
    pre_dispatch_gates(scenario_path, scenario, fixture_root, scratch)
    record = make_record(identity, args["root"], scratch, limits, repo_root)
    record["entity_graph"]["root_type"] = str(scenario.get("entity_family", "unknown"))
    record["workflow_policy"]["gate_policies"] = configured_gate_policies(
        scratch, scenario.get("entity_family", "unknown"), repo_root,
    )
    refresh_workflow_policy(record)
    prelude_stop = None
    prelude_reason = ""
    prelude_path = Path(args["prelude"]).resolve() if args["prelude"] else None
    if prelude_path is None and isinstance((scenario.get("stage_matrix") or {}).get("prelude"), dict):
        prelude_path = evidence_root / "prelude.jsonl"
        prelude_command = [
            str(repo_root / "bench" / "scripts" / "lifecycle-prelude.sh"),
            "--scenario", str(scenario_path), "--run-id", args["run_id"],
            "--output", str(prelude_path), "--fixture-root", str(fixture_root),
            "--scratch-root", str(scratch), "--evaluator-root", str(scenario_path.parent.resolve()),
        ]
        if args["replay"]:
            prelude_command.extend(["--replay", str(Path(args["replay"]).resolve())])
        process = subprocess.run(prelude_command, text=True, capture_output=True, check=False)
        if process.returncode not in {0, 1} or not prelude_path.is_file():
            detail = process.stderr.strip() or process.stdout.strip()
            raise RuntimeError(f"lifecycle prelude failed without retained evidence ({process.returncode}): {detail}")
    if prelude_path is not None:
        prelude = load_json(prelude_path.read_text(encoding="utf-8"), "lifecycle prelude")
        if prelude.get("scenario_id") != identity["scenario_id"]:
            raise RuntimeError("lifecycle prelude scenario_id does not match the scenario package")
        try:
            retained_prelude_path = prelude_path.relative_to(evidence_root).as_posix()
        except ValueError:
            retained_prelude_path = str(prelude_path)
        record["prelude"] = {
            "path": retained_prelude_path,
            "digest": sha256_file(prelude_path),
            "terminal_outcome": prelude["terminal_outcome"],
            "stages": bounded(prelude.get("prelude") or []),
            "replay": bounded(prelude.get("replay") or {}),
        }
        record["questions"] = bounded(prelude.get("questions") or [])
        identity["reference_digests"].append(sha256_file(prelude_path))
        if prelude.get("terminal_outcome") not in {"complete", "not_applicable"} or prelude.get("publication_eligible") is not True:
            prelude_terminal = str(prelude.get("terminal_outcome") or "unresolved_gate")
            prelude_stop = "unresolved_gate" if prelude_terminal in {"replay_desync", "unresolved_gate"} else "error"
            prelude_reason = str(prelude.get("reason") or f"prelude stopped with {prelude_terminal}")
    initial_roots = {
        "agent_fixture_checkout": {"path": str(fixture_root), "worker_access": "read_write", "identity_digest": tree_digest(fixture_root)},
        "scratch_shark_project": {"path": str(scratch), "worker_access": "authorized_surfaces_only", "identity_digest": tree_digest(scratch)},
        "evaluator_only": {"path": str(scenario_path.parent.resolve()), "worker_access": "never_during_dispatch", "identity_digest": tree_digest(scenario_path.parent)},
    }
    stage_index = []
    ordinal = 0
    generated = 0
    started = time.monotonic()
    queue = [] if prelude_stop else [args["root"]]
    generated_entities = set()
    stage_visits = {}
    review_rounds = {}
    seen_review_note_ids = set()
    evidence_errors = []
    terminal = prelude_stop or "complete"
    reason = prelude_reason or "all eligible dispatches completed"
    active_lease = {"entity": "", "session": "", "adapter_process": None}

    def release_on_signal(signum, _frame):
        stop_process_group(active_lease["adapter_process"])
        active_lease["adapter_process"] = None
        raise Cancellation(f"received signal {signum}")

    signal.signal(signal.SIGINT, release_on_signal)
    signal.signal(signal.SIGTERM, release_on_signal)

    try:
        while queue:
            requested = queue.pop(0)
            pre_dispatch_gates(scenario_path, scenario, fixture_root, scratch)
            prompt_path = scratch / "prompts" / f"{ordinal + 1:04d}"
            prompt_path.parent.mkdir(parents=True, exist_ok=True)
            response = run_command(shark, ["next", requested, "--json", "--prompt-out", str(prompt_path)], scratch)
            if response.get("action") == "parallel_candidates":
                candidates = fork_candidates(response)
                record["entity_graph"]["fork_candidates"].append({"response": bounded(response), "candidates": candidates})
                record["entity_graph"]["selected_keys"].extend(item["entity_key"] for item in candidates)
                record["entity_graph"]["selected_types"].extend(item["entity_type"] for item in candidates)
                record["entity_graph"]["resolved_via"] = "fork_response"
                queue[0:0] = [*[item["entity_key"] for item in candidates], requested]
                continue
            if response.get("action") == "archive":
                continue
            if response.get("action") != "spawn_agent":
                terminal = str(response.get("action", "error")) if response.get("action") in STOP_OUTCOMES else "error"
                reason = str(response.get("error") or f"keyed dispatch returned action {response.get('action')!r}")
                break
            entity = str(response.get("entity_key", ""))
            if not entity:
                raise RuntimeError("keyed response omitted entity_key")
            prompt = str(response.get("prompt", ""))
            expected_digest = str(response.get("prompt_sha256", ""))
            expected_bytes = response.get("prompt_bytes")
            actual = prompt.encode("utf-8")
            if sha256_bytes(actual) != expected_digest or len(actual) != expected_bytes:
                raise RuntimeError(f"prompt digest or byte-count mismatch for {entity}")
            if prompt_path.read_bytes() != actual:
                raise RuntimeError(f"prompt-out bytes differ from response for {entity}")
            candidate = {}

            response_record = bounded(response)
            response_record.pop("prompt", None)
            dispatch = {"ordinal": ordinal + 1, "requested_key": requested, "response": response_record, "claim": {}, "worker": {}, "heartbeats": [], "outcome": "error", "transition": {}, "release": {}, "started_at": timestamp(), "ended_at": "", "evidence_refs": {"prompt_sha256": expected_digest, "prompt_bytes": expected_bytes, "candidate_snapshot_digest": "0" * 64}}
            record["dispatches"].append(dispatch)
            record["entity_graph"]["ordinals"].append(dispatch["ordinal"])
            if entity != args["root"] and response.get("entity_type") == "task":
                generated_entities.add(entity)
                generated = len(generated_entities)
            session = ""
            worker_result = {}
            advanced = False
            stage_started_ns = time.monotonic_ns()
            fixture_input_digest = tree_digest(fixture_root)
            provider_started_ns = None
            provider_ended_ns = None
            try:
                claim = run_command(shark, ["claim", entity, "--by", os.environ.get("LIFECYCLE_RUNNER_ID", args["run_id"]), "--json"], scratch)
                session = str(claim.get("session_id", ""))
                if not session:
                    raise RuntimeError(f"claim for {entity} omitted session_id")
                active_lease["entity"] = entity
                active_lease["session"] = session
                dispatch["claim"] = bounded(claim)
                request = dict(response)
                request["session_id"] = session
                request["runner_id"] = os.environ.get("LIFECYCLE_RUNNER_ID", args["run_id"])
                request["allowed_outcomes"] = allowed_outcomes(scratch, response)
                provider_started_ns = time.monotonic_ns()
                try:
                    worker_result, heartbeats = adapter_result(
                        adapter, request, scratch, args["mode"], shark, fixture_root,
                        started + limits["max_wall_clock_seconds"], active_lease,
                    )
                finally:
                    provider_ended_ns = time.monotonic_ns()
                dispatch["heartbeats"] = heartbeats
                if worker_result.get("session_id") not in {None, session}:
                    raise RuntimeError(f"worker session mismatch for {entity}")
                dispatch["worker"] = {"worker_id": worker_result.get("worker_id", ""), "session_id": worker_result.get("session_id", session), "kind": worker_result.get("kind", ""), "recommended_outcome": worker_result.get("recommended_outcome"), "evidence": bounded(worker_result.get("evidence", {})), "usage": bounded(worker_result.get("usage", {})), "cost_usd": worker_result.get("cost_usd", 0.0)}
                kind = worker_result.get("kind")
                if kind == "question":
                    question_key = route_worker_question(
                        worker_result, entity, session, request["runner_id"], scratch, shark
                    )
                    dispatch["worker"]["question_key"] = question_key
                    dispatch["outcome"] = "pause"
                    terminal = "pause"
                    reason = f"worker question routed to {question_key}"
                    outcome = None
                else:
                    outcome = worker_result.get("recommended_outcome") if kind == "final" else None
                if not outcome:
                    if kind != "question":
                        terminal = "missing_outcome"
                        reason = f"worker for {entity} did not return a final recommended_outcome"
                        dispatch["outcome"] = terminal
                else:
                    dispatch["outcome"] = str(outcome)
                    run_command(shark, ["status", "advance", entity, "--outcome", str(outcome), "--session", session, "--from-status", str(response.get("status", "")), "--agent", f"{response.get('agent_type', '')}@{response.get('provider', '')}"], scratch, expect_json=False)
                    dispatch["transition"] = {"outcome": str(outcome), "session_id": session, "from_status": response.get("status", "")}
                    advanced = True
            except ResourceLimit as exc:
                terminal = "resource_limit"
                record["limits"]["first_exceeded"] = "max_wall_clock_seconds"
                reason = str(exc)
                dispatch["outcome"] = terminal
            except Cancellation as exc:
                terminal = "cancellation"
                reason = str(exc)
                dispatch["outcome"] = terminal
            except LeaseLoss as exc:
                terminal = "lease_loss"
                reason = str(exc)
                dispatch["outcome"] = terminal
            except RuntimeError as exc:
                terminal = "worker_failure" if session else "error"
                reason = str(exc)
                dispatch["outcome"] = terminal
            finally:
                if session:
                    try:
                        dispatch["release"] = bounded(run_command(shark, ["release", entity, "--session", session, "--outcome", dispatch["outcome"]], scratch, expect_json=False))
                    except RuntimeError as exc:
                        dispatch["release"] = {"error": str(exc)}
                        if terminal == "complete":
                            terminal = "error"
                            reason = str(exc)
                    active_lease["entity"] = ""
                    active_lease["session"] = ""
            dispatch["ended_at"] = timestamp()
            cost = float(worker_result.get("cost_usd", (worker_result.get("usage") or {}).get("cost_usd", 0.0)) or 0.0)
            dispatch["cost_usd"] = cost
            record["limits"]["observed_cost_usd"] += cost
            record["limits"]["observed_generated_tasks"] = generated
            refresh_candidate(
                candidate, fixture_root, scratch, execution_adapter,
                adapter_name, adapter_version, toolchain_identity,
                started + limits["max_wall_clock_seconds"], active_lease,
            )
            stage_ended_ns = time.monotonic_ns()
            elapsed = max(0.0, time.monotonic() - started)
            record["limits"]["observed_wall_clock_seconds"] = elapsed
            stage_candidate = dict(candidate)
            visit_key = (entity, str(response.get("status", "development")))
            rework_count = stage_visits.get(visit_key, 0)
            stage_visits[visit_key] = rework_count + 1
            record_prior_artifact_consumption(
                record, evidence_root, stage_index,
                str(response.get("status", "development")), dispatch["started_at"],
            )
            evidence_stage, snapshot_path, artifact = write_stage_evidence(
                evidence_root, record, dispatch, stage_candidate, fixture_root,
                fixture_input_digest, execution_adapter, adapter,
                stage_started_ns, provider_started_ns, provider_ended_ns, stage_ended_ns,
                rework_count,
            )
            dispatch["evidence_refs"]["candidate_snapshot_digest"] = stage_candidate["snapshot_digest"]
            lifecycle_stage = stage_record(dispatch, stage_candidate)
            lifecycle_stage["snapshot_digest"] = stage_candidate["snapshot_digest"]
            lifecycle_stage["output_paths"] = [artifact["path"]]
            lifecycle_stage["output_digests"] = [artifact["digest"]]
            lifecycle_stage["artifacts"] = [artifact]
            lifecycle_stage["errors"] = list(evidence_stage["errors"])
            evidence_errors.extend(
                {"dispatch_ordinal": dispatch["ordinal"], **item}
                for item in evidence_stage["errors"]
            )
            lifecycle_stage["input_lineage"] = list(evidence_stage["input_lineage"])
            lifecycle_stage["rework"] = rework_count > 0
            lifecycle_stage["replay_lineage"] = list(evidence_stage["replay_lineage"])
            stage_elapsed_seconds = (stage_ended_ns - stage_started_ns) / 1_000_000_000
            # I-05's time ledger is explicitly nanosecond-valued, while I-07's
            # intervals reconcile against elapsed_seconds and are consumed by F10
            # as seconds. Keep the two contracts distinct at this boundary.
            lifecycle_stage["intervals"] = [
                {"category": category, "start": start / 1_000_000_000, "end": end / 1_000_000_000}
                for category, ranges in evidence_stage["time_ledger"]["intervals"].items()
                for start, end in ranges
            ]
            lifecycle_stage["elapsed_seconds"] = stage_elapsed_seconds
            record["stages"].append(lifecycle_stage)
            stage_index.append({
                "dispatch_ordinal": dispatch["ordinal"], "stage_key": evidence_stage["stage_key"],
                "stage_category": evidence_stage["stage_category"], "snapshot_path": snapshot_path,
                "snapshot_digest": evidence_stage["snapshot_digest"],
            })
            prompt_digest = str(response.get("prompt_sha256"))
            record["identity"]["rendered_prompt_digests"].append(prompt_digest)
            effort = str(response.get("effort") or "default")
            record["identity"]["provider_identity"].append({
                "stage": str(response.get("status")), "provider": str(response.get("provider")),
                "model": str(response.get("model")), "effort": effort,
            })
            if evidence_stage["stage_category"] in {"review", "qa", "uat"}:
                gate_id = evidence_stage["stage_key"]
                review_rounds[gate_id] = review_rounds.get(gate_id, 0) + 1
                gate, captured_policy = capture_review_gate(
                    shark, scratch, evidence_root, entity, response, stage_candidate,
                    review_rounds[gate_id], seen_review_note_ids, repo_root,
                    gate_policy_for(record, response, repo_root),
                )
                record["review_gates"].append(gate)
                captured_policy["fixes_allowed"] = (
                    captured_policy.get("fixes_allowed")
                    or "fail" in request.get("allowed_outcomes", [])
                )
                captured_policy["policy_digest"] = canonical_digest({
                    key: value for key, value in captured_policy.items()
                    if key != "policy_digest"
                })
                gate["policy_ref"]["policy_digest"] = captured_policy["policy_digest"]
                retain_gate_policy(record, captured_policy)
                refresh_workflow_policy(record)
            ordinal += 1
            write_partial(output, record)
            if terminal != "complete" and terminal != "resource_limit":
                break
            exceeded = None
            if record["limits"]["observed_cost_usd"] >= limits["max_cost_usd"]:
                exceeded = "max_cost_usd"
            elif elapsed >= limits["max_wall_clock_seconds"]:
                exceeded = "max_wall_clock_seconds"
            elif generated >= limits["max_generated_tasks"]:
                exceeded = "max_generated_tasks"
            if exceeded:
                terminal = "resource_limit"
                record["limits"]["first_exceeded"] = exceeded
                reason = f"resource ceiling exceeded: {exceeded}"
                break
            if advanced and args["mode"] != "dry-run":
                queue.insert(0, requested)
        if terminal == "complete" and evidence_errors:
            terminal = "error"
            reason = "required stage evidence is unavailable: " + "; ".join(
                f"dispatch={item['dispatch_ordinal']} {item['kind']} {item['detail']}"
                for item in evidence_errors
            )
        record["outcome"] = {"terminal": terminal, "reason": "completed with complete evidence" if terminal == "complete" else reason, "partial_evidence": terminal != "complete" and bool(record["dispatches"]), "publication_eligible": terminal == "complete"}
    except (RuntimeError, OSError, ValueError, TypeError) as exc:
        terminal = "cancellation" if isinstance(exc, Cancellation) else "error"
        reason = str(exc)
        record["outcome"] = {"terminal": terminal, "reason": reason, "partial_evidence": bool(record["dispatches"]), "publication_eligible": False}
    reached_gate_ids = {gate.get("gate_id") for gate in record["review_gates"]}
    last_snapshot = record["stages"][-1]["candidate"]["snapshot_digest"] if record["stages"] else None
    for gate_id in record["workflow_policy"].get("enabled_gates") or []:
        if gate_id in reached_gate_ids:
            continue
        gate_policy = next(
            policy for policy in record["workflow_policy"].get("gate_policies") or []
            if policy["gate_id"] == gate_id
        )
        record["review_gates"].append({
            "gate_id": gate_id,
            "reached": False,
            "state": "not_reached",
            "round": 0,
            "findings": [],
            "candidate_ref": {"snapshot_digest": last_snapshot},
            "policy_ref": {"policy_digest": gate_policy["policy_digest"]},
        })
    refresh_workflow_policy(record)
    policy = record["workflow_policy"]
    policy["workflow_policy_identity_digest"] = canonical_digest({key: value for key, value in policy.items() if key != "workflow_policy_identity_digest"})
    ended_at = timestamp()
    write_i05_bundle(evidence_root, record, scenario, initial_roots, stage_index, terminal, ended_at)
    output.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
    output.with_suffix(output.suffix + ".partial").unlink(missing_ok=True)
    # Named stop outcomes are valid, retained I-07 results. Their
    # publication eligibility is false, but that is not a runner execution
    # error; F09 must still evaluate them and F10 must retain the evidence.
    return 0


try:
    raise SystemExit(main(sys.argv[1:]))
except (RuntimeError, OSError, ValueError, TypeError) as exc:
    print(f"run-lifecycle: {exc}", file=sys.stderr)
    raise SystemExit(1)
PY
