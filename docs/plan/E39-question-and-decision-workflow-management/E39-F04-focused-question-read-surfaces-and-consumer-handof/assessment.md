# E39-F04 Assessment — Focused Question Read Surfaces and Consumer Handoff

## Scope validation

**Classification: FEATURE.**

F04 delivers a stakeholder-visible read contract rather than a single
implementation step: safe full Question retrieval, focused open-by-responder
and blocking-for query paths, compact-versus-full disclosure handling, and a
stable consumer handoff for E38-F09. The current finite list surface supports
only `status`, `requester`, and `blocking`; it has no responder-state or
`question_blocks` query. Delivering the missing contract requires coordinated
repository/service projections, CLI and HTTP transports, tests, and the
consumer-facing handoff. It is neither a one-to-three-file change nor a
reclassification candidate.

Evidence:

- `internal/repository/question/repository.go` exposes only the finite base
  `QuestionListFilter` and base-record query.
- `internal/services/question_service.go` intentionally maps that same finite
  filter; it is the appropriate domain seam for focused reads.
- `internal/cli/commands/question.go` and
  `internal/api/question_handler.go` expose only the existing base list
  contract and explicitly reject unsupported filters.
- `internal/services/question_blocker.go` already supplies the deliberately
  compact I-03 projection (`QuestionKey`, `Summary`, `ResolutionOwner`, and
  `CurrentResponder`); F04 must retain that disclosure boundary instead of
  copying full Question state into a blocked-work prompt.
- `docs/plan/E39-question-and-decision-workflow-management/requirements.md`
  REQ-F-006 and `architecture.md` §Interfaces, migration, security, and ADRs
  require the two focused query surfaces and the full-record authorization
  boundary. `E39-cross-epic-map.md` identifies E38-F09 as X-06's deferred
  activation consumer.

## Complexity triage

**Score:** 15/27
**Tier:** STANDARD

### Technical complexity

1. **File impact: 3/3** — the likely implementation spans Question
   repository/query tests, service/projection tests, CLI command/tests, HTTP
   handler/route tests, viewer or consumer-projection tests, and feature
   contract/UAT evidence: more than ten changed or added files.
2. **Pattern novelty: 1/3** — it adapts established finite filter,
   service-owned query, Cobra, and HTTP-handler patterns. The disclosure rule
   is a new application of the existing compact `QuestionBlock` projection,
   not a new architecture.
3. **Data model: 0/3** — F01-F03 already persist Questions,
   `question_state`, and `question_blocks`; F04 needs indexed reads over those
   structures, not a schema or migration change.
4. **API surface: 3/3** — it adds supported focused CLI and HTTP query
   contracts rather than merely changing an internal implementation.
5. **Cross-feature dependencies: 2/3** — it consumes I-01 (registered
   Question), I-02 (validated responder state), and I-03 (direct blocking
   predicate/compact handoff). E38-F09 is a downstream consumer, not an F04
   implementation dependency.
6. **UI complexity: 1/3** — the CLI needs new compact, bounded read output;
   no rich viewer interaction or graphical UI is in scope.

### Execution complexity

7. **Task estimation: 2/3** — estimated four to six tasks: query/projection
   design, repository/service implementation, CLI transport, HTTP transport,
   disclosure and consumer-handoff tests, and focused UAT evidence.
8. **Regression risk: 2/3** — this is behavior-preserving work on established
   public read surfaces. A permissive filter or projection can leak responder
   response material or alter existing list behavior, so compatibility and
   redaction tests are required.
9. **Execution effort: 1/3** — approximately one to two weeks, driven by
   cross-layer contract tests rather than data migration or new runtime work.

**Technical total:** 10/18  
**Execution total:** 5/9  
**Overall total:** 15/27

## Tier assignment

**Assigned tier: STANDARD.** F04 uses established patterns but crosses the
repository, service, CLI, HTTP, and consumer-disclosure boundaries, and it
depends on all three completed E39 producer contracts. It is appropriately a
single feature: its query and handoff capabilities make one coherent, bounded
external-read outcome and do not warrant a further split before design.

## Autonomous-build feasibility

- Task count: estimated 4-6 (threshold <=10)
- Regression risk: 2/3 (threshold <=1)
- Execution effort: 1/3 (threshold <=1)
- Circular dependencies: no — E39-F01 through F03 are completed; E38-F09 is a
  blocked downstream activation consumer only.

**Recommendation: manual execution recommended.** The standard workflow
should first pin the focused projection fields, authorization/disclosure
rules, accepted CLI/API names and filters, and X-06 handoff shape. Do not
repair E38-F09, introduce a host queue, copy mutable state into chat or
`docs/council/`, or add a migration unless a new design decision proves one is
needed.
