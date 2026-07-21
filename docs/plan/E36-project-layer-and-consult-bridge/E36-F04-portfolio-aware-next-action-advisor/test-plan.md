# Test plan: E36-F04 - Portfolio-aware next-action advisor

**Created:** 2026-07-20
**Revalidated:** 2026-07-20 after architect refinement
**Feature PRD:** `docs/plan/E36-project-layer-and-consult-bridge/E36-F04-portfolio-aware-next-action-advisor/feature.md`
**Technical specification:** `docs/plan/E36-project-layer-and-consult-bridge/E36-F04-portfolio-aware-next-action-advisor/spec.md`
**Parent epic:** `docs/plan/E36-project-layer-and-consult-bridge/epic.md`
**Parent UAT plan:** Not present
**Status:** APPROVED

## Scope and quality strategy

This plan verifies the two production modes of `shark next`, the portfolio
read model, configured-workflow classification, read-only claim inspection,
deterministic graph analysis, partial-evidence behavior, Rider help guidance,
JSON sanitization, and the bounded SQLite performance contract. It treats
`feature.md` as the feature intent and `spec.md` as the proposed production
contract.

The test boundary follows the repository's testing architecture:

- CLI tests call the real Cobra handler or argument/flag validator and mock the
  advisor or existing keyed adapters. They never use a database.
- Service tests call `PortfolioAdviceService.Advise` and use function-field
  repository mocks. They do not mock eligibility, graph, warning, prompt, or
  sanitization logic.
- Claim-service tests call `ClaimService.ListActiveReadOnly` with a fixed
  evaluation time and mock only `claim.Repository`.
- Repository tests use the real test database, clean before and after each
  fixture, and verify the two new set-oriented queries.
- The bounded performance integration lives with the repository tests because
  repository tests are the only tests permitted to use a real database.
- Rider and documentation contract tests read the production Markdown files,
  not copied fixtures.

The plan does not authorize a recommendation score, dispatch simulation,
schema change, HTTP endpoint, document scan from Go, or mutation from bare
advice mode.

## Spec drift resolution

### Resolved findings

| ID | Status | Original finding | Resolution evidence |
|---|---|---|---|
| DRIFT-001 | **RESOLVED** | The parent epic declared only F01-F03 and applied “exactly one Go change” to all of E36. | `epic.md` now registers F04 as a later additive fourth slice while preserving the original plan's authority for F01-F03. `requirements.md` adds REQ-F15 through REQ-F19 and scopes REQ-NF2 to the original F01-F03 Go change. `scope.md` makes the same distinction and preserves the single-project, no-schema, advisory-document boundaries. |
| DRIFT-002 | **RESOLVED** | `spec.md` treated nonexistent `internal/services/epic_analytics_service_test.go` as a file to modify. | Live-tree verification confirmed that path is absent and `internal/services/epic_service_test.go` exists. The spec now creates the focused `epic_analytics_service_test.go` file and explicitly keeps the existing higher-level progress regressions unchanged. |
| DRIFT-003 | **RESOLVED** | AC-019 used an empty claim collection to assert claim-object sanitization. | AC-019 now requires two service-produced partitions: an empty envelope for non-null array serialization and a populated-claim envelope whose source rows contain secret session/note fields for non-vacuous sanitization assertions. TC-019 matches both partitions. |
| DRIFT-004 | **INCORPORATED** | The operations section required portfolio span counts but did not name stable attribute keys. | **Observability design** defines the exact attribute keys. The tracing implementation task must copy that contract unchanged. |

All source-document blockers are resolved. The agreed command boundary remains
unchanged: bare `shark next` is read-only portfolio advice, keyed
`shark next <key>` retains dispatch and normalization, and `--preview` does not
exist on `shark next`.

No scope creep was found between `feature.md` and the behavioral design in
`spec.md`. The feature PRD calls its JSON example conceptual and delegates exact
fields to the specification, so `ordering.layers` in that example does not
conflict with the exact `dependency_layers` and `roadmap_layers` fields.

### Traceability matrix

| Feature requirement or success condition | Specification requirement(s) | Acceptance criteria | Planned evidence | Coverage |
|---|---|---|---|---|
| Bare `shark next` returns one read-only portfolio envelope | REQ-F-001, REQ-NF-001 | AC-001, AC-004 | TC-001, TC-004 | Yes |
| Keyed dispatch remains unchanged | REQ-F-001, REQ-NF-007 | AC-002, AC-003 | TC-002, TC-003 | Yes |
| Return all non-terminal epic evidence using configured workflow status | REQ-F-002 | AC-005, AC-015, AC-019 | TC-005, TC-015, TC-019 | Yes |
| Preserve relevant epic relationships and normalized hard precedence | REQ-F-003 | AC-006, AC-007 | TC-006, TC-007 | Yes |
| Produce deterministic layers without inventing total order | REQ-F-004, REQ-NF-002 | AC-008-AC-012 | TC-008-TC-012 | Yes |
| Distinguish child blockers from root eligibility | REQ-F-002, REQ-F-004 | AC-013 | TC-013 | Yes |
| Degrade with typed, actionable evidence warnings | REQ-F-006 | AC-014, AC-015 | TC-014, TC-015 | Yes |
| Give the receiving agent the authority and recommendation contract | REQ-F-005 | AC-016 | TC-016 | Yes |
| State-aware Rider help consumes advice; static help stays zero-state | REQ-F-007 | AC-017 | TC-017 | Yes |
| Remove only `shark next --preview` | REQ-F-009 | AC-018 | TC-018 | Yes |
| Encode arrays, sanitize claims, and preserve stable order | REQ-F-008, REQ-NF-005 | AC-019 | TC-019 | Yes |
| Use four set-oriented reads and meet the local threshold | REQ-NF-003, REQ-NF-004 | AC-020 | TC-020 | Yes |
| Add no schema, network, HTTP, or product-document read | REQ-NF-006 | Cross-cutting architecture gate | TC-001, TC-020 plus static diff review | Yes |
| Preserve CLI-service-repository layering and quality gates | REQ-NF-008 | Cross-cutting architecture gate | Test infrastructure plus `make fmt`, `make lint`, `make test` | Yes |

### Design-element coverage

Task specifications do not exist yet, so task ownership cannot be validated in
this feature-level pass. Task review must map every row below to one implementing
task before development begins.

| Design element | AC coverage | Required test location | Task coverage |
|---|---|---|---|
| Portfolio DTOs and exact JSON fields | AC-001, AC-019 | `internal/models/portfolio_advice_test.go` or service contract test | Pending task generation |
| Set-oriented child and relationship queries | AC-006, AC-007, AC-020 | `internal/repository/portfolio/repository_test.go` | Pending task generation |
| Advice orchestration and partial evidence | AC-004-AC-016 | `internal/services/portfolio_advice_service_test.go` | Pending task generation |
| Pure graph normalization and layering | AC-006-AC-012 | `internal/services/portfolio_graph_test.go` | Pending task generation |
| Read-only active claims | AC-004, AC-019 | `internal/services/claim_service_test.go` | Pending task generation |
| Shared epic progress formula | AC-001, AC-013 | New `internal/services/epic_analytics_service_test.go`, existing `internal/services/epic_service_test.go` regressions, and advice service test | Pending task generation |
| Zero/one/many CLI routing and preview removal | AC-001-AC-003, AC-018 | `internal/cli/commands/next_portfolio_test.go` | Pending task generation |
| Rider help and mode documentation | AC-017 | `tests/contracts/e36_f04_rider_help_test.go` | Pending task generation |
| Portfolio trace attributes | AC-001, AC-009-AC-014 | CLI tracing test with in-memory exporter | Pending task generation |
| No persistence or API change | REQ-NF-006 | Schema/API diff assertion in review checklist | Pending task generation |

