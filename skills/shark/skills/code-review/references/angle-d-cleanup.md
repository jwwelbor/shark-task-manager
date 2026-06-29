# Angle D: Reuse, Complexity, Idioms, Altitude

You are performing a cleanup and quality review. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

Substitute these values wherever the instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT.

---

## Part 1: Reuse / DRY

Before flagging duplication, **search first**:

```bash
grep -rn "<function-name>" PROJECT_ROOT --include="*.py"
grep -rn "<class-name>" PROJECT_ROOT --include="*.ts"
rg "<concept>" --type py --type ts PROJECT_ROOT
```

For each new utility, helper, or abstraction the diff introduces, check whether the codebase already has one. If it does:
- Document the existing symbol with its `file:line`
- Show a diff-ready refactor demonstrating how the new code would call it

Severity: blocker if a major reimplementation (>10 lines of equivalent logic); non-blocker if a small parallel utility.

## Part 2: Complexity & size hotspots

Run tooling where available:

```bash
# Python
uv run radon cc CHANGED_FILES --min=B 2>/dev/null || python -m radon cc CHANGED_FILES --min=B

# TypeScript / JS
npx eslint CHANGED_FILES --rule 'complexity: ["error", 10]' --no-eslintrc 2>/dev/null

# Go
gocyclo -over 10 CHANGED_FILES 2>/dev/null
```

Flag using these thresholds (defaults if project doesn't specify):
- Cyclomatic complexity > 10
- File > 500 lines
- Function > 50 lines
- Nesting depth ≥ 4

**Pre-existing complexity** above threshold → non-blocker (triage), unless this diff substantially expanded it.
**New complexity** above threshold → blocker.

## Part 3: Idiomatic language patterns

| Language | Check |
|---|---|
| Python | Comprehensions vs manual loops, context managers for resources, dataclasses vs dicts for structured data, `pathlib` over `os.path`, no string-formatted SQL, `isinstance` not `type()` |
| TypeScript | No `any` without justification, prefer `unknown` for untrusted input, discriminated unions over boolean flags, proper `Promise<T>` typing |
| Go | Error wrapping with context (`fmt.Errorf("...: %w", err)`), no swallowed errors, goroutine lifecycle, idiomatic struct embedding, `context.Context` propagation |
| All | No nested ternaries (prefer if/else), no commented-out code, no debug prints (`console.log`, `print("debug")`), no premature abstraction |

Show before/after when the idiomatic alternative isn't obvious.

## Part 4: Altitude (right depth of fix)

Check that each change is implemented at the right depth:
- Special cases layered on top of shared infrastructure (instead of fixing the shared code) → suspect
- A workaround in a caller for a known bug in a callee, rather than fixing the callee → non-blocker flag for discussion
- Fragile hardcoding of values that should be configuration or constants
- Fundamental changes to an existing pattern, workflow, storage shape, validation path, or dispatch path must be consciously justified. Ask whether the change would be defensible to another architect and whether the PR used the smallest change that achieves the stated behavior. If not, flag it as `ALTITUDE` or `ARCHITECTURE`.

Flag altitude issues as non-blockers with a concrete suggestion for where the fix should live.

## Part 5: Simplification

Flag unnecessary complexity introduced:
- Redundant/derivable state (a variable that can be computed from other state)
- Copy-paste with slight variation (should be a shared helper with a parameter)
- Dead code left behind after a refactor
- Overly deep nesting that a guard-clause would flatten

Name the simpler form explicitly.

---

## Output format

```json
{
  "reviewed_files": ["path/to/file.py"],
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "DRY|COMPLEXITY|IDIOM|ALTITUDE|ARCHITECTURE|SIMPLIFICATION",
      "summary": "one-sentence description",
      "diagnosis": "what is duplicated / too complex / non-idiomatic",
      "evidence": "grep result or code excerpt",
      "correction": "concrete cost if left as-is, and the preferred alternative"
    }
  ]
}
```

Correctness bugs always outrank cleanup — focus on genuine issues with concrete cost, not style preferences.

Return `{"reviewed_files": [...], "findings": []}` if nothing found. `reviewed_files` must list every changed file you opened or inspected. Return ONLY the JSON object, no other text.
