# Test Plan: External Template Engine for Orchestrator Instructions

**Feature:** E07-F30 - Template Engine
**Complexity Tier:** STANDARD
**Version:** 1.0
**Last Updated:** 2026-02-14

---

## Executive Summary

This test plan validates the external template engine feature across three phases: foundation (MVP), high-value templates (12 templates), and full migration (62 templates). Testing focuses on backward compatibility (zero breaking changes), template rendering correctness, performance (< 5ms per render), and error handling. Test coverage target: 95%+ for template engine code, 100% backward compatibility verification.

**User Goal Traceability:**
- **Template Author (Primary):** Readable multiline formatting, conditionals, shared partials, validation errors
- **Development Team (Secondary):** Gradual migration, no breaking changes, testable templates
- **AI Orchestrator (Tertiary):** Correct rendering, no runtime errors, fast performance

**Test Strategy:** Unit tests for template functions, integration tests for service layer, regression tests for backward compatibility, performance benchmarks for rendering speed.

---

## 1. Acceptance Criteria Test Matrix

### US-E07-F30-001: Backward Compatible Template Engine

#### AC-1.1: If `instruction_template` ends with `.tmpl`, use template engine

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-1.1-T01 | Template with `.tmpl` extension | `instruction_template: "task/ready_for_development.tmpl"` | Template engine renders file content | Valid template file exists |
| AC-1.1-T02 | Case sensitivity check | `instruction_template: "task/ready_for_development.TMPL"` | **Fails** - case-sensitive match | Extension must be lowercase `.tmpl` |
| AC-1.1-T03 | Multiple `.tmpl` in path | `instruction_template: "templates.old/task.tmpl"` | Template engine renders (checks suffix only) | Only final extension matters |
| AC-1.1-T04 | Missing template file | `instruction_template: "task/nonexistent.tmpl"` | Error logged, empty string returned | Graceful degradation |

**Traceability:** Template Author needs clear detection logic → no ambiguity about which rendering path is used.

---

#### AC-1.2: If `instruction_template` doesn't end with `.tmpl`, use legacy string replacement

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-1.2-T01 | Inline template string | `instruction_template: "Launch {agent_type} for {task_id}"` | Legacy replacement: "Launch developer for E07-F30-001" | Standard inline template |
| AC-1.2-T02 | Template with `.txt` extension | `instruction_template: "task/readme.txt"` | Legacy replacement treats as literal string | No template rendering |
| AC-1.2-T03 | Empty instruction template | `instruction_template: ""` | Returns empty string | No crash |
| AC-1.2-T04 | No placeholders in inline | `instruction_template: "Static instruction"` | Returns "Static instruction" unchanged | No variable substitution needed |

**Traceability:** Development Team needs zero breaking changes → existing inline templates continue working.

---

#### AC-1.3: All existing inline templates render identically (zero breaking changes)

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-1.3-T01 | Regression test suite | All 62 inline templates from `.sharkconfig.json` | Capture output with test data, verify character-for-character match | Comprehensive regression |
| AC-1.3-T02 | Special characters in placeholders | `{title}` = "Task: Design & Test" | Renders "Task: Design & Test" (no escaping) | Ampersands, quotes, braces |
| AC-1.3-T03 | Empty placeholder values | `{related_docs}` = "" | Renders empty string (no "null" or "undefined") | Graceful empty handling |
| AC-1.3-T04 | Unicode in placeholders | `{title}` = "实现模板引擎" | Renders Unicode correctly | Non-ASCII characters |

**Test Data:** Use production-like placeholder values from existing epics/features/tasks.

**Traceability:** Development Team requires gradual migration → no forced changes to existing workflows.

---

#### AC-1.4: Template engine singleton initialized at startup with precompilation

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-1.4-T01 | First call initializes engine | Call `GetOrchestratorEngine()` | Engine created once via `sync.Once` | Lazy initialization |
| AC-1.4-T02 | Subsequent calls reuse singleton | Call `GetOrchestratorEngine()` 10 times | Same instance returned (pointer equality) | Thread-safe singleton |
| AC-1.4-T03 | Templates precompiled at init | Measure time for first vs second render | First render ~1-5ms, subsequent < 1ms | Parse once, execute many |
| AC-1.4-T04 | Concurrent access safety | 100 goroutines call `GetOrchestratorEngine()` | All receive same instance, no race conditions | Thread-safe initialization |

**Performance Benchmark:** Startup time should increase by < 50ms for 62 template precompilation.

**Traceability:** AI Orchestrator needs fast rendering → precompilation reduces runtime overhead.

---

#### AC-1.5: Template syntax errors fail fast at startup with helpful messages

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-1.5-T01 | Unclosed conditional | `{{if .condition}}...` (no `{{end}}`) | Panic at startup with file/line: "task/test.tmpl:5: unexpected EOF" | Parse error caught early |
| AC-1.5-T02 | Invalid function call | `{{unknown_func .var}}` | Panic at startup: "function 'unknown_func' not defined" | Function validation |
| AC-1.5-T03 | Malformed variable | `{{.var with space}}` | Panic at startup: "unexpected space in variable" | Syntax validation |
| AC-1.5-T04 | All templates valid | No syntax errors in any `.tmpl` file | Application starts successfully | Happy path |

**Test Setup:** Create temporary `templates/` directory with malformed templates, verify startup fails.

**Traceability:** Template Author needs clear validation errors → fast feedback loop.

---

### US-E07-F30-002: External Template Files with Conditionals

#### AC-2.1: Templates support `{{if .RelatedDocs}}` conditionals to hide empty sections

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.1-T01 | Empty string hides section | Template: `{{if .related_docs}}Docs: {{.related_docs}}{{end}}`, vars: `related_docs=""` | Output: "" (empty) | Section hidden |
| AC-2.1-T02 | Populated string shows section | Template: same as above, vars: `related_docs="prd.md, arch.md"` | Output: "Docs: prd.md, arch.md" | Section shown |
| AC-2.1-T03 | Whitespace-only treated as empty | vars: `related_docs="   "` | Output: "" (hidden) | Use `isEmpty` custom function |
| AC-2.1-T04 | Nested conditionals | `{{if .tier}}{{if eq .tier "SIMPLE"}}Brief{{end}}{{end}}` | Renders "Brief" if tier="SIMPLE", else empty | Nesting works |

