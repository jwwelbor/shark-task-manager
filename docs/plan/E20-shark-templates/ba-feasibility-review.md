# BA Feasibility Review: E20 -- Shark Templates

**Review Date**: 2026-03-18
**Reviewer**: Business Analyst (Claude Opus 4.6)
**Epic**: E20 -- Shark Templates
**Status at Review**: ready_for_feasibility_review_ba

---

## Executive Summary

E20 proposes externalizing workflow and template configuration from `.sharkconfig.json` into a dedicated `.sharkworkflow.json` file and standardizing the task workflow block structure to match the other four entity types. The research report confirms the problem is real, the solution is technically feasible with moderate effort (~400-600 lines of new/modified code), and the existing infrastructure provides a strong foundation. From a business perspective, this epic is well-scoped, low-risk, backward-compatible, and supportive of downstream epics (E21, E19). No showstopper concerns were identified.

**Overall Assessment: APPROVED**

---

## (1) Cross-Epic Conflict Analysis

### E21 -- Entity Polymorphism and Duplication Reduction (Draft)

**Finding: SYNERGISTIC, no conflict.**

E21 proposes a shared `Entity` interface and generic `EntityService` to eliminate ~1,255+ lines of cross-entity duplication. E20's task workflow standardization directly benefits E21 by removing the task workflow special case. After E20, all five entity types use identical `{entity}_workflow` block structure, which means E21's generic workflow-loading code does not need to handle two different config formats.

The research report correctly identifies that E20 should ideally complete before E21 begins service-layer generalization. The two epics touch different layers (E20: config loading; E21: models/services/repositories), so parallel work on non-overlapping components is possible, but sequencing E20 first is preferred.

**Risk**: None. E20 is an enabler for E21.

### E19 -- Sprint Management and Planning System (Draft)

**Finding: SUPPORTIVE, no conflict.**

E19 introduces sprints as a new entity type with a `S###` key pattern and its own lifecycle. If sprints need workflow configuration (e.g., `sprint_workflow`), E20's externalized workflow file provides the natural home for it. The consistent `{entity}_workflow` block pattern established by E20 would make adding a sixth entity workflow straightforward.

**Risk**: None. E19 does not depend on E20, but would benefit from it.

### E15 -- Service Layer Architecture Refactoring (In Progress)

**Finding: COMPATIBLE, minor interaction.**

E15 is migrating CLI commands to use the service layer. E20 changes config loading, which is consumed by services via `workflow.Service`. The research confirms that `workflow.Service` initialization already accepts workflow data from the config layer transparently, so E15's refactoring is compatible with E20's config changes. No coordination required beyond normal code review.

**Risk**: None.

### E11 -- Configurable Status Workflow System (Completed) and E16 -- Multi-Level Workflow (Completed)

**Finding: ALIGNED, E20 is a natural successor.**

E11 established the config-driven workflow pattern. E16 introduced per-entity workflow blocks for epic/feature/bug/change. E20 completes the pattern by adding `task_workflow` block support and externalizing all workflow config. The research confirms E20 builds directly on E16's `parseWorkflowSection()` function.

**Risk**: None. E20 extends completed work without modifying it.

### Summary

No cross-epic conflicts detected. E20 is either neutral or positively supportive of all identified related epics.

---

## (2) Market Viability Assessment

### Does research support the business case?

**Finding: SUPPORTED.**

The research report validates the business case on three dimensions:

1. **Configuration externalization is an established industry pattern.** The report cites ESLint, Docker Compose, Kubernetes, GitHub Actions, and Git itself as precedents for splitting configuration by concern domain. This is not a novel or risky approach -- it follows well-understood patterns.

2. **The problem is real and quantified.** `.sharkconfig.json` is 1,759 lines. Workflow definitions for five entity types consume the vast majority of that space. The task workflow inconsistency (legacy top-level keys vs. per-entity blocks) is a documented bifurcation in the config loading code. These are not theoretical concerns -- they affect every developer who modifies workflow configuration.

3. **The estimated effort is proportionate to the value.** ~400-600 lines of new/modified code to solve a real configuration management problem, standardize five entity workflows, and enable future capabilities (workflow presets, template sharing). The effort-to-value ratio is favorable.

### Is the business value rating appropriate?

The epic rates its business value as **Medium**, which is appropriate. The direct user-facing impact is modest (developers get a smaller config file to edit), but the indirect value is higher: consistent config structure simplifies future development, reduces subtle bugs from the task workflow special case, and enables future features (workflow presets, template set sharing) that the scope document correctly identifies as future epic candidates.

**Risk**: None. The business case is solid and proportionate.

---

## (3) Scope Coherence Assessment

### Do requirements still make sense given research findings?

**Finding: COHERENT, requirements are well-matched to research findings.**

The research report analyzed each requirement individually and found all Must Have requirements (REQ-F-001 through REQ-F-007) to be feasible with identified implementation paths in the existing codebase. Key observations:

