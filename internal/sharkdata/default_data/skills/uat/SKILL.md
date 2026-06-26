---
name: uat
description: Orchestrate User Acceptance Testing for a feature — gather evidence, run Codex as the sole assessor, present a no-opinion report to the user, record results, route rejections, and triage non-blocking findings.
when_to_use: when running UAT approval for a feature whose tasks are at the approval stage
inputs:
  - feature_id: opaque feature identifier (string)
  - feature_title: feature title for report headers (string)
  - epic_id: opaque epic identifier (string)
  - epic_title: epic title for report headers (string)
  - feature_prd_path: absolute path to the feature PRD markdown
  - epic_prd_path: absolute path to the epic PRD markdown
  - task_specs: list of {task_id, title, spec_path, ac_list} for tasks under this feature ready for approval
  - sibling_features: list of {feature_id, title, completion_status, integration_notes} for cross-feature integration context
  - design_doc_paths: list of absolute paths to relevant design documents (optional)
  - uat_doc_path: absolute path where the UAT test guide lives or should be created
  - evidence_path: absolute path where the raw evidence file should be written
  - results_path: absolute path where the session results document should be written
  - codex_command: pre-rendered codex exec command for the red-team review (caller assembles)
  - doc_only_mode: bool — when true, generate/refresh the UAT document only and skip the session
outputs:
  - verdict: "Accept" | "Accept with Conditions" | "Reject" | "Insufficient Evidence"
  - uat_document: markdown written to uat_doc_path (when generated/refreshed)
  - evidence_document: markdown written to evidence_path
  - codex_assessment: raw codex output (the sole assessment)
  - results_document: markdown written to results_path
  - rejected_tasks: list of {task_id, classification: "AC Violation" | "Spec Gap", unmet_criteria, fix_required}
  - non_blocking_findings: list of {description, codex_severity, ac_or_location_ref} ready for triage routing
  - approved_task_ids: list of task IDs the user approved
user_invocable: false
version: 3.0.0
---

# User Acceptance Testing (UAT)

UAT validates that a feature fulfills its **epic-level** role and integrates correctly with sibling features — not "does it pass tests" but "does it do what we needed."

**Announce at start:** "I'm using the UAT skill to orchestrate acceptance testing for this feature."

## UAT vs QA — what UAT specifically does

QA happens before UAT and verifies technical correctness: tests pass, AC met at the task level, code quality / security / performance.

UAT covers what QA cannot:

- **Epic-level alignment** — does this feature fulfill its role in the epic?
- **Cross-feature integration** — does it work with sibling features?
- **Design intent validation** — does it match the original vision/PRD?
- **Holistic perspective** — Application → Epic → Feature hierarchy.
- **Stakeholder view** — not technical, but goal-oriented.

## The UAT procedure

### Step 1 — Generate the UAT document (if not present at `uat_doc_path`)

Skip this step if the UAT doc already exists and is current. Otherwise, create the document by combining the inputs:

