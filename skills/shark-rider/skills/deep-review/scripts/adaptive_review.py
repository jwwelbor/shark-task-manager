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
import sys
import tempfile
import time
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path


VERDICT_RE = re.compile(r"\bVERDICT\s*:\s*(PASS(?:-with-triage)?|FAIL)\b", re.I)
FINDINGS_RE = re.compile(r"\b(FINDINGS|FINDING[S]?\s+TABLE)\b", re.I)
SCOPE_RE = re.compile(r"\b(REVIEWED\s+SCOPE|CHANGED\s+FILES|SCOPE)\b", re.I)
MAX_REVIEW_OUTPUT_BYTES = 8 * 1024 * 1024


def run_bounded_reviewer(argv, cwd, timeout):
    """Run an untrusted reviewer with bounded in-memory stdout/stderr."""
    process = subprocess.Popen(
        argv,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        start_new_session=True,
    )
    selector = selectors.DefaultSelector()
    buffers = {process.stdout: bytearray(), process.stderr: bytearray()}
    for stream in buffers:
        selector.register(stream, selectors.EVENT_READ)
    deadline = time.monotonic() + timeout
    exceeded = False
    timed_out = False
    kill_deadline = None

    def kill_group():
        nonlocal kill_deadline
        if kill_deadline is not None:
            return
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        kill_deadline = time.monotonic() + 5

    while selector.get_map():
        if kill_deadline is not None and time.monotonic() >= kill_deadline:
            for stream in list(buffers):
                if stream in selector.get_map():
                    selector.unregister(stream)
            break
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            timed_out = True
            kill_group()
            remaining = 5
        events = selector.select(min(remaining, 0.25))
        if not events and process.poll() is not None:
            for stream in list(buffers):
                selector.unregister(stream)
            break
        for key, _ in events:
            chunk = os.read(key.fileobj.fileno(), 65536)
            if not chunk:
                selector.unregister(key.fileobj)
                continue
            buffer = buffers[key.fileobj]
            if len(buffer) + len(chunk) > MAX_REVIEW_OUTPUT_BYTES:
                buffer.extend(chunk[: MAX_REVIEW_OUTPUT_BYTES - len(buffer)])
                exceeded = True
                kill_group()
            else:
                buffer.extend(chunk)
        if (exceeded or timed_out) and process.poll() is not None:
            # Continue draining already-buffered pipe data, but never retain
            # more than the fixed cap.
            continue
    process.wait()
    selector.close()
    return subprocess.CompletedProcess(
        argv,
        1 if exceeded or timed_out else process.returncode,
        bytes(buffers[process.stdout]).decode("utf-8", errors="replace"),
        bytes(buffers[process.stderr]).decode("utf-8", errors="replace"),
    )


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
    """Require enough structure to prevent an auth/error/partial transcript passing."""
    normalized = text.strip()
    return bool(
        len(normalized) >= 80
        and VERDICT_RE.search(normalized)
        and FINDINGS_RE.search(normalized)
        and SCOPE_RE.search(normalized)
    )


def _cli_argv(model: str, override: str | None, prompt: str) -> list[str]:
    if override:
        return [*shlex.split(override), prompt]
    if model == "codex":
        return ["codex", "exec", "--sandbox", "read-only", "--skip-git-repo-check", prompt]
    return ["claude", "-p", prompt, "--permission-mode", "plan", "--output-format", "text"]


