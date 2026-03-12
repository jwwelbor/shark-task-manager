# Template System (v2)

File-based templating for AI agent instructions with external `.tmpl` files, template variables, and reusable partials.

## Overview

Shark v2 introduces file-based templates for agent instructions, replacing inline strings in `.sharkconfig.json`. This provides:

- **Externalized instructions**: Easy to edit without touching config
- **Template variables**: Dynamic content based on entity data
- **Reusable partials**: Shared template fragments
- **Version control friendly**: Diff and merge instructions easily
- **Multi-line support**: Natural formatting for complex instructions

## Template Directory Structure

```
templates/
├── epic/                          # Epic status templates
│   ├── ready_for_research.tmpl
│   ├── ready_for_feasibility_review_ba.tmpl
│   └── ready_for_feasibility_review_tech.tmpl
├── feature/                       # Feature status templates
│   ├── ready_for_research.tmpl
│   ├── ready_for_refinement_ba.tmpl
│   ├── ready_for_refinement_tech.tmpl
│   └── ready_for_test_planning.tmpl
├── task/                          # Task status templates
│   ├── ready_for_development.tmpl
│   ├── ready_for_code_review.tmpl
│   ├── ready_for_qa.tmpl
│   └── ready_for_refinement_ba.tmpl
└── partials/                      # Reusable template fragments
    ├── _read_section.tmpl
    ├── _tdd_process.tmpl
    └── _exit_gate.tmpl
```

## Template Configuration

Templates are referenced in `orchestrator_action` blocks:

```json
{
  "status_metadata": {
    "ready_for_development": {
      "description": "Ready for implementation",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation"],
        "instruction_template": "task/ready_for_development.tmpl"
      }
    }
  }
}
```

**Path resolution:**
- Relative to `templates/` directory in project root
- No leading slash
- Must include `.tmpl` extension

## Template Variables

Available variables depend on entity type.

### Common Variables (All Entity Types)

```go
{{.key}}               // Entity key (e.g., "E07", "E07-F01", "T-E07-F01-001", "B001", "CC-001")
{{.id}}                // Alias for key (backward compat)
{{.title}}             // Entity title
{{.status}}            // Current workflow status
{{.slug}}              // URL-friendly slug
{{.file_path}}         // Path to entity file
{{.created_at}}        // RFC3339 timestamp
{{.updated_at}}        // RFC3339 timestamp
{{.description}}       // Entity description (if set)
```

### Task Templates

```go
{{.task_key}}          // Alias for key when entity is a task
{{.epic_key}}          // Parent epic key (e.g., "E07") — parsed from task key
{{.feature_key}}       // Parent feature key (e.g., "E07-F01") — parsed from task key
{{.agent_type}}        // Agent type (e.g., "developer")
{{.priority}}          // Task priority (1-10)
{{.execution_order}}   // Execution order within feature
{{.depends_on}}        // Dependency string
{{.blocked_reason}}    // Blocked reason (if blocked)
{{.completion_notes}}  // Completion notes
{{.files_changed}}     // Files changed
{{.related_docs}}      // Related documentation paths (if set)
{{.related_tasks}}     // Related task keys (if set)
{{.complexity_tier}}   // Complexity tier from metadata
```

**Backward-compat aliases** (still work but canonical names preferred):
- `{{.task_id}}` → use `{{.key}}` or `{{.task_key}}`
- `{{.epic_id}}` → use `{{.epic_key}}` (now correctly resolves to parent epic key)
- `{{.feature_id}}` → use `{{.feature_key}}` (now correctly resolves to parent feature key)

### Feature Templates

```go
{{.epic_key}}          // Parent epic key (e.g., "E07") — parsed from feature key
{{.execution_order}}   // Execution order within epic
{{.related_docs}}      // Related documentation
{{.related_features}}  // Related feature keys
{{.complexity_tier}}   // Complexity tier from metadata
```

**Backward-compat aliases**: `{{.feature_id}}` → use `{{.key}}`, `{{.epic_id}}` → use `{{.epic_key}}`

### Epic Templates

```go
{{.priority}}          // Epic priority
{{.business_value}}    // Business value
{{.related_docs}}      // Related documentation
{{.related_epics}}     // Related epic keys
```

