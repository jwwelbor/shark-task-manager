#!/usr/bin/env bash
# run-lifecycle.sh --scenario <package.yaml> --run-id <id> --root <key>
#                    --scratch-root <dir> [--output <lifecycle.jsonl>]
#                    [--limits <policy.yaml>] [--mode contract|dry-run]
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

LIFECYCLE_BENCH_DIR="$BENCH_DIR" python3 - "$@" <<'PY'
import hashlib
import json
import os
import shutil
import signal
import subprocess
import sys
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


def usage():
    print(
        "usage: run-lifecycle.sh --scenario <package.yaml> --run-id <id> "
        "--root <key> --scratch-root <dir> [--output <path>] "
        "[--limits <policy.yaml>] [--mode contract|dry-run]",
        file=sys.stderr,
    )
    raise SystemExit(2)


def parse_args(argv):
    values = {"mode": "live", "output": "", "limits": ""}
    required = {"--scenario": "scenario", "--run-id": "run_id", "--root": "root", "--scratch-root": "scratch_root"}
    index = 0
    while index < len(argv):
        option = argv[index]
        if option in required or option in {"--output", "--limits", "--mode"}:
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


def git_bytes(repo_root, args):
    try:
        completed = subprocess.run(
            ["git", *args], cwd=repo_root, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        raise RuntimeError(f"cannot derive candidate identity with git {' '.join(args)}: {exc}") from exc
    return completed.stdout


def candidate_identity(repo_root):
    """Derive comparison identity from the actual committed and dirty state."""
    base_commit = git_bytes(repo_root, ["merge-base", "HEAD", "main"]).decode().strip()
    if not base_commit:
        raise RuntimeError("git merge-base returned an empty base commit")
    tracked_paths = git_bytes(repo_root, ["diff", "--name-only", "-z", "main...HEAD"])
    binary_diff = git_bytes(repo_root, ["diff", "--binary", "main...HEAD"])
    dirty_manifest = git_bytes(repo_root, ["status", "--porcelain=v1", "--untracked-files=all"])
    test_paths = git_bytes(repo_root, ["ls-files", "-z", "--", "*_test.go", "**/*_test.go"])
    test_material = bytearray()
    for path in sorted(filter(None, test_paths.decode().split("\0"))):
        test_material.extend(path.encode("utf-8"))
        test_material.append(0)
        test_material.extend((Path(repo_root) / path).read_bytes())
        test_material.append(0)
    components = {
        "base_commit": base_commit,
        "tree_digest": sha256_bytes(git_bytes(repo_root, ["rev-parse", "HEAD^{tree}"])),
        "binary_diff_digest": sha256_bytes(binary_diff),
        "changed_path_digest": sha256_bytes(tracked_paths),
        "dirty_untracked_manifest": sha256_bytes(dirty_manifest),
        "test_suite_digest": sha256_bytes(bytes(test_material)),
    }
    candidate = dict(components)
    candidate["identity_digest"] = canonical_digest(components)
    candidate["snapshot_digest"] = canonical_digest(candidate)
    return candidate


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


def run_command(shark, args, cwd):
    try:
        completed = subprocess.run([shark, *args], cwd=cwd, text=True, capture_output=True, check=False)
    except OSError as exc:
        raise RuntimeError(f"unable to execute shark {' '.join(args)}: {exc}") from exc
    if completed.returncode != 0:
        detail = completed.stderr.strip() or completed.stdout.strip()
        raise RuntimeError(f"shark {' '.join(args)} failed ({completed.returncode}): {detail}")
    return load_json(completed.stdout, f"shark {' '.join(args)}")


def write_partial(path, record):
    partial = path.with_suffix(path.suffix + ".partial")
    partial.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")


def scenario_identity(scenario_path, scenario):
    fixture = scenario.get("fixture") or {}
    adapter = scenario.get("adapter") or {}
    version = scenario.get("scenario_version", "1")
    return {
        "schema_version": "1.0",
        "run_id": "",
        "scenario_id": str(scenario.get("scenario_id", scenario_path.stem)),
        "scenario_version": str(version),
        "fixture_id": str(fixture.get("fixture_id", "unknown")),
        "fixture_digest": sha256_file(scenario_path),
        "adapter_id": str(adapter.get("name", "unknown")),
        "adapter_version": str(adapter.get("version", "unknown")),
        "shark_binary_digest": "0" * 64,
        "shark_content_digest": sha256_bytes(git_bytes(Path.cwd(), ["rev-parse", "HEAD^{tree}"])),
        "roots": {
            "agent_fixture_checkout": str(Path(fixture.get("submodule_path", scenario_path.parent)).resolve()),
            "scratch_shark_project": "",
            "evaluator_only": str((scenario_path.parent / "evaluator").resolve()),
        },
    }


def candidate_template():
    raise RuntimeError("candidate_template requires a derived candidate identity")


def make_record(identity, root, scratch, limits):
    return {
        "identity": identity,
        "entity_graph": {
            "root_key": root, "root_type": "unknown", "resolved_via": "scenario_root",
            "fork_candidates": [], "selected_keys": [], "selected_types": [],
            "ordinals": [], "ineligible": [],
        },
        "dispatches": [], "stages": [],
        "workflow_policy": {
            "enabled_gates": [], "gate_order": [],
            "reviewer": {"provider": "unknown", "model": "unknown", "effort": ""},
            "prompt_digest": "0" * 64, "review_bundle_digest": "0" * 64,
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
    return {
        "dispatch_ordinal": dispatch["ordinal"], "stage": dispatch["response"].get("status", "development"),
        "category": "code", "snapshot_digest": candidate["snapshot_digest"],
        "prompt_digest": dispatch["response"].get("prompt_sha256", "0" * 64),
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": {"provider": dispatch["response"].get("provider", "unknown"), "model": dispatch["response"].get("model", "unknown")},
        "cost_usd": dispatch.get("cost_usd", 0.0), "elapsed_seconds": dispatch.get("elapsed_seconds", 0.0),
        "errors": [], "rework": False, "intervals": [], "candidate": candidate,
        "artifacts": [], "access_events": [], "evidence_refs": [],
    }


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


def adapter_result(adapter, request, cwd, mode, shark):
    if mode in {"contract", "dry-run"}:
        return ({"worker_id": "offline-worker", "session_id": request["session_id"], "kind": "final", "recommended_outcome": "pass", "evidence": {"mode": mode}}, [])
    try:
        process = subprocess.Popen([adapter], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, cwd=cwd, text=True)
        process.stdin.write(json.dumps(request, separators=(",", ":")))
        process.stdin.close()
    except OSError as exc:
        raise RuntimeError(f"unable to execute lifecycle adapter: {exc}") from exc
    explicit_interval = os.environ.get("LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS")
    if explicit_interval:
        try:
            heartbeat_interval = max(0.001, float(explicit_interval))
        except ValueError as exc:
            process.terminate()
            raise RuntimeError(f"invalid LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS: {explicit_interval}") from exc
    else:
        try:
            ttl = float(os.environ.get("SHARK_CLAIM_TTL_SECONDS", "180"))
        except ValueError as exc:
            process.terminate()
            raise RuntimeError("invalid SHARK_CLAIM_TTL_SECONDS") from exc
        heartbeat_interval = max(1.0, min(60.0, ttl / 3.0))
    last_heartbeat = time.monotonic()
    heartbeat_events = []
    try:
        while process.poll() is None:
            now = time.monotonic()
            if now - last_heartbeat >= heartbeat_interval:
                try:
                    heartbeat = run_command(shark, ["heartbeat", request["entity_key"], "--session", request["session_id"], "--progress", "0.5", "--note", str(request.get("runner_id", "lifecycle"))], cwd)
                    heartbeat_events.append({"session_id": request["session_id"], "at": timestamp(), "response": bounded(heartbeat)})
                except RuntimeError as exc:
                    process.terminate()
                    process.wait(timeout=2)
                    raise LeaseLoss(f"heartbeat failed for {request['entity_key']}: {exc}") from exc
                last_heartbeat = now
            time.sleep(min(0.05, heartbeat_interval / 4.0))
        stdout = process.stdout.read()
        stderr = process.stderr.read()
        process.wait()
    except (BrokenPipeError, OSError, subprocess.TimeoutExpired) as exc:
        if process.poll() is None:
            process.terminate()
        raise RuntimeError(f"lifecycle adapter process failed: {exc}") from exc
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
    )
    run_command(shark, ["link", question_key, entity, "--type", "question_blocks"], cwd)
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
    default_output = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "runs" / args["run_id"] / "lifecycle.jsonl"
    output = Path(args["output"]).resolve() if args["output"] else Path(os.environ.get("LIFECYCLE_OUTPUT", str(default_output))).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    limits = limits_from(scenario, args["limits"])
    shark = os.environ.get("SHARK_BIN", "shark")
    resolved_shark = shutil.which(shark)
    if not resolved_shark:
        raise RuntimeError(f"shark executable not found on PATH: {shark}")
    adapter = os.environ.get("LIFECYCLE_ADAPTER") or os.environ.get("LIFECYCLE_ADAPTER_PATH", "")
    if args["mode"] not in {"contract", "dry-run"} and not adapter:
        raise RuntimeError("LIFECYCLE_ADAPTER is required for live lifecycle runs")

    identity = scenario_identity(scenario_path, scenario)
    identity["run_id"] = args["run_id"]
    identity["roots"]["scratch_shark_project"] = str(scratch)
    identity["shark_binary_digest"] = sha256_file(Path(resolved_shark))
    candidate = candidate_identity(Path.cwd())
    record = make_record(identity, args["root"], scratch, limits)
    record["entity_graph"]["root_type"] = str(scenario.get("entity_family", "unknown"))
    record["workflow_policy"]["reviewer"] = {"provider": "fixture", "model": "fixture", "effort": ""}
    ordinal = 0
    generated = 0
    started = time.monotonic()
    queue = [args["root"]]
    processed = set()
    terminal = "complete"
    reason = "all eligible dispatches completed"
    active_lease = {"entity": "", "session": ""}

    def release_on_signal(signum, _frame):
        if active_lease["session"]:
            try:
                run_command(shark, ["release", active_lease["entity"], "--session", active_lease["session"], "--outcome", "cancellation"], scratch)
            except RuntimeError:
                pass
        raise SystemExit(128 + signum)

    signal.signal(signal.SIGINT, release_on_signal)
    signal.signal(signal.SIGTERM, release_on_signal)

    try:
        while queue:
            requested = queue.pop(0)
            if requested in processed:
                continue
            processed.add(requested)
            prompt_path = scratch / "prompts" / f"{ordinal + 1:04d}"
            prompt_path.parent.mkdir(parents=True, exist_ok=True)
            response = run_command(shark, ["next", requested, "--json", "--prompt-out", str(prompt_path)], scratch)
            if response.get("action") == "parallel_candidates":
                candidates = fork_candidates(response)
                record["entity_graph"]["fork_candidates"].append({"response": bounded(response), "candidates": candidates})
                record["entity_graph"]["selected_keys"].extend(item["entity_key"] for item in candidates)
                record["entity_graph"]["selected_types"].extend(item["entity_type"] for item in candidates)
                record["entity_graph"]["resolved_via"] = "fork_response"
                queue[0:0] = [item["entity_key"] for item in candidates]
                queue.sort()
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

            response_record = bounded(response)
            response_record.pop("prompt", None)
            dispatch = {"ordinal": ordinal + 1, "requested_key": requested, "response": response_record, "claim": {}, "worker": {}, "heartbeats": [], "outcome": "error", "transition": {}, "release": {}, "started_at": timestamp(), "ended_at": "", "evidence_refs": {"prompt_sha256": expected_digest, "prompt_bytes": expected_bytes, "candidate_snapshot_digest": candidate["snapshot_digest"]}}
            record["dispatches"].append(dispatch)
            record["entity_graph"]["ordinals"].append(dispatch["ordinal"])
            if requested != args["root"]:
                generated += 1
            session = ""
            worker_result = {}
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
                worker_result, heartbeats = adapter_result(adapter, request, scratch, args["mode"], shark)
                dispatch["heartbeats"] = heartbeats
                if worker_result.get("session_id") not in {None, session}:
                    raise RuntimeError(f"worker session mismatch for {entity}")
                dispatch["worker"] = {"worker_id": worker_result.get("worker_id", ""), "session_id": worker_result.get("session_id", session), "kind": worker_result.get("kind", ""), "recommended_outcome": worker_result.get("recommended_outcome"), "evidence": bounded(worker_result.get("evidence", {}))}
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
                    run_command(shark, ["status", "advance", entity, "--outcome", str(outcome), "--session", session, "--from-status", str(response.get("status", "")), "--agent", f"{response.get('agent_type', '')}@{response.get('provider', '')}"], scratch)
                    dispatch["transition"] = {"outcome": str(outcome), "session_id": session, "from_status": response.get("status", "")}
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
                        dispatch["release"] = bounded(run_command(shark, ["release", entity, "--session", session, "--outcome", dispatch["outcome"]], scratch))
                    except RuntimeError as exc:
                        dispatch["release"] = {"error": str(exc)}
                        if terminal == "complete":
                            terminal = "error"
                            reason = str(exc)
                    active_lease["entity"] = ""
                    active_lease["session"] = ""
            dispatch["ended_at"] = timestamp()
            elapsed = max(0.0, time.monotonic() - started)
            cost = float(worker_result.get("cost_usd", (worker_result.get("usage") or {}).get("cost_usd", 0.0)) or 0.0)
            dispatch["cost_usd"] = cost
            record["limits"]["observed_cost_usd"] += cost
            record["limits"]["observed_wall_clock_seconds"] = elapsed
            record["limits"]["observed_generated_tasks"] = generated
            stage_candidate = dict(candidate)
            dispatch["evidence_refs"]["candidate_snapshot_digest"] = stage_candidate["snapshot_digest"]
            record["stages"].append(stage_record(dispatch, stage_candidate))
            ordinal += 1
            write_partial(output, record)
            if terminal != "complete" and terminal != "resource_limit":
                break
            exceeded = None
            if record["limits"]["observed_cost_usd"] > limits["max_cost_usd"]:
                exceeded = "max_cost_usd"
            elif elapsed > limits["max_wall_clock_seconds"]:
                exceeded = "max_wall_clock_seconds"
            elif generated > limits["max_generated_tasks"]:
                exceeded = "max_generated_tasks"
            if exceeded:
                terminal = "resource_limit"
                record["limits"]["first_exceeded"] = exceeded
                reason = f"resource ceiling exceeded: {exceeded}"
                break
        record["outcome"] = {"terminal": terminal, "reason": "" if terminal == "complete" else reason, "partial_evidence": bool(record["dispatches"]), "publication_eligible": terminal == "complete"}
    except (RuntimeError, OSError, ValueError, TypeError) as exc:
        terminal = "error"
        reason = str(exc)
        record["outcome"] = {"terminal": terminal, "reason": reason, "partial_evidence": bool(record["dispatches"]), "publication_eligible": False}
    output.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
    output.with_suffix(output.suffix + ".partial").unlink(missing_ok=True)
    if terminal != "complete" and terminal != "resource_limit":
        return 1
    return 0


try:
    raise SystemExit(main(sys.argv[1:]))
except (RuntimeError, OSError, ValueError, TypeError) as exc:
    print(f"run-lifecycle: {exc}", file=sys.stderr)
    raise SystemExit(1)
PY