## Acceptance-criteria review

| AC | Unambiguous | Testable | Traceable | Complete | Expected output | Review note |
|---|---|---|---|---|---|---|
| AC-001 | Yes | Yes | Yes | Yes | Yes | Structural fail-on-call seams prove keyed initialization is unreachable. |
| AC-002 | Yes | Yes | Yes | Yes | Yes | REQ-NF-007 closes “all current normalization behavior” to the existing response fields, exit behavior, cascade resolution, prompt assembly, and permitted status normalization; TC-002 enumerates that finite surface. |
| AC-003 | Yes | Yes | Yes | Yes | Yes | Cobra validation is the production seam. |
| AC-004 | Yes | Yes | Yes | Yes | Yes | Fixed evaluation time closes TTL boundary ambiguity. |
| AC-005 | Yes | Yes | Yes | Yes | Yes | Include a literal `completed` status configured as non-terminal to catch hard-coding. |
| AC-006 | Yes | Yes | Yes | Yes | Yes | Relationship output and eligibility are both asserted. |
| AC-007 | Yes | Yes | Yes | Yes | Yes | Stored and normalized directions are asserted separately. |
| AC-008 | Yes | Yes | Yes | Yes | Yes | Repeat and shuffled-input comparisons prove determinism. |
| AC-009 | Yes | Yes | Yes | Yes | Yes | Exact warning code, keys, unlayered list, and termination are asserted. |
| AC-010 | Yes | Yes | Yes | Yes | Yes | Hard graph remains independently usable. |
| AC-011 | Yes | Yes | Yes | Yes | Yes | Contributing relationship types are finite and asserted. |
| AC-012 | Yes | Yes | Yes | Yes | Yes | Reachability, warning, and lexical layer order are asserted. |
| AC-013 | Yes | Yes | Yes | Yes | Yes | All-direct-features-blocked is an explicit boundary case. |
| AC-014 | Yes | Yes | Yes | Yes | Yes | Decision table separates child, relationship, claim, fatal epic, cancellation, dangling, and unknown-status failures. |
| AC-015 | Yes | Yes | Yes | Yes | Yes | Every collection and prompt outcome is exact. |
| AC-016 | Yes | Yes | Yes | Yes | Yes | Required prompt clauses and prohibited actions are enumerated. |
| AC-017 | Yes | Yes | Yes | Yes | Yes | Four help partitions are explicit. |
| AC-018 | Yes | Yes | Yes | Yes | Yes | Lifecycle preview preservation is a separate negative assertion. |
| AC-019 | Yes | Yes | Yes | Yes | Yes | The required empty and populated partitions prove array serialization and claim sanitization independently. |
| AC-020 | Yes | Yes | Yes | Yes | Yes | Exact fixture sizes, four read counts, and 1-second threshold are specified. |

## ISTQB technique application

| AC | Technique(s) | Test case | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning + contract-surface enumeration | TC-001 | Zero arguments selects a distinct public response and call graph. |
| AC-002 | Regression comparison + contract-surface enumeration | TC-002 | Existing keyed wire and mutation contracts form a finite surface. |
| AC-003 | Boundary value analysis | TC-003 | Positional argument count has boundaries 0, 1, and 2. |
| AC-004 | Boundary value analysis + attack-class enumeration | TC-004 | Claim expiry has `TTL-1`, `TTL`, `TTL+1`; forbidden writes are enumerated. |
| AC-005 | Equivalence partitioning | TC-005 | Configured terminal, configured non-terminal, and unknown status are distinct classes. |
| AC-006 | State transition + decision table | TC-006 | Hard dependency satisfaction changes when the prerequisite becomes terminal. |
| AC-007 | State transition + contract-surface enumeration | TC-007 | `blocks` reverses no endpoints and gates the target until terminal. |
| AC-008 | State transition + metamorphic testing | TC-008 | A follows-chain yields fixed layers independent of input order. |
| AC-009 | State transition + attack-class enumeration | TC-009 | A hard cycle must remain visible and bounded. |
| AC-010 | Decision table | TC-010 | Hard acyclic/roadmap cyclic is distinct from both graphs cyclic. |
| AC-011 | Contract-surface enumeration | TC-011 | Every contributing stored type and both normalized directions are asserted. |
| AC-012 | Reachability partitioning + stable-order comparison | TC-012 | Comparable and incomparable eligible roots produce different warnings. |
| AC-013 | Decision table + boundary value analysis | TC-013 | Zero, one, some, and all direct features blocked determine eligibility. |
| AC-014 | Decision table + fault injection | TC-014 | Each evidence source has a distinct degradation contract. |
| AC-015 | Boundary value analysis | TC-015 | Zero non-terminal epics is the lower collection boundary. |
| AC-016 | Contract-surface enumeration | TC-016 | Prompt requirements and forbidden behaviors are finite text clauses. |
| AC-017 | Decision table + contract-surface enumeration | TC-017 | State-aware, fast, commands, known verb, and unknown verb paths differ. |
| AC-018 | Equivalence partitioning + regression comparison | TC-018 | Removed next flag and retained lifecycle flag are separate command classes. |
| AC-019 | Boundary value analysis + attack-class enumeration | TC-019 | Empty/populated collections and sensitive-field leakage classes are enumerated. |
| AC-020 | Boundary value analysis + performance testing | TC-020 | The specified maximum fixture and exact query budget define the gate. |

## ISO 25010 coverage matrix

`N/A` includes a short reason. Maintainability evidence is the named isolated
test seam plus the repository quality gate.

