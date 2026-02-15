# Architecture Design: External Template Engine for Orchestrator Instructions

**Feature:** E07-F30
**Complexity Tier:** STANDARD
**Version:** 1.0
**Last Updated:** 2026-02-14

---

## Executive Summary

This document defines the technical architecture for externalizing orchestrator instruction templates from `.sharkconfig.json` to standalone `.tmpl` files with Go's `text/template` engine. The design achieves 100% backward compatibility through filename-based detection (`.tmpl` suffix triggers template engine, otherwise uses legacy string replacement), leverages proven patterns from existing `internal/templates/renderer.go`, and requires zero schema changes. Integration occurs at a single point (`OrchestratorAction.PopulateTemplate()`), enabling gradual migration of 62 templates across 3 phases while maintaining existing workflows.

**Key Architectural Decisions:**
1. **Backward Compatible Detection**: `.tmpl` suffix triggers template engine, otherwise legacy string replacement
2. **Reuse Existing Patterns**: Follow proven `text/template` pattern from `internal/templates/renderer.go`
3. **Single Integration Point**: Extend `OrchestratorAction.PopulateTemplate()` method only
4. **No Schema Changes**: Existing `instruction_template` field works for both inline strings and filenames
5. **Singleton Engine**: Global precompiled template engine for performance

---

## System Overview

### Component Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CLI Commands Layer                                │
│  (task next-status, feature next-status, epic next-status)          │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Service Layer                                     │
│  - EpicService.resolveAction()                                       │
│  - FeatureService.resolveAction()                                    │
│  - DisplayService.ResolveEpicAction()/ResolveFeatureAction()         │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                Config Layer (Integration Point)                      │
│  OrchestratorAction.PopulateTemplate(vars map[string]string)         │
│      │                                                               │
│      ├──> Detection: Does InstructionTemplate end with .tmpl?        │
│      │                                                               │
│      ├──> YES: Call OrchestratorRenderer.Render(template, vars)     │
│      │          └─> New external template path                      │
│      │                                                               │
│      └──> NO: Use legacy string replacement                         │
│                └─> Existing inline template path                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                ┌───────────────┴───────────────┐
                ▼                               ▼
┌──────────────────────────────┐  ┌────────────────────────────────┐
│  New Template Engine          │  │  Legacy String Replacement    │
│                               │  │                               │
│ OrchestratorRenderer          │  │ strings.NewReplacer()         │
│  - Load .tmpl file            │  │  - Replace {key} placeholders │
│  - Parse with text/template   │  │                               │
│  - Execute with vars          │  │                               │
│  - Return rendered string     │  │                               │
└──────────────┬────────────────┘  └────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Template File System                               │
│                                                                       │
│  templates/                                                           │
│  ├── epic/                                                            │
│  │   ├── ready_for_research.tmpl                                     │
│  │   ├── ready_for_refinement_ba.tmpl                                │
│  │   └── ready_for_feasibility_review_tech.tmpl                      │
│  ├── feature/                                                         │
│  │   ├── ready_for_research.tmpl                                     │
│  │   ├── ready_for_triage.tmpl                                       │
│  │   ├── ready_for_refinement_ba.tmpl                                │
│  │   ├── ready_for_refinement_tech.tmpl                              │
│  │   └── ready_for_test_planning.tmpl                                │
│  ├── task/                                                            │
│  │   ├── ready_for_development.tmpl                                  │
│  │   ├── ready_for_code_review.tmpl                                  │
│  │   ├── ready_for_qa.tmpl                                           │
│  │   ├── ready_for_refinement_ba.tmpl                                │
│  │   └── ready_for_refinement_tech.tmpl                              │
│  └── partials/                                                        │
│      ├── _tdd_process.tmpl                                           │
│      ├── _exit_gate.tmpl                                             │
│      └── _read_section.tmpl                                          │
└──────────────────────────────────────────────────────────────────────┘
```

### Data Flow

**Template Resolution Flow:**

1. **Service Layer** calls `resolveAction(ctx, entity, status)` → builds placeholder map via `*PlaceholdersWithRelated()`
2. **Config Layer** calls `action.PopulateTemplate(vars)` → **SINGLE INTEGRATION POINT**
3. **Detection Logic**:
   - If `InstructionTemplate` ends with `.tmpl` → route to template engine
   - Otherwise → route to legacy string replacement
4. **Template Engine Path**:
   - Load `.tmpl` file from `templates/` directory
   - Parse with Go `text/template` and custom functions
   - Execute template with placeholder data
   - Return rendered instruction string
5. **Legacy Path**:
   - Build `{key}` → `value` replacements
   - Use `strings.NewReplacer()` to replace inline
   - Return populated instruction string
6. **Result**: `PopulatedAction` with rendered instruction string

---

## Key Architectural Decisions

### Decision 1: Backward Compatible Detection via Filename Suffix

**Decision:** If `instruction_template` ends with `.tmpl`, use template engine; otherwise use legacy string replacement.

**Rationale:**
- Zero breaking changes - existing inline templates continue working unchanged
- No config schema changes required
- Clear intent - `.tmpl` extension explicitly signals template file
- Simple detection logic - single `strings.HasSuffix()` check
- Enables gradual migration - convert templates at any pace

**Alternatives Considered:**
- ❌ New config field `use_template_engine: true` - requires schema change, migration effort
- ❌ Auto-detect by checking file existence - ambiguous, error-prone
- ❌ Flag day migration - high risk, no gradual rollout

**Example:**
```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "task/ready_for_development.tmpl"  // Uses template engine
    }
  },
  "ready_for_approval": {
    "orchestrator_action": {
      "instruction_template": "Launch developer for {task_id}..."  // Uses legacy
    }
  }
}
```

---

### Decision 2: Reuse Existing Template Infrastructure Pattern

**Decision:** Create `OrchestratorRenderer` following the proven pattern from `internal/templates/renderer.go`.

**Rationale:**
- `Renderer` already uses Go `text/template` successfully for task markdown files
- Proven custom function pattern (`templateFuncs()`) works well
- Template loading strategy (embedded + filesystem fallback) is battle-tested
- Consistent architecture across codebase
- Reduces implementation risk by reusing known-good patterns

**Existing Pattern (from `renderer.go`):**
```go
type Renderer struct {
    loader *Loader
}

