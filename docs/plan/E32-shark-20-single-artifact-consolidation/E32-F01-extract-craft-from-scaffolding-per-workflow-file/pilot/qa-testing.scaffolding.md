# Sidecar — extracted scaffolding from `quality/workflows/qa-testing.md`

This file captures the workflow scaffolding (gates, fetches, mutations, advancements) that was removed from the craft version. F4 lands these blocks in the appropriate prompt files (likely `prompts/feature/in_qa.md` or `prompts/task/in_qa.md`).

Each block is tagged: `# fetch`, `# gate`, `# mutate`, `# advance`, `# preflight`, `# convention`.

---

## # fetch — task details

```bash
shark get E07-F08-001 --json
```

Extract from JSON: `task_id`, `title`, `description`, `acceptance_criteria`, `notes`, `context` (related docs, files).

These map to craft inputs `task_id`, `task_spec_path`, `acceptance_criteria`.

---

## # fetch — feature path resolution

```bash
FEATURE_JSON=$(shark get $FEATURE_ID --json)
FEATURE_PATH=$(echo "$FEATURE_JSON" | jq -r '.path')
FEATURE_FILENAME=$(echo "$FEATURE_JSON" | jq -r '.filename')
cat "$FEATURE_PATH/$FEATURE_FILENAME"
```

Maps to craft input `feature_prd_path`. The "do NOT assume the filename is prd.md" comment is a host-contract detail — belongs in the prompt, not the craft.

---

## # convention — file path conventions

```
docs/plan/$EPIC_ID/$FEATURE_ID/04-api-specification.md  # api spec
docs/plan/$EPIC_ID/$FEATURE_ID/test_plans/*${TASK_ID}*  # pre-existing test plan
docs/plan/$EPIC_ID/$FEATURE_ID/qa_reports/${TIMESTAMP}-${TASK_ID}-qa-results.md
docs/plan/$EPIC_ID/$FEATURE_ID/qa_reports/${TIMESTAMP}-${TASK_ID}-exploratory-findings.md
docs/development/credentials.md  # test credentials
docs/development/testing-guide.md
```

These are shark/repo conventions — the prompt resolves them and passes via `qa_report_path`, `qa_exploratory_path`, `test_plan_path`, `api_spec_path`.

---

## # mutate — failure note storage

```bash
shark task note add E07-F08-001 --type testing \
  "Test failure: 'expired link' test failed - link still works after 1 hour"
```

The act of storing a finding into shark is scaffolding. The craft produces structured output (`bugs`, `blockers`); the prompt translates them into shark notes after the craft returns.

---

## # mutate — bug-fix routing

```bash
shark task note add E07-F08-001 --type blocker \
  "Bug fix required - QA failed. Review qa_reports/${TIMESTAMP}-${TASK_ID}-qa-results.md"
shark task context set E07-F08-001 --field bug_fix --value true
```

Routes the task into the bug-fix workflow on FAIL. Pure shark state mutation — belongs in the prompt's `# on_fail` section.

---

## # mutate — pass note

```bash
shark task note add E07-F08-001 --type testing \
  "QA PASS - see qa_reports/${TIMESTAMP}-${TASK_ID}-qa-results.md"
```

Pass-side variant of the same mutation. The `--type comment` minor-issues note is also a mutate.

---

## # gate — codex red-team is mandatory before advancing

The codex methodology is craft (preserved). The **mandate** — "you cannot advance to UAT without codex returning PASS or with documented blockers acknowledged" — is a workflow-level invariant. From user memory: "Codex red-team is mandatory at QA and UAT, never skip."

**Belongs in prompt**:

```
# After craft returns:
if craft.codex_verdict != "PASS" and not craft.codex_blockers_acknowledged:
    set verdict = FAIL
    do not advance
```

---

## # gate — frontend visual verification mandatory if frontend code present

Methodology (load page, compare to designs, document) is craft. The **mandate** — "if `has_frontend=true`, the verdict must include a frontend verification section or QA fails" — is a gate.

**Belongs in prompt**:

```
# After craft returns:
if has_frontend and craft.frontend_verification_section is missing:
    set verdict = FAIL
    do not advance
```

This is consistent with the user's feedback memory: "Must load frontend pages in browser before approving at code review/QA."

---