- **Epic context** from `epic_prd_path` and `epic_title` (epic goal, this feature's role).
- **Cross-feature integration** from `sibling_features` (data sharing, workflow handoffs, shared UI/nav, dependencies).
- **Design intent** from `feature_prd_path` (key requirements, design decisions). Quote the actual PRD text — paraphrasing loses the contract.
- **Cross-feature integration test scenarios** — for each sibling, identify integration points and write scenarios that exercise them.
- **Epic acceptance validation table** — each epic AC × this feature's contribution × verification status.
- **Feature acceptance validation table** — each feature AC × verification status.
- **Per-scenario task tracing** — list tasks covered by each scenario.
- **Last UAT Status section** — placeholder updated after each session.

The `references/uat-template.md` describes the exact document structure.

If `doc_only_mode=true`, stop here and return.

### Step 2 — Gather evidence (Claude collects, no analysis)

**Claude's role is evidence collection only.** Do not assess, judge, or rate scenarios. Claude reviewing Claude's work adds no value (monoculture blind spot). Just gather raw materials for Codex.

For each test scenario in the UAT document, collect:

1. **Spec quotes** — the exact PRD/AC language being verified (the actual text, not a summary).
2. **Implementation code** — relevant code excerpts with `file:line` references.
3. **Test code** — test functions including setup, execution, assertions, input/output contracts.
4. **Use case trace** — what calls this code? What consumes its output? How is it wired into the runtime entry point?
5. **Test execution output** — run tests/commands and capture raw output (pass/fail, errors, coverage).

Write all collected evidence to `evidence_path` so Codex can read it alongside the source artifacts.

### Step 3 — Codex red-team review (sole assessor, MANDATORY)

Codex is the **sole assessor**. Run `codex_command` and pass it the evidence file alongside the source artifact paths.

This step MUST complete before presenting anything to the user. If Codex fails after retries, the verdict is **"Insufficient Evidence"** — do not proceed to user review.

The reason Codex is the sole assessor: Claude reviewing Claude's implementation is a monoculture blind spot. The independent codex review with explicit enumeration discipline catches things both Claude and Claude-the-reviewer would miss. Per the user's standing instruction, codex red-team is mandatory at UAT and is never skipped.

### Step 4 — Compile the report (evidence + Codex assessment, NO Claude opinion)

Build the UAT report containing:

1. **Evidence summary** — per scenario: spec quotes, code references, test output. Facts only, no Claude judgment.
2. **Codex assessment** — full Codex findings: per-criterion verdicts, CRITICAL/HIGH issues, wiring checks, contract verification, overall recommendation.
3. **Codex verdict** — Accept / Accept with Conditions / Reject / Insufficient Evidence.

**Do NOT add Claude's opinion, agreement/disagreement analysis, or recommendations.** The report presents evidence + Codex assessment for the human to review.

### Step 5 — Human review (present and get verdict)

Present the report to the user. The user reviews evidence + Codex's assessment, then makes the final judgment. The user may:

- **Approve all** — proceed to mark tasks completed (only after explicit approval).
- **Reject specific scenarios** — record which and why.
- **Request re-review** — if evidence is insufficient.
- **Override Codex** — user judgment is final.

Use a question-asking mechanism appropriate to the host (for example, a host-provided question tool or a direct CLI prompt). Never auto-approve.

### Step 6 — Record results and update state

Write a session results document to `results_path` containing:

- Session metadata (date, feature, epic).
- Final verdict.
- Codex assessment summary with severity-ranked findings.
- User's per-scenario decisions.
- Approved task IDs.

**Do NOT auto-complete tasks.** Tasks are marked completed only after explicit user approval — the host workflow handles the actual state transition (this skill returns `approved_task_ids`).

### Step 7 — Route rejections (per-task classification)

When UAT rejects specific tasks, route each failing task individually. **Do not blanket-reject the whole feature** — passing tasks stay approved.

For each failing task:

1. **Classify the rejection reason**:
   - **AC Violation** — the task spec was clear; the implementation didn't meet it. The fix is implementation-only.
   - **Spec Gap** — the requirement was missing or ambiguous in the task spec. Needs refinement before re-implementation.

2. **Record the rejection in the task spec** (append a section to the task `.md` file):

   ```markdown
   ## UAT Rejection (<YYYY-MM-DD>)

   **Classification:** AC Violation | Spec Gap
   **Unmet criteria:** <AC-ID> — <quote the AC text>

   **What happened:** <brief description of the failure>

   **Fix required:**
   1. <concrete step>
   2. <concrete step>

   **Full UAT results:** <results_path>
   ```

3. Return the rejection in `rejected_tasks` so the host workflow can route the task back to development with the appropriate context (bug-fix flag, blocker note, status reset).

### Step 8 — Triage non-blocking findings (MANDATORY for every verdict)

Regardless of verdict — Accept, Accept with Conditions, Reject, or Insufficient Evidence — every Codex finding labeled **non-blocking** and every "Required Follow-ups" item MUST be returned in `non_blocking_findings` so the host can route them through `/triage`.

**Severity passthrough:** include each finding's Codex-assigned severity (CRITICAL / HIGH / MEDIUM / LOW) in the entry. Without severity, every triaged finding looks equally urgent and floods the backlog.

**Severity mapping**:
- **Codex CRITICAL** → re-open the gate (these aren't non-blocking; they belong in `rejected_tasks` or as blockers, not here).
- **Codex HIGH** → severity=high (typically `bug` or high-priority `tech-debt`).
- **Codex MEDIUM** → severity=medium.
- **Codex LOW / "nit"** → severity=low.

Each finding entry should be self-contained (AC reference, file path, or wiring concern) so a future reader can act without reopening the UAT log.

**Why this is mandatory**: findings written only into the UAT log are invisible to backlog grooming and quietly disappear. "Accept with Conditions" must mean the conditions exist as tracked work, not as bullets in a log nobody re-reads.

If there are zero non-blocking findings, return an empty list and state explicitly in `results_document`: "No non-blocking findings — nothing to triage."

## Verdict rules

- **Accept** — Codex returns PASS / Accept; user approves all scenarios.
- **Accept with Conditions** — Codex returns PASS-with-concerns; user approves but requires the conditions be tracked (handled via `non_blocking_findings`).
- **Reject** — Codex returns FAIL on blocking findings; or user rejects specific scenarios.
- **Insufficient Evidence** — Codex failed to complete OR evidence file is incomplete/missing key artifacts. Stop — do not present to user; instead, return a summary of what's missing.

## Output contract summary

| Output | When set | Used by host for |
|--------|----------|------------------|
| `verdict` | Always | Determining workflow next-step |
| `uat_document` | When Step 1 ran | File write |
| `evidence_document` | Always | File write |
| `codex_assessment` | When Step 3 succeeded | File write (embedded in `results_document`) |
| `results_document` | When Step 6 ran | File write |
| `rejected_tasks` | When verdict is Reject | Routing each rejected task back to development |
| `non_blocking_findings` | Always (may be empty) | Routing each through `/triage` with severity |
| `approved_task_ids` | When verdict is Accept / Accept with Conditions | Marking tasks completed |

## Remember

- **Evidence collection is not analysis.** Don't editorialize.
- **Codex is the sole assessor.** Never skip.
- **No Claude opinion in the report.** The user makes the call from evidence + Codex assessment.
- **No auto-complete.** Tasks transition only after explicit user approval.
- **Every non-blocking finding gets triaged.** Otherwise it's invisible.
