# Codex Provider Capability Reference

Scope: OpenAI Codex CLI only (REQ-F-013). Copilot and Antigravity are
E38-F10's scope, documentation-backed until run on an installed host. Every
claim below cites a captured evidence marker from the locally installed CLI
(REQ-F-012 — a missing capability is data that causes a documented
fallback, never license to invent a command). Captured against installed
`codex-cli 0.146.0` on 2026-08-03; re-verify against the installed version
before relying on any claim below.

## Supported Operations

- **Spawn** — `codex exec [PROMPT] [--json] [-o <FILE>]` runs Codex non-interactively: `[PROMPT]` (or stdin) is the initial instruction, `--json` streams JSONL events, `-o, --output-last-message <FILE>` writes the agent's final message. **Evidence:** `codex exec --help` (captured 2026-08-03) — `exec` is listed as "Run Codex non-interactively" with `[PROMPT]`, `--json`, and `-o, --output-last-message <FILE>` all present.
- **Post-exit resume** — `codex exec resume [SESSION_ID] [PROMPT]` (alias `codex resume [SESSION_ID] [PROMPT]`) sends a new prompt into a previously recorded session only after the prior invocation has exited; `--last` selects the most recent session without an id. It is a cold refresh, never a same-live-worker follow-up. **Evidence:** `codex exec resume --help` (captured 2026-08-03) — "Resume a previous session by id or pick the most recent with --last", accepting a `[PROMPT]` argument documented as "Prompt to send after resuming the session".
- **Progress / wait** — `codex exec --json` prints a JSONL event stream while the run is in flight, and the parent process's own blocking wait on subprocess exit is the wait mechanism; no separate poll command is offered or needed. **Evidence:** `codex exec --help` (captured 2026-08-03) — `--json  "Print events to stdout as JSONL"`.
- **Isolation (directed)** — `-C, --cd <DIR>` points Codex at a working root the parent already provisioned (e.g. an existing git worktree), `--add-dir <DIR>` grants additional writable directories, and `-s, --sandbox {read-only,workspace-write,danger-full-access}` bounds what the run may touch. Codex does not create the isolated directory itself — the parent must provision it first (see Automatic isolation, below). **Evidence:** `codex exec --help` (captured 2026-08-03) — `-C, --cd <DIR>`, `--add-dir <DIR>`, and `-s, --sandbox <SANDBOX_MODE>` are all present.
- **Resume** — `codex resume [SESSION_ID] [--last] [--all]` (interactive) and `codex exec resume [SESSION_ID] [--last]` (non-interactive) continue a previously recorded session; `codex fork [SESSION_ID] [--last]` branches a previous session into a new one. **Evidence:** `codex resume --help` and `codex fork --help` (captured 2026-08-03) — both accept `[SESSION_ID]`/`--last` and describe resuming/forking "a previous interactive session".

## Unsupported Operations

- **Live follow-up (same still-running worker)** — No captured Codex CLI operation delivers a prompt to a `codex exec` worker while its original invocation remains in flight. `codex exec resume` is only valid after that invocation exits, so a live consultation must use the bounded-replacement fallback rather than attempting resume. **Evidence:** none captured — `codex exec resume --help` documents resuming a previous recorded session, while `codex exec --help` exposes no live-message/follow-up operation.
- **Interrupt (stop a live in-flight run)** — No flag or subcommand stops a running `codex exec`/`codex` process short of OS-level process signaling; that is not a Codex-specific command and is out of scope for this reference. **Evidence:** none captured — absent from the full `codex --help` command list (`exec, review, login, logout, mcp, plugin, mcp-server, app-server, remote-control, completion, update, doctor, sandbox, debug, apply, resume, archive, delete, unarchive, fork, cloud, exec-server, features, help`) and absent from `codex exec --help`'s option list, both captured 2026-08-03.
- **List (live in-flight workers)** — `codex resume --all` / `codex fork --all` enumerate previously *recorded* sessions (an on-disk history), not currently-running processes; no command was found that lists live Codex workers. **Evidence:** none captured — `codex resume --help`/`codex fork --help` (captured 2026-08-03) describe `--all` as "Show all sessions (disables cwd filtering...)" against the recorded-session picker, not a live-process list, and no separate list-active-workers surface exists in the top-level `codex --help` command list.
- **Automatic isolation (self-provisioned worktree)** — Codex has no equivalent of a `--worktree`-style flag; it only accepts a caller-supplied working directory (see Isolation (directed), above). **Evidence:** none captured — the token `worktree` does not appear anywhere in the full `codex --help` output, captured 2026-08-03.

## Sequential Fallback

Whenever Live follow-up, Interrupt, List, or Automatic isolation is unsupported, or
Isolation/Post-exit resume cannot be evidenced on the installed host, the
parent MUST fall back to `Sequential` execution: dispatch one Codex worker
at a time in the parent's own checkout (per `context/operating-model.md`'s
Sequential default), never inventing an unverified `codex` command to
substitute for a missing capability (REQ-F-012). A write wave that needs
isolation may still use Isolation (directed) above — a parent-provisioned
working directory — without that authorizing a parallel topology; see
`context/operating-model.md` for the isolation/dependency rule.

## Evidence Log

Commands captured against installed `codex-cli 0.146.0` on 2026-08-03. Each
entry below is the exact CLI surface a claim above cites; re-run these
commands to re-verify this reference against a different installed version
before relying on it.

1. `codex --help` — top-level `Commands:` list: `exec, review, login, logout, mcp, plugin, mcp-server, app-server, remote-control, completion, update, doctor, sandbox, debug, apply, resume, archive, delete, unarchive, fork, cloud, exec-server, features, help`. No `interrupt`, `kill`, `stop`, `cancel`, or `worktree` token appears anywhere in the full output.
2. `codex exec --help` — `[PROMPT]` argument ("Initial instructions for the agent... If stdin is piped..."), `--json` ("Print events to stdout as JSONL"), `-o, --output-last-message <FILE>`, `-C, --cd <DIR>`, `--add-dir <DIR>`, `-s, --sandbox <SANDBOX_MODE>` (`read-only`, `workspace-write`, `danger-full-access`).
3. `codex exec resume --help` — "Resume a previous session by id or pick the most recent with --last"; accepts `[SESSION_ID]` and a `[PROMPT]` "Prompt to send after resuming the session".
4. `codex resume --help` — "Resume a previous interactive session (picker by default; use --last to continue the most recent)"; `--all` "Show all sessions (disables cwd filtering and shows CWD column)".
5. `codex fork --help` — "Fork a previous interactive session (picker by default; use --last to fork the most recent)"; same `[SESSION_ID]`/`--last`/`--all` shape as `resume`.
6. `codex app-server --help` / `codex remote-control --help` — both explicitly marked `[experimental]`; neither documents a stable interrupt or live-worker-list capability, so neither is cited as evidence for any Supported Operations claim above.
