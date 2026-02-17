# API Contracts: External Template Engine for Orchestrator Instructions

**Feature:** E07-F30
**Complexity Tier:** STANDARD
**Version:** 1.0
**Last Updated:** 2026-02-14

---

## Overview

This document defines the **internal Go API contracts** for the template engine feature. While this feature does not expose REST/HTTP endpoints, it defines critical Go interfaces and method signatures that serve as contracts between components.

**Key Contracts:**
1. **OrchestratorRenderer Interface** - Template rendering engine
2. **OrchestratorAction.PopulateTemplate()** - Integration point (modified)
3. **Template Function Map** - Custom template functions
4. **Placeholder Builder Functions** - Extended with complexity_tier

---

## 1. OrchestratorRenderer Interface

### Type Definition

```go
// Package: internal/templates

// OrchestratorRenderer handles rendering of orchestrator action templates
type OrchestratorRenderer struct {
    templates   *template.Template  // Precompiled template set
    templateDir string               // Base directory for templates
}
```

### Constructor

```go
// NewOrchestratorRenderer creates a new template renderer with precompiled templates
// Returns error if templates fail to parse (syntax errors)
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error)
```

**Parameters:**
- `templateDir` (string): Base directory containing template subdirectories (epic/, feature/, task/, partials/)

**Returns:**
- `*OrchestratorRenderer`: Initialized renderer with precompiled templates
- `error`: Parse error if any template has syntax errors

**Errors:**
- `"failed to parse templates: %w"` - Template syntax error (includes file/line info)
- `"template directory not found: %s"` - Directory doesn't exist

**Example:**
```go
renderer, err := NewOrchestratorRenderer("templates")
if err != nil {
    log.Fatalf("Failed to initialize template engine: %v", err)
}
```

---

### Singleton Accessor

```go
// GetOrchestratorEngine returns the global singleton template engine
// Initializes lazily on first call
func GetOrchestratorEngine() *OrchestratorRenderer
```

**Returns:**
- `*OrchestratorRenderer`: Global singleton instance

**Thread-Safety:** Uses `sync.Once` for safe concurrent access

**Panic Conditions:**
- Panics if template initialization fails (fail-fast at startup)

**Example:**
```go
engine := templates.GetOrchestratorEngine()
rendered, err := engine.Render("task/ready_for_development.tmpl", vars)
```

---

### Render Method

```go
// Render executes a template with the given variables
// Template name is relative to templateDir (e.g., "task/ready_for_development.tmpl")
func (r *OrchestratorRenderer) Render(templateName string, vars map[string]string) (string, error)
```

**Parameters:**
- `templateName` (string): Relative path to template (e.g., `"task/ready_for_development.tmpl"`)
- `vars` (map[string]string): Placeholder variables from `*PlaceholdersWithRelated()` functions

**Returns:**
- `string`: Rendered template output
- `error`: Execution error if template fails to render

**Errors:**
- `"template not found: %s"` - Template name not in precompiled set
- `"failed to execute template: %w"` - Execution error (missing variable, type mismatch, etc.)

**Performance:**
- Target: < 5ms per call (precompiled templates)
- Memory: ~100KB for 62 precompiled templates

**Example:**
```go
vars := map[string]string{
    "task_id":     "E07-F30-001",
    "title":       "Implement template engine",
    "file_path":   "docs/plan/E07-enhancements/E07-F30-template-engine/tasks/001.md",
    "related_docs": "prd.md, architecture.md",
    "complexity_tier": "STANDARD",
}

rendered, err := renderer.Render("task/ready_for_development.tmpl", vars)
if err != nil {
    log.Printf("Template rendering failed: %v", err)
    return ""
}
```

---

## 2. OrchestratorAction Contract (Modified)

### Type Definition (Unchanged)

```go
// Package: internal/config

type OrchestratorAction struct {
    Action              string   `json:"action"`
    AgentType           string   `json:"agent_type,omitempty"`
    Skills              []string `json:"skills,omitempty"`
    InstructionTemplate string   `json:"instruction_template"`  // Can be inline string OR .tmpl filename
}
```

