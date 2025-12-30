---
feature_key: E11-F05-workflow-visualization
epic_key: E11
title: Workflow Visualization
description: Generate visual workflow diagrams from configuration to communicate process to stakeholders and team members
---

# Workflow Visualization

**Feature Key**: E11-F05-workflow-visualization

**Status**: DEFERRED - Not yet implemented (Could-Have requirement)

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Scope**: [Scope](../../scope.md)

---

## Goal

### Problem

While workflow configuration in `.sharkconfig.json` is machine-readable and enables runtime enforcement, it is not intuitive for human stakeholders to understand complex workflows by reading JSON. This creates communication challenges:

1. **No visual representation**: JSON config is difficult to parse mentally for complex workflows (14+ statuses)
2. **Stakeholder communication**: Project managers cannot easily share workflow with non-technical stakeholders
3. **Onboarding friction**: New team members must read JSON to understand workflow progression
4. **No diagram generation**: Teams create workflow diagrams manually in draw.io or similar tools (manual sync with config)
5. **Documentation drift**: Manual diagrams become outdated when workflow config changes

Without visualization, teams maintain two sources of truth: the JSON config (actual workflow) and separate diagram documentation (human-readable). This duplication leads to drift where documentation no longer matches reality.

### Solution

Generate visual workflow diagrams automatically from `.sharkconfig.json` configuration:

1. **Mermaid State Diagram**: Generate Mermaid diagram syntax from workflow config
2. **CLI Command**: `shark workflow graph` outputs Mermaid code that can be saved to file
3. **Automatic Sync**: Diagram always matches config (single source of truth)
4. **Status Metadata Integration**: Include status descriptions and phase colors in diagram
5. **GitHub Rendering**: Mermaid syntax renders natively in GitHub markdown, PRs, issues

This enables teams to generate up-to-date workflow diagrams on-demand, ensuring documentation matches actual configuration.

### Impact

- **Communication Efficiency**: Project managers share workflow diagrams with stakeholders without manual diagram creation (estimated 90% time savings)
- **Onboarding Speed**: New team members understand workflow visually (estimated 60% reduction in onboarding time for workflow understanding)
- **Documentation Accuracy**: Diagram always matches config (eliminates documentation drift)
- **Change Visualization**: Git diffs show workflow changes visually when diagram is committed alongside config

---

## User Personas

### Persona 1: Project Manager (Workflow Communicator)

**Profile**:
- **Role/Title**: Project Manager responsible for communicating team process to stakeholders
- **Experience Level**: Moderate technical proficiency, comfortable with markdown and documentation tools
- **Key Characteristics**:
  - Needs to explain workflow to executives, clients, and new team members
  - Wants visual diagrams for presentations and documentation
  - Maintains project documentation in markdown (GitHub wiki, README files)
  - Values automation that reduces manual work

**Goals Related to This Feature**:
1. Generate workflow diagram automatically from config
2. Embed diagram in markdown documentation (GitHub wiki, README)
3. Ensure diagram stays synchronized with config changes
4. Communicate workflow changes visually in pull requests

**Pain Points This Feature Addresses**:
- Previously created workflow diagrams manually in draw.io (time-consuming)
- Manual diagrams became outdated when config changed (documentation drift)
- No easy way to show workflow changes in code review
- Stakeholders couldn't understand JSON config directly

**Success Looks Like**:
Project manager runs `shark workflow graph > docs/workflow.mmd`, commits both config and diagram, and diagram renders in GitHub with visual representation of 14-status workflow. When config changes, diagram regenerates automatically.

---

### Persona 2: Tech Lead (Workflow Designer)

**Profile**:
- **Role/Title**: Tech Lead responsible for designing and evolving team workflow
- **Experience Level**: Expert technical proficiency, deep understanding of workflow systems
- **Key Characteristics**:
  - Designs complex workflows with multiple phases and transitions
  - Needs to visualize workflow before deploying to team
  - Uses diagrams to validate workflow logic (no orphaned statuses, all paths lead to completion)
  - Reviews workflow changes in pull requests

