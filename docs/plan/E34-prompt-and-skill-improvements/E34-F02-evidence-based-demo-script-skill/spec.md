# E34-F02 Specification: Evidence-Based Demo Script Skill

## Scope and traceability

This feature adds the portable, explicit demo-preparation capability described
in the feature brief's **Goal**, **Requirements**, and **Integration Contract**.
It advances the Epic E34 **Goal** and **Success Criteria** by making an
evidence-backed skill composable through Shark Rider; it does not restate the
epic's business context. The parent has no `architecture.md`; the applicable
system constraints are therefore the feature research report's **Capability
map** and the established Rider and embedded-bundle patterns named below.

The capability map sets these boundaries:

- Extend Rider's host-local Mode-3 verb catalog and the embedded craft-skill
  bundle.
- Reuse Shark data-plane commands for entity, related-document, and note
  discovery; reuse the UAT evidence and independent-verdict vocabulary without
  turning demo preparation into UAT.
- Consume E34-F03's I-01 readiness evidence read-only; E34-F02 is its
  activation owner and closure key.
- Do not add a `shark demo` Go command, workflow status, schema, repository,
  service, entity type, environment provisioner, or automatic triage creator.

## Requirements

### Functional requirements

#### REQ-F-001 — Expose an explicit portable Rider demo action

Extend the Rider router, static help, and capability reference with
`/shark-rider demo <epic-key|feature-key|sprint-key> [--draft]`. Its
host-local procedure shall validate the target as an epic, feature, or
sprint, collect only documented Shark state and linked project guidance,
retrieve `demo-script` with `shark skill get`, and coordinate artifact
generation. It shall not add a CLI command or advance an entity workflow.

> **Scope note:** Extended 2026-08-31 to include sprint targets, tracking the
> sprint-demo capability shipped in PR #186 (2026-08-17) for E19 sprint work —
> see the shark decision note on E34-F02 for the decision record.

**Traceability:** Feature REQ-F-001; Epic E34 Goal and Success Criteria.

#### REQ-F-002 — Generate traceable, surface-neutral scenario maps

The `demo-script` skill and template shall require each scenario to record:
stakeholder value; source requirement or acceptance criterion; prerequisites
and demo data; presenter actions; expected observable result; evidence type and
path; evidence environment and date; acceptance/readiness classification;
reset or recovery instructions; and known limitations.

The evidence model shall support UI captures or recordings, CLI transcripts,
API request/response plus resulting state, SDK runnable examples, pipeline
artifacts or data, infrastructure health or metric evidence, and background
trigger/log/result evidence. The skill may use only project-documented
commands, access, environments, and capture methods.

**Traceability:** Feature REQ-F-002 and REQ-NF-001/002; Epic E34 Goal.

#### REQ-F-003 — Classify claims without granting acceptance authority

Normal mode shall verify the source and evidence path before placing a claim in
`Demonstrated now`; `--draft` may retain an uncaptured-evidence scenario only
when it labels the gap. Generated scripts shall separate `Demonstrated now`,
`Not demonstrated / pending integration`, and `Accepted risks and overrides`.
Completion markers, a demo artifact, or an owner decision shall be context, not
proof.

The recipe shall read the latest independent assessor verdict separately from
the owner decision and open conditions. A blocking verdict overridden with
`override-accept` remains visible under accepted risks; a `contract-only` or
open activation obligation remains pending integration until the activation
owner supplies closed live production-path proof. An entity with no complete,
observable scenario shall return an evidence/decomposition gap and a
`/shark-rider triage` candidate, not fabricate a walkthrough or create work.

**Traceability:** Feature REQ-F-002 and REQ-F-004; Epic E34 Success Criteria.

#### REQ-F-004 — Persist and register the generated demo artifact

