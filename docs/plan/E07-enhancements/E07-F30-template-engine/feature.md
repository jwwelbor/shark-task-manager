# E07-F30: External Template Engine for Orchestrator Instructions

**Feature Key**: E07-F30-template-engine

---

## Problem

Orchestrator instruction templates are currently embedded as JSON strings in `.sharkconfig.json`. This approach has hit critical pain points:

1. **Unreadable**: Templates are 200-500+ character single-line JSON strings with no formatting
2. **Unmaintainable**: 62 instruction templates across 3 workflows, all embedded in one massive JSON file
3. **No conditionals**: Can't hide empty sections (e.g., "Related docs: " when no docs exist)
4. **Copy-paste hell**: Common patterns (TDD process, READ sections, EXIT GATE) duplicated across 18+ templates
5. **JSON escaping nightmare**: Quotes, braces, newlines all need escaping
6. **Terrible git diffs**: One-word change = entire 500-char line changed
7. **No validation**: Typos in placeholders only discovered at runtime
8. **Growing complexity**: Just added complexity tiers + 4 relational variables ({related_docs}, {related_tasks}, {related_features}, {related_epics}), templates getting worse

---

## Solution

**Move instruction templates to external files and use Go's stdlib `text/template` engine.**

### Architecture

```
shark-task-manager/
├── .sharkconfig.json              # Lightweight config, references templates
├── templates/
│   ├── epic/
│   │   ├── ready_for_refinement.tmpl
│   │   ├── ready_for_research.tmpl
│   │   └── ...
│   ├── feature/
│   │   ├── ready_for_scope_validation.tmpl
│   │   ├── ready_for_triage.tmpl
│   │   └── ...
│   ├── task/
│   │   ├── ready_for_development.tmpl
│   │   ├── ready_for_code_review.tmpl
│   │   └── ...
│   └── partials/
│       ├── _tdd_process.tmpl
│       ├── _exit_gate.tmpl
│       └── _read_section.tmpl
```

### Example Template

**File**: `templates/task/ready_for_development.tmpl`

```go
Launch developer with test-driven-development skill for task {{.TaskID}}: "{{.Title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.FilePath}}
(2) Feature test plan (09-test-plan.md)
(3) Feature architecture docs
{{- if .RelatedDocs}}
(4) Related docs: {{.RelatedDocs}}
{{- end}}
{{- if .RelatedTasks}}
({{if .RelatedDocs}}5{{else}}4{{end}}) Related tasks: {{.RelatedTasks}}
{{- end}}

{{template "tdd_process" .}}

EXIT GATE:
- All test cases pass
- Implementation matches spec{{if .RelatedDocs}} and related docs{{end}}
- Code follows conventions

Advance: shark task next-status {{.TaskID}}.
```

### Template Engine Conditional Features

**Yes, Go templates support rich conditionals and logic:**

**Conditionals:**
```go
{{if .RelatedDocs}}
  Read related docs: {{.RelatedDocs}}
{{end}}

{{if .RelatedDocs}}
  Read docs: {{.RelatedDocs}}
{{else}}
  No related documentation available
{{end}}

{{if and .RelatedDocs .RelatedTasks}}
  Both docs and tasks available
{{end}}
```

**Nested conditionals:**
```go
{{if .ComplexityTier}}
  {{if eq .ComplexityTier "SIMPLE"}}
    Lightweight approach
  {{else if eq .ComplexityTier "STANDARD"}}
    Focused approach
  {{else}}
    Comprehensive approach
  {{end}}
{{end}}
```

**Loops:**
```go
{{range .RelatedFeatures}}
  - Feature: {{.}}
{{end}}
```

**Whitespace control:**
```go
{{- if .X}}  {{/* trim leading whitespace */}}
{{if .X -}}  {{/* trim trailing whitespace */}}
```

**Benefits**:
- ✅ Multiline formatting (readable!)
- ✅ **Conditionals hide empty sections** (if/else/else if)
- ✅ **Smart auto-numbering** (conditional sections renumber automatically)
- ✅ **Complexity tier scaling** (different output per tier)
- ✅ Shared partials (change once, update everywhere)
- ✅ Clean git diffs (line-by-line changes)
- ✅ Comments with `{{/* comment */}}`
- ✅ Pre-compilation catches errors at startup

---

## Benefits

### For Template Authors

