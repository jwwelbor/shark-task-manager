#!/usr/bin/env bash
# run-lifecycle.sh --scenario <package.yaml> --run-id <id> --root <key>
#                    --scratch-root <dir> [--output <lifecycle.jsonl>]
#                    [--limits <policy.yaml>] [--i05-bundle-dir <dir>]
#                    [--replay <i06-result.json>] [--prelude <prelude.jsonl>]
#                    [--mode contract|dry-run|resolve-route]
#
# Host-side F08 controller. Shark remains the owner of prompt assembly,
# claims, leases, workflow routing, and Question state; this script only
# drives the public keyed command sequence and records bounded evidence.
#
# --i05-bundle-dir (T-E40-F12-001, spec.md REQ-F-001): the I-05 evidence
# bundle directory. Required in --mode live (the run fails before the first
# dispatch when absent); optional in --mode contract/dry-run (no-op when
# absent, matching pre-existing offline behavior); rejected in
# --mode resolve-route (exits 2, never writes) since route resolution is
# already fenced off from live stage evidence (REQ-F-001).
#
# --mode resolve-route (T-E40-F11-007, spec.md REQ-F-005/§2.3.1): traces the
# configured success route (every outcome resolved as "pass") for a scenario
# without requiring any artifact only a live worker can produce -- no
# candidate/scratch-content digests, no prompt-out byte verification, no
# resource-ceiling enforcement. Writes a single JSONL record carrying
# `evidence_mode: "route_resolution_only"` so it can never be mistaken for
# live I-05 stage evidence (AC-F11-15; enforced by verify-stage-evidence.sh).
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

sys.path.insert(0, os.path.join(os.environ["LIFECYCLE_BENCH_DIR"], "scripts", "lib"))
from e40_evidence import CANDIDATE_IDENTITY_FIELDS, canonical_digest, sha256_bytes, sha256_file  # noqa: E402

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


MAX_TRANSITION_REJECTIONS_PER_STAGE = 2


def usage():
    print(
        "usage: run-lifecycle.sh --scenario <package.yaml> --run-id <id> "
        "--root <key> --scratch-root <dir> [--output <path>] "
        "[--limits <policy.yaml>] [--i05-bundle-dir <dir>] "
        "[--fixture-root <dir>] "
        "[--replay <i06-result.json>] [--prelude <prelude.jsonl>] "
        "[--mode contract|dry-run|resolve-route]",
        file=sys.stderr,
    )
    raise SystemExit(2)


def parse_args(argv):
    values = {"mode": "live", "output": "", "limits": "", "i05_bundle_dir": "", "prelude": "", "replay": "", "fixture_root": ""}
    required = {"--scenario": "scenario", "--run-id": "run_id", "--root": "root", "--scratch-root": "scratch_root"}
    optional = {"--output", "--limits", "--mode", "--i05-bundle-dir", "--prelude", "--replay", "--fixture-root"}
    index = 0
    while index < len(argv):
        option = argv[index]
        if option in required or option in optional:
            if index + 1 >= len(argv):
                usage()
            values[required.get(option, option[2:].replace("-", "_") or "mode")] = argv[index + 1]
            index += 2
            continue
        usage()
    if any(not values.get(name) for name in required.values()):
        usage()
    if values["mode"] not in {"live", "contract", "dry-run", "resolve-route"}:
        print(f"run-lifecycle: unsupported mode: {values['mode']}", file=sys.stderr)
        raise SystemExit(2)
    # REQ-F-001: the bundle directory is required in --mode live (fail before
    # the first dispatch, never run and silently produce no evidence) and
    # rejected outright in --mode resolve-route (route resolution stays
    # fenced off from live stage evidence at the producer end too). This
    # check runs entirely inside parse_args(), before any `shark` binary is
    # resolved or invoked, so a missing/rejected option never reaches the
    # dispatch loop.
    if values["mode"] == "live" and not values["i05_bundle_dir"]:
        print("run-lifecycle: --i05-bundle-dir is required for --mode live", file=sys.stderr)
        raise SystemExit(2)
    if values["mode"] == "resolve-route" and values["i05_bundle_dir"]:
        print("run-lifecycle: --i05-bundle-dir is not supported with --mode resolve-route", file=sys.stderr)
        raise SystemExit(2)
    return values


def stage_snapshot_digest(snapshot):
    """REQ-F-003/REQ-F-015 `snapshot_digest`. NOT `canonical_digest()`'s own
    compact form despite spec.md's prose naming it: the real, unmodified,
    already-shipped `replay-stage-evidence.sh` (REQ-F-016's actual
    acceptance mechanism for this field -- verify-stage-evidence.sh carries
    no snapshot_digest check at all, spec.md Spec Drift item 2) computes
    `recompute_snapshot_digest()` as `"sha256:" +
    sha256(json.dumps(payload, sort_keys=True))` using Python's DEFAULT
    separators/ensure_ascii, not canonical_digest()'s compact,
    ensure_ascii=False form. Matching spec.md's prose instead of the real
    script's code would make every produced snapshot fail replay -- this
    function matches the real script byte-for-byte instead."""
    payload = {key: value for key, value in snapshot.items() if key != "snapshot_digest"}
    return "sha256:" + sha256_bytes(json.dumps(payload, sort_keys=True).encode("utf-8"))


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
    test_identity_error=None,
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
    if test_identity_error is not None:
        test_ids = []
    else:
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
    test_identity_error=None,
):
    # The stage snapshot is a post-dispatch observation. Re-derive every
    # fixture identity field after the worker returns so an edit made by this
    # stage cannot first appear in the following stage's evidence.
    current = candidate_identity(
        fixture_root, execution_adapter, adapter_name, adapter_version,
        toolchain_identity, deadline, active_state, test_identity_error,
    )
    candidate.clear()
    candidate.update(current)
    candidate["scratch_content_digest"] = scratch_content_digest(scratch)
    # scratch_content_digest is only knowable after the worker has returned.
    # Hash the canonical seven-field identity only after it is present.
    candidate["identity_digest"] = canonical_digest({field: candidate[field] for field in CANDIDATE_IDENTITY_FIELDS})
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


def is_repairable_research_rejection(error, response, entity):
    """Identify Shark's research-evidence validator failure contract."""
    if str(response.get("status", "")) != "research":
        return False
    entity_type = str(response.get("entity_type", ""))
    marker = f"cannot advance {entity_type} {entity} from research:"
    return bool(entity_type) and marker in str(error)


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


def content_digest(root):
    """T-E40-F12-005/REQ-F-012 (ADR-F12-03): byte-for-byte identical to
    `evaluate-lifecycle.sh`'s own `content_digest()` (bench/scripts/
    evaluate-lifecycle.sh:102) -- same traversal order, same separator
    bytes -- so the two independently-computed digests can ever agree.
    Duplicated rather than imported: each of run-lifecycle.sh and
    evaluate-lifecycle.sh is a single embedded-Python heredoc with no
    importable module boundary between them. Returns None when `root` is
    not a directory, matching the real function's own contract."""
    root = os.path.abspath(root)
    if not os.path.isdir(root):
        return None
    hasher = hashlib.sha256()
    for current, directories, files in os.walk(root, followlinks=False):
        directories.sort()
        files.sort()
        for name in files:
            path = os.path.join(current, name)
            relative = os.path.relpath(path, root).replace(os.sep, "/")
            hasher.update(relative.encode("utf-8")); hasher.update(b"\0")
            with open(path, "rb") as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                    hasher.update(chunk)
            hasher.update(b"\0")
    return hasher.hexdigest()


def scenario_identity(scenario_path, scenario, scratch, fixture_root, limits):
    fixture = scenario.get("fixture") or {}
    adapter = scenario.get("adapter") or {}
    version = scenario.get("scenario_version", "1")
    # REQ-F-012/ADR-F12-03 (TD-132): content_root is the installed Shark-data
    # canonical content tree INSIDE the scratch Shark project -- the
    # `shark admin install-shark-data` default install location relative to
    # the project root -- never the repo's own
    # internal/sharkdata/default_data path and never the legacy whole-repo
    # `git rev-parse HEAD^{tree}` hash. A half-wiring that declares
    # content_root while keeping the legacy digest is forbidden (spec.md
    # sec 1.4): shark_content_digest MUST be the walk-based digest of this
    # same content_root, computed by the same scheme evaluate-lifecycle.sh's
    # content_digest() uses, so the crosscheck can ever agree. When
    # shark-data has not been installed into this scratch project,
    # content_digest() returns None (not a directory yet) and
    # shark_content_digest falls back to the digest of empty bytes -- never
    # to the retired whole-repo scheme -- mirroring scratch_content_digest()'s
    # own "root does not exist" convention above.
    content_root = str((Path(scratch) / "shark-data").resolve())
    content_digest_value = content_digest(content_root) or sha256_bytes(b"")
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
        "content_root": content_root,
        "content_digest_scheme": "walk_v1",
        "shark_content_digest": content_digest_value,
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
        "category": "code", "snapshot_digest": candidate["snapshot_digest"],
        "prompt_digest": dispatch["response"].get("prompt_sha256", "0" * 64),
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": usage,
        "cost_usd": dispatch.get("cost_usd", 0.0), "elapsed_seconds": dispatch.get("elapsed_seconds", 0.0),
        "errors": [], "rework": False, "intervals": [], "candidate": candidate,
        "artifacts": [], "access_events": [],
        "evidence_refs": {"candidate_snapshot_digest": candidate["snapshot_digest"]},
    }


I05_OWNED_ENTRIES = ("bundle.json", "stages", "access.jsonl", "transcripts")

# spec.md §2.3: `shark next`'s `entity_type` maps to a
# `shark admin workflow list <level>` level name unchanged, with one
# normalization -- tech-debt -> tech_debt.
WORKFLOW_LEVEL_NORMALIZE = {"tech-debt": "tech_debt"}


def prelude_lineage(prelude):
    """Project resolver-owned replay join keys before bounding the prelude."""
    if not isinstance(prelude, dict):
        raise RuntimeError("lifecycle prelude must be a JSON object")
    replay = prelude.get("replay")
    if replay is None:
        replay = {}
    if not isinstance(replay, dict):
        raise RuntimeError("lifecycle prelude replay must be a JSON object")
    replay_bundle = replay.get("replay_bundle")
    if replay_bundle is None:
        replay_bundle = {}
    if not isinstance(replay_bundle, dict):
        raise RuntimeError("lifecycle prelude replay_bundle must be a JSON object")
    reference = replay_bundle.get("bundle_path")
    lineage = []
    for stage in replay.get("stages") or []:
        if not isinstance(stage, dict):
            continue
        # I-06 REQ-F-009 assigns the resolver-owned stage ledger as the
        # authoritative source. Artifact claims are optional and must not
        # create I-05 replay lineage.
        for entry in stage.get("consumed_entries") or []:
            if isinstance(entry, dict) and entry.get("entry_digest"):
                lineage.append({
                    "replay_reference": reference,
                    "entry_digest": entry["entry_digest"],
                })
    return sorted(lineage, key=lambda item: (str(item["replay_reference"]), str(item["entry_digest"])))


