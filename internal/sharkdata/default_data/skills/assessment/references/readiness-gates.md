---
inputs:
  # This reference is loaded by the assessment skill during readiness_check mode.
  # The caller (SKILL.md craft) supplies:
  #   - gate_id (string) — selects which gate definition to apply
  #   - phase_artifacts (map) — artifacts produced in current phase
  #   - upstream_state (map) — outputs from prior phases (complexity_tier, prd_path, etc.)
  #
  # Gate IDs in this reference are tool-agnostic — they describe phases of feature
  # development, not specific status names in any particular project tracker. The host
  # workflow is responsible for mapping its own status vocabulary to gate IDs.
outputs:
  # The reference informs PASS/FAIL decisions and populates structured criteria results.
  # No advancement / status-mutation actions — those are host-layer concerns.
---

# Readiness Gates — Phase Transition Requirements

Validation criteria for advancing features and tasks through workflow phases.

## Overview

Readiness gates ensure work items meet quality standards before advancing to the next phase. Each phase has entry criteria, deliverables, and exit criteria that must be satisfied.

**Key principle**: gates protect downstream phases from incomplete work. Better to catch issues early than discover them during implementation or testing.

**Gate verdicts are binary**: PASS or FAIL. There is no "conditional pass." If a criterion can't be confirmed, it fails — and the feature returns to the prior phase for completion.

---

## Feature-Level Readiness Gates

Each feature-level gate has a stable `gate_id`. The host workflow maps its own status vocabulary to these IDs.

### G1_triage — Ready for Triage

**Phase**: initial assessment.
**Purpose**: determine if work belongs at feature level and assign complexity tier.

**Entry criteria**:

- [ ] Feature exists with a title.
- [ ] Feature artifact (file, record, or other persistent form) exists.
- [ ] Feature metadata contains required fields (parent epic, feature key, status).
- [ ] Parent epic exists and is valid.

**Validation questions**:

1. Is this actually a feature (not a task)? See `scope-criteria.md`.
2. Does it deliver standalone user/stakeholder value?
3. Is it small enough to complete (not an epic)?

**Exit criteria**:

- [ ] Complexity score calculated (0-27 points).
- [ ] Tier assigned (SIMPLE, STANDARD, COMPLEX).
- [ ] Routing decision made (skip BA/arch OR full workflow).
- [ ] Complexity metadata stored.

**Artifacts**:

- Complexity triage report (in feature notes or file).
- Tier assignment with score justification.

---

### G2_research — Ready for Research (COMPLEX only)

**Phase**: deep analysis for complex features.
**Purpose**: understand existing patterns, identify integration points, assess feasibility.

**Entry criteria**:

- [ ] Complexity tier = COMPLEX.
- [ ] Feature description provides sufficient context.
- [ ] Epic PRD available for business context.

**Research deliverables**:

- [ ] Codebase analysis report (existing patterns, similar implementations).
- [ ] Integration points identified.
- [ ] Technical risks documented.
- [ ] Effort estimate refined.
- [ ] Recommendation: proceed OR split OR redesign.

**Exit criteria**:

- [ ] All research deliverables completed.
- [ ] Technical approach validated as feasible.
- [ ] No blockers identified (or blockers have a mitigation plan).
- [ ] Research findings documented in related-docs.

**Validation questions**:

1. Do we understand the existing codebase sufficiently?
2. Are integration points clear?
3. Are technical risks acceptable?
4. Should the feature be split into smaller features?

---

### G3_ba_refinement — Ready for BA Refinement

**Phase**: business analysis.
**Purpose**: define clear requirements, acceptance criteria, and business rules.

**Entry criteria** (STANDARD/COMPLEX only):

- [ ] Complexity tier = STANDARD or COMPLEX.
- [ ] Feature description exists.
- [ ] Epic PRD provides business context.
- [ ] Research complete (if COMPLEX tier).

**BA deliverables**:

- [ ] Feature PRD with all required sections (see `specification-writing` skill).
- [ ] User stories defined.
- [ ] Acceptance criteria clear and measurable.
- [ ] Business rules documented.
- [ ] Edge cases identified.
- [ ] Success metrics defined.

**Exit criteria**:

- [ ] Feature PRD complete and reviewed.
- [ ] Acceptance criteria are SMART (Specific, Measurable, Achievable, Relevant, Time-bound).
- [ ] No ambiguities or contradictions in requirements.
- [ ] Business stakeholder alignment (if applicable).

**Validation questions**:

1. Are requirements clear enough to design a technical solution?
2. Are acceptance criteria testable?
3. Are business rules complete and consistent?
4. Are edge cases handled?

**Artifacts**:

- Feature PRD document.
- Related-docs registration.

---

### G4_tech_refinement — Ready for Technical Refinement

**Phase**: architecture and design.
**Purpose**: define technical approach, API contracts, data models, and implementation strategy.

**Entry criteria**:

- [ ] BA refinement complete (PRD exists).
- [ ] Requirements are clear and unambiguous.
- [ ] Research findings available (if COMPLEX).

**Architecture deliverables**:

