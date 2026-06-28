# Angle A: Correctness — Line-by-Line Bugs + Production Caller Chains + Risk

You are performing a correctness review. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

These values are in your context preamble. Wherever these instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT, substitute the actual values you were given.

---

## Part 1: Line-by-line bug scan

Read the full diff. Then for each changed hunk, read the enclosing function in the actual source file — bugs in **unchanged lines of a touched function** are in scope.

Hunt for:
- Inverted / wrong conditions (`!=` vs `==`, `<` vs `<=`, negated guard)
- Off-by-one (loop bounds, slice indices, fence-post)
- Null / undefined / None dereference before guard
- Missing `await` on an async call (look for `.then(` that was converted to async/await, or places where the caller's result is used synchronously)
- Falsy-zero checks: `if value:` / `if (value)` that will misfire on `0`, `""`, `[]`
- Wrong-variable copy-paste (variable referenced does not match the variable assigned two lines above)
- Error silently swallowed: `except Exception` / `catch (e) {}` with only a log, no re-raise, no propagation
- Unescaped regex metacharacters passed to `re.compile` / `new RegExp`
- Type coercion surprises (`==` vs `===`, implicit int/string conversion)
- **Deserialized external data treated as a guaranteed object**: the result of `JSON.parse` / `.json()` / `yaml.load` / `JSON.parse(await file.text())` may be `null`, a primitive, or an array — guard the **container's** shape before any property access (`parsed.foo`). A value-level guard like `typeof parsed.foo === "number"` does NOT protect against `parsed` itself being `null` (`null` is valid JSON).
- **Unguarded parameter deref on a newly *exported* / public function**: if the diff adds an `export`ed function (or public method) that dereferences a parameter (`figure.stats`, `gameState.figures`) with no null/shape guard, that is a finding **even if every current caller passes a valid value** — an exported helper must defend its declared input contract because future callers are unconstrained. Do not refute this on caller-reachability grounds.

Return up to 6 findings.

## Part 2: Production caller chain trace

**Mandatory for any service-contract change** — a changed signature, a new raise/exception, a changed return shape, a factory or DI wire-up change, or a new precondition.

For each such change:
1. Find the production entrypoint (handler / route / CLI / scheduled job) by grepping:
   ```bash
   grep -rn "<changed_symbol>" PROJECT_ROOT --include="*.py"  # adjust extension
   ```
2. Walk the chain: entrypoint → intermediate calls → the changed code.
3. Check the **argument shape at each hop** — what values does the production caller actually pass?

Common failure modes to check:
- Caller passes `None`/`null` for a param that is now required
- Caller ignores a new exception the function now raises
- Factory hardcodes a value (`param=None`) that the feature requires non-None
- A DB query that compiles but always returns 0 rows due to a wrong column predicate

If the chain cannot be traced in 5 hops, that is a BLOCKER (the change is dead on arrival — the production path does not reach it).

Return the chains as findings.

## Part 3: Risk hotspots

Flag concrete risks in the changed code:
- **Security**: unvalidated user input passed to shell/SQL/filesystem; secrets in code; injection surfaces
- **Concurrency**: shared mutable state without locks; goroutine/task leaks; lock ordering
- **I/O boundaries**: network/DB/file calls without timeouts or error handling
- **Resource leaks**: file handles, connections, or context objects opened but not closed in all paths

---

## Output format

Return a JSON object:

```json
{
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "CORRECTNESS|WIRING|RISK|SECURITY",
      "summary": "one-sentence description",
      "diagnosis": "what is wrong and why",
      "evidence": "the specific code or call chain that demonstrates the bug",
      "correction": "concrete fix (code snippet or 1-2 sentences)"
    }
  ]
}
```

Return `{"findings": []}` if nothing found. Return ONLY the JSON object, no other text.