| AC | Functional suitability | Performance efficiency | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-001 | ✅ TC-001 no keyed setup | ✅ TC-001 two-mode command | ✅ TC-001 distinct mode | ✅ TC-001 fatal error path | ✅ TC-001 no mutator | ✅ thin CLI test | N/A: existing Go CLI |
| AC-002 | ✅ TC-002 | N/A: unchanged path | ✅ TC-002 wire regression | ✅ TC-002 existing output | ✅ TC-002 normalization | ✅ TC-002 no new surface | ✅ existing regression suite | N/A: existing Go CLI |
| AC-003 | ✅ TC-003 | N/A: argument check | ✅ TC-003 Cobra behavior | ✅ TC-003 usage error | ✅ TC-003 no call | ✅ TC-003 no state access | ✅ validator test | N/A: Cobra |
| AC-004 | ✅ TC-004 | N/A: one list/filter | ✅ TC-004 existing lease rows | N/A: internal evidence | ✅ TC-004 TTL boundary | ✅ TC-004 no notes/session output | ✅ claim-service isolation | N/A: Go service |
| AC-005 | ✅ TC-005 | N/A: classifier lookup | ✅ TC-005 custom workflow | N/A: internal evidence | ✅ TC-005 unknown status | N/A: local status data | ✅ config-driven test | ✅ workflow-config variants |
| AC-006 | ✅ TC-006 | ✅ linear graph assertion | ✅ terminal endpoint row | N/A: machine envelope | ✅ TC-006 satisfaction transition | N/A: local links | ✅ normalized-edge test | N/A: Go service |
| AC-007 | ✅ TC-007 | ✅ linear graph assertion | ✅ stored `blocks` | N/A: machine envelope | ✅ TC-007 satisfaction transition | N/A: local links | ✅ normalized-edge test | N/A: Go service |
| AC-008 | ✅ TC-008 | ✅ repeated bounded call | ✅ stored `follows` | N/A: machine envelope | ✅ TC-008 deterministic | N/A: local links | ✅ pure graph test | N/A: Go service |
| AC-009 | ✅ TC-009 | ✅ no-hang timeout | ✅ legacy cycle | ✅ actionable warning | ✅ TC-009 partial layers | N/A: local links | ✅ cycle regression | N/A: Go service |
| AC-010 | ✅ TC-010 | ✅ no-hang timeout | ✅ legacy follows cycle | ✅ actionable warning | ✅ hard graph retained | N/A: local links | ✅ graph separation | N/A: Go service |
| AC-011 | ✅ TC-011 | N/A: two-edge case | ✅ legacy contradiction | ✅ actionable warning | ✅ evidence retained | N/A: local links | ✅ diagnostic test | N/A: Go service |
| AC-012 | ✅ TC-012 | N/A: reachability check | ✅ unordered portfolio | ✅ explicit ambiguity | ✅ no invented order | N/A: local links | ✅ stable layer test | N/A: Go service |
| AC-013 | ✅ TC-013 | N/A: set aggregation | ✅ child workflow states | ✅ blocker evidence | ✅ mixed-child behavior | N/A: local state | ✅ decision-table test | N/A: Go service |
| AC-014 | ✅ TC-014 | N/A: failure paths | ✅ partial response | ✅ actionable warnings | ✅ TC-014 degradation | ✅ sanitized errors | ✅ source-specific mocks | N/A: Go service |
| AC-015 | ✅ TC-015 | N/A: empty fast path | ✅ empty database | ✅ clear no-root prompt | ✅ allocated empties | ✅ no leaked state | ✅ empty fixture | N/A: Go service |
| AC-016 | ✅ TC-016 | N/A: static prompt | ✅ Rider handoff | ✅ explicit advice | ✅ refuses guessing | ✅ no claim/dispatch | ✅ text contract | N/A: Markdown/Go string |
| AC-017 | ✅ TC-017 | ✅ static paths zero-state | ✅ existing help modes | ✅ clear mode help | ✅ bounded gap result | ✅ no static state read | ✅ contract test | ✅ Markdown procedure |
| AC-018 | ✅ TC-018 | N/A: flag parsing | ✅ lifecycle preview retained | ✅ unknown-flag error | ✅ no execution | ✅ no state access | ✅ flag regression | N/A: Cobra |
| AC-019 | ✅ TC-019 | N/A: marshal contract | ✅ stable JSON shape | ✅ machine-readable arrays | ✅ non-null empties | ✅ populated claim sanitization | ✅ DTO test | ✅ JSON |
| AC-020 | ✅ TC-020 | ✅ TC-020 <1 s, four reads | ✅ local SQLite | N/A: benchmark | ✅ deterministic fixture | ✅ no writes/network | ✅ query-count test | ✅ local SQLite builds |

### Coverage gaps

| Gap | Status | Resolution |
|---|---|---|
| Parent UAT scenarios | Deferred, non-blocking for test design | E36 has no `uat-plan.md`. The integration scenarios below trace directly to feature success conditions and must be incorporated if an epic UAT plan is later created. |
| Task-to-design coverage | Deferred until task generation | No task specs exist. First task review must map all design-element rows to tasks. |
| Codex red-team command | Unavailable, non-blocking | No concrete `codex_command` was supplied. See **Codex test-plan red-team**. |

## Observability design

The implementation contract uses the existing `shark.next` trace span. It must
not add prompt text, claim holder, entity claim keys, session IDs, notes, SQL,
or document contents to telemetry.

| Behavior | Metric | Log | Trace evidence | Alert threshold | Test evidence |
|---|---|---|---|---|---|
| Bare advice succeeds | None required | None required | `shark.next` has `mode="portfolio_advice"`, `portfolio.candidate_count=<int>`, `portfolio.relationship_count=<int>`, `portfolio.graph_warning_count=<int>`, `portfolio.evidence_complete=<bool>` | N/A: local CLI, no SLO | TC-001 with in-memory span exporter |
| Keyed dispatch succeeds | Existing telemetry | Existing telemetry | Preserve existing `entity_key`, `entity_type`, `status`, `action`, and prompt-size attributes; this feature adds no keyed-mode telemetry | Existing operations contract | TC-002 existing tracing regressions |
| Candidate and relationship evidence is assembled | None required | None required | The bare-advice span records candidate and relationship counts | N/A | TC-005-TC-008 compare the counts with the returned envelope |
| Graph defect is returned | None required | None required | `portfolio.graph_warning_count` equals the combined graph-warning count; no keys or relationship payloads are recorded | N/A | TC-009-TC-012 |
| Evidence is incomplete | None required | No raw dependency error | `portfolio.evidence_complete=false`; candidate and available-relationship counts reflect the returned envelope | N/A | TC-014 |
| Active claims are filtered, attached, and sanitized | Internal — no separate metric because claim identity is sensitive and the envelope is the direct result | None; do not log claim identity | The enclosing bare-advice span records only evidence completeness and aggregate counts; it records no holder, entity key, session, note, or heartbeat | N/A | TC-004 and TC-019 assert the service result and telemetry exclusions |
| Eligibility and progress are calculated | Internal — no separate metric because the values are returned per epic | None required | The enclosing span's candidate count proves the advice path completed; eligibility and progress remain envelope evidence | N/A | TC-005, TC-006, and TC-013 assert exact values through `Advise` |
| Advisor prompt is generated | Internal — no metric; prompt is deterministic contract text | None; do not log prompt text | No prompt content or byte sample is added by the portfolio branch | N/A | TC-016 asserts the returned prompt and telemetry exclusion |
| Empty collections and safe JSON fields are marshaled | Internal — no runtime metric; serialized output is the production evidence | None required | The enclosing span records only bounded counts and completeness | N/A | TC-015 and TC-019 decode the production JSON |
| Rider help consumes advice or stays static | Internal Markdown procedure — no new runtime instrumentation | None required | N/A: no new Rider runtime is introduced | N/A | TC-017 reads the production Rider artifacts and checks every mode |
| No candidates exist | None required | None required | `portfolio.candidate_count=0`, `portfolio.evidence_complete=true` when all reads succeeded | N/A | TC-015 |
| Performance fixture | Test-only timing | None required | Existing `shark.next` duration plus the count attributes | Test fails at 1 second | TC-020 |

No per-epic metric or log is permitted. The envelope is the operator-visible
evidence; the span records only bounded counts and completeness.

## Integration scenarios and UAT contributions

