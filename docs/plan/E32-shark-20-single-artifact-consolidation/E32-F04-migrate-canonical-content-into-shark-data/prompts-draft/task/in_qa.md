---
entity_type: task
status: in_qa
agent_type: qa
includes:
  - agents/qa.md
  - prompts/_partials/_fetch_entity_context.md
  - prompts/_partials/_resolve_spec_paths.md
  - prompts/_partials/_codex_qa_prompt.md
  - prompts/_partials/_advance.md
  - prompts/_partials/_route_back_on_fail.md
skill_paths:
  # Skills are referenced by path in the rendered prompt, NOT inlined. The
  # agent reads them at runtime via filesystem tools. Paths are resolved
  # relative to the shark-data root at render time.
  - skills/quality/workflows/qa-testing.md
variables:
  - id        # task ID
  - title
  - file_path
  - related_docs  # optional
  - related_tasks # optional
  - is_resume     # optional
---

{{template "_resume_preamble" .}}{{if eq .is_resume "true"}}RESUME QA testing for task {{.task_id}}: "{{.title}}".

Check for existing QA test results or reports. If all mapped test cases pass and acceptance criteria are validated, advance immediately. If partial testing exists, continue from where testing left off.

---

{{end}}You are the qa agent. Persona and tool configuration:

{{template "_qa_agent" .}}{{/* expands shark-data/agents/qa.md inlined via {{include: agents/qa.md}} resolved at render time */}}

Task to perform: QA on {{.task_id}}: "{{.title}}".

## Step 1 — Fetch entity context

{{template "_fetch_entity_context" .}}

## Step 2 — Resolve QA artifact paths

{{template "_resolve_spec_paths" (dict "domains" (list "QA_REPORTS"))}}

```bash
QA_REPORT_PATH="${QA_REPORTS_DIR}/${TIMESTAMP}-${TASK_ID}-qa-results.md"
QA_EXPLORATORY_PATH="${QA_REPORTS_DIR}/${TIMESTAMP}-${TASK_ID}-exploratory-findings.md"
ARTIFACT_PATH="$QA_REPORT_PATH"   # used by _advance / _route_back_on_fail partials
```

## Step 3 — Load the QA craft skill (path reference, not inlined)

Read the QA methodology from this file before proceeding:

> **Skill:** `shark-data/skills/quality/workflows/qa-testing.md`

The path is resolved relative to the project root (where `shark-data/` lives). Use your file-read tool to load it. The skill declares its `inputs:` contract in YAML frontmatter — Steps 4–7 below populate those inputs.

## Step 4 — Detect frontend code

```bash
HAS_FRONTEND=false
for f in $IMPL_PATHS; do
  case "$f" in
    *.tsx|*.jsx|*.vue|*.svelte|src/components/*|src/pages/*|src/styles/*) HAS_FRONTEND=true ;;
  esac
done
```

## Step 5 — Look up pre-existing test plan

```bash
TEST_PLAN_PATH=$(ls docs/plan/${EPIC_ID}/${FEATURE_ID}/test_plans/*${TASK_ID}* 2>/dev/null | head -1)
```

If `TEST_PLAN_PATH` is non-empty, the QA craft uses it as primary validation reference (TC-NNN scenarios).

## Step 6 — Construct codex command

```bash
CODEX_COMMAND=$(cat <<'EOF'
codex exec -m gpt-5.2-codex -s read-only \
  -c model_reasoning_effort=high \
  --skip-git-repo-check \
  "$(cat <<'PROMPT'
{{template "_codex_qa_prompt" .}}
PROMPT
)"
EOF
)
```

## Step 7 — Execute the QA craft

Now execute the QA methodology you loaded in Step 3, populated with these inputs:

- `task_id`: $TASK_ID
- `task_spec_path`: $ENT_SPEC_PATH
- `feature_prd_path`: $FEATURE_PRD_PATH
- `acceptance_criteria`: $ACCEPTANCE_CRITERIA
- `impl_paths`, `test_paths`: from `git diff` and the task's recorded scope
- `has_frontend`: $HAS_FRONTEND
- `dev_server_command`: from project config (if `has_frontend=true`)
- `test_plan_path`: $TEST_PLAN_PATH (may be empty)
- `codex_command`: $CODEX_COMMAND
- `qa_report_path`: $QA_REPORT_PATH
- `qa_exploratory_path`: $QA_EXPLORATORY_PATH

Follow the craft's procedure verbatim. Return its declared outputs.

## Step 8 — Workflow gates

After the craft returns:

```bash
# Gate 1 — Codex red-team mandatory
if [ "$CRAFT_VERDICT" = "Insufficient Evidence" ] || \
   ([ "$CODEX_VERDICT" != "PASS" ] && [ -z "$CODEX_BLOCKERS_ACK" ]); then
  echo "GATE FAIL: codex red-team did not return PASS and blockers were not acknowledged."
  CRAFT_VERDICT=FAIL
fi

# Gate 2 — Frontend visual verification mandatory if frontend code present
if [ "$HAS_FRONTEND" = "true" ] && ! grep -q "Frontend Visual Verification" "$QA_REPORT_PATH"; then
  echo "GATE FAIL: task includes frontend code but QA report has no frontend verification section."
  CRAFT_VERDICT=FAIL
fi
```

## Step 9 — Route based on verdict

```bash
case "$CRAFT_VERDICT" in
  PASS)
    {{template "_advance" (dict "advance_note_type" "testing" "advance_summary" "QA PASS")}}
    ;;
  FAIL)
    {{template "_route_back_on_fail" (dict "fail_reason_summary" "QA FAIL — see report for details" "fail_target_status" "ready_for_development")}}
    # Surface bug findings as separate blocker notes for traceability
    for bug in $CRAFT_BUGS; do
      shark task note add $TASK_ID --type blocker --content "$bug"
    done
    ;;
esac
```

## Step 10 — Return summary

```
DONE: $TASK_ID

Verdict: $CRAFT_VERDICT
QA Report: $QA_REPORT_PATH
Exploratory: $QA_EXPLORATORY_PATH

[For PASS]: tests passed counts, AC verification counts, performance observations
[For FAIL]: bug list with severity, reproduction summary, fix location
```
