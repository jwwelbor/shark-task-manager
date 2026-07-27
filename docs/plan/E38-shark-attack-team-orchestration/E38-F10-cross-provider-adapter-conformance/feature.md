---
feature_key: E38-F10-cross-provider-adapter-conformance
epic_key: E38
title: Cross-Provider Adapter Conformance
description: Capture Phase 2 of dev-artifacts/shark-attack-v2-plan/implementation-plan.md: feature-detected GitHub Copilot and Google Antigravity adapters, common provider conformance fixtures, explicit unsupported capabilities, and safe sequential or replacement-worker fallbacks. Documentation-backed adapter policy remains decision-gated until installed-host verification.
---

# Cross-Provider Adapter Conformance

**Feature Key**: E38-F10-cross-provider-adapter-conformance

## Triage breadcrumb

The v2 core must not imply that every host supports the same messaging, resume,
interrupt, isolation, or model-preference behavior. Copilot and Antigravity were
not locally executable during plan verification, so prose claims alone cannot
establish conformance.

Capture feature-detected provider mappings, one common adapter fixture, explicit
unsupported values, and safe sequential or replacement-worker fallbacks for
GitHub Copilot CLI and Google Antigravity. Installed environments must provide
captured native evidence before an adapter is treated as verified.

Source: `dev-artifacts/shark-attack-v2-plan/implementation-plan.md`, especially
the provider capability matrix, cross-host conformance fixture, and Phase 2.

The documentation-backed experimental-provider policy remains an open decision.
This feature must not grant providers authority or assume overlapping writes are
safe without isolation.
