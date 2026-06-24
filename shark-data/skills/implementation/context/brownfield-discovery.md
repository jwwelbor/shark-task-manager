# Brownfield Discovery: Pre-Implementation Workflow for Existing Codebases

## When to Use This

Use this workflow **before** any `implement-*.md` workflow when working in an established codebase — one with existing services, conventions, domain models, and history. If you're adding a feature, fixing a bug, or making a change in a system that already has code serving users, start here.

Skip this only for truly greenfield work (new project, empty directory).

## The Governing Principle

The cost of reading, searching, and planning is low. The cost of breaking existing behavior or building a second version of something that already exists is high.

**Default assumption:** the capability already exists somewhere in the codebase. Creating something new is the exception, and must be justified.

---

## Step 1: Understand the Existing System

Read the code around the area you plan to change. Trace the relevant data flow and control flow through nearby modules. Find examples of similar features or patterns already used in the codebase.

Follow the existing conventions, abstractions, naming, and architectural decisions — even when they seem imperfect. In an established system, consistency is usually more valuable than local optimization.

**Output:** List the modules, patterns, and conventions relevant to the change.

## Step 2: Understand the Domain

Identify the business concepts involved in the change. Learn how this codebase defines and models them: what they are called, how they relate to each other, and where their boundaries are drawn.

Before introducing anything new, confirm whether the concept already exists in another form. Misunderstanding the domain leads to duplicate or conflicting models.

**Output:** List the domain entities involved and where they live in the codebase.

## Step 3: Search for Existing Services and Capabilities

Assume the capability already exists somewhere in the codebase until you can prove otherwise. Search broadly across:

- Service layers and shared modules
- Utilities and registries
- Adjacent domains and integration points

**Search using business entity names, not just technical verbs.**

For example, if the task involves pricing, do not only search for `calculatePrice`. Also search for terms like `Price`, `Rate`, `Discount`, `LineItem`, `Quote`, or other domain-specific entities.

Do not create a new service, helper, or abstraction unless you have verified that an existing one cannot be reused or extended. Duplicating existing behavior in a brownfield system creates long-term divergence and maintenance cost.

**Output:** Existing capabilities that can be reused or extended, with file paths.

## Step 4: Read the Tests

Review the tests for the modules and flows you expect to touch. Use them to understand:

- Intended behavior
- Important edge cases
- Implicit contracts

If coverage is missing in the area of change, call that out explicitly and write characterization tests first to lock in existing behavior before modifying anything.

**Output:** Summary of test coverage and any gaps that need characterization tests.

## Step 5: Check the History

Review git blame, commit history, and related pull requests for the files you plan to modify. Understand why the code looks the way it does.

Seemingly odd logic may exist because of a prior bug, performance constraint, integration quirk, or edge case. Learn the reason before changing the implementation.

**Output:** Any non-obvious constraints or decisions discovered in the history.

## Step 6: Identify the Blast Radius

Map dependencies and downstream effects before making changes. Check:

- All call sites
- Serialized structures and API boundaries
- Background jobs and event consumers
- Configuration dependencies
- External integrations

Assume usage may exist in non-obvious places. Be explicit about what might break.

**Output:** List of downstream dependencies and potential breakage points.

---

## Implementation Gate

**Do not begin writing code until all six steps are complete.**

Before proceeding to an `implement-*.md` workflow, provide:

1. A brief summary of what you learned from each step
2. The specific existing modules/services/tests you plan to reuse or extend
3. The expected blast radius
4. The implementation approach (where the change lives, what's reused, what's new and why)

If any step cannot be completed, stop and say exactly what is missing and why implementation would be risky without it.

Once the gate is passed, proceed to the appropriate implementation workflow:
- `../workflows/implement-backend.md` — for service/business logic
- `../workflows/implement-api.md` — for HTTP endpoints
- `../workflows/implement-frontend.md` — for UI components
- `../workflows/implement-database.md` — for schema changes
- `../workflows/implement-tests.md` — for test-only work

Those workflows handle planning, safety nets (validation gates), and TDD — this workflow handles the discovery that must come first.
