# User Journeys

**Epic**: [Bug and Change-Card Management System](./epic.md)

---

## Overview

This document maps the key user workflows enabled by this epic. There are three primary journeys: bug lifecycle (report through resolution), change-card lifecycle (propose through completion), and cross-entity discovery (finding and linking related items).

---

## Journey 1: Bug Lifecycle -- Report, Triage, Fix, Verify

**Persona**: Developer (report, fix), QA Engineer (triage, verify)

**Goal**: Track a defect from discovery through verified resolution

**Preconditions**:
- Shark is initialized with the advanced workflow profile
- Bug entity type is enabled
- User has CLI access to the project

### Happy Path

1. **Developer Reports Bug**
   - User action: Runs `shark bug create "Login fails on Safari when 2FA is enabled" --severity=high --link=E07-F01`
   - System response: Creates bug B001 in `reported` status, generates markdown file at `docs/bugs/B001.md`, links to feature E07-F01
   - Expected outcome: Bug appears in `shark bug list` and `shark status` dashboard

2. **QA Engineer Triages Bug**
   - User action: Runs `shark bug list --status=reported` to see new bugs, then `shark bug triage B001 --severity=high --assign=developer`
   - System response: Advances B001 to `triaged` status, assigns severity, records agent assignment
   - Expected outcome: Bug shows in developer's assigned work via `shark bug list --status=triaged --agent=developer`

3. **Developer Picks Up Bug**
   - User action: Runs `shark bug get B001` to read reproduction steps, then `shark status advance B001` to move to `in_fix`
   - System response: Advances B001 to `in_fix`, records timestamp
   - Expected outcome: Bug status reflects active work; dashboard shows it is being fixed

4. **Developer Completes Fix**
   - User action: Runs `shark status advance B001` after implementing the fix
   - System response: Advances B001 to `in_verification`
   - Expected outcome: Bug appears in QA engineer's verification queue

5. **QA Engineer Verifies Fix**
   - User action: Tests the fix, then runs `shark status advance B001`
   - System response: Advances B001 to `resolved`, records resolution timestamp
   - Expected outcome: Bug is marked resolved; resolution time is calculable from report-to-resolved timestamps

**Success Outcome**: Bug is tracked from discovery to verified resolution with full audit trail. Resolution time and severity are captured for analytics.

### Alternative Paths

**Alt Path A: Bug Marked as Duplicate**
- **Trigger**: QA engineer identifies B001 as a duplicate of an existing bug
- **Branch Point**: After Step 2 (triage)
- **Flow**:
  1. QA runs `shark status set B001 duplicate` with a note referencing the original bug
  2. Bug is closed as duplicate; original bug remains active
- **Outcome**: Duplicate is tracked but does not inflate bug counts; link to original is preserved

**Alt Path B: Bug Marked as Won't Fix**
- **Trigger**: Team decides the reported behavior is intentional or the fix cost exceeds the impact
- **Branch Point**: After Step 2 (triage) or Step 4 (fix complete)
- **Flow**:
  1. Developer or QA runs `shark status set B001 wont_fix` with a note explaining the rationale
  2. Bug is closed as wont_fix
- **Outcome**: Decision is documented; bug does not count as unresolved in metrics

**Alt Path C: Verification Fails**
- **Trigger**: QA engineer tests the fix and finds it incomplete
- **Branch Point**: After Step 5 (verification)
- **Flow**:
  1. QA runs `shark status set B001 in_fix` to send it back to the developer
  2. QA adds a note explaining what failed verification
  3. Developer re-fixes and re-submits for verification
- **Outcome**: Bug cycles between in_fix and in_verification until the fix passes

### Critical Decision Points

- **Decision at Step 2**: QA engineer decides severity. Critical bugs may trigger immediate attention; low severity bugs may be deferred. Severity is a context field, not a status.
- **Decision at Step 5**: QA engineer decides whether the fix passes verification. This is the quality gate that prevents incomplete fixes from being marked resolved.

---

## Journey 2: Change-Card Lifecycle -- Propose, Approve, Implement