- [ ] Architecture document with technical approach.
- [ ] API contracts defined (endpoints, request/response schemas).
- [ ] Data model designed (tables, relationships, constraints).
- [ ] Component interaction diagrams.
- [ ] Technology stack decisions documented.
- [ ] Security considerations addressed.
- [ ] Performance requirements identified.

**Exit criteria**:

- [ ] Architecture document complete and reviewed.
- [ ] Technical approach is sound and feasible.
- [ ] API contracts align with business requirements.
- [ ] Data model satisfies all use cases.
- [ ] No unresolved technical risks.
- [ ] Tech-lead approval (if required).

**Validation questions**:

1. Does the technical design satisfy all business requirements?
2. Are API contracts complete and consistent?
3. Is the data model normalized and scalable?
4. Are security requirements addressed?
5. Are performance requirements realistic?

**Artifacts**:

- Architecture document.
- API contract specifications.
- Data model diagram / DDL.
- Related-docs registration.

---

### G5_task_generation — Ready for Task Generation

**Phase**: task breakdown.
**Purpose**: create executable implementation tasks.

**Entry criteria**:

- [ ] Feature PRD exists (STANDARD/COMPLEX) OR feature description clear (SIMPLE).
- [ ] Architecture document exists (STANDARD/COMPLEX) OR approach is obvious (SIMPLE).
- [ ] Technical approach validated.
- [ ] No blockers or unresolved dependencies.

**Task generation deliverables**:

- [ ] 2-15 implementation tasks created.
- [ ] Tasks properly sequenced (execution order).
- [ ] Dependencies identified.
- [ ] Agent types assigned (backend, frontend, qa, etc.).
- [ ] Each task has clear acceptance criteria.
- [ ] Task artifacts created with metadata.

**Exit criteria**:

- [ ] All tasks created.
- [ ] Task order makes sense (foundational tasks first).
- [ ] No circular dependencies.
- [ ] Each task is completable in <2 days.
- [ ] Tasks cover all requirements from PRD.
- [ ] Test planning complete (for TDD workflow).

**Validation questions**:

1. Do tasks cover all acceptance criteria from PRD?
2. Are tasks properly sequenced?
3. Are dependencies correctly identified?
4. Is each task small enough (<2 days)?
5. Are test scenarios defined?

**Artifacts**:

- Task files / records.
- Test plan (if using advanced workflow).
- Task dependency graph (if complex dependencies).

---

### G6_autonomous_build — Ready to Build (Autonomous Build)

**Phase**: autonomous implementation attempt.
**Purpose**: validate that the feature is suitable for autonomous AI build.

**Entry criteria**:

- [ ] Complexity tier = SIMPLE or STANDARD.
- [ ] All tasks created and ready.
- [ ] Test plan exists (if advanced workflow).
- [ ] No blockers.

**Autonomous build feasibility checks**:

- [ ] Task count ≤ 10 (context window limit).
- [ ] Regression risk ≤ 1 (low risk, additive changes).
- [ ] Execution effort ≤ 2 weeks (fits in single session).
- [ ] No circular dependencies (AI can sequence).
- [ ] All specs are complete (no ambiguities).

**Build process** (host-orchestrated, listed for context):

1. Tech-director reviews feature and tasks.
2. Product-manager dispatches agents per task.
3. Developers implement with TDD.
4. Tech-lead reviews code.
5. QA validates against test plan.
6. All tasks reach the approval-ready state.

**Abort conditions** (stop autonomous build):

- Task count exceeds context capacity.
- Specification contradictions discovered.
- Unresolvable blocker encountered.
- Regression risk higher than assessed.

**Exit criteria** (successful autonomous build):

- [ ] All tasks completed.
- [ ] All tests passing.
- [ ] Code review passed.
- [ ] QA validation passed.
- [ ] Feature ready for UAT.

**Artifacts**:

- Implementation code.
- Test suites.
- Code review reports.
- QA test results.

---

### G7_uat — Ready for Approval (UAT)

**Phase**: user acceptance testing.
**Purpose**: validate that the feature meets business requirements and design intent.

**Entry criteria**:

- [ ] All implementation tasks completed.
- [ ] All tests passing.
- [ ] Code review approved.
- [ ] QA validation passed.

**UAT deliverables**:

- [ ] UAT test guide generated.
- [ ] Test scenarios derived from PRD acceptance criteria.
- [ ] Cross-feature integration tests defined.
- [ ] Epic alignment validated.

**UAT process** (host-orchestrated):

1. Generate or retrieve UAT guide.
2. Execute test scenarios interactively.
3. User validates each scenario (PASS/FAIL).
4. Record results.
5. Fix failures or approve.

**Exit criteria**:

- [ ] All UAT scenarios passed.
- [ ] Feature meets acceptance criteria.
- [ ] No critical bugs found.
- [ ] Epic-level requirements satisfied.
- [ ] User/stakeholder sign-off.

**Validation questions**:

1. Does the feature deliver promised value?
2. Does it align with epic goals?
3. Does it integrate correctly with related features?
4. Are there any critical bugs?

**Artifacts**:

- UAT guide document.
- UAT results report.
- Approval sign-off.