def workflow_level_for(entity_type):
    return WORKFLOW_LEVEL_NORMALIZE.get(entity_type, entity_type)


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


_STAGE_CATEGORY_MAP_CACHE = {}


def stage_category_map():
    """REQ-F-004/ADR-F12-04: the closed phase -> stage_category table lives
    in bench/evidence/stage-category-map.yaml, never embedded here. Loaded
    once per run and cached."""
    if "phases" not in _STAGE_CATEGORY_MAP_CACHE:
        map_path = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "evidence" / "stage-category-map.yaml"
        try:
            data = yaml.safe_load(map_path.read_text(encoding="utf-8")) or {}
        except (OSError, yaml.YAMLError) as exc:
            raise RuntimeError(f"cannot read stage-category map {map_path}: {exc}") from exc
        phases = data.get("phases")
        if not isinstance(phases, dict):
            raise RuntimeError(f"stage-category map {map_path} is missing a phases table")
        _STAGE_CATEGORY_MAP_CACHE["phases"] = phases
    return _STAGE_CATEGORY_MAP_CACHE["phases"]


_USAGE_MAPPING_CACHE = {}


def mapped_provider(response):
    """A dispatched step's `response["provider"]` is the short label the
    workflow step config declares (e.g. "anthropic"), not the specific
    usage-mapping.yaml provider key it corresponds to (e.g.
    "anthropic_claude_cli") -- normalize before calling `resolve_usage()`,
    never look up the bare label directly."""
    provider = str(response.get("provider", "")).lower()
    return "anthropic_claude_cli" if "anthropic" in provider or "claude" in provider else provider


def usage_mapping_providers():
    """X-09/ADR-F06-04: usage-mapping.yaml is the single owner of the
    semantic-slot -> envelope-path bindings. Loaded once per run and
    cached; never a hard-coded envelope path in this producer."""
    if "providers" not in _USAGE_MAPPING_CACHE:
        map_path = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "evidence" / "usage-mapping.yaml"
        try:
            data = yaml.safe_load(map_path.read_text(encoding="utf-8")) or {}
        except (OSError, yaml.YAMLError) as exc:
            raise RuntimeError(f"cannot read usage mapping {map_path}: {exc}") from exc
        providers = data.get("providers")
        if not isinstance(providers, dict):
            raise RuntimeError(f"usage mapping {map_path} is missing a providers table")
        _USAGE_MAPPING_CACHE["providers"] = providers
    return _USAGE_MAPPING_CACHE["providers"]


def _envelope_lookup(envelope, path):
    """Resolve a dotted usage-mapping.yaml `envelope_path` against the
    worker envelope, or the one special-cased expression the mapping itself
    declares for `model_ids` (`bench/evidence/usage-mapping.yaml`). Returns
    (value, found)."""
    if path == "sorted(modelUsage keys)":
        model_usage = envelope.get("modelUsage")
        if not isinstance(model_usage, dict):
            return None, False
        return sorted(model_usage.keys()), True
    current = envelope
    for part in path.split("."):
        if not isinstance(current, dict) or part not in current:
            return None, False
        current = current[part]
    return current, True


def resolve_usage(provider_name, envelope):
    """REQ-F-003/X-09: populate the snapshot's `usage` block by semantic
    slot name through bench/evidence/usage-mapping.yaml, never by a
    hard-coded envelope path. A provider absent from the mapping (or
    declared `status: unmapped`, e.g. openai_codex_cli today) yields an
    empty usage block plus one `unmapped_provider` error -- never a
    per-slot guess. For a mapped provider, a slot whose envelope_path
    cannot be resolved is omitted from `usage` (never zero, never null)
    with a matching `usage_slot_unavailable` error naming the slot and
    path."""
    providers = usage_mapping_providers()
    block = providers.get(provider_name) if provider_name else None
    if not isinstance(block, dict) or block.get("status") != "mapped":
        return {}, [{"kind": "unmapped_provider", "provider": provider_name or ""}]
    usage = {}
    errors = []
    for slot, binding in sorted((block.get("slots") or {}).items()):
        path = binding.get("envelope_path") if isinstance(binding, dict) else None
        if not path:
            continue
        value, found = _envelope_lookup(envelope, path)
        if not found or value in (None, "", []):
            errors.append({"kind": "usage_slot_unavailable", "slot": slot, "envelope_path": path})
            continue
        usage[slot] = value
    return usage, errors


def provider_usage_envelope(worker_result):
    """Select the bounded provider-envelope evidence when an adapter supplied
    it, retaining the legacy direct-envelope route for existing adapters.

    The adapter's normalized ``usage`` object intentionally has different
    field names from usage-mapping.yaml. Mapping must therefore consume its
    measurement-only ``provider_usage_envelope`` rather than silently treating
    every mapped raw-envelope slot as unavailable.
    """
    if not isinstance(worker_result, dict):
        return {}
    envelope = worker_result.get("provider_usage_envelope")
    return envelope if isinstance(envelope, dict) else worker_result


def test_suite_reference(repo_root):
    """AC-006's two replay-guard fields beyond REQ-F-006's six candidate
    identity fields (`replay-stage-evidence.sh` reads both):
    `test_suite_ids` and `test_suite_dir`. The candidate's test corpus is
    Shark's own `*_test.go` files -- the same `git ls-files` query
    `candidate_identity()` already uses for `test_suite_digest` (same
    plumbing command, called independently here so `candidate_identity()`
    itself stays unmodified/reused verbatim). Unlike a scenario's single
    Python `tests/` directory, Go tests are not confined to one directory,
    so ids are file-level (one id per test file, not per test function) and
    `test_suite_dir` is the longest common directory prefix of those files
    -- "." when they share none."""
    test_paths = sorted(filter(None, git_bytes(repo_root, ["ls-files", "-z", "--", "*_test.go", "**/*_test.go"]).decode().split("\0")))
    if not test_paths:
        return [], "."
    if len(test_paths) == 1:
        common = os.path.dirname(test_paths[0])
    else:
        common = os.path.commonpath(test_paths)
    return test_paths, (common or ".")


def i05_schema_version():
    """REQ-F-002: `schema_version` is read from bench/evidence/i05-schema.yaml
    at run time, never hard-coded, so the producer cannot silently drift from
    a schema bump."""
    schema_path = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "evidence" / "i05-schema.yaml"
    try:
        schema = yaml.safe_load(schema_path.read_text(encoding="utf-8")) or {}
    except (OSError, yaml.YAMLError) as exc:
        raise RuntimeError(f"cannot read I-05 schema {schema_path}: {exc}") from exc
    version = schema.get("schema_version")
    if not isinstance(version, str) or not version:
        raise RuntimeError(f"I-05 schema {schema_path} is missing schema_version")
    return version


# T-E40-F12-003 (REQ-F-005/§2.5): one additive, bounded heartbeat retry
# before declaring LeaseLoss. Existing behavior (immediate LeaseLoss on
# heartbeat failure) becomes "retry once after this backoff, THEN LeaseLoss
# on a second failure" -- the backoff+retry window is exactly what §2.5's
# closed table calls `retry_or_backoff` ("only on a heartbeat retry").
HEARTBEAT_RETRY_BACKOFF_SECONDS = 0.05

# The six categories verify-stage-evidence.sh's validate_time_ledger()
# accepts, per bench/evidence/i05-schema.yaml's interval_category vocabulary.
# Pinned here (not read from the schema at call time) so every emitted
# time_ledger always carries a fully-populated `intervals` object -- an
# absent key is indistinguishable from "never happened" to a reader, but an
# empty list is an honest "observed, zero occurrences."
TIME_LEDGER_CATEGORIES = (
    "provider_active", "tool_and_test", "queue_or_claim_wait",
    "replay_or_human_gate_wait", "retry_or_backoff", "unclassified",
)


def _subtract_claimed(interval, claimed_union):
    """Return the pieces of `interval` (a (start, end) tuple, integer ns)
    not covered by any (start, end) span in `claimed_union`. Used to keep an
    untrusted, envelope-reported `provider_active` interval (T-E40-F12-003,
    AC-009 negative case) from ever overlapping a category the driver itself
    already observed -- driver-observed windows always take priority over
    whatever the envelope claims."""
    pieces = [interval]
    for claimed_start, claimed_end in claimed_union:
        next_pieces = []
        for start, end in pieces:
            if claimed_end <= start or claimed_start >= end:
                next_pieces.append((start, end))
                continue
            if claimed_start > start:
                next_pieces.append((start, claimed_start))
            if claimed_end < end:
                next_pieces.append((claimed_end, end))
        pieces = next_pieces
    return [(start, end) for start, end in pieces if end > start]


def provider_active_claims(worker_envelope, adapter_start_ns, adapter_end_ns, claimed_so_far):
    """T-E40-F12-003 (REQ-F-005, ADR-F12-05): read explicit `provider_active`
    intervals from the worker envelope's TOP LEVEL `time_ledger` block (the
    "envelope placement note", spec.md sec 2.5 -- never from `evidence`,
    which the adapter's SAFE_EVIDENCE_KEYS would silently strip). This is
    the producer-side half of a contract the adapter does not implement yet
    (spec.md sec 1.4 item 6): each `[start_ns, end_ns]` pair is nanosecond
    offsets relative to the adapter subprocess's own spawn instant
    (`adapter_start_ns`), converted here to the run's shared monotonic
    timeline and clipped to `[adapter_start_ns, adapter_end_ns]` -- the
    real, driver-observed bound on anything the subprocess could have
    reported, which also means a reported interval can never encroach on a
    category recorded after `adapter_result()` returns (release,
    `refresh_candidate()`). Any piece already covered by a category the
    driver itself observed (`claimed_so_far`) is subtracted first, never
    double-claimed. Returns a list of (start, end) tuples in `provider_active`
    (never `[]` mutated into a fabricated non-empty entry -- absent/invalid
    envelope data yields an empty list, per ADR-F12-05: unattributable time
    is never assigned to `provider_active`)."""
    ledger = worker_envelope.get("time_ledger") if isinstance(worker_envelope, dict) else None
    raw = ledger.get("provider_active") if isinstance(ledger, dict) else None
    if not isinstance(raw, list):
        return []
    claimed_union = []
    for _category, start, end in sorted(claimed_so_far, key=lambda item: item[1]):
        if claimed_union and start <= claimed_union[-1][1]:
            claimed_union[-1] = (claimed_union[-1][0], max(claimed_union[-1][1], end))
        else:
            claimed_union.append((start, end))
    claims = []
    for pair in raw:
        if not (isinstance(pair, list) and len(pair) == 2 and all(isinstance(value, (int, float)) for value in pair)):
            continue
        start = adapter_start_ns + int(pair[0])
        end = adapter_start_ns + int(pair[1])
        start = max(start, adapter_start_ns)
        end = min(end, adapter_end_ns)
        if end <= start:
            continue
        accepted = _subtract_claimed((start, end), claimed_union)
        claims.extend(accepted)
        # Later windows from this same envelope must not re-claim a fragment
        # already accepted above. Keep the local union normalized just as the
        # driver-observed input union was normalized before this loop.
        for accepted_start, accepted_end in accepted:
            claimed_union.append((accepted_start, accepted_end))
        claimed_union.sort()
        normalized = []
        for claimed_start, claimed_end in claimed_union:
            if normalized and claimed_start <= normalized[-1][1]:
                normalized[-1] = (normalized[-1][0], max(normalized[-1][1], claimed_end))
            else:
                normalized.append((claimed_start, claimed_end))
        claimed_union = normalized
    return claims