| Scenario | Production boundaries | Feature success condition | Verification |
|---|---|---|---|
| Advice-to-dispatch handoff | `shark next` → advice envelope/prompt → explicit `/shark-rider run <key>` → `shark next <key>` | Advice chooses context; execution remains explicit | TC-001, TC-002, TC-016, TC-017 prove no implicit claim, dispatch, or transition. |
| Workflow-configured portfolio | Epic repository → workflow classifiers → advice service | Only non-terminal epics appear; blocked and unknown states are explained | TC-005, TC-013, TC-014. |
| Relationship graph | Relationship repository → normalization → eligibility/layers/warnings | Every relevant stored relationship appears without invented total order | TC-006-TC-012. |
| Claim continuity without cleanup | Claim repository → `ListActiveReadOnly` → ancestor attachment → sanitized DTO | Live work informs advice while expired rows remain persisted | TC-004, TC-019. |
| Partial read | Epic list succeeds; one optional source fails | Available evidence is returned and the prompt refuses guessing | TC-014. |
| Empty portfolio | Configured terminal classification removes all epics | No recommendation can be made; arrays remain valid | TC-015. |
| Local scale gate | Real SQLite fixture → four set reads → service assembly → JSON model | 200/5,000/10,000 fixture completes within 1 second | TC-020. |
| Documentation handoff | Rider help → bare advice prompt; architecture and route guide → keyed distinction | Operators can tell advice from dispatch | TC-016-TC-018 plus document contract checks. |

E36 has no parent UAT plan. If one is added, it should reference these eight
scenarios rather than duplicate lower-level test cases.

### Cross-feature contract tests (I-##)

None. `spec.md` explicitly states that E36 has no interaction map and declares
no I-## IDs. The CLI, service, repository, and Rider-help boundaries are owned
within E36-F04, so this plan does not invent an interaction ID.

### Cross-epic integration tests (X-##)

None registered. The E36 feature spec and global
`docs/product/cross-epic-integration-map.md` contain no E36 row. E19 ordering,
E35 claims, E16 workflow classification, and E32 keyed dispatch are code-level
prior art, not authorized X-## product contracts. Coverage remains local to
TC-002, TC-004-TC-012, and TC-018 until the product map assigns an owner.

## Test infrastructure

| Area | Existing production-shaped pattern | Required addition |
|---|---|---|
| CLI command | `internal/cli/commands/next_test.go` and `next_cache_test.go` drive `resolveNext` with action/entity adapter mocks. CLI rules prohibit a real DB. | Add an advisor interface/accessor override and a fail-on-call keyed-adapter factory seam in `next_portfolio_test.go`; execute real Cobra validation and `runNext`. |
| Service orchestration | Service tests use function-field repository mocks and assert calls/arguments. | Add narrow epic, snapshot, active-claim, and workflow-provider mocks; default unimplemented functions must fail tests. |
| Workflow configuration | Existing service tests construct workflow services from test definitions rather than status-name booleans. | Add a fixture whose epic terminal status is `shipped_custom`, whose `completed` status is non-terminal, and whose blocked status is `held_custom`. |
| Claim filtering | `internal/services/claim_service_test.go` has `mockClaimRepo`, TTL fixtures, and `ReclaimExpired` call tracking. | Add `ListActiveReadOnly` table cases for TTL disabled, `TTL-1`, `TTL`, `TTL+1`, repository error, and zero reclaim calls. |
| Epic progress | Existing progress cases are in `internal/services/epic_service_test.go`; the specification now creates `internal/services/epic_analytics_service_test.go` for the extracted helper. | Preserve empty, cancelled-only, completed, archived, mixed, clamp, and rounding behavior in the focused helper tests while keeping the higher-level regressions unchanged. |
| Repository | Repository tests use `test.GetTestDB()`, parameterized inserts, and explicit cleanup. | Add helpers that seed uniquely prefixed epics/features/tasks/relationships/claims and clean in reverse FK order. |
| Graph | No dedicated portfolio graph helper exists yet. | Add table-driven pure tests and randomized input permutations; acceptance tests still enter through `Advise`. |
| Tracing | `internal/services/task_service_tracing_test.go` uses `tracetest.NewInMemoryExporter`. | Reuse the in-memory exporter to inspect `shark.next` count/completeness attributes. |
| Rider docs | `tests/contracts/e38_f07_interactions_test.go` reads production Rider files from the repository. | Add `tests/contracts/e36_f04_rider_help_test.go`; do not use copied Markdown fixtures. |
| Performance | Repository tests are the only tests allowed to use a real DB. | Put the bounded real-SQLite fixture under `internal/repository/portfolio`; separately assert service dependency call counts with mocks. |

Required commands after implementation:

```text
go test ./internal/services -run 'Portfolio|ClaimService_ListActiveReadOnly|Epic.*Progress'
go test ./internal/cli/commands -run 'Next.*Portfolio|NextCommand.*Preview'
go test ./internal/repository/portfolio -run 'Portfolio|Performance'
go test ./tests/contracts -run E36F04
make fmt
make lint
make test
```

No conditional pass is allowed. If any test or fixture is corrected during QA,
rerun the full `make fmt`, `make lint`, and `make test` sequence.

## Caller-path contracts

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `runNext(cmd, []string{})` | Injected `PortfolioAdviceService.Advise(ctx)` interface plus fail-on-call keyed factory | Cobra validator, JSON marshal, `newNextAdapterCache`, action service, transitioner | A late branch constructs mutation-capable keyed adapters before returning advice. |
| TC-002 | `runNext(cmd, []string{"E36"})` | Existing `nextAdapterCache` action/entity service mocks | `resolveNext`, prompt assembly, wire normalization | A shared advice refactor changes `NextResponse` or skips normalization. |
| TC-003 | `nextCmd.Args(nextCmd, []string{"E36", "extra"})` through Cobra execution | None; argument validator is production code | Precomputed validation error or handler mock | A permissive validator ignores the second key and dispatches E36. |
| TC-004 | `PortfolioAdviceService.Advise(ctx)` wired to real `ClaimService.ListActiveReadOnly(ctx, evaluatedAt)` | `claim.Repository` plus other read repositories | Claim service, expiry predicate, evaluation clock, any write method | Advice calls `ClaimService.List`, which reclaims the expired row. |
| TC-005 | `PortfolioAdviceService.Advise(ctx)` with real workflow classifiers | Epic/snapshot/claim read interfaces | Terminal/blocked classifier or a filtered epic list | A literal `completed` check hides a configured non-terminal epic or includes `shipped_custom`. |
| TC-006 | `PortfolioAdviceService.Advise(ctx)` | Raw epic, child-state, relationship, and claim readers | Relationship normalization, eligibility, layering | `depends_on` is normalized in the stored direction and layers E03 before E02. |
| TC-007 | `PortfolioAdviceService.Advise(ctx)` | Raw readers | Relationship normalization or eligibility | `A blocks B` is treated as `B → A`, leaving B eligible. |
| TC-008 | `PortfolioAdviceService.Advise(ctx)` repeated with permuted raw rows | Raw readers | Sort, graph helper, returned envelope | Map or repository iteration changes layer order between calls. |
| TC-009 | `PortfolioAdviceService.Advise(ctx)` under a bounded test context | Raw readers | Cycle detector, warning builder, graph result | The graph loop hangs or silently deletes one hard edge. |
| TC-010 | `PortfolioAdviceService.Advise(ctx)` | Raw readers | Separate hard and combined graph construction | A follows-cycle discards the usable hard dependency layers. |
| TC-011 | `PortfolioAdviceService.Advise(ctx)` | Raw readers | Normalization, deduplication, diagnostic types | The service keeps only one direction and omits `CONTRADICTORY_ORDER`. |
| TC-012 | `PortfolioAdviceService.Advise(ctx)` | Raw readers | Reachability, layer sorting, warning builder | The service uses priority to invent E01 before E02. |
| TC-013 | `PortfolioAdviceService.Advise(ctx)` | Raw child-state reader | Direct-feature aggregation or eligibility logic | One blocked child incorrectly makes its whole epic ineligible. |
| TC-014 | `PortfolioAdviceService.Advise(ctx)` | One failing raw reader per subcase | Error classifier, warning sanitizer, prompt builder | An optional read error aborts, leaks SQL, or leaves eligibility falsely eligible. |
| TC-015 | `PortfolioAdviceService.Advise(ctx)` | Empty epic and other raw readers | Empty-slice initialization or prompt builder | Nil slices marshal as `null` or the prompt asks for a nonexistent root. |
| TC-016 | `PortfolioAdviceService.Advise(ctx)` | Successful raw readers | Prompt builder/string contract | The prompt uses product docs as status authority or authorizes dispatch. |
| TC-017 | Production `skills/shark-rider/verbs/help.md` read by the Rider router contract test | Repository filesystem read | Copied fixture or a mocked help result | Static help starts calling Shark, or state-aware help still treats bare next as dispatch. |
| TC-018 | Cobra parse/execute for `nextCmd --preview` and `taskNextStatusCmd --preview` | Mocked advisor/lifecycle service below handlers | Flag definitions, Cobra parser, fabricated unknown-flag error | Preview remains accepted on next or is accidentally removed from lifecycle preview. |
| TC-019 | `json.Marshal(PortfolioAdviceEnvelope{...})` using production DTO tags and populated service output | Raw readers for populated service subcase | Shadow JSON struct, post-marshal key deletion, copied golden only | Empty slices become null or session/note fields leak from the domain claim. |
| TC-020 | `PortfolioAdviceService.Advise(ctx)` with the real portfolio/epic/claim repositories in the cleaned repository test DB | Fixed clock only; a counting wrapper may observe reads without replacing SQL | Service assembly, SQL repositories, graph helper | N+1 queries pass unit tests but exceed four reads or 1 second at target scale. |

