# UAT Document Template

Use this template structure when generating UAT documents. UAT validates that features meet **epic-level requirements** and integrate properly with sibling features.

---

```markdown
# User Acceptance Testing Guide

**Feature:** {feature-key} - {feature-title}
**Epic:** {epic-key} - {epic-title}
**Generated:** {YYYY-MM-DD}
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:**
{Epic description and primary objectives from epic.md or shark}

**This Feature's Role in the Epic:**
{How this feature contributes to the epic's goals. Why does this feature exist?}

**Related Features in Epic:**

| Feature | Title | Status | Integration Points |
|---------|-------|--------|-------------------|
| {key} | {title} | {status} | {how it relates to this feature} |

**Document Sources:**
- Epic PRD: {path from project or fallback}
- Feature PRD: {path from project or fallback}
- Design Docs: {paths from related-docs}

---

## Design Intent

**From Epic PRD:**
> {Quote key requirements/vision from epic document}

**From Feature PRD:**
> {Quote key requirements/acceptance criteria from feature document}

**Key Design Decisions:**
- {Decision 1 and rationale}
- {Decision 2 and rationale}

**Original Vision:**
{What was this feature supposed to accomplish? What problem does it solve?}

---

## Prerequisites

### Environment
- [ ] Application running at: {URL or localhost:port}
- [ ] Test database seeded with required data
- [ ] Required services running: {list services}
- [ ] Related features deployed: {list sibling features needed}

### Credentials
- **Test User:** {email/username}
- **Password:** {password or "see .env.test"}

### Data Setup
{Any test data that needs to exist, including data from related features}

---

## Cross-Feature Integration

### Integration Point 1: {Feature A} → {This Feature}
**Scenario:** {Description of how features interact}
**Features Involved:** {list feature keys}

#### Test Steps
1. {Step in Feature A}
2. {Transition to this feature}
3. {Verify integration works}

#### Expected Results
- [ ] {Integration outcome 1}
- [ ] {Integration outcome 2}

---

### Integration Point 2: {This Feature} → {Feature B}
**Scenario:** {Description of downstream integration}
**Features Involved:** {list feature keys}

#### Test Steps
1. {Action in this feature}
2. {Verify downstream effect in Feature B}

#### Expected Results
- [ ] {Integration outcome}

---

## Epic Acceptance Validation

These criteria come from the **EPIC level**. Validate that this feature contributes correctly to epic goals.

| Epic AC | Description | This Feature's Contribution | Status |
|---------|-------------|----------------------------|--------|
| {epic-ac-id} | {epic requirement} | {how this feature helps} | |

---

## Feature Acceptance Validation

These criteria come from the **FEATURE PRD**. Validate each is met.

| Feature AC | Description | Status |
|------------|-------------|--------|
| {feature-ac-id} | {requirement from PRD} | |

---

## Test Scenarios

### Scenario 1: {Primary Happy Path - Epic Context}

**Epic Alignment:** {Which epic goal this validates}
**Acceptance Criteria:** {AC from epic or feature PRD}
**Tasks Covered:** {task-ids}

#### Spec Requirements
**From Epic PRD:**
> {Direct quote from epic showing the requirement this scenario validates}

**From Feature PRD:**
> {Direct quote from feature PRD with acceptance criteria language}

**Success Criteria (from spec):**
- {Measurable criterion 1 derived from spec language}
- {Measurable criterion 2 derived from spec language}

#### Implementation Code
```{language}
{Actual code excerpt implementing this scenario's functionality}
{Include file path and line numbers as comment at top}
{Show the key function/class/method, not the entire file}
```

#### Test Code (Setup → Execute → Verify)
```{language}
{Actual test code for this scenario}
{Show fixture/setup so user can see how it's instantiated}
{Show assertions so user can see what's being checked}
```

**Input:** {What goes in - parameters, request body, initial state}
**Expected Output:** {What comes out - return value, response, final state}

#### Use Case
{How this code gets used in the real system:}
1. {Trigger: what initiates this flow}
2. {Process: what calls this code, with what parameters}
3. {Result: what consumes the output, what happens next}

**Upstream:** {What feeds into this}
**Downstream:** {What depends on this output}

#### Verification Steps

1. {Step with expected result}
2. {Step with expected result}
3. {Step with expected result}

#### Spec Fidelity Check

| Spec Requirement | Expected | Actual | Match? |
|------------------|----------|--------|--------|
| {From spec quote above} | {Expected behavior} | {Observed behavior} | |

---

### Scenario 2: {Cross-Feature Integration}

**Features Involved:** {list feature keys}
**Integration Type:** {data sharing / workflow handoff / UI navigation}

#### Spec Requirements
**From Epic PRD:**
> {Quote showing how these features should work together}

**From Feature PRDs:**
> {Quotes from both features showing the integration contract}

#### Integration Code
```{language}
{Code showing where Feature A connects to Feature B}
{Show the interface/contract between features}
```

#### Integration Test Code
```{language}
{Test that validates the cross-feature integration}
{Show setup of both features, the interaction, and verification}
```

#### Verification Steps

1. {Start in related feature}
2. {Transition to this feature}
3. {Verify integrated behavior}

#### Expected Results
- [ ] {Integration works as designed}
- [ ] {Data flows correctly between features}

---

### Scenario 3: {Error Handling / Edge Case}

**Acceptance Criteria:** {AC-XXX from PRD}
**Tasks Covered:** {task-ids}

#### Spec Requirements
> {Quote from PRD about error handling / edge case behavior}

#### Implementation Code
```{language}
{Code showing the error handling path}
```

#### Test Code
```{language}
{Test that triggers the error condition and verifies handling}
```

**Input (error trigger):** {What causes the error}
**Expected Output:** {How the system should respond}

#### Verification Steps

1. {Step that triggers error condition}
2. {Verify error handling}

#### Expected Results
- [ ] {Error message displayed}
- [ ] {User can recover}
- [ ] {No data corruption}

---

## Interactive Tests

For tests requiring user interaction or file operations:

### {Test Name}
```bash
python dev-artifacts/{feature-key}/interactive_{test}.py
```

**Instructions:**
1. Run the script
2. {User action required}
3. {Verification step}

---

## Test Scripts Reference

| Script | Purpose | Run Command |
|--------|---------|-------------|
| `test_{name}.py` | {purpose} | `python dev-artifacts/{feature-key}/test_{name}.py` |

---

## Issues Found

| Issue # | Category | Description | Severity | Task to Fix |
|---------|----------|-------------|----------|-------------|
| | Epic Alignment | | | |
| | Integration | | | |
| | Feature AC | | | |

**Severity Levels:**
- **Critical:** Blocks epic goals, breaks integration with other features
- **Major:** Feature doesn't meet acceptance criteria, significant UX issues
- **Minor:** Cosmetic, edge case, has workaround

**Category Types:**
- **Epic Alignment:** Feature doesn't fulfill its role in the epic
- **Integration:** Breaks with other features in the epic
- **Feature AC:** Doesn't meet feature-level acceptance criteria
- **Design Intent:** Doesn't match original vision/design

---

## Sign-Off Checklist

### Epic Alignment
- [ ] Feature fulfills its intended role in the epic
- [ ] Contributes to epic-level acceptance criteria
- [ ] Aligns with original epic vision/design intent

### Cross-Feature Integration
- [ ] Works correctly with all related features in epic
- [ ] Data flows properly between features
- [ ] No regressions in sibling features

### Feature Acceptance
- [ ] All acceptance criteria from feature PRD verified
- [ ] All test scenarios pass
- [ ] Error handling works as expected

### Quality Checks
- [ ] Performance acceptable (page loads < 2s)
- [ ] No console errors during testing
- [ ] Responsive design verified (if applicable)
- [ ] Consistent with other features in epic (branding, UX patterns)

### Final Approval
- [ ] **UAT PASSED** - Feature approved for production
- [ ] **UAT FAILED** - Issues documented above need resolution

**Tester:** _______________________
**Date:** _______________________
**Notes:**

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | (none yet) |
| Result | - |
| Passed | - |
| Open Issues | - |
| Results File | - |

**Previous Sessions:** None

---

## Appendix: Source Data

<details>
<summary>Epic Data (click to expand)</summary>

```json
{epic entity details}
```

</details>

<details>
<summary>Feature Data (click to expand)</summary>

```json
{feature entity details}
```

</details>

<details>
<summary>Related Documents</summary>

**Epic Documents:**
{epic related documents list}

**Feature Documents:**
{feature related documents list}

</details>

<details>
<summary>Sibling Features</summary>

{sibling features list with status}

</details>
```

---

## Template Usage Notes

1. **Epic Context** - Always frame the feature within its epic. Explain WHY this feature exists.
2. **Design Intent** - Quote directly from epic and feature PRDs. Don't paraphrase or invent.
3. **Cross-Feature Integration** - Critical for UAT. Test how this feature works with siblings.
4. **Epic AC vs Feature AC** - Separate these clearly. Epic ACs are higher-level goals.
5. **Issues Categories** - Categorize issues by type (epic alignment, integration, feature AC).
6. **Last UAT Status** - Leave empty; UAT agent fills this with session results.
7. **Document Sources** - Get paths from the project management system first, fall back to conventions.

### Show/Tell Principle

Each scenario MUST include evidence the user can verify independently:

- **Spec quotes** - Direct quotes from PRD, not summaries. User needs to see the original language.
- **Implementation code** - Actual code excerpts with file:line references. Read the file and include the relevant function/class. Don't just reference it.
- **Test code** - Full test function including setup, execution, and assertions. For backend/engine code without UI, tests ARE the user interface - they show how the code is instantiated, what goes in, and what comes out.
- **Use cases** - How the code fits into the system. What calls it, what consumes its output. Even for low-level code, show the chain: trigger → this code → downstream consumer.
- **Spec fidelity table** - Close the loop: for each spec requirement, show expected vs actual. Don't make the user infer whether the spec was met.

**The goal:** A user who has never seen the code should be able to read a scenario and understand: what was promised (spec), how it was built (code), how it's tested (tests), how it's used (use case), and whether it matches (fidelity check).
