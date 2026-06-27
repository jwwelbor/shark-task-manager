# Angle C: Cross-File Tracer + Structural Sibling Check

You are performing a cross-file contract and structural review. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

Substitute these values wherever the instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT.

---

## Part 1: Cross-file contract tracer

For each function/method the diff changes, find its callers:

```bash
grep -rn "<symbol>" PROJECT_ROOT --include="*.py"   # adjust extension as needed
grep -rn "<symbol>" PROJECT_ROOT --include="*.ts"
```

For each caller, check whether the change breaks the call site:

- **New precondition**: the function now requires a non-None/non-empty argument — does every caller satisfy it?
- **Changed return shape**: the function now returns a different type or structure — do callers handle it?
- **New exception**: the function now raises where it previously returned a value — do callers catch it?
- **Timing / ordering dependency**: the function must now be called after some initialization — is every caller in the correct position?
- **Callees**: does a parallel change in the same PR make an existing call unsafe? (two changes that are each fine in isolation but conflict together)

Pay special attention to functions that **now raise instead of returning a value** — find every caller and verify exception handling.

## Part 2: Structural sibling check

**Mandatory when the diff adds a class that implements a protocol, ABC, or interface.**

When the diff introduces such a class:

1. Find all other implementors:
   ```bash
   # Python
   grep -rn "class.*(<ProtocolName>)" PROJECT_ROOT --include="*.py"
   grep -rn "from.*import.*<ProtocolName>" PROJECT_ROOT --include="*.py"
   # TypeScript
   grep -rn "implements <InterfaceName>" PROJECT_ROOT --include="*.ts"
   grep -rn "extends <BaseName>" PROJECT_ROOT --include="*.ts"
   ```

2. Count the siblings. Check whether they share a common base class that provides real code (not just the protocol signature).

3. Compare implementations: do ≥3 siblings have near-identical method bodies, constructor patterns, or error handling?

**Severity rules:**
- **BLOCKER**: ≥3 siblings implement the same protocol with near-identical method bodies but **no** shared base class. This is the most expensive technical debt pattern — every future change requires N parallel edits.
- **Non-blocker (triage)**: ≥3 siblings with a shared base exist, but the new class in this diff skips the base without justification.
- **Non-blocker (triage)**: 2 siblings with identical implementations — not yet worth a base class, but flag for monitoring.

Include a sibling inventory table in your findings when this check triggers:

| Protocol/ABC | Sibling count | Shared base? | New class uses base? | Identical methods |
|---|---|---|---|---|

If the diff does not add a class implementing a protocol/ABC/interface, note "Structural sibling check: not applicable" and skip this part.

---

## Output format

```json
{
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "CONTRACT|SIBLING",
      "summary": "one-sentence description",
      "diagnosis": "what contract is broken or what sibling pattern is violated",
      "evidence": "grep output or code showing the issue",
      "correction": "concrete fix"
    }
  ]
}
```

Return `{"findings": []}` if nothing found. Return ONLY the JSON object, no other text.