**Persona**: Developer (propose, implement), Product Owner (approve/decline)

**Goal**: Track a small enhancement from proposal through completion without full feature decomposition

**Preconditions**:
- Shark is initialized with the advanced workflow profile
- Change-card entity type is enabled
- User has CLI access

### Happy Path

1. **Developer Proposes Change-Card**
   - User action: Runs `shark change create "Add keyboard shortcuts for common actions" --link=E17`
   - System response: Creates change-card C001 in `proposed` status, generates markdown file at `docs/changes/C001.md`, links to epic E17
   - Expected outcome: Change-card appears in `shark change list` and is visible to product owner

2. **Product Owner Reviews Proposal**
   - User action: Runs `shark change list --status=proposed` to see pending proposals, then `shark change get C001` for details
   - System response: Displays change-card details including description, linked epic, and proposer
   - Expected outcome: Product owner has enough context to make an approval decision

3. **Product Owner Approves Change-Card**
   - User action: Runs `shark change approve C001`
   - System response: Advances C001 to `approved` status
   - Expected outcome: Change-card appears in developer's available work queue

4. **Developer Implements Change-Card**
   - User action: Runs `shark status advance C001` to move to `in_progress`, implements the change, then runs `shark status advance C001` again
   - System response: Advances C001 through `in_progress` to `completed`
   - Expected outcome: Change-card is marked complete; throughput metrics are updated

**Success Outcome**: Small enhancement is tracked from idea to completion in under 5 minutes of CLI interaction total, with approval gate ensuring only valuable changes are implemented.

### Alternative Paths

**Alt Path A: Change-Card Declined**
- **Trigger**: Product owner decides the change is not worth implementing now
- **Branch Point**: After Step 2 (review)
- **Flow**:
  1. Product owner runs `shark status set C001 declined` with a note explaining the rationale
  2. Change-card is closed as declined
- **Outcome**: Decision is documented; developer understands why and can repropose later with additional justification

**Alt Path B: Change-Card Promoted to Feature**
- **Trigger**: During review, product owner determines the change is larger than a change-card warrants
- **Branch Point**: After Step 2 (review)
- **Flow**:
  1. Product owner creates a feature: `shark feature create E17 "Keyboard Shortcuts System"`
  2. Product owner links the change-card as context and declines it: `shark status set C001 declined` with note "Promoted to feature E17-F03"
- **Outcome**: Work is tracked at the appropriate level of decomposition; change-card serves as the originating context

### Critical Decision Points

- **Decision at Step 3**: Product owner decides approve vs. decline. This is the value gate that prevents low-impact changes from consuming development capacity.
- **Decision at Step 2 (Alt B)**: Product owner decides whether the change-card scope warrants promotion to a full feature with task decomposition.

---

## Journey 3: Cross-Entity Discovery and Linking

**Persona**: All personas (Developer, QA Engineer, Product Owner)

**Goal**: Find bugs and change-cards related to a specific feature, epic, or task

**Preconditions**:
- Bugs and/or change-cards exist with links to features or epics
- User knows the key of the entity they want to inspect

### Happy Path

1. **User Queries Related Bugs for a Feature**
   - User action: Runs `shark bug list --link=E07-F01` or `shark search "login" --type=bug`
   - System response: Returns bugs linked to feature E07-F01 or matching the search query
   - Expected outcome: User sees all bugs related to a specific area of the codebase

2. **User Views Feature Status with Bug Context**
   - User action: Runs `shark status E07-F01`
   - System response: Feature status includes a "Related Bugs" section showing open bug count and severity breakdown
   - Expected outcome: User understands the quality state of a feature at a glance

3. **User Checks Project-Wide Bug and Change-Card Metrics**
   - User action: Runs `shark analytics`
   - System response: Analytics include bug count by status, severity distribution, average resolution time, and change-card throughput
   - Expected outcome: User can assess overall quality trends and the balance between planned and reactive work

**Success Outcome**: Users can navigate from any entity to its related bugs and change-cards, and dashboards provide aggregate quality metrics without manual data collection.

---

*See also*: [Requirements](./requirements.md)
