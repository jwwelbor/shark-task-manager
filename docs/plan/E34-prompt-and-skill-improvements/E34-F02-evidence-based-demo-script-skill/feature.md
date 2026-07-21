---
feature_key: E34-F02-evidence-based-demo-script-skill
epic_key: E34
title: Evidence-Based Demo Script Skill
description: Provide a portable Shark Rider demo-script recipe that turns completed epic or feature evidence into an accurate, evidence-backed walkthrough without treating a demo as a UAT gate.
---

# Evidence-Based Demo Script Skill

**Feature Key**: E34-F02-evidence-based-demo-script-skill

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Project teams need a clear way to demonstrate delivered work to stakeholders, but existing demo-script guidance is coupled to a specific web stack, GitHub Actions, Playwright, screenshots, and repository layout. Those assumptions exclude valid Shark projects such as CLIs, APIs, libraries, batch pipelines, infrastructure, and background processes. Without a portable workflow, demo claims can drift from the actual delivered scope or omit the evidence needed for a presenter to show the result confidently.

### Solution
Add an explicit Mode 3 Rider action, `/shark-rider demo <epic|feature> [--draft]`, backed by an embedded `demo-script` skill and template. The action will gather Shark entity state and project guidance, organize completed work into stakeholder-oriented scenarios, and produce a walkthrough whose claims are tied to observable evidence. It must support appropriate evidence for each product surface rather than assuming screenshots are always available.

### Impact
- Demo scripts can be generated for epic and feature scopes across supported project types, including non-UI products.
- Every normal-mode scenario has a traceable source requirement and verified observable evidence; incomplete work is identified as not demonstrated.
- Demo preparation remains separate from UAT and does not add a mandatory default workflow status.

---

## User Stories

### Must-Have Stories

**Story 1**: As a project presenter, I want an accurate demo script for a completed epic or feature so that I can show stakeholder value without overstating incomplete or unverified work.

**Acceptance Criteria**:
- [ ] The action accepts only epic or feature keys and reads their relevant scope, statuses, acceptance criteria, notes, related documents, and completed child work.
- [ ] The resulting script groups epic work into user journeys rather than a raw feature inventory, and feature work into its outcomes and relevant integrations.
- [ ] Incomplete work is explicitly placed under “Not demonstrated” rather than presented as complete.

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
   - **Description**: For each scenario, record stakeholder value, source acceptance criteria, prerequisites/demo data, presenter actions, expected observable result, evidence type/path, reset or recovery instructions, and known limitations.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Every product claim traces to a Shark entity or a project product document.
     - [ ] Normal mode verifies that referenced evidence exists and identifies its environment/date.
     - [ ] `--draft` may produce a script with uncaptured evidence, clearly marking those gaps.

3. **REQ-F-003**: Persist demo artifacts and references
   - **Description**: Write the generated artifact at `docs/demos/<entity-key>/demo-script.md`, with evidence kept under its `evidence/` directory, then attach the script through Shark’s related-document and reference-note contracts.
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] The created demo script can be discovered from the relevant epic or feature.
     - [ ] Any defects found while preparing the demo are returned as triage candidates rather than created automatically.

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

---

*Last Updated*: 2026-07-21
