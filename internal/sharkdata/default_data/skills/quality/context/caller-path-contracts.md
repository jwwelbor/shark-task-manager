# Caller-Path Contracts Reference

Use this file when Step 5.8 of `workflows/test-planning.md` needs the full contract definition.

For every test case, specify the production caller signature it must drive and the lowest allowed mock seam, so tests cannot mock above the place where production breaks.

## Required fields

| Field | What it specifies | Why it matters |
|---|---|---|
| **Caller-path entrypoint** | Exact production function or method to call, with the argument shape production uses | Forces the test to drive the real production call signature |
| **Lowest allowed mock seam** | Deepest layer where mocking is permitted | Prevents mocking above the bug |
| **Forbidden mocks** | Seams that must not be mocked because production-equivalent behavior is required | Makes the trap explicit |
| **Counter-factual** | What a buggy implementation would do that this test would catch | Proves the test would fail for the wrong implementation |

## Example block

```markdown
**Caller-Path Contract:**
- Entrypoint: `analyzer.analyze_page(upload_id=<UUID>, page_index=<int>, page_width_pts=<float>, page_height_pts=<float>)`
- Lowest allowed mock seam: `SourceImageRepository`
- Forbidden mocks: do not call `analyze_page(source_page_id=<x>)`
- Counter-factual: against an implementation that filters by `WHERE source_page_id IS NULL`, the query returns zero rows and the `image_count == 3` assertion fails
```

## Internal-only opt-out

For pure value objects, private data classes, or unit-level algorithmic functions that have no production caller above them in the diff, the test may declare:

`Caller-path entrypoint: internal — function under test is the production entrypoint`

That opt-out requires a one-line justification and should be treated as rare.
