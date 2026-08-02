#!/usr/bin/env python3
"""
Hook Name: shark-config-guardrail.py
Event: PreToolUse (Bash)
Purpose: Block `shark admin init` and `shark cloud init` against the live
project checkout.

Both commands rewrite .sharkconfig.json. In this repo that file routes to a
real Turso cloud database via env-var placeholders ($SHARK_DB_BACKEND,
$SHARK_DB_URL) rather than the local shark-tasks.db file. `admin init --force`
(or `cloud init`) silently replaces that pointer with local-SQLite defaults,
which looks like success but cuts off access to the real project data with no
error until a later command mysteriously "can't find" an entity that
obviously exists.

Incident (2026-08-01, bug B048): a research subagent ran
`shark admin init --non-interactive --force` mid-task to get a clean
environment for reproduction, not realizing it would rewrite the live
project's config. Caught only because a subsequent `shark get` failed with a
misleading "repository not found" error; recovered via `git checkout --
.sharkconfig.json`.

Fixes this class of mistake structurally: agents (main loop or subagents)
that need to run mutating shark commands should use an isolated scratch
project (scripts/shark-scratch-env.sh) instead of the live checkout. This
hook denies the two specific commands unconditionally (it has no cwd check —
it matches on command text alone, not on whether the shell is currently
inside the repo) and points at that script rather than relying on every
future agent reading a warning in CLAUDE.md.
"""

import json
import re
import sys

# Matches `shark admin init` / `shark cloud init` however the binary is
# invoked (`shark`, `./bin/shark`, `$HOME/go/bin/shark`, etc.), with global
# persistent flags (--db, --config, --json, -v, --no-color, ...) optionally
# interleaved between the tokens, and regardless of shell chaining (&&, ;, |).
# This is a plain substring match with no execution context — it also denies
# commands that merely mention the phrase (e.g. in a quoted string or
# comment), which is an intentional fail-safe trade-off.
DANGEROUS_PATTERN = re.compile(
    r"\bshark\b(?:\s+-{1,2}\S+)*\s+(admin|cloud)(?:\s+-{1,2}\S+)*\s+init\b"
)

DENY_MESSAGE = (
    "Blocked: `shark admin init` / `shark cloud init` rewrite "
    ".sharkconfig.json, which routes to this project's real Turso cloud "
    "database via env-var placeholders. Running either against the live "
    "checkout can silently replace that pointer with local-SQLite defaults "
    "(see B048 incident, 2026-08-01).\n\n"
    "If you need an isolated shark project to reproduce a bug or test CLI "
    "behavior, use:\n"
    "  scripts/shark-scratch-env.sh [name]\n"
    "It bootstraps a fresh, fully isolated project under /tmp — run mutating "
    "shark commands there instead.\n\n"
    "If you specifically intend to reconfigure THIS project's database "
    "backend, that's a deliberate, rare action — ask the user to run it "
    "directly rather than doing it from an agent."
)


def deny(reason):
    print(
        json.dumps(
            {
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": "deny",
                    "permissionDecisionReason": reason,
                }
            }
        )
    )
    sys.exit(0)


def main():
    try:
        event = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        print(f"[shark-config-guardrail] ERROR: failed to parse hook input: {e}", file=sys.stderr)
        return  # fail open

    if not isinstance(event, dict):
        return

    if event.get("tool_name") != "Bash":
        return

    tool_input = event.get("tool_input", {})
    if not isinstance(tool_input, dict):
        return

    command = tool_input.get("command", "")
    if not isinstance(command, str) or not command:
        return

    if DANGEROUS_PATTERN.search(command):
        deny(DENY_MESSAGE)


if __name__ == "__main__":
    main()