**Template Example:**
```go
READ:
(1) Task spec at {{.file_path}}
{{- if .related_docs}}
(2) Related docs: {{.related_docs}}
{{- end}}
```

**Traceability:** Template Author needs to hide empty sections → no "Related docs: " when none exist.

---

#### AC-2.2: Templates support `{{if eq .ComplexityTier "SIMPLE"}}` for tier-specific output

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.2-T01 | SIMPLE tier branch | Template: `{{if eq .complexity_tier "SIMPLE"}}Lightweight{{end}}`, vars: `complexity_tier="SIMPLE"` | Output: "Lightweight" | Branch executed |
| AC-2.2-T02 | STANDARD tier (no match) | Same template, vars: `complexity_tier="STANDARD"` | Output: "" (empty) | Branch not executed |
| AC-2.2-T03 | Case-sensitive comparison | vars: `complexity_tier="simple"` (lowercase) | Output: "" (no match) | Case matters |
| AC-2.2-T04 | Missing tier field | vars: `complexity_tier=""` | Output: "" (no match) | Graceful empty handling |

**Traceability:** Template Author needs tier-specific instructions → different output depth per SIMPLE/STANDARD/COMPLEX.

---

#### AC-2.3: Templates support `{{else if}}` and `{{else}}` for multiple branches

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.3-T01 | Three-way branch (SIMPLE) | `{{if eq .tier "SIMPLE"}}Brief{{else if eq .tier "STANDARD"}}Focused{{else}}Full{{end}}`, vars: `tier="SIMPLE"` | Output: "Brief" | First branch |
| AC-2.3-T02 | Three-way branch (STANDARD) | Same template, vars: `tier="STANDARD"` | Output: "Focused" | Second branch |
| AC-2.3-T03 | Three-way branch (COMPLEX) | Same template, vars: `tier="COMPLEX"` | Output: "Full" | Else branch |
| AC-2.3-T04 | Empty tier falls to else | vars: `tier=""` | Output: "Full" | Else is default |

**Traceability:** Template Author needs multiple output variants → PRD requires different instructions for 3 tiers.

---

#### AC-2.4: Whitespace control with `{{- if}}` trims leading/trailing whitespace

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.4-T01 | Trim leading whitespace | `Before\n{{- if .show}}After{{end}}` with `show=true` | Output: "BeforeAfter" (newline removed) | Leading trim works |
| AC-2.4-T02 | Trim trailing whitespace | `{{if .show -}}\nAfter{{end}}` with `show=true` | Output: "After" (newline removed) | Trailing trim works |
| AC-2.4-T03 | Both sides trim | `Before\n{{- if .show -}}\nAfter{{end}}` | Output: "BeforeAfter" | Double trim |
| AC-2.4-T04 | No trim without dash | `Before\n{{if .show}}After{{end}}` | Output: "Before\nAfter" (newline preserved) | Default preserves whitespace |

**Traceability:** Template Author needs clean formatting → avoid extra blank lines in rendered output.

---

#### AC-2.5: Templates can use `{{range}}` for looping over related items

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.5-T01 | Range over slice (CSV) | Template: `{{range split .related_docs ","}}{{.}}{{end}}`, vars: `related_docs="a.md,b.md"` | Output: "a.mdb.md" | Basic iteration |
| AC-2.5-T02 | Empty slice | vars: `related_docs=""` | Output: "" (no iterations) | Graceful empty |
| AC-2.5-T03 | Single item | vars: `related_docs="single.md"` | Output: "single.md" | Single iteration |
| AC-2.5-T04 | Range with index | `{{range $i, $doc := split .related_docs ","}}{{$i}}: {{$doc}}{{end}}` | Output: "0: a.md1: b.md" | Index access |

**Note:** Requires `split` custom function in Phase 2/3 (not MVP). For MVP, use `{{.related_docs}}` directly (already comma-separated).

**Traceability:** Template Author needs to iterate over lists → Phase 3 enhancement for advanced templates.

---

#### AC-2.6: Custom functions available: `eq`, `ne`, `and`, `or`, `isEmpty`, `join`

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-2.6-T01 | `eq` function | `{{if eq .status "completed"}}Done{{end}}`, vars: `status="completed"` | Output: "Done" | Equality works |
| AC-2.6-T02 | `ne` function | `{{if ne .status "todo"}}Active{{end}}`, vars: `status="in_progress"` | Output: "Active" | Not-equal works |
| AC-2.6-T03 | `isEmpty` function | `{{if isEmpty .related_docs}}None{{end}}`, vars: `related_docs=""` | Output: "None" | Empty string detection |
| AC-2.6-T04 | `isEmpty` with whitespace | vars: `related_docs="   "` | Output: "None" | Whitespace is empty |
| AC-2.6-T05 | `and` logical operator | `{{if and .show .enabled}}Visible{{end}}`, vars: `show=true, enabled=true` | Output: "Visible" | Both true |
| AC-2.6-T06 | `or` logical operator | `{{if or .show .debug}}Visible{{end}}`, vars: `show=false, debug=true` | Output: "Visible" | One true |
| AC-2.6-T07 | `join` function (Phase 3) | `{{join .skills ", "}}`, vars: `skills=["go", "testing"]` | Output: "go, testing" | Array join |

**Traceability:** Template Author needs rich conditionals → PRD requires if/else, tier-based branching, empty checks.

---

### US-E07-F30-003: Shared Partials

