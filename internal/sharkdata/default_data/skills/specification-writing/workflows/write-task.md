---
inputs:
  - feature_id: opaque feature identifier (string)
  - feature_prd_path: absolute path to the feature PRD markdown
  - feature_test_plan_path: absolute path to the feature-level test plan (e.g., `09-test-plan.md`)
  - feature_directory: absolute path to the feature's directory (where design docs and tasks live)
  - design_doc_paths: object mapping design slugs to absolute paths
      (architecture, database, api_spec, frontend, security_performance, performance, implementation_phases, test_plan, test_criteria) — values may be null if doc absent
  - wireframes_path: absolute path to `wireframes.md` (REQUIRED if any task touches frontend code; null otherwise)
  - prototype_path: absolute path to `prototype.md` (optional)
  - research_report_path: absolute path to validated unified research report (REQUIRED; STOP if missing)
  - feature_research_report_path: absolute path to feature research report (optional but strongly preferred for Brownfield Context)
  - interaction_map_path: absolute path to parent `<epic-id>-interaction-map.md` if present
  - tasks_directory: absolute path where task spec files should be written
  - has_frontend: bool — true if any task in this feature touches frontend code
  - existing_acceptance_criteria_tc_ids: list of TC-IDs declared in `feature_test_plan_path` (the canonical AC source)
outputs:
  - created_tasks: list of {task_slug, task_title, agent, order, depends_on, file_path, ac_tc_ids}
  - decisions_log: list of {decision, rationale, file_referenced}
  - dependencies_identified: list of {from_task, to_task, kind: "data" | "api" | "ux" | "infra"}
  - documentation_gaps: list of {missing_doc, impact_on_tasks, recommendation}
  - cross_feature_contracts: list of {interaction_id, task_slug, role: "produces" | "consumes", shape_source, contract_test_pointer}
  - import_graph_warnings: list of {symbol, deletion_task, still_referenced_by_task} — empty if no risky deletions
  - ac_quality_warnings: list of {task_slug, ac_id, anti_pattern} — should be empty before returning
---

# Workflow: Write Tasks (craft)

## Purpose

Generate agent-executable implementation tasks from existing technical design documentation. Tasks break down implementation into logical phases and components that specialized agents can execute independently.

## Core Responsibility

You will read comprehensive technical design documentation and create focused, agent-executable tasks. Your tasks must be **high-level directives that reference the design documents**, NOT detailed code tutorials.

## CRITICAL: You Create High-Level Directives, Not Code Tutorials

Think of yourself as a project manager creating work tickets for specialized teams. Each task should tell an agent WHAT to build and WHY, while referencing the detailed HOW from the design documents.

### NEVER WRITE IN TASKS:
- SQL statements, DDL, migrations, or database queries
- Python, TypeScript, JavaScript, or any programming language code
- Bash scripts, shell commands, or CLI instructions
- Configuration files (YAML, JSON, TOML, etc.)
- Step-by-step code implementation instructions
- Line-by-line coding tutorials
- Detailed implementation procedures

### ONLY WRITE IN TASKS:
- Clear goal and success criteria
- WHAT needs to be built (high-level requirements)
- WHY it's needed (business/technical rationale)
- References to design doc sections for details
- List of files that will need changes
- Integration points and dependencies
- Validation requirements (what to test, not how)
- Edge cases and performance considerations

**KEY PRINCIPLE**: Tasks are executive summaries with references, not implementation manuals.

## Hard Gates Before You Begin

Several inputs must be present before tasks can be written. The host should already have validated these, but you re-check defensively:

1. **Feature PRD** at `feature_prd_path` — readable and substantive.
2. **Feature test plan** at `feature_test_plan_path` — present AND complete. Scan for:
   - At least one test case with a TC-ID
   - A Caller-Path Contracts table or per-test-case caller-path blocks
   - An ISTQB technique application matrix
   - An ISO 25010 coverage matrix

   If the test plan is missing or any of these sections are absent, STOP and report which section is missing. Tasks derive their acceptance criteria from test-case TC-IDs; without a complete plan there is nothing to derive from.

3. **Research report** at `research_report_path` — required. If missing, STOP. Its Capability map prevents re-implementing established capabilities.

4. **Wireframes** — if `has_frontend=true` and `wireframes_path` is null, STOP and recommend the host run the feature-design workflow first. Do not silently generate frontend tasks without wireframes.

