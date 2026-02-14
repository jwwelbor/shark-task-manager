# Enhancement Notes for E07-F29

## Template Variables for Notes (Future Enhancement)

**Date**: 2026-02-13
**Type**: Enhancement Suggestion

### Proposal

Extend the template variable system to support **notes** as template variables, similar to how `{related_docs}` and `{related_tasks}` work.

### Suggested Syntax

1. **All notes**: `{related_notes}` - Include all notes associated with the entity
2. **Filtered by type**: `{notes:note-type}` - Include only notes of a specific type

### Examples

```markdown
# Template with filtered notes
Agent instruction: Review the implementation notes for context.

Implementation Notes:
{notes:implementation}

Design Decisions:
{notes:design}

All Related Notes:
{related_notes}
```

### Use Cases

1. **Orchestrator Instructions**: Include relevant notes in instruction templates so AI agents have full context
2. **Type-Filtered Context**: Filter notes by type (implementation, design, testing, etc.) to provide targeted context
3. **Documentation**: Automatically include related notes in generated documentation

### Implementation Approach

Follow the same pattern as E07-F29:
- Extend `TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders` functions
- Add helper function `formatNotesAsText(notes []Note, noteType string) string`
- Parse note type from placeholder syntax: `{notes:implementation}`
- Query notes by entity ID and optional type filter
- Format as text with note metadata (author, timestamp, type)

### Format Suggestion

CSV format may not be ideal for notes (unlike file paths). Consider:

```
--- Note (implementation, 2026-02-13, developer) ---
This is the note content here.

--- Note (design, 2026-02-12, architect) ---
Another note here.
```

Or simpler markdown format:

```markdown
## Implementation Notes

This is the note content here.

## Design Notes

Another note here.
```

### Dependencies

- Requires notes storage system (may already exist, see `shark notes search`)
- Note type taxonomy/schema
- Note-to-entity relationship (task_notes, feature_notes, epic_notes tables?)

### Priority

Could-Have for MVP, Should-Have for future releases once notes system is more mature.

---

*This enhancement was suggested during E07-F29 orchestration as a natural extension of the related_docs and related_tasks template variables.*
