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
  - rejected_tasks: list of {task_id, classification: "AC Violation" | "Spec Gap", defect_class, unmet_criteria, fix_required}
  - non_blocking_findings: list of {description, codex_severity, ac_or_location_ref} ready for triage routing
  - approved_task_ids: list of task IDs approved for completion (owner-ratified when owner review is on; derived from the Codex verdict otherwise)
user_invocable: false
version: 3.1.0
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

Codex is the **sole assessor**. Run `codex_command` and pass it the evidence file alongside the source artifact paths. The reviewer follows the assessment rubric in `references/redteam-rubric.md` — the enumeration discipline ("enumerate, don't iterate"), the five verification-check categories (wiring/reachability, contract consistency, AC enumeration, coverage gaps, standards), and the required output format (AC-assessment and E2E-wiring tables, verdict).

Whoever assembles `codex_command` MUST follow the **Codex invocation contract** below — the flags, timeout, standing guardrails, and re-verification scoping rules are load-bearing; each was learned from a real multi-round rejection spiral.

This step MUST complete before presenting anything to the user. If Codex fails after retries, the verdict is **"Insufficient Evidence"** — do not proceed to user review.

The reason Codex is the sole assessor: Claude reviewing Claude's implementation is a monoculture blind spot. The independent codex review with explicit enumeration discipline catches things both Claude and Claude-the-reviewer would miss. Per the user's standing instruction, codex red-team is mandatory at UAT and is never skipped.

#### Codex invocation contract (for whoever assembles `codex_command`)

**Flags:** `codex exec -s read-only -c model_reasoning_effort=high --skip-git-repo-check`. Omit `-m` — model support is account-dependent and a wrong `-m` hard-errors the run.

**Timeout:** at least 580 seconds. Five-minute budgets kill runs mid-investigation, and a retry after truncation burns a second full run — one long run beats two truncated ones. If it times out at the long budget, retry once.

**Standing guardrails — include in every prompt:**

- **Name the project's noise files as off-limits** ("Do NOT read uv.lock / package-lock.json / go.sum or other lockfiles") — reviewers have burned 5-minute budgets in lockfile rabbit holes.
- **Declare what the sandbox cannot do** ("Do NOT attempt to run tests — no venv/toolchain in the sandbox; verify statically") so codex doesn't spend budget discovering it.
- **Require a delimited verdict line**: "No matter how much investigation time remains, ALWAYS end with a final line `VERDICT: <verdict>`" — this guarantees a parseable verdict under timeout pressure.
- **Stay outcome-neutral.** Never anchor the reviewer ("third and hopefully final re-verification" is verdict-anchoring). State the round number and scope; never the hoped-for result.

**Re-verification rounds are NEVER fix-scoped.** A round ≥2 prompt always runs the full three-part procedure from `skills/quality/workflows/defect-class-sweep.md`'s "Full-class re-verification" section: (a) verify the named fixes, (b) re-run the full enumeration over the declared search scope, and (c) a full-rubric sanity pass over the feature surface. Narrow asks get narrow answers — every recorded rejection spiral happened because a re-review asked only "confirm finding N is fixed" and codex answered exactly the question asked. Make the broad ask the default so it doesn't depend on the orchestrator remembering.

#### Staged-integration gate integrity

Record the independent assessor verdict, the separate owner decision, its conditions, and the demonstrability disposition as distinct facts.
An owner `override-accept` / Accept with Conditions decision may be considered only for a complete, predeclared `contract-only` edge; it does not change the assessor verdict and does not make pending integration verified end-to-end delivery.

Missing live wiring, authentication, authorization, integrity, unsafe exposure, and an unmet current-feature acceptance criterion remain blocking even when a future owner is named or an owner selects `override-accept`.

The activation owner's UAT evidence must prove the real caller chain, shared-contract evidence, a production-path integration test, and a wiring-removal counterfactual. An internal activation obligation blocks epic completion until that closure is demonstrated. An external obligation may remain open only with a named future owner and a documented roadmap decision.

### Step 4 — Compile the report (evidence + Codex assessment, NO Claude opinion)

Build the UAT report containing:

1. **Evidence summary** — per scenario: spec quotes, code references, test output. Facts only, no Claude judgment.
2. **Codex assessment** — full Codex findings: per-criterion verdicts, CRITICAL/HIGH issues, wiring checks, contract verification, overall recommendation.
3. **Codex verdict** — Accept / Accept with Conditions / Reject / Insufficient Evidence.

**Do NOT add Claude's opinion, agreement/disagreement analysis, or recommendations.** The report presents evidence + Codex assessment for the human to review.

### Step 5 — Owner review (config-gated)

Whether a human ratifies the verdict before completion is a **project configuration decision**, not a skill default. Resolve the mode from the active project's `require_owner_approval` configuration field:

- **Owner review ON** — the field is `true` or lists this entity's workflow level. Do not ask for approval in-session; release the Codex verdict as the recommended outcome and let the workflow park the entity at its owner-approval step. The owner reviews the written report there and releases pass (complete) or fail (rework) via a status advance. The owner may override Codex in either direction — owner judgment is final.
- **Owner review OFF** — the field is absent, `false`, or does not list this level. The Codex verdict is final: do not stop to ask. Map the verdict to the outcome per the Verdict rules and populate `approved_task_ids` with every passing task. The report and results documents are the owner's asynchronous audit trail.