The procedure shall write `docs/demos/<entity-key>/demo-script.md` and retain
supporting evidence below `docs/demos/<entity-key>/evidence/`. After creation,
it shall use the existing `shark related-docs add` contract to attach the
script to the selected epic or feature, and the existing `shark create note
--type=reference` contract to record the artifact reference; for a sprint
target, which has no related-document parent option, the reference note alone
records the artifact. Discovered discrepancies remain explicit triage
candidates requiring normal deduplication and user confirmation.

**Traceability:** Feature REQ-F-003; research Capability map, related-document
discovery and reference-note linkage.

### Non-functional requirements

- **REQ-NF-001 — Stack and environment neutrality:** The verb, skill, and
  template must not assume a framework, package manager, browser, deployment
  provider, credential, endpoint, or capture tool. Missing documented guidance
  is an evidence gap in normal mode and an explicitly marked gap in draft mode.
- **REQ-NF-002 — Truthful and safe output:** Every demonstrated expected result
  is observable, every normal-mode claim has a committed-scope source and
  existing dated/environment-scoped evidence, and generated scripts omit
  secrets and hardcoded environment-specific endpoints.
- **REQ-NF-003 — No workflow expansion:** The capability is an explicit Rider
  action and embedded content only; it introduces no default workflow transition
  and never repeats, replaces, or approves UAT.
- **REQ-NF-004 — Bundle compatibility:** The skill has a normalized manifest
  identity and remains retrievable by `shark skill get`; affected prompt/help
  content renders with the existing bundle validation and golden-render paths.

### Acceptance criteria

- **AC-001:** `/shark-rider demo` recognizes epic, feature, and sprint
  targets, supports `--draft`, retrieves `demo-script` through `shark skill
  get`, and appears in Rider routing and static help without a `shark demo`
  command or workflow transition.
- **AC-002:** A normal-mode script has a traceable, observable, existing
  evidence item with environment/date for every `Demonstrated now` scenario;
  each scenario contains all REQ-F-002 fields.
- **AC-003:** The template and focused tests cover UI, CLI, API, SDK, pipeline,
  infrastructure, and background-process evidence without assuming a
  stack-specific toolchain.
- **AC-004:** When setup or capture guidance is absent, normal mode reports an
  evidence gap and draft mode labels the uncaptured step; neither invents a
  command, credential, deployment, or endpoint.
- **AC-005:** Contract-only/open activation claims appear only under `Not
  demonstrated / pending integration`; an overridden blocking verdict retains
  the assessor verdict, owner decision, conditions, and risk under `Accepted
  risks and overrides`.
- **AC-006:** The script is written below `docs/demos/<entity-key>/`, linked as
  a related document, and recorded with a reference note; discrepancies are
  offered as triage candidates only.
- **AC-007:** `shark admin validate-data`, focused Rider/bundle tests, rendered
  prompt goldens where changed, and the repository quality gate pass.

### Out of scope

- A mandatory demo status, a default workflow gate, UAT approval, or a rewrite
  of assessor verdicts and owner decisions.
- Environment provisioning, capture tooling installation, deployment, account
  creation, credential discovery, or fabricated proof.
- A Go CLI command, API, database schema/migration, service, repository,
  entity type, or persistent readiness ledger.
- Automatic creation of bugs, tasks, or any other backlog item from a demo
  discrepancy.
- Redefining E34-F03's staged-readiness semantics or adding another I-## test.

## Architecture

### Component changes

This is a content-and-host-procedure feature. It follows the Mode-3 boundary
in `skills/shark-rider/SKILL.md`, the concise local-verb convention in
`skills/shark-rider/verbs/{triage,revalidate}.md`, and the embedded-bundle
layout in `internal/sharkdata/default_data/README.md`.