## Acceptance test cases

### TC-001: Bare next selects the isolated advice call graph

**Feature requirement:** Feature command contract and success conditions;
REQ-F-001, REQ-NF-001, REQ-NF-006.
**Acceptance criterion:** AC-001.
**Technique applied:** Equivalence partitioning and contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, usability, reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `runNext(cmd, []string{})` after Cobra accepts zero arguments.
- **Lowest allowed mock seam:** The advisor service interface and explicit
  fail-on-call keyed adapter/action-service factories.
- **Forbidden mocks:** Cobra argument validation, JSON marshaling, the branch
  itself, or a prebuilt command response.
- **Counter-factual:** A branch placed after keyed initialization calls the
  fail-on-call factory even though the envelope looks correct.

**Preconditions:** The mocked advisor returns E01 evidence and empty allocated
relationship/warning collections. A keyed adapter factory panics or records a
failure if called. The in-memory trace exporter is active.

**Input:** Execute `shark next` with no positional arguments.

**Expected output:** Exit 0; stdout contains exactly one JSON object with
`mode="portfolio_advice"`; the advisor is called once with the command context;
keyed adapter, transitioner, action service, prompt assembly, and status
normalization are called zero times. The `shark.next` span has the portfolio
attributes specified above.

**Edge cases:** Advisor returns a wrapped fatal epic-list error: command returns
non-zero, writes no envelope, and marks the span as error.

**Negative cases:** No claim, history, relationship, document, network, keyed
dispatch, or status mutation method is reachable.

### TC-002: Keyed next preserves the complete dispatch contract

**Feature requirement:** Feature command contract; REQ-F-001 and REQ-NF-007.
**Acceptance criterion:** AC-002.
**Technique applied:** Regression comparison and contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `runNext(cmd, []string{"E36"})` through existing parse,
  `newNextAdapterCache`, `resolveNext`, prompt assembly, and wire normalization.
- **Lowest allowed mock seam:** Existing action/entity service mocks inside
  `nextAdapterCache`.
- **Forbidden mocks:** `resolveNext`, `normalizeWireAction`, prompt assembly, or
  the final `NextResponse`.
- **Counter-factual:** A unified advice/dispatch response drops action fields or
  prevents a legitimate cascade-completion transition.

**Preconditions:** Reuse existing keyed fixtures for a `spawn_agent` step, a
cascade-complete parent, an agentless `advance_status` step, a pause, and a
prompt-rendering failure.

**Input:** Execute `shark next E36` for each fixture.

**Expected output:** Existing `NextResponse` fields, exit behavior, canonical
prompt assembly, cascade traversal, action normalization, and permitted status
normalization match the pre-feature regression suite byte-for-field.

**Edge cases:** Root pause, archive, unresolved placeholders, and maximum
cascade-depth guard retain their existing outcomes.

**Negative cases:** The portfolio advisor is not constructed or called; no
`mode="portfolio_advice"` field appears in keyed output.

### TC-003: More than one key is rejected before any service call

**Feature requirement:** REQ-F-001 CLI argument contract.
**Acceptance criterion:** AC-003.
**Technique applied:** Boundary value analysis.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** Cobra execution of `nextCmd` with
  `[]string{"E36", "extra"}`.
- **Lowest allowed mock seam:** None; both advisor and keyed factory are
  fail-on-call sentinels.
- **Forbidden mocks:** Argument validator or a fabricated usage error.
- **Counter-factual:** `cobra.ArbitraryArgs` accepts the extra argument and
  dispatches the first key.

**Preconditions:** Service factories fail the test if invoked.

**Input:** Zero, one, and two positional arguments; the asserted rejection is
the two-argument partition.

**Expected output:** Zero and one pass argument validation; two returns Cobra's
exact “accepts at most 1 arg(s)” class of error, exits non-zero, and calls no
service.

**Edge cases:** Three arguments are also rejected before handler execution.

**Negative cases:** No partial envelope, dispatch response, read, or write.

### TC-004: Expired claims are filtered at one time without cleanup

**Feature requirement:** Feature read-only success condition; REQ-F-006,
REQ-NF-001, and REQ-NF-005.
**Acceptance criterion:** AC-004.
**Technique applied:** Boundary value analysis and attack-class enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` wired to real
  `ClaimService.ListActiveReadOnly(ctx, evaluatedAt)`.
- **Lowest allowed mock seam:** `claim.Repository.List`; all mutation functions
  are fail-on-call.
- **Forbidden mocks:** `ClaimService`, `EntityClaim.IsExpired`, the captured
  evaluation time, or a prefiltered claim slice.
- **Counter-factual:** Reusing `ClaimService.List` calls `ReclaimExpired` and
  deletes the expired row.

**Preconditions:** Evaluation time is `2026-07-20T18:00:00Z`; TTL is 60 minutes.
Claims heartbeat at `17:00:01Z` (live), `17:00:00Z` (exact boundary per
`IsExpired`), and `16:59:59Z` (expired). Repository call counters cover claim,
reclaim, release, renew, history, and session writes.

**Input:** Build advice once and marshal its active-work evidence.

**Expected output:** Active-work inclusion exactly matches the existing
`IsExpired` boundary policy; the expired claim remains in the repository
fixture; `List` is called once; every mutation counter is zero. A separate
TTL=0 case preserves all claims.

**Edge cases:** Feature and task claims attach through returned descendant rows;
an epic claim attaches directly; a standalone bug claim is ignored.

**Negative cases:** No reclaim, release, renew, insert, update, delete, history,
or session record occurs; no session ID or note appears in output.

### TC-005: Epic visibility follows configured workflow classification

**Feature requirement:** REQ-F-002 and the configured-state risk mitigation.
**Acceptance criterion:** AC-005.
**Technique applied:** Equivalence partitioning.
**ISO 25010 characteristics:** Functional suitability, compatibility,
reliability, maintainability, portability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` with the real configured
  epic workflow classifier.
