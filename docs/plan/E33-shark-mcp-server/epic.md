---
epic_key: E33
title: Shark MCP Server
description: A `shark mcp` command that exposes shark's entity operations (epics, features, tasks, bugs, change-cards, tech-debt, ideas, notes, sprints, tags) as a Model Context Protocol server, giving AI clients a typed, schema-driven interface instead of guessing at CLI flag combinations.
size: XL
---

# Shark MCP Server

**Epic Key**: E33

---

## Goal

### Problem

AI agents (Claude, Codex, Cursor, etc.) that drive shark today have to interact with it as a stdout-parsing CLI. The CLI is large (8+ entity families, dozens of subcommands, repeatable flags, positional vs flag forms, `--json` toggle, `--field` extraction), and agents discover it through trial and error — running `--help`, parsing prose, then constructing commands that fail on the first try because of subtle flag shape mismatches. The friction shows up as wasted turns, malformed entity creates, and brittle scripts that re-parse human-formatted output. The shark web dashboard already demonstrates that the same service layer can expose a *structured* surface to non-CLI consumers; we want the same for AI clients.

### Solution

Add a `shark mcp` command that launches a Model Context Protocol server backed by the existing service layer (the same `TaskService`, `FeatureService`, `EpicService`, etc. that `shark web` uses). Each entity operation becomes an MCP tool with a JSON schema for its arguments and a structured response — so the client knows up front what the operation accepts, what it returns, and what errors mean, without parsing `--help` or stdout. Default transport is stdio (matches how Claude Code and most MCP clients connect to local servers); HTTP/SSE is a later-phase consideration if remote use cases emerge.

### Impact

- AI clients invoke entity operations as typed tool calls instead of CLI command strings, eliminating the "trial-and-error on the CLI" loop
- One source of truth for shark's surface: services already exist, MCP becomes the third entry point alongside CLI and web (no duplicate business logic)
- Discoverability via `tools/list` — the client enumerates capabilities at session start instead of grepping help text
- Errors surface as structured MCP error responses (typed codes, messages) instead of exit codes + stderr that the agent has to interpret
- Lowers the bar for new shark integrations — any MCP-aware editor or agent picks up shark without learning the CLI

---

## Business Value

**Rating**: High

Shark's primary user is AI agents driving the AI-DLC workflow. Today every agent integration pays a per-session tax of learning CLI shape (or worse, forgetting it between sessions and re-learning). An MCP server collapses that tax to a single schema fetch and unlocks integrations with the growing set of MCP-aware clients (Claude Code, Codex, Cursor, IDE extensions) without us shipping per-client adapters. Strategic alignment is high: shark exists to be driven by agents, and MCP is the emerging standard for tools-as-context.

---

## Quick Reference

**Primary Users**: AI agents (Claude Code, Codex, Cursor, IDE-embedded assistants) and the humans configuring them.

**Key Capabilities** (initial scope — refinement will firm these up):
- `shark mcp` launches a stdio MCP server backed by the existing service layer
- Tools cover the entity surface: `create_*`, `get_*`, `list_*`, `update_*`, `delete_*`, `status_*`, `note_*`, `link_*`, `tag_*`, `context_*`, `search`, `sprint_*` for the entities shark already supports (epic, feature, task, bug, change-card, tech-debt, idea, note, sprint, tag)
- JSON schemas on tool inputs so clients validate arguments before invoking
- Structured responses (the same entity DTOs the HTTP API returns) — no stdout parsing
- Resources expose static reference material (workflow definition, status metadata, entity-type catalog) so clients can read configuration without running tools
- Honors the same project-root auto-detection and config (`.sharkconfig.json`, Turso vs local backend) as the CLI

**Success Criteria** (to be sharpened in refinement):
- Claude Code (or another MCP-aware client) can drive a complete shark workflow — create epic → feature → tasks → advance statuses → complete — using only MCP tool calls, no shell-out to `./bin/shark`
- All entity types currently writable via `shark create` are creatable via the MCP server
- Tool schemas are derived from the service layer (not hand-maintained in two places) so they cannot drift
- The MCP server reuses the service layer; no business logic lives in the MCP package

**Out of Scope (initial release)**:
- Remote/HTTP transport (stdio only initially; SSE/HTTP is a follow-on if demand appears)
- Auth/multi-tenant (single-user, single-project, runs on the same machine as the client)
- Exposing destructive admin operations (`admin init`, schema migrations) as tools — those stay CLI-only for now
- A new entity surface — MCP must not become the place where new capabilities land before the CLI/web have them

---

## Architectural Anchor

The shape mirrors `shark web` (E27):

| Layer | CLI | Web (E27) | MCP (E33) |
|---|---|---|---|
| Entry point | `shark <cmd>` | `shark web` → HTTP server | `shark mcp` → stdio MCP server |
| Wiring | `internal/cli` | `cmd/server` + `internal/viewer` | new `internal/mcp` (server) + `cmd/mcp` or extend existing |
| Business logic | Services (`internal/services`) | Same services | Same services |
| Transport | argv + stdout | HTTP/JSON | MCP/JSON-RPC over stdio |

Refinement will pick the Go MCP SDK, decide whether the server binary is folded into `cmd/server` (analogous to how web was) or split, and produce the tool/resource catalog with concrete schemas.

---

## Epic Components

The following supporting documents will be produced during refinement (D-artifacts):

- **personas.md** — primary persona is the AI agent; secondary is the human operator configuring the MCP client
- **user-journeys.md** — "agent drives a feature end-to-end via MCP" and "operator wires shark mcp into Claude Code"
- **requirements.md** — functional (tool catalog, schemas, transport, project-root resolution, error model) and non-functional (latency budget per tool call, no business logic duplication, parity with CLI for the entity surface)
- **scope.md** — confirms the out-of-scope boundaries above and any new exclusions surfaced during refinement
- **success-metrics.md** — measurable success criteria once requirements are firm

---

*Last Updated*: 2026-05-12