def reconcile_time_ledger(origin_ns, timing):
    """T-E40-F12-003 (REQ-F-005, AC-008): build one snapshot's `time_ledger`
    from the real, driver-observed `timing` accumulator
    (`{"stage_start": ns, "stage_end": ns, "claimed": [(category, start, end), ...]}`,
    every value an absolute `time.monotonic_ns()` reading) by gap-filling
    every span within `[stage_start, stage_end)` that no claimed category
    covers into `unclassified`. This is the ONLY mechanism that can
    guarantee `verify-stage-evidence.sh`'s reconciliation invariant
    (`residual_ns <= reconciliation_epsilon_ns`) regardless of what did or
    did not happen on a given dispatch (a failed claim, a worker failure
    before any envelope existed, ...): every real path still calls this
    with a valid stage window, and whatever real time it cannot name a
    category for becomes an honest `unclassified` span rather than a
    reconciliation failure. `origin_ns` is subtracted once, here, so every
    timestamp in the emitted ledger is relative to a single run-scoped
    origin (spec.md sec 2.5), matching every OTHER stage's ledger in the
    same bundle."""
    stage_start = timing["stage_start"] - origin_ns
    stage_end = timing["stage_end"] - origin_ns
    if stage_end <= stage_start:
        # Defensive only: real work always takes >0ns. Never let a
        # collapsed window reach validate_time_ledger() as a ScriptError
        # (stage_end <= stage_start) instead of a real, if trivial, ledger.
        stage_end = stage_start + 1
    claimed = []
    for category, start, end in timing["claimed"]:
        start = max(start - origin_ns, stage_start)
        end = min(end - origin_ns, stage_end)
        if end > start:
            claimed.append((category, start, end))
    intervals = {name: [] for name in TIME_LEDGER_CATEGORIES}
    for category, start, end in claimed:
        intervals[category].append([start, end])
    ordered = sorted(claimed, key=lambda item: (item[1], item[2]))
    cursor = stage_start
    for _category, start, end in ordered:
        if start > cursor:
            intervals["unclassified"].append([cursor, start])
        cursor = max(cursor, end)
    if cursor < stage_end:
        intervals["unclassified"].append([cursor, stage_end])
    return {
        "stage_start": stage_start,
        "stage_end": stage_end,
        "reconciliation_epsilon_ns": 1_000_000,
        "intervals": intervals,
    }


def stop_outcome_triad(terminal, reason):
    """T-E40-F12-004 (REQ-F-009, spec.md sec 2.6): the closed 11-row
    stop-outcome -> eligibility table, as a single, named, top-level
    function -- mirroring `content_digest()`'s and `canonical_digest()`'s
    shape -- so TC-014's source-extraction technique (the one
    `tc075_content-identity_x12_test.sh` already established for
    `content_digest()`) has a clean, unindented boundary to regex against.
    `run-lifecycle.sh` is a single embedded-Python script with no
    importable module, so this is the real seam: no direct `import` is
    possible, and the logic must not stay inlined in `main()`'s loop body
    with no named function boundary.

    `terminal` is the same value the run's own I-07 `outcome.terminal`
    carries; `reason` is that record's own `outcome.reason` -- already
    carrying the row-specific name (the ceiling, the entity, the blocking
    gate, the routed Question key, the archived entity, the signal, or the
    exceeded window) from its own call site, never re-derived here. Returns
    `(stop_outcome, publication_eligible, ineligibility_reasons)`."""
    stop_outcome = terminal if terminal in STOP_OUTCOMES else None
    publication_eligible = terminal == "complete"
    ineligibility_reasons = [] if publication_eligible else [str(reason or terminal)]
    return stop_outcome, publication_eligible, ineligibility_reasons


