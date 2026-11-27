---
epic_key: E41
title: Multi-Harness Model Routing
description: Resolve portable Shark workflow execution across qualified harnesses with complexity based routing, independent review and reproducible benchmarking.
---

# Multi-Harness Model Routing

**E41** · Last Updated 2026-11-25

## Problem statement and business justification

A workflow author can request a model and reasoning effort in Shark today, yet the worker may receive only the model. At the assessed baseline, the core runner dispatched through two CLIs, neither received the declared effort, and Copilot execution existed in the benchmark adapter. Selecting a different reviewer role also gave a gate owner no proof that a different model reviewed the work. [Current-state assessment](./00-research-report.md)

A routing profile lets an operator change which approved worker performs a step while the author keeps the same workflow. Recording the requested route, the selected route and what the worker actually used lets a reviewer verify independence and a benchmark maintainer explain differences between runs.

## Goals and success criteria

At epic acceptance, demonstrate all four outcomes against the [acceptance matrix](./09-test-plan.md).

| Goal | Release measure |
|---|---|
| Switch worker assignments without editing the workflow | Two named profiles each resolve all three declared complexities against identical workflow and prompt inputs. Repeating a plan with the same pinned inputs yields the same decisions and digest. |
| Honor the operator’s execution policy | The qualification record covers Claude Code, Codex CLI and GitHub Copilot CLI. Every mode advertised as supported has versioned real-CLI evidence for model, explicit effort where supported, permissions and result handling. |
| Trust review and recovery | Across the review and interruption fixtures, admit zero reviews with missing required proof, replay zero uncertain writes automatically and apply each accepted result at most once. |
| Migrate one workflow at a time | All legacy regression, mixed-version and rollback cases pass. E40 can read retained version 1 artifacts and reproduce the selected assignments without editing source workflow bytes. |

## Scope

E41 covers author-declared complexity, named route exceptions, profile selection and explanation, qualified CLI execution, candidate-bound review, bounded recovery, Rider inline execution, and E40 comparison and migration. E35 owns business outcomes and status transitions. New harness protocols, automatic complexity classification and changes to benchmark publication policy fall outside this epic. Feature PRDs reference this document for business context. [Scope boundaries](./scope.md) · [Requirements catalog](./requirements.md)

## Constraints and assumptions

- Keep legacy behavior until a workflow opts in. Roll out route selection and compatibility first, then qualified adapters, then independent review and recovery. Rider and E40 adoption depends on compatible consumers. [Implementation phases](./08-implementation-phases.md)
- The initial qualification targets are Anthropic through Claude Code, OpenAI through Codex CLI and GitHub-managed models through Copilot CLI. The operator approves versions, identities, disposable environments and spend before authenticated qualification. Implementation requires separate approval. [Design decisions](./decisions.md)
- Treat compatibility, account entitlement and current availability as distinct facts. A cached model listing alone cannot establish that a worker ran. Initialize the advisory executable/version inventory during `shark admin init` when missing, or on the first routing lookup if still absent; allow an operator to refresh it explicitly. Revalidate the selected route at launch. [Discovery scope](./scope.md#discovery)
- For independent review, the proposed default is a fresh session and a different canonical model. An explicitly declared set of separation dimensions must pass in full. After an interruption, mutating work with uncertain effects pauses for reconciliation. Bounded replay requires affirmative evidence of safety. [Proposed architecture](./02-architecture.md)

## Stakeholder impact

Workflow authors classify a step once and can see which route wins. Project operators control allowed workers, credentials, permissions and budgets, and receive a reason when work pauses. Gate owners can trace a review to the exact candidate and its producers. Benchmark maintainers can change the execution profile and still compare the same workload. [Stakeholder responsibilities](./personas.md) · [User journeys](./user-journeys.md)

## High-level acceptance criteria (UAT scenarios)

| ID | Action | Observable result |
|---|---|---|
| UAT-01 | An author previews low, medium and high steps under each of two profiles, then selects a conflicting pin or unsupported effort. | Each valid preview identifies the winning route and its source. An invalid selection identifies the conflict before launch. Preview performs zero claims, transitions or inference calls. |
| UAT-02 | An operator initializes Shark, checks or refreshes the local harness inventory, then runs a fixture through a qualified CLI with explicit model, effort and permissions. | The advisory inventory reports installed executable paths and versions; refresh updates the snapshot. Launch revalidates the selected route, the worker enforces the permission boundary, and a strict identity mismatch prevents admission. |
| UAT-03 | A gate owner requests independent review, then repeats with an alias of the author’s model, missing author evidence or a changed candidate. | A fresh eligible reviewer can review the pinned candidate. Each negative fixture fails admission under the default policy. |
| UAT-04 | An external worker fails before launch, or a worker loses its lease after a possible write. | The first case can use the next eligible configured route within budget. The second pauses for reconciliation. Restarting after result acceptance never applies it twice. |
| UAT-05 | Rider exhausts external routes after safe failures, then repeats with a policy denial or a valid negative workflow outcome. | The eligible host attempts the keyed step inline exactly once in the first case and records inline mode. The other cases end with zero inline attempts. |
| UAT-06 | A benchmark maintainer compares two profiles, then changes a frozen input before a rerun. | Both original runs retain the same workload hashes and separate route evidence. E40 rejects the mismatched rerun for comparison and preserves its publication gates. |
| UAT-07 | An operator upgrades an unmodified workflow, opts one workflow in, then rolls back at a run boundary using the release guide. | Existing assignments and result parsing match the legacy fixtures. An incompatible consumer rejects strict work before launch. Prior results remain readable after rollback. |
