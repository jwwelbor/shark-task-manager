# Wireframes (Feature-Level)

Produce `docs/plan/<epic-id>/<feature-id>/wireframes.md`: low-fidelity
wireframes for a specific feature's key screens and flows.

This is a feature-level artifact. It runs during feature refinement after
product-level design evidence reaches its exit point. It is not part of the
product-design artifact sequence.

## Prerequisites

Required:

- The active feature PRD path from workflow-provided feature metadata.
- D08 personas, usually `docs/product/D08-user-personas.md`.
- D09 journey maps, usually `docs/product/D09-journey-maps.md`.

Recommended:

- D11 friction points when this feature resolves a known journey break.
- Existing architecture or frontend-design documents when the feature extends
  an established interface.

Use the active feature PRD path and feature metadata supplied by the host. If
the PRD path or feature key is unavailable, ask the host for the feature key and
PRD path before continuing.

If D08 personas are not referenced in the PRD already, stop and ask the host to
connect the feature scope to the relevant persona evidence before wireframing.

## Scope

Read the PRD and extract:

- The user goal this feature addresses.
- The primary persona, using a D08 persona ID.
- The D09 journey stage this feature supports.
- The key screens and states needed.
- Acceptance criteria that constrain the design.

Wireframes cover exactly what is in PRD scope. Do not extend scope here.

## Screens to Cover

For each screen, document all relevant states:

- **Default** - loaded with no user action yet.
- **Populated** - loaded with representative real data.
- **Empty** - no data or no configured content yet.
- **Loading** - async operations in progress.
- **Error** - validation, network, permission, or system failure.
- **Success** - the user completes the task or operation.

Missing states are the most common wireframe gap. Check each screen against this
state list before saving.

## Wireframe Format

Use ASCII art or structured text. Represent:

- Content blocks with labeled placeholders, not final copy.
- Navigation elements.
- Form fields with labels and input types.
- Buttons and calls to action with labels.
- Annotations explaining non-obvious behavior.

Do not specify colors or final styling. This is low-fidelity by design.

Example:

```text
+--------------------------------------------------+
| [Header: App name]              [User menu v]    |
+--------------------------------------------------+
| <- Back    [Screen title]              [Action]  |
+--------------------------------------------------+
|                                                  |
|  [Primary content area]                          |
|  +--------------------------------------------+  |
|  | [Field label]                              |  |
|  | [Input ________________]                   |  |
|  | Helper text                                |  |
|  +--------------------------------------------+  |
|                                                  |
|                          [Cancel]  [Save ->]     |
+--------------------------------------------------+
```

## Responsive Considerations

For each screen, note:

- Mobile behavior below 768px: what collapses, stacks, hides, or changes order.
- Touch behavior that differs meaningfully from pointer behavior.
- Any content or control that requires a different density strategy on narrow
  screens.

Only document responsive behavior where it meaningfully differs from the
default layout.

## Quality Gate Before Saving

- [ ] All screens from the PRD scope are covered.
- [ ] All relevant states are documented for each screen.
- [ ] The user flow connecting screens is mapped.
- [ ] Primary persona uses a D08 persona ID.
- [ ] Journey stage uses a D09 stage name.
- [ ] Non-obvious design choices have rationale.
- [ ] No screen describes implementation internals; only what the user sees.

## Output Template

```markdown
# Wireframes: [Feature Name]

*Feature: [Feature key] - [PRD feature name]*
*Primary persona: [D08 ID - e.g., P1]*
*Journey stage: [D09 stage this feature supports]*
*Source PRD: [feature PRD path]*

## Screen Inventory

1. [Screen 1 name]
2. [Screen 2 name]

---

## Screen 1: [Name]

**Purpose:** [What the user accomplishes here]

### Default State

[ASCII wireframe]

### Interactions

1. [User action] -> [Result/navigation]
2. [User action] -> [Result/navigation]

### States

- **Populated:** [Description]
- **Empty:** [Description]
- **Loading:** [Description]
- **Error:** [Specific error and display]
- **Success:** [Description]

### Responsive Notes

- **Mobile:** [What changes below 768px]
- **Touch:** [Any touch-specific behavior]

### Accessibility

- Keyboard tab order: [describe]
- Screen reader: [key announcements]

### Rationale

[Non-obvious design decisions and why they serve the persona/journey]

---

## Screen 2: [Name]

[Same structure]

---

## User Flows

### Primary Flow: [Goal]

Screen 1 -> [action] -> Screen 2 -> [action] -> [success state]

### Error Recovery Flow

[What happens when the highest-risk failure occurs]

---

Version: 1.0 - YYYY-MM-DD - author: [name]

*Next: create prototype.md if interaction testing is needed; otherwise proceed to task writing.*
```

Save to `docs/plan/<epic-id>/<feature-id>/wireframes.md`.