5. **Cross-feature interactions** — if the feature PRD has a "Cross-feature
   interactions" section, read `interaction_map_path` and verify every referenced
   I-## exists. Missing map rows are blockers.

## SIMPLE-tier mode

The host (feature/task_generation.md) determines the feature's complexity tier from the latest `COMPLEXITY:` decision note and tells you when a feature is SIMPLE. When it is, the following Hard Gates above are **waived** — do not STOP on their absence:

- **Gate 1 (Feature PRD / spec.md)** — waived. A SIMPLE feature does not require a full spec.md before tasks can be written.
- **Gate 2 (Feature test plan / test-plan.md)** — waived, including its TC-ID, Caller-Path Contracts, ISTQB matrix, and ISO 25010 matrix requirements (BUG-7: those matrices exist to size STANDARD/COMPLEX test surfaces and do not apply to XS/SIMPLE work).
- **Gate 3 (Research report)** — never waived. SIMPLE work still consumes its validated Capability map.

Gates 4 (Wireframes) and 5 (Cross-feature interactions) are **not** waived — they still apply if their trigger condition is met.

Inline replacement for the waived gates, per task file:

- **Acceptance Criteria** — write concrete, enumerable criteria directly in the task file instead of TC-ID references (there is no test-plan.md to reference). For code changes: specific pass/fail conditions. For doc-only tasks: verification steps (links resolve, lint/format passes, reviewer checklist).
- **Test Cases** — write the concrete test case(s) directly in the task file (what to run or check), instead of "See test-plan.md Section N".
- **Research evidence** — cite the research report and its Capability map in "Notes for Agent".

Everything else in this workflow (line count, no code blocks, cross-references not copies, dependency DAG, TDD structure) still applies unchanged — SIMPLE-tier mode only removes the requirement for upstream planning documents, not task-quality discipline.

## Your Process

### Step 0: Detect Available Documentation

Inspect `design_doc_paths`. For each missing doc, decide its impact and surface it in `documentation_gaps`. Present a summary to the user (or surface it via `documentation_gaps`):

```
Documentation Analysis:

Available: <list>
Missing:   <list>

Task Detail Level: HIGH | MEDIUM | LOW
Implications: <which task types can/cannot be generated>

Recommendation: <proceed | complete docs first | adapt scope>
```

Adjust task generation strategy:

- **PRD only** → high-level planning tasks, research tasks, design tasks
- **PRD + Architecture** → architecture implementation tasks, integration tasks
- **PRD + Architecture + Database** → add database schema tasks
- **PRD + Architecture + API** → add backend service tasks
- **PRD + Architecture + Frontend** → add UI component tasks
- **Full docs** → comprehensive implementation tasks

### Step 1: Analyze Available Design Documents

Read and understand all available design documents (skip missing ones):

1. **PRD** (REQUIRED) — feature requirements and goals
2. **Architecture** (if present) — system layers and integration points
3. **Database** (if present) — tables, relationships, data requirements
4. **API** (if present) — endpoints, contracts, business logic
5. **Frontend** (if present) — component hierarchy and state management
6. **Security/Performance** (if present) — critical security and performance requirements
7. **Implementation Phases** (if present) — planned phase breakdown
8. **Test Plan** (REQUIRED) — every task references which test cases from this plan it covers, enabling TDD at the task level

### Step 2: Validate Contract Consistency (CONDITIONAL)

**Only if API and Frontend design documents both exist.**

1. **Check API specification document** — codebase analysis section complete; DTOs fully defined with exact field names and types; Contract Synchronization Table shows matching frontend/backend expectations; contract testing requirements specified for both sides.

2. **Cross-reference with Frontend/Backend docs** — frontend should reference the same DTO names; backend services should use the same DTO names; data transformations documented on both sides.

3. **Flag missing information** — if contracts are incomplete or mismatched, note in tasks; if codebase analysis is missing, warn that parallel code paths might be created; if DTOs aren't synchronized, highlight the risk.

If API or Frontend docs are missing, skip this validation and note the impact in `documentation_gaps`.

### Step 2.5: Identify Cross-Feature Contracts

Before decomposing implementation work, extract any I-## rows from the feature
PRD's `Cross-feature interactions` section.

- Interfaces crossing outside this feature keep their I-## ID from the epic
  interaction map.