class I05BundleWriter:
    """T-E40-F12-001/002 (ADR-F12-01): the live I-05 evidence bundle
    producer, living inside run-lifecycle.sh's own dispatch loop rather than
    a shared library. Owns the `bundle.json` triad (REQ-F-002: written before
    the first dispatch, rewritten after every dispatch, rewritten at
    termination), the per-dispatch stage index this task's own tests
    reconcile against real snapshot files (AC-003), the producer-owned reset
    bounded to its four owned entries (REQ-F-011), REQ-F-014's additive
    join-field emission (`scenario_id`, `scenario_version`, `dispatches`)
    read live from the same in-process `identity`/`record` objects the loop
    already holds -- never re-derived -- and, as of T-E40-F12-002, the
    stage-snapshot writer itself: `stage_category` (REQ-F-004, via the
    closed `stage-category-map.yaml` table), `usage` (REQ-F-003/X-09, via
    `usage-mapping.yaml`), `candidate` (REQ-F-006, `code`/`review` only),
    per-snapshot `errors[]`, and `access.jsonl` (REQ-F-006).

    As of T-E40-F12-003: `time_ledger`'s real monotonic instrumentation
    (REQ-F-005, via `reconcile_time_ledger()`) and `transcripts/`
    materialization (REQ-F-007). `origin_ns` is captured once, here, at
    writer construction (before the dispatch loop starts) as the run's
    single shared `time.monotonic_ns()` origin every stage's `time_ledger`
    is relative to (spec.md sec 2.5). The `roots` triad (REQ-F-008) needed
    no change: `scenario_identity()` already resolves
    `agent_fixture_checkout` to a real, existing directory.

    As of T-E40-F12-004: the full §2.6 stop-outcome eligibility triad
    (`finalize()`, via the top-level `stop_outcome_triad()`), wired into
    both the normal termination path (`main()`) and the signal-termination
    path (`release_on_signal()`), and fail-loud writes for every
    bundle-artifact write this class owns.

    As of T-E40-F12-005: `content_root`/`content_digest_scheme` and the
    walk-based `shark_content_digest` scheme (TD-132) are populated by
    `scenario_identity()` before this class is constructed; no change to
    this class itself was needed (the `roots` triad note above already
    explains why).
    """

    def __init__(self, bundle_dir, scenario_path, scenario, identity, run_id, record):
        self.dir = Path(bundle_dir).resolve()
        self.scenario_path = scenario_path
        self.scenario = scenario
        self.identity = identity
        self.run_id = run_id
        self.record = record
        self.stages_dir = self.dir / "stages"
        self.transcripts_dir = self.dir / "transcripts"
        self.origin_ns = time.monotonic_ns()
        self._stages_index = []
        self._phase_cache = {}
        self._stage_visit_counts = {}
        self._prelude_lineage = []
        self.dir.mkdir(parents=True, exist_ok=True)
        self._refuse_symlinks()
        self._reset_owned_entries()
        self.stages_dir.mkdir(parents=True, exist_ok=True)
        # REQ-F-007: a real directory (never a symlink -- _refuse_symlinks()
        # above already checked), created up front so it exists even for a
        # zero-dispatch run (retain_pair's own transcripts/ requirement,
        # research-report Finding 6, is unconditional on dispatch count).
        self.transcripts_dir.mkdir(parents=True, exist_ok=True)
        self._create_access_log()
        self._write_bundle(reached=False, reached_at=None, stop_outcome=None, publication_eligible=True, ineligibility_reasons=())

    def set_prelude_lineage(self, lineage):
        """Retain the typed replay projection, never the unbounded prelude."""
        self._prelude_lineage = list(lineage)

    def _refuse_symlinks(self):
        for name in I05_OWNED_ENTRIES:
            path = self.dir / name
            if path.is_symlink():
                raise RuntimeError(f"i05 bundle dir entry is a symlink, refusing to reset it: {path}")

    def _reset_owned_entries(self):
        """REQ-F-011: remove exactly the four producer-owned entries, never
        the directory itself and never any other entry inside it."""
        for name in I05_OWNED_ENTRIES:
            path = self.dir / name
            if not path.exists():
                continue
            if path.is_dir():
                shutil.rmtree(path)
            else:
                path.unlink()

    def _create_access_log(self):
        """REQ-F-006: `access.jsonl` is created at run start (after the
        reset above) so it exists even for a run with zero evaluator
        access, and is only ever appended to afterward."""
        access_path = self.dir / "access.jsonl"
        try:
            access_path.touch(exist_ok=True)
        except OSError as exc:
            raise RuntimeError(f"failed to create I-05 access.jsonl {access_path}: {exc}") from exc

    def _append_access_events(self, events):
        """REQ-F-006 negative case: any `evaluator_access` entry a snapshot
        carries is also appended here, verbatim. This producer never
        performs evaluator access during dispatch (REQ-F-006's own
        rationale), so `events` is empty on every real run today; this
        exists so a snapshot that ever does carry one is never silently
        dropped from the bundle-level log."""
        if not events:
            return
        access_path = self.dir / "access.jsonl"
        try:
            with access_path.open("a", encoding="utf-8") as handle:
                for event in events:
                    handle.write(json.dumps(event, sort_keys=True, separators=(",", ":"), ensure_ascii=False) + "\n")
        except OSError as exc:
            raise RuntimeError(f"failed to append I-05 access.jsonl {access_path}: {exc}") from exc

    def _workflow_statuses(self, shark, cwd, level):
        """ADR-F12-04: `shark admin workflow list <level> --json`, invoked
        at most once per workflow level per run and cached in-process."""
        if level not in self._phase_cache:
            response = run_command(shark, ["admin", "workflow", "list", level, "--json"], cwd)
            statuses = {}
            for entry in response.get("levels", []):
                if entry.get("level") != level:
                    continue
                for status in entry.get("statuses", []):
                    name = status.get("name")
                    if isinstance(name, str) and name:
                        statuses[name] = status.get("phase", "") or ""
            self._phase_cache[level] = statuses
        return self._phase_cache[level]

    def _stage_category_for(self, shark, cwd, entity_type, status_name):
        """REQ-F-004: resolve `stage_category` through the closed
        `stage-category-map.yaml` table keyed by workflow phase. A status
        with no phase, or a phase absent from the table, yields (None, an
        `unknown_stage_category` error naming the status and phase) --
        never a substituted category."""
        level = workflow_level_for(entity_type)
        phase = self._workflow_statuses(shark, cwd, level).get(status_name, "")
        row = stage_category_map().get(phase) if phase else None
        category = row.get("stage_category") if isinstance(row, dict) else None
        if not phase or row is None or not category:
            return None, {"kind": "unknown_stage_category", "status": status_name, "phase": phase}
        return category, None

    def _bundle_dict(self, reached, reached_at, stop_outcome, publication_eligible, ineligibility_reasons):
        stage_matrix = self.scenario.get("stage_matrix") or {}
        body = {
            "schema_version": i05_schema_version(),
            "scenario": {
                "scenario_id": self.identity["scenario_id"],
                "scenario_version": self.identity["scenario_version"],
                "entity_family": str(self.scenario.get("entity_family", "unknown")),
            },
            "run_id": self.run_id,
            "roots": dict(self.identity["roots"]),
            "stage_matrix_source": {
                "package_path": str(self.scenario_path),
                "package_digest": self.identity["fixture_digest"],
                "prelude": stage_matrix.get("prelude") or {},
                "lifecycle": stage_matrix.get("lifecycle") or {},
            },
            "stages": list(self._stages_index),
            "terminal_status": {"reached": reached, "reached_at": reached_at},
            "publication_eligible": publication_eligible,
            "ineligibility_reasons": list(ineligibility_reasons),
            # REQ-F-014: additive-only join references, read live from the
            # same identity/record objects the loop already holds.
            "scenario_id": self.identity["scenario_id"],
            "scenario_version": self.identity["scenario_version"],
            "dispatches": self.record["dispatches"],
        }
        if stop_outcome is not None:
            body["stop_outcome"] = stop_outcome
        return body


    def _write_bundle(self, *, reached, reached_at, stop_outcome, publication_eligible, ineligibility_reasons):
        body = self._bundle_dict(reached, reached_at, stop_outcome, publication_eligible, ineligibility_reasons)
        path = self.dir / "bundle.json"
        try:
            path.write_text(json.dumps(body, sort_keys=True, separators=(",", ":"), ensure_ascii=False) + "\n", encoding="utf-8")
        except OSError as exc:
            raise RuntimeError(f"failed to write I-05 bundle.json {path}: {exc}") from exc

    def record_stage(
        self, dispatch, stage_candidate, shark, cwd, worker_envelope, timing,
        fixture_root, fixture_input_digest, execution_adapter, lifecycle_adapter,
    ):
        """Called immediately after `record["stages"].append(...)` for the
        same dispatch (ADR-F12-01): writes one immutable stage snapshot file
        carrying every bench/README.md "Stage-snapshot field reference"
        field this task owns (REQ-F-003/004/006/007), a real `time_ledger`
        (REQ-F-005, via `reconcile_time_ledger()`), and a materialized
        transcript artifact (REQ-F-007); materializes E40-F07's replay
        consumption join; and rewrites `bundle.json`'s triad (REQ-F-002/003
        index). `artifacts`'s interior remains E40-F07's own scope -- this
        task writes its shape-valid, honestly-empty placeholder.

        `worker_envelope` is the FULL control envelope `adapter_result()`
        returned for this dispatch (never the bounded `dispatch["worker"]
        ["evidence"]` sub-field -- spec.md's own "envelope placement note"
        warns the adapter's SAFE_EVIDENCE_KEYS silently strips a usage/
        timing block placed there).

        `timing` is the per-dispatch accumulator
        (`{"stage_start": ns, "stage_end": ns, "claimed": [(category, start, end), ...]}`)
        the caller's dispatch loop and `adapter_result()` both appended real
        `time.monotonic_ns()` observations into, including on every
        exception path -- see `reconcile_time_ledger()`.

        Returns the snapshot's own `errors` list so the caller can fold
        evidence-unavailability signals (e.g. `test_suite_unavailable`,
        `unknown_stage_category`) into its own end-of-run trust gate."""
        ordinal = dispatch["ordinal"]
        response = dispatch["response"]
        stage_key = str(response.get("status", "")) or "unknown"
        entity_key = str(response.get("entity_key", ""))
        entity_type = str(response.get("entity_type", ""))
        entity = {"entity_key": entity_key, "entity_type": entity_type}
        provider = str(response.get("provider", ""))

        errors = []
        category, category_error = self._stage_category_for(shark, cwd, entity_type, stage_key)
        if category_error is not None:
            errors.append(category_error)
        if not provider:
            errors.append({"kind": "missing_provider"})
        usage, usage_errors = resolve_usage(
            mapped_provider(response), provider_usage_envelope(worker_envelope)
        )
        errors.extend(usage_errors)
        if stage_candidate.get("test_identity_error"):
            errors.append({
                "kind": "test_suite_unavailable",
                "detail": str(stage_candidate["test_identity_error"]),
            })

        visit_key = (entity_key, stage_key)
        rework_count = self._stage_visit_counts.get(visit_key, 0)
        self._stage_visit_counts[visit_key] = rework_count + 1

        snapshot = {
            "dispatch_ordinal": ordinal,
            "entity": entity,
            "stage_key": stage_key,
            "prompt_digest": str(dispatch["evidence_refs"].get("prompt_sha256", "")),
            "input_lineage": stage_input_lineage(
                self.record, dispatch, fixture_root, fixture_input_digest,
                execution_adapter, lifecycle_adapter,
            ),
            "artifacts": [],
            "usage": usage,
            "time_ledger": reconcile_time_ledger(self.origin_ns, timing),
            "rework_count": rework_count,
            "evaluator_access": [],
        }
        if category is not None:
            snapshot["stage_category"] = category
        if str(self.scenario.get("entity_family", "")) == "feature":
            snapshot["replay_lineage"] = self._prelude_lineage
        if category in {"code", "review"}:
            test_suite_ids, test_suite_dir = test_suite_reference(Path.cwd())
            candidate = dict(stage_candidate)
            candidate["test_suite_ids"] = test_suite_ids
            candidate["test_suite_dir"] = test_suite_dir
            snapshot["candidate"] = candidate
        snapshot["errors"] = errors
        snapshot["provider"] = provider

        # T-E40-F12-007 (REQ-NF-002): times only the write path itself --
        # serialize + write stages/<n>.json + transcript + access.jsonl
        # append + bundle.json rewrite -- never the category/usage
        # resolution above, which can itself shell out to `shark`. Emitted
        # once per dispatch, opt-in only, so it never affects a default run.
        write_start_ns = time.monotonic_ns()

        snapshot_digest = stage_snapshot_digest(snapshot)
        snapshot["snapshot_digest"] = snapshot_digest
        snapshot_path = self.stages_dir / f"{ordinal}-{stage_key}.json"
        try:
            snapshot_path.write_text(json.dumps(snapshot, sort_keys=True, separators=(",", ":"), ensure_ascii=False) + "\n", encoding="utf-8")
        except OSError as exc:
            raise RuntimeError(f"failed to write I-05 stage snapshot {snapshot_path}: {exc}") from exc

        # REQ-F-007: one bounded transcript artifact per dispatch, into the
        # real transcripts/ directory __init__ already materialized.
        # `bounded()` truncates/redacts exactly like every other recorded
        # response fragment (REQ-NF-003) -- the full envelope, never the
        # rendered prompt (never passed to this function at all), so no
        # prompt bytes or provider credentials can land here.
        transcript_path = self.transcripts_dir / f"{ordinal}-{stage_key}.txt"
        transcript_body = json.dumps(
            bounded(worker_envelope if isinstance(worker_envelope, dict) else {}),
            sort_keys=True, separators=(",", ":"), ensure_ascii=False,
        )
        try:
            transcript_path.write_text(transcript_body, encoding="utf-8")
        except OSError as exc:
            raise RuntimeError(f"failed to write I-05 transcript {transcript_path}: {exc}") from exc

        self._append_access_events(snapshot["evaluator_access"])
        self._stages_index.append({
            "dispatch_ordinal": ordinal,
            "stage_key": stage_key,
            "stage_category": category,
            # bundle_dir-relative, matching every real reader's own
            # resolve_within() (verify-stage-evidence.sh,
            # replay-stage-evidence.sh) -- both reject an absolute
            # snapshot_path outright.
            "snapshot_path": str(snapshot_path.relative_to(self.dir)),
            "snapshot_digest": snapshot_digest,
        })
        self._write_bundle(reached=False, reached_at=None, stop_outcome=None, publication_eligible=True, ineligibility_reasons=())

        if os.environ.get("LIFECYCLE_BENCH_TIMING") == "1":
            write_ms = (time.monotonic_ns() - write_start_ns) / 1_000_000
            print(f"i05_write_ms={write_ms:.3f}", file=sys.stderr)

        return errors

    def finalize(self, terminal, reason):
        """REQ-F-002's termination rewrite: the full §2.6 stop-outcome
        eligibility triad, via the top-level `stop_outcome_triad()`
        (T-E40-F12-004). Called from both `main()`'s normal end-of-run path
        and `release_on_signal()`'s signal-termination path, so a
        signal-terminated run's `bundle.json` triad is never left reflecting
        a stale per-dispatch state."""
        stop_outcome, publication_eligible, ineligibility_reasons = stop_outcome_triad(terminal, reason)
        self._write_bundle(reached=True, reached_at=timestamp(), stop_outcome=stop_outcome, publication_eligible=publication_eligible, ineligibility_reasons=ineligibility_reasons)