**Backward-compat alias**: `{{.epic_id}}` → use `{{.key}}`

### Bug Templates

```go
{{.severity}}              // Bug severity (critical, high, medium, low)
{{.linked_entity_type}}    // Linked entity type (epic, feature, task)
{{.linked_entity_key}}     // Linked entity key
```

### Change-Card Templates

```go
{{.priority}}          // Change-card priority
{{.requested_by}}      // Who requested the change
{{.assigned_to}}       // Who is assigned
{{.justification}}     // Change justification
{{.impact_analysis}}   // Impact analysis text
{{.rollback_plan}}     // Rollback plan text
```

## Template Syntax

### Basic Substitution

```go
Task {{.key}}: "{{.title}}"
```

**Rendered:**
```
Task T-E07-F01-001: "Implement token generation"
```

### Conditional Rendering

```go
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
```

**Rendered (with related_docs):**
```
(4) Related docs: docs/design/api-spec.md
```

**Rendered (without related_docs):**
```
(nothing)
```

### Whitespace Control

```go
{{- if .condition}}     // Remove leading whitespace
{{.value -}}            // Remove trailing whitespace
{{- .value -}}          // Remove both
```

### Nested Conditionals

```go
{{- if .related_tasks}}
({{if .related_docs}}3{{else}}2{{end}}) Related tasks: {{.related_tasks}}
{{- end}}
```

**Logic:**
- If `related_tasks` exists:
  - If `related_docs` exists: number as (3)
  - If `related_docs` missing: number as (2)

## Partial Templates

Reusable template fragments defined in `templates/partials/`.

### Defining Partials

Create file in `templates/partials/_partial_name.tmpl`:

```go
{{define "_read_section"}}READ:
(1) {{.primary_doc}}
{{- if .related_docs}}
(2) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}3{{else}}2{{end}}) Related tasks: {{.related_tasks}}
{{- end}}{{end}}
```

**Naming convention:**
- Prefix with `_` (e.g., `_read_section`)
- Use `{{define "_name"}}...{{end}}`

### Using Partials

Include in any template:

```go
LOAD: test-driven-development skill.

{{template "_read_section" .}}

{{template "_tdd_process" .}}

EXIT GATE:
- Implementation complete
- Tests pass
```

**Rendered:**
```
LOAD: test-driven-development skill.

READ:
(1) Task spec at docs/plan/.../T-E07-F01-001.md
(2) Related docs: docs/design/api-spec.md

TDD PROCESS:
1. Write failing test
2. Implement minimal code
3. Refactor

EXIT GATE:
- Implementation complete
- Tests pass
```

## Template Functions

Go template functions available:

### String Functions

#### `join`

Join string array with separator.

```go
{{join .skills ", "}}
```

**Example:**
```go
Input: .skills = ["test-driven-development", "implementation"]
Output: "test-driven-development, implementation"
```

#### `quote`

Quote each string in array.

```go
{{quote .related_tasks}}
```

**Example:**
```go
Input: .related_tasks = ["T-E07-F01-002", "T-E07-F01-003"]
Output: ["T-E07-F01-002", "T-E07-F01-003"]
```

#### `isEmpty`

Check if string is empty or whitespace.

```go
{{if isEmpty .description}}No description{{end}}
```

### Time Functions

#### `formatTime`

Format time as RFC3339.

```go
Created: {{formatTime .created_at}}
```

**Output:**
```
Created: 2026-02-16T10:30:00Z
```

#### `formatDate`

Format time as YYYY-MM-DD.

```go
Date: {{formatDate .created_at}}
```

**Output:**
```
Date: 2026-02-16
```

## Example Templates

### Simple Task Template

`templates/task/ready_for_development.tmpl`:

```go
Launch developer for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature architecture docs
(3) Feature test plan (09-test-plan.md)

IMPLEMENT using TDD:
1. Write failing test
2. Implement minimal code
3. Refactor

EXIT GATE:
- Tests pass
- Implementation matches spec

Advance: shark task next-status {{.task_id}}.
```

### Complex Feature Template

`templates/feature/ready_for_refinement_ba.tmpl`:

```go
Write feature PRD for {{.id}}: "{{.title}}".

LOAD: specification-writing workflow write-feature.md.

READ:
(1) Feature description at {{.file_path}}
(2) Parent epic PRD for context
(3) Complexity triage report (if exists)
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}

ADAPT TO COMPLEXITY TIER:
- SIMPLE: Lightweight PRD (1-2 pages, focus on acceptance criteria)
- STANDARD: Full PRD per write-feature.md (all sections)
- COMPLEX: Comprehensive PRD with detailed integration analysis

PRODUCE:
- prd.md in feature directory
- User stories with acceptance criteria
- API contracts (if applicable)
- Integration points

EXIT GATE varies by tier:
- SIMPLE: Acceptance criteria clear, implementation path obvious
- STANDARD: All PRD sections complete, no TBDs
- COMPLEX: Full PRD checklist, cross-feature dependencies mapped

Advance: shark feature next-status {{.id}}.
```

### Template with Partials

`templates/task/ready_for_code_review.tmpl`:

```go
Launch tech-lead for code review of task {{.task_id}}: "{{.title}}".

LOAD: quality skill.

{{template "_read_section" .}}

REVIEW CHECKLIST:
1. Implementation matches task spec
2. Tests cover requirements
3. Follows codebase conventions
4. No code smells

{{template "_exit_gate" .}}
```

## Template Best Practices

### Structure

**Use consistent formatting:**
```go
SECTION HEADER:
(1) First item
(2) Second item

Another section:
- Bullet point
- Another bullet
```

**Separate concerns:**
```go
LOAD: skills

READ: documentation

PROCESS: instructions

EXIT GATE: completion criteria

Advance: command
```

### Variable Usage

**Always quote string variables in prose:**
```go
Task "{{.title}}" ready.           // GOOD
Task {{.title}} ready.             // BAD (breaks with spaces in title)
```

**Use conditional rendering for optional fields:**
```go
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
```

**Don't assume all variables exist:**
```go
// BAD - crashes if related_docs is nil
Related: {{.related_docs}}

// GOOD - safe with conditional
{{- if .related_docs}}
Related: {{.related_docs}}
{{- end}}
```

### Whitespace

**Use whitespace control for clean output:**
```go
READ:
{{- if .related_docs}}
(1) Related docs: {{.related_docs}}
{{- end}}
```

**Without `-`:**
```

(1) Related docs: foo.md

```

**With `-`:**
```
(1) Related docs: foo.md
```

### Partials

**Extract repeated patterns:**
```go
// BAD - repeated in every template
READ:
(1) Task spec at {{.file_path}}
(2) Feature docs
(3) Epic context

// GOOD - defined once in partial
{{template "_read_section" .}}
```

**Name partials descriptively:**
```go
_read_section.tmpl          // GOOD
_tdd_process.tmpl           // GOOD
_common.tmpl                // BAD (vague)
_utils.tmpl                 // BAD (vague)
```

## Template Validation

### Syntax Validation

Test template syntax:

```bash
# Render template with test data
go run internal/templates/validate.go task/ready_for_development.tmpl

# Or use shark command (if available)
shark template validate task/ready_for_development.tmpl
```

### Common Errors

**Undefined variable:**
```
template: task:5:10: executing "task" at <.bad_var>: can't evaluate field bad_var
```

**Fix:** Remove undefined variable or add to TemplateData

**Unclosed template:**
```
template: task:12: unclosed action
```

**Fix:** Add closing `{{end}}` or `}}`

**Missing partial:**
```
template: no template "_missing_partial" associated with template "task"
```

**Fix:** Create partial or remove template call

## Migration from Inline Strings

### Before (v1 - Inline)

```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "Launch developer for task {task_id}. LOAD: TDD. READ: (1) Task spec at {file_path}. IMPLEMENT using TDD. EXIT GATE: Tests pass. Advance: shark task next-status {task_id}."
    }
  }
}
```

**Problems:**
- Hard to read (one long line)
- Hard to edit (JSON escaping)
- No multi-line support
- No reusable components

### After (v2 - File-Based)

**In config:**
```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "task/ready_for_development.tmpl"
    }
  }
}
```