1. **Readability**: Multiline formatting, syntax highlighting in editors, inline comments
2. **Maintainability**: Edit templates in normal text files, git diffs show line-by-line changes
3. **Conditionals**: Show/hide sections based on data (e.g., hide "Related docs:" when empty)
4. **Smart numbering**: READ sections auto-renumber when conditionals add/remove items
5. **Complexity scaling**: Single template can adapt output based on complexity tier
   ```go
   {{if eq .ComplexityTier "SIMPLE"}}
     Brief instructions
   {{else if eq .ComplexityTier "STANDARD"}}
     Focused instructions
   {{else}}
     Comprehensive instructions
   {{end}}
   ```

### For Development Teams

6. **Reusability**: Shared partials for common sections (change once, update everywhere)
7. **Validation**: Pre-compile templates at startup catches syntax errors before production
8. **Testability**: Unit test templates in isolation with mock data
9. **Zero migration**: Existing inline templates continue to work unchanged
10. **Gradual adoption**: Convert templates at your own pace (no flag day)

---

## Real-World Example: Complexity Tier Scaling

**Problem**: Feature BA refinement needs different output depth based on complexity tier

**Current approach** (inline string):
```json
{
  "instruction_template": "Write feature PRD for {id}. CHECK COMPLEXITY TIER: Read feature metadata via shark feature get {id} --json | jq -r '.metadata.complexity_tier'. SCALE OUTPUT BY TIER: SIMPLE (1-page summary), STANDARD (focused PRD 2-3 pages), COMPLEX (full 6-file PRD). PRODUCE: ..."
}
```

Problems:
- Repetitive tier check logic in every template
- No actual conditional output (just instructions to agent)
- Agent must interpret tier and scale output manually

**With template engine**:

**File**: `templates/feature/ready_for_refinement_ba.tmpl`

```go
Write feature PRD for {{.ID}}: "{{.Title}}".

LOAD: specification-writing workflow write-feature-prd.md.

READ:
(1) Triage report or research report
(2) Parent epic PRD
(3) Feature description at {{.FilePath}}
{{- if .RelatedDocs}}
(4) Related docs: {{.RelatedDocs}}
{{- end}}

{{if eq .ComplexityTier "SIMPLE"}}
PRODUCE: 1-page PRD summary
- Goal (2-3 sentences)
- Key user story (single primary story)
- Core acceptance criteria (3-5 testable criteria)

EXIT GATE:
- Core AC testable
- Aligns with epic requirements
{{else if eq .ComplexityTier "STANDARD"}}
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

Advance: shark feature next-status {{.ID}}.
```

**Benefits**:
- ✅ Template itself adapts based on tier (agent gets right instructions)
- ✅ No repetitive tier-checking boilerplate
- ✅ Clear, distinct output expectations per tier
- ✅ Maintainable: change tier logic in one place
- ✅ Testable: render template with different tier values

---

## Migration Strategy

### Phase 1: Foundation (Week 1)
**Goal**: Add template engine with zero breaking changes

**Tasks**:
1. Create `internal/templates/engine.go` with Go template support
2. Add filename detection logic: if `instruction_template` ends with `.tmpl`, use engine; otherwise use legacy `PopulateTemplate`
3. Create `templates/` directory structure
4. Write unit tests for engine
5. Add template pre-compilation at startup (catches syntax errors early)

**Deliverable**: Engine ready, 100% backward compatible

**Migration example**:
```json
// Before (still works)
"instruction_template": "Launch developer for {task_id}"

// After (opt-in)
"instruction_template": "task/ready_for_development.tmpl"
```

---

### Phase 2: High-Value Templates (Week 2)
**Goal**: Convert 12 most complex templates to prove value

**Convert these templates**:
1. **Task execution** (5 files):
   - `task/ready_for_development.tmpl`
   - `task/ready_for_code_review.tmpl`
   - `task/ready_for_qa.tmpl`
   - `task/ready_for_refinement_ba.tmpl`
   - `task/ready_for_refinement_tech.tmpl`

2. **Feature planning** (4 files):
   - `feature/ready_for_research.tmpl`
   - `feature/ready_for_refinement_ba.tmpl`
   - `feature/ready_for_refinement_tech.tmpl`
   - `feature/ready_for_test_planning.tmpl`

3. **Epic strategic** (3 files):
   - `epic/ready_for_research.tmpl`
   - `epic/ready_for_feasibility_review_ba.tmpl`
   - `epic/ready_for_feasibility_review_tech.tmpl`

**Create partials**:
- `partials/_tdd_process.tmpl` (used by 5 task templates)
- `partials/_exit_gate.tmpl` (used by all templates)
- `partials/_read_section.tmpl` (used by 12+ templates)

**Update .sharkconfig.json**:
```json
{
  "template_directory": "templates",
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "instruction_template": "task/ready_for_development.tmpl"  // Changed from inline string
      }
    }
  }
}
```