def validate_scope(path: str, base_commit: str, diff_path: str) -> dict[str, object]:
    """Validate the frozen manifest and candidate before accepting a review."""
    scope = json.loads(Path(path).read_text(encoding="utf-8"))
    root = scope["project_root"]
    def git(*args):
        return subprocess.run(["git", *args], cwd=root, check=True,
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout
    candidate = git("rev-parse", "HEAD").decode().strip()
    tree = git("rev-parse", "HEAD^{tree}").decode().strip()
    if git("status", "--porcelain", "--untracked-files=all"):
        raise ValueError("candidate worktree is dirty")
    if candidate != scope["candidate_commit"] or tree != scope["candidate_tree"]:
        raise ValueError("candidate changed since scope capture")
    if base_commit != scope["base_commit"] or diff_path != scope["diff_path"]:
        raise ValueError("review arguments do not match captured scope")
    diff = Path(diff_path).read_bytes()
    expected_diff = git("diff", "--binary", base_commit, candidate)
    if diff != expected_diff or hashlib.sha256(diff).hexdigest() != scope["diff_sha256"]:
        raise ValueError("diff changed since scope capture")
    manifest = [os.fsdecode(p) for p in git("diff", "--name-only", "-z", base_commit, candidate).split(b"\0") if p]
    if manifest != scope["changed_files"] or len(manifest) != scope["changed_file_count"]:
        raise ValueError("scope manifest does not match candidate")
    source = scope.get("source_snapshot")
    if source:
        source_root = source["source_root"]
        if source["candidate_commit"] != candidate or source["candidate_tree"] != tree:
            raise ValueError("source snapshot identity does not match candidate")
        def source_git(*args, env=None):
            return subprocess.run(["git", "-C", source_root, *args], env=env, check=True,
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.decode().strip()
        if source_git("rev-parse", "HEAD") != source["source_commit"]:
            raise ValueError("source commit changed since capture")
        paths = source["selected_paths"]
        if not isinstance(paths, list) or not paths or not all(isinstance(p, str) and p and not Path(p).is_absolute() and '..' not in Path(p).parts and p != '.' for p in paths):
            raise ValueError("invalid source snapshot paths")
        with tempfile.TemporaryDirectory(prefix="review-verify-index-") as temp:
            env = dict(os.environ, GIT_INDEX_FILE=str(Path(temp) / "index"))
            source_git("read-tree", source["source_commit"], env=env)
            source_git("--literal-pathspecs", "add", "-A", "--", *paths, env=env)
            if source_git("write-tree", env=env) != tree:
                raise ValueError("source candidate changed since capture")
    return scope


def bind_scope(metadata: dict[str, object], args: argparse.Namespace) -> str | None:
    try:
        if not args.scope_file:
            raise ValueError("captured scope is required")
        scope = validate_scope(args.scope_file, args.base_commit, args.diff_path)
        if hasattr(args, "project_root") and Path(args.project_root).resolve() != Path(scope["project_root"]).resolve():
            raise ValueError("review execution root does not match captured scope")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, subprocess.CalledProcessError) as exc:
        metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        metadata["fallback_reason"] = f"scope validation failed: {exc}"
        return str(exc)
    for key in ("candidate_commit", "candidate_tree", "diff_sha256", "changed_files"):
        metadata[key] = scope[key]
    return None


def _report_header(metadata: dict[str, object], branch: str = "unknown") -> str:
    fields = [
        f"**Generated:** {metadata['generated']} · **Tool:** `/deep-review` ({metadata['runner_mode']}) · **Branch:** `{branch}`",
        f"**Runner metadata:** `runner_mode={metadata['runner_mode']}` `specialists_completed={metadata['specialists_completed']}` `consolidator_completed={metadata['consolidator_completed']}`",
        f"**Adversarial model:** `{metadata.get('adversarial_model') or 'none'}` · **Fallback reason:** {metadata.get('fallback_reason') or 'none'}",
        f"**Base commit:** `{metadata['base_commit']}` · **Diff:** `{metadata['diff_path']}`",
        f"**Candidate:** `{metadata.get('candidate_commit', 'unknown')}` · **Tree:** `{metadata.get('candidate_tree', 'unknown')}` · **Diff SHA-256:** `{metadata.get('diff_sha256', 'unknown')}`",
        f"**Changed files:** {json.dumps(metadata.get('changed_files', 'unknown'), ensure_ascii=True)}",
        f"**Requested model:** `{metadata.get('requested_model') or 'unknown'}` · **Verified effective model:** `{metadata.get('effective_model') or 'unknown'}`",
        f"**Verdict:** {metadata['verdict']}",
    ]
    return "# Overall Code Review\n\n" + "\n".join(fields) + "\n\n---\n\n"


def persist_report(path: Path, body: str, metadata: dict[str, object], branch: str = "unknown") -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(_report_header(metadata, branch) + body.rstrip() + "\n", encoding="utf-8")


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

    prompt = Path(args.prompt_file).read_text(encoding="utf-8")
    preferred = str(choice["adversarial_model"])
    candidates = cli_candidates(caps, args.host, preferred)
    failures: list[str] = []
    output = ""
    selected_model = preferred
    for model in candidates:
        override = args.claude_command if model == "claude" else args.codex_command
        argv = _cli_argv(model, override, prompt)
        try:
            completed = run_bounded_reviewer(argv, args.project_root, args.timeout)
        except (OSError, subprocess.TimeoutExpired) as exc:
            failures.append(f"{model}: {type(exc).__name__}")
            continue
        candidate_output = (completed.stdout or "").strip()
        if completed.returncode != 0:
            failures.append(f"{model}: non-zero exit status {completed.returncode}")
            continue
        if not valid_adversarial_output(candidate_output):
            failures.append(f"{model}: empty, malformed, or partial reviewer output")
            continue
        output = candidate_output
        selected_model = model
        break

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
    persist_report(Path(args.review_output_path), output, base, args.branch)
    print(json.dumps({**base, "report_path": args.review_output_path, "exit_status": completed.returncode}))
    return 0


def write_report(args: argparse.Namespace) -> int:
    """Persist Workflow/dispatch output using the same metadata contract as CLI fallback."""
    metadata = {
        "generated": datetime.now(timezone.utc).isoformat(),
        "runner_mode": args.runner_mode,
        "specialists_completed": args.specialists_completed,
        "consolidator_completed": args.consolidator_completed,
        "adversarial_model": args.adversarial_model or None,
        "fallback_reason": args.fallback_reason or None,
        "base_commit": args.base_commit,
        "diff_path": args.diff_path,
        "verdict": args.verdict,
    }
    metadata["requested_model"] = args.requested_model
    metadata["effective_model"] = args.effective_model
    scope_error = bind_scope(metadata, args)
    incomplete_six = args.runner_mode in {"canonical-six-angle", "dispatched-six-angle"} and (args.specialists_completed != 6 or not args.consolidator_completed)
    incomplete_cli = args.runner_mode == "adversarial-cli" and (args.specialists_completed != 0 or not args.consolidator_completed or not args.adversarial_model)
    diagnostic = args.runner_mode in {"manual-diagnostic", "incomplete"}
    if scope_error or incomplete_six or incomplete_cli or diagnostic:
        if not diagnostic:
            metadata["runner_mode"] = "incomplete"
        metadata["verdict"] = "INCOMPLETE"
        metadata["fallback_reason"] = metadata.get("fallback_reason") or "review evidence is incomplete or diagnostic"
    body = Path(args.body_file).read_text(encoding="utf-8")
    persist_report(Path(args.review_output_path), body, metadata, args.branch)
    print(json.dumps({**metadata, "report_path": args.review_output_path}))
    return 2 if scope_error or incomplete_six or incomplete_cli or diagnostic else 0


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="command", required=True)
    select = sub.add_parser("select")
    select.add_argument("--workflow-available", action=argparse.BooleanOptionalAction, default=None)
    select.add_argument("--agent-dispatch-available", action=argparse.BooleanOptionalAction, default=None)
    select.add_argument("--host", choices=["codex", "claude", "unknown"], default=None)
    select.add_argument("--codex-command")
    select.add_argument("--claude-command")
    run = sub.add_parser("run-cli")
    run.add_argument("--prompt-file", required=True)
    run.add_argument("--project-root", required=True)
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
    write.add_argument("--review-output-path", required=True)
    write.add_argument("--runner-mode", required=True, choices=["canonical-six-angle", "dispatched-six-angle", "adversarial-cli", "manual-diagnostic", "incomplete"])
    write.add_argument("--specialists-completed", required=True, type=int)
    write.add_argument("--consolidator-completed", required=True, type=lambda value: value.lower() == "true")
    write.add_argument("--adversarial-model")
    write.add_argument("--fallback-reason")
    write.add_argument("--base-commit", required=True)
    write.add_argument("--diff-path", required=True)
    write.add_argument("--verdict", required=True)
    write.add_argument("--branch", default="unknown")
    write.add_argument("--scope-file")
    write.add_argument("--requested-model")
    write.add_argument("--effective-model")
    return p


def main() -> int:
    args = parser().parse_args()
    if args.command == "select":
        caps = detect_capabilities(args.workflow_available, args.agent_dispatch_available, args.codex_command, args.claude_command)
        print(json.dumps({"capabilities": asdict(caps), **select_runner(caps, args.host)}))
        return 0
    if args.command == "write-report":
        return write_report(args)
    return run_cli(args)


if __name__ == "__main__":
    sys.exit(main())
