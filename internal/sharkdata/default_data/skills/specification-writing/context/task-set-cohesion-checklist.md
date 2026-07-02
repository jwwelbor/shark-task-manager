# Task Set Cohesion Checklist

Use this file when `workflows/write-task.md` needs the full post-draft validation checklist.

After drafting tasks, verify:

1. Every TC-ID in the feature test plan maps to at least one task.
2. Every task references the specific TC-IDs it must satisfy.
3. The dependency graph is acyclic.
4. For deletions or renames, import-graph checks show no downstream task still depends on the removed symbol.
5. API and frontend tasks reference the same DTO contracts when both exist.
6. Every cross-feature I-## interaction appears in at least one producing or consuming task with the same shape source and contract test pointer.

Do not return a task set with a known import-time break, orphaned test cases, or uncovered cross-feature contracts.