#### AC-3.1: Partials use `_prefix.tmpl` naming convention

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-3.1-T01 | Partial filename starts with `_` | File: `templates/partials/_tdd_process.tmpl` | Template loads successfully | Naming convention followed |
| AC-3.1-T02 | Non-partial file in partials dir | File: `templates/partials/not_partial.tmpl` | Still loads (no enforcement) | Convention, not rule |
| AC-3.1-T03 | Partial in subdirectory | File: `templates/partials/common/_read_section.tmpl` | Loads if path included in `ParseGlob` | Subdirectory support |

**Rationale:** Naming convention is for human readability, not enforced by engine.

**Traceability:** Template Author needs to identify partials → underscore prefix is conventional Go template pattern.

---

#### AC-3.2: Templates include partials with `{{template "tdd_process" .}}`

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-3.2-T01 | Include partial by name | Template: `{{template "_tdd_process" .}}`, Partial defines: `{{define "_tdd_process"}}TDD steps{{end}}` | Output: "TDD steps" | Basic include |
| AC-3.2-T02 | Pass context to partial | Partial uses `{{.task_id}}`, main template passes `.` | Partial renders with task_id | Context passed |
| AC-3.2-T03 | Missing partial | Template: `{{template "_nonexistent" .}}` | Error logged, empty output | Graceful degradation |
| AC-3.2-T04 | Nested partials | Partial A includes Partial B | Both render correctly | Nesting works (depth < 3) |

**Template Example:**
```go
{{define "_tdd_process"}}
TDD PROCESS:
1. Write failing test first (red)
2. Implement minimum code to pass (green)
3. Refactor while keeping tests green
{{end}}
```

**Traceability:** Template Author needs code reuse → change TDD process once, updates 5+ templates.

---

#### AC-3.3: Partials stored in `templates/partials/` directory

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-3.3-T01 | Partial in correct directory | File: `templates/partials/_exit_gate.tmpl` | Loads during precompilation | Standard location |
| AC-3.3-T02 | Partial in entity subdirectory | File: `templates/task/_tdd_process.tmpl` | Also loads (ParseGlob includes all subdirs) | Alternative location |
| AC-3.3-T03 | Empty partials directory | `templates/partials/` exists but empty | No error, just no partials | Graceful empty |

**Implementation:** `ParseGlob("templates/*/*.tmpl")` includes `partials/` and `task/`, `feature/`, `epic/`.

**Traceability:** Development Team needs organized templates → separation of partials from entity templates.

---

#### AC-3.4: At least 3 partials created: `_tdd_process.tmpl`, `_exit_gate.tmpl`, `_read_section.tmpl`

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-3.4-T01 | TDD process partial exists | File: `templates/partials/_tdd_process.tmpl` | Contains 4-step TDD process | Used in 5+ task templates |
| AC-3.4-T02 | Exit gate partial exists | File: `templates/partials/_exit_gate.tmpl` | Defines exit criteria format | Used in 12+ templates |
| AC-3.4-T03 | Read section partial exists | File: `templates/partials/_read_section.tmpl` | Lists numbered read items | Used in 18+ templates |
| AC-3.4-T04 | All partials referenced | Grep `.tmpl` files for `{{template "_tdd_process"}}` etc. | Find 5+ references for TDD, 12+ for exit gate | Partials actually used |

**Verification:** Manual inspection of created partial files + usage count in templates.

**Traceability:** Template Author needs immediate productivity win → 3 partials reduce duplication by ~80 lines each.

---

#### AC-3.5: Changes to partials automatically affect all templates using them

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-3.5-T01 | Modify partial content | Change `_tdd_process.tmpl` step 2: "green" → "pass", restart app | All 5 task templates using partial show "pass" | Propagation works |
| AC-3.5-T02 | Add new line to partial | Add step 5 to TDD process | All templates now show 5 steps | Addition propagates |
| AC-3.5-T03 | Remove partial content | Delete step 4 from partial | All templates now show 3 steps | Deletion propagates |
| AC-3.5-T04 | Hot reload (Phase 4) | Modify partial without restart | **Phase 4 enhancement** - not MVP | Future feature |

**Test Method:** Integration test that modifies partial file, restarts engine, verifies all templates render with new content.

**Traceability:** Template Author needs maintainability → change once, update everywhere.

---

### US-E07-F30-004: High-Value Template Migration

#### AC-4.1-4.3: 12 specific templates converted (task 5, feature 4, epic 3)

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-4.1-T01 | Task templates exist | Files: `task/ready_for_development.tmpl`, `task/ready_for_code_review.tmpl`, `task/ready_for_qa.tmpl`, `task/ready_for_refinement_ba.tmpl`, `task/ready_for_refinement_tech.tmpl` | All 5 files exist in `templates/task/` | File creation |
| AC-4.2-T01 | Feature templates exist | Files: `feature/ready_for_research.tmpl`, `feature/ready_for_refinement_ba.tmpl`, `feature/ready_for_refinement_tech.tmpl`, `feature/ready_for_test_planning.tmpl` | All 4 files exist in `templates/feature/` | File creation |
| AC-4.3-T01 | Epic templates exist | Files: `epic/ready_for_research.tmpl`, `epic/ready_for_feasibility_review_ba.tmpl`, `epic/ready_for_feasibility_review_tech.tmpl` | All 3 files exist in `templates/epic/` | File creation |

**Verification:** File existence check via `ls templates/task/*.tmpl | wc -l` should return 5.

**Traceability:** Development Team needs immediate value → 12 most complex templates gain readability first.

---

#### AC-4.4: `.sharkconfig.json` updated to reference `.tmpl` files instead of inline strings

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-4.4-T01 | Config references external file | `.sharkconfig.json`: `"instruction_template": "task/ready_for_development.tmpl"` | Config loads successfully | External reference |
| AC-4.4-T02 | Remaining inline templates unchanged | Statuses not migrated still have inline strings | Legacy path works | Mixed inline + external |
| AC-4.4-T03 | All 12 migrated templates referenced | Count `.tmpl` references in config | Should be 12 references | Coverage check |

**Verification:** `grep -c '\.tmpl"' .sharkconfig.json` should return 12.

**Traceability:** Development Team needs no breaking changes → gradual migration, not flag day.

