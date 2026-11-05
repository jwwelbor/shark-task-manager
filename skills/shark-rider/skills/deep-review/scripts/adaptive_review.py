#!/usr/bin/env python3
"""Select and run the strongest available deep-review runner.

The script is deliberately independent of Claude Code/Codex APIs.  Host skills
use ``select`` for capability routing; ``run-cli`` is the bounded, read-only
out-of-band fallback and report writer. If the preferred alternate CLI cannot
complete the review, run-cli retries an installed comparable CLI.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import selectors
import signal
import shlex
import shutil
import subprocess
from bounded_git import run as run_bounded_git
import sys
import tempfile
import time
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path

from source_inventory import source_inventory
from safe_review_io import MAX_ARTIFACT_BYTES, read_review_file
from review_status import filter_status, registered_worktrees, runtime_prefix, runtime_project, selection_overlaps_runtime, validate_index_flags


VERDICT_RE = re.compile(r"\bVERDICT\s*:?\s*\*{0,2}\s*:?\s*(PASS(?:-with-triage)?|FAIL)\b", re.I | re.ASCII)
MAX_REVIEW_OUTPUT_BYTES = 8 * 1024 * 1024


def _kill_reviewer_group(pid):
    try:
        os.killpg(pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def _read_reviewer_output(selector, events, buffers):
    exceeded = False
    for key, _ in events:
        chunk = os.read(key.fileobj.fileno(), 65536)
        if not chunk:
            selector.unregister(key.fileobj)
            continue
        buffer = buffers[key.fileobj]
        room = MAX_REVIEW_OUTPUT_BYTES - len(buffer)
        buffer.extend(chunk[:room])
        if len(chunk) > room:
            exceeded = True
    return exceeded


def _drain_reviewer(process, selector, buffers, deadline):
    exceeded = False
    timed_out = False
    kill_deadline = None
    while selector.get_map():
        now = time.monotonic()
        if kill_deadline is not None and now >= kill_deadline:
            break
        remaining = deadline - now
        if remaining <= 0:
            timed_out = True
            remaining = 5
        events = selector.select(min(remaining, 0.25))
        exceeded = _read_reviewer_output(selector, events, buffers) or exceeded
        if (exceeded or timed_out) and kill_deadline is None:
            _kill_reviewer_group(process.pid)
            kill_deadline = time.monotonic() + 5
    return exceeded, timed_out


def run_bounded_reviewer(argv, cwd, timeout):
    """Run an untrusted reviewer with bounded in-memory stdout/stderr."""
    if os.name == "nt":
        raise OSError("bounded reviewer process-tree cleanup is unavailable on Windows")
    process = subprocess.Popen(argv, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
    selector = selectors.DefaultSelector()
    buffers = {process.stdout: bytearray(), process.stderr: bytearray()}
    for stream in buffers:
        selector.register(stream, selectors.EVENT_READ)
    deadline = time.monotonic() + timeout
    try:
        exceeded, timed_out = _drain_reviewer(process, selector, buffers, deadline)
        try:
            process.wait(timeout=max(0, deadline - time.monotonic()))
        except subprocess.TimeoutExpired:
            timed_out = True
            _kill_reviewer_group(process.pid)
            process.wait()
        return subprocess.CompletedProcess(
            argv, 1 if exceeded or timed_out else process.returncode,
            bytes(buffers[process.stdout]).decode("utf-8", errors="replace"),
            bytes(buffers[process.stderr]).decode("utf-8", errors="replace"),
        )
    finally:
        # Descendants may close both pipes while continuing to run.
        _kill_reviewer_group(process.pid)
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            pass
        selector.close()
        process.stdout.close()
        process.stderr.close()


@dataclass(frozen=True)
class Capabilities:
    workflow: bool
    agent_dispatch: bool
    codex: bool
    claude: bool


def _flag(value: bool | None, env_name: str) -> bool:
    if value is not None:
        return value
    return os.environ.get(env_name, "").lower() in {"1", "true", "yes", "on"}


def detect_capabilities(
    workflow: bool | None = None,
    agent_dispatch: bool | None = None,
    codex_command: str | None = None,
    claude_command: str | None = None,
) -> Capabilities:
    return Capabilities(
        workflow=_flag(workflow, "DEEP_REVIEW_WORKFLOW"),
        agent_dispatch=_flag(agent_dispatch, "DEEP_REVIEW_AGENT_DISPATCH"),
        codex=bool(codex_command or shutil.which("codex")),
        claude=bool(claude_command or shutil.which("claude")),
    )


def select_runner(caps: Capabilities, host: str | None = None) -> dict[str, object]:
    host = (host or os.environ.get("DEEP_REVIEW_HOST") or "unknown").lower()
    if caps.workflow:
        return {"runner_mode": "canonical-six-angle", "fallback_reason": None}
    if caps.agent_dispatch:
        return {"runner_mode": "dispatched-six-angle", "fallback_reason": "native Workflow unavailable"}

    alternate = "claude" if host == "codex" else "codex" if host == "claude" else None
    available = {"codex": caps.codex, "claude": caps.claude}
    if alternate and available[alternate]:
        return {
            "runner_mode": "adversarial-cli",
            "adversarial_model": alternate,
            "fallback_reason": "Workflow and agent dispatch unavailable",
        }
    if host in available and available[host]:
        return {
            "runner_mode": "adversarial-cli",
            "adversarial_model": host,
            "fallback_reason": (
                "preferred alternate CLI unavailable; using the host's "
                "comparable CLI"
            ),
        }
    for model in ("claude", "codex"):
        if available[model]:
            return {
                "runner_mode": "adversarial-cli",
                "adversarial_model": model,
                "fallback_reason": "host identity unknown; selected available alternate CLI",
            }
    return {
        "runner_mode": "incomplete",
        "adversarial_model": None,
        "fallback_reason": "no Workflow, agent dispatch, codex, or claude capability available",
    }


def cli_candidates(caps: Capabilities, host: str | None, preferred: str) -> list[str]:
    """Return preferred CLI followed by usable comparable fallbacks."""
    host_name = (host or os.environ.get("DEEP_REVIEW_HOST") or "unknown").lower()
    available = {"codex": caps.codex, "claude": caps.claude}
    candidates = [preferred]
    if host_name in available and available[host_name] and host_name not in candidates:
        candidates.append(host_name)
    for model in ("claude", "codex"):
        if available[model] and model not in candidates:
            candidates.append(model)
    return candidates


def valid_adversarial_output(text: str) -> bool:
    """Apply the same report contract to CLI and six-angle output."""
    verdict = VERDICT_RE.search(text)
    return bool(verdict and valid_review_report(text, verdict.group(1)))


def _cli_argv(model: str, override: str | None, prompt: str) -> list[str]:
    if override:
        return [*shlex.split(override), prompt]
    if model == "codex":
        return ["codex", "exec", "--sandbox", "read-only", "--skip-git-repo-check", prompt]
    return ["claude", "-p", prompt, "--permission-mode", "plan", "--output-format", "text"]


def _scope_git(root, *args, env=None):
    return run_bounded_git(["git", "-c", "core.fsmonitor=false", "-C", str(root), *args],
                          env=env, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout


def _common_git_dir(root):
    value = Path(os.fsdecode(_scope_git(root, "rev-parse", "--git-common-dir")).strip())
    return (value if value.is_absolute() else Path(root) / value).resolve(strict=True)


def _validate_scope_manifest(scope, base_commit, diff_path, root):
    validate_index_flags(_scope_git(root, "ls-files", "-v", "-z"))
    candidate = _scope_git(root, "rev-parse", "HEAD").decode().strip()
    tree = _scope_git(root, "rev-parse", "HEAD^{tree}").decode().strip()
    status = _scope_git(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
    runtime_project = scope.get("runtime_project", ".")
    if filter_status(status, runtime_prefix(runtime_project), registered_worktrees(root, runtime_project)):
        raise ValueError("candidate worktree is dirty")
    if candidate != scope["candidate_commit"] or tree != scope["candidate_tree"]:
        raise ValueError("candidate changed since scope capture")
    if base_commit != scope["base_commit"] or diff_path != scope["diff_path"]:
        raise ValueError("review arguments do not match captured scope")
    _scope_git(root, "merge-base", "--is-ancestor", base_commit, candidate)
    diff = _validate_scope_diff(scope, base_commit, diff_path, root, candidate)
    return candidate, tree, diff


def _validate_scope_diff(scope, base_commit, diff_path, root, candidate):
    diff = read_review_file(diff_path, 128 * 1024 * 1024)
    expected_diff = _scope_git(root, "diff", "--no-ext-diff", "--binary", base_commit, candidate)
    if diff != expected_diff or hashlib.sha256(diff).hexdigest() != scope["diff_sha256"]:
        raise ValueError("diff changed since scope capture")
    manifest = [os.fsdecode(p) for p in _scope_git(root, "diff", "--name-only", "-z", base_commit, candidate).split(b"\0") if p]
    if manifest != scope["changed_files"] or len(manifest) != scope["changed_file_count"]:
        raise ValueError("scope manifest does not match candidate")
    return diff


def _snapshot_marker(root, candidate):
    subject = _scope_git(root, "log", "-1", "--format=%s", candidate).decode().strip()
    marker_path = Path(os.fsdecode(_scope_git(root, "rev-parse", "--git-path", "shark-review-source.json")).strip())
    if not marker_path.is_absolute():
        marker_path = Path(root) / marker_path
    marker = json.loads(read_review_file(marker_path, 2 * 1024 * 1024).decode("utf-8")) if os.path.lexists(marker_path) else None
    return subject, marker


def _validate_snapshot_metadata(source, marker, root, candidate, tree, expected_runtime):
    source_keys = {"project_root", "source_root", "source_commit", "candidate_commit", "candidate_tree", "selected_paths", "selected_index_sha256", "source_inventory_sha256"}
    if not isinstance(source, dict) or set(source) not in (source_keys, source_keys | {"runtime_project"}):
        raise ValueError("invalid source snapshot fields")
    if source.get("runtime_project", ".") != expected_runtime:
        raise ValueError("snapshot Shark project location mismatch")
    if marker is None or marker != source:
        raise ValueError("isolated snapshot marker does not match source metadata")
    if Path(source["project_root"]).resolve() != Path(root).resolve():
        raise ValueError("snapshot project metadata mismatch")
    if source["candidate_commit"] != candidate or source["candidate_tree"] != tree:
        raise ValueError("source snapshot identity does not match candidate")
    if _common_git_dir(root) != _common_git_dir(source["source_root"]):
        raise ValueError("snapshot repository mismatch")


def _valid_snapshot_path(path):
    return isinstance(path, str) and bool(path) and not Path(path).is_absolute() and '..' not in Path(path).parts and path != '.'


def _snapshot_paths(source):
    paths = source["selected_paths"]
    if not isinstance(paths, list) or not paths or not all(_valid_snapshot_path(p) for p in paths):
        raise ValueError("invalid source snapshot paths")
    prefix = runtime_prefix(source.get("runtime_project", "."))
    worktrees = registered_worktrees(source["source_root"], source.get("runtime_project", "."))
    if any(selection_overlaps_runtime(os.fsencode(p), reserved) for p in paths for reserved in [prefix, *worktrees]):
        raise ValueError("runtime artifacts cannot be selected as source")
    return paths


def _validate_snapshot_content(source, tree):
    root = source["source_root"]
    if _scope_git(root, "rev-parse", "HEAD").decode().strip() != source["source_commit"]:
        raise ValueError("source commit changed since capture")
    paths = _snapshot_paths(source)
    if source_inventory(root, source.get("runtime_project", ".")) != source["source_inventory_sha256"]:
        raise ValueError("source change inventory changed; reassess candidate scope")
    staged_diff = _scope_git(root, "--literal-pathspecs", "diff", "--no-ext-diff", "--cached", "--binary", source["source_commit"], "--", *paths)
    if hashlib.sha256(staged_diff).hexdigest() != source["selected_index_sha256"]:
        raise ValueError("source index changed since capture")
    with tempfile.TemporaryDirectory(prefix="review-verify-index-") as temp:
        env = dict(os.environ, GIT_INDEX_FILE=str(Path(temp) / "index"))
        _scope_git(root, "read-tree", source["source_commit"], env=env)
        _scope_git(root, "--literal-pathspecs", "add", "-A", "--", *paths, env=env)
        if _scope_git(root, "write-tree", env=env).decode().strip() != tree:
            raise ValueError("source candidate changed since capture")


def _validate_scope_identity(scope):
    if not all(isinstance(scope.get(key), str) and re.fullmatch(r"(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})", scope[key]) for key in ("base_commit", "candidate_commit", "candidate_tree")):
        raise ValueError("invalid candidate identity")


def _validate_source_root(scope, reviewed_root, expected_source_root):
    source = scope.get("source_snapshot")
    claimed = source["source_root"] if source else reviewed_root
    if Path(claimed).resolve(strict=True) != Path(expected_source_root).resolve(strict=True):
        raise ValueError("review source root does not match trusted source")


def validate_scope(scope: dict[str, object], base_commit: str, diff_path: str, expected_root: str, expected_source_root: str, expected_shark_root: str | None = None, captured: dict | None = None) -> dict[str, object]:
    """Validate the frozen manifest and candidate before accepting a review."""
    root = scope["project_root"]
    if Path(root).resolve(strict=True) != Path(expected_root).resolve(strict=True):
        raise ValueError("review execution root does not match captured scope")
    _validate_source_root(scope, root, expected_source_root)
    expected_runtime = runtime_project(expected_root, expected_shark_root or expected_root)
    if scope.get("runtime_project", ".") != expected_runtime:
        raise ValueError("review Shark project location mismatch")
    _validate_scope_identity(scope)
    candidate, tree, diff = _validate_scope_manifest(scope, base_commit, diff_path, root)
    source = scope.get("source_snapshot")
    subject, marker = _snapshot_marker(root, candidate)
    if (subject == "Shark isolated review snapshot v1" or marker) and not source:
        raise ValueError("isolated snapshot lacks source metadata")
    if source:
        _validate_snapshot_metadata(source, marker, root, candidate, tree, expected_runtime)
        _validate_snapshot_content(source, tree)
    if captured is not None:
        captured["diff"] = diff
    return scope


def bind_scope(metadata: dict[str, object], args: argparse.Namespace) -> str | None:
    metadata.pop("_scope", None)
    metadata.pop("_diff_bytes", None)
    try:
        if not args.scope_file:
            raise ValueError("captured scope is required")
        scope = json.loads(read_review_file(args.scope_file, 32 * 1024 * 1024).decode("utf-8"))
        captured = {}
        validate_scope(scope, args.base_commit, args.diff_path, args.project_root, args.source_root, args.shark_project_root, captured)
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
        metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        metadata["fallback_reason"] = f"scope validation failed: {exc}"
        return str(exc)
    for key in ("candidate_commit", "candidate_tree", "diff_sha256", "changed_files"):
        metadata[key] = scope[key]
    metadata["repair_plan_sha256"] = scope.get("repair_plan_sha256")
    metadata["_scope"] = scope
    metadata["_diff_bytes"] = captured["diff"]
    return None


def _review_identity(scope):
    expected = {"commit": scope["candidate_commit"], "tree": scope["candidate_tree"], "diff_sha256": scope["diff_sha256"]}
    if scope.get("repair_plan_sha256"):
        expected["repair_plan_sha256"] = scope["repair_plan_sha256"]
    return expected


def _matching_identity(identity, scope):
    return isinstance(identity, dict) and all(identity.get(key) == value for key, value in _review_identity(scope).items())


def _valid_finding(finding, fields):
    if not isinstance(finding, dict) or finding.get("severity") not in {"blocker", "non-blocker", "nit"}:
        return False
    if type(finding.get("line")) is not int or finding["line"] < 1:
        return False
    return all(isinstance(finding.get(key), str) and finding[key] for key in fields)


def _angle_findings(name, result, scope):
    identity = result.get("candidate_identity") if isinstance(result, dict) else None
    if not _matching_identity(identity, scope):
        raise ValueError(f"angle {name} has missing or mismatched candidate identity")
    files = result.get("reviewed_files")
    findings = result.get("findings")
    if not isinstance(files, list) or not all(isinstance(item, str) for item in files) or not set(scope["changed_files"]).issubset(files) or not isinstance(findings, list):
        raise ValueError(f"angle {name} has incomplete coverage or malformed findings")
    for finding in findings:
        if not _valid_finding(finding, ("file", "rule", "summary", "diagnosis")):
            raise ValueError(f"angle {name} has malformed finding")
    return findings


def validate_six_angles(data: bytes, scope: dict[str, object]) -> list[dict[str, object]]:
    """Validate actual per-angle evidence before claiming six completed angles."""
    if not data:
        raise ValueError("six-angle results are required")
    angles = json.loads(data.decode("utf-8"))
    if not isinstance(angles, dict) or set(angles) != set("abcdef"):
        raise ValueError("six distinct angle results are required")
    return [finding for name, result in angles.items() for finding in _angle_findings(name, result, scope)]


def _finding_key(finding):
    return tuple(finding[key] for key in ("file", "line", "rule", "summary", "diagnosis"))


def _validate_decision(finding, verdict):
    if not _valid_finding(finding, ("file", "rule", "summary", "diagnosis", "disposition", "reason")):
        raise ValueError("malformed review decision")
    if not finding["disposition"].strip() or not finding["reason"].strip():
        raise ValueError("review decision requires a disposition and reason")
    if verdict.upper().startswith("PASS") and finding["severity"] == "blocker" and finding["disposition"] != "refuted":
        raise ValueError("passing review contains an unresolved blocker")
    if verdict.upper() == "PASS" and finding["disposition"] != "refuted":
        raise ValueError("clean pass contains an undispositioned finding")
    if finding["severity"] == "blocker" and finding["disposition"] == "refuted" and re.search(r"\bnot\s+refuted\b", finding["reason"], re.I):
        raise ValueError("refutation reason contradicts its disposition")


def _check_raw_findings(raw_findings, by_key, verdict):
    for finding in raw_findings:
        decision = by_key.get(_finding_key(finding))
        if not decision:
            raise ValueError("raw finding has no explicit disposition")
        if verdict.upper().startswith("PASS") and finding["severity"] == "blocker" and not _resolved_raw_blocker(decision, verdict):
            raise ValueError("raw blocker has no refutation or justified downgrade")


def _resolved_raw_blocker(decision, verdict):
    if decision["severity"] == "blocker":
        return decision["disposition"] == "refuted"
    return verdict.upper() == "PASS-WITH-TRIAGE" and decision["disposition"] != "refuted"


def validate_review_decisions(decisions: object, scope: dict[str, object], raw_findings: list[dict[str, object]], verdict: str) -> None:
    """Bind explicit dispositions to the candidate and every reported blocker."""
    if not isinstance(decisions, dict) or not isinstance(decisions.get("candidate_identity"), dict):
        raise ValueError("review decisions require candidate identity")
    if not _matching_identity(decisions["candidate_identity"], scope):
        raise ValueError("review decisions identify a different candidate or plan")
    findings = decisions.get("findings")
    if not isinstance(findings, list):
        raise ValueError("review decisions require a findings array")
    by_key = {}
    for finding in findings:
        _validate_decision(finding, verdict)
        key = _finding_key(finding)
        if key in by_key:
            raise ValueError("duplicate review decision")
        by_key[key] = finding
    _check_raw_findings(raw_findings, by_key, verdict)


def valid_review_report(body: str, verdict: str) -> bool:
    """Reject empty, diagnostic, contradictory or structurally incomplete reports."""
    if len(body.strip()) < 80 or re.search(r"\bINCOMPLETE\b", body, re.I | re.ASCII):
        return False
    if verdict.upper().startswith("PASS"):
        normalized = body.lower().replace("no blockers remain", "").replace("no unresolved blockers", "")
        if re.search(r"\bnot\s+refuted\b|\bblockers?\s+remain\b|\bunresolved\s+blockers?\b", normalized, re.ASCII):
            return False
    verdicts = re.findall(r"\bVERDICT\s*:?\s*\*{0,2}\s*:?\s*(PASS(?:-with-triage)?|FAIL)\b", body, re.I | re.ASCII)
    if not verdicts or any(v.upper() != verdict.upper() for v in verdicts):
        return False
    return bool(re.search(r"\b(scope|coverage)\b", body, re.I | re.ASCII) and
                re.search(r"\b(findings|defects)\b", body, re.I | re.ASCII))


def valid_passing_report(body: str, verdict: str) -> bool:
    return verdict.upper().startswith("PASS") and valid_review_report(body, verdict)


def _display(metadata, key, fallback):
    return metadata.get(key) or fallback


def _report_header(metadata: dict[str, object], branch: str = "unknown") -> str:
    fields = [
        f"**Generated:** {metadata['generated']} · **Tool:** `/deep-review` ({metadata['runner_mode']}) · **Branch:** `{branch}`",
        f"**Runner metadata:** `runner_mode={metadata['runner_mode']}` `specialists_completed={metadata['specialists_completed']}` `consolidator_completed={metadata['consolidator_completed']}`",
        f"**Adversarial model:** `{_display(metadata, 'adversarial_model', 'none')}` · **Fallback reason:** {_display(metadata, 'fallback_reason', 'none')}",
        f"**Base commit:** `{metadata['base_commit']}` · **Diff:** `{metadata['diff_path']}`",
        f"**Candidate:** `{metadata.get('candidate_commit', 'unknown')}` · **Tree:** `{metadata.get('candidate_tree', 'unknown')}` · **Diff SHA-256:** `{metadata.get('diff_sha256', 'unknown')}`",
        f"**Repair plan SHA-256:** `{_display(metadata, 'repair_plan_sha256', 'not-applicable')}`",
        f"**Changed files:** {json.dumps(metadata.get('changed_files', 'unknown'), ensure_ascii=True)}",
        f"**Retained scope:** `{_display(metadata, 'retained_scope', 'unavailable')}` · **Angle results:** `{_display(metadata, 'retained_angles', 'unavailable')}`",
        f"**Retained scope SHA-256:** `{_display(metadata, 'scope_sha256', 'unavailable')}`",
        f"**Review decisions:** `{_display(metadata, 'retained_decisions', 'unavailable')}`",
        f"**Review decisions SHA-256:** `{_display(metadata, 'decisions_sha256', 'unavailable')}` · **Angle results SHA-256:** `{_display(metadata, 'angles_sha256', 'unavailable')}`",
        f"**Requested model:** `{_display(metadata, 'requested_model', 'unknown')}` · **Verified effective model:** `{_display(metadata, 'effective_model', 'unknown')}`",
        f"**Verdict:** {metadata['verdict']}",
    ]
    return "# Overall Code Review\n\n" + "\n".join(fields) + "\n\n---\n\n"


def _write_new_bytes(path, data):
    if len(data) > MAX_ARTIFACT_BYTES:
        raise ValueError("retained review artifact exceeds size limit")
    with path.open("xb") as output:
        output.write(data)


def persist_report(path: Path, body: str, metadata: dict[str, object], branch: str = "unknown") -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    # Reserve the report first: exclusive creation also rejects a final symlink.
    # A partial write stays visible for diagnosis and cannot replace old evidence.
    with path.open("x", encoding="utf-8") as report:
        decisions = metadata.pop("_decisions", None)
        if decisions is not None:
            retained_decisions = Path(str(path) + ".decisions.json")
            data = (json.dumps(decisions, indent=2) + "\n").encode("utf-8")
            _write_new_bytes(retained_decisions, data)
            metadata["retained_decisions"] = str(retained_decisions.resolve())
            metadata["decisions_sha256"] = hashlib.sha256(data).hexdigest()
        angle_bytes = metadata.pop("_angle_bytes", None)
        if angle_bytes is not None:
            retained_angles = Path(str(path) + ".angles.json")
            data = angle_bytes
            _write_new_bytes(retained_angles, data)
            metadata["retained_angles"] = str(retained_angles.resolve())
            metadata["angles_sha256"] = hashlib.sha256(data).hexdigest()
        scope = metadata.pop("_scope", None)
        if scope is not None:
            retained = dict(scope)
            retained_diff = Path(str(path) + ".diff")
            _write_new_bytes(retained_diff, metadata.pop("_diff_bytes"))
            retained["diff_path"] = str(retained_diff.resolve())
            retained_scope = Path(str(path) + ".scope.json")
            data = (json.dumps(retained, indent=2) + "\n").encode("utf-8")
            _write_new_bytes(retained_scope, data)
            metadata["retained_scope"] = str(retained_scope.resolve())
            metadata["scope_sha256"] = hashlib.sha256(data).hexdigest()
        content = _report_header(metadata, branch) + body.rstrip() + "\n"
        if len(content.encode("utf-8")) > MAX_ARTIFACT_BYTES:
            raise ValueError("retained review report exceeds size limit")
        report.write(content)


def _cli_prompt(args, scope):
    prompt = read_review_file(args.prompt_file, 2 * 1024 * 1024).decode("utf-8")
    prompt += "\nReturn ONLY one JSON object with report (the complete Markdown report) and decisions (an object with candidate_identity and findings). "
    prompt += "Each finding must have file, positive integer line, severity, rule, summary, diagnosis, disposition and a nonempty reason. "
    prompt += "Use disposition refuted only for a blocker conclusively disproved. Use an empty findings array only when none were found. "
    return prompt + "The exact candidate_identity is: " + json.dumps(_review_identity(scope))


def _parse_cli_packet(output, scope):
    packet = json.loads(output.strip())
    report, decisions = packet["report"], packet["decisions"]
    if not isinstance(report, str) or not valid_adversarial_output(report):
        raise ValueError("malformed report")
    verdict = VERDICT_RE.search(report).group(1).upper()
    validate_review_decisions(decisions, scope, [], verdict)
    if not valid_review_report(report, verdict):
        raise ValueError("incomplete or contradictory report")
    return report, decisions


def _try_cli_candidates(args, candidates, prompt, scope):
    failures = []
    for model in candidates:
        override = args.claude_command if model == "claude" else args.codex_command
        try:
            completed = run_bounded_reviewer(_cli_argv(model, override, prompt), args.project_root, args.timeout)
        except (OSError, subprocess.TimeoutExpired) as exc:
            failures.append(f"{model}: {type(exc).__name__}: {exc}")
            continue
        if completed.returncode != 0:
            failures.append(f"{model}: non-zero exit status {completed.returncode}")
            continue
        try:
            output, decisions = _parse_cli_packet(completed.stdout or "", scope)
            return output, decisions, model, failures
        except (json.JSONDecodeError, KeyError, TypeError, ValueError, AttributeError) as exc:
            failures.append(f"{model}: invalid structured review: {exc}")
    return "", None, None, failures


def run_cli(args: argparse.Namespace) -> int:
    caps = detect_capabilities(
        args.workflow_available,
        args.agent_dispatch_available,
        args.codex_command,
        args.claude_command,
    )
    choice = select_runner(caps, args.host)
    now = datetime.now(timezone.utc).isoformat()
    base = {
        "generated": now,
        "runner_mode": choice["runner_mode"],
        "specialists_completed": 0,
        "consolidator_completed": False,
        "adversarial_model": choice.get("adversarial_model"),
        "fallback_reason": choice.get("fallback_reason"),
        "base_commit": args.base_commit,
        "diff_path": args.diff_path,
        "verdict": "INCOMPLETE",
    }
    base["requested_model"] = args.requested_model
    base["effective_model"] = args.effective_model
    scope_error = bind_scope(base, args)
    if scope_error:
        persist_report(Path(args.review_output_path), "**INCOMPLETE** — " + scope_error, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2
    if choice["runner_mode"] != "adversarial-cli":
        body = (
            "## Review status\n\n"
            "**INCOMPLETE** — no out-of-band automated reviewer was available for this run. "
            "Manual review is diagnostic only and cannot satisfy the pre-merge gate."
        )
        persist_report(Path(args.review_output_path), body, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2

    scope = base["_scope"]
    preferred = str(choice["adversarial_model"])
    candidates = cli_candidates(caps, args.host, preferred)
    try:
        prompt = _cli_prompt(args, scope)
    except (OSError, ValueError, UnicodeError) as exc:
        base["runner_mode"] = "incomplete"
        base["fallback_reason"] = f"review prompt unavailable: {exc}"
        persist_report(Path(args.review_output_path), "**INCOMPLETE** — " + base["fallback_reason"], base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2
    output, selected_decisions, selected_model, failures = _try_cli_candidates(args, candidates, prompt, scope)

    if not output:
        base["adversarial_model"] = None
        base["fallback_reason"] = "; ".join(
            [str(choice["fallback_reason"]), *failures]
        )
        body = "## Review status\n\n**INCOMPLETE** — alternate-model CLI did not produce a complete review."
        persist_report(Path(args.review_output_path), body, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2

    verdict = VERDICT_RE.search(output).group(1).upper()
    if failures:
        base["fallback_reason"] = "; ".join(
            [str(choice["fallback_reason"]), *failures, f"using {selected_model} CLI"]
        )
    base.update(
        {
            "adversarial_model": selected_model,
            "specialists_completed": 0,
            "consolidator_completed": True,
            "verdict": verdict,
        }
    )
    scope_error = bind_scope(base, args)
    if scope_error:
        persist_report(Path(args.review_output_path), "**INCOMPLETE** — " + scope_error, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2
    try:
        if scope != base["_scope"]:
            raise ValueError("captured scope changed during review")
        validate_review_decisions(selected_decisions, base["_scope"], [], verdict)
    except (ValueError, KeyError, TypeError) as exc:
        base["runner_mode"] = "incomplete"
        base["verdict"] = "INCOMPLETE"
        base["fallback_reason"] = f"review decisions no longer match the validated scope: {exc}"
        persist_report(Path(args.review_output_path), "**INCOMPLETE** — " + base["fallback_reason"], base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2
    base["_decisions"] = selected_decisions
    persist_report(Path(args.review_output_path), output, base, args.branch)
    print(json.dumps({**base, "report_path": args.review_output_path, "exit_status": 0}))
    return 0


def _writer_angles(args, metadata, scope_error):
    is_six = args.runner_mode in {"canonical-six-angle", "dispatched-six-angle"}
    incomplete = is_six and (args.specialists_completed != 6 or not args.consolidator_completed)
    try:
        if args.angle_results:
            # Retain exactly the bytes checked, including partial evidence.
            metadata["_angle_bytes"] = read_review_file(args.angle_results, 32 * 1024 * 1024)
    except (OSError, ValueError) as exc:
        metadata["fallback_reason"] = f"angle evidence unavailable: {exc}"
        return [], is_six
    if not is_six or scope_error or incomplete:
        return [], incomplete
    try:
        findings = validate_six_angles(metadata.get("_angle_bytes"), metadata["_scope"])
        return findings, False
    except (OSError, ValueError, KeyError, TypeError) as exc:
        metadata["fallback_reason"] = f"angle evidence invalid: {exc}"
        return [], True


def _writer_decisions(args, metadata, scope_error, findings, body):
    passing = args.verdict.upper() in {"PASS", "PASS-WITH-TRIAGE"}
    failing = args.verdict.upper() == "FAIL"
    invalid_pass = passing and not valid_passing_report(body, args.verdict)
    invalid_fail = failing and not valid_review_report(body, "FAIL")
    if not (passing or failing):
        return invalid_pass, invalid_fail
    try:
        if not args.review_decisions or scope_error:
            raise ValueError("completed review requires candidate-bound decisions")
        decisions = json.loads(read_review_file(args.review_decisions, 32 * 1024 * 1024).decode("utf-8"))
        validate_review_decisions(decisions, metadata["_scope"], findings, args.verdict)
        metadata["_decisions"] = decisions
    except (OSError, ValueError, KeyError, TypeError):
        if passing:
            invalid_pass = True
        else:
            metadata["fallback_reason"] = "failed review decisions were unavailable or malformed"
            invalid_fail = True
    return invalid_pass, invalid_fail


def _writer_metadata(args):
    return {
        "generated": datetime.now(timezone.utc).isoformat(),
        "runner_mode": args.runner_mode,
        "specialists_completed": args.specialists_completed,
        "consolidator_completed": args.consolidator_completed,
        "adversarial_model": args.adversarial_model or None,
        "fallback_reason": args.fallback_reason or None,
        "base_commit": args.base_commit,
        "diff_path": args.diff_path,
        "verdict": args.verdict,
        "requested_model": args.requested_model,
        "effective_model": args.effective_model,
    }


def _writer_runner_state(args, metadata, scope_error, incomplete_six):
    def meaningful(value):
        return bool(value and str(value).strip().strip("`").strip().lower() not in {"", "none", "unknown"})
    incomplete_cli = args.runner_mode == "adversarial-cli" and (args.specialists_completed != 0 or not args.consolidator_completed or not meaningful(args.adversarial_model) or not meaningful(args.fallback_reason))
    incomplete_dispatched = args.runner_mode == "dispatched-six-angle" and not meaningful(args.fallback_reason)
    diagnostic = args.runner_mode in {"manual-diagnostic", "incomplete"}
    invalid_verdict = args.verdict.upper() not in {"PASS", "PASS-WITH-TRIAGE", "FAIL"}
    incomplete = any((scope_error, incomplete_six, incomplete_cli, incomplete_dispatched, diagnostic, invalid_verdict))
    if incomplete:
        if not diagnostic:
            metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        metadata["fallback_reason"] = metadata.get("fallback_reason") or "review evidence is incomplete or diagnostic"
    return bool(incomplete)


def _finish_written_report(args, metadata, body, scope_error, invalid):
    if invalid:
        metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        if not scope_error:
            metadata["fallback_reason"] = "report body or review decisions are incomplete or contradictory"
    persist_report(Path(args.review_output_path), body, metadata, args.branch)
    print(json.dumps({**metadata, "report_path": args.review_output_path}))


def write_report(args: argparse.Namespace) -> int:
    """Persist Workflow/dispatch output using the same metadata contract as CLI fallback."""
    metadata = _writer_metadata(args)
    scope_error = bind_scope(metadata, args)
    angle_findings, incomplete_six = _writer_angles(args, metadata, scope_error)
    incomplete = _writer_runner_state(args, metadata, scope_error, incomplete_six)
    try:
        body = read_review_file(args.body_file, 32 * 1024 * 1024).decode("utf-8")
    except (OSError, ValueError, UnicodeError) as exc:
        metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        metadata["fallback_reason"] = f"review body unavailable: {exc}"
        _finish_written_report(args, metadata, "**INCOMPLETE** — " + metadata["fallback_reason"], scope_error, False)
        return 2
    invalid = any(_writer_decisions(args, metadata, scope_error, angle_findings, body))
    _finish_written_report(args, metadata, body, scope_error, invalid)
    return 2 if incomplete or invalid else 0


def specialist_count(value: str) -> int | str:
    if value == "unknown":
        return value
    try:
        count = int(value)
    except ValueError as exc:
        raise argparse.ArgumentTypeError("expected 0–6 or unknown") from exc
    if count < 0 or count > 6:
        raise argparse.ArgumentTypeError("expected 0–6 or unknown")
    return count


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="command", required=True)
    validate = sub.add_parser("validate-scope")
    validate.add_argument("--scope-file", required=True)
    validate.add_argument("--project-root", required=True)
    validate.add_argument("--source-root", required=True)
    validate.add_argument("--shark-project-root")
    select = sub.add_parser("select")
    select.add_argument("--workflow-available", action=argparse.BooleanOptionalAction, default=None)
    select.add_argument("--agent-dispatch-available", action=argparse.BooleanOptionalAction, default=None)
    select.add_argument("--host", choices=["codex", "claude", "unknown"], default=None)
    select.add_argument("--codex-command")
    select.add_argument("--claude-command")
    run = sub.add_parser("run-cli")
    run.add_argument("--prompt-file", required=True)
    run.add_argument("--project-root", required=True)
    run.add_argument("--source-root", required=True)
    run.add_argument("--shark-project-root")
    run.add_argument("--diff-path", required=True)
    run.add_argument("--base-commit", required=True)
    run.add_argument("--review-output-path", required=True)
    run.add_argument("--branch", default="unknown")
    run.add_argument("--scope-file")
    run.add_argument("--requested-model")
    run.add_argument("--effective-model")
    run.add_argument("--host", choices=["codex", "claude", "unknown"], default=None)
    run.add_argument("--timeout", type=float, default=600)
    run.add_argument("--workflow-available", action=argparse.BooleanOptionalAction, default=None)
    run.add_argument("--agent-dispatch-available", action=argparse.BooleanOptionalAction, default=None)
    run.add_argument("--codex-command")
    run.add_argument("--claude-command")
    write = sub.add_parser("write-report")
    write.add_argument("--body-file", required=True)
    write.add_argument("--project-root", required=True)
    write.add_argument("--source-root", required=True)
    write.add_argument("--shark-project-root")
    write.add_argument("--review-output-path", required=True)
    write.add_argument("--runner-mode", required=True, choices=["canonical-six-angle", "dispatched-six-angle", "adversarial-cli", "manual-diagnostic", "incomplete"])
    write.add_argument("--specialists-completed", required=True, type=specialist_count)
    write.add_argument("--consolidator-completed", required=True, type=lambda value: value.lower() == "true")
    write.add_argument("--adversarial-model")
    write.add_argument("--fallback-reason")
    write.add_argument("--base-commit", required=True)
    write.add_argument("--diff-path", required=True)
    write.add_argument("--verdict", required=True)
    write.add_argument("--branch", default="unknown")
    write.add_argument("--scope-file")
    write.add_argument("--angle-results")
    write.add_argument("--review-decisions")
    write.add_argument("--requested-model")
    write.add_argument("--effective-model")
    return p


def main() -> int:
    args = parser().parse_args()
    if args.command == "validate-scope":
        try:
            scope = json.loads(read_review_file(args.scope_file, 32 * 1024 * 1024).decode("utf-8"))
            validate_scope(scope, scope["base_commit"], scope["diff_path"], args.project_root, args.source_root, args.shark_project_root)
        except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
            print(json.dumps({"valid": False, "error": str(exc)}))
            return 2
        print(json.dumps({"valid": True, "candidate_tree": scope["candidate_tree"], "diff_sha256": scope["diff_sha256"]}))
        return 0
    if args.command == "select":
        caps = detect_capabilities(args.workflow_available, args.agent_dispatch_available, args.codex_command, args.claude_command)
        print(json.dumps({"capabilities": asdict(caps), **select_runner(caps, args.host)}))
        return 0
    if args.command == "write-report":
        return write_report(args)
    return run_cli(args)


if __name__ == "__main__":
    sys.exit(main())