**In templates/task/ready_for_development.tmpl:**
```go
Launch developer for task {{.task_id}}.

LOAD: test-driven-development.

READ:
(1) Task spec at {{.file_path}}

IMPLEMENT using TDD:
1. Write test
2. Implement
3. Refactor

EXIT GATE:
- Tests pass

Advance: shark task next-status {{.task_id}}.
```

**Benefits:**
- Natural formatting
- Easy to edit
- Version control friendly
- Reusable partials
- Template functions

## Template Library

### Task Templates

| Template | Agent | Purpose |
|----------|-------|---------|
| `ready_for_refinement_ba.tmpl` | business-analyst | Requirements specification |
| `ready_for_refinement_tech.tmpl` | architect | Technical design |
| `ready_for_development.tmpl` | developer | Implementation |
| `ready_for_code_review.tmpl` | tech-lead | Code review |
| `ready_for_qa.tmpl` | qa | Testing |

### Feature Templates

| Template | Agent | Purpose |
|----------|-------|---------|
| `ready_for_research.tmpl` | researcher | Codebase research |
| `ready_for_refinement_ba.tmpl` | business-analyst | Feature PRD |
| `ready_for_refinement_tech.tmpl` | architect | Architecture design |
| `ready_for_test_planning.tmpl` | qa | Test strategy |

### Epic Templates

| Template | Agent | Purpose |
|----------|-------|---------|
| `ready_for_research.tmpl` | researcher | Market/tech research |
| `ready_for_feasibility_review_ba.tmpl` | business-analyst | Business feasibility |
| `ready_for_feasibility_review_tech.tmpl` | architect | Technical feasibility |

### Partials

| Partial | Purpose |
|---------|---------|
| `_read_section.tmpl` | Standard READ section with optional related docs/tasks |
| `_tdd_process.tmpl` | TDD process instructions |
| `_exit_gate.tmpl` | Standard exit gate format |

## Advanced Usage

### Dynamic Numbering

Adjust numbering based on present fields:

```go
READ:
(1) Task spec at {{.file_path}}
{{- if .related_docs}}
(2) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}3{{else}}2{{end}}) Related tasks: {{.related_tasks}}
{{- end}}
```

### Multi-Path Instructions

Different instructions based on complexity:

```go
{{- if eq .complexity_tier "SIMPLE"}}
SIMPLE tier: Create 1-3 focused tasks.
{{- else if eq .complexity_tier "STANDARD"}}
STANDARD tier: Create 3-8 tasks with dependencies.
{{- else}}
COMPLEX tier: Create comprehensive task breakdown.
{{- end}}
```

### Template Composition

Build complex templates from partials:

```go
{{template "_header" .}}
{{template "_load_skills" .}}
{{template "_read_section" .}}
{{template "_process" .}}
{{template "_exit_gate" .}}
{{template "_footer" .}}
```

## Troubleshooting

### Template Not Found

**Error:**
```
failed to load template: template not found: task/ready_for_development.tmpl
```

**Causes:**
1. File doesn't exist at `templates/task/ready_for_development.tmpl`
2. Wrong path in config (e.g., missing `.tmpl` extension)
3. Wrong working directory

**Solution:**
```bash
# Verify file exists
ls templates/task/ready_for_development.tmpl

# Check path in config
cat .sharkconfig.json | jq '.status_metadata.ready_for_development.orchestrator_action.instruction_template'
```

### Variable Not Rendering

**Error:**
```
template: task:5: executing "task" at <.task_id>: can't evaluate field task_id
```

**Cause:** Variable not passed to template

**Solution:** Check `TemplateData` struct includes field

### Partial Not Found

**Error:**
```
template: no template "_read_section" associated with template "task"
```

**Cause:** Partial not loaded or not defined

**Solution:**
```bash
# Verify partial exists
ls templates/partials/_read_section.tmpl

# Check partial defines itself
grep 'define "_read_section"' templates/partials/_read_section.tmpl
```

## Related Documentation

- **[configuration.md](configuration.md)** - Config file reference
- **[workflow-configuration.md](workflow-configuration.md)** - Orchestrator actions
- **Go Templates**: https://pkg.go.dev/text/template