### PopulateTemplate Method (Extended)

```go
// PopulateTemplate renders the instruction template with placeholder variables
// If InstructionTemplate ends with .tmpl, uses template engine; otherwise uses legacy string replacement
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string
```

**Parameters:**
- `vars` (map[string]string): Placeholder variables (from `*PlaceholdersWithRelated()` functions)

**Returns:**
- `string`: Rendered instruction string
- Empty string on error (graceful degradation)

**Behavior:**
1. **Detection:** If `InstructionTemplate` ends with `.tmpl`:
   - Use template engine (`templates.GetOrchestratorEngine().Render()`)
   - Log errors, return empty string on failure
2. **Otherwise:** Use legacy string replacement (`strings.NewReplacer()`)

**Backward Compatibility:**
- ✅ Existing inline templates continue working unchanged
- ✅ No breaking changes to method signature
- ✅ Gradual migration (mix of inline and .tmpl templates)

**Example:**
```go
// Template engine path
action := &OrchestratorAction{
    InstructionTemplate: "task/ready_for_development.tmpl",
}
instruction := action.PopulateTemplate(vars)  // Uses template engine

// Legacy path
action := &OrchestratorAction{
    InstructionTemplate: "Launch developer for {task_id}",
}
instruction := action.PopulateTemplate(vars)  // Uses string replacement
```

---

## 3. Template Function Map

### Custom Functions

```go
// Package: internal/templates

// orchestratorFuncs returns custom template functions available in all templates
func orchestratorFuncs() template.FuncMap
```

**Returns:**
- `template.FuncMap`: Map of function name → function implementation

**Available Functions:**

#### Conditionals

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `eq` | `func(a, b interface{}) bool` | Equality check | `{{if eq .tier "SIMPLE"}}...{{end}}` |
| `ne` | `func(a, b interface{}) bool` | Not equal | `{{if ne .status "completed"}}...{{end}}` |

#### String Helpers

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `join` | `func(items []string, sep string) string` | Join slice with separator | `{{join .skills ", "}}` |
| `isEmpty` | `func(s string) bool` | Check if string is empty/whitespace | `{{if isEmpty .related_docs}}None{{end}}` |
| `trim` | `func(s string) string` | Trim leading/trailing whitespace | `{{trim .title}}` |

#### Complexity Tier Helpers

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `isSimple` | `func(tier string) bool` | Check if tier is SIMPLE | `{{if isSimple .complexity_tier}}...{{end}}` |
| `isStandard` | `func(tier string) bool` | Check if tier is STANDARD | `{{if isStandard .complexity_tier}}...{{end}}` |
| `isComplex` | `func(tier string) bool` | Check if tier is COMPLEX | `{{if isComplex .complexity_tier}}...{{end}}` |

**Example Usage in Template:**
```go
{{if eq .complexity_tier "SIMPLE"}}
  Brief instructions
{{else if eq .complexity_tier "STANDARD"}}
  Focused instructions
{{else}}
  Comprehensive instructions
{{end}}

{{if isEmpty .related_docs}}
  No related documentation.
{{else}}
  Related docs: {{.related_docs}}
{{end}}
```

---

## 4. Placeholder Builder Functions (Extended)

### TaskPlaceholdersWithRelated (Extended)

```go
// Package: internal/config

// TaskPlaceholdersWithRelated builds placeholder map with related entities and complexity tier
func TaskPlaceholdersWithRelated(
    ctx context.Context,
    task *models.Task,
    docRepo repository.DocumentRepository,
    relRepo repository.TaskRelationshipRepository,
) map[string]string
```

**Parameters:**
- `ctx` (context.Context): Request context
- `task` (*models.Task): Task entity
- `docRepo` (DocumentRepository): Repository for related documents
- `relRepo` (TaskRelationshipRepository): Repository for related tasks

