---
research_schema: 2
entity_key: E39-F04
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - api
  - data
  - documentation
related_work: true
---

# Research report: Focused Question read surfaces and consumer handoff

## Scope

E39-F04 consumes I-01 (the registered Question record), I-02 (validated
serial responder state), and I-03 (the compact direct-block handoff). It
extends the existing safe Question metadata read path with two focused reads:
open Questions by current responder and Questions directly blocking a supplied
entity. It also defines X-06's producer-side handoff for E38-F09.

F04 does not change Question workflow, claims, response recording, blocking
qualification, relationship mutation, or generic `blocks` behavior. It does
not repair E38-F09, add a host queue, copy mutable Question state into chat or
`docs/council/`, add rich viewer interaction, or add a migration.

Use **full Question read** for an assigned responder or resolution owner. Use
the **compact handoff** for linked-work callers. A **focused read** returns a
bounded, purpose-specific projection; it is not a way to expose response
summaries, evidence pointers, `question_state`, claims, or relationship IDs.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E39-question-and-decision-workflow-management/E39-F04-focused-question-read-surfaces-and-consumer-handof/feature.md`, `docs/plan/E39-question-and-decision-workflow-management/epic.md`, and `docs/plan/E39-question-and-decision-workflow-management/requirements.md` REQ-F-006 define the focused reads, full-read disclosure boundary, and no-host-queue scope.
- [x] `affected_implementation_or_contract` — Evidence: `internal/repository/question/repository.go` (`QuestionListFilter` and `List`), `internal/services/question_service.go` (`QuestionListFilter` and `ListQuestions`), `internal/cli/commands/question.go` (`questionListFlagNames` and `runQuestionList`), `internal/api/question_handler.go` (`questionListQueryKeys` and `ListQuestions`), and `internal/models/question.go` (`ProjectQuestion`) define the currently supported finite filters and metadata-only transport contract that F04 extends.
- [x] `related_work` — Evidence: `docs/plan/E39-question-and-decision-workflow-management/research-report.md`, `docs/plan/E39-question-and-decision-workflow-management/architecture.md`, `docs/plan/E39-question-and-decision-workflow-management/E39-interaction-map.md`, the F01–F03 research reports, and `docs/plan/E39-question-and-decision-workflow-management/E39-cross-epic-map.md` establish I-01 through I-03 as F04 inputs and E38-F09 as X-06's blocked activation consumer.
- [x] `pattern_contract` — Evidence: `internal/models/question.go` (`ProjectQuestion`), `internal/cli/commands/question.go`, `internal/api/question_handler.go`, and `internal/cli/commands/view.go` use a metadata-only projection that omits `ContextData`, file path, and size from user-visible transports. F04 must preserve that projection-first pattern.
- [x] `dependency_impact` — Evidence: `internal/models/question.go` (`QuestionState.CurrentResponder`) derives the serial recipient; `internal/services/question_blocker.go` (`QuestionBlocker.Check`) reads only incoming `question_blocks` and returns I-03's four compact fields. The F04 query seam must consume these sources without creating a second predicate or mutating either entity.

## Capability map

| Capability | Evidence | Decision | F04 responsibility |
| --- | --- | --- | --- |
| Registered Question retrieval and base list | F01 research report; `QuestionRepository`; `QuestionService` | REUSE | Keep existing get/list behavior compatible and build focused reads through the same repository/service boundary. |
| Safe metadata projection | `models.ProjectQuestion`; CLI/API Question transports | EXTEND | Reuse the metadata-only shape for compact focused results. Define a distinct full-read projection rather than serializing the persisted model. |
| Serial current responder | F02 research report; `QuestionState.CurrentResponder` | REUSE | Filter open Questions by the derived pending responder. Do not expose responses or change routing. |
| Direct blocking predicate and I-03 | F03 research report; `QuestionBlocker.Check` | REUSE | Use the same direct `question_blocks`, `blocking=true`, and open-state qualification for `blocking-for`; do not reproduce or broaden it. |
| Finite CLI and HTTP query contracts | `question.go`; `question_handler.go` | EXTEND | Add explicit supported routes/flags and reject unknown parameters before service invocation. Preserve existing list flags. |
| Full-read authorization/disclosure | Parent architecture §Interfaces, migration, security, and ADRs; REQ-F-006 | NEW | Add a service-owned policy seam that permits full response material only to the assigned responder or resolution owner; all other callers receive compact data. |
| E38-F09 live resume | `E39-cross-epic-map.md`; E38-F09 `feature.md` | EXTEND (producer handoff only) | Publish stable X-06 consumer documentation and testable compact/read contracts. Do not implement its provider adapters or continuation logic. |
| Host queue, viewer redesign, generic relationship traversal, or state copies | F04 `feature.md`; parent scope; F03 research report | CONTRADICTS | Keep these alternatives out of F04. |

## Findings

1. **The existing public list surface is deliberately finite but cannot answer
   the two F04 questions.** The repository and service accept only status,
   requester, blocking, limit, and offset. CLI flags and HTTP query-key
   validation mirror that same set. Add named focused contracts rather than
   accepting an unimplemented shared filter.

2. **The current safe projection already protects the base record but is not
   a full-read authorization contract.** `ProjectQuestion` excludes raw
   `ContextData`, which contains I-02 state and response history. F04 needs
   separate compact and authorized-full projection types so transports cannot
   accidentally expose raw persistence fields.

3. **Open-by-responder must use derived I-02 state.** `CurrentResponder()`
   returns the first pending responder. The query must include only Questions
   in the F02 dispatchable/open lifecycle and must not infer a recipient from
   a claim, completed responder, response text, or resolution owner.

4. **Blocking-for must consume F03's direct predicate, not create a second
   relationship interpretation.** `QuestionBlocker` limits the relation to
   incoming `question_blocks`, checks `blocking`, accepts only `open` or
   `answering`, and returns a deterministic compact record. The F04 query must
   preserve direct-only and stable-order behavior while returning only the
   permitted compact fields to a linked-work caller.

5. **X-06 is a producer contract, not an E38 implementation task.** The
   cross-epic map assigns E39-F04 to publish the generic lifecycle/read
   contract and E38-F09 to add its consumer coverage when it resumes. F04 can
   document a stable handoff and verify the provider-neutral surface without
   selecting a host adapter or resuming F09.

## Decisions

1. **Add focused read operations at the Question service boundary.** Define
   explicit, bounded inputs for current responder and blocked entity, then
   translate them into parameterized repository reads. Do not extend the base
   list filter with loose query fields.

2. **Use two projection levels.** Keep the existing metadata projection as the
   compact/default result. Add an authorized full-read projection containing
   only the response/provenance fields needed by an assigned responder or
   resolution owner, never raw `ContextData`. Keep evidence pointers and
   response summaries out of compact and blocked-work results.

3. **Centralize disclosure authorization in the service.** CLI and HTTP
   handlers pass a caller identity/policy context to the same service check.
   The service allows full detail only for the current responder or resolution
   owner and otherwise returns the compact projection or a typed access error.
   The exact authentication source is an integration seam; do not assume one
   in the repository or transport.

4. **Expose focused contracts with explicit names and preserve compatibility.**
   The specification must choose stable CLI and HTTP names for
   `open-by-responder` and `blocking-for`, document their finite parameters,
   pagination, canonical ordering, empty result, and error shapes, and prove
   existing get/list results remain unchanged.

5. **Publish X-06 as a versioned producer handoff.** Document the compact
   blocked-work result, safe full-read access rule, and no-queue/no-copy
   boundary. Add producer tests and an E38-F09 activation breadcrumb; leave
   adapter selection and consumer execution to E38-F09.

## Sources

- `docs/plan/E39-question-and-decision-workflow-management/E39-F04-focused-question-read-surfaces-and-consumer-handof/feature.md`
- Parent contracts: `epic.md`, `requirements.md`, `scope.md`,
  `research-report.md`, `architecture.md`, `E39-interaction-map.md`, and
  `E39-cross-epic-map.md`
- Upstream contracts: F01, F02, and F03 `research-report.md`; F03 `spec.md`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `internal/models/question.go`
- `internal/repository/question/repository.go`
- `internal/services/question_service.go` and `question_blocker.go`
- `internal/cli/commands/question.go` and `view.go`
- `internal/api/question_handler.go`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md`

RECOMMENDED OUTCOME: standard