- **Lowest allowed mock seam:** Raw epic/snapshot/claim readers.
- **Forbidden mocks:** `IsTerminalStatus`, `IsBlockedStatus`, or an epic list
  already filtered by status.
- **Counter-factual:** Hard-coded `completed` filtering excludes E02 and retains
  E01 despite the custom workflow declaring the opposite.

**Preconditions:** Workflow defines `shipped_custom` as terminal,
`held_custom` as blocked, and `completed` as non-terminal. Store E01 in
`shipped_custom`, E02 in `completed`, and E03 in `active`.

**Input:** Call `Advise` with all evidence sources successful.

**Expected output:** E01 is absent; E02 and E03 are present in lexical key order;
E02 is classified according to the custom workflow, not its literal name.

**Edge cases:** An unknown epic status remains visible with eligibility
`unknown` and `UNKNOWN_WORKFLOW_STATUS` listing its key.

**Negative cases:** No literal `completed`, `archived`, or `blocked` branch
determines visibility or eligibility.

### TC-006: Depends-on satisfaction and layers use prerequisite direction

**Feature requirement:** REQ-F-003 and REQ-F-004.
**Acceptance criterion:** AC-006.
**Technique applied:** State transition and decision table.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Readers return raw stored epic and relationship
  rows.
- **Forbidden mocks:** Edge normalization, satisfaction, eligibility, or graph
  layers.
- **Counter-factual:** Treating stored `depends_on` direction as precedence puts
  E03 before E02 and fails the exact layer assertion.

**Preconditions:** E01 is terminal; E02 and E03 are non-terminal. Store
`E03 depends_on E02` and `E02 depends_on E01`.

**Input:** Call advice with successful descendant/claim reads.

**Expected output:** E03 is `ineligible` with
`unresolved_dependency:E02`; E02 is not blocked by E01; dependency layers are
`[["E02"], ["E03"]]`. Both stored relationship rows are returned because each
has a non-terminal endpoint. The E02/E01 relationship is hard and satisfied;
the E03/E02 relationship is hard and unresolved.

**Edge cases:** When E02 becomes terminal, E03 becomes eligible and the
unresolved edge disappears from candidate layers.

**Negative cases:** Terminal E01 is not emitted as a candidate; no satisfied
edge blocks its dependent.

### TC-007: Blocks gates the target until the blocker is terminal

**Feature requirement:** REQ-F-003 and relationship semantics.
**Acceptance criterion:** AC-007.
**Technique applied:** State transition and contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Relationship normalization, satisfaction, blocked-item
  creation, or eligibility.
- **Counter-factual:** Reversing `blocks` produces E02 before E01 and leaves E02
  eligible.

**Preconditions:** E01 and E02 are non-terminal; store `E01 blocks E02`.

**Input:** Call advice, then repeat with E01 terminal.

**Expected output:** Initially E02 is ineligible with `blocked_by:E01`, its
blocked items include an `incoming_block`, relationship precedence is
E01 → E02, and layers are `[["E01"], ["E02"]]`. After E01 becomes terminal,
the relationship is satisfied and E02 is not hard-blocked.

**Edge cases:** Duplicate identical `blocks` rows yield one normalized edge
while retaining contributing type evidence.

**Negative cases:** A `follows` relationship never creates `blocked_by` or an
ineligible result.

### TC-008: Follows chains produce stable roadmap layers

**Feature requirement:** REQ-F-004 and REQ-NF-002.
**Acceptance criterion:** AC-008.
**Technique applied:** State transition and metamorphic testing.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** Repeated `PortfolioAdviceService.Advise(ctx)` calls.
- **Lowest allowed mock seam:** Readers return the same logical rows in
  different permutations.
- **Forbidden mocks:** Sort, normalizer, Kahn traversal, or output DTO.
- **Counter-factual:** Map iteration produces a different layer or warning order
  after the input permutation.

**Preconditions:** Three non-terminal epics; store `E03 follows E02` and
`E02 follows E01`.

**Input:** Call advice at least 20 times while permuting epic and relationship
row order deterministically.

**Expected output:** Every call has roadmap layers
`[["E01"], ["E02"], ["E03"]]`; dependency layers put all three in one lexical
layer because no hard edges exist; complete serialized ordering is identical.

**Edge cases:** Duplicate follows rows do not duplicate keys or warnings.

**Negative cases:** Priority, business value, progress, timestamps, and input
order do not break a layer tie.

### TC-009: Hard cycles remain visible and terminate

**Feature requirement:** Feature success condition and REQ-F-004.
**Acceptance criterion:** AC-009.
**Technique applied:** State transition and attack-class enumeration.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, usability, reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` under a bounded context.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Cycle detector, warning construction, or unlayered list.
- **Counter-factual:** A traversal loops forever or removes an edge and returns a
  plausible total order.

**Preconditions:** E01 depends on E02 and E02 depends on E01; E03 is independent.

**Input:** Build advice with a 100 ms test timeout guard for the small fixture.

**Expected output:** The call terminates normally; warning code is
`HARD_ORDER_CYCLE`; warning keys and `unlayered_epics` are exactly
`["E01", "E02"]`; E03 remains in the acyclic partial layer; warning order is
stable. The trace warning count matches the envelope.

**Edge cases:** A self-cycle reports the one affected key; a larger cycle lists
all and only remaining nodes lexically.

**Negative cases:** No edge repair, deletion, arbitrary winner, hang, panic, or
silent cycle break.

### TC-010: Roadmap cycles do not erase usable hard layers

**Feature requirement:** REQ-F-004.
**Acceptance criterion:** AC-010.
**Technique applied:** Decision table.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, usability, reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Hard/combined graph builders or warning classifier.
- **Counter-factual:** One combined graph result marks the hard graph cyclic and
  discards valid dependency layers.

**Preconditions:** Hard edge E01 → E03 is acyclic. Follows edges normalize to
E01 → E02 and E02 → E01.

**Input:** Build advice once.

**Expected output:** `ROADMAP_ORDER_CYCLE` is present; `HARD_ORDER_CYCLE` is
absent; dependency layers remain usable and place E01 before E03; roadmap
unlayered keys identify the follows-cycle participants.

**Edge cases:** If a hard cycle is also present, both diagnostics follow the
specified code/key/message sort order.

**Negative cases:** Soft order does not change eligibility or hard satisfaction.

### TC-011: Opposing normalized precedence reports contributing types

**Feature requirement:** REQ-F-004.
**Acceptance criterion:** AC-011.
**Technique applied:** Contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Normalization, deduplication, or diagnostic aggregation.
- **Counter-factual:** Deduplicating by unordered key pair drops direction and
  omits the contradiction.

**Preconditions:** Store `E01 depends_on E02` (E02 → E01) and
`E01 blocks E02` (E01 → E02).

**Input:** Build advice with both non-terminal.

**Expected output:** One `CONTRADICTORY_ORDER` warning lists E01 and E02
lexically and names both `depends_on` and `blocks`; both original relationship
records remain in the envelope; affected keys are not forced into a total order.

**Edge cases:** Duplicate rows do not duplicate the warning; a follows/hard
opposition also names both contributing types.

**Negative cases:** The service neither repairs relationships nor chooses a
direction from priority or insertion order.

### TC-012: Incomparable eligible roots stay in one lexical layer

**Feature requirement:** REQ-F-004 and feature success conditions.
**Acceptance criterion:** AC-012.
**Technique applied:** Reachability partitioning and stable-order comparison.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Reachability, warning builder, layer sorting, or a
  recommendation score.
- **Counter-factual:** A weighted sort returns `[["E02"], ["E01"]]` based on
  business value.

**Preconditions:** E01 and E02 are eligible, first-layer candidates with no
precedence path. E02 has higher priority and business value.

**Input:** Build advice.

**Expected output:** `MISSING_ORDERING` names both keys; the applicable layer is
`["E01", "E02"]`; both remain eligible; no subsequent layer or selected key is
invented.

**Edge cases:** Adding a path E01 → E02 removes the warning and separates the
layers; an ineligible first-layer epic does not create a missing-order warning
between eligible candidates.

**Negative cases:** Priority, business value, progress, and active work never
create hidden precedence.

### TC-013: A blocked child is evidence, not automatic root disqualification

**Feature requirement:** REQ-F-002 and eligibility rules.
**Acceptance criterion:** AC-013.
**Technique applied:** Decision table and boundary value analysis.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Raw descendant-state reader.
- **Forbidden mocks:** Blocked-state classifier, direct-feature aggregation,
  blocked-item mapping, or eligibility.
- **Counter-factual:** `anyBlockedChild` marks E01 ineligible despite its active
  dispatchable feature.

**Preconditions:** E01 has direct feature F01 in configured blocked status and
F02 in a non-terminal, non-blocked status. Child rows include their direct
parents.

**Input:** Build advice, then repeat with zero, one, and all non-terminal direct
features blocked.

**Expected output:** In the mixed case, F01 appears as a `workflow_blocked`
item, F02 remains available evidence, and E01 is eligible absent other blockers.
When all non-terminal direct features are blocked, E01 is ineligible with
`all_direct_features_blocked`.

**Edge cases:** No direct features does not vacuously trigger “all blocked”;
terminal/cancelled direct features follow the configured classification and
progress rules.

**Negative cases:** A blocked task or one blocked child alone does not
automatically disqualify the root.

### TC-014: Evidence-source failures degrade by decision table

**Feature requirement:** Advisory prompt gap handling; REQ-F-006.
**Acceptance criterion:** AC-014.
**Technique applied:** Decision table and fault injection.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)`.
- **Lowest allowed mock seam:** Fail exactly one raw reader per subtest.
- **Forbidden mocks:** Error classifier, warning sanitizer, eligibility, prompt,
  or returned DTO.