def capture_review_gate(
    shark, scratch, bundle_dir, entity, response, candidate, round_number,
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
    with tempfile.TemporaryDirectory(dir=bundle_dir) as temporary:
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
    """Retain every review-like gate reachable through any workflow outcome.

    Returns (policies, workflow_found): workflow_found distinguishes "no
    per-entity workflow YAML exists at all" from "a workflow YAML exists but
    legitimately configures zero review/qa/uat/approval gates" -- both yield
    an empty policies list, but is_configured_gate must treat them
    differently (fall back to keyword matching only in the former case).
    """
    family_name = str(entity_family).replace("_", "-")
    candidates = [family_name, family_name.replace("-card", "")]
    workflow_path = next(
        (scratch / "shark-data" / "workflow" / f"{name}.yaml" for name in candidates
         if (scratch / "shark-data" / "workflow" / f"{name}.yaml").is_file()),
        None,
    )
    if workflow_path is None:
        return [], False
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
    return policies, True


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


def is_configured_gate(configured_gate_ids, has_configured_workflow, gate_id):
    """Whether gate_id (the entity's current status) is a configured review/
    qa/uat/approval/code_review gate.

    Fix for the two-classifier defect (review finding F5): dispatch-time
    gate capture used to decide this by keyword-matching the status STRING
    (stage_category(gate_id) in {"review","qa","uat"}) -- independent of and
    sometimes disagreeing with configured_gate_policies(), which derives the
    same decision from the workflow YAML's own authoritative `phase:` field.
    A custom step name whose phase was gate-like but whose text didn't match
    a keyword (e.g. phase: qa, step name "verification") was silently
    recorded not_reached in I-07 even though it was genuinely dispatched --
    fabricating the exact "gate never ran" evidence I-05/I-07 exist to make
    impossible.

    configured_gate_ids/has_configured_workflow MUST be a frozen snapshot
    taken once, right after configured_gate_policies() runs, before the
    dispatch loop starts -- NOT a live read of
    record["workflow_policy"]["gate_policies"]. That list is mutated during
    dispatch by gate_policy_for()'s own fallback-append (a gate dispatched
    but absent from the configured set gets a synthesized entry appended so
    later lookups/backfill can find it). Reading the live list here would
    make an earlier dispatch's fallback-append silently redefine "configured"
    for every later dispatch in the SAME no-workflow-YAML run, one gate can
    permanently hide every gate dispatched after it. Only when no workflow
    YAML was found at all (has_configured_workflow is False) does this fall
    back to the keyword match, to preserve prior behavior for callers with
    no `steps:` schema to consult.
    """
    if has_configured_workflow:
        return gate_id in configured_gate_ids
    return stage_category(gate_id) in {"review", "qa", "uat"}


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


def adapter_result(adapter, request, cwd, mode, shark, timing, fixture_root, deadline, active_state):
    """`timing` is the caller's per-dispatch accumulator (see
    `reconcile_time_ledger()`): a mutable dict this function appends real
    `time.monotonic_ns()` observations into (T-E40-F12-003, REQ-F-005). It is
    mutated in place, not returned, so every observation this function makes
    -- including a heartbeat retry backoff window recorded right before a
    `LeaseLoss` this function itself raises -- survives whichever exception
    path the caller takes (never lost the way a return-value-only tuple
    would be on a raise). `deadline` (time.monotonic()) enforces the
    scenario's max_wall_clock_seconds ceiling on the in-flight provider
    process itself, not just the dispatch loop around it. `active_state` is
    the caller's shared lease-state dict; it always holds the live adapter
    `Popen` under "adapter_process" (or None) so a concurrent signal handler
    can stop the whole process group instead of only the parent process."""
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
    adapter_start_ns = time.monotonic_ns()
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
    heartbeat_args_base = ["heartbeat", request["entity_key"], "--session", request["session_id"], "--progress", "0.5", "--note", str(request.get("runner_id", "lifecycle"))]
    try:
        while process.poll() is None:
            now = time.monotonic()
            if now >= deadline:
                stop_process_group(process)
                active_state["adapter_process"] = None
                raise ResourceLimit("resource ceiling exceeded during provider dispatch: max_wall_clock_seconds")
            if now - last_heartbeat >= heartbeat_interval:
                try:
                    heartbeat = run_command(shark, heartbeat_args_base, cwd)
                except RuntimeError:
                    # T-E40-F12-003 (REQ-F-005/§2.5): one bounded retry,
                    # after a fixed backoff, before declaring LeaseLoss.
                    # The backoff+retry window is `retry_or_backoff`,
                    # recorded on BOTH the eventual-success and the
                    # eventual-failure branch below, so a real observation
                    # is never dropped on the exception path.
                    retry_start_ns = time.monotonic_ns()
                    time.sleep(HEARTBEAT_RETRY_BACKOFF_SECONDS)
                    try:
                        heartbeat = run_command(shark, heartbeat_args_base, cwd)
                    except RuntimeError as exc:
                        timing["claimed"].append(("retry_or_backoff", retry_start_ns, time.monotonic_ns()))
                        stop_process_group(process)
                        active_state["adapter_process"] = None
                        raise LeaseLoss(f"heartbeat failed for {request['entity_key']}: {exc}") from exc
                    timing["claimed"].append(("retry_or_backoff", retry_start_ns, time.monotonic_ns()))
                heartbeat_events.append({"session_id": request["session_id"], "at": timestamp(), "response": bounded(heartbeat)})
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
    adapter_end_ns = time.monotonic_ns()
    active_state["adapter_process"] = None
    if process.returncode != 0:
        raise RuntimeError(f"lifecycle adapter failed ({process.returncode}): {stderr.strip()}")
    envelope = load_json(stdout, "lifecycle adapter")
    if isinstance(envelope, dict):
        claims = provider_active_claims(envelope, adapter_start_ns, adapter_end_ns, timing["claimed"])
        timing["claimed"].extend(("provider_active", start, end) for start, end in claims)
    return envelope, heartbeat_events


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


# AC-F11-14, generalized: `entity_service.go`'s `requiresResearchEvidence()`
# gates every entity family's "research"-phase step the same way -- a real
# `<key>.research-report.md` produced by a live researcher agent
# (`internal/research/validator.go#ValidateEntity`) -- and wraps a missing/
# invalid report as `"cannot advance %s %s from research: research report:
# %w"` (entity_service.go). `resolve_route` never spawns a worker, so this
# failure is expected for ANY family whose research step it dispatches, not
# only the one family (feature) that happens to ship a pre-committed
# replay bundle. Matching on the Go error's own wrapped-error prefix keeps
# this generalized to any family/step declaring the same live-worker-only
# artifact dependency, rather than re-hardcoding a family allowlist.
RESEARCH_ARTIFACT_DEFERRED_MARKER = "research report:"


def dispatch_failure_deferred_reason(reason):
    """Is a `run_command` failure raised during route dispatch actually an
    expected, live-worker-only-artifact absence (AC-F11-14: `artifact_deferred`),
    rather than a genuine route defect? See module-level comment above.
    """
    if RESEARCH_ARTIFACT_DEFERRED_MARKER in reason:
        return reason
    return ""


def replay_artifact_deferred_reason(scenario_path, scenario):
    """AC-F11-14 (feature-specific half): is the I-06 replay bundle a
    feature scenario's own `resolve_route` pass would need (never invoked by
    resolve_route itself, but required downstream) present? Unlike
    `pre_dispatch_gates`, this checks existence and shape only -- it never
    invokes verify-replay-isolation.sh, which requires a real fixture
    checkout and scratch tree (exactly the live-worker-only artifacts
    AC-F11-13 says route resolution must not require). Returns "" when
    nothing is deferred, else a human-readable reason. This is a static,
    scenario-package-level check (the bundle either ships in the package or
    it doesn't); the research-report class above is instead detected
    reactively from an actual dispatch failure, because no scenario package
    ships a pre-committed research report the way feature packages ship a
    replay bundle.
    """
    if scenario.get("entity_family") != "feature":
        return ""
    replay_reference = scenario.get("replay_reference")
    if not isinstance(replay_reference, str) or not replay_reference.strip():
        return "feature scenario is missing replay_reference"
    package_root = scenario_path.parent.resolve()
    bundle_path = (package_root / replay_reference).resolve()
    try:
        bundle_path.relative_to(package_root)
    except ValueError:
        return "replay_reference escapes the scenario package"
    if not bundle_path.is_file():
        return f"replay bundle is missing: {bundle_path}"
    return ""


def _dispatch_and_advance(shark, scratch, run_id, requested, next_ordinal, response, dispatches, stages):
    """resolve_route()'s per-step spawn_agent handling: build the dispatch
    and stage records for the requested entity and append them to the
    caller's lists, then claim/advance/release it along the "pass" route.
    The dispatch/stage records are appended BEFORE the claim/advance
    attempt (matching pre-extraction behavior) so a failure there (e.g. an
    artifact-deferred dispatch) still leaves the traced stage in the
    ledger -- only raises RuntimeError, never returns a value.
    """
    entity = str(response.get("entity_key", ""))
    if not entity:
        raise RuntimeError("keyed response omitted entity_key")
    stage_id = str(response.get("status", ""))
    dispatches.append({
        "ordinal": next_ordinal,
        "requested_key": requested,
        "entity_key": entity,
        "stage_id": stage_id,
        "route": "pass",
    })
    stages.append({
        "stage_id": stage_id,
        "route": "pass",
        "provider": str(response.get("provider", "")),
        "model": str(response.get("model", "")),
        "effort": str(response.get("effort", "")),
        "terminal": False,
        "artifact_dependencies_deferred": True,
    })
    session = ""
    # try/except/else (not try/finally): a `finally` block's own
    # release() call raising would replace the ORIGINAL exception
    # (e.g. a genuine status-advance route_defect) with the cleanup
    # failure, masking the real defect. Attempting release only on
    # the two disjoint paths below means a cleanup failure is
    # always attempted, but never allowed to hide a defect that
    # already occurred -- and, on the success path, a cleanup
    # failure still surfaces normally (nothing to mask there).
    try:
        claim = run_command(
            shark,
            ["claim", entity, "--by", os.environ.get("LIFECYCLE_RUNNER_ID", run_id), "--json"],
            scratch,
        )
        session = str(claim.get("session_id", ""))
        if not session:
            raise RuntimeError(f"claim for {entity} omitted session_id")
        run_command(
            shark,
            [
                "status", "advance", entity, "--outcome", "pass",
                "--session", session, "--from-status", stage_id,
                "--agent", f"{response.get('agent_type', '')}@{response.get('provider', '')}",
                "--json",
            ],
            scratch,
        )
    except RuntimeError:
        if session:
            try:
                run_command(shark, ["release", entity, "--session", session, "--outcome", "route_resolution", "--json"], scratch)
            except RuntimeError:
                pass  # cleanup failure must never mask the defect being propagated below
        raise
    else:
        if session:
            run_command(shark, ["release", entity, "--session", session, "--outcome", "route_resolution", "--json"], scratch)


def _classify_route_result(terminal, route_defect_reason, artifact_deferred_reason, reason):
    """resolve_route()'s post-loop resolution/cause_class classification.
    Same precedence order as before extraction: route_defect_reason >
    artifact_deferred_reason > terminal == "error" > resolved. Returns
    (resolution, cause_class, reason, terminal) -- terminal is echoed back
    since the artifact_deferred branch can rewrite it.
    """
    if route_defect_reason:
        resolution = "failed"
        cause_class = "route_defect"
        reason = route_defect_reason
    elif artifact_deferred_reason:
        resolution = "resolved"
        cause_class = "artifact_deferred"
        reason = artifact_deferred_reason
        # The deferred artifact does not fail resolution (AC-F11-14) --
        # a dispatch-loop exception that was reclassified as deferred above
        # must not leave a stale "error" terminal contradicting that.
        if terminal == "error":
            terminal = "complete"
    elif terminal == "error":
        # A harness/CLI output-contract failure (see below): resolution
        # still fails -- the route was not actually traced to a terminal
        # step -- but it is never reported as route_defect.
        resolution = "failed"
        cause_class = None
    else:
        resolution = "resolved"
        cause_class = None
    return resolution, cause_class, reason, terminal


def resolve_route(args, scenario_path, scenario, scratch, shark):
    """--mode resolve-route (AC-F11-13/14/15): trace the configured success
    route (every outcome resolved as "pass") using real `shark next` /
    `status advance` calls against the scratch project, without any of the
    live-run evidence machinery (candidate identity, scratch-content digest,
    prompt-out verification, resource ceilings, adapter subprocess).
    """
    default_output = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "runs" / args["run_id"] / "resolution.jsonl"
    output = Path(args["output"]).resolve() if args["output"] else Path(os.environ.get("LIFECYCLE_OUTPUT", str(default_output))).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)

    scenario_id = str(scenario.get("scenario_id", scenario_path.stem))
    scenario_version = str(scenario.get("scenario_version", "1"))
    family = str(scenario.get("entity_family", "unknown"))

    # AC-F11-14: an absent live-only artifact is noted, not fatal -- route
    # resolution keeps tracing regardless of what this returns.
    artifact_deferred_reason = replay_artifact_deferred_reason(scenario_path, scenario)

    dispatches = []
    stages = []
    terminal = "complete"
    reason = "all eligible dispatches completed"
    route_defect_reason = ""
    queue = [args["root"]]
    finished = set()
    next_ordinal = 0

    try:
        while queue:
            requested = queue.pop(0)
            if requested in finished:
                continue
            finished.add(requested)
            next_ordinal += 1
            # --prompt-out is unused by route resolution (AC-F11-13: no
            # artifact only a live worker can produce is required) -- it is
            # still passed so `shark next` behaves identically to every
            # other mode's caller-path, matching the convention every
            # existing stub `shark` in this suite (tc061, tc079) already
            # assumes.
            prompt_path = scratch / "route-prompts" / f"{next_ordinal:04d}"
            prompt_path.parent.mkdir(parents=True, exist_ok=True)
            response = run_command(shark, ["next", requested, "--json", "--prompt-out", str(prompt_path)], scratch)
            action = response.get("action")
            if action == "parallel_candidates":
                candidates = fork_candidates(response)
                queue[0:0] = [item["entity_key"] for item in candidates]
                queue.sort()
                continue
            if action != "spawn_agent":
                terminal = str(action) if action in STOP_OUTCOMES else "error"
                reason = str(response.get("error") or f"keyed dispatch returned action {action!r}")
                if terminal == "error" and not route_defect_reason:
                    route_defect_reason = reason
                continue
            _dispatch_and_advance(
                shark, scratch, args["run_id"], requested, len(dispatches) + 1, response,
                dispatches, stages,
            )
    except RuntimeError as exc:
        terminal = "error"
        reason = str(exc)
        # AC-F11-14 draws a hard line at "an outcome target naming an
        # undefined step, a missing dispatch resolution" -- a `shark`
        # subcommand that exits non-zero (or can't be executed at all) is
        # evidence of exactly that. A command that exits 0 but prints text
        # `load_json` can't parse is a DIFFERENT thing: the harness/CLI
        # output contract was violated, not the workflow route. Conflating
        # the two would mislabel a caller-path defect as a route defect --
        # the same failure-to-distinguish AC-F11-14 exists to forbid, just
        # one level down. §2.3.2's closed cause_class enum has no slot for
        # "the CLI didn't answer in JSON", so this stays cause_class: null
        # with an honest reason rather than an invented eighth value.
        #
        # A dispatch failure caused by an absent live-worker-only artifact
        # (e.g. a missing research report at a "research"-phase step) is
        # a THIRD thing, generalized across every family -- see
        # dispatch_failure_deferred_reason(). It is reported as
        # artifact_deferred, never route_defect, regardless of which
        # family's step raised it.
        if "returned non-JSON output" not in reason:
            deferred = dispatch_failure_deferred_reason(reason)
            if deferred and not artifact_deferred_reason:
                artifact_deferred_reason = deferred
            elif not deferred:
                route_defect_reason = reason

    if stages:
        stages[-1]["terminal"] = True

    resolution, cause_class, reason, terminal = _classify_route_result(
        terminal, route_defect_reason, artifact_deferred_reason, reason
    )

    record = {
        "evidence_mode": "route_resolution_only",
        "scenario_id": scenario_id,
        "scenario_version": scenario_version,
        "family": family,
        "root": args["root"],
        "run_id": args["run_id"],
        "resolution": resolution,
        "cause_class": cause_class,
        "reason": reason,
        "dispatch_count": len(dispatches),
        "stage_count": len(stages),
        "dispatches": dispatches,
        "stages": stages,
        "outcome": {"terminal": terminal, "reason": reason},
        "artifact_dependencies_deferred": True,
    }
    output.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
    return 1 if resolution == "failed" else 0


