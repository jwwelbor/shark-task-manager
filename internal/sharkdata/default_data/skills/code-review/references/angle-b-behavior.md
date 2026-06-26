# Angle B: Removed Behavior + SOLID Compliance

You are performing a removed-behavior and architecture review. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

These values are in your context preamble. Substitute them wherever these instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT.

---

## Part 1: Removed-behavior audit

For every line the diff **deletes or replaces**, identify the invariant or behavior it enforced, then search the new code for where that invariant is re-established. If you cannot find it, that is a candidate finding.

Classes of removed behavior to look for:
- **Removed guards**: a null check, a bounds check, a permission check, a rate-limit
- **Dropped error paths**: a try/except or try/catch block that was protecting a call
- **Narrowed validation**: a validator that now accepts fewer cases (or more cases than it should)
- **Deleted test covering a real case**: a test removed alongside an implementation change — the AC may now be unverified
- **Removed rollback / cleanup code**: a `finally` block, a context manager `__exit__`, a defer/defer() call
- **Weakened contract**: a function that previously raised on bad input now silently returns a default

For each: read the new code to verify the invariant is NOT re-established elsewhere. Only flag it if the behavior is genuinely dropped, not moved.

## Part 2: SOLID compliance

Read CHANGED_FILES and evaluate each modified class/module against SOLID:

**Single Responsibility (SRP)**: Does each class/module have one clear job? Flag classes that grew to own multiple unrelated concerns in this diff.

**Open/Closed (OCP)**: Is the change extensible without modifying stable code paths? Flag if new behavior was added by modifying an existing function rather than extending a strategy/protocol.

**Liskov Substitution (LSP)**: If the diff modifies a subclass or protocol implementor, does it maintain the parent's contract? Flag if a subclass now raises where the base doesn't, or returns a narrower type.

**Interface Segregation (ISP)**: Flag large interfaces forced onto classes that only use a subset, if this diff introduces or widens such an interface.

**Dependency Inversion (DIP)**: Flag if the diff introduces a direct instantiation of a concrete class where an abstraction should be used (e.g., `SomeService()` inline instead of injected dependency).

For each SOLID violation: provide `file:line`, the principle violated, evidence (the specific code), and a concrete refactor.

---

## Output format

```json
{
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "REMOVED-BEHAVIOR|SRP|OCP|LSP|ISP|DIP",
      "summary": "one-sentence description",
      "diagnosis": "what invariant or principle is violated",
      "evidence": "specific code showing the violation",
      "correction": "concrete fix or refactor"
    }
  ]
}
```

Return `{"findings": []}` if nothing found. Return ONLY the JSON object, no other text.