| Surface | Files to modify or create | Change |
|---|---|---|
| Rider routing and capability discovery | `skills/shark-rider/SKILL.md`, `skills/shark-rider/verbs/demo.md`, `skills/shark-rider/verbs/help.md` | Add `demo` to recognized verbs, document the explicit Mode-3 procedure and safe target/flag handling, and expose static help. |
| Portable craft skill | `internal/sharkdata/default_data/skills/demo-script/SKILL.md`, `internal/sharkdata/default_data/skills/demo-script/context/demo-script-template.md` | Define the evidence collection, readiness classification, output validation, and reusable scenario template. |
| Embedded bundle registration | `internal/sharkdata/default_data/manifest.yaml`, `internal/sharkdata/default_data/skills/README.md` | Register normalized identity `demo-script` as canonical and index its contributor-facing purpose. |
| Procedure and bundle regression tests | `internal/cli/commands/interaction_prompts_test.go`, `internal/sharkdata/embed_test.go`, and changed files under `internal/cli/commands/testdata/rendered-prompts/` if any Rider content becomes rendered prompt content | Verify the documented F02 assets/references, bundle-manifest identity, and existing rendering contract. Add focused tests rather than a new runtime test harness. |
| Generated user artifact | `docs/demos/<entity-key>/demo-script.md`, `docs/demos/<entity-key>/evidence/` | Runtime output location created by the recipe; not a shipped repository fixture. |

### Data model and persistence

No Shark data model, schema, migration, API, repository, service, or Go command
changes are required. The generated markdown artifact and its evidence directory
are filesystem output. Discoverability reuses the existing related-document and
entity-note records; the feature creates no new persistence type.

The I-01 readiness shape remains design-time/documented input gathered from
Shark state and feature artifacts. It is not copied into a new table or used to
derive workflow status.

### API and interface contracts

The host-facing interface is:

```text
/shark-rider demo <epic-key|feature-key|sprint-key> [--draft]
```

`demo.md` must reject every other entity type. In normal mode it must stop
before declaring a claim demonstrated if required evidence or documented
guidance is absent. In draft mode it may write a marked incomplete scenario.
The procedure uses existing data-plane interfaces only:

```text
shark get <key> --json
shark list <epic> [feature] --json
shark related-docs list --epic=<epic-key> --json
shark related-docs list --feature=<feature-key> --json
shark sprint get <sprint-key> --json
shark sprint backlog <sprint-key> --all --json
shark skill get demo-script
shark related-docs add "Demo Script" docs/demos/<entity-key>/demo-script.md --epic=<epic-key>
shark related-docs add "Demo Script" docs/demos/<entity-key>/demo-script.md --feature=<feature-key>
shark create note <key> "Demo script: docs/demos/<entity-key>/demo-script.md" --type=reference
```

The final two operations occur only after a script is successfully created;
they are artifact-linking operations, not lifecycle transitions. The procedure
does not call claim, status-transition, approval, provisioning, or automatic
triage commands.

### I-01 readiness input contract