func (r *Renderer) Render(agentType string, data TemplateData) (string, error) {
    tmplContent, err := r.loader.LoadTemplate(agentType)
    tmpl, err := template.New("task").Funcs(templateFuncs()).Parse(tmplContent)
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }
    return buf.String(), nil
}
```

**New Pattern (for orchestrator templates):**
```go
type OrchestratorRenderer struct {
    templates *template.Template  // Precompiled templates
    templateDir string
}

func (r *OrchestratorRenderer) Render(templateName string, vars map[string]string) (string, error) {
    tmpl := r.templates.Lookup(templateName)
    if tmpl == nil {
        return "", fmt.Errorf("template not found: %s", templateName)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }
    return buf.String(), nil
}
```

---

### Decision 3: Single Integration Point at Config Layer

**Decision:** Extend only `OrchestratorAction.PopulateTemplate()` method; all other code remains unchanged.

**Rationale:**
- Minimizes code changes - single method modification
- Clear separation of concerns - detection logic isolated
- Service layer unchanged - maintains existing contracts
- Easy to test - single method to mock/test
- Easy to rollback - single point of change

**Integration Point (orchestrator_action.go):**
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    // Detection: if template ends with .tmpl, use template engine
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        engine := templates.GetOrchestratorEngine()
        rendered, err := engine.Render(oa.InstructionTemplate, vars)
        if err != nil {
            // Log error, return empty (graceful degradation)
            log.Printf("Template rendering error: %v", err)
            return ""
        }
        return rendered
    }

    // Otherwise, use legacy string replacement (UNCHANGED)
    if len(vars) == 0 {
        return oa.InstructionTemplate
    }

    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }
    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

**Call Sites (UNCHANGED):**
- `EpicService.resolveAction()` - line 230
- `FeatureService.resolveAction()` - line 233
- `DisplayService.ResolveEpicAction()` - display_service.go:294
- `DisplayService.ResolveFeatureAction()` - similar pattern

---

### Decision 4: Precompiled Template Singleton

**Decision:** Initialize template engine as singleton at startup, precompile all `.tmpl` files.

**Rationale:**
- **Performance**: Compile once, execute many times (< 5ms per render)
- **Fail Fast**: Syntax errors discovered at startup, not runtime
- **Memory Efficient**: Single template set in memory
- **Thread-Safe**: `sync.Once` ensures single initialization

**Implementation:**
```go
// internal/templates/orchestrator_engine.go

