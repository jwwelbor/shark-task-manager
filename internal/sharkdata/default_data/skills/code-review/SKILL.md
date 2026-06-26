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

Parse the JSON output for `diff_path`, `changed_files`, and `project_root`.

### 3. Launch the workflow

```javascript
Workflow({
  scriptPath: `${skill_dir}/scripts/review_workflow.js`,
  args: {
    diff_path,
    changed_files,
    project_root,
    skill_dir,
    coding_standards_path,   // from get_diff.sh output — null if not found
    // optional — include if available:
    // task_spec_path: "/path/to/task.md",
    // feature_prd_path: "/path/to/prd.md",
    // acceptance_criteria: [{ ac_id: "AC-1", text: "..." }]
  }
})
```

The workflow runs all 6 angle agents in parallel, then the consolidator. The return value is the final markdown report — print it to the user.

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
| `references/angle-c-sibling.md` | Cross-file contract breaks, structural sibling check |
| `references/angle-d-cleanup.md` | Reuse/DRY, complexity tooling, idiomatic patterns, altitude |
| `references/angle-e-tests.md` | Tests design, counter-factual check per AC |
| `references/angle-f-standards.md` | Standards crosswalk with citations, error handling, security |
| `references/consolidator.md` | Dedup, verify, triage, PASS/PASS-with-triage/FAIL report structure |

Optional inputs the workflow can accept when reviewing within a task context: `task_spec_path`, `feature_prd_path`, `acceptance_criteria` — angle E uses these for counter-factual checking.
