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

## Part 3: Serialization & state-field round-trip

**Mandatory when the diff touches a serialize/deserialize pair OR reads a shared state field.** This catches a class of cross-function contract break that Part 1 misses entirely: the two sides never call each other — they communicate through a serialized key or a shared state object, so signature-based caller tracing never connects them.

1. **Export ↔ import key symmetry.** For any write/read pair the diff touches — `downloadJson`/`importJson`, `toJSON`/`fromJSON`, `encode`/`decode`, `serialize`/`deserialize`, `persist`/`load` — list the keys each side writes and reads. Every key the producer writes must be read under the **same name/path** by the consumer, and vice versa. A key written under one path and read under a different path is a **BLOCKER** (the round-trip silently drops or misreads data).

2. **State-field producer→consumer trace.** For any state field the diff *reads* (e.g. `state.warband.campaign.games`), grep for where that exact path is *written*:
   ```bash
   grep -rn "campaign.games" PROJECT_ROOT --include="*.js"   # adjust path/extension
   ```
   Confirm the writer and the reader use the **same object path**. A reader pointed at a path that no writer ever populates (so it always sees the default/empty value) is a **BLOCKER** — the logic depending on it is dead code. Watch specifically for two parallel state trees (e.g. `state.campaign` vs `state.warband.campaign`) where one is written and the other is read.

Include both the write-site `file:line` and the read-site `file:line` in any finding. If the diff touches neither a serialize/deserialize pair nor a shared-state read, note "Round-trip check: not applicable" and skip.

---

## Output format

```json
{
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "CONTRACT|SIBLING|ROUNDTRIP",
      "summary": "one-sentence description",
      "diagnosis": "what contract is broken, what sibling pattern is violated, or which write/read paths diverge",
      "evidence": "grep output or code showing the issue",
      "correction": "concrete fix"
    }
  ]
}
```

Return `{"findings": []}` if nothing found. Return ONLY the JSON object, no other text.
