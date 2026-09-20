---
feature_key: E40-F13-copilot-backed-model-routing-for-e40-lifecycle-ben
epic_key: E40
title: Copilot-backed model routing for E40 lifecycle benchmarks
description: Enable E40 lifecycle benchmarks to execute the provider and model selected by each Shark workflow route through GitHub Copilot CLI. The present adapter supports Claude/Codex-oriented paths but not Copilot; the global provider-command override cannot model per-route selection. This feature must add a provider-aware Copilot execution path, normalize Copilot JSONL into E40's bounded worker control/usage contract, pin and retain actual Copilot model identity, constrain execution to isolated scenario scratch roots, and support versioned model-routing profiles for valid baseline/variant capture. Evidence: Copilot CLI is installed; current E40 six-family preflight is blocked on runtime adapter and six I-05 bundle directories; B064 fixed the prior caller-path defect; E38-F10 recorded the earlier missing Copilot host evidence. Do not invent USD cost when Copilot only exposes credits or usage.
---
## Triage capture

GitHub Copilot CLI is now available on the benchmark host, but the E40 lifecycle execution path cannot treat it as a dispatch-selected provider. The adapter must execute the model requested by the Shark route, not force a single harness-wide command.

Observed boundaries:

- Copilot supports non-interactive prompts, pinned model selection, and JSONL output; E40 requires a bounded terminal control envelope and measurement projection before it can use that output.
- The provider/model mapping must be profile-owned and immutable in the run manifest, so a model swap becomes a valid, comparable variant rather than hidden drift.
- Worker tools must be limited to the isolated scenario scratch root; Shark remains the owner of claims, heartbeats, transitions, and releases.
- Discover the authenticated account model catalog and verify usage/cost fields with a real one-step canary. Record unavailable USD cost honestly; never substitute zero or infer it from credits.
- Existing B064 repaired the original provider-backed caller-chain failure. Existing E38-F10 recorded lack of Copilot host evidence. Neither supplies Copilot routing and E40 capture support.

This is an intake record, not a specification or task breakdown.