def finalize_stage_evidence(
    record, evidence_errors, dispatch, candidate, response, entity,
    fixture_root, fixture_input_digest, execution_adapter, adapter,
    i05_writer, shark, scratch, worker_result, timing, rc_start_ns,
    stage_visits, started,
):
    """Builds this dispatch's I-07 lifecycle_stage record (and, if an I-05
    bundle is configured, its I-05 stage-evidence snapshot), appending both
    to record/evidence_errors in place, and records observed wall-clock
    elapsed time. Returns (stage_candidate, gate_category) -- both are
    consumed by the review-gate capture step (capture_gate_if_configured)
    that follows in the caller. gate_category is None when no I-05 bundle
    is configured (nothing wrote a stage_category for this dispatch).

    Extracted from main()'s dispatch loop (review finding F4): a
    self-contained post-dispatch bookkeeping step that reads a handful of
    locals and mutates record/evidence_errors/stage_visits in place, never
    rebinding a loop-carried scalar (ordinal/terminal/reason) -- the
    extraction changes nothing about the surrounding try/except/finally
    control flow those scalars live in.
    """
    # REQ-F-005/§2.5: `stage_end` is captured immediately after
    # `refresh_candidate()` completes -- the producer's own
    # serialization/write happens after this point and is
    # deliberately outside the stage window (spec.md §2.5).
    timing["stage_end"] = time.monotonic_ns()
    timing["claimed"].append(("tool_and_test", rc_start_ns, timing["stage_end"]))
    elapsed = max(0.0, time.monotonic() - started)
    record["limits"]["observed_wall_clock_seconds"] = elapsed
    stage_candidate = dict(candidate)
    visit_key = (entity, str(response.get("status", "development")))
    rework_count = stage_visits.get(visit_key, 0)
    stage_visits[visit_key] = rework_count + 1
    lifecycle_stage = stage_record(dispatch, stage_candidate)
    # I-05's time ledger is explicitly nanosecond-valued, while I-07's
    # intervals reconcile against elapsed_seconds and are consumed by F10
    # as seconds. Keep the two contracts distinct at this boundary. Only
    # relative (end - start) spans within this stage matter here, so a
    # fixed zero origin (rather than the I05BundleWriter's own
    # run-scoped origin_ns) is a faithful, self-contained ledger.
    # elapsed_seconds is derived from this SAME ledger's own
    # stage_start/stage_end (timing's own window, which starts before
    # the claim and ends after refresh_candidate()) so the interval
    # sum always reconciles exactly against elapsed_seconds.
    stage_ledger = reconcile_time_ledger(0, timing)
    lifecycle_stage["intervals"] = [
        {"category": category, "start": start / 1_000_000_000, "end": end / 1_000_000_000}
        for category, ranges in stage_ledger["intervals"].items()
        for start, end in ranges
    ]
    lifecycle_stage["elapsed_seconds"] = (stage_ledger["stage_end"] - stage_ledger["stage_start"]) / 1_000_000_000
    lifecycle_stage["rework"] = rework_count > 0
    # REQ-F-003/X-09: the same resolve_usage() call I05BundleWriter's
    # own record_stage() makes for this dispatch's snapshot, so I-07's
    # stage errors[] reflects the identical usage-mapping outcome
    # rather than staying an empty placeholder.
    _usage, usage_errors = resolve_usage(
        mapped_provider(response), provider_usage_envelope(worker_result)
    )
    if stage_candidate.get("test_identity_error"):
        usage_errors.append({
            "kind": "test_suite_unavailable",
            "detail": str(stage_candidate["test_identity_error"]),
        })
    lifecycle_stage["errors"] = usage_errors
    # Same stage_input_lineage() call I05BundleWriter's own
    # record_stage() makes for this dispatch's snapshot, so I-07's
    # stage input_lineage[] reflects the identical typed input
    # identities rather than staying an empty placeholder.
    lifecycle_stage["input_lineage"] = stage_input_lineage(
        record, dispatch, fixture_root, fixture_input_digest,
        execution_adapter, adapter,
    )
    record["stages"].append(lifecycle_stage)
    gate_category = None
    if i05_writer is not None:
        stage_errors = i05_writer.record_stage(
            dispatch, stage_candidate, shark, scratch, worker_result, timing,
            fixture_root, fixture_input_digest, execution_adapter, adapter,
        )
        gate_category = i05_writer._stages_index[-1]["stage_category"]
        # unmapped_provider means the provider is legitimately outside
        # X-09's coverage today (e.g. openai_codex_cli, or a fixture
        # stub in tests) -- a known, non-fatal gap. Every other kind
        # (usage_slot_unavailable for a MAPPED provider, missing_
        # provider, test_suite_unavailable, unknown_stage_category)
        # means required evidence for evidence THAT SHOULD be
        # collectible went missing -- fatal to publication eligibility.
        evidence_errors.extend(
            {"dispatch_ordinal": dispatch["ordinal"], **item}
            for item in stage_errors
            if item.get("kind") != "unmapped_provider"
        )
    prompt_digest = str(response.get("prompt_sha256"))
    record["identity"]["rendered_prompt_digests"].append(prompt_digest)
    effort = str(response.get("effort") or "default")
    record["identity"]["provider_identity"].append({
        "stage": str(response.get("status")), "provider": str(response.get("provider")),
        "model": str(response.get("model")), "effort": effort,
    })
    return stage_candidate, gate_category