---

#### AC-4.5: All 12 templates render identically to inline versions (regression test)

| Test ID | Test Case | Input | Expected Output | Edge Cases |
|---------|-----------|-------|-----------------|------------|
| AC-4.5-T01 | Regression test for each template | Capture inline output, render external template with same data | Character-for-character match | Perfect fidelity |
| AC-4.5-T02 | Test with empty placeholders | `related_docs=""`, `related_tasks=""` | External template hides sections (better than inline) | **Acceptable difference** (improvement) |
| AC-4.5-T03 | Test with all placeholders | Full data: task_id, title, tier, related_docs, etc. | Exact match | Comprehensive data |
| AC-4.5-T04 | Whitespace differences | External template may have cleaner whitespace | **Acceptable** if semantically identical | Formatting improvement |

**Acceptance Criteria:** Semantic equivalence (same information), but external templates may have improved formatting (fewer blank lines).

**Traceability:** Development Team requires zero regressions → workflows must produce same results.

---

---

## 2. API Contract Test Cases

### OrchestratorRenderer Interface

#### Constructor Tests

| Test ID | Test Case | Input | Expected Output | Error Handling |
|---------|-----------|-------|-----------------|----------------|
| API-01 | Create renderer with valid dir | `NewOrchestratorRenderer("templates")` | Returns `*OrchestratorRenderer`, no error | Templates precompiled |
| API-02 | Create renderer with missing dir | `NewOrchestratorRenderer("nonexistent")` | Returns nil, error: "template directory not found" | Graceful error |
| API-03 | Create renderer with malformed template | Template has syntax error: `{{if .x}}` (no end) | Returns nil, error with file/line: "templates/task/test.tmpl:5: unexpected EOF" | Parse error caught |
| API-04 | Create renderer with empty dir | `templates/` exists but empty | Returns renderer with no templates, no error | Empty is valid |

**Traceability:** AI Orchestrator needs reliable initialization → fail fast on invalid templates.

---

#### Singleton Accessor Tests

| Test ID | Test Case | Input | Expected Output | Thread Safety |
|---------|-----------|-------|-----------------|----------------|
| API-05 | First call creates singleton | `GetOrchestratorEngine()` | Returns `*OrchestratorRenderer` | Uses `sync.Once` |
| API-06 | Subsequent calls return same | Call 100 times, compare pointers | All pointers equal | Thread-safe |
| API-07 | Concurrent access | 1000 goroutines call `GetOrchestratorEngine()` | All receive same instance, no race | Data race detector passes |

**Test Tool:** `go test -race` to detect race conditions.

**Traceability:** AI Orchestrator needs thread-safe access → CLI commands may run concurrently.

---

#### Render Method Tests

| Test ID | Test Case | Input | Expected Output | Performance |
|---------|-----------|-------|-----------------|-------------|
| API-08 | Render existing template | `Render("task/ready_for_development.tmpl", vars)` | Returns rendered string, no error | < 5ms |
| API-09 | Render missing template | `Render("task/nonexistent.tmpl", vars)` | Returns "", error: "template not found: task/nonexistent.tmpl" | Graceful error |
| API-10 | Render with empty vars | `Render("task/test.tmpl", map[string]string{})` | Returns template with empty placeholders | No crash |
| API-11 | Render with missing var | Template uses `{{.missing_var}}`, var not provided | Error: "failed to execute template: missing variable" | Execution error |
| API-12 | Render performance benchmark | Render 1000 times with same data | Average < 1ms (precompiled), p95 < 5ms | Performance SLA |

**Benchmark Tool:** `go test -bench=BenchmarkRender -benchmem`

**Traceability:** AI Orchestrator needs fast rendering → precompilation ensures < 5ms SLA.

---

### OrchestratorAction.PopulateTemplate() Contract

| Test ID | Test Case | Input | Expected Output | Backward Compatibility |
|---------|-----------|-------|-----------------|------------------------|
| API-13 | Inline template (legacy) | `InstructionTemplate: "Launch {agent_type}"`, vars: `agent_type="dev"` | Returns "Launch dev" | Legacy path works |
| API-14 | External template (new) | `InstructionTemplate: "task/test.tmpl"`, vars: same | Returns rendered template content | New path works |
| API-15 | Empty template | `InstructionTemplate: ""`, vars: any | Returns "" | No crash |
| API-16 | Nil vars map | `InstructionTemplate: "..."`, vars: `nil` | Returns template unchanged (inline) or error (external) | Graceful nil |
| API-17 | Execution error handling | External template fails to render | Logs error, returns empty string | Graceful degradation |

**Test Method:** Unit tests for `PopulateTemplate` method with both inline and `.tmpl` templates.

**Traceability:** Development Team needs backward compatibility → existing inline templates unchanged.

---

### Template Function Map Contract

| Test ID | Test Case | Input | Expected Output | Function Correctness |
|---------|-----------|-------|-----------------|---------------------|
| API-18 | `eq` function | `eq "a" "a"` | true | Equality works |
| API-19 | `eq` different types | `eq 1 "1"` | false (different types) | Type-safe |
| API-20 | `ne` function | `ne "a" "b"` | true | Not-equal works |
| API-21 | `isEmpty` empty string | `isEmpty ""` | true | Empty detection |
| API-22 | `isEmpty` whitespace | `isEmpty "   "` | true | Whitespace is empty |
| API-23 | `isEmpty` non-empty | `isEmpty "text"` | false | Non-empty detection |
| API-24 | `isSimple` function | `isSimple "SIMPLE"` | true | Tier helper |
| API-25 | `isStandard` function | `isStandard "STANDARD"` | true | Tier helper |
| API-26 | `isComplex` function | `isComplex "COMPLEX"` | true | Tier helper |
| API-27 | `join` function (Phase 3) | `join ["a", "b"] ", "` | "a, b" | Array join |

**Test Method:** Unit tests for `orchestratorFuncs()` function map.

**Traceability:** Template Author needs reliable functions → custom functions must work correctly.

