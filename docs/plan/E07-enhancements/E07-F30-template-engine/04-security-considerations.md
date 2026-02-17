# Security Considerations: External Template Engine for Orchestrator Instructions

**Feature:** E07-F30
**Complexity Tier:** STANDARD
**Version:** 1.0
**Last Updated:** 2026-02-14

---

## Overview

This document analyzes security risks introduced by the external template engine and defines mitigations. Since orchestrator instruction templates control AI agent behavior, template injection or malicious templates could potentially compromise development workflows.

**Risk Summary:**
- **Template Injection:** LOW (placeholders from trusted sources only)
- **Arbitrary File Access:** LOW (restricted to templates/ directory)
- **Code Execution:** NONE (text/template has no code execution)
- **Denial of Service:** LOW (precompiled templates, no recursive includes)
- **Supply Chain:** MEDIUM (templates in version control, code review required)

---

## 1. Template Injection Attacks

### Risk Description

**Attack Vector:** Malicious placeholder values could inject template directives (e.g., `{{.malicious_var}}` containing `{{template "evil"}}`)

**Impact:** HIGH if successful
- Execute arbitrary template logic
- Access sensitive data from other placeholders
- Bypass conditional logic

**Likelihood:** LOW

**Why Low:**
- Placeholders come from **trusted sources only**:
  - Database fields (epic/feature/task titles, keys, status)
  - Config-driven metadata (complexity_tier, agent_type)
  - Relationship data (related docs/tasks keys)
- **No user-provided input** in placeholders
- All database values validated before storage

---

### Mitigation

**Primary Defense: Trusted Data Sources**

Placeholder values originate from:
1. **Database Records:**
   - Epic/feature/task titles, keys, status (validated at creation)
   - Stored via parameterized queries (SQL injection prevention)
   - No user input bypasses validation

2. **Config-Driven Metadata:**
   - `complexity_tier`: Enum values (SIMPLE, STANDARD, COMPLEX) from .sharkconfig.json
   - `agent_type`: Validated agent types from workflow config
   - Admin-controlled configuration (not end-user input)

3. **Relationship Data:**
   - Related doc paths: From `documents` table (validated file paths)
   - Related task/feature/epic keys: From relationship tables (validated foreign keys)

**Secondary Defense: Template Parsing**

Go's `text/template` parses templates **once at startup**. Runtime data (placeholders) cannot inject new template directives:

```go
// Template compiled at startup (safe)
tmpl := template.Must(template.New("task").Parse("Task: {{.task_id}}"))

// Placeholder value at runtime (cannot inject directives)
vars := map[string]string{
    "task_id": "{{template 'evil'}}",  // Rendered as literal string, NOT executed
}

// Output: "Task: {{template 'evil'}}" (safe literal)
```

**Result:** Template injection via placeholders is **impossible** with precompiled templates.

---

### Code Review Requirements

**Template Files:** All `.tmpl` files must be code-reviewed before merge (same as Go code)

**Review Checklist:**
- [ ] No hardcoded secrets or sensitive data
- [ ] No references to external files outside `templates/`
- [ ] Conditionals use validated placeholder variables only
- [ ] No recursive partial includes (depth > 3)

---

## 2. Arbitrary File Access

### Risk Description

**Attack Vector:** Template engine could read arbitrary files via malicious template references.

**Impact:** MEDIUM if successful
- Read sensitive files (e.g., `/etc/passwd`, `.env`, `shark-tasks.db`)
- Expose database credentials
- Read user data from file system

**Likelihood:** LOW

**Why Low:**
- Templates loaded from **restricted directory only** (`templates/`)
- `ParseGlob()` restricted to `*.tmpl` pattern
- No dynamic template paths from external input

---

### Mitigation

**Primary Defense: Directory Restriction**

Templates loaded only from configured directory (default: `templates/`):

```go
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
    // Only load templates from templateDir (admin-controlled)
    tmpl, err := template.New("orchestrator").
        Funcs(orchestratorFuncs()).
        ParseGlob(filepath.Join(templateDir, "*/*.tmpl"))  // Restricted pattern

    if err != nil {
        return nil, fmt.Errorf("failed to parse templates: %w", err)
    }

    return &OrchestratorRenderer{templates: tmpl}, nil
}
```

**Path Traversal Prevention:**
- `templateDir` sourced from `.sharkconfig.json` (admin-controlled, not user input)
- `ParseGlob()` pattern `*/*.tmpl` prevents parent directory access (`../` not matched)
- Template names validated at runtime (no `..` allowed)

**Secondary Defense: No Dynamic Paths**

Template names come from **static config only** (`.sharkconfig.json`):

