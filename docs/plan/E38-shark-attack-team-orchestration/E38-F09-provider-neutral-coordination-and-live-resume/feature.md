---
feature_key: E38-F09-provider-neutral-coordination-and-live-resume
epic_key: E38
title: Provider-Neutral Coordination and Live Resume
description: Capture the Shark Attack v2 core from dev-artifacts/shark-attack-v2-plan/implementation-plan.md: independent coordination/topology selection, live question routing, same-worker resume or bounded replacement, prompt-hash provenance, Codex and Claude adapters, skill restructuring, and authored/embedded parity. This is a new v2 increment beyond completed E38-F04 and E38-F07; open design decisions remain unresolved.
---

# Provider-Neutral Coordination and Live Resume

**Feature Key**: E38-F09-provider-neutral-coordination-and-live-resume

## Triage breadcrumb

The first Shark Attack proved Batch coordination but did not demonstrate a live
developer-to-council question loop or reliable continuation in the same worker
context. The chair still had to relay prompts and answers manually.

Capture the v2 core as a provider-neutral coordination protocol with independent
coordination-level and execution-topology selection, bounded specialist routing,
same-worker follow-up where supported, immutable handoff plus replacement where
it is not, parent-owned Shark authority, and prompt-hash provenance at the host
adapter boundary. Initial installed-host coverage is Codex and Claude Code.

Source: `dev-artifacts/shark-attack-v2-plan/implementation-plan.md`, especially
the operating model, adapter contract, Tranche C, and Phase 1.

This is an incremental v2 capability beyond completed E38-F04 and E38-F07; it
must reuse their authority boundary instead of reopening or duplicating their
delivered scope. Decisions about capability-profile names, worker-owned child
pulling, and the canonical authored/embedded direction remain unresolved.