---

### Placeholder Builder Contract Extension

| Test ID | Test Case | Input | Expected Output | New Field |
|---------|-----------|-------|-----------------|-----------|
| API-28 | Task with complexity tier | Task metadata: `complexity_tier="STANDARD"` | Placeholder map includes `"complexity_tier": "STANDARD"` | Field added |
| API-29 | Task without complexity tier | Task metadata: empty | Placeholder map: `"complexity_tier": ""` | Empty default |
| API-30 | Feature with complexity tier | Feature metadata: `complexity_tier="COMPLEX"` | Placeholder map includes field | Feature support |
| API-31 | Existing placeholders unchanged | Task has task_id, title, etc. | All existing placeholders present | No breaking changes |

**Test Method:** Unit tests for `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`.

**Traceability:** Template Author needs complexity tier → PRD requires tier-based branching in templates.

---

---

## 3. Component Test Strategy

### OrchestratorRenderer Component

**Component Responsibilities:**
- Load and precompile `.tmpl` files from `templates/` directory
- Execute templates with placeholder data
- Provide custom template functions
- Handle partial template includes

**Test Coverage Goals:**
- **Unit Tests:** 95%+ coverage of `orchestrator_renderer.go`
- **Integration Tests:** Full workflow rendering (service → template → output)
- **Error Tests:** All error paths (missing template, syntax error, execution error)

**Test Organization:**
```
internal/templates/
├── orchestrator_renderer.go        # Implementation
├── orchestrator_renderer_test.go   # Unit + integration tests
└── orchestrator_test_fixtures/     # Test templates
    ├── valid_template.tmpl
    ├── invalid_syntax.tmpl
    └── missing_partial.tmpl
```

**Key Test Cases:**
1. **Precompilation Tests:** Verify all templates load at startup, syntax errors fail fast
2. **Rendering Tests:** Verify conditionals, partials, custom functions work
3. **Performance Tests:** Benchmark rendering speed (< 5ms), memory usage (< 100KB)
4. **Error Tests:** Missing template, execution error, partial not found

**Existing Pattern to Follow:** `internal/templates/renderer_test.go` (task markdown templates)
- Use `testify/assert` and `testify/require` for assertions
- Table-driven tests for multiple scenarios
- Mock loader for testing (if needed)

---

### Template Functions Component

**Component Responsibilities:**
- Provide custom template functions (`eq`, `ne`, `isEmpty`, etc.)
- Return `template.FuncMap` for template engine

**Test Coverage Goals:**
- **Unit Tests:** 100% coverage of all custom functions
- **Edge Case Tests:** Empty strings, nil values, type mismatches

