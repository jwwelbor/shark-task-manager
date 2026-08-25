---
name: demo-script
description: Prepare truthful, portable demo scenario maps from documented project evidence without granting acceptance authority.
---

# Demo Script

## Purpose and boundary

Use this craft skill to prepare a traceable demo script for an epic, feature, or
sprint. It answers how to truthfully show documented delivered behavior. It is
not a workflow status, UAT gate, acceptance decision, deployment recipe, or
backlog creator. The Rider procedure supplies documented project state, linked
guidance, and the selected artifact location; this skill must not assume a
framework, package manager, browser, deployment provider, credential,
endpoint, environment, or capture tool.

Use [the scenario template](context/demo-script-template.md) for every
scenario. Preserve the labels exactly:

- `Demonstrated now`
- `Not demonstrated / pending integration`
- `Accepted risks and overrides`

## Evidence collection

1. Read the selected entity's committed requirements, acceptance criteria,
   related documents, and documented project guidance. Use only commands,
   access, environments, and capture methods that those sources document.
   For a sprint, use the assigned backlog as the scope snapshot and trace each
   item to its documented parent requirements; do not include unassigned work.
2. Select an evidence surface that fits the capability: UI capture or
   recording, CLI transcript, API request/response plus resulting state, SDK
   runnable example, pipeline artifact or data, infrastructure health or metric
   evidence, or background trigger/log/result evidence.
3. For a `Demonstrated now` scenario, require a traceable source and existing,
   observable evidence with its environment and date. A completion marker,
   owner decision, or demo artifact is context, not proof.
4. When documented setup, capture guidance, or evidence is absent, normal mode
   reports an evidence or decomposition gap. Draft mode may retain the scenario
   only when it explicitly labels the uncaptured evidence and missing guidance.
   Do not invent commands, credentials, deployments, endpoints, or proof.
5. Remove or redact secrets and avoid hard-coded environment-specific URLs.

## Readiness classification

Consume the I-01 readiness handoff read-only from
`E34-interaction-map.md#i-01-readiness-evidence-shape`. Preserve the shared
contract-test pointer `TC-I-01-READINESS-SYMMETRY`; do not create a
consumer-only twin test.
E34-F03 is the producer; E34-F02 is the consumer, activation owner, and
closure key for the live demo-script caller chain.

Record these nine fields without changing their meaning:

| Field | Meaning in the demo script |
|---|---|
| `assessor_verdict` | Independent assessment, never rewritten by an owner decision |
| `owner_decision` | Separate approval or override decision and its conditions |
| `open_conditions` | Open activation and other conditions that remain visible |
| `gate_mode` | Readiness mode, including `contract-only` where declared |
| `activation_owner` | Owner responsible for live activation proof |
| `closure_key` | Key that closes the tracked activation obligation |
| `counterpart_status` | Current live status read at review time |
| `review_basis` | Documented material used for the review |
| `demonstrability_disposition` | Whether the claim is currently demonstrable |

Keep a `contract-only` or open activation obligation in `Not demonstrated /
pending integration` with `pending-integration` until the activation owner
supplies closed, live production-path proof. A blocking assessor verdict with
an `override-accept` owner decision remains under `Accepted risks and
overrides`, retaining the assessor verdict, owner decision, open conditions,
and risk. Do not treat an override as demonstrated delivery.

## Artifact and discrepancy boundaries

The Rider procedure writes the script below
`docs/demos/<entity-key>/demo-script.md` and keeps supporting material below
`docs/demos/<entity-key>/evidence/`. It links only a successfully created
artifact through the existing related-document and reference-note contracts.

Describe discrepancies as triage candidates only. They require normal
deduplication and user confirmation; this skill never creates backlog work or
changes acceptance state.

## Final check

Before returning a script, confirm every scenario has all template fields, each
`Demonstrated now` claim has existing environment/date-scoped evidence, draft
gaps are plainly labeled, and readiness facts remain classifications rather
than approval decisions.
