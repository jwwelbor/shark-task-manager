# Complexity Triage Report: E27-F13

**Feature**: E27-F13 - Sprint Mode Dashboard, Tree, and Planning Surface  
**Epic**: E27 - Shark Status Viewer - Local Web Dashboard  
**Assessment Date**: 2026-05-07  
**Assessed By**: Codex orchestration pass

## Scope Validation

**Classification**: FEATURE

**Why not a task**:
- User-visible sprint surface with three subviews: `Overview`, `Plan`, and `Report`
- Cross-cuts viewer UI, navigation, data loading, and planning/reporting behavior
- Requires coordination with existing sprint services and viewer APIs
- More than one implementation step and more than one layer of the stack

## Complexity Score

**Total**: 17/27  
**Tier**: COMPLEX  
**Routing Decision**: `ready_for_research`

## Dimension Scoring Matrix

### Technical Complexity

1. **File Impact**: **3/3**
   - Likely touches 10+ files across `internal/services`, `internal/api/viewer`, `internal/viewer/assets`, `internal/viewer/server`, and tests.
   - The sprint surface is not a small surgical change to one existing component.

2. **Pattern Novelty**: **2/3**
   - Extends the existing single-file viewer pattern, but introduces a major new mode and new view composition.
   - Reuses the current viewer shell rather than inventing a new framework, but the interaction model is materially new.

3. **Data Model**: **0/3**
   - No schema change is implied by the seed scope.
   - Sprint data should come from existing sprint/repository structures.

4. **API Surface**: **2/3**
   - Sprint overview, plan, and report views will likely require new viewer service methods and response DTOs.
   - Even if the implementation leans on existing sprint services, the viewer contract will need expansion.

5. **Cross-Feature Dependencies**: **2/3**
   - Depends on existing viewer, sprint, analytics, hierarchy, and workflow plumbing.
   - The feature spans multiple existing subsystems and cannot be isolated to one package.

6. **UI Complexity**: **3/3**
   - New navigation mode, persistent sprint tree, tabbed subviews, planning surface, reporting surface, and detail drawer behavior.
   - The wireframes show a dense, stateful interface with several coordinated panes.

### Execution Complexity

7. **Task Estimation**: **3/3**
   - Estimated 8+ implementation tasks once decomposed properly.
   - Likely work includes data plumbing, view state, sprint tree, overview, plan/report views, responsive behavior, and tests.

8. **Regression Risk**: **2/3**
   - Additive in intent, but it touches the primary local viewer shell and shared viewer services.
   - Existing dashboard behavior could regress if layout/state management is not carefully isolated.

9. **Execution Effort**: **2/3**
   - Estimated at roughly 3-4 weeks of work across design, wiring, implementation, and verification.
   - Too large for a quick refinement-only pass.

## Summary

This is a real feature, not a task. The work is a multi-layer sprint mode extension to an already substantial viewer application, with enough UI and service complexity to justify the COMPLEX tier.

## Next Step

Route to `ready_for_research` so the feature can produce a codebase-aware implementation plan before specification or build work.
