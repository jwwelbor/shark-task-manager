---
feature_key: E34-F02-evidence-based-demo-script-skill
epic_key: E34
title: Evidence-Based Demo Script Skill
description: Provide a portable Shark Rider demo-script recipe that turns accepted epic or feature evidence into an accurate, readiness-aware walkthrough without treating completion status, an owner override, or a demo as a UAT verdict.
---

# Evidence-Based Demo Script Skill

**Feature Key**: E34-F02-evidence-based-demo-script-skill

---

## Epic

- **Epic PRD**: [Epic](../epic.md)
- **Epic Architecture**: [Architecture](../architecture.md)

---

## Goal

### Problem
Project teams need a clear way to demonstrate delivered work to stakeholders, but existing demo-script guidance is coupled to a specific web stack, GitHub Actions, Playwright, screenshots, and repository layout. Those assumptions exclude valid Shark projects such as CLIs, APIs, libraries, batch pipelines, infrastructure, and background processes. Without a portable workflow, demo claims can drift from the actual delivered scope or omit the evidence needed for a presenter to show the result confidently.

### Solution
Add an explicit Mode 3 Rider action, `/shark-rider demo <epic|feature> [--draft]`, backed by an embedded `demo-script` skill and template. The action will gather Shark entity state, acceptance/readiness evidence, and project guidance; organize delivered work into stakeholder-oriented scenarios; and produce a walkthrough whose claims are tied to observable evidence. It must support appropriate evidence for each product surface rather than assuming screenshots are always available.

### Impact
- Demo scripts can be generated for epic and feature scopes across supported project types, including non-UI products.
- Every normal-mode scenario has a traceable source requirement and verified observable evidence; incomplete work is identified as not demonstrated.
- Contract-only behavior, open activation obligations, and owner-overridden findings remain visible and cannot be presented as verified end-to-end delivery.
- Demo preparation remains separate from UAT and does not add a mandatory default workflow status.

---

## User Stories

### Must-Have Stories

**Story 1**: As a project presenter, I want an accurate demo script for a completed epic or feature so that I can show stakeholder value without overstating incomplete or unverified work.

**Acceptance Criteria**:
- [ ] The action accepts epic, feature, or sprint keys and reads their relevant scope, statuses, acceptance criteria, notes, related documents, completed child work, latest UAT assessor verdict, separate owner decision, open conditions, and integration activation state.
- [ ] The resulting script groups epic work into user journeys rather than a raw feature inventory, and feature work into its outcomes and relevant integrations.
- [ ] Incomplete work is explicitly placed under “Not demonstrated” rather than presented as complete.
- [ ] Completed status is treated as context rather than proof; a contract-only obligation or owner-overridden rejection cannot become a verified normal-mode claim.

> **Scope note:** Extended 2026-08-31 to include sprint targets, tracking the
> sprint-demo capability shipped in PR #186 (2026-08-17) for E19 sprint work —
> see the shark decision note on E34-F02 for the decision record.

**Story 2**: As a maintainer of a CLI, API, library, pipeline, or infrastructure project, I want evidence requirements that match my product surface so that I can demonstrate the delivered behavior without fabricating screenshots.

**Acceptance Criteria**:
- [ ] The workflow supports screenshots/recordings for UI, command transcripts for CLIs, requests/responses plus resulting state for APIs, runnable examples for SDKs, artifacts/data for pipelines, health/metric evidence for infrastructure, and trigger/log/result evidence for background processes.
- [ ] The workflow reports an evidence gap when documented project guidance is insufficient instead of inventing commands, credentials, environments, or capture tools.

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Provide a portable Rider recipe
   - **Description**: Ship a `demo` Rider verb plus an embedded `demo-script` skill and reference template. The verb retrieves the portable instructions through `shark skill get`, not through a new Go command, database table, or entity type.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `/shark-rider demo <epic-key>` and `/shark-rider demo <feature-key>` resolve the documented recipe.
     - [ ] Rider router/help/capability references, bundle manifest, and skill documentation expose the new capability.