1. **Must Have requirements are tightly scoped.** They cover exactly the problem described in the epic (file separation, task standardization, backward compatibility, init update) without scope creep. Nothing extraneous was included.

2. **Should Have requirements are appropriate.** Config validation (REQ-F-010) and config show source (REQ-F-011) are natural developer experience improvements that the existing validation and display infrastructure can support with low effort.

3. **Could Have requirements are correctly classified.** Deprecation warnings (REQ-F-020) and workflow export (REQ-F-021) are valuable but not essential for the core value proposition. Deferring them is the right call if time pressure exists.

4. **Non-functional requirements are concrete and measurable.** Zero breaking changes (REQ-NF-001) with 100% backward compatibility is verifiable. The 5ms performance target (REQ-NF-002) is reasonable given the research's finding that JSON parsing ~1,500 lines takes 1-2ms. The single code path requirement (REQ-NF-003) is achievable because only `workflow_parser.go` needs file-source awareness.

5. **Out-of-scope items are well-justified.** The scope document explicitly excludes template marketplace, per-entity-instance overrides, GUI config editor, YAML/TOML support, and automatic inline cleanup. Each exclusion has a clear rationale and documented workaround. The future epic candidates table provides a clear path for deferred work.

### Minor observation: Feature structure

E20-F01 (Template Configuration Externalization) has been cancelled and absorbed into E20-F02 (Enhancements and Maintenance). E20-F02 currently uses a boilerplate template with no filled-in content. This is acceptable at the feasibility review stage -- feature PRDs are typically elaborated during refinement, not during the epic feasibility phase. However, this should be addressed before moving to development: E20-F02 needs its own problem/solution/impact statement and user stories.

**Risk**: None at this stage. The feature-level elaboration gap is expected and will be addressed in subsequent workflow phases.

---

## (4) Business Risk Assessment

### Showstopper Risks

**Finding: NONE.**

1. **Backward compatibility risk is mitigated by design.** The fallback chain (workflow file > `.sharkconfig.json` inline > built-in defaults) ensures existing projects work without any changes. This is the correct default-safe approach. The research confirms the existing code already handles missing config gracefully.

2. **Regression risk is manageable.** The existing test suite in `workflow_multilevel_test.go` and `workflow_test.go` covers all entity levels. The research recommends adding an integration test that loads the real `.sharkconfig.json` before and after changes, which is a sensible mitigation.

3. **Cache invalidation risk is identified and mitigable.** The research flags that the caching layer keys on file path, and with two files, stale cache could mask changes. The mitigation (clear both caches when either file is loaded) is straightforward. The existing `ClearWorkflowCache()` function already clears both caches.

4. **No user migration burden.** The workflow file is optional. Existing projects continue to work without it. New projects get it via `shark init`. This is the ideal adoption model for a configuration change -- zero cost to ignore, immediate benefit to adopt.

5. **Blast radius is limited.** Changes are confined to the config loading layer (`internal/config/`), the init command (`internal/cli/commands/init.go`), and test fixtures. Core business logic, services, repositories, and CLI commands are unaffected. The `MultiLevelWorkflow` struct serves as an abstraction boundary that shields consumers from the two-file model.

### Minor Risks

1. **E20-F02 feature PRD is not yet elaborated.** The feature document uses boilerplate template content. This needs to be filled in during the refinement phase before development begins. This is a process gap, not a business risk.

2. **File naming inconsistency.** The epic PRD references `.sharkworkflow.json` while E20-F01 (now cancelled) referenced `.sharkworkflow.config`. The epic PRD's `.sharkworkflow.json` is the resolved decision (JSON format, `.json` extension), which is consistent with `.sharkconfig.json`. This should be verified during refinement to ensure all documents reference the same filename.

---

## Recommended Actions

### Immediate (Before Advancing)

None required. The epic is ready to advance.

### During Refinement Phase

1. **Elaborate E20-F02 feature PRD.** Fill in the problem/solution/impact, user stories, and acceptance criteria using the content from the epic PRD and requirements documents. The cancelled E20-F01 feature PRD has useful content that should be carried forward into E20-F02.

2. **Verify filename consistency.** Confirm that all documents reference `.sharkworkflow.json` (not `.sharkworkflow.config`) as the default workflow file name.

3. **Confirm E20 sequencing relative to E21.** The research recommends E20 complete before E21 begins service-layer changes. Document this ordering dependency in both epic PRDs.

---

## Conclusion

E20 is a well-defined, low-risk, backward-compatible epic that addresses a real configuration management problem and establishes infrastructure for future capabilities. The research confirms technical feasibility, the requirements are coherent and appropriately prioritized, and no cross-epic conflicts exist. The business value is proportionate to the effort.

**Assessment: APPROVED**

The epic should advance to the next workflow phase.

---

*Review completed: 2026-03-18*
*Reviewer: Business Analyst Agent (Claude Opus 4.6, 1M context)*