**Test Organization:**
```go
func TestTemplateFuncs_Eq(t *testing.T) {
    funcs := orchestratorFuncs()
    eqFunc := funcs["eq"].(func(a, b interface{}) bool)

    tests := []struct {
        name   string
        a, b   interface{}
        want   bool
    }{
        {"equal strings", "a", "a", true},
        {"different strings", "a", "b", false},
        {"equal ints", 1, 1, true},
        {"different types", 1, "1", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := eqFunc(tt.a, tt.b)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

**Existing Pattern:** `internal/templates/renderer_test.go` lines 363-405 (template function tests)

---

### Placeholder Builders Component

**Component Responsibilities:**
- Build `map[string]string` from task/feature/epic entities
- Include complexity tier from metadata
- Include related docs/tasks/features/epics

**Test Coverage Goals:**
- **Unit Tests:** 90%+ coverage of `*PlaceholdersWithRelated` functions
- **Edge Case Tests:** Missing metadata, empty related entities, nil repos

**Test Organization:**
```go
func TestTaskPlaceholdersWithRelated_ComplexityTier(t *testing.T) {
    task := &models.Task{
        Key:      "E07-F30-001",
        Title:    "Test Task",
        Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
    }

    placeholders := TaskPlaceholdersWithRelated(ctx, task, docRepo, relRepo)

    assert.Equal(t, "STANDARD", placeholders["complexity_tier"])
    assert.Equal(t, "E07-F30-001", placeholders["task_id"])
}
```

**Existing Pattern:** `internal/config/template_helpers.go` (existing placeholder builders)

---

### Integration: Service Layer → Template Engine

**Integration Points:**
- `EpicService.resolveAction()` calls `action.PopulateTemplate(vars)`
- `FeatureService.resolveAction()` calls `action.PopulateTemplate(vars)`
- `DisplayService.ResolveEpicAction()`, `ResolveFeatureAction()` similar pattern

**Test Coverage Goals:**
- **Integration Tests:** Verify full flow from service → config → template engine
- **Backward Compatibility Tests:** Inline templates still work
- **Mixed Template Tests:** Some inline, some external in same config

**Test Organization:**
```go
func TestFeatureService_ResolveAction_ExternalTemplate(t *testing.T) {
    // Setup: Feature in "ready_for_refinement_ba" status
    feature := &models.Feature{
        Key:      "E07-F30",
        Title:    "template engine",
        Status:   "ready_for_refinement_ba",
        Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
    }

    // Mock config with external template
    config := &config.Config{
        StatusMetadata: map[string]config.StatusMetadata{
            "ready_for_refinement_ba": {
                OrchestratorAction: &config.OrchestratorAction{
                    Action:              "spawn_agent",
                    AgentType:           "ba",
                    InstructionTemplate: "feature/ready_for_refinement_ba.tmpl", // External
                },
            },
        },
    }

    // Execute
    action := featureService.resolveAction(ctx, feature, feature.Status)

    // Verify
    assert.NotEmpty(t, action.Instruction)
    assert.Contains(t, action.Instruction, "E07-F30") // Feature key rendered
    assert.Contains(t, action.Instruction, "STANDARD") // Tier rendered (if template uses it)
}
```

**Existing Pattern:** `internal/services/feature_service_test.go` (if exists, or create new file)

---

---

## 4. Integration Scenarios

### Scenario 1: Backward Compatibility (Inline + External Templates)

**Goal:** Verify mixed configuration (some inline, some external) works without breaking changes.

**Setup:**
1. Configure `.sharkconfig.json` with 12 external templates (`.tmpl` files)
2. Leave remaining 50 templates as inline strings
3. Create test epic/feature/task in each status

**Test Steps:**
1. Transition task through statuses: todo → in_progress → ready_for_development (external) → ready_for_code_review (external) → completed
2. Capture orchestrator action instruction at each status
3. Verify:
   - External templates render correctly (rich formatting, conditionals work)
   - Inline templates render identically to before (string replacement)
   - No errors or warnings in logs
   - Workflow progresses normally

**Expected Result:**
- All statuses produce valid instructions
- External templates show improved formatting (fewer blank lines)
- Inline templates unchanged (character-for-character match with baseline)

**Traceability:** Development Team needs gradual migration → no flag day, mixed templates work.

---

### Scenario 2: Service Layer Integration (Epic/Feature/Task Actions)

**Goal:** Verify orchestrator actions work across all entity types (epic, feature, task).

**Setup:**
1. Create external templates for each entity type:
   - `epic/ready_for_research.tmpl`
   - `feature/ready_for_refinement_ba.tmpl`
   - `task/ready_for_development.tmpl`
2. Configure `.sharkconfig.json` to reference these templates
3. Create test epic E99, feature E99-F99, task E99-F99-999

**Test Steps:**
1. **Epic action:** Call `epic next-status E99` with status="ready_for_research"
   - Verify: `EpicService.resolveAction()` renders epic template
   - Check: Instruction contains epic key, title, related docs
2. **Feature action:** Call `feature next-status E99-F99` with status="ready_for_refinement_ba"
   - Verify: `FeatureService.resolveAction()` renders feature template
   - Check: Instruction contains feature key, complexity tier (if set), related features
3. **Task action:** Call `task next-status E99-F99-999` with status="ready_for_development"
   - Verify: `DisplayService.ResolveTaskAction()` renders task template
   - Check: Instruction contains task key, TDD process (from partial), exit gate

**Expected Result:**
- All entity types render their respective templates correctly
- Placeholders populated with correct entity data
- Partials included successfully (e.g., TDD process in task template)

**Traceability:** AI Orchestrator needs consistent behavior → all entity types use same template engine.

---

### Scenario 3: Complexity Tier Scaling (SIMPLE vs STANDARD vs COMPLEX)

**Goal:** Verify templates adapt output based on complexity tier metadata.

**Setup:**
1. Create feature BA refinement template with tier-based branching:
   ```go
   {{if eq .complexity_tier "SIMPLE"}}
   PRODUCE: 1-page PRD summary
   {{else if eq .complexity_tier "STANDARD"}}
   PRODUCE: Focused PRD (2-3 pages)
   {{else}}
   PRODUCE: Full 6-file PRD
   {{end}}
   ```
2. Create 3 test features:
   - F01: metadata `complexity_tier="SIMPLE"`
   - F02: metadata `complexity_tier="STANDARD"`
   - F03: metadata `complexity_tier="COMPLEX"`

**Test Steps:**
1. Transition each feature to `ready_for_refinement_ba`
2. Capture rendered instruction for each
3. Verify:
   - F01 instruction contains "1-page PRD summary"
   - F02 instruction contains "Focused PRD (2-3 pages)"
   - F03 instruction contains "Full 6-file PRD"

**Expected Result:**
- Each tier produces different output
- Template conditional logic works correctly
- Placeholder `complexity_tier` populated from metadata

**Traceability:** Template Author needs tier-specific output → PRD requires scaling by complexity.

---

### Scenario 4: Partial Template Propagation

**Goal:** Verify changes to partials affect all templates using them.

**Setup:**
1. Create partial `_tdd_process.tmpl` with 4 steps
2. Create 3 task templates that include this partial:
   - `task/ready_for_development.tmpl`
   - `task/ready_for_code_review.tmpl`
   - `task/ready_for_qa.tmpl`
3. Create 3 test tasks, one for each template

**Test Steps:**
1. **Initial state:** Render all 3 tasks, verify TDD process shows 4 steps
2. **Modify partial:** Add 5th step to `_tdd_process.tmpl`: "5. Document changes"
3. **Restart app:** Reinitialize template engine (simulates deployment)
4. **Re-render tasks:** Render all 3 tasks again
5. **Verify:** All 3 tasks now show 5 steps in TDD process

**Expected Result:**
- Single change to partial propagates to all 3 task templates
- No need to modify individual task templates
- Demonstrates maintainability win

**Traceability:** Template Author needs maintainability → change once, update everywhere.

---

### Scenario 5: Error Handling (Missing Template, Execution Error)

**Goal:** Verify graceful degradation when templates fail.

**Setup:**
1. Configure `.sharkconfig.json` with:
   - Valid template: `task/ready_for_development.tmpl`
   - Missing template: `task/nonexistent.tmpl`
   - Template with execution error: `task/error.tmpl` (references `{{.missing_var}}`)
2. Create 3 test tasks, one for each scenario

**Test Steps:**
1. **Valid template:** Transition task to `ready_for_development`
   - Verify: Renders correctly, no errors
2. **Missing template:** Transition task to status with `nonexistent.tmpl`
   - Verify: Error logged, empty instruction returned, workflow continues
3. **Execution error:** Transition task to status with `error.tmpl`
   - Verify: Error logged, empty instruction returned, workflow continues

**Expected Result:**
- Valid template works normally
- Missing template: Error logged ("template not found"), empty instruction, no crash
- Execution error: Error logged ("failed to execute template"), empty instruction, no crash
- All cases: Workflow continues (orchestrator sees empty instruction, not crash)

**Traceability:** AI Orchestrator needs reliability → template errors don't break workflows.

---

---

## 5. Test Data & Fixtures

### Test Templates

**Location:** `internal/templates/test_fixtures/`

**Test Template 1: Basic Variables**
```go
// File: test_fixtures/basic.tmpl
Task: {{.task_id}}
Title: {{.title}}
Status: {{.status}}
```

**Test Template 2: Conditionals**
```go
// File: test_fixtures/conditional.tmpl
{{if .related_docs}}
Related: {{.related_docs}}
{{else}}
No related docs.
{{end}}
```

**Test Template 3: Complexity Tier**
```go
// File: test_fixtures/tier.tmpl
{{if eq .complexity_tier "SIMPLE"}}
Brief instructions
{{else if eq .complexity_tier "STANDARD"}}
Focused instructions
{{else}}
Comprehensive instructions
{{end}}
```

**Test Template 4: Partial Include**
```go
// File: test_fixtures/with_partial.tmpl
Main template content.
{{template "_test_partial" .}}
End of template.

