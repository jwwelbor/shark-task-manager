---
feature_key: E34-F01-harness-aware-prompt-rendering
epic_key: E34
title: Harness-aware prompt rendering
description: Record harness metadata at claim/dispatch time and use it to render harness-specific prompt variants when Shark prepares workflow prompts.
---

# Harness-aware prompt rendering

**Feature Key**: E34-F01-harness-aware-prompt-rendering

---

## Epic

- **Epic PRD**: [Epic](../epic.md)
- **Epic Architecture**: [Architecture](../architecture.md)

---

## Goal

### Problem
Shark currently renders a single prompt shape without a reliable way to tailor instructions to the LLM harness that will execute the work. That leaves prompt authors unable to return harness-specific guidance even when different harnesses need different operational instructions, tool assumptions, or formatting. Earlier work added provider/model fields in parts of the dispatch path, but this capture indicates the system still needs a stable way for the harness to identify itself at claim time, including harness type and version.

### Solution
Extend the harness-to-Shark handshake so the harness sends its type, version, and current model when work is claimed or dispatched, then expose that metadata to prompt rendering. Prompt/template authors should be able to branch on harness metadata and return different instructions for different harness families without duplicating entire workflows.

### Impact
- Prompt rendering can return instructions that match the active harness instead of relying on one generic prompt for all runners.
- Harness-specific operational guidance becomes part of the workflow contract rather than hidden in external docs or agent-side conventions.
- Future prompt audits can verify rendered output against concrete harness metadata.

---

## User Stories

### Must-Have Stories

**Story 1**: As a workflow author, I want Shark to know which harness is running claimed work so that rendered prompts can include harness-specific instructions.

**Acceptance Criteria**:
- [ ] The running harness can report a stable harness type and version.
- [ ] The current model can be attached to the same dispatch context.
- [ ] Prompt rendering can branch on harness metadata when building instructions.

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Capture harness metadata
   - **Description**: Shark must accept harness identity metadata from the running harness, including harness type, harness version, and current model, as part of the claim/dispatch context.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Metadata survives long enough to influence prompt rendering for the claimed work.
     - [ ] Missing metadata has a defined fallback behavior.

2. **REQ-F-002**: Render harness-specific prompt variants
   - **Description**: Workflow prompt rendering must support conditional prompt content based on harness metadata so authors can return different instructions for different harnesses.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Prompt authors have a supported mechanism to branch on harness metadata.
     - [ ] The default path still renders a generic prompt when no harness-specific variant exists.

### Non-Functional Requirements

1. **REQ-NF-001**: Backward compatibility
   - **Description**: Existing workflows without harness-conditional logic must continue to render successfully.
   - **Measurement**: Existing prompt workflows still render without requiring harness-specific edits.

---

## Acceptance Criteria

**Scenario 1: Harness-specific render path**
- **Given** a harness claims work and reports its type, version, and current model
- **When** Shark renders the next workflow prompt
- **Then** the renderer can select harness-specific instruction content
- **And** the returned prompt reflects the active harness contract

---

## Out of Scope

1. **Full prompt redesign for every workflow**
   - **Why**: This triage item is about enabling harness-aware rendering, not rewriting all existing prompts immediately.
   - **Future**: Existing workflows can adopt harness-specific branches incrementally after the mechanism exists.

2. **Choosing a final template syntax in this triage pass**
   - **Why**: The user suggested a possible custom template `if` mechanism, but syntax/design selection belongs in the feature design and implementation work.
   - **Future**: The implementation can choose the simplest supported conditional rendering approach.

---

## Success Metrics

1. **Harness metadata available during render**
   - **What**: Whether harness type, version, and model are accessible to prompt rendering for claimed work.
   - **Target**: 100% of supported harness-driven runs expose this metadata to the renderer.
   - **Measurement**: Verified through dispatch/render tests that exercise at least one harness-specific branch and one fallback branch.

---

*Last Updated*: 2026-07-06
