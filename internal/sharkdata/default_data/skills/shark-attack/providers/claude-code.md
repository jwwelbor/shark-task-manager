# Claude Code Provider Capability Reference

Scope: Claude Code CLI only (REQ-F-013). Copilot and Antigravity are
E38-F10's scope, documentation-backed until run on an installed host. Every
claim below cites a captured evidence marker from the locally installed CLI
(REQ-F-012 — a missing capability is data that causes a documented
fallback, never license to invent a command). Captured against installed
Claude Code `2.1.220` on 2026-08-03; re-verify against the installed
version before relying on any claim below.

## Supported Operations

- **Spawn** — `claude -p "<prompt>" --output-format json` runs a single non-interactive turn and exits; `claude --bg "<prompt>"` (alias `--background`) starts the session as a background agent and returns immediately, to be managed with `claude agents`. **Evidence:** `claude --help` (captured 2026-08-03) — `-p, --print  "Print response and exit..."` and `--bg, --background  "Start the session as a background agent and return immediately (manage with 'claude agents')"`.
- **Follow-up** — `claude --resume <session-id> "<new prompt>"` (short form `-r`) continues a specific session with a new instruction, supplied via the top-level `[prompt]` argument. **Evidence:** `claude --help` (captured 2026-08-03) — `-r, --resume [value]  "Resume a conversation by session ID..."` combined with the documented `Usage: claude [options] [command] [prompt]` and `prompt  Your prompt` argument.
- **Progress / wait** — `claude --output-format stream-json` (with `--print`) streams turn-by-turn events while a run is in flight, `--include-partial-messages` adds partial-message chunks, and `claude agents --json` polls the state of any dispatched session (interactive or background). **Evidence:** `claude --help` (captured 2026-08-03) — `--output-format <format> ... "stream-json" (realtime streaming)` and `--include-partial-messages`; `claude agents --help`'s `--json  "Print active sessions (interactive and background) as a JSON array and exit"`.
- **List (active sessions)** — `claude agents --json` enumerates active sessions (interactive and background); `--all` additionally includes completed background sessions; `--cwd <path>` filters to sessions started under a given path. **Evidence:** `claude agents --help` (captured 2026-08-03) — `--json  "Print active sessions (interactive and background) as a JSON array and exit (for scripting; does not require a TTY)"` and `--all  "With --json: also include completed background sessions"`.
- **Isolation** — `claude -w [name]` (`--worktree`) creates a new git worktree for the session, giving a write worker its own filesystem; `--tmux` additionally runs the worktree session in its own tmux session when requested (requires `--worktree`). **Evidence:** `claude --help` (captured 2026-08-03) — `-w, --worktree [name]  "Create a new git worktree for this session (optionally specify a name)"`.
- **Resume** — `claude --resume [value]` (interactive picker or a specific session ID), `claude --session-id <uuid>` (pin a specific session id up front), and `claude --fork-session` (resume into a new session id rather than reusing the original) all continue a prior bounded context. **Evidence:** `claude --help` (captured 2026-08-03) — `-r, --resume [value]`, `--session-id <uuid>  "Use a specific session ID for the conversation..."`, and `--fork-session  "When resuming, create a new session ID instead of reusing the original..."`.

## Unsupported Operations

- **Interrupt (stop a live in-flight session)** — No flag or `claude agents` subcommand stops or kills a running interactive, print-mode, or background session; only OS-level process signaling was observed, which is not a Claude Code-specific command and is out of scope for this reference. **Evidence:** none captured — absent from the full `claude --help` output (no `kill`, `stop`, `interrupt`, or `cancel` token anywhere) and absent from `claude agents --help`'s option list, both captured 2026-08-03.

## Sequential Fallback

Whenever Interrupt is unsupported, or Follow-up/Isolation/Resume cannot be
evidenced on the installed host, the parent MUST fall back to `Sequential`
execution: dispatch one Claude Code worker at a time in the parent's own
checkout (per `context/operating-model.md`'s Sequential default), never
inventing an unverified `claude` command or `claude agents` subcommand to
substitute for the missing capability (REQ-F-012). Because `claude agents`
does support listing, a parent that needs to confirm a background session
is still running may poll `claude agents --json` while continuing to run
additional work sequentially — polling for status is not itself a
parallel-topology authorization (see `context/operating-model.md`).

## Evidence Log

Commands captured against installed Claude Code `2.1.220` on 2026-08-03.
Each entry below is the exact CLI surface a claim above cites; re-run these
commands to re-verify this reference against a different installed version
before relying on it.

1. `claude --help` — top-level `Usage: claude [options] [command] [prompt]`; relevant options: `-p, --print`, `--bg, --background`, `-r, --resume [value]`, `--fork-session`, `--session-id <uuid>`, `-w, --worktree [name]`, `--tmux`, `--output-format <format>` (`text`, `json`, `stream-json`), `--include-partial-messages`; top-level `Commands:` list: `agents, auth, auto-mode, doctor, gateway, install, mcp, plugin|plugins, project, setup-token, ultrareview, update|upgrade`. No `kill`, `stop`, `interrupt`, or `cancel` token appears anywhere in the full output.
2. `claude agents --help` — "Manage background agents"; `--json  "Print active sessions (interactive and background) as a JSON array and exit..."`, `--all  "With --json: also include completed background sessions"`, `--cwd <path>  "Show only background sessions started under <path>"`. No kill/stop/interrupt option is present.