// File: test_fixtures/_test_partial.tmpl
{{define "_test_partial"}}
Partial content here.
{{end}}
```

**Test Template 5: Malformed (Syntax Error)**
```go
// File: test_fixtures/malformed.tmpl
{{if .condition}}
No closing end tag!
```

---

### Test Data: Placeholder Maps

**Minimal Task Placeholders:**
```go
vars := map[string]string{
    "task_id":        "E07-F30-001",
    "title":          "Implement template engine",
    "status":         "ready_for_development",
    "epic":           "E07",
    "feature":        "E07-F30",
    "agent_type":     "developer",
    "file_path":      "docs/plan/E07-enhancements/E07-F30-template-engine/tasks/001.md",
    "related_docs":   "",
    "related_tasks":  "",
    "complexity_tier": "",
}
```

**Full Task Placeholders (with related entities):**
```go
vars := map[string]string{
    "task_id":         "E07-F30-001",
    "title":           "Implement template engine",
    "status":          "ready_for_development",
    "epic":            "E07",
    "feature":         "E07-F30",
    "agent_type":      "developer",
    "priority":        "5",
    "file_path":       "docs/plan/E07-enhancements/E07-F30-template-engine/tasks/001.md",
    "related_docs":    "prd.md, architecture.md, api-contracts.md",
    "related_tasks":   "E07-F29-003",
    "complexity_tier": "STANDARD",
}
```

**Feature Placeholders:**
```go
vars := map[string]string{
    "id":              "E07-F30",
    "title":           "template engine",
    "epic":            "E07",
    "file_path":       "docs/plan/E07-enhancements/E07-F30-template-engine/feature.md",
    "related_docs":    "feature-research-tactical.md",
    "related_features": "E07-F29",
    "complexity_tier": "STANDARD",
}
```

---

### Regression Test Baseline Data

**Purpose:** Capture expected output for all 62 inline templates to verify backward compatibility.

**Process:**
1. **Before migration:** Run all inline templates with test data, capture output to files
2. **After migration:** Run same templates (now external `.tmpl` files) with same data
3. **Compare:** Diff outputs character-by-character (or semantically if whitespace differs)

**Storage Location:** `internal/templates/test_fixtures/baseline/`

**File Naming:**
- `baseline/task-ready_for_development-inline.txt` (captured before)
- `baseline/task-ready_for_development-external.txt` (captured after)

**Tool:** `diff -u baseline/inline.txt baseline/external.txt` or Go test comparison.

---

---

## 6. Performance & Security Testing

### Performance Benchmarks

**Target SLA:** Template rendering < 5ms per call (p95), < 1ms average (precompiled).

**Benchmark Tests:**

| Benchmark | Target | Measurement |
|-----------|--------|-------------|
| Template precompilation time (62 templates) | < 50ms | Startup overhead |
| First render (lazy init) | < 10ms | Initialization + first execution |
| Subsequent renders (cached) | < 1ms avg, < 5ms p95 | Execution only |
| Memory footprint (62 precompiled templates) | < 100KB | Heap allocation |
| Concurrent renders (1000 goroutines) | No contention, same as single-threaded | Thread-safety |

**Go Benchmark Pattern:**
```go
func BenchmarkOrchestratorRenderer_Render(b *testing.B) {
    renderer, _ := NewOrchestratorRenderer("templates")
    vars := map[string]string{
        "task_id": "E07-F30-001",
        "title":   "Test Task",
        // ... full vars
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = renderer.Render("task/ready_for_development.tmpl", vars)
    }
}
```

**Run:** `go test -bench=BenchmarkOrchestratorRenderer -benchmem`

**Expected Output:**
```
BenchmarkOrchestratorRenderer_Render-8   1000000   800 ns/op   256 B/op   4 allocs/op
```

**Acceptance:** < 1ms per op (1,000,000 ns), minimal allocations (< 500 B/op).

---

### Security Tests

**Security Concerns:** Template injection, arbitrary file access, code execution (see `04-security-considerations.md`).

**Security Test Cases:**

| Test ID | Test Case | Attack Vector | Expected Behavior |
|---------|-----------|---------------|-------------------|
| SEC-01 | Template injection via placeholder | Placeholder value: `{{template "evil"}}` | Rendered as literal string, NOT executed |
| SEC-02 | Path traversal in template name | Template: `../../etc/passwd` | Error: "template not found" (ParseGlob restricts to `templates/`) |
| SEC-03 | Arbitrary file read | Template uses `{{readFile "/etc/passwd"}}` | Error: "function 'readFile' not defined" (no file read functions) |
| SEC-04 | Code execution attempt | Template uses `{{exec "rm -rf /"}}` | Error: "function 'exec' not defined" (no code execution) |
| SEC-05 | Secret exposure | Placeholder value: `API_KEY=secret123` | Rendered literally (no environment variable access) |

**Test Method:** Unit tests with malicious inputs, verify no security bypass.

**Traceability:** Development Team needs secure templates → no injection, no file access, no code execution.

---

---

## 7. Test Execution Strategy

### Phase 1: Foundation (MVP)

**Test Deliverables:**
- [ ] Unit tests for `OrchestratorRenderer` (95%+ coverage)
- [ ] Unit tests for custom template functions (100% coverage)
- [ ] Integration tests for `PopulateTemplate` (backward compatibility)
- [ ] Regression tests for inline templates (all 62 still work)
- [ ] Performance benchmarks (< 5ms rendering)

**Exit Criteria:**
- All unit tests pass
- All integration tests pass
- Regression tests confirm zero breaking changes
- Performance benchmarks meet SLA
- Security tests confirm no vulnerabilities

**Test Automation:** CI/CD runs `make test` on every commit.

---

### Phase 2: High-Value Templates (12 templates)

**Test Deliverables:**
- [ ] Integration tests for 12 external templates
- [ ] Regression tests for 12 templates (inline vs external output comparison)
- [ ] Partial template tests (3 partials: TDD, exit gate, read section)
- [ ] Service layer integration tests (epic, feature, task actions)

**Exit Criteria:**
- All 12 external templates render correctly
- Regression tests confirm identical output (or improved formatting)
- Partials propagate changes to all using templates
- Service layer integration tests pass

**Test Automation:** CI/CD runs template validation: `shark config validate --templates` (Phase 3 feature, manual in Phase 2).

---

### Phase 3: Full Migration (62 templates) - OUT OF SCOPE FOR MVP

**Test Deliverables:**
- [ ] Integration tests for all 62 external templates
- [ ] Full regression test suite (all templates)
- [ ] Partial library tests (6-10 partials)
- [ ] Template validation command tests (`shark config validate --templates`)
- [ ] Migration tool tests (`shark config migrate-templates`)

**Exit Criteria:**
- All 62 templates externalized and tested
- Full regression suite passes
- Template validation command works
- Migration tool verified (if implemented)

---

### Continuous Testing

**CI/CD Pipeline:**
1. **Pre-commit:** `make fmt && make lint` (code formatting, static analysis)
2. **On commit:** `make test` (unit + integration tests)
3. **On PR:** Full test suite + regression tests + performance benchmarks
4. **On merge to main:** Deploy to staging, run full test suite

**Test Coverage Report:** `make test-coverage` generates HTML report, minimum 95% for new code.

**Performance Regression Detection:** Benchmark results compared to baseline, alert if > 10% slower.

---

---

## 8. Test Tooling & Infrastructure

### Test Framework

**Go Testing:** Standard `testing` package
**Assertions:** `github.com/stretchr/testify/assert`, `github.com/stretchr/testify/require`
**Mocking:** `github.com/stretchr/testify/mock` (if needed for repositories)

**Existing Pattern:** Follow `internal/templates/renderer_test.go` patterns
- Table-driven tests for multiple scenarios
- Subtests with `t.Run()`
- Helper functions for test setup

---

### Test Execution

**Commands:**
```bash
# Run all tests
make test

