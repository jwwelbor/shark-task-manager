# E34 Scope Boundaries

**Epic**: [Prompt and Skill Improvements](./epic.md)

## In scope

1. Layered, reusable prompt and skill content for interaction tracking,
   demonstration, Questions, defect classes, state planning, decisions, tiers,
   and quality gates.
2. A bounded GateResult contract and parent-owned persistence in Shark Rider
   and the Go core runner.
3. Canonical route-based workflow changes required for a final epic integration
   review.
4. Content manifest/index updates, prompt rendering, workflow validation,
   parity tests, and full repository quality gates.
5. Read-only override drift classification, explicit canonical baseline
   provenance, upgrade summary integration, and acknowledgement metadata.
6. A planned cross-repository WWGM adoption item that promotes reusable
   behavior, retains local policy, removes stale overrides, adds local
   safeguards, and reuses existing change records.
7. Later E40 benchmark scenarios that do not block E34 delivery.

## Out of scope

### Application-specific validation in Shark defaults

WWGM method-length scripts, test-selection scripts, Python environment setup,
test database derivation, skip policy, coding standards, lint configuration,
model selection, and workflow order remain in WWGM. Shark defines the generic
evidence and guard contract only.

### Automatic override modification

Shark does not merge, patch, delete, disable, or rewrite override files. It
reports digests, baselines, and classifications so an operator can reconcile
them explicitly.

### New defect, decision, or interaction storage

This epic reuses typed notes, existing entity workflows, I/X maps, Questions,
and councils. It does not add a review-finding table, recurrence table,
decision entity, or interaction database entity.

### Runtime application state machines

Closed transition tables and state-aware tests are planning requirements. Shark
does not generate or execute a project's domain state machine.

### Retry-count escalation

No automatic architect dispatch at round three or owner hard-stop at round
five is added. Escalation follows completed-sweep evidence and existing
Question/council materiality.

### Global owner-approval policy

The final integration gate cannot silently supersede feature rejection, but
E34 does not add or change a global `require_owner_approval` setting. WWGM's
historical E04-F02 record is reconciled as project work.

### QA for every STANDARD feature

STANDARD retains a combined code-review/QA gate and no standalone QA artifact.
E34 aligns prompts with that route instead of adding a gate to satisfy stale
artifact expectations.

### Premature rules-routing infrastructure

A generic `rules.py` selector and editor hook remain deferred until more than
one project or measured context growth justifies them. WWGM receives a thin
root `AGENTS.md` and local executable guards.

### E40 as a delivery gate

E40 is underway and can later measure these workflow changes. E34 does not wait
for its corpus, harness, or baseline report.

## Alternatives considered

### Let gate workers write Shark notes directly

Rejected because it violates parent-owned lifecycle authority and weakens
session binding, ordering, and replay safety.

### Extend free-form directive lines independently in Rider and core runner

Rejected because it preserves two parsers and makes nested finding, sweep, and
evidence validation brittle.

### Escalate from review-round count

Rejected because the count does not say whether a finding is new, recurring,
already dispositioned, or outside an earlier sweep. Durable class evidence is
the general signal.

### Limit cross-entity state analysis to one foreign-key hop

Rejected because consumers also cross services, APIs, events, files, CLI
outputs, and epics. Interaction maps and production caller paths define the
real boundary.

### Automatically three-way merge overrides

Rejected because Markdown templates and route-based YAML carry semantic policy
that cannot be merged safely from text position alone.

## Future candidates

- Cross-project rules indexing after broader evidence
- Automated benchmark comparison after E40 is ready
- Aggregate analytics over structured GateResult notes after adoption data
  demonstrates a concrete reporting need
- Finer-grained overlay/patch override semantics as a separate design effort

*See also*: [Requirements catalog](./requirements.md)