var (
    engineOnce     sync.Once
    engineInstance *OrchestratorRenderer
    engineError    error
)

func GetOrchestratorEngine() *OrchestratorRenderer {
    engineOnce.Do(func() {
        templateDir := "templates"  // Configurable via .sharkconfig.json
        engineInstance, engineError = NewOrchestratorRenderer(templateDir)
        if engineError != nil {
            log.Fatalf("Failed to initialize template engine: %v", engineError)
        }
    })
    return engineInstance
}

func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
    // Precompile all .tmpl files in templateDir
    tmpl, err := template.New("orchestrator").
        Funcs(orchestratorFuncs()).
        ParseGlob(filepath.Join(templateDir, "*/*.tmpl"))

    if err != nil {
        return nil, fmt.Errorf("failed to parse templates: %w", err)
    }

    return &OrchestratorRenderer{
        templates:   tmpl,
        templateDir: templateDir,
    }, nil
}
```

**Alternative Considered:**
- ❌ Parse templates on-demand - slower (5-50ms per render), runtime errors
- ❌ Per-request parsing - 100x slower, no caching benefits

---

### Decision 5: No Schema Changes Required

**Decision:** Reuse existing `instruction_template` field for both inline strings and file references.

**Rationale:**
- Zero migration effort for config file
- Backward compatible with existing configs
- Simple to understand - single field, dual purpose
- No version migration code needed

**Existing Schema (UNCHANGED):**
```go
type OrchestratorAction struct {
    Action              string   `json:"action"`
    AgentType           string   `json:"agent_type,omitempty"`
    Skills              []string `json:"skills,omitempty"`
    InstructionTemplate string   `json:"instruction_template"`  // Works for both!
}
```

**Optional Config Addition:**
```go
type Config struct {
    TemplateDirectory string `json:"template_directory,omitempty"`  // Default: "templates"
    // ... existing fields unchanged
}
```

---

## Component Design

### 1. OrchestratorRenderer (NEW)

**File:** `internal/templates/orchestrator_renderer.go`

**Responsibilities:**
- Load and precompile `.tmpl` files from `templates/` directory
- Execute templates with placeholder data
- Provide custom template functions (conditionals, string helpers)
- Handle partial template includes

**Interface:**
```go
type OrchestratorRenderer struct {
    templates   *template.Template  // Precompiled template set
    templateDir string               // Base directory for templates
}

func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error)
func GetOrchestratorEngine() *OrchestratorRenderer  // Singleton accessor
func (r *OrchestratorRenderer) Render(templateName string, vars map[string]string) (string, error)
```

**Custom Functions:**
```go
func orchestratorFuncs() template.FuncMap {
    return template.FuncMap{
        // Conditionals (for tier-specific output)
        "eq":      func(a, b interface{}) bool { return a == b },
        "ne":      func(a, b interface{}) bool { return a != b },

        // String helpers
        "join":    strings.Join,
        "isEmpty": func(s string) bool { return strings.TrimSpace(s) == "" },
        "trim":    strings.TrimSpace,

        // Complexity tier helpers (convenience)
        "isSimple":   func(tier string) bool { return tier == "SIMPLE" },
        "isStandard": func(tier string) bool { return tier == "STANDARD" },
        "isComplex":  func(tier string) bool { return tier == "COMPLEX" },
    }
}
```

---

### 2. Template Directory Structure (NEW)

**Location:** `templates/` at project root

**Naming Conventions:**
- Entity-based directories: `epic/`, `feature/`, `task/`
- Status-based filenames: `ready_for_<phase>.tmpl`
- Partial prefix: `_partial_name.tmpl` (e.g., `_tdd_process.tmpl`)

**Directory Layout:**
```
templates/
├── epic/
│   ├── ready_for_research.tmpl
│   ├── ready_for_refinement_ba.tmpl
│   └── ready_for_feasibility_review_tech.tmpl
├── feature/
│   ├── ready_for_research.tmpl
│   ├── ready_for_triage.tmpl
│   ├── ready_for_refinement_ba.tmpl
│   ├── ready_for_refinement_tech.tmpl
│   └── ready_for_test_planning.tmpl
├── task/
│   ├── ready_for_development.tmpl
│   ├── ready_for_code_review.tmpl
│   ├── ready_for_qa.tmpl
│   ├── ready_for_refinement_ba.tmpl
│   └── ready_for_refinement_tech.tmpl
└── partials/
    ├── _tdd_process.tmpl
    ├── _exit_gate.tmpl
    └── _read_section.tmpl
