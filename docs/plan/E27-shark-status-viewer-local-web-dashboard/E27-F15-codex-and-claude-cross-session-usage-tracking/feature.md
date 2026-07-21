---
feature_key: E27-F15
epic_key: E27
title: Codex and Claude Cross-Session Usage Tracking
description: Track metadata-only token usage and session correlation for Shark-launched and interactive Codex and Claude CLI sessions.
status: draft
---

# Codex and Claude Cross-Session Usage Tracking

## Problem

Shark records claim timing and outcomes, but it does not capture the input and
output tokens consumed by Codex and Claude across CLI sessions. Operators cannot
compare agent usage, connect cost and token volume to Shark work, or investigate
usage trends from the existing dashboard.

## Proposed capability

Capture provider-reported usage from native JSON and OpenTelemetry events, use
lifecycle hooks only for safe session and project correlation, and expose the
metadata through a dedicated Usage view. Extend the existing viewer server into
the stable `shark serve` process that hosts both the dashboard and local telemetry
receiver. Never persist prompts, responses, transcripts, or tool arguments.

## Placement and related foundations

This is a user-facing dashboard feature under E27. It reuses the Codex and Claude
dispatchers from E22 and the OpenTelemetry foundation from E23; neither existing
epic currently records provider token usage.

## Approved design

See [implementation-plan.md](implementation-plan.md) for the complete agreed
implementation and test plan.

*Captured through Shark Rider triage on 2026-07-21.*