2. **REQ-F-002**: Build an evidence-backed scenario map
   - **Description**: For each scenario, record stakeholder value, source acceptance criteria, prerequisites/demo data, presenter actions, expected observable result, evidence type/path, acceptance/readiness classification, reset or recovery instructions, and known limitations.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Every product claim traces to a Shark entity or a project product document.
     - [ ] Normal mode verifies that referenced evidence exists and identifies its environment/date.
     - [ ] `--draft` may produce a script with uncaptured evidence, clearly marking those gaps.
     - [ ] The script separates `Demonstrated now`, `Not demonstrated / pending integration`, and `Accepted risks and overrides`.
     - [ ] `Demonstrated now` excludes contract-only behavior, open activation obligations, and behavior accepted through an override of a blocking assessor verdict.

3. **REQ-F-003**: Persist demo artifacts and references
   - **Description**: Write the generated artifact at `docs/demos/<entity-key>/demo-script.md`, with evidence kept under its `evidence/` directory, then attach the script through Shark’s related-document and reference-note contracts.
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] The created demo script can be discovered from the relevant epic or feature.
     - [ ] Any defects found while preparing the demo are returned as triage candidates rather than created automatically.

4. **REQ-F-004**: Preserve acceptance and integration readiness
   - **Description**: Read the latest independent UAT verdict separately from the owner's decision, then account for open conditions, I-##/X-## gate modes, activation owners, closure keys, counterpart status, and the declared review basis before classifying a claim as demonstrable.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] The script preserves a blocking assessor verdict when an owner records `override-accept`; it reports both facts instead of rewriting the verdict.
     - [ ] A `contract-only` claim remains under `Not demonstrated / pending integration` until its activation owner provides live production-path evidence and closes the tracked obligation.
     - [ ] An entity with no complete observable scenario produces an evidence/decomposition gap and a triage candidate rather than an invented walkthrough.

### Non-Functional Requirements

1. **REQ-NF-001**: Stack and environment neutrality
   - **Description**: The generic skill must not assume a framework, package manager, browser, deployment provider, credentials, or capture tool. It may run only commands documented by the project.
   - **Measurement**: Recipe and template tests cover non-UI evidence paths and verify missing guidance becomes an explicit gap.

2. **REQ-NF-002**: Safe, truthful output
   - **Description**: Demo scripts must not include secrets or hardcoded environment-specific endpoints, and every expected result must be observable.
   - **Measurement**: Validation rejects missing evidence in normal mode and checks generated scripts for required traceability and safety boundaries.

---

## Acceptance Criteria

**Scenario 1: Generate a verified feature demo**
- **Given** a feature with completed tasks, acceptance criteria, and documented project guidance
- **When** a presenter runs `/shark-rider demo <feature-key>`
- **Then** Rider produces an evidence-backed scenario walkthrough at the documented demo path
- **And** each demonstrated claim maps to committed scope and observable evidence.

**Scenario 2: Produce a safe draft when evidence cannot be captured**
- **Given** an eligible epic or feature whose architecture documentation does not explain how to run or capture a required surface
- **When** the presenter runs the action with `--draft`
- **Then** the generated script records the evidence gap and uncaptured steps
- **And** it does not invent setup commands, credentials, deployments, or proof.

**Scenario 3: Preserve the UAT boundary**
- **Given** a project that uses the default feature and epic workflows
- **When** demo-script support is installed
- **Then** it remains an explicit Rider action with no required default status transition
- **And** it does not replace or repeat UAT approval, review verdicts, or workflow advancement.

**Scenario 4: Preserve conditional and overridden acceptance evidence**
- **Given** a completed entity with a contract-only interaction, an open activation obligation, or an owner override of a blocking assessor verdict
- **When** a presenter runs `/shark-rider demo <entity-key>`
- **Then** currently verified behavior appears under `Demonstrated now`
- **And** pending integration appears under `Not demonstrated / pending integration`
- **And** the original verdict, owner decision, accepted risk, and open keys appear under `Accepted risks and overrides`.

---

## Integration Contract

E34-F03, Deliverable Feature Decomposition and Staged Integration Acceptance,
is the producer of the acceptance/readiness model consumed here. E34-F02 should
follow E34-F03 in specification order and must not define competing meanings for
assessor verdict, owner decision, `live`, `contract-only`, activation owner, or
closure state.