The skill consumes, but does not produce, **I-01** from E34-F03. The exact
shape source is
[`E34-interaction-map.md#i-01-readiness-evidence-shape`](../E34-interaction-map.md#i-01-readiness-evidence-shape),
and the shared structural contract-test pointer is
**TC-I-01-READINESS-SYMMETRY** at
`internal/cli/commands/interaction_prompts_test.go::TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY`.

The procedure reads these nine values without transforming their meaning:
`assessor_verdict`, `owner_decision`, `open_conditions`, `gate_mode`,
`activation_owner`, `closure_key`, `counterpart_status`, `review_basis`, and
`demonstrability_disposition`. It reads current counterpart status live from
Shark at execution/review time. A `contract-only` I-01 stays
`pending-integration` until E34-F02, the named activation owner and closure
key, proves live production-path use and closes the tracked obligation.

### Key technical decisions

1. **Use a host-local Mode-3 verb plus an embedded skill.** This extends the
   capability map's established Rider routing and reusable craft-skill delivery
   surfaces. It keeps the CLI as the data plane and avoids a new runtime
   feature.
2. **Select evidence by documented product surface.** The portable template
   supports the required evidence classes while refusing undocumented setup;
   this satisfies the feature's neutrality and truthful-output boundaries.
3. **Keep readiness as classification, never acceptance.** This reuses the
   UAT evidence boundary and consumes I-01 read-only, preserving an independent
   assessor verdict and a separate owner decision.
4. **Use artifact links and reference notes for discoverability.** This reuses
   the existing related-document and note contracts, rather than introducing a
   demo entity or persistence surface.
5. **Test content contracts at their existing seams.** Bundle validation and
   focused file/render checks prove registration and resolution; scenario
   examples and review prove truthfulness rather than simulating a policy
   engine.

### Integration with existing code

- `skills/shark-rider/SKILL.md` dispatches `demo` to
  `skills/shark-rider/verbs/demo.md`; the procedure retrieves the reusable
  content through `shark skill get demo-script`, consistent with Rider's
  content-bundle retrieval rule.
- `internal/sharkdata/default_data/manifest.yaml` must name the same
  `demo-script` identity used by the directory and `SKILL.md` frontmatter, so
  the `internal/sharkdata` validator accepts the embedded bundle.
- Existing `shark related-docs add` and `shark create note --type=reference`
  operations provide discovery links after successful artifact creation; no
  direct database access or new function signature is introduced.
- `internal/cli/commands/interaction_prompts_test.go` and
  `internal/sharkdata/embed_test.go` are the existing focused test seams for
  E34 content references and bundle integrity. The new checks must preserve
  their renderer/validator patterns and avoid a database-backed test.
- The demonstration skill's I-01 consumer behavior must mirror the map and
  executable `TC-I-01-READINESS-SYMMETRY`; no twin interaction contract test is
  added.

## Cross-feature interactions

**Consumes: I-01 — readiness evidence shape.** Producer: E34-F03 Deliverable
Feature Decomposition and Staged Integration Acceptance. Consumer: E34-F02
Evidence-Based Demo Script Skill. The exact shape source is
[`E34-interaction-map.md#i-01-readiness-evidence-shape`](../E34-interaction-map.md#i-01-readiness-evidence-shape).
The exact shared structural contract-test pointer is
**TC-I-01-READINESS-SYMMETRY** at
`internal/cli/commands/interaction_prompts_test.go::TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY`.
E34-F02 consumes the nine fields read-only to classify demo claims; it is the
map-declared activation owner and closure key. It creates no second I-## ID or
consumer-only twin test.

## Cross-epic integrations

No E34-specific cross-epic map exists, and
`docs/product/cross-epic-integration-map.md` has no X-## row naming E34-F02.
This feature therefore declares no X-## identifier. A newly discovered
cross-epic handoff must first be assigned by the product map (and an E34 map if
added) with ownership, exact contract source, UX/CX handoff notes, and test or
progress-log deferral before this specification may mirror it.

## Delivery and verification plan

1. Add the Rider verb, router/static-help references, embedded `demo-script`
   skill, template, manifest entry, and skill index together.
2. Add focused bundle/reference tests and scenario-template examples covering
   each allowed evidence surface and normal-versus-draft gap handling.
3. Verify `shark admin validate-data`, focused tests, affected prompt golden
   checks where applicable, then `make fmt`, `make lint`, and `make test`.
4. Manually review a generated normal-mode and draft script against the source
   requirement/evidence and I-01 classification boundary; confirm no secret,
   endpoint, UAT verdict, or pending integration is overstated.

## Exit-gate check

- Every requirement and acceptance criterion has a concrete content/procedure
  surface and verifiable outcome.
- All implementation and generated-artifact paths are listed in **Component
  changes**; no critical section has a TBD.
- The architecture extends documented Rider, bundle, related-document, note,
  and test patterns and explicitly excludes runtime expansion.
- I-01 is the only touched cross-feature interaction and mirrors the parent map
  exactly in identity, shape source, shared test pointer, activation owner, and
  closure key.
- No X-## row applies, so none is invented.
