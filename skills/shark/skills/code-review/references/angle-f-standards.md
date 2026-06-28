# Angle F: Standards Crosswalk + Coding Conventions

You are performing a standards and conventions review. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

Substitute these values wherever the instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT.

---

## Citation discipline (MANDATORY)

Every finding in this angle **must cite** the specific section of the project's standards document that is violated.

Format: `{file:line} — {rule} — {standards-doc §X.Y "quoted clause"}`

**No citation = the finding is opinion, not a standards violation.** If no standards section covers a finding, mark it "opinion-only" and downgrade to a nit. Do not fabricate citations.

---

## Step 1: Load the coding standards

Your context preamble includes `CODING_STANDARDS_PATH`. Use it:

- If `CODING_STANDARDS_PATH` is set: **read that file now**. Every finding in this angle must cite a specific section from it. Format: `{file:line} — {rule} — {§X.Y "quoted clause"}`. No citation = opinion-only, downgrade to nit.
- If `CODING_STANDARDS_PATH` is not set (null / "not found"): every standards violation downgrades to opinion-level (nit). Note this at the top of your findings. Do not fabricate a path or invent sections.

Do not re-search for the standards file — it was already located by `get_diff.sh`. Trust the value in your context.

## Step 2: Standards crosswalk

Read the diff and each changed file. For each potential violation, find the exact section in the standards doc before flagging.

Build a table:

| finding | standards doc | section | quoted clause |
|---|---|---|---|
| `path/to/file.py:42` — broad `except` | coding-standards.md | §4.2 Error Handling | "All exceptions must be typed and re-raised or logged with severity ≥ ERROR" |

## Step 3: Naming conventions

Check changed code against the project's naming conventions for:
- Variables and constants
- Functions and methods
- Classes and modules
- Files and directories

Flag deviations with the relevant standard. If no naming standard is documented, skip.

## Step 4: Error handling patterns

Beyond the SOLID/correctness angles, check for project-specific error handling requirements:
- **Broad catch-alls introduced by the diff**: any new `except Exception`, `catch (e) {}`, `catch (Throwable)`, or bare `except:` without a stated diagnose-layer justification per the project's error-handling standard is a **blocker**.
- Custom exception hierarchies — are new exceptions in the right class hierarchy?
- Error propagation — does the project require specific patterns (wrap-and-re-raise, structured error returns, etc.)?

## Step 5: Type safety

- Missing type annotations on new public functions/methods (flag if the project requires them)
- Use of `Any` / `object` / untyped collections where the type is knowable
- Optional chaining / null-safety violations

## Step 6: Logging standards

- PII or sensitive data in log messages
- Wrong log level (debug-level noise at WARNING, real errors at INFO)
- Missing correlation IDs or request context if required by the project
- `print()` statements that should be logger calls

## Step 7: Security

Flag in changed code only (not pre-existing):
- User input passed directly to shell, SQL, filesystem, or eval
- Hardcoded credentials or API keys
- Missing input validation at a system boundary (HTTP handler, CLI argument, file read, **file upload / import handler**)
- **Deserialized external input accessed without a shape check**: `JSON.parse`, `.json()`, `yaml.load`, or a parsed uploaded file whose result is used as an object before validating it is a non-null object of the expected shape (valid JSON includes `null`, primitives, and arrays — any of which break member access)
- Insecure defaults (no TLS verification, permissive CORS, etc.)

---

## Output format

```json
{
  "findings": [
    {
      "file": "path/to/file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "CODING-STANDARD|NAMING|ERROR-HANDLING|TYPES|LOGGING|SECURITY",
      "summary": "one-sentence description",
      "diagnosis": "which standard is violated",
      "evidence": "standards-doc §X.Y: \"quoted clause\" vs the actual code",
      "correction": "concrete fix"
    }
  ]
}
```

Return `{"findings": []}` if nothing found. Return ONLY the JSON object, no other text.
