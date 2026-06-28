---
name: code-review
description: >
  Multi-angle parallel code review. Six specialist subagents run in parallel (A: line-by-line bugs +
  production caller chains, B: removed behavior + SOLID, C: cross-file contracts + structural sibling
  check, D: reuse/complexity/idioms, E: tests design + counter-factual, F: standards crosswalk with
  citations), then a consolidator verifies and ranks findings into a PASS / PASS-with-triage / FAIL
  report with Blocker / Non-blocker / Nit triage. Triggered by /code-review. Flags: --fix (apply
  one-liner safe fixes), --comment (post as inline GitHub PR comments). Prompts live in
  references/angle-*.md and references/consolidator.md — readable on any platform even without the
  Workflow tool.
---

# Code Review

Six parallel review angles, then a consolidating final pass. Angle prompts are in `references/` — the JS workflow reads them at runtime, and the fallback path uses them directly.

---

## Primary path (Workflow tool available)

### 1. Locate the skill directory

```bash
skill_dir=$(find "$HOME/.claude" -name "SKILL.md" -path "*/code-review/SKILL.md" 2>/dev/null | head -1 | xargs dirname)
echo "skill_dir=$skill_dir"
```

### 2. Capture the diff

```bash
bash "$skill_dir/scripts/get_diff.sh"
```

Parse the JSON output for `diff_path`, `changed_files`, `project_root`, `coding_standards_path`, `branch`, and `review_output_path`. Keep `review_output_path` — you write the final report there in step 4.

### 3. Launch the workflow

Parse the effort token from the command arguments (`low` | `medium` | `high` | `xhigh` | `max`) if the user passed one (e.g. `/code-review high`). Pass it through as `effort` so the angle agents and consolidator reason at that depth. If no token is given, omit `effort` to inherit session effort. `low`/`medium` emit fewer, high-confidence findings; `high`→`max` broaden coverage (the right default for a pre-merge gate).

```javascript
Workflow({
  scriptPath: `${skill_dir}/scripts/review_workflow.js`,
  args: {
    diff_path,
    changed_files,
    project_root,
    skill_dir,
    coding_standards_path,   // from get_diff.sh output — null if not found
    effort,                  // from the /code-review <level> token — omit to inherit session effort
    // optional — include if available:
    // task_spec_path: "/path/to/task.md",
    // feature_prd_path: "/path/to/prd.md",
    // acceptance_criteria: [{ ac_id: "AC-1", text: "..." }]
  }
})
```

The workflow runs all 6 angle agents in parallel, then the consolidator. The return value is the final markdown report — print it to the user.

### 4. Persist the report

Always save the report to `review_output_path` (from `get_diff.sh`) so the gate is auditable — this is the **overall** multi-angle review, distinct from any per-feature AC-checklist review in the same `code_review/` folder. Do this before handling `--fix` / `--comment`.

```bash
mkdir -p "$(dirname "<review_output_path>")"
```

Then Write the file with a self-describing header followed by the report body:

```markdown
# Overall Code Review — <branch>

**Generated:** <YYYY-MM-DD> · **Tool:** `/code-review` (6-angle automated) · **Diff:** `main...HEAD` · **Effort:** <effort or "session default">
**Verdict:** <PASS | PASS-with-triage | FAIL — from the report's Executive Summary>

---

<full markdown report returned by the workflow>
```

Tell the user where it was saved (`review_output_path`). If the write fails (e.g. read-only path), print the report inline and note that persistence was skipped — never drop the report.

---

## Fallback path (no Workflow tool, or non-Claude-Code platform)

If the Workflow tool is unavailable, run the angles manually:

### 1. Locate skill_dir and capture diff (same as above)

### 2. Read each angle prompt file

Read `references/angle-a-bugs.md` through `references/angle-f-standards.md` and `references/consolidator.md`. Each file is self-contained.

### 3. Launch 6 parallel Agent calls in a single message

Include this context preamble in each agent's prompt, substituting actual values:
```
CONTEXT:
- DIFF_PATH:              <diff_path>
- CHANGED_FILES:          <list>
- PROJECT_ROOT:           <project_root>
- CODING_STANDARDS_PATH:  <coding_standards_path, or "not found">
```

Then append the full contents of the corresponding `references/angle-X.md` file.

**Send all 6 Agent calls in a single response** so they run concurrently. Wait for all 6 to return.

### 4. Run the consolidator

Once all angles return, build a final agent prompt:
- Preamble with DIFF_PATH and ALL_FINDINGS (the combined JSON arrays)
- Full contents of `references/consolidator.md`

The consolidator returns the markdown report.

### 5. Persist the report

Save it to `review_output_path` exactly as in the primary path's step 4 (header + report body), then tell the user where it landed.

---

## Flags

### `--fix`
After presenting the report, apply one-liner safe fixes for each CONFIRMED/BLOCKER finding:
- Wrong variable name (copy-paste error)
- Missing `await` on an async call
- Inverted boolean condition
- Off-by-one in a loop bound

Skip structural refactors or multi-file changes. Announce each fix before applying it.

### `--comment`
After presenting the report, post findings as inline PR review comments:

```bash
# Summary comment
gh pr review --comment -b "<verdict paragraph>"

# Per-finding inline comments
gh api repos/:owner/:repo/pulls/:pull_number/comments \
  --method POST \
  -f body="**[rule]** finding summary\n\n> evidence\n\nCorrection: ..." \
  -f path="<file>" \
  -F line=<line> \
  -f side="RIGHT"
```

Use `gh pr view --json number` and `gh repo view --json owner,name` to get the required values.

---

## Angle reference

| File | Covers |
|---|---|
| `references/angle-a-bugs.md` | Line-by-line bugs, production caller chain trace, risk hotspots |
| `references/angle-b-behavior.md` | Removed behavior / dropped guards, SOLID compliance |
| `references/angle-c-sibling.md` | Cross-file contract breaks, structural sibling check, serialization/state-field round-trip (export↔import key symmetry, producer→consumer path trace) |
| `references/angle-d-cleanup.md` | Reuse/DRY, complexity tooling, idiomatic patterns, altitude |
| `references/angle-e-tests.md` | Tests design, counter-factual check per AC |
| `references/angle-f-standards.md` | Standards crosswalk with citations, error handling, security |
| `references/consolidator.md` | Dedup, verify, triage, PASS/PASS-with-triage/FAIL report structure |

Optional inputs the workflow can accept when reviewing within a task context: `task_spec_path`, `feature_prd_path`, `acceptance_criteria` — angle E uses these for counter-factual checking.
