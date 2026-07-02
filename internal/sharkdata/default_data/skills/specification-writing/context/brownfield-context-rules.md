# Brownfield Context Rules

Use this file when `workflows/write-task.md` needs the full Brownfield Context contract for generated tasks.

Every task must include a `## Brownfield Context` section with:

## Existing Files to Read Before Implementing

- Relevant files and why they matter
- Adjacent code that must not be broken

## Patterns to Follow

- Existing pattern names with file references
- Naming conventions already established

## Integration Surface

- Existing components this task connects to
- Existing tests that must keep passing

## Scope Boundary

- Files or components out of scope
- Public interfaces that must not change

## Source priority

Build Brownfield Context from, in order:

1. Prior-art report
2. Feature research report
3. Feature architecture docs
4. Existing codebase patterns

If the prior-art report marks a capability as REUSE, the task must say not to re-implement it.