**Returns:**
- `map[string]string`: Placeholder variables including:
  - All fields from `TaskPlaceholders()` (task_id, title, status, epic, feature, etc.)
  - `related_docs` (comma-separated list)
  - `related_tasks` (comma-separated list)
  - **NEW:** `complexity_tier` (from task.Metadata["complexity_tier"])

**Example Output:**
```go
{
    "task_id": "E07-F30-001",
    "title": "Implement template engine",
    "status": "ready_for_development",
    "epic": "E07",
    "feature": "E07-F30",
    "agent_type": "developer",
    "file_path": "docs/plan/E07-enhancements/E07-F30-template-engine/tasks/001.md",
    "related_docs": "prd.md, architecture.md",
    "related_tasks": "E07-F29-003",
    "complexity_tier": "STANDARD",  // NEW
}
```

---

### FeaturePlaceholdersWithRelated (Extended)

```go
// FeaturePlaceholdersWithRelated builds placeholder map with related entities and complexity tier
func FeaturePlaceholdersWithRelated(
    ctx context.Context,
    feature *models.Feature,
    docRepo repository.DocumentRepository,
    relRepo repository.FeatureRelationshipRepository,
) map[string]string
```

**Returns:**
- `map[string]string`: Includes `complexity_tier` from `feature.Metadata["complexity_tier"]`

**Example Output:**
```go
{
    "id": "E07-F30",
    "title": "template engine",
    "epic": "E07",
    "file_path": "docs/plan/E07-enhancements/E07-F30-template-engine/feature.md",
    "related_docs": "research.md, proposal.md",
    "related_features": "E07-F29",
    "complexity_tier": "STANDARD",  // NEW
}
```

---

### EpicPlaceholdersWithRelated (Extended)

```go
// EpicPlaceholdersWithRelated builds placeholder map with related entities
func EpicPlaceholdersWithRelated(
    epic *models.Epic,
    docRepo repository.DocumentRepository,
    relRepo repository.EpicRelationshipRepository,
    ctx context.Context,
) map[string]string
```

**Returns:**
- `map[string]string`: Includes related epics and documents (no complexity_tier for epics)

**Note:** Epics do not have complexity_tier metadata (feature/task-level only).

---

## 5. Template File Contracts

### File Naming Convention

**Pattern:** `<entity>/<status>.tmpl`

**Examples:**
- `task/ready_for_development.tmpl`
- `feature/ready_for_refinement_ba.tmpl`
- `epic/ready_for_research.tmpl`

**Partial Templates:** Prefix with `_`
- `partials/_tdd_process.tmpl`
- `partials/_exit_gate.tmpl`

---

### Template Data Contract

**Available Variables:**

All templates receive `map[string]string` with these fields (depending on entity type):

**Task Templates:**
- `task_id` - Task key (e.g., "E07-F30-001")
- `title` - Task title
- `status` - Current status
- `epic` - Epic key
- `feature` - Feature key
- `agent_type` - Agent type (developer, architect, etc.)
- `priority` - Priority (1-10)
- `file_path` - Markdown file path
- `related_docs` - Comma-separated doc paths (may be empty)
- `related_tasks` - Comma-separated task keys (may be empty)
- `complexity_tier` - SIMPLE, STANDARD, COMPLEX (may be empty)

**Feature Templates:**
- `id` - Feature key (e.g., "E07-F30")
- `title` - Feature title
- `epic` - Epic key
- `file_path` - Feature markdown file path
- `related_docs` - Comma-separated doc paths (may be empty)
- `related_features` - Comma-separated feature keys (may be empty)
- `complexity_tier` - SIMPLE, STANDARD, COMPLEX (may be empty)

**Epic Templates:**
- `id` - Epic key (e.g., "E07")
- `title` - Epic title
- `file_path` - Epic markdown file path
- `related_docs` - Comma-separated doc paths (may be empty)
- `related_epics` - Comma-separated epic keys (may be empty)

---

### Template Syntax Contract

**Required Syntax:** Go `text/template` syntax