```json
{
  "ready_for_development": {
    "orchestrator_action": {
      "instruction_template": "task/ready_for_development.tmpl"  // Static, not user input
    }
  }
}
```

**Result:** Arbitrary file access is **not possible** without admin access to `.sharkconfig.json`.

---

### Configuration Validation

**Required:**
- Validate `template_directory` is relative path or absolute within project root
- Reject paths with `..` (parent directory traversal)
- Log warning if `template_directory` is outside project root

**Example:**
```go
func validateTemplateDirectory(dir string) error {
    if strings.Contains(dir, "..") {
        return fmt.Errorf("template directory cannot contain '..'")
    }

    if filepath.IsAbs(dir) {
        projectRoot := getProjectRoot()
        if !strings.HasPrefix(dir, projectRoot) {
            log.Printf("WARNING: template directory outside project root: %s", dir)
        }
    }

    return nil
}
```

---

## 3. Code Execution Risks

### Risk Description

**Attack Vector:** Template engine could execute arbitrary code via malicious template functions.

**Impact:** CRITICAL if successful
- Execute shell commands
- Modify file system
- Access network resources
- Compromise host system

**Likelihood:** NONE

**Why None:**
- Go's `text/template` has **no code execution capability**
- Custom functions explicitly defined, no dynamic function loading
- No shell access, no file writes, no network calls

---

### Mitigation

**Primary Defense: text/template Limitations**

Go's `text/template` package is **designed for safe text generation**:
- No `eval()` or code execution primitives
- No system calls
- No file system writes
- No network access

**Secondary Defense: Restricted Function Map**

Only explicitly defined functions are available in templates:

```go
func orchestratorFuncs() template.FuncMap {
    return template.FuncMap{
        // SAFE: Comparison functions (no side effects)
        "eq":      func(a, b interface{}) bool { return a == b },
        "ne":      func(a, b interface{}) bool { return a != b },

        // SAFE: String helpers (no side effects)
        "join":    strings.Join,
        "isEmpty": func(s string) bool { return strings.TrimSpace(s) == "" },
        "trim":    strings.TrimSpace,

        // SAFE: Convenience helpers (no side effects)
        "isSimple":   func(tier string) bool { return tier == "SIMPLE" },
        "isStandard": func(tier string) bool { return tier == "STANDARD" },
        "isComplex":  func(tier string) bool { return tier == "COMPLEX" },
    }
}
```

**Prohibited Functions:**
- ❌ File I/O (`os.Open`, `ioutil.WriteFile`)
- ❌ Shell execution (`exec.Command`, `os/exec`)
- ❌ Network access (`http.Get`, `net.Dial`)
- ❌ Dynamic code loading (`plugin.Open`, `reflect.Call`)

**Result:** Code execution is **architecturally impossible** with `text/template` and restricted function map.

---

## 4. Denial of Service (DoS)

### Risk Description

**Attack Vector:** Malicious templates could cause resource exhaustion:
- Infinite loops
- Deeply nested conditionals
- Recursive partial includes
- Large output generation

**Impact:** MEDIUM if successful
- High CPU usage
- Memory exhaustion
- Application hang/crash
- Degraded performance for all users

**Likelihood:** LOW

**Why Low:**
- Templates precompiled (no parse-time DoS)
- No user-provided templates at runtime
- Go template engine has built-in recursion limits

---

### Mitigation

**Primary Defense: Precompilation**

Templates parsed **once at startup**, not per request:
- Malicious syntax caught at startup (fail-fast)
- No runtime parsing overhead
- DoS limited to startup phase (single occurrence)

**Secondary Defense: Go Template Limits**

Go's `text/template` has built-in protections:
- **Recursion limit:** ~1000 depth (prevents stack overflow)
- **Execution timeout:** No built-in timeout (manual mitigation needed)

**Recommended: Execution Timeout (Phase 4 Enhancement)**

Add execution timeout for template rendering:

```go
func (r *OrchestratorRenderer) RenderWithTimeout(templateName string, vars map[string]string, timeout time.Duration) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    resultCh := make(chan string, 1)
    errCh := make(chan error, 1)

    go func() {
        result, err := r.Render(templateName, vars)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- result
    }()

    select {
    case result := <-resultCh:
        return result, nil
    case err := <-errCh:
        return "", err
    case <-ctx.Done():
        return "", fmt.Errorf("template execution timeout after %v", timeout)
    }
}
```

**Template Complexity Limits (Phase 4 Enhancement):**

Lint templates for complexity:
- Max nesting depth: 5 levels
- Max partial includes: 10 per template
- Max output size: 10KB per template

---

### Code Review Requirements

