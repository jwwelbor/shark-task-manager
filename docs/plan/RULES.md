# Documentation Standards for docs/plan/

These rules govern ALL documents created under `docs/plan/`. Every agent that reads or writes files in this directory MUST follow these rules. Violations will cause rework and wasted tokens.

---

## Rule 1: No Implementation Code in Any Document

**NO document under docs/plan/ may contain programming language code blocks.**

This means:
- No Go, Python, TypeScript, JavaScript, Rust, Java, SQL, Bash, or any other language
- No struct/class/interface definitions in code syntax
- No function/method implementations
- No code fences (triple backticks) except for `mermaid` diagrams and `bash` commands for shark CLI usage only

**How to describe technical concepts without code:**

| Instead of this... | Write this... |
|---|---|
| `type Foo struct { Bar string }` | "Foo is a data structure with a Bar field (string, required)" |
| `func (s *Svc) DoThing(ctx, key) error` | "DoThing accepts a context and key string, returns an error if the entity is not found" |
| Full method implementation | "DoThing validates the entity exists, retrieves its history, and returns results sorted chronologically. See tech spec Section 4.3 for the full contract." |
| Before/after code refactoring examples | "Refactor runFoo to call svc.DoThing() instead of accessing the repository directly" |

**Why this rule exists:** If a specification takes as many words to describe as it would to implement, then we should be implementing, not specifying. Design documents describe contracts, constraints, and decisions. Implementation agents decide HOW to code it.

---

## Rule 2: Document Size Limits

| Document Type | Target Lines | Maximum Lines |
|---|---|---|
| Epic PRD (each of 6 files) | 50-150 | 200 |
| Research Report | 100-200 | 300 |
| Architecture Doc (02-*.md) | 150-200 | 250 |
| Backend/Frontend Design (04-*.md, 05-*.md) | 150-200 | 250 |
| Test Plan (09-*.md) | 100-200 | 300 |
| Task Specification | 50-100 | 100 |

If a document exceeds its maximum, you are over-specifying. Refactor by:
- Removing code blocks (see Rule 1)
- Replacing duplicated content with cross-references (see Rule 3)
- Moving implementation details to a lower-level document

---

## Rule 3: Cross-Reference, Don't Duplicate

Each piece of information should exist in exactly ONE document. All other documents reference it.

**Information ownership:**
- **What the feature does and why** -> Feature PRD (feature.md / prd.md)
- **What the codebase looks like today** -> Research Report (research-report.md / 00-research-report.md)
- **What interfaces and contracts look like** -> Architecture/Design Docs (02-*.md, 04-*.md)
- **What to test and acceptance criteria** -> Test Plan (09-test-plan.md)
- **What to build (work items)** -> Task Specs (tasks/T-*.md)

**How to cross-reference:**
- Use relative markdown links: `See [Tech Spec - Section Name](../04-backend-design.md#section-name)`
- Name the specific section, don't just link the file
- Tasks should reference design docs for ALL technical details

**Red flags for duplication:**
- Same interface described in tech spec AND task file AND feature PRD
- Same list of files appearing in multiple documents
- Same acceptance criteria in both test plan and task spec (tasks should reference test plan, not copy it)

---

## Rule 4: Standard File Naming

Use these standard names. Do not create custom-named alternatives.

| Purpose | Standard Name | Bad Examples |
|---|---|---|
| Feature PRD | `prd.md` or `feature.md` | `feature-requirements.md` |
| Research Report | `00-research-report.md` or `research-report.md` | `analysis.md` |
| System Architecture | `02-architecture.md` | `tech-design.md`, `technical-spec.md` |
| Data Design | `03-data-design.md` | `database-schema.md` |
| Backend Design | `04-backend-design.md` | `api-spec.md`, `technical-spec.md` |
| Frontend Design | `05-frontend-design.md` | `ui-design.md` |
| Security Design | `06-security-design.md` | `security.md` |
| Performance Design | `07-performance-design.md` | `performance.md` |
| Implementation Phases | `08-implementation-phases.md` | `phases.md` |
| Test Plan | `09-test-plan.md` | `test-strategy.md` |

Creating a second file with a non-standard name (e.g., both `tech-design.md` and `technical-spec.md`) is NEVER acceptable. One document per concern.

---

## Rule 5: Tasks Are Directives, Not Tutorials

Task files (`tasks/T-*.md`) are work orders for implementation agents. They must:

1. **State WHAT to build** — goal, scope, affected files
2. **State WHY** — business/technical rationale, which requirement it satisfies
3. **Reference WHERE details live** — links to design doc sections
4. **Define DONE** — acceptance criteria (measurable, testable outcomes)
5. **NOT describe HOW to code it** — the implementation agent decides that

A good task reads like a clear work order. A bad task reads like a coding tutorial.

**Good task example (30 lines):**
> Goal: Add GetTaskHistory method to TaskService.
>
> This method enables migration of history.go and task_history.go from fat-controller pattern.
> See 04-backend-design.md Section 4.1 for the interface contract and repository requirements.
>
> Acceptance Criteria:
> - Method exists and compiles
> - Returns history in chronological order
> - Returns error for unknown task keys
> - make test passes with 0 failures

**Bad task example (200+ lines):**
> [contains full Go source code for the method, struct definitions, constructor changes, test implementations...]

---

## Rule 6: Register Documents in Shark

Every document created under docs/plan/ MUST be registered as a related document in the shark database:

```bash
# For feature-level docs:
shark related-docs add "<Doc Title>" <file-path> --feature=<feature-key>

# For task specs:
shark related-docs add "<Task Title>" <file-path> --task=<task-key>
```

Verify registration: `shark related-docs list --feature=<feature-key>`

If agents can't discover documents through shark, the orchestration system breaks down.

---

## Quick Self-Check

Before saving any document under docs/plan/, verify:

- [ ] Zero programming language code blocks (only mermaid and shark CLI allowed)
- [ ] Under the line limit for this document type
- [ ] No content duplicated from another document (cross-references instead)
- [ ] Using standard file naming
- [ ] Registered in shark as a related document