- **Counter-factual:** A broad `err != nil` aborts every failure, or the service
  retains `eligible` after losing required relationship evidence.

**Preconditions:** Core epics E01 and E02 load. Reader failures use sentinel
errors containing fake SQL and secret strings to verify sanitization.

**Input and expected decision table:**

| Failure | Envelope/error expectation |
|---|---|
| Child-state read | Partial envelope; `evidence_complete=false`; `CHILD_STATE_UNAVAILABLE`; affected eligibility `unknown`; empty descendant blocker/work arrays. |
| Relationship read | Partial envelope; empty relationships/layers; `RELATIONSHIP_STATE_UNAVAILABLE`; relationship-dependent eligibility `unknown`. |
| Claim read | Partial envelope; eligibility retained; empty active work; `CLAIM_STATE_UNAVAILABLE`. |
| Unknown entity status | Entity visible; affected eligibility `unknown`; `UNKNOWN_WORKFLOW_STATUS` with key. |
| Dangling endpoint | Edge omitted from layers; `DANGLING_RELATIONSHIP`; evidence incomplete. |
| Epic list read | Wrapped fatal error; no envelope. |
| Cancelled context | Error preserving `context.Canceled`; no misleading envelope. |

**Observability evidence:** Span has `portfolio.evidence_complete=false` for
partial cases and records only counts, never sentinel SQL/secret text.

**Edge cases:** Two optional failures produce both typed warnings in stable
order; warning key arrays are allocated and sorted.

**Negative cases:** No SQL detail, session ID, claim note, product document,
guess, or false eligible result is returned.

### TC-015: Empty portfolio returns allocated arrays and no-root guidance

**Feature requirement:** REQ-F-002, REQ-F-008, and prompt contract.
**Acceptance criterion:** AC-015.
**Technique applied:** Boundary value analysis.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` with successful empty
  readers.
- **Lowest allowed mock seam:** Raw readers.
- **Forbidden mocks:** Slice initialization, prompt builder, or JSON marshal.
- **Counter-factual:** Nil defaults produce `null`, or a generic prompt still
  demands an epic recommendation.

**Preconditions:** Epic list is empty or contains only configured terminal
epics; all optional reads succeed.

**Input:** Build and marshal advice.

**Expected output:** Exit-success envelope has `evidence_complete=true`;
`epics`, `relationships`, `dependency_layers`, `roadmap_layers`,
`unlayered_epics`, top-level warnings, and ordering warnings are all `[]`;
prompt states that no root can be recommended from current Shark state.

**Edge cases:** One non-terminal epic removes the no-root branch and is eligible
when evidence is complete and no blocker exists.

**Negative cases:** No `null` collection, empty prompt, invented epic key, or
automatic archive/cleanup.

### TC-016: Advisor prompt preserves authority and ends at advice

**Feature requirement:** Advisory prompt contract; REQ-F-005 and REQ-F-006.
**Acceptance criterion:** AC-016.
**Technique applied:** Contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` and its returned `prompt`.
- **Lowest allowed mock seam:** Successful raw readers.
- **Forbidden mocks:** Prompt builder or a copied expected prompt returned by a
  fake advisor.
- **Counter-factual:** A stale prompt treats `progress.md` as workflow state,
  recommends by weighted score, or invokes keyed dispatch.

**Preconditions:** Two eligible epics and complete evidence; a separate partial
case sets `evidence_complete=false`.

**Input:** Inspect the complete and partial prompts.

**Expected output:** In the specified order, the prompt names `docs/product/`
and especially `progress.md` and `cross-epic-integration-map.md`; states Shark
state/relationship/blocker/claim authority; treats documents as intent only;
prioritizes hard precedence before qualitative fields without weights; requests
exactly one eligible epic key, “why now” evidence, and the strongest eligible
alternative; reports gaps/contradictions instead of guessing; and ends without
claim, dispatch, or advance authority.

**Edge cases:** Empty and partial portfolios retain non-empty, condition-specific
instructions.

**Negative cases:** Prompt does not include entity prompt text, claim secrets,
SQL, a preselected key, or permission to mutate state.

### TC-017: Rider help calls advice only in the state-aware variant

**Feature requirement:** Rider integration; REQ-F-007.
**Acceptance criterion:** AC-017.
**Technique applied:** Decision table and contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, usability, reliability, security, maintainability, portability.

**Caller-path contract:**

- **Entrypoint:** Production `skills/shark-rider/verbs/help.md` as routed by
  `skills/shark-rider/SKILL.md`.
- **Lowest allowed mock seam:** Repository filesystem read in the contract test.
- **Forbidden mocks:** Copied help fixture, fabricated router result, or an
  assertion only against comments.
- **Counter-factual:** Updated top-level docs coexist with stale help text that
  still forbids bare next, so real Rider help never consumes advice.

