#!/usr/bin/env python3
"""Select and run the strongest available deep-review runner.

The script is deliberately independent of Claude Code/Codex APIs.  Host skills
use ``select`` for capability routing; ``run-cli`` is the bounded, read-only
out-of-band fallback and report writer.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path


VERDICT_RE = re.compile(r"\bVERDICT\s*:\s*(PASS(?:-with-triage)?|FAIL)\b", re.I)
FINDINGS_RE = re.compile(r"\b(FINDINGS|FINDING[S]?\s+TABLE)\b", re.I)
SCOPE_RE = re.compile(r"\b(REVIEWED\s+SCOPE|CHANGED\s+FILES|SCOPE)\b", re.I)
MAX_REVIEW_OUTPUT_BYTES = 8 * 1024 * 1024


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


def _report_header(metadata: dict[str, object], branch: str = "unknown") -> str:
    fields = [
        f"**Generated:** {metadata['generated']} · **Tool:** `/deep-review` ({metadata['runner_mode']}) · **Branch:** `{branch}`",
        f"**Runner metadata:** `runner_mode={metadata['runner_mode']}` `specialists_completed={metadata['specialists_completed']}` `consolidator_completed={metadata['consolidator_completed']}`",
        f"**Adversarial model:** `{metadata.get('adversarial_model') or 'none'}` · **Fallback reason:** {metadata.get('fallback_reason') or 'none'}",
        f"**Base commit:** `{metadata['base_commit']}` · **Diff:** `{metadata['diff_path']}`",
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
    override = args.claude_command if choice["adversarial_model"] == "claude" else args.codex_command
    argv = _cli_argv(str(choice["adversarial_model"]), override, prompt)
    try:
        with tempfile.TemporaryFile() as stdout_file, tempfile.TemporaryFile() as stderr_file:
            process = subprocess.Popen(
                argv,
                cwd=args.project_root,
                stdout=stdout_file,
                stderr=stderr_file,
                text=False,
            )
            try:
                process.wait(timeout=args.timeout)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise
            stdout_file.seek(0)
            stderr_file.seek(0)
            stdout_bytes = stdout_file.read(MAX_REVIEW_OUTPUT_BYTES + 1)
            stderr_bytes = stderr_file.read(MAX_REVIEW_OUTPUT_BYTES + 1)
            completed = subprocess.CompletedProcess(
                argv,
                process.returncode,
                stdout_bytes.decode("utf-8", errors="replace"),
                stderr_bytes.decode("utf-8", errors="replace"),
            )
            if len(stdout_bytes) > MAX_REVIEW_OUTPUT_BYTES or len(stderr_bytes) > MAX_REVIEW_OUTPUT_BYTES:
                completed = subprocess.CompletedProcess(
                    argv,
                    1,
                    completed.stdout[:MAX_REVIEW_OUTPUT_BYTES],
                    completed.stderr[:MAX_REVIEW_OUTPUT_BYTES],
                )
    except (OSError, subprocess.TimeoutExpired) as exc:
        base["fallback_reason"] = f"{choice['fallback_reason']}; CLI failed: {type(exc).__name__}"
        body = "## Review status\n\n**INCOMPLETE** — alternate-model CLI did not produce a complete review."
        persist_report(Path(args.review_output_path), body, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path}))
        return 2

    output = (completed.stdout or "").strip()
    if completed.returncode != 0 or not valid_adversarial_output(output):
        reason = "non-zero exit status" if completed.returncode else "empty, malformed, or partial reviewer output"
        base["fallback_reason"] = f"{choice['fallback_reason']}; {reason}"
        body = "## Review status\n\n**INCOMPLETE** — alternate-model CLI did not produce a complete review."
        persist_report(Path(args.review_output_path), body, base, args.branch)
        print(json.dumps({**base, "report_path": args.review_output_path, "exit_status": completed.returncode}))
        return 2

    verdict = VERDICT_RE.search(output).group(1).upper()
    base.update({"specialists_completed": 0, "consolidator_completed": True, "verdict": verdict})
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
    persist_report(Path(args.review_output_path), Path(args.body_file).read_text(encoding="utf-8"), metadata, args.branch)
    print(json.dumps({**metadata, "report_path": args.review_output_path}))
    return 0


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
