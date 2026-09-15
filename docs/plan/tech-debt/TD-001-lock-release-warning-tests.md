---
tech_debt_key: TD-001
title: Test gate lock-release warning paths
status: identified
category: testing
severity: low
size: S
---

# Test gate lock-release warning paths

## Description

Add a narrow test seam or helper that can make `RunLock.Release` fail in the
gate-persistence coordinator and run-resume paths. Assert that each path logs
the intended warning and preserves its primary result or error.

## Scope

Do not change production lock-release behavior merely to make it testable.
Keep the seam local to release-error reporting and cover both callers.