# Run specific package tests
go test -v ./internal/templates

# Run specific test
go test -v ./internal/templates -run TestOrchestratorRenderer_Render

# Run with coverage
make test-coverage

# Run benchmarks
go test -bench=BenchmarkOrchestratorRenderer -benchmem

# Run with race detector
go test -race ./...
```

---

### Test Data Management

**Test Fixtures Location:** `internal/templates/test_fixtures/`

**Test Template Organization:**
```
test_fixtures/
├── valid/
│   ├── basic.tmpl
│   ├── conditional.tmpl
│   └── tier.tmpl
├── partials/
│   └── _test_partial.tmpl
├── invalid/
│   ├── malformed.tmpl
│   └── missing_partial.tmpl
└── baseline/
    ├── task-ready_for_development-inline.txt
    └── task-ready_for_development-external.txt
```

**Cleanup:** Test fixtures committed to Git for reproducibility.

---

### Continuous Integration

**GitHub Actions Workflow:**
```yaml
name: Test Template Engine

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run tests
        run: make test
      - name: Check coverage
        run: make test-coverage
      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

---

---

## 9. Exit Gate Summary

### STANDARD Tier Exit Gate (Required for this feature)

**Coverage Requirements:**
- [x] Every acceptance criterion has test cases (100% AC coverage)
- [x] API contracts tested (OrchestratorRenderer, PopulateTemplate, functions, placeholders)
- [x] Integration points identified and tested (service layer, backward compatibility)

**Test Plan Completeness:**
- [x] AC test matrix with inputs/expected outputs/edge cases
- [x] API contract test cases for all Go interfaces
- [x] Component test strategy for OrchestratorRenderer, functions, placeholders
- [x] Integration scenarios for service layer, tiers, partials, errors
- [x] Performance and security testing approach

**Actionable for TDD:**
- [x] Test cases specific enough to write tests first
- [x] Expected outputs clearly defined
- [x] Edge cases identified for defensive coding
- [x] Error handling scenarios documented

**Traceability to User Goals:**
- [x] Template Author: Readable templates, conditionals, partials, validation
- [x] Development Team: Gradual migration, backward compatibility, testable
- [x] AI Orchestrator: Correct rendering, fast performance, reliable

---

## Appendix: Test Case Mapping to User Stories

| User Story | Acceptance Criteria | Test Cases | Integration Scenarios |
|------------|---------------------|------------|----------------------|
| US-E07-F30-001 | AC-1.1 to AC-1.5 | AC-1.1-T01 to AC-1.5-T04 | Scenario 1 (Backward Compatibility) |
| US-E07-F30-002 | AC-2.1 to AC-2.6 | AC-2.1-T01 to AC-2.6-T07 | Scenario 3 (Complexity Tier) |
| US-E07-F30-003 | AC-3.1 to AC-3.5 | AC-3.1-T01 to AC-3.5-T04 | Scenario 4 (Partial Propagation) |
| US-E07-F30-004 | AC-4.1 to AC-4.5 | AC-4.1-T01 to AC-4.5-T04 | Scenario 2 (Service Integration) |

---

**Document Status:** Complete for STANDARD tier
**Next Step:** Developer TDD implementation using this test plan
**Validation:** QA to verify test coverage upon implementation
