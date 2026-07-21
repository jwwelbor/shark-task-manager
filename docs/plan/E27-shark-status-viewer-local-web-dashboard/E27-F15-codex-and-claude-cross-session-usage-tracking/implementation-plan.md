# Cross-Session Codex and Claude Token Tracking

## Summary

Add provider-neutral agent usage tracking to Shark in two phases:

1. Capture exact token usage from Codex and Claude processes launched by Shark.
2. Replace `shark web` with `shark serve`, combining the existing dashboard with a loopback OTLP receiver for interactive CLI sessions.

Lifecycle hooks provide session registration and project correlation; native JSON/OTel events remain the source of truth for token counts. Collection is forward-only and never stores prompts, responses, transcripts, or tool arguments.

## Key Changes

### Usage persistence and normalization

- Add schema migration 29 with separate `agent_sessions` and `agent_usage_events` tables; do not overload claim-oriented `work_sessions`.
- Identify sessions by provider plus provider session ID, with optional Shark entity/run attribution.
- Store normalized token buckets:
  - uncached input
  - cache-read input
  - cache-creation input
  - output
  - reasoning output
  - provider-reported cost, when available
- Preserve provider semantics in raw columns while deriving total input for aggregation.
- Deduplicate Codex turns and Claude requests using their native turn/request identifiers.
- Default retention to 90 days through `agent_usage.retention_days`; `0` disables expiry. Prune on writes and `shark serve` startup.

### Shark-launched agents

- Add `--json` to Codex execution and parse `turn.completed.usage`.
- Continue using Claude's JSON output and extract its usage and cost metadata.
- Extend dispatch results with normalized usage events while preserving the existing assistant-text `Stdout` behavior.
- Persist usage with exact run, entity, provider, model, duration, and exit-status attribution.
- Collection from Shark-launched agents must work without `shark serve`.

### Interactive sessions and hooks

- Configure both providers' native OTel logs exporters to send HTTP/protobuf to `http://127.0.0.1:<port>/v1/logs`.
- Configure lifecycle hooks:
  - Codex: `SessionStart`, turn-scoped `Stop`, `SubagentStart`, `SubagentStop`, and compaction events.
  - Claude: `SessionStart`, `SessionEnd`, `SubagentStart`, and `SubagentStop`.
- Add `shark hook relay --provider codex|claude`, which reads the native hook JSON from stdin, resolves the Shark project, strips content-bearing fields, and registers the lifecycle event with `shark serve`.
- Make the relay fail open with a short timeout so unavailable telemetry can never block an agent session.
- Accept OTLP usage only for sessions registered to the project served by the current Shark process. Reject project mismatches and unknown sessions to prevent another project's global CLI telemetry from entering the database.
- Interactive sessions remain unattributed unless a Shark entity is explicitly supplied, such as through `SHARK_ENTITY_KEY`; never guess from potentially ambiguous active claims.
- Do not infer a Codex session end. Display its first/last-seen timestamps; use Claude's real `SessionEnd` when supplied.

### `shark serve` and receiver contract

- Remove `shark web` and introduce `shark serve [--port N] [--no-open]`.
- Serve the dashboard, viewer API, lifecycle endpoint, and OTLP `/v1/logs` endpoint on one loopback-only listener.
- Use a stable port: `--port`, then `serve.port`, then `7777`. Fail clearly if occupied; never fall back to another port.
- Read legacy `web.port` only when `serve.port` is absent and emit a deprecation warning.
- Enforce protobuf content type, request-size limits, bounded processing, idempotent writes, and graceful malformed-event responses.
- Disable provider prompt logging and ignore all content-bearing OTel attributes even if received.
- Report the listener URL, OTLP endpoint, project identity, and retention policy at startup.

### Reporting experience

- Add a top-level **Usage** view to the existing dashboard.
- Show:
  - date, provider, model, source, entity, and attribution filters
  - input, output, cache, reasoning, session, duration, and reported-cost summaries
  - usage-over-time and provider/model breakdowns
  - paginated sessions with a metadata-only detail view
- Distinguish receiver online, no registered provider sessions, no events yet, and recently received events.
- Add a compact usage card to the main dashboard and attributed totals on entity pages; both link to the filtered Usage view.
- Explain cache-token differences between providers and label cost as unavailable rather than estimating Codex cost.
- Ensure keyboard navigation, visible focus, screen-reader labels, non-color-only status, and WCAG-compliant contrast.

## Interfaces and Setup Documentation

- Add read-only viewer endpoints for usage summary, session listing, session detail, and receiver/provider health.
- Add a local lifecycle registration endpoint used only by `shark hook relay`.
- Publish an agent-oriented integration guide containing:
  - exact Codex and Claude OTel settings
  - idempotent hook configuration snippets
  - instructions to merge rather than overwrite existing settings
  - privacy guarantees and fields collected
  - verification steps using `shark serve` and the Usage health panel
  - troubleshooting for busy ports, missing hooks, project mismatches, and unavailable receivers
- Shark will not edit Codex or Claude configuration automatically; a user can point an AI coding agent at this guide to perform and verify the integration.

## Test Plan

- Test schema migration, uniqueness, normalization, aggregation, attribution, and 90-day pruning.
- Use captured fixtures for Codex JSONL, Claude JSON, and both providers' OTLP protobuf events.
- Verify cached input is not double-counted and absent cost/reasoning fields remain null.
- Test duplicate delivery, malformed protobuf, oversized bodies, unknown sessions, project mismatches, and receiver restart behavior.
- Test dispatchers with fake executables, including successful runs, nonzero exits, partial JSON, and preservation of assistant text.
- Verify the hook relay strips transcript/content fields, exits successfully when the receiver is down, and routes only to the resolved project.
- Verify `shark serve` fixed-port precedence, busy-port failure, browser behavior, graceful shutdown, legacy config fallback, and removal of `shark web`.
- Add API/UI contract tests plus browser UAT for filters, empty/offline states, session drill-down, entity links, keyboard operation, and responsive laptop layout.
- Run an end-to-end acceptance scenario for one Shark-launched and one interactive session from each provider, confirming exact provider-reported totals and zero stored content.

## Assumptions

- This is forward-only; existing CLI transcripts and historical sessions are not imported.
- Interactive capture requires `shark serve` to be running.
- Only one project-local receiver uses the configured port at a time.
- OTel and native JSON provide token truth; hooks provide lifecycle and project correlation only.
- The dedicated Usage view and contextual links follow the `cx-designer` consultation recommendation.