## # gate — design refs from PRD must appear in task spec

User memory: "Design references must flow from PRD/epic into task specs as hard requirements."

This is enforced upstream of QA (in the task spec phase), not inside QA. But QA verifies the task spec is correctly populated — that's a craft check ("compare against design references" — Step 4 of craft) AND a gate at the spec-completion phase (separate prompt).

**No QA-prompt action**; mentioned here for traceability.

---

## # advance — on PASS

```
# After craft returns PASS:
shark status advance <task_id>      # or: shark status set <task_id> ready_for_uat
```

Status name `ready_for_uat` (or whatever the next step is) is shark vocabulary — pure scaffolding.

---

## # advance — on FAIL

```
# After craft returns FAIL:
shark status set <task_id> in_dev   # route back to development
# (combined with the mutate above to set bug_fix=true context)
```

---

## # preflight — extract acceptance criteria from spec

The craft requires `acceptance_criteria` as a pre-extracted list. The prompt parses the task spec's `## Acceptance Criteria` section and constructs the list before calling the craft.

```bash
# Pseudocode in prompt:
ac_list = parse_section(task_spec_path, "Acceptance Criteria")
```

---

## # preflight — codex command construction

The craft expects `codex_command` as a fully-formed string. The prompt assembles it from inputs:

```bash
codex_command="codex exec -m gpt-5.2-codex -s read-only \
  -c model_reasoning_effort=high \
  --skip-git-repo-check \
  \"You are performing an independent QA red-team review for task ${task_id}.

Read these files and cross-check the implementation against the spec:
- Task spec: ${task_spec_path}
- Feature PRD: ${feature_prd_path}
- Implementation files: ${impl_paths_joined}
- Test files: ${test_paths_joined}

ENUMERATE — DO NOT ITERATE...\""
```

Note: the codex prompt body itself (the "ENUMERATE — DO NOT ITERATE", the A/B/C/D enumeration structure) is **methodology** = craft. The path interpolation and `task_id` substitution are scaffolding. F4 may resolve this by:

- Storing the codex prompt body as a partial: `_partials/_codex_qa_prompt.md`.
- Having the prompt assemble the final command by combining the partial with substituted paths.

---

## # preflight — frontend code detection

The craft expects `has_frontend` as a boolean. The prompt detects it by inspecting `impl_paths` for frontend file extensions / known directories:

```bash
has_frontend=false
for f in $impl_paths; do
  case "$f" in
    *.tsx|*.jsx|*.vue|*.svelte|src/components/*|src/pages/*|src/styles/*) has_frontend=true ;;
  esac
done
```

Detection logic is generic enough that it could go in either layer. Putting it in the prompt keeps the craft's boolean-input contract clean.

---

## # convention — verdict naming

The craft returns `PASS | FAIL` as a string. The prompt translates that into shark verdict commands:

- `PASS` → `shark status advance` (or set to next status).
- `FAIL` → `shark status set ... in_dev` + `shark task context set --field bug_fix true`.

---

## Summary of removed lines

| Original line range | Type | Where it goes |
|---------------------|------|---------------|
| `shark get $ID --json` calls (Steps 1, 2) | fetch | Prompt — supplies `task_id`, `feature_prd_path`, `acceptance_criteria` |
| `docs/plan/$EPIC_ID/$FEATURE_ID/...` paths | convention | Prompt — supplies `qa_report_path`, `qa_exploratory_path`, `test_plan_path`, `api_spec_path` |
| `shark task note add` calls (Steps 4, 7, 8) | mutate | Prompt — translates craft output to shark notes |
| `shark task context set --field bug_fix` (Step 7, 8) | mutate | Prompt — runs on FAIL |
| Status advance / set (Step 8 implicit) | advance | Prompt — runs on PASS / FAIL |
| Codex command construction (Step 5.7) | preflight | Prompt — assembles `codex_command`; methodology stays in craft |
| Codex MANDATORY framing | gate | Prompt — gate after craft returns |
| Frontend MANDATORY framing | gate | Prompt — gate after craft returns |
| `has_frontend` detection logic | preflight | Prompt — supplies boolean to craft |

**Approximate line reduction**: ~30–40 lines of the original 596 are pure scaffolding. The bulk of the file is methodology that stays in the craft.