**Goals Related to This Feature**:
1. Visualize workflow during design to validate logic
2. Use diagram to identify workflow problems (unreachable statuses, missing transitions)
3. Review workflow changes visually in code review (see what transitions were added/removed)

**Pain Points This Feature Addresses**:
- Previously validated workflow by reading JSON config (mental graph traversal)
- No visual tool to verify workflow logic before deployment
- Difficult to review workflow changes in pull requests (JSON diffs are hard to parse)

**Success Looks Like**:
Tech lead edits workflow config, runs `shark workflow graph`, sees visual diagram, identifies missing transition from `in_code_review` back to `in_development`, adds transition, and regenerates diagram to verify fix.

---

### Persona 3: New Team Member (Workflow Learner)

**Profile**:
- **Role/Title**: New developer joining project mid-stream
- **Experience Level**: Junior to mid-level developer, learning team processes
- **Key Characteristics**:
  - Needs to understand workflow quickly
  - Learns better from visual diagrams than JSON config
  - Refers to documentation frequently during first weeks
  - Values clear, up-to-date documentation

**Goals Related to This Feature**:
1. Understand workflow progression visually
2. Learn which statuses exist and valid transitions between them
3. Reference diagram when unsure where task should transition next

**Pain Points This Feature Addresses**:
- Previously had to read JSON config to understand workflow (difficult for visual learners)
- No single visual reference for workflow progression
- Diagram in wiki was outdated (didn't match current config)

**Success Looks Like**:
New team member reads project README, sees embedded Mermaid diagram showing 14-status workflow with phase colors, understands workflow progression at a glance, and refers to diagram when transitioning tasks.

---

## User Stories

### Could-Have Stories (Optional per Epic Requirements)

**Story 1**: As a project manager, I want to generate workflow diagrams so that I can communicate process to stakeholders.

**Acceptance Criteria**:
- [ ] `shark workflow graph` command generates Mermaid state diagram syntax
- [ ] Output includes all statuses and transitions from config
- [ ] Output can be saved to file: `shark workflow graph > workflow.mmd`
- [ ] Diagram renders correctly in Mermaid-compatible tools (GitHub, VS Code, Mermaid Live Editor)
- [ ] Special statuses (`_start_`, `_complete_`) highlighted in diagram

**Implementation Status**: ⏳ Not implemented (Could-Have requirement, deferred)

---

**Story 2**: As a tech lead, I want diagrams to include status metadata so that phase colors and descriptions are visible.

**Acceptance Criteria**:
- [ ] Status colors from metadata applied to diagram nodes
- [ ] Status descriptions included as diagram notes or tooltips
- [ ] Phase grouping visible in diagram (statuses grouped by phase)
- [ ] Agent types shown in diagram for each status

**Implementation Status**: ⏳ Not implemented (Could-Have requirement, deferred)

---

**Story 3**: As a developer, I want diagram to show transition direction so that I understand workflow flow.

**Acceptance Criteria**:
- [ ] Arrows show direction of valid transitions
- [ ] Bidirectional transitions shown with bidirectional arrows (e.g., `in_development ↔ ready_for_review`)
- [ ] Terminal statuses (no outgoing transitions) visually distinct
- [ ] Start statuses (initial states) visually distinct

**Implementation Status**: ⏳ Not implemented (Could-Have requirement, deferred)

---

### Should-Have Stories (Future Enhancements)

**Story 4**: As a project manager, I want to customize diagram layout so that it matches team's mental model.

**Acceptance Criteria**:
- [ ] `--layout` flag supports different layouts (top-to-bottom, left-to-right)
- [ ] `--group-by-phase` flag groups statuses by workflow phase
- [ ] `--simple` flag generates minimal diagram (no metadata, just statuses and transitions)
- [ ] Customization options saved in config for consistency

**Implementation Status**: ⏳ Future enhancement (not in initial scope)

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when workflow has circular references, I want diagram to visualize the cycle so that I can see the problem.

**Acceptance Criteria**:
- [ ] Circular transitions shown with bidirectional arrows
- [ ] Cycles that prevent reaching `_complete_` statuses highlighted in diagram (warning color)
- [ ] Diagram includes note: "Warning: Circular reference detected"

**Implementation Status**: ⏳ Not implemented

---

**Error Story 2**: As a developer, when workflow has unreachable statuses, I want diagram to highlight orphaned nodes.

**Acceptance Criteria**:
- [ ] Unreachable statuses shown in different color (e.g., red)
- [ ] Diagram includes note: "Warning: Unreachable status"
- [ ] Validation errors from `shark workflow validate` reflected in diagram

**Implementation Status**: ⏳ Not implemented

---

## Requirements Traceability

This feature implements the following epic requirements:

### Functional Requirements

- **REQ-F-018**: Generate Workflow Diagram (Could-Have)
  - Status: Not implemented
  - Priority: Could-Have (optional for MVP)

### Non-Functional Requirements

None (visualization is user-facing enhancement, no system-level requirements)

---

## Out of Scope

### Explicitly Excluded from This Feature

1. **Interactive Diagram Editing**
   - **Why**: Config is source of truth, diagram is read-only visualization
   - **Rationale**: Editing diagram would require two-way sync (complex and error-prone)
   - **Workaround**: Edit config, regenerate diagram

2. **Diagram Formats Beyond Mermaid**
   - **Why**: Mermaid is sufficient for GitHub integration and is widely supported
   - **Future**: May add GraphViz or PlantUML support if user demand emerges
   - **Workaround**: Mermaid syntax can be converted to other formats using external tools

3. **Automatic Diagram Commits**
   - **Why**: Git workflow decisions should be explicit, not automated
   - **Rationale**: Team decides whether to commit diagram or generate on-demand
   - **Workaround**: Manually commit diagram after generation or add to pre-commit hook

4. **Embedded Diagram in CLI Output**
   - **Why**: CLI output is terminal-based, not image-based
   - **Rationale**: Mermaid rendering requires browser or specialized viewer
   - **Workaround**: Use `shark workflow graph | mermaid-cli` for image generation

5. **Workflow Diff Visualization**
   - **Why**: Complex feature requiring Git integration
   - **Future**: May add in separate enhancement
   - **Workaround**: Generate diagram before and after config change, compare manually

---

## Success Metrics

### Primary Metrics

Since this feature is DEFERRED (Could-Have), no success metrics are actively tracked. If implemented in future:

1. **Diagram Generation Adoption**
   - **What**: Percentage of projects with custom workflows that generate diagrams
   - **Target**: >40% of projects use `shark workflow graph` within 90 days
   - **Measurement**: CLI telemetry or survey

2. **Documentation Currency**
   - **What**: Percentage of projects where diagram matches config (no drift)
   - **Target**: >90% of projects with diagrams have up-to-date diagrams
   - **Measurement**: Analyze Git commits (diagram updated when config updated)

3. **Stakeholder Satisfaction**
   - **What**: Project manager satisfaction with diagram for stakeholder communication
   - **Target**: >80% find diagram "very useful" for stakeholder communication
   - **Measurement**: User survey

---

## Implementation Summary

### Tasks NOT Yet Created

This feature is DEFERRED (Could-Have requirement). When prioritized for implementation, tasks would include:

1. **T-E11-F05-001**: Implement Mermaid diagram generation
   - Status: Not started
   - Scope: Generate Mermaid state diagram syntax from workflow config
   - Deliverables:
     - Mermaid diagram generator in `/internal/workflow/diagram.go`
     - `shark workflow graph` CLI command
     - Output all statuses and transitions
     - Highlight special statuses (`_start_`, `_complete_`)

2. **T-E11-F05-002**: Integrate status metadata into diagram
   - Status: Not started
   - Scope: Apply colors, descriptions, and phase grouping to diagram
   - Deliverables:
     - Status colors applied to diagram nodes
     - Descriptions included as notes
     - Phase grouping visible in diagram

3. **T-E11-F05-003**: Write diagram generation tests
   - Status: Not started
   - Scope: Unit tests for diagram generator
   - Deliverables:
     - Test Mermaid syntax output
     - Test edge cases (circular refs, unreachable statuses)
     - Test metadata integration

---

## Dependencies & Integrations

### Dependencies

- **F01: Workflow Configuration & Validation** (COMPLETED)
  - Provides workflow config structure for reading statuses and transitions
  - Provides validation logic for detecting workflow errors

- **F04: Agent Targeting & Metadata** (COMPLETED)
  - Provides status metadata (color, description, phase) for diagram enrichment

### Integration Points (When Implemented)

- **Workflow Config**: Read `status_flow` and `status_metadata` sections
- **CLI Command**: `shark workflow graph` outputs Mermaid syntax
- **File Output**: Standard output can be redirected to `.mmd` file
- **GitHub Integration**: Mermaid renders in markdown files on GitHub

---

## Testing Strategy (When Implemented)

### Unit Tests

**Scope**:
- Mermaid syntax generation for simple workflow
- Mermaid syntax generation for complex workflow (14+ statuses)
- Metadata integration (colors, descriptions, phases)
- Special status highlighting (`_start_`, `_complete_`)
- Edge cases (circular refs, unreachable statuses)

**Test Cases**:
1. **Simple Workflow**: 4-status workflow generates valid Mermaid syntax
2. **Complex Workflow**: 14-status workflow generates valid Mermaid syntax
3. **Circular Transitions**: Bidirectional transitions shown correctly
4. **Unreachable Statuses**: Highlighted in different color with warning
5. **Metadata Integration**: Colors and descriptions included in diagram

### Manual Testing

**Scope**:
- Mermaid diagram renders correctly in GitHub
- Mermaid diagram renders correctly in VS Code (Mermaid plugin)
- Mermaid diagram renders correctly in Mermaid Live Editor
- Diagram layout is readable for complex workflows

---

## Documentation Requirements (When Implemented)

### User Guide

Create guides for:
- "Generating Workflow Diagrams" - How to use `shark workflow graph`
- "Embedding Diagrams in Documentation" - How to include Mermaid diagrams in markdown
- "Customizing Diagram Appearance" - How to use layout and grouping options
- "Mermaid Syntax Reference" - Understanding generated Mermaid code

### Examples

- Example Mermaid diagram for simple workflow
- Example Mermaid diagram for standard 14-status workflow
- Example Mermaid diagram with phase grouping
- Example embedded in GitHub README

---

## Compliance & Security Considerations

### No Security Implications

- Diagram generation is read-only operation (no data modification)
- No sensitive data in diagrams (only workflow structure)
- Diagram output is text-based (no binary formats or security risks)

---

## Future Enhancements

Potential enhancements if feature is implemented:

1. **Multiple Diagram Formats**: GraphViz, PlantUML support
2. **Interactive Diagrams**: Clickable statuses with drill-down to task lists
3. **Workflow Diff Visualization**: Visual comparison of workflow changes
4. **Diagram Customization**: Layout options, color themes, grouping
5. **Automatic Diagram Updates**: Pre-commit hook to regenerate diagram
6. **Diagram Embedding**: Render diagram in CLI using ASCII art
7. **Workflow Templates with Diagrams**: Ship with preset workflows and diagrams

---

## Rationale for Deferral

This feature is deferred (Could-Have) because:

1. **Lower Priority**: Core workflow functionality (F01-F04) provides value without visualization
2. **Workaround Exists**: `shark workflow list` provides text-based visualization
3. **Limited User Demand**: Text-based config sufficient for most technical teams
4. **Implementation Complexity**: Diagram generation adds complexity for optional feature
5. **Resource Constraints**: Focus on Must-Have and Should-Have features first

### When to Prioritize

This feature should be implemented when:
- Multiple teams request visual diagrams for stakeholder communication
- Documentation drift becomes a common problem
- Onboarding new team members takes longer due to workflow complexity
- Project managers spend significant time creating workflow diagrams manually

### Workarounds Until Implemented

Teams can:
1. Use `shark workflow list` for text-based workflow visualization
2. Create diagrams manually in draw.io or similar tools
3. Use online Mermaid editors to create diagrams from config (manual process)
4. Use `shark workflow validate` to detect workflow errors without visual representation

---

*Last Updated*: 2025-12-29
*Status*: DEFERRED - Could-Have requirement, not yet implemented
*Priority*: Low - implement when user demand emerges
*Workaround*: `shark workflow list` provides text-based visualization