def capture_gate_if_configured(
    record, configured_gate_ids, has_configured_workflow, i05_writer,
    shark, scratch, entity, response, stage_candidate,
    review_rounds, seen_review_note_ids, repo_root, request,
):
    """Captures this dispatch's review/qa/uat gate evidence, if
    is_configured_gate says this dispatch's status is a configured gate.
    Mutates record/review_rounds in place.

    Extracted from main()'s dispatch loop (review finding F4) alongside
    finalize_stage_evidence -- the second of the two self-contained,
    non-scalar-rebinding post-dispatch steps.

    Fix for the two-classifier defect (review finding F5): gate capture
    used to decide "is this dispatch a review-like gate" from
    `gate_category` (derived from stage-category-map.yaml's phase->category
    table via i05_writer/_stage_category_for) -- independent of and
    sometimes disagreeing with configured_gate_policies()'s own read of the
    workflow YAML's `phase:` field. Both decisions now consult the same
    frozen, pre-loop `configured_gate_ids`/`has_configured_workflow`
    snapshot via is_configured_gate.

    Still guarded on i05_writer being configured, matching prior behavior:
    capture_review_gate() writes its gate bundle under evidence_root via
    tempfile.TemporaryDirectory(dir=evidence_root), which would silently
    fall back to the system temp dir instead of raising on evidence_root
    being None -- there is no I-05 bundle location to capture a gate
    against when --i05-bundle-dir was never passed.
    """
    if i05_writer is None:
        return
    gate_id = str(response.get("status", "review"))
    if not is_configured_gate(configured_gate_ids, has_configured_workflow, gate_id):
        return
    review_rounds[gate_id] = review_rounds.get(gate_id, 0) + 1
    gate, captured_policy = capture_review_gate(
        shark, scratch, i05_writer.dir, entity, response, stage_candidate,
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

    # resolve-route (T-E40-F11-007) deliberately skips resource-ceiling
    # enforcement (module docstring above) -- it resolves the `shark`
    # binary and returns without ever calling limits_from(). Every other
    # mode's ceiling-positivity check (limits_from(), AC-T3) must run
    # BEFORE this function resolves the `shark` binary from PATH: a caller
    # invoking a mode that fails cheap, static ceiling validation should
    # get that refusal even with no `shark` on PATH at all, not a
    # PATH-resolution error that masks the real one (tc080's AC-T3 block).
    if args["mode"] == "resolve-route":
        shark = os.environ.get("SHARK_BIN", "shark")
        if not shutil.which(shark):
            raise RuntimeError(f"shark executable not found on PATH: {shark}")
        return resolve_route(args, scenario_path, scenario, scratch, shark)

    # A caller-supplied --fixture-root (run-lifecycle-batch.sh's dispatch_pair,
    # which clones and base_sha-verifies an isolated checkout before dispatch)
    # takes priority over the scenario-declared submodule_path -- the shared,
    # unpinned checkout that path resolves to is exactly what the isolated
    # clone exists to avoid.
    if args["fixture_root"]:
        fixture_root = Path(args["fixture_root"]).resolve()
    else:
        fixture_decl = (scenario.get("fixture") or {}).get("submodule_path", scenario_path.parent)
        fixture_root = Path(fixture_decl).resolve()
    if not fixture_root.is_dir() or not (fixture_root / ".git").exists():
        raise RuntimeError(f"agent fixture checkout is not a git checkout: {fixture_root}")
    default_output = Path(os.environ.get("LIFECYCLE_BENCH_DIR", ".")) / "runs" / args["run_id"] / "lifecycle.jsonl"
    output = Path(args["output"]).resolve() if args["output"] else Path(os.environ.get("LIFECYCLE_OUTPUT", str(default_output))).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    limits = limits_from(scenario, args["limits"])
    adapter = os.environ.get("LIFECYCLE_ADAPTER") or os.environ.get("LIFECYCLE_ADAPTER_PATH", "")
    if args["mode"] not in {"contract", "dry-run"} and not adapter:
        raise RuntimeError("LIFECYCLE_ADAPTER is required for live lifecycle runs")

    shark = os.environ.get("SHARK_BIN", "shark")
    resolved_shark = shutil.which(shark)
    if not resolved_shark:
        raise RuntimeError(f"shark executable not found on PATH: {shark}")

    repo_root = Path(os.environ["LIFECYCLE_BENCH_DIR"]).parent.resolve()
    adapter_decl = scenario.get("adapter") or {}
    adapter_name = str(adapter_decl.get("name", "unknown"))
    adapter_version = str(adapter_decl.get("version", "unknown"))
    execution_adapter = repo_root / "bench" / "adapters" / adapter_name / "adapter.sh"
    if not execution_adapter.is_file() or not os.access(execution_adapter, os.X_OK):
        raise RuntimeError(f"registered execution adapter is unavailable: {execution_adapter}")
    toolchain_identity = scenario.get("toolchain_identity") or []
    identity = scenario_identity(scenario_path, scenario, scratch, fixture_root, limits)
    identity["run_id"] = args["run_id"]
    identity["dispatch_id"] = f"{args['run_id']}:1"
    identity["dispatch_ordinal"] = 1
    identity["scenario_path"] = str(scenario_path)
    identity["roots"]["scratch_shark_project"] = str(scratch)
    identity["shark_binary_digest"] = sha256_file(Path(resolved_shark))
    pre_dispatch_gates(scenario_path, scenario, fixture_root, scratch)
    record = make_record(identity, args["root"], scratch, limits, repo_root)
    record["entity_graph"]["root_type"] = str(scenario.get("entity_family", "unknown"))
    record["workflow_policy"]["reviewer"] = {"provider": "fixture", "model": "fixture", "effort": ""}
    configured_gate_policy_list, has_configured_workflow = configured_gate_policies(
        scratch, scenario.get("entity_family", "unknown"), repo_root,
    )
    record["workflow_policy"]["gate_policies"] = configured_gate_policy_list
    # Frozen at setup time, before any dispatch: is_configured_gate must never
    # read the live record["workflow_policy"]["gate_policies"] list, since
    # gate_policy_for() appends synthesized fallback entries to it during
    # dispatch (see is_configured_gate's own docstring for why that matters).
    configured_gate_ids = frozenset(policy["gate_id"] for policy in configured_gate_policy_list)
    refresh_workflow_policy(record)
    i05_writer = None
    if args["i05_bundle_dir"]:
        i05_writer = I05BundleWriter(args["i05_bundle_dir"], scenario_path, scenario, identity, args["run_id"], record)

    prelude_stop = None
    prelude_reason = ""
    prelude_evidence_dir = i05_writer.dir if i05_writer is not None else scratch / "prelude-evidence"
    prelude_evidence_dir.mkdir(parents=True, exist_ok=True)
    prelude_path = Path(args["prelude"]).resolve() if args["prelude"] else None
    if prelude_path is None and isinstance((scenario.get("stage_matrix") or {}).get("prelude"), dict):
        prelude_path = prelude_evidence_dir / "prelude.jsonl"
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
        if not isinstance(prelude, dict):
            raise RuntimeError("lifecycle prelude must be a JSON object")
        if prelude.get("scenario_id") != identity["scenario_id"]:
            raise RuntimeError("lifecycle prelude scenario_id does not match the scenario package")
        # Derive the semantic I-06 join projection while the parsed prelude
        # is still complete. The lifecycle record below remains diagnostic
        # and bounded, so it must never become this semantic source.
        if i05_writer is not None:
            i05_writer.set_prelude_lineage(prelude_lineage(prelude))
        try:
            retained_prelude_path = prelude_path.relative_to(prelude_evidence_dir).as_posix()
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
    ordinal = 0
    generated = 0
    started = time.monotonic()
    queue = [] if prelude_stop else [args["root"]]
    generated_entities = set()
    stage_visits = {}
    transition_rejections = {}
    review_rounds = {}
    seen_review_note_ids = set()
    evidence_errors = []
    terminal = prelude_stop or "complete"
    reason = prelude_reason or "all eligible dispatches completed"
    active_lease = {"entity": "", "session": "", "adapter_process": None, "cancel_signum": None}
    # Guards the queue.insert(0, requested) re-dispatch below: a real `shark
    # next` response changes once its entity actually advances, so seeing the
    # identical (entity_key, status) pair right after successfully advancing
    # it means no forward progress was made -- drop the redundant dispatch
    # instead of claiming/advancing it again forever. Only set on a
    # successful advance (never on retry_after_transition_rejection), so a
    # legitimate rejected-transition retry of the same entity/status is
    # unaffected.
    last_advanced_signature = None
    # Guards the fork branch's own `queue[0:0] = [*candidates, requested]`
    # re-push: a root that offers the identical candidate-key set on a
    # repeat resolution has nothing new to fork -- re-adding `requested`
    # again would just re-fork the same set forever once every candidate in
    # it has already been dispatched. Different candidate sets from the
    # same root (a genuinely advanced frontier) get their own signature and
    # are unaffected.
    seen_fork_signatures = set()

    def release_on_signal(signum, _frame):
        # T-E40-F12-004 (REQ-F-009, AC-014): Cancellation is a RuntimeError
        # subclass, so raising it here (instead of SystemExit) routes through
        # the dispatch loop's own except/finally (session release) and, on
        # propagation, main()'s end-of-run i05_writer.finalize() call --
        # never a subprocess call from inside a signal handler.
        stop_process_group(active_lease["adapter_process"])
        active_lease["adapter_process"] = None
        active_lease["cancel_signum"] = signum
        raise Cancellation(f"received signal {signum}")

    signal.signal(signal.SIGINT, release_on_signal)
    signal.signal(signal.SIGTERM, release_on_signal)

    try:
        while queue:
            requested = queue.pop(0)
            pre_dispatch_gates(scenario_path, scenario, fixture_root, scratch)
            prompt_path = scratch / "prompts" / f"{ordinal + 1:04d}"
            prompt_path.parent.mkdir(parents=True, exist_ok=True)
            # REQ-F-005/§2.5: `stage_start` is captured immediately before
            # this dispatch's `shark next` call -- used only if this queue
            # item turns into a real spawn_agent dispatch below.
            stage_start_ns = time.monotonic_ns()
            response = run_command(shark, ["next", requested, "--json", "--prompt-out", str(prompt_path)], scratch)
            if response.get("action") == "parallel_candidates":
                candidates = fork_candidates(response)
                record["entity_graph"]["fork_candidates"].append({"response": bounded(response), "candidates": candidates})
                record["entity_graph"]["selected_keys"].extend(item["entity_key"] for item in candidates)
                record["entity_graph"]["selected_types"].extend(item["entity_type"] for item in candidates)
                record["entity_graph"]["resolved_via"] = "fork_response"
                candidate_keys = [item["entity_key"] for item in candidates]
                fork_signature = (requested, tuple(candidate_keys))
                if fork_signature not in seen_fork_signatures:
                    seen_fork_signatures.add(fork_signature)
                    queue[0:0] = [*candidate_keys, requested]
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
            if (entity, str(response.get("status", ""))) == last_advanced_signature:
                continue
            prompt = str(response.get("prompt", ""))
            expected_digest = str(response.get("prompt_sha256", ""))
            expected_bytes = response.get("prompt_bytes")
            actual = prompt.encode("utf-8")
            if sha256_bytes(actual) != expected_digest or len(actual) != expected_bytes:
                raise RuntimeError(f"prompt digest or byte-count mismatch for {entity}")
            if prompt_path.read_bytes() != actual:
                raise RuntimeError(f"prompt-out bytes differ from response for {entity}")
            ci_start_ns = time.monotonic_ns()
            # T-E40-F09 first code-review round: candidate identity must
            # reflect the fixture checkout the agent actually modifies, not
            # this harness's own working directory (Path.cwd() lags fixture
            # edits entirely -- the exact defect that review round found).
            candidate = candidate_identity(
                fixture_root, execution_adapter, adapter_name, adapter_version, toolchain_identity,
                started + limits["max_wall_clock_seconds"], active_lease,
            )
            timing = {"stage_start": stage_start_ns, "claimed": [("tool_and_test", ci_start_ns, time.monotonic_ns())]}

            response_record = bounded(response)
            response_record.pop("prompt", None)
            dispatch = {
                "ordinal": ordinal + 1,
                "requested_key": requested,
                "response": response_record,
                "claim": {"session_id": None},
                "worker": {
                    "worker_id": None,
                    "session_id": None,
                    "kind": None,
                    "evidence": [],
                },
                "heartbeats": [],
                "outcome": "error",
                "transition": {},
                "release": {},
                "started_at": timestamp(),
                "ended_at": "",
                "evidence_refs": {
                    "prompt_sha256": expected_digest,
                    "prompt_bytes": expected_bytes,
                    "candidate_snapshot_digest": "0" * 64,
                },
            }
            record["dispatches"].append(dispatch)
            record["entity_graph"]["ordinals"].append(dispatch["ordinal"])
            if entity != args["root"] and response.get("entity_type") == "task":
                generated_entities.add(entity)
                generated = len(generated_entities)
            session = ""
            worker_result = {}
            request = {}
            advanced = False
            retry_after_transition_rejection = False
            fixture_input_digest = tree_digest(fixture_root)
            try:
                claim_start_ns = time.monotonic_ns()
                claim = run_command(shark, ["claim", entity, "--by", os.environ.get("LIFECYCLE_RUNNER_ID", args["run_id"]), "--json"], scratch)
                timing["claimed"].append(("queue_or_claim_wait", claim_start_ns, time.monotonic_ns()))
                session = str(claim.get("session_id", ""))
                if not session:
                    raise RuntimeError(f"claim for {entity} omitted session_id")
                active_lease["entity"] = entity
                active_lease["session"] = session
                dispatch["claim"] = bounded(claim)
                dispatch["worker"]["session_id"] = session
                request = dict(response)
                request["session_id"] = session
                request["runner_id"] = os.environ.get("LIFECYCLE_RUNNER_ID", args["run_id"])
                request["allowed_outcomes"] = allowed_outcomes(scratch, response)
                worker_result, heartbeats = adapter_result(
                    adapter, request, scratch, args["mode"], shark, timing, fixture_root,
                    started + limits["max_wall_clock_seconds"], active_lease,
                )
                dispatch["heartbeats"] = heartbeats
                if worker_result.get("session_id") not in {None, session}:
                    raise RuntimeError(f"worker session mismatch for {entity}")
                dispatch["worker"] = {"worker_id": worker_result.get("worker_id", ""), "session_id": worker_result.get("session_id", session), "kind": worker_result.get("kind", ""), "recommended_outcome": worker_result.get("recommended_outcome"), "evidence": bounded(worker_result.get("evidence", {})), "usage": bounded(worker_result.get("usage", {})), "cost_usd": worker_result.get("cost_usd", 0.0)}
                kind = worker_result.get("kind")
                if kind == "question":
                    question_start_ns = time.monotonic_ns()
                    question_key = route_worker_question(
                        worker_result, entity, session, request["runner_id"], scratch, shark
                    )
                    timing["claimed"].append(("replay_or_human_gate_wait", question_start_ns, time.monotonic_ns()))
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
                    try:
                        advance_response = run_command(shark, ["status", "advance", entity, "--outcome", str(outcome), "--session", session, "--from-status", str(response.get("status", "")), "--agent", f"{response.get('agent_type', '')}@{response.get('provider', '')}", "--json"], scratch)
                    except RuntimeError as exc:
                        rejection = bounded(str(exc))
                        if not is_repairable_research_rejection(exc, response, entity):
                            dispatch["transition"] = {
                                "outcome": str(outcome), "session_id": session,
                                "from_status": response.get("status", ""),
                                "accepted": False, "rejection": rejection,
                                "retry_scheduled": False,
                            }
                            raise
                        rejection_key = (entity, str(response.get("status", "")))
                        rejection_count = transition_rejections.get(rejection_key, 0) + 1
                        transition_rejections[rejection_key] = rejection_count
                        retry_after_transition_rejection = rejection_count <= MAX_TRANSITION_REJECTIONS_PER_STAGE
                        dispatch["transition"] = {
                            "outcome": str(outcome), "session_id": session,
                            "from_status": response.get("status", ""),
                            "accepted": False, "rejection": rejection,
                            "retry_scheduled": retry_after_transition_rejection,
                            "rejection_attempt": rejection_count,
                        }
                        dispatch["outcome"] = "fail" if retry_after_transition_rejection else "worker_failure"
                        if retry_after_transition_rejection:
                            note = (
                                f"Lifecycle transition rejected worker outcome {outcome!r}: {rejection}. "
                                "Continue from the existing artifact, correct the named validation failure, "
                                "and recommend pass only after re-validating it."
                            )
                            run_command(
                                shark,
                                ["notes", "add", entity, "--type", "testing", note,
                                 "--created-by", os.environ.get("LIFECYCLE_RUNNER_ID", args["run_id"])],
                                scratch, expect_json=False,
                            )
                        else:
                            terminal = "worker_failure"
                            reason = (
                                f"status transition for {entity} from {response.get('status', '')} "
                                f"was rejected {rejection_count} times: {rejection}"
                            )
                    else:
                        dispatch["transition"] = {"outcome": str(outcome), "session_id": session, "from_status": response.get("status", ""), "to_status": str(advance_response.get("new_status", ""))}
                        advanced = True
                        last_advanced_signature = (entity, str(response.get("status", "")))
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
                    release_start_ns = time.monotonic_ns()
                    try:
                        dispatch["release"] = bounded(run_command(shark, ["release", entity, "--session", session, "--outcome", dispatch["outcome"], "--json"], scratch))
                    except RuntimeError as exc:
                        dispatch["release"] = {"error": str(exc)}
                        if terminal == "complete":
                            terminal = "error"
                            reason = str(exc)
                    finally:
                        timing["claimed"].append(("queue_or_claim_wait", release_start_ns, time.monotonic_ns()))
                    active_lease["entity"] = ""
                    active_lease["session"] = ""
            dispatch["ended_at"] = timestamp()
            cost = float(worker_result.get("cost_usd", (worker_result.get("usage") or {}).get("cost_usd", 0.0)) or 0.0)
            dispatch["cost_usd"] = cost
            record["limits"]["observed_cost_usd"] += cost
            record["limits"]["observed_generated_tasks"] = generated
            rc_start_ns = time.monotonic_ns()
            try:
                refresh_candidate(
                    candidate, fixture_root, scratch, execution_adapter,
                    adapter_name, adapter_version, toolchain_identity,
                    started + limits["max_wall_clock_seconds"], active_lease,
                )
            except Cancellation as exc:
                terminal = "cancellation"
                reason = str(exc)
                refresh_candidate(
                    candidate, fixture_root, scratch, execution_adapter,
                    adapter_name, adapter_version, toolchain_identity,
                    test_identity_error=reason,
                )
            stage_candidate, gate_category = finalize_stage_evidence(
                record, evidence_errors, dispatch, candidate, response, entity,
                fixture_root, fixture_input_digest, execution_adapter, adapter,
                i05_writer, shark, scratch, worker_result, timing, rc_start_ns,
                stage_visits, started,
            )
            capture_gate_if_configured(
                record, configured_gate_ids, has_configured_workflow, i05_writer,
                shark, scratch, entity, response, stage_candidate,
                review_rounds, seen_review_note_ids, repo_root, request,
            )
            ordinal += 1
            write_partial(output, record)
            if terminal != "complete" and terminal != "resource_limit":
                break
            exceeded = None
            if record["limits"]["observed_cost_usd"] >= limits["max_cost_usd"]:
                exceeded = "max_cost_usd"
            elif record["limits"]["observed_wall_clock_seconds"] >= limits["max_wall_clock_seconds"]:
                exceeded = "max_wall_clock_seconds"
            elif generated >= limits["max_generated_tasks"]:
                exceeded = "max_generated_tasks"
            if exceeded:
                terminal = "resource_limit"
                record["limits"]["first_exceeded"] = exceeded
                reason = f"resource ceiling exceeded: {exceeded}"
                break
            if (advanced or retry_after_transition_rejection) and args["mode"] != "dry-run":
                queue.insert(0, requested)
        if terminal == "complete" and evidence_errors:
            terminal = "error"
            reason = "required stage evidence is unavailable: " + "; ".join(
                f"dispatch={item['dispatch_ordinal']} {item['kind']} "
                f"{item.get('detail') or item.get('status') or item.get('slot') or item.get('provider') or ''}"
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
    if i05_writer is not None:
        i05_writer.finalize(terminal, reason)
    output.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
    output.with_suffix(output.suffix + ".partial").unlink(missing_ok=True)
    cancel_signum = active_lease.get("cancel_signum")
    if terminal == "cancellation" and cancel_signum is not None:
        # POSIX 128+signum -- the numeric exit code any caller checking $?
        # sees whether the process actually died by signal or (as here)
        # exited with this value directly. This mirrors the pre-5f80d23c
        # signal handler's own `raise SystemExit(128 + signum)`; only where
        # that finalize/release work happens moved (out of the signal
        # handler and into the normal flow above), not this exit value.
        return 128 + cancel_signum
    # Named stop outcomes are valid, retained I-07 results. resource_limit
    # (hit a cost/time/task ceiling) is grouped with "complete" -- exit 0 --
    # since it is an expected, planned boundary, not a problem; the other
    # named stop outcomes (missing_outcome, worker_failure, error, pause,
    # lease_loss, ...) exit 1 so run-lifecycle-batch.sh's own run_rc
    # handling can still tell "ran to a retained, evaluable stop outcome"
    # apart from "never produced evidence at all" (exit 2, below).
    return 0 if terminal in ("complete", "resource_limit") else 1


try:
    raise SystemExit(main(sys.argv[1:]))
except (RuntimeError, OSError, ValueError, TypeError) as exc:
    # Reached only for a failure before/outside main()'s own dispatch-loop
    # try/except (line ~2365) -- i.e. no lifecycle.jsonl or bundle.json was
    # ever finalized. Exit 2 (not 1, which main() now reserves for a
    # completed, evidence-bearing named stop outcome) keeps that
    # distinction visible to run-lifecycle-batch.sh's own run_rc handling.
    print(f"run-lifecycle: {exc}", file=sys.stderr)
    raise SystemExit(2)
PY