---

## Task-Level Readiness Gates

### T1_development — Ready for Development

**Entry criteria**:

- [ ] Task created with metadata.
- [ ] Parent feature has PRD and architecture docs.
- [ ] Dependencies are met (all upstream tasks completed).
- [ ] Agent type assigned.
- [ ] Test plan available (if advanced workflow).

**Exit criteria**:

- [ ] Implementation complete.
- [ ] Tests written and passing.
- [ ] Self-validated against acceptance criteria.

---

### T2_code_review — Ready for Code Review

**Entry criteria**:

- [ ] Implementation complete.
- [ ] Tests written and passing.
- [ ] Code follows project standards.
- [ ] No compilation/linting errors.

**Tech-lead review checklist**:

- [ ] Code quality (readability, maintainability).
- [ ] Tests comprehensive (edge cases, error paths).
- [ ] Follows architectural patterns.
- [ ] Security best practices.
- [ ] Performance considerations.
- [ ] Documentation adequate.

**Exit criteria**:

- [ ] Code review approved, OR
- [ ] Minor issues auto-fixed, OR
- [ ] Major issues → return to development with fix scope.

---

### T3_qa — Ready for QA

**Entry criteria**:

- [ ] Code review passed.
- [ ] All tests passing.
- [ ] Feature branch builds successfully.

**QA validation**:

- [ ] Run test plan scenarios for this task.
- [ ] Validate against task acceptance criteria.
- [ ] Check integration with related tasks.
- [ ] Exploratory testing.
- [ ] Verify design alignment.

**Exit criteria**:

- [ ] All test scenarios passed, OR
- [ ] Issues found → create fix tasks.

---

### T4_approval — Ready for Approval

**Entry criteria**:

- [ ] QA validation passed.
- [ ] All tests passing.
- [ ] Acceptance criteria met.
- [ ] No blockers.

**Approval process**:

- Included in feature-level UAT.
- User validates the task's contribution to the feature.

**Exit criteria**:

- [ ] UAT passed for the parent feature (including this task).
- [ ] No regression issues.

---

## Gate Bypass Conditions

### When to Skip Gates

Some gates can be skipped based on tier or context.

**Skip BA Refinement (G3)**:

- Tier = SIMPLE.
- Feature description is self-explanatory.
- No business rules or edge cases.

**Skip Technical Refinement (G4)**:

- Tier = SIMPLE.
- Technical approach is obvious.
- No new patterns or architecture needed.

**Skip Research (G2)**:

- Tier = SIMPLE or STANDARD.
- Existing patterns well understood.
- No complex integration.

**Skip Autonomous Build (G6)**:

- Tier = COMPLEX.
- Task count > 10.
- Regression risk > 1.
- Requires human strategic decision-making.

---

## Gate Failure Handling

### What Happens When a Gate Fails

**Scenario**: a feature enters G4 (technical refinement) but the architecture review identifies incomplete requirements.

**Action**:

1. Return the feature to G3 (BA refinement).
2. Document what's missing in rejection notes.
3. BA refines requirements.
4. Re-validate G3.
5. Proceed to G4 again.

**Rejection record** (stored by host):

- Specific issue identified.
- What needs to be completed.
- Who needs to address it.
- Severity (blocking vs advisory).

---

## Gate Metrics

### Tracking Gate Effectiveness

Monitor these metrics to validate gate effectiveness:

**Gate pass rate**:

```
Pass Rate = (Features passing gate on first attempt) / (Total features)
```

- Target: >80% for most gates.
- <60% → gate criteria may be unclear or upstream phase incomplete.

**Defect leakage**:

```
Leakage = (Defects found after gate) / (Total defects)
```

- Target: <20% for critical gates (code review, QA).
- High leakage → gate criteria need strengthening.

**Rework rate**:

```
Rework = (Features returned to previous phase) / (Total features)
```

- Target: <15%.
- High rework → upstream phases incomplete.

---

## Summary

**Key principles**:

1. Gates ensure quality before advancing.
2. Gates are checkpoints, not barriers (facilitate progress).
3. Failed gates trigger rework, not rejection.
4. Gates can be skipped when tier/context justifies it.
5. Gate criteria should be clear and measurable.
6. Verdicts are binary: PASS or FAIL. No conditional passes.

**Quality philosophy**:

- Catch issues early (cheaper to fix).
- Validate completeness before handoff.
- Clear criteria reduce ambiguity.
- Gates protect team time and focus.

**Decision tree**:

```
Feature enters phase
  │
  ├─ Check entry criteria
  │  │
  │  ├─ All criteria met → proceed with phase work
  │  └─ Criteria missing → reject, document gaps, return to previous phase
  │
  ├─ Complete phase deliverables
  │
  └─ Check exit criteria
     │
     ├─ All criteria met → advance to next phase
     └─ Criteria not met → stay in phase, complete deliverables
```

**Related references**:

- `complexity-dimensions.md` — scoring criteria for triage gate.
- `scope-criteria.md` — feature vs task classification for initial gate.
- `tier-thresholds.md` — tier-based routing decisions affecting gates.