- Internal task-to-task contracts may use CONTRACT-### IDs.
- Do NOT invent new CONTRACT-### IDs for cross-feature wires.
- Each producing or consuming task must include an `Integration Contracts >
  Cross-feature` subsection with the I-## ID, role, shape source, and contract
  test pointer copied verbatim from the feature PRD.

Populate `cross_feature_contracts` as tasks are assigned.

### Step 3: Determine Task Scope and Sequencing

Create separate tasks for logical components. The structure adapts to available design documents.

For each task, decide:

- **Required**: title, agent type, scope summary, AC mapping to TC-IDs.
- **Recommended**: execution order (1, 2, 3...) and dependencies (`depends_on` task slugs).
- **Optional**: priority.

#### Standard sequence when all docs are present

1. **Contract Validation** — backend DTO + frontend interface implementation matching spec; contract validation tests on both sides. Always first when API + Frontend exist.
2. **Database Setup** — schema, migrations, RLS policies, indexes. Depends on contract.
3. **API Implementation** — backend services, endpoints, error handling. Depends on contract + database.
4. **Frontend Development** — UI components, API integration, validation. Depends on contract + API.
5. **Integration & Testing** — end-to-end integration and validation. Depends on frontend.
6. **Deployment & Monitoring** — deployment, monitoring, production validation.

#### Adapt to partial documentation

- **PRD + Architecture only** — Architecture Implementation, Define Detailed Design, Integration Planning.
- **PRD + Architecture + Database (no API/Frontend)** — Schema Implementation, Data Access Layer Design, API Design Task.
- **PRD + Architecture + Backend (no Frontend)** — Backend API Implementation, API Documentation, Frontend Design Task.
- **PRD + Security/Performance only (infrastructure/DevOps)** — Infrastructure Setup, Security Implementation, Performance Optimization, Deployment Pipeline.

**General principle**: generate tasks for components with specifications, create "design tasks" for missing specifications, adjust dependencies accordingly.

#### Dependency / order rules

- Lower order numbers execute first; same order can run in parallel.
- Use both `--order` and `--depends-on` for clarity. Order is the suggested sequence within the feature; dependencies are hard prerequisites.
- Typical pattern: database → API → frontend → tests → deployment.

### Step 4: Write the Task File

For each task, write a markdown file under `tasks_directory` with these sections:

- **Goal** — single, clear objective.
- **Success Criteria** — measurable checkpoints.
- **Acceptance Criteria as TC-ID references** — list the TC-IDs from the feature test plan that this task must satisfy. **Do NOT restate ACs in your own words** — that creates a drift surface between task AC, test-plan AC, and PRD AC. Format: `AC-T1: TC-005, TC-006, TC-007 (see <test-plan-path>)`. The TC entries already have Caller-Path Contracts prescribing the production entrypoint and forbidden mocks.
- **Implementation Guidance** — references to design docs, NOT code.
- **Validation Gates** — what to test, NOT how.
- **Context & Resources** — links to design doc sections.
- **Integration Contracts** — internal CONTRACT-### and cross-feature I-## rows
  this task produces or consumes.
- **Design References** (MANDATORY for frontend-touching tasks — see below).
- **Brownfield Context** (REQUIRED — see below).
- **Notes for Agent** — patterns, edge cases, considerations.

#### Design References (MANDATORY)

For every task that touches frontend code, you MUST first cite:

- `wireframes_path` (the canonical wireframes file)
- `prototype_path` (if present)

Then include any additional design references found in the PRD, epic, or related documents (mockups, Figma links, screenshots, UI specs). These references are **hard requirements** — code review and QA load the page in a browser and compare against them; implementations that don't match are rejected. See `../context/task-template.md` for format.

If `wireframes_path` is null and the task touches frontend code, STOP — do not silently generate frontend tasks without wireframes.

#### Integration Contracts (MANDATORY when applicable)

For any I-## this task produces or consumes, include:

```markdown
## Integration Contracts

### Cross-feature
- I-##: produces|consumes; shape source: <architecture.md#section>;
  contract test: <shared contract test pointer>
```

The shape source and contract test pointer must match the feature PRD exactly.
Producer and consumer tasks reference the same pointer; do not create twin tests.

#### Brownfield Context (REQUIRED)

Every task must include a `## Brownfield Context` section.

Use `../context/brownfield-context-rules.md` for the required section shape and source-priority rules.

If the research report is missing, create one research sub-task (order=0) before implementation tasks, with the goal of producing a research report that subsequent tasks can reference.

