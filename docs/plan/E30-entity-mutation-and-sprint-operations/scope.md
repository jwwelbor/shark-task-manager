# Scope Boundaries

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)

---

## In Scope

- Mutation APIs for the entity types already supported by Shark
- Updating routine fields such as title, description, priority, agent assignment, and execution order where applicable
- Note create/update/delete flows
- Dependency add/remove flows
- Explicit status transitions that preserve workflow validation and history
- Sprint stage/remove/ready actions and related planning interactions
- Viewer integration for inline editing and jump-back navigation

---

## Out Of Scope

- Replacing the existing CLI workflows
- Arbitrary schema editing or custom user-defined fields
- Bulk import/export tooling
- Authentication and authorization redesign
- Cross-project permissions or collaboration workflows
- Rebuilding the viewer in a new frontend framework
- Automatic mutation suggestions or AI-assisted edits
- Retroactive editing of historical status records

---

## Boundary Decisions

1. **Status changes remain workflow-driven**
   - The epic will not introduce a generic "set any field to any value" write path for status.
   - Reason: Shark status transitions carry audit and validation rules that must stay visible.

2. **Dependencies remain explicit relations**
   - Dependency changes will not be hidden inside a general entity patch.
   - Reason: dependency management affects workflow readiness and should be traceable.

3. **Sprint actions stay explicit**
   - Stage/remove/ready actions must be user-initiated and visible.
   - Reason: Sprint mode should remain read-first, with write actions clearly subordinate.

4. **Deep reporting enhancements are deferred**
   - The first release focuses on actionable planning and report visibility, not advanced forecasting or chart customization.
   - Reason: keep the epic decomposable and avoid overcommitting to an analytics rebuild.

---

## Future Considerations

- Inline rich-text note editing
- Bulk mutation tools for many items at once
- More advanced sprint forecast and comparison views
- Expanded mutation support for any entity types not already covered by the existing model
