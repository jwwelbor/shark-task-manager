# E27-F03 Complexity Assessment Summary

**Status**: Assessment Complete → Ready for Research  
**Assessment Date**: 2026-04-11  
**Assessor**: Researcher Agent  

---

## Executive Summary

Feature **E27-F03: Single-File SPA - IDE-Style Dark Dashboard Interface** has been assessed as **COMPLEX** (score 22/27).

**Key Finding**: This is a substantial UI implementation (~2300 LOC) with high visual complexity and architectural novelty. It requires upfront research to validate API contracts and extract visual requirements before formal specification.

---

## Complexity Score: 22/27 (COMPLEX Tier)

### Scoring by Dimension

| # | Dimension | Score | Assessment |
|---|-----------|-------|------------|
| 1 | Scope & File Impact | 3/3 | Single large HTML file with comprehensive functionality |
| 2 | UI/UX Complexity | 3/3 | 4 application states, 10+ interactive components, 30-color palette |
| 3 | Data Integration | 2.5/3 | 7 API endpoints, moderate data transformation required |
| 4 | State Management | 2/3 | 6 state variables, 6 primary transitions |
| 5 | JS Logic & Interaction | 2.5/3 | Complex tree operations, 7 async calls, moderate algorithms |
| 6 | Testing Complexity | 2.5/3 | Single file = hard unit testing; visual testing required |
| 7 | Styling & Visual Polish | 2.5/3 | Extensive CSS, 30 distinct colors, precision layout required |
| 8 | Pattern Novelty & Risk | 2.5/3 | Novel SPA pattern in project; moderate architectural risks |
| 9 | Development Effort | 2/3 | ~2300 LOC estimated; moderate execution difficulty |

**Tier Classification**:
- SIMPLE: 0-6 points
- STANDARD: 7-15 points
- **COMPLEX: 16+ points** ← This feature

---

## Feature Components Overview

The feature implements a complete dark-themed IDE-style dashboard interface with:

1. **Pick Folder Screen** — Initial state with folder selection
2. **Dashboard View** — Overview with 4 sections:
   - Status Breakdown cards (per entity type)
   - Feature Progress bars (interactive)
   - Active Transitions (last 25 sorted by recency)
   - Stale Entities (non-terminal, >7 days old)
3. **Entity Detail View** — Properties panel + Info/Transitions toggle + markdown content
4. **History Drawer** — Reverse-chronological status changes
5. **Doc View** — Plain markdown rendering
6. **Sidebar Tree Navigator** — Hierarchical Epics/Features/Tasks with expand/collapse

---

## Key Complexity Drivers

### 1. High UI Complexity (Score: 3/3)
- 4 distinct application states with different layouts
- 10+ interactive components (tree nodes, buttons, toggles, nav)
- 30-color status palette requiring precise color mapping
- Multiple responsive layout modes
- Icon usage (folder, arrows, status dots)

### 2. Large Single File (Score: 3/3)
- Estimated ~2300 lines total:
  - HTML: ~300 lines
  - CSS: ~800 lines
  - JavaScript: ~1200 lines
- No build step, no modules = all code in one file
- Maintainability concern as file grows

### 3. API Dependency Risk (Score: 2.5/3)
- Consumes 7 `/api/v1/viewer/` endpoints from E27-F02
- **Critical Blocker**: Feature cannot be implemented until E27-F02 (Viewer API) is complete
- Entire SPA depends on correct API response structure
- If API contract differs from SPA expectations, whole feature breaks

### 4. Tree Navigation Complexity (Score: 2.5/3)
- Complex tree operations:
  - Expand/collapse recursive nodes
  - Navigate to selected entity
  - Expand ancestors if node is collapsed
  - Scroll to view in sidebar
- Requires careful event handling and state tracking

### 5. Visual/Functional Testing Challenge (Score: 2.5/3)
- Single HTML file difficult to unit test without jsdom setup
- No module structure for isolated testing
- Requires **visual/functional testing** against reference screenshots
- Manual testing against UI spec (docs/status-viewer-ui.md)
- Browser compatibility verification needed

---

## Routing Decision: READY_FOR_RESEARCH

**Reason**: This COMPLEX feature requires research before formal specification.

**Research Priorities**:

1. **API Contract Validation** (CRITICAL)
   - Verify E27-F02 API response structure
   - Confirm all 7 endpoints return expected data shape
   - Document required vs. optional fields
   - Check color values in workflow-meta endpoint

2. **Reference Materials Analysis**
   - Study 3 reference screenshots (/home/jwwel/Pictures/status/tracker/)
   - Extract exact pixel dimensions, spacing, typography
   - Identify visual patterns and conventions
   - Document hover states and animations

3. **UI Spec Verification**
   - Review docs/status-viewer-ui.md for precision
   - Validate 30 status colors against specification
   - Confirm indentation values (16px, 40px, 56px)
   - Verify application states and transitions

4. **Risk Assessment**
   - Single file maintainability strategy
   - Performance implications (large file, many DOM nodes)
   - Browser API dependencies (File System API for folder picker)
   - Minification/optimization approach

5. **Task Decomposition**
   - Plan breakdown into implementable, testable tasks
   - Estimate 5-8 tasks for ~2300 LOC
   - Identify dependencies between tasks
   - Sequence visual components for testing

---

## Dependencies

### Blocking Dependencies
- **E27-F02** (Viewer API Endpoints) — MUST be complete and tested before implementation starts

### Non-Blocking Dependencies
- **E27-F05** (Server Wiring) — Can proceed in parallel; handles embedding and serving

---

## Reference Materials

1. **UI Specification**: `/home/jwwel/projects/shark-task-manager/docs/status-viewer-ui.md`
2. **Reference Screenshots**: `/home/jwwel/Pictures/status/tracker/` (3 images of working prototype)
3. **Feature Description**: Feature file in this directory
4. **Parent Epic PRD**: Epic.md in parent E27 directory
5. **Architecture Context**: CLAUDE.md

---

## Next Steps (For Researcher Agent)

After assessment routing to `ready_for_research`, the researcher agent will:

1. Analyze existing codebase for related implementations
2. Validate E27-F02 API contract
3. Study reference screenshots and UI spec
4. Document integration points and extension strategies
5. Create comprehensive research report with findings
6. Produce actionable task decomposition plan

Output: Feature research report suitable for architect/designer in specification phase.

---

## Notes

- Assessment note with detailed reasoning stored in feature
- All 9 complexity dimensions documented with evidence
- Risk mitigation strategies identified
- Clear research priorities for next phase