**Preconditions:** Read the production Rider skill and help verb.

**Input and expected decision table:**

| Help input | State-call expectation |
|---|---|
| `/shark-rider help` | Runs bare `shark next` once, consumes its prompt inline, reports one recommendation or gap, and does not dispatch. |
| `/shark-rider help --fast` | Zero Shark state calls; static verb list. |
| `/shark-rider help commands` | Zero Shark state calls; static command reference. |
| `/shark-rider help <known-verb>` | Zero Shark state calls; static verb help. |
| `/shark-rider help <unknown-verb>` | Zero Shark state calls; bounded static correction. |

**Edge cases:** Incomplete evidence follows the envelope prompt and reports the
gap; it does not spawn a subagent.

**Negative cases:** Static modes never invoke bare or keyed next; state-aware
help never calls `shark next <key>`, claims, advances, or passes the prompt to a
spawned worker.

### TC-018: Preview is removed only from next

**Feature requirement:** Out-of-scope boundary; REQ-F-009.
**Acceptance criterion:** AC-018.
**Technique applied:** Equivalence partitioning and regression comparison.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability.

**Caller-path contract:**

- **Entrypoint:** Cobra execution of `shark next --preview` and
  `shark task next-status E36-F04-001 --preview`.
- **Lowest allowed mock seam:** Advisor and lifecycle services below their
  handlers.
- **Forbidden mocks:** Cobra flags/parser or a fabricated error/result.
- **Counter-factual:** Removing the shared-looking flag deletes lifecycle
  preview too, or an inert next flag remains accepted.

**Preconditions:** `nextCmd.Flags().Lookup("preview") == nil`; task lifecycle
preview flag remains registered with its existing handler.

**Input:** Execute both commands.

**Expected output:** `shark next --preview` returns Cobra unknown-flag error,
non-zero, with zero service calls. Task lifecycle preview returns its established
read-only transition preview and performs no transition.

**Edge cases:** `shark next E36 --preview` is also rejected before keyed
resolution.

**Negative cases:** No new advisor preview, dispatch simulation, or removal of
the lifecycle preview contract.

### TC-019: JSON collections and populated claims are safe and stable

**Feature requirement:** Exact data model; REQ-F-008 and REQ-NF-005.
**Acceptance criterion:** AC-019.
**Technique applied:** Boundary value analysis and attack-class enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability,
reliability, security, maintainability, portability.

**Caller-path contract:**

- **Entrypoint:** `json.Marshal` on the production
  `models.PortfolioAdviceEnvelope` returned by `Advise`.
- **Lowest allowed mock seam:** Raw readers for the populated service fixture.
- **Forbidden mocks:** Shadow JSON structs, post-marshal map deletion, or an
  empty-only golden fixture.
- **Counter-factual:** Embedding `EntityClaim` leaks `session_id` and `note`, or
  nil slices marshal as `null`.

**Preconditions:** Partition A contains no relationships, blockers, claims,
layers, or warnings. Partition B contains an epic, feature, and task live claim
with holder `qa-1`, session `secret-session`, note `secret-note`, heartbeat
`2026-07-20T15:04:05Z`, and progress values `nil`, `0`, and `1` across rows.

**Input:** Marshal both service-produced envelopes and decode them to
`map[string]any` for field-presence assertions.

**Expected output:** Every collection is a JSON array in both partitions. Claim
objects expose only `entity_type`, `entity_key`, `claimed_by`,
`last_heartbeat`, and nullable/bounded `progress`; timestamp is RFC 3339 UTC;
progress accepts nil, 0, and 1. No JSON object at any depth contains
`session_id` or `note`. Epic, relationship, reason, blocker, active-work,
layer, and warning arrays follow the exact specified sort keys.

**Edge cases:** Out-of-range progress is rejected, clamped only if the spec is
amended to allow it, or converted into typed incomplete evidence; it must not
silently serialize outside 0-1. `business_value` remains present as JSON null.

**Negative cases:** No `null` collection, omitted required field, sensitive
claim field, local-time timestamp, or map-order-dependent output.

### TC-020: Target-scale SQLite assembly stays within time and query budget

**Feature requirement:** Performance design; REQ-NF-003 and REQ-NF-004.
**Acceptance criterion:** AC-020.
**Technique applied:** Boundary value analysis and performance testing.
**ISO 25010 characteristics:** Functional suitability, performance efficiency,
compatibility, reliability, security, maintainability, portability.

**Caller-path contract:**

- **Entrypoint:** `PortfolioAdviceService.Advise(ctx)` wired through the real
  epic, portfolio, and claim repositories in
  `internal/repository/portfolio/repository_test.go`.
- **Lowest allowed mock seam:** Fixed evaluation clock only; a read-counting DB
  wrapper may observe but must not replace SQL.
- **Forbidden mocks:** Repository results, graph helper, service assembly, or a
  reduced fixture used for the pass threshold.
- **Counter-factual:** Per-epic child/link queries pass small mock tests but
  exceed four reads and 1 second at 200 epics.

**Preconditions:** Clean local SQLite test DB. Seed exactly 200 epics, 5,000
feature/task descendants, and 10,000 supported relevant epic relationships with
an acyclic deterministic pattern; seed representative live and expired claims.
Warm only schema/setup paths, not the result cache. Capture one evaluation time.

**Input:** Run the complete portfolio assembly using a monotonic timer. Repeat
enough times for diagnostics, but apply the acceptance threshold to the defined
single local fixture under the repository's non-parallel performance test.

**Expected output:** Complete envelope is correct and deterministic; wall time
is less than 1 second; dependency calls are at most four set reads: epic list,
child states, epic relationships, and claims. No per-epic query, write, network
call, HTTP call, or document scan occurs.

**Edge cases:** Empty and one-epic fixtures validate lower boundaries; a 201st
epic may be diagnostic but does not redefine the 200-epic acceptance threshold.

**Negative cases:** No N+1 query, schema/version change, lease cleanup,
relationship repair, history insert, telemetry payload leak, or flaky parallel
execution against the shared repository DB.

## Codex test-plan red-team

**Verdict:** UNAVAILABLE (non-blocking review gap)
**Issues raised by Codex:** 0
**Issues addressed:** 0
**Issues deferred:** 1

No concrete `codex_command` was included in the dispatch input. Per the
test-planning workflow, this plan does not invent or substitute an external
review command. The red-team review must be rerun if the orchestrator supplies a
pre-rendered command. The manual completeness pass covered open-endedness,
technique fit, enumeration, ISO 25010 decisions, observability, negative cases,
and production caller paths for every AC.

## Recommendations and verdict

**Verdict: APPROVED.** DRIFT-001 through DRIFT-003 are resolved in the source
documents, and DRIFT-004 is incorporated into this plan's observability
contract. Every feature acceptance criterion and declared integration boundary
has a non-vacuous test design.

Implementation planning must carry forward these non-blocking controls:

1. Copy the exact observability attribute contract from this plan into the task
   that owns CLI tracing.
2. Map every design-element row to an implementation task and preserve the
   caller-path and mock boundaries in each task's test requirements.
3. Create `internal/services/epic_analytics_service_test.go` for the focused
   helper tests and keep `internal/services/epic_service_test.go` as the
   existing higher-level regression suite.

No I-## or X-## coverage is missing because no authoritative map declares one
for E36-F04. The absent parent UAT plan and unavailable Codex command remain
documented non-blocking gaps; neither changes this approval.