**Supported Features:**
- **Variables:** `{{.variable_name}}`
- **Conditionals:** `{{if .condition}}...{{else}}...{{end}}`
- **Equality:** `{{if eq .tier "SIMPLE"}}...{{end}}`
- **Partials:** `{{template "_partial_name" .}}`
- **Whitespace control:** `{{- if}}` (trim leading), `{{if -}}` (trim trailing)
- **Comments:** `{{/* comment */}}`

**Example:**
```go
Launch {{.agent_type}} for task {{.task_id}}: "{{.title}}".

READ:
(1) Task spec at {{.file_path}}
{{- if .related_docs}}
(2) Related docs: {{.related_docs}}
{{- end}}

{{if eq .complexity_tier "SIMPLE"}}
EXIT GATE:
- Core functionality works
{{else}}
EXIT GATE:
- All acceptance criteria pass
- Code review approved
{{end}}

Advance: shark task next-status {{.task_id}}.
```

---

## 6. Error Contract

### Template Syntax Errors (Startup)

**Timing:** Fail fast at application startup (precompilation)

**Error Format:**
```
failed to parse templates: template: task/ready_for_development.tmpl:12: unexpected "}" in command
```

**Includes:**
- File name
- Line number
- Syntax error description

**Behavior:** Application exits with error (cannot start with invalid templates)

---

### Template Execution Errors (Runtime)

**Timing:** During `PopulateTemplate()` call

**Error Handling:**
- Log error: `log.Printf("Template rendering failed for %s: %v", templateName, err)`
- Return empty string (graceful degradation)
- Workflow continues (orchestrator sees empty instruction)

**Common Causes:**
- Missing variable referenced in template
- Type mismatch (e.g., expecting slice, got string)
- Invalid partial template reference

---

### Missing Template Errors

**Timing:** During `Render()` call

**Error Format:**
```
template not found: task/nonexistent_template.tmpl
```

**Behavior:** Return error, caller logs and returns empty string

---

## 7. Performance Contract

### Template Rendering Performance

**Target:** < 5ms per `Render()` call

**Measurement Points:**
- Entry: `action.PopulateTemplate(vars)` called
- Exit: Rendered string returned

**Guaranteed:**
- Templates precompiled (no parse cost at render time)
- In-memory execution (no file I/O)
- Single allocation for output buffer

---

### Startup Performance

**Impact:** +10-50ms for template precompilation

**Mitigation:** Lazy initialization (engine created on first orchestrator action)

---

### Memory Footprint

**Impact:** ~100KB for 62 precompiled templates

**Measurement:** Heap allocation for `template.Template` set

---

## 8. Backward Compatibility Contract

### Guarantees

1. **Existing inline templates continue working indefinitely**
   - No deprecation warnings in MVP
   - No forced migration
   - Service layer contracts unchanged

2. **Zero breaking changes**
   - All existing tests pass
   - Method signatures unchanged
   - Config schema unchanged

3. **Gradual migration**
   - Mix of inline and `.tmpl` templates supported
   - No flag day required
   - Per-template opt-in

---

### Non-Breaking Changes

**What changed:**
- `OrchestratorAction.PopulateTemplate()` implementation (internal logic only)
- `*PlaceholdersWithRelated()` functions add `complexity_tier` field (additive, no impact)

**What stayed the same:**
- Method signatures
- Return types
- Error handling patterns
- Service layer integration points
- Config schema

---

## Summary

**Key Contracts:**
1. **OrchestratorRenderer:** Singleton template engine with `Render(templateName, vars)` method
2. **PopulateTemplate():** Extended to detect `.tmpl` suffix and route to engine
3. **Custom Functions:** 8 template functions (eq, ne, isEmpty, isSimple, etc.)
4. **Placeholder Maps:** Extended with `complexity_tier` field
5. **Performance:** < 5ms render time, ~100KB memory footprint
6. **Backward Compatibility:** 100% compatible, no breaking changes

**Next Steps:** Implement OrchestratorRenderer in Phase 1 (Foundation)