### Step 5: Validate Task File Quality (MANDATORY)

For each task file you wrote, check:

- **Line count**: file must be 50–100 lines (excluding frontmatter). If over 100, you are over-specifying — rewrite as a higher-level directive with design-doc references.
- **Zero implementation code blocks**: search for triple-backtick fences. The ONLY acceptable code fences in a task file are illustrative shell snippets that reference *workflow tooling* — NOT Go, Python, TypeScript, SQL, or any implementation language. Replace any implementation code with prose: "See [Design Doc — Section Name](../path#section)."
- **Cross-references, not copies**: if you find yourself writing interface definitions, struct fields, or method signatures, STOP. Instead: "Implement the X interface as specified in [Tech Spec — Section Y](../04-backend-design.md#section-y)."
- **No before/after code patterns**: do not include "Before:" and "After:" code blocks. Instead: "Refactor X to call service method Y. See tech spec Section Z for the service contract."

If any task fails these checks, rewrite it before returning.

### Step 6: Verify Task Set Cohesion

After all tasks are drafted:

1. **Test plan coverage**: every TC-ID in the feature test plan maps to at least one task; every task references the specific test cases it must satisfy; no orphaned TC-IDs.
2. **Dependency graph is acyclic**: trace the `depends_on` graph. If circular, refactor.
3. **Import-graph sanity check (MANDATORY for tasks that delete or rename existing symbols)**: for each task that deletes a class, function, module, or service, grep the codebase for the symbol's name and verify no other task in the feature still imports or references it.

   ```bash
   # For each symbol the task deletes:
   grep -rn "<DeletedSymbol>" --include="*.py" --include="*.ts" .
   # Cross-reference hits against the file list of OTHER tasks in this feature.
   ```

   If a downstream task in this feature still imports a symbol an earlier task deletes, either (a) reorder tasks so the import-removing task runs first, (b) split the deletion into "remove the call sites" + "remove the definition" across two tasks, or (c) move the deletion to a later task. Capture in `import_graph_warnings`. **Do not return a task list with a known import-time break.**

4. **Contract synchronization** (if applicable): API task references exact DTO specifications from design doc; frontend task references the exact same DTO specifications; contract validation tests included in both.
5. **Cross-feature interaction coverage**: every I-## declared by the feature PRD appears in at least one producing or consuming task, with the same shape source and contract test pointer.

Use `../context/task-set-cohesion-checklist.md` for the full checklist and failure conditions.

## Writing Acceptance Criteria (CRITICAL — this is where rejection loops are born or prevented)

The single strongest predictor of whether a task will sail through the workflow or spiral through repeated rejection rounds is **the style of its acceptance criteria**.

Use `../context/ac-writing-rules.md` for the full concrete-vs-open-ended examples, anti-pattern catalog, and rewrite guidance.

Hard rule:

- Acceptance criteria must be enumerable and verifiable in finite work.
- If an AC would cause an agent to keep finding "one more exploit", narrow the scope or enumerate the model explicitly.
- If the AC still reads as a vibe rather than a contract, add to `ac_quality_warnings` and rewrite it before returning.

### Verify-before-citing rules

A spec that names a thing is a *claim* that the thing exists. If the AC cites a column, file, fixture, or count that the codebase or feature PRD doesn't actually have, every downstream agent burns a round catching it — and the catches happen at QA or UAT, not at the spec stage.

**1. Schema and column names — grep before citing.**

When an AC names a database column, model field, table, or enum value, run:

```bash
grep -rn "<column_name>" --include="*.py" backend/db/models/ backend/models/  # adapt to project layout
# or, for a more thorough check:
grep -rn "<column_name>" --include="*.py" --include="*.sql" --include="*.prisma" .
```

If the column doesn't exist, either fix the AC to cite an existing column or add a database task whose deliverable creates that column. Never cite a column that doesn't exist yet without naming the task that creates it.

**2. No "byte-equivalent to prior" claims.**

Do not write ACs of the form *"PNG output byte-equivalent to the previous fitz-rendered image"*, *"checksum identical to v1"*, or *"diff-zero against the legacy implementation"* UNLESS a stored reference artifact is committed and cited by path. Different rasterizers produce different bytes; different parsers tokenize differently; different hashers... well, you get it. Equality across a library swap is almost always physically impossible.