```

---

### 3. Template Placeholder Extension

**File:** `internal/config/template_helpers.go` (EXTEND)

**Changes:** Add `complexity_tier` to placeholder maps (minimal extension)

**Before:**
```go
func TaskPlaceholdersWithRelated(ctx, task, docRepo, relRepo) map[string]string {
    placeholders := TaskPlaceholders(task)
    // Add related_docs, related_tasks, etc.
    return placeholders
}
```

**After:**
```go
func TaskPlaceholdersWithRelated(ctx, task, docRepo, relRepo) map[string]string {
    placeholders := TaskPlaceholders(task)
    // Add related_docs, related_tasks, etc. (unchanged)

    // NEW: Add complexity_tier from metadata
    if task.Metadata != nil {
        if tier, ok := task.Metadata["complexity_tier"].(string); ok {
            placeholders["complexity_tier"] = tier
        }
    }

    return placeholders
}
```

---

## Integration Points

### 1. Service Layer (NO CHANGES)

**Files:**
- `internal/services/epic_service.go` (lines 202-238)
- `internal/services/feature_service.go` (lines 210-238)
- `internal/services/display_service.go` (line 294)

**Pattern (UNCHANGED):**
```go
func (s *EpicService) resolveAction(ctx context.Context, epic *models.Epic, status string) *config.PopulatedAction {
    // ... get workflow config ...

    // Build placeholders (unchanged)
    var vars map[string]string
    if s.docRepo != nil && s.relRepo != nil {
        vars = config.EpicPlaceholdersWithRelated(epic, s.docRepo, s.relRepo, ctx)
    } else {
        vars = config.EpicPlaceholders(epic)
    }

    // INTEGRATION POINT: Populate template (method extended internally)
    instruction := action.PopulateTemplate(vars)

    return &config.PopulatedAction{
        Action:      action.Action,
        AgentType:   action.AgentType,
        Skills:      action.Skills,
        Instruction: instruction,
    }
}
```

---

### 2. Config Layer (SINGLE CHANGE)

**File:** `internal/config/orchestrator_action.go`

**Method to Extend:** `OrchestratorAction.PopulateTemplate()` (line 140)

**Before:**
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    if len(vars) == 0 {
        return oa.InstructionTemplate
    }

    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }

    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

**After:**
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    // NEW: If template ends with .tmpl, use template engine
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        engine := templates.GetOrchestratorEngine()
        rendered, err := engine.Render(oa.InstructionTemplate, vars)
        if err != nil {
            // Log error, return empty (graceful degradation)
            log.Printf("Template rendering failed for %s: %v", oa.InstructionTemplate, err)
            return ""
        }
        return rendered
    }

    // UNCHANGED: Legacy string replacement
    if len(vars) == 0 {
        return oa.InstructionTemplate
    }

    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }

    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

---

## Template Examples

### Example 1: Task Development Template with Conditionals

**File:** `templates/task/ready_for_development.tmpl`

```go
Launch developer with test-driven-development skill for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md)
(3) Feature architecture docs
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}5{{else}}4{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

{{template "_tdd_process" .}}

EXIT GATE:
- All test cases pass
- Implementation matches spec{{if .related_docs}} and related docs{{end}}
- Code follows conventions

Advance: shark task next-status {{.task_id}}.
```

**Features:**
- ✅ Conditionals hide empty sections (`{{if .related_docs}}`)
- ✅ Smart numbering adapts (`{{if .related_docs}}5{{else}}4{{end}}`)
- ✅ Partial template include (`{{template "_tdd_process" .}}`)
- ✅ Whitespace control (`{{- if}}` trims leading whitespace)

---

### Example 2: Feature BA Refinement with Complexity Tier Scaling

**File:** `templates/feature/ready_for_refinement_ba.tmpl`

```go
Write feature PRD for {{.id}}: "{{.title}}".

LOAD: specification-writing workflow write-feature-prd.md.

READ:
(1) Triage report or research report
(2) Parent epic PRD
(3) Feature description at {{.file_path}}
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}