**Consumes: I-01** from E34-F03. The authoritative shape source is
[`E34-interaction-map.md#i-01-readiness-evidence-shape`](../E34-interaction-map.md#i-01-readiness-evidence-shape),
and the shared structural contract-test pointer is
**TC-I-01-READINESS-SYMMETRY** at
`internal/cli/commands/interaction_prompts_test.go::TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY`.
E34-F02 consumes this exact nine-field readiness shape read-only:

| Field | I-01 value |
|---|---|
| `assessor_verdict` | Independent UAT assessment, recorded without owner-decision rewrite |
| `owner_decision` | Separate approval or `override-accept` decision with conditions |
| `open_conditions` | Open activation and any recorded conditions remain visible |
| `gate_mode` | `contract-only` until E34-F02 proves live production-path use |
| `activation_owner` | E34-F02 |
| `closure_key` | E34-F02 |
| `counterpart_status` | `E34-F03` is `completed` in the live Shark read on 2026-08-25; re-read it at review/UAT time and record any change rather than copying a stale value |
| `review_basis` | Accumulated E34 branch with the map and both feature specifications present |
| `demonstrability_disposition` | `pending-integration` until live wiring closes; no override makes it demonstrated-now |

The predeclared `contract-only` edge is E34-F03 (completed producer) to
E34-F02 (activation owner and closure key); its review basis is the accumulated
E34 branch, the interaction map, both feature specifications, and shared
**TC-002**. The assessor verdict remains independent, an owner decision may
only be `override-accept` with explicit conditions, and the demonstrability
disposition remains `pending-integration` until E34-F02 proves the real caller
chain. The map table supplies the counterpart identities and shared contract
evidence; they are not extra readiness fields. E34-F02 must not create a twin test,
assign a new I-## ID, or redefine this producer contract. When the activation
remains open, E34-F02 classifies the claim as `pending-integration`; completion
markers, a demo, or an owner override do not substitute for closed
production-path evidence.

---

## Out of Scope

1. **A mandatory demo workflow gate**
   - **Why**: UAT establishes acceptance; demo generation explains how to truthfully present what was delivered. Making demos mandatory would block projects that do not need an interactive demonstration.
   - **Future**: Projects may adopt an opt-in workflow profile if a release-specific demo gate is needed.

2. **Automatic environment provisioning or capture setup**
   - **Why**: The portable workflow must not install dependencies, deploy, create accounts, or invent credentials.
   - **Future**: Individual projects may document their own safe setup/capture procedures for the skill to follow.

3. **Automatic backlog creation for discovered discrepancies**
   - **Why**: Backlog capture still requires duplicate search and user confirmation.
   - **Future**: Present discrepancies as `/shark-rider triage` candidates.

4. **Defining feature decomposition or UAT policy**
   - **Why**: E34-F03 owns deliverable-decomposition and staged-integration acceptance. This feature consumes its results and reports demonstrability without becoming an acceptance authority.
   - **Future**: None; preserve the producer/consumer boundary.

---

## Success Metrics

1. **Traceable verified scenarios**
   - **What**: The share of normal-mode demo scenarios with both a committed-scope source and verified evidence.
   - **Target**: 100% of scenarios generated in normal mode.
   - **Measurement**: Recipe validation checks scenario traceability and evidence existence before the script is marked verified.

2. **Cross-surface portability**
   - **What**: The evidence models supported by the reusable recipe.
   - **Target**: Support UI, CLI, API, library/SDK, batch/data pipeline, infrastructure, and background-process evidence without a stack-specific prerequisite.
   - **Measurement**: Contract tests and template examples cover each evidence category.

3. **Truthful readiness classification**
   - **What**: The share of normal-mode claims whose assessor verdict, owner decision, evidence, and integration activation state all support their output section.
   - **Target**: 100% of normal-mode claims.
   - **Measurement**: Contract tests cover fully demonstrated, contract-only, open-condition, and override-accepted entities.

---

*Last Updated*: 2026-07-21