In both modes, the Step 4 report is written **before** any outcome is released — asynchronous owner review must always be possible. If the user interactively asks to review or override at any point, their judgment is final regardless of mode.

### Step 6 — Record results and update state

Write a session results document to `results_path` containing:

- Session metadata (date, feature, epic).
- Final verdict.
- Codex assessment summary with severity-ranked findings.
- User's per-scenario decisions.
- Approved task IDs.

**This skill never transitions state itself.** It returns `approved_task_ids`; the host workflow performs the actual transitions. Whether a human ratifies before completion is governed by the owner-review mode (Step 5), not by this skill.

### Step 7 — Route rejections (per-task classification)

When UAT rejects specific tasks, route each failing task individually. **Do not blanket-reject the whole feature** — passing tasks stay approved.

For each failing task:

1. **Classify the rejection reason**:
   - **AC Violation** — the task spec was clear; the implementation didn't meet it. The fix is implementation-only.
   - **Spec Gap** — the requirement was missing or ambiguous in the task spec. Needs refinement before re-implementation.

2. **State the defect class** — one line naming the *general* class of the defect, not the point instance (e.g. "schema required-list omits fields the code dereferences unconditionally", not "field X missing from schema Y"). This is mandatory: the point finding tells the developer what to fix; the class statement tells them what to *sweep for*. A rejection without a class statement produces a point fix and another rejection round on the next instance of the same class.

3. **Record the rejection in the task spec** (append a section to the task `.md` file):

   ```markdown
   ## UAT Rejection (<YYYY-MM-DD>)

   **Classification:** AC Violation | Spec Gap
   **Defect class:** <one-line general class statement>
   **Unmet criteria:** <AC-ID> — <quote the AC text>

   **What happened:** <brief description of the failure>

   **Fix required:**
   1. Apply the defect-class sweep procedure (`skills/quality/workflows/defect-class-sweep.md`) FIRST: enumerate every site in the touched module(s) matching the defect class above per its "Enumeration procedure" section, disposition each per that section (not just the cited instance), and list the swept sites in the completion note.
   2. <concrete step for the cited instance>
   3. <concrete step>

   **Full UAT results:** <results_path>
   ```

4. Return the rejection in `rejected_tasks` (including `defect_class`) so the host workflow can route the task back to development with the appropriate context (bug-fix flag, blocker note, status reset).

**Re-review mandate is class-scoped, never fix-scoped.** When the fixed work returns to UAT, the next codex round re-runs the full rubric over the touched surface plus the full-class re-verification from `skills/quality/workflows/defect-class-sweep.md` (see the Codex invocation contract in Step 3) — never "confirm finding N is fixed."

### Step 8 — Triage non-blocking findings (MANDATORY for every verdict)

Regardless of verdict — Accept, Accept with Conditions, Reject, or Insufficient Evidence — every Codex finding labeled **non-blocking** and every "Required Follow-ups" item MUST be returned in `non_blocking_findings` so the host can route them through `/triage`.

**Severity passthrough:** include each finding's Codex-assigned severity (CRITICAL / HIGH / MEDIUM / LOW) in the entry. Without severity, every triaged finding looks equally urgent and floods the backlog.

**Severity mapping**:
- **Codex CRITICAL / HIGH** → re-open the gate (these aren't non-blocking; they belong in `rejected_tasks` or as blockers, not here — see Verdict rules).
- **Codex MEDIUM** → severity=medium.
- **Codex LOW / "nit"** → severity=low.

Each finding entry should be self-contained (AC reference, file path, or wiring concern) so a future reader can act without reopening the UAT log.

**Why this is mandatory**: findings written only into the UAT log are invisible to backlog grooming and quietly disappear. "Accept with Conditions" must mean the conditions exist as tracked work, not as bullets in a log nobody re-reads.

If there are zero non-blocking findings, return an empty list and state explicitly in `results_document`: "No non-blocking findings — nothing to triage."

## Verdict rules

The severity→verdict mapping is **pinned** so verdicts are comparable across rounds and map deterministically onto workflow outcomes — a "concerns" verdict carrying a HIGH finding is a category error:

- **Any CRITICAL or HIGH finding → Reject.** No exceptions; HIGH findings never ride along as "conditions."
- **MEDIUM findings only → Accept with Conditions** (each condition tracked via `non_blocking_findings`).
- **LOW / nit findings only → Accept** with notes.

Full verdict set:

- **Accept** — Codex returns Accept (ratified by the owner at the owner-approval gate when owner review is on).
- **Accept with Conditions** — Codex found MEDIUM-severity issues only; the conditions become tracked work (handled via `non_blocking_findings`).
- **Reject** — any CRITICAL/HIGH or blocking finding; or user rejects specific scenarios.
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
- **No state changes from this skill.** The host workflow transitions; the `require_owner_approval` config decides whether a human ratifies first.
- **Every non-blocking finding gets triaged.** Otherwise it's invisible.