**Template Complexity Checklist:**
- [ ] No recursive partial includes (e.g., `_partial_a.tmpl` includes `_partial_b.tmpl` which includes `_partial_a.tmpl`)
- [ ] Nesting depth < 5 levels
- [ ] Conditionals reference bounded data (no unbounded loops)
- [ ] Partials limited to < 10 includes per template

---

## 5. Supply Chain Security

### Risk Description

**Attack Vector:** Malicious templates committed to version control could compromise development workflows.

**Impact:** HIGH if successful
- Misleading AI agent instructions
- Incorrect workflow execution
- Data corruption via bad orchestrator actions

**Likelihood:** MEDIUM

**Why Medium:**
- Templates stored in Git (same supply chain as code)
- Code review required but templates may be less scrutinized than Go code
- Template syntax less familiar to reviewers

---

### Mitigation

**Primary Defense: Code Review**

**Requirement:** All `.tmpl` files must go through pull request review (same as `.go` files)

**Review Checklist:**
- [ ] Template purpose clear and documented
- [ ] No hardcoded secrets or credentials
- [ ] Conditionals use validated placeholder variables
- [ ] Output matches intended orchestrator action
- [ ] No suspicious partial includes or file references

**Secondary Defense: Template Validation**

**Command:** `shark config validate --templates` (Phase 3)

Validates:
- Syntax correctness (parseability)
- All referenced placeholders exist
- All partial templates exist
- No recursive includes

**CI/CD Integration:**

```yaml
# .github/workflows/validate.yml
- name: Validate Templates
  run: ./bin/shark config validate --templates
```

**Tertiary Defense: Signed Commits**

Require GPG-signed commits for template changes (optional, high-security environments).

---

## 6. Configuration Security

### Risk Description

**Attack Vector:** Malicious `.sharkconfig.json` could reference malicious templates.

**Impact:** HIGH if successful
- Load templates from attacker-controlled directory
- Execute malicious template logic
- Compromise workflow integrity

**Likelihood:** LOW

**Why Low:**
- `.sharkconfig.json` requires admin/developer access to modify
- File committed to version control (code review required)
- Changes visible in Git history

---

### Mitigation

**Primary Defense: Git Version Control**

`.sharkconfig.json` changes require:
1. Commit to version control
2. Pull request review
3. CI validation

**Secondary Defense: Configuration Validation**

Validate `.sharkconfig.json` on load:

```go
func loadConfig(path string) (*Config, error) {
    config, err := parseConfig(path)
    if err != nil {
        return nil, err
    }

    // Validate template directory
    if err := validateTemplateDirectory(config.TemplateDirectory); err != nil {
        return nil, fmt.Errorf("invalid template directory: %w", err)
    }

    // Validate all instruction_template references
    for status, metadata := range config.StatusMetadata {
        if action := metadata.OrchestratorAction; action != nil {
            if strings.HasSuffix(action.InstructionTemplate, ".tmpl") {
                if err := validateTemplateReference(action.InstructionTemplate); err != nil {
                    return nil, fmt.Errorf("invalid template reference in status %s: %w", status, err)
                }
            }
        }
    }

    return config, nil
}

func validateTemplateReference(ref string) error {
    // Reject absolute paths
    if filepath.IsAbs(ref) {
        return fmt.Errorf("template reference must be relative: %s", ref)
    }

    // Reject parent directory traversal
    if strings.Contains(ref, "..") {
        return fmt.Errorf("template reference cannot contain '..': %s", ref)
    }

    // Ensure .tmpl extension
    if !strings.HasSuffix(ref, ".tmpl") {
        return fmt.Errorf("template reference must end with .tmpl: %s", ref)
    }

    return nil
}
```

---

## 7. Secrets Management

### Risk Description

**Attack Vector:** Templates could inadvertently expose secrets in rendered output.

**Impact:** CRITICAL if successful
- Database credentials in orchestrator instructions
- API keys visible in logs
- Auth tokens leaked to AI agents

**Likelihood:** LOW

**Why Low:**
- Placeholders from database only (no secrets in database)
- No environment variable access in templates
- No file reads from templates

---

### Mitigation

**Primary Defense: No Secret Placeholders**

Placeholder sources exclude secrets:
- ❌ Environment variables (no `os.Getenv()` function)
- ❌ Config secrets (not included in placeholder maps)
- ❌ File contents (no file read functions)

**Secondary Defense: Code Review**

**Template Review Checklist:**
- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No references to secret files (.env, credentials.json)
- [ ] No debugging output that could leak data

**Tertiary Defense: Secret Scanning (CI)**

Use `gitleaks` or similar to scan `.tmpl` files for secrets:

```yaml
# .github/workflows/security.yml
- name: Scan for Secrets
  uses: trufflesecurity/trufflehog@main
  with:
    path: ./templates/
```

---

## 8. Output Validation

### Risk Description