{{if eq .complexity_tier "SIMPLE"}}
PRODUCE: 1-page PRD summary
- Goal (2-3 sentences)
- Key user story (single primary story)
- Core acceptance criteria (3-5 testable criteria)

EXIT GATE:
- Core AC testable
- Aligns with epic requirements
{{else if eq .complexity_tier "STANDARD"}}
PRODUCE: Focused PRD (2-3 pages)
- Goal
- Key personas (2-3 primary personas)
- User stories (MoSCoW prioritization)
- Functional requirements (core only)
- Acceptance criteria (testable, unambiguous)
- Out of scope

EXIT GATE:
- MoSCoW complete
- AC testable and unambiguous
- No vague language
- Related features referenced
{{else}}
PRODUCE: Full 6-file PRD per write-feature-prd.md:
- epic.md (comprehensive goal, background, stakeholders)
- personas.md (all relevant personas with context)
- user-journeys.md (end-to-end journey maps)
- requirements.md (functional + non-functional)
- success-metrics.md (measurable outcomes)
- scope.md (in scope, out of scope, assumptions)

EXIT GATE:
- All 6 files created and cross-referenced
- Write-feature-prd.md checklist passes
- No vague language or TBDs
- Related features fully integrated
{{end}}

Advance: shark feature next-status {{.id}}.
```

**Features:**
- ✅ Complexity tier branching (`{{if eq .complexity_tier "SIMPLE"}}`)
- ✅ Three distinct output depths (SIMPLE, STANDARD, COMPLEX)
- ✅ Different EXIT GATE criteria per tier
- ✅ Readable multiline formatting

---

### Example 3: Shared Partial Template

**File:** `templates/partials/_tdd_process.tmpl`

```go
{{define "_tdd_process"}}
TDD PROCESS:
1. Write failing test first (red)
2. Implement minimum code to pass (green)
3. Refactor while keeping tests green
4. Commit when test suite passes
{{end}}
```

**Usage in task templates:**
```go
{{template "_tdd_process" .}}
```

**Benefits:**
- Change once, updates all templates using it
- Consistent TDD instructions across 5+ task templates
- Reduces duplication from ~80 lines to 1 line per template

---

## Error Handling

### 1. Template Syntax Errors (Fail Fast)

**Strategy:** Precompile all templates at startup, fail immediately with helpful errors.

```go
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
    tmpl, err := template.New("orchestrator").
        Funcs(orchestratorFuncs()).
        ParseGlob(filepath.Join(templateDir, "*/*.tmpl"))

    if err != nil {
        // Syntax error - fail fast with file/line info
        return nil, fmt.Errorf("template syntax error: %w", err)
    }

    return &OrchestratorRenderer{templates: tmpl}, nil
}
```

**Behavior:**
- ❌ Syntax errors → `shark` fails to start with error message
- ✅ Forces fix before production
- ✅ No runtime surprises

---

### 2. Template Execution Errors (Graceful Degradation)

**Strategy:** Log error, return empty string (don't break workflow).

```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        engine := templates.GetOrchestratorEngine()
        rendered, err := engine.Render(oa.InstructionTemplate, vars)
        if err != nil {
            // Log error but don't panic (graceful degradation)
            log.Printf("Template rendering failed for %s: %v", oa.InstructionTemplate, err)
            return ""  // Empty instruction is better than crash
        }
        return rendered
    }
    // ... legacy path ...
}
```

**Behavior:**
- ⚠️ Execution errors (missing variable, etc.) → log warning, return empty
- ✅ Workflow continues (orchestrator sees empty instruction)
- ✅ Visible in logs for debugging

---

### 3. Missing Template Files (Fail Fast)

**Strategy:** Precompilation catches missing templates at startup.

```go
func (r *OrchestratorRenderer) Render(templateName string, vars map[string]string) (string, error) {
    tmpl := r.templates.Lookup(templateName)
    if tmpl == nil {
        return "", fmt.Errorf("template not found: %s", templateName)
    }
    // ... execute ...
}
```

**Behavior:**
- ❌ Missing template → error at first use
- ✅ Clear error message with template name
- ⚠️ Note: Template validation command (Phase 3) will catch this proactively

---

## Security Considerations

### 1. Template Injection Prevention

**Risk:** Malicious placeholders could execute arbitrary code.

**Mitigation:** Use `text/template` (not `html/template` which auto-escapes), but placeholders come from trusted sources only:
- Database fields (epic/feature/task titles, keys)
- Config-driven metadata (complexity tier, agent type)
- Relationship data (related docs/tasks keys)

**No User Input:** Placeholders are NOT user-provided; they're generated internally from validated database records.

---

### 2. File System Access

**Risk:** Template engine could read arbitrary files.

**Mitigation:**
- Templates loaded only from `templates/` directory (configurable)
- ParseGlob restricted to `*.tmpl` files in subdirectories
- No dynamic template paths from user input
- Template directory configurable via `.sharkconfig.json` (admin-controlled)

---

### 3. Template Complexity Limits

**Risk:** Infinite loops or deeply nested templates could cause DoS.

**Mitigation (Phase 3 - Optional):**
- Template linting to enforce max depth/complexity
- Execution timeout (future enhancement)
- For now: Manual code review of templates

---

## Performance Characteristics

### 1. Template Rendering Performance

**Target:** < 5ms per `PopulateTemplate()` call

**Approach:**
- Precompiled templates (parse once at startup)
- In-memory template cache
- No file I/O during rendering
- Simple data structures (map[string]string)

**Benchmark (Expected):**
- Legacy string replacement: ~0.1ms (baseline)
- Template engine (precompiled): ~1-5ms (acceptable overhead)

---

### 2. Startup Time Impact

**Impact:** +10-50ms for template precompilation (62 templates)

**Mitigation:**
- Lazy initialization (only load engine when first orchestrator action triggered)
- Singleton pattern ensures single initialization
- Parallel template parsing (Go's ParseGlob is concurrent)

**Trade-off:** Slower startup (once) vs. faster rendering (every action) - acceptable.

---

### 3. Memory Footprint

**Impact:** ~100KB for 62 precompiled templates

**Mitigation:** Negligible compared to database and Go runtime (~10MB)

---

## Testing Strategy

### 1. Unit Tests

**File:** `internal/templates/orchestrator_renderer_test.go`

**Coverage:**
- Custom function tests (eq, ne, isEmpty, isSimple, etc.)
- Template rendering with mock data
- Error handling (missing template, syntax error)
- Partial template includes
- Conditional logic (if/else branches)

**Example:**
```go
func TestOrchestratorRenderer_Conditionals(t *testing.T) {
    renderer := setupTestRenderer(t)

    tests := []struct {
        name     string
        template string
        vars     map[string]string
        want     string
    }{
        {
            name:     "hide empty related_docs",
            template: "{{if .related_docs}}Related: {{.related_docs}}{{end}}",
            vars:     map[string]string{"related_docs": ""},
            want:     "",
        },
        {
            name:     "show populated related_docs",
            template: "{{if .related_docs}}Related: {{.related_docs}}{{end}}",
            vars:     map[string]string{"related_docs": "doc1.md, doc2.md"},
            want:     "Related: doc1.md, doc2.md",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := renderer.RenderString(tt.template, tt.vars)
            if err != nil {
                t.Fatalf("Render() error = %v", err)
            }
            if got != tt.want {
                t.Errorf("Render() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

---

### 2. Integration Tests

**File:** `internal/config/orchestrator_action_test.go`

**Coverage:**
- Backward compatibility (inline templates still work)
- `.tmpl` detection triggers engine
- Service layer integration (resolveAction)
- Error propagation (template errors don't break workflows)

**Example:**
```go
func TestOrchestratorAction_PopulateTemplate_BackwardCompatibility(t *testing.T) {
    tests := []struct {
        name     string
        template string
        vars     map[string]string
        want     string
    }{
        {
            name:     "inline template uses legacy",
            template: "Launch {agent_type} for {task_id}",
            vars:     map[string]string{"agent_type": "developer", "task_id": "E07-F01-001"},
            want:     "Launch developer for E07-F01-001",
        },
        {
            name:     ".tmpl template uses engine",
            template: "task/test_template.tmpl",
            vars:     map[string]string{"task_id": "E07-F01-001"},
            want:     "Task: E07-F01-001",  // Rendered from test template
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            action := &OrchestratorAction{InstructionTemplate: tt.template}
            got := action.PopulateTemplate(tt.vars)
            if got != tt.want {
                t.Errorf("PopulateTemplate() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

---

### 3. Regression Tests

**Goal:** Ensure converted templates render identically to inline versions.

**Approach:**
1. Capture output of all 62 inline templates with test data
2. Convert templates to external files
3. Render external templates with same test data
4. Assert outputs match character-for-character

**File:** `internal/config/orchestrator_regression_test.go`

---

## Migration Path

### Phase 1: Foundation (2-3 days)

**Deliverable:** Engine ready, 100% backward compatible

**Tasks:**
1. Create `internal/templates/orchestrator_renderer.go`
2. Implement custom template functions
3. Add `.tmpl` detection to `OrchestratorAction.PopulateTemplate()`
4. Create `templates/` directory structure
5. Write unit tests (90%+ coverage)
6. Add singleton initialization

**Exit Criteria:**
- All existing tests pass (no regressions)
- New template rendering tests pass
- Startup succeeds with no templates (graceful)

---

### Phase 2: High-Value Templates (3-4 days)

**Deliverable:** 12 most complex templates externalized

**Templates to Convert:**
- **Task execution (5):** ready_for_development, ready_for_code_review, ready_for_qa, ready_for_refinement_ba, ready_for_refinement_tech
- **Feature planning (4):** ready_for_research, ready_for_refinement_ba, ready_for_refinement_tech, ready_for_test_planning
- **Epic strategic (3):** ready_for_research, ready_for_feasibility_review_ba, ready_for_feasibility_review_tech

**Partials to Create:**
- `_tdd_process.tmpl` (5 task templates use it)
- `_exit_gate.tmpl` (all templates use it)
- `_read_section.tmpl` (12+ templates use it)

**Update `.sharkconfig.json`:**
```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "instruction_template": "task/ready_for_development.tmpl"  // Changed
      }
    }
  }
}
```

**Exit Criteria:**
- 12 templates render identically to inline versions (regression test)
- `.sharkconfig.json` references `.tmpl` files
- Partials work correctly (change propagates)

---

### Phase 3: Full Migration (4-5 days) - OUT OF SCOPE FOR MVP

**Deliverable:** All 62 templates externalized

**Tasks:**
1. Convert remaining 50 templates
2. Expand partial library to 6-10 partials
3. Add `shark config validate --templates` command
4. Create optional migration tool: `shark config migrate-templates`
5. Update documentation with template authoring guide
6. Add deprecation notice for inline templates (warning, not error)

**Exit Criteria:**
- All 62 templates externalized
- Full workflow regression tests pass
- Documentation complete

---

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking existing inline templates | High | Low | Filename detection (`.tmpl` suffix) maintains backward compatibility; extensive regression testing |
| Template syntax errors at runtime | Medium | Medium | Precompile templates at startup, fail fast with line numbers; validation command in Phase 3 |
| Performance degradation | Low | Low | Precompiled templates cached in memory; benchmark < 5ms per render; startup cost amortized |
| Template complexity explosion | Medium | Medium | Template linting in Phase 4; code review for early templates sets standards |
| Partial template name conflicts | Low | Low | Use `_` prefix convention; namespace by entity type (task/, feature/, epic/) |
| Missing `.tmpl` files break workflows | Medium | Low | Fail fast at startup if referenced template missing; clear error messages |

---

## Summary of Design Decisions

### What Changes
1. **New component:** `internal/templates/orchestrator_renderer.go` (template engine)
2. **New directory:** `templates/` with entity-based subdirectories
3. **Modified method:** `OrchestratorAction.PopulateTemplate()` (add detection logic)
4. **Extended function:** `*PlaceholdersWithRelated()` (add complexity_tier field)

### What Stays the Same
1. **Service layer:** No changes to `resolveAction()` methods
2. **Config schema:** `instruction_template` field unchanged
3. **Legacy support:** Inline templates continue working indefinitely
4. **Database:** No schema changes, no migrations
5. **CLI commands:** No changes to any commands

### Backward Compatibility Guarantees
- ✅ All existing inline templates continue working unchanged
- ✅ Zero breaking changes to workflows
- ✅ Gradual migration at user's pace (no flag day)
- ✅ Service layer contracts unchanged
- ✅ Config schema unchanged

---

**Architecture Review Status:** Ready for Approval
**Next Steps:** Create implementation tasks for Phase 1 (Foundation)