**Deliverable**: 12 templates externalized, immediate readability/maintainability wins

---

### Phase 3: Full Migration (Week 3-4)
**Goal**: Migrate all 62 templates, complete ecosystem

**Tasks**:
1. Convert remaining 50 templates to external files
2. Expand partial library to 6-10 partials
3. Add `shark config validate` command to check template syntax
4. Create optional migration tool: `shark config migrate-templates` (auto-converts inline to files)
5. Update documentation with template authoring guide
6. Add deprecation notice for inline templates (warning, not error)

**Deliverable**: Complete external template system, all workflows using .tmpl files

---

### Phase 4: Advanced Features (Future)
**Enhancements as needed**:
1. Template inheritance (base templates with overrides)
2. Template testing framework
3. Hot reload in dev mode (watch templates/, reload on change)
4. Template linting (enforce conventions, max complexity)
5. Template library (shareable across projects)

---

## Technical Design

### Backward Compatible Approach

**Key insight: Use filename detection to choose rendering path**

The system automatically detects whether to use the new template engine or legacy string replacement:

```go
// Rendering logic in repository layer
func renderInstruction(instructionTemplate string, data map[string]interface{}) (string, error) {
    // If value ends with .tmpl, it's a template file reference
    if strings.HasSuffix(instructionTemplate, ".tmpl") {
        return templateEngine.Render(instructionTemplate, data)
    }

    // Otherwise, use legacy string replacement
    return PopulateTemplate(instructionTemplate, data), nil
}
```

**Config Examples:**

```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "task/ready_for_development.tmpl"
    }
  }
}
```
→ Uses **new template engine** (ends with .tmpl)

```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "Launch developer for task {task_id}..."
    }
  }
}
```
→ Uses **legacy string replacement** (doesn't end with .tmpl)

**Benefits:**
- ✅ Zero breaking changes (existing inline templates work unchanged)
- ✅ No migration required (convert templates at your own pace)
- ✅ Simple detection (filename suffix, no new config fields)
- ✅ Clear intent (`.tmpl` extension = template file)

### Template Engine

```go
// internal/templates/engine.go

type Engine struct {
    templates *template.Template
    templateDir string
}

func NewEngine(templateDir string) (*Engine, error)
func (e *Engine) Render(templateName string, data interface{}) (string, error)
func (e *Engine) LoadTemplate(path string) error  // Reload single template (dev mode)
```

### Config Updates

**No config schema changes required!** The existing `instruction_template` field works for both:

```go
type OrchestratorAction struct {
    Action              string   `json:"action"`
    InstructionTemplate string   `json:"instruction_template"`  // Can be inline string OR filename
    AgentType           string   `json:"agent_type,omitempty"`
    Skills              []string `json:"skills,omitempty"`
}
```

**Optional config addition:**

```go
type Config struct {
    TemplateDirectory string `json:"template_directory,omitempty"`  // Default: "templates"
    // ... existing fields
}
```

---

## Success Criteria

### MVP (Phases 1-2)
- [ ] Engine supports both inline and file templates
- [ ] 12 high-value templates converted
- [ ] 3+ reusable partials created
- [ ] Zero breaking changes
- [ ] Template rendering tests passing

### Full (Phase 3)
- [ ] All 62 templates externalized
- [ ] 6-10 partials in library
- [ ] `shark config validate` checks templates
- [ ] Migration tool available
- [ ] 95%+ test coverage

---

## Out of Scope

1. **Non-Go template engines** (Jinja2, Handlebars) - Go stdlib sufficient
2. **Visual template editor** - CLI/text-file workflow only
3. **Template marketplace** - Just git for sharing
4. **YAML config migration** - Keep .sharkconfig.json as JSON

---

## ROI Estimate

**Investment**: 36 hours for full migration

**Returns**:
- 75% reduction in template editing time
- 50% reduction in template-related bugs
- 2x velocity on template changes
- 40% reduction in onboarding time

**Payback**: ~2 weeks

---

## Dependencies

- E07-F29: Template variables ({related_docs}, {related_tasks}, etc.)
- Go stdlib text/template (already available)
- Existing PopulateTemplate function

---

## Related Documents

- Full proposal: `/home/jwwel/.claude/docs/external-prompt-templates-proposal.md`
- Template variable guide: `/home/jwwel/.claude/docs/instruction-template-variable-usage-guide.md`
- Current config: `.sharkconfig.json`

---

**Status**: DRAFT
**Date**: 2026-02-14
**Epic**: E07 - Enhancements