**Attack Vector:** Malicious templates could generate misleading or harmful orchestrator instructions.

**Impact:** MEDIUM if successful
- AI agents receive incorrect instructions
- Development workflows disrupted
- Data corruption from bad actions

**Likelihood:** LOW

**Why Low:**
- Templates version-controlled and code-reviewed
- Output is text only (no code execution)
- AI agents validate instructions before acting

---

### Mitigation

**Primary Defense: Code Review**

**Template Output Review:**
- [ ] Instructions match intended workflow action
- [ ] No misleading or ambiguous directives
- [ ] EXIT GATE criteria clear and testable
- [ ] References valid file paths and entity keys

**Secondary Defense: Testing**

**Regression Tests:** Capture expected output for all templates with test data.

**Example:**
```go
func TestTemplate_ReadyForDevelopment_Output(t *testing.T) {
    renderer := setupTestRenderer(t)

    vars := map[string]string{
        "task_id": "E07-F30-001",
        "title": "Test Task",
        "file_path": "docs/plan/test.md",
    }

    output, err := renderer.Render("task/ready_for_development.tmpl", vars)
    if err != nil {
        t.Fatalf("Render() error = %v", err)
    }

    // Validate critical sections present
    mustContain(t, output, "Launch developer")
    mustContain(t, output, "E07-F30-001")
    mustContain(t, output, "shark task next-status")
}
```

---

## Security Checklist

### Development Phase

**Code Review (MANDATORY for all .tmpl files):**
- [ ] No hardcoded secrets or credentials
- [ ] No references to files outside `templates/`
- [ ] Conditionals use validated placeholder variables only
- [ ] No recursive partial includes
- [ ] Output matches intended orchestrator action
- [ ] Nesting depth < 5 levels
- [ ] Partials limited to < 10 includes per template

**Testing:**
- [ ] Template rendering unit tests pass
- [ ] Regression tests verify output unchanged
- [ ] Error handling tests cover missing templates
- [ ] Performance tests verify < 5ms render time

---

### Deployment Phase

**CI/CD Validation:**
- [ ] `shark config validate --templates` passes
- [ ] Secret scanning passes (no hardcoded secrets)
- [ ] Lint checks pass (complexity limits)
- [ ] All tests pass (unit + integration)

**Configuration:**
- [ ] `.sharkconfig.json` validated (no path traversal)
- [ ] `template_directory` within project root
- [ ] All template references valid (files exist)

---

### Operational Phase

**Monitoring:**
- [ ] Log template rendering errors
- [ ] Alert on repeated rendering failures
- [ ] Monitor render time (< 5ms threshold)

**Audit:**
- [ ] Review template changes in Git history monthly
- [ ] Scan for new secrets in templates quarterly
- [ ] Validate template complexity limits quarterly

---

## Risk Summary Table

| Risk | Impact | Likelihood | Mitigation | Residual Risk |
|------|--------|------------|------------|---------------|
| Template Injection | High | Low | Trusted data sources, precompiled templates | **Low** |
| Arbitrary File Access | Medium | Low | Directory restriction, static paths | **Low** |
| Code Execution | Critical | None | text/template limitations, restricted functions | **None** |
| Denial of Service | Medium | Low | Precompilation, Go limits, code review | **Low** |
| Supply Chain | High | Medium | Code review, validation, signed commits | **Medium** |
| Configuration Tampering | High | Low | Git version control, validation | **Low** |
| Secret Exposure | Critical | Low | No secret placeholders, code review | **Low** |
| Output Manipulation | Medium | Low | Code review, regression tests | **Low** |

**Overall Risk Level:** **LOW** (with code review and validation in place)

---

## Security Recommendations

### Immediate (Phase 1 - Foundation)

1. ✅ Use `text/template` (not `html/template`)
2. ✅ Restrict template directory to `templates/`
3. ✅ Validate `.sharkconfig.json` template references
4. ✅ Precompile templates at startup (fail-fast)
5. ✅ Code review all `.tmpl` files

### Short-Term (Phase 2 - High-Value Templates)

6. ✅ Add template validation to CI/CD
7. ✅ Implement regression tests for template output
8. ✅ Document security review checklist

### Long-Term (Phase 3 - Full Migration)

9. ⏳ Add `shark config validate --templates` command
10. ⏳ Implement secret scanning for `.tmpl` files
11. ⏳ Add template complexity linting

### Future Enhancements (Phase 4)

12. ⏳ Add execution timeout (5 second max)
13. ⏳ Implement template complexity limits enforcement
14. ⏳ Add template output size limits (10KB max)
15. ⏳ Consider signed commits for template changes

---

**Security Review Status:** Approved for Phase 1 Implementation
**Next Steps:** Implement security validations in OrchestratorRenderer