If you find yourself reaching for "byte-equivalent", you almost certainly want **"deterministic"** or **"visually equivalent within ε"** instead, with an explicit tolerance and comparison method, OR commit reference bytes/snapshots and cite the file path.

**3. Numerics match the feature PRD verbatim.**

When an AC has a count or threshold (*"5 fixture PDFs"*, *"200ms p95"*, *"≥80% coverage"*), grep the feature PRD's measurable-outcomes / acceptance-criteria sections for the same number. If the task spec weakens it (*"at least 1 fixture"*) while the PRD said 5, code review and QA will accept the weaker version per their phase contract — and UAT will reject. The number in the task must equal the number in the PRD, character-for-character.

**Required check before finalizing each task:** for each AC, run the three verifications above. They are 30 seconds of grep apiece and prevent a 30-minute review-and-rework loop.

### When in doubt: scope, then enumerate

A task with AC "X is immutable" produces 6 rejection rounds (R1: `.append`, R2: dict mutation, R3: nested freeze, R4: subclass coercion, R5: subclass keys, R6: scalar subclass injection) before someone scopes it down to "attribute rebinding only" — which is what the AC should have done from day one.

The fix: the AC should have said *"X fields cannot be mutated by: (1) attribute rebinding, (2) `.append`/`.clear`/`.pop`/`.update` on collection fields, (3) item assignment on dict fields, (4) nested mutation of dict values via shared references, (5) coercion via mutable subclass instances. TC-T2-01..12 cover these cases."* Six rounds collapse to one.

## Task Quality Standards

Each task must meet:

1. **Focused Scope**: single, clear objective that can be completed independently.
2. **High-Level**: tells WHAT to build, not HOW to code it.
3. **Reference-Rich**: links to design docs instead of duplicating content.
4. **Testable**: clear success criteria and validation gates.
5. **Concise**: 50–100 lines maximum (excluding frontmatter).
6. **Agent-Appropriate**: assigned to the right specialized agent.
7. **Dependency-Aware**: clearly states what must complete first.
8. **Time-Bounded**: realistic effort estimate (2–12 hours typically).
9. **AC Quality**: every AC is concrete (file:line / contract / explicit I/O / enumerated set), not open-ended robustness.

## Common Mistakes to Avoid

- **Writing implementation tutorials** — tasks are not step-by-step coding guides.
- **Including code samples** — no SQL, Python, TypeScript, or any language code (only exception: design docs may have pseudocode).
- **Being too prescriptive** — trust the implementation agent's expertise; provide requirements and constraints, not micro-instructions.
- **Duplicating design doc content** — use references and links.
- **Creating overlapping tasks** — each task should have distinct, non-overlapping scope; clear handoff points.

## Handling Incomplete or Missing Documentation

### When Design Documents Are Missing

If design documents are missing entirely, present the gap analysis (already captured in `documentation_gaps`), wait for user confirmation, then adapt the task structure to match what's available. Generate:

- Implementation tasks for documented components.
- Design/specification tasks for undocumented components.
- Research tasks for uncertain areas.

### When Design Documents Are Incomplete or Unclear

1. Identify what's missing or ambiguous.
2. Make reasonable assumptions based on best practices.
3. Document assumptions in the task's "Notes for Agent" section.
4. Recommend updating the design docs with the clarification.
5. Consider creating a preliminary task to complete the design doc.

### Task Quality with Partial Documentation

Tasks generated from partial documentation will be:

- **Higher-level**: more strategic, less tactical.
- **Research-oriented**: may include investigation and design work.
- **Flexible**: allow implementation agents more decision-making authority.
- **Documentation-focused**: emphasize creating missing documentation.

This is acceptable. Not all features require full design documentation upfront.

## Final Validation

When complete:

1. **Validate dependencies form a logical execution sequence** — `dependencies_identified` is acyclic.
2. **Verify test plan coverage** — every TC-ID maps to at least one task; every task references its TC-IDs; no orphans.
3. **Confirm contract synchronization** (if applicable) — API and frontend tasks reference the same DTOs.
4. **Confirm import-graph sanity** — `import_graph_warnings` is empty.
5. **Confirm AC quality** — `ac_quality_warnings` is empty.
6. **Confirm cross-feature contracts** — `cross_feature_contracts` covers every
   I-## from the feature PRD and contains no rewritten IDs or pointers.

Your tasks are the execution plan that transforms design documentation into working code. They must be clear, focused, and actionable while trusting implementation agents to apply their expertise.
