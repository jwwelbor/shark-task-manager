export const meta = {
  name: 'code-review',
  description: 'Multi-angle parallel code review: 6 specialist subagents read their angle prompts from the skill references directory, run in parallel, then a consolidator verifies and ranks findings.',
  phases: [
    { title: 'Review', detail: 'Angles A–F run in parallel from .md prompt files' },
    { title: 'Consolidate', detail: 'Deduplicate, verify, triage, emit PASS/PASS-with-triage/FAIL report' },
  ],
}

// ─── Args ────────────────────────────────────────────────────────────────────
// Required: diff_path, changed_files (array), project_root, skill_dir
// Optional scope metadata: changed_file_count, diff_shortstat
// Optional: task_spec_path, feature_prd_path, acceptance_criteria (array of {ac_id, text})

const { diff_path, changed_files, changed_file_count, diff_shortstat, project_root, skill_dir, coding_standards_path, task_spec_path, feature_prd_path, acceptance_criteria, effort } = args

if (!diff_path || !skill_dir) {
  throw new Error('review_workflow.js requires args.diff_path and args.skill_dir')
}

// Effort flows from the /code-review <level> token (low|medium|high|xhigh|max).
// Higher effort = deeper per-angle reasoning and broader coverage. Undefined → inherit session effort.
const reviewEffort = effort || undefined

const changedFiles = Array.isArray(changed_files)
  ? changed_files
  : String(changed_files || '').split('\n').filter(Boolean)
const fileList = changedFiles.join('\n')
const changedFileCount = changed_file_count || changedFiles.length
const diffScopeLine = diff_shortstat
  ? `- DIFF_SHORTSTAT: ${diff_shortstat}`
  : `- DIFF_SHORTSTAT: (not provided)`
const refsDir = `${skill_dir}/references`

const FINDING_SCHEMA = {
  type: 'object',
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          file:      { type: 'string' },
          line:      { type: 'number' },
          severity:  { type: 'string', enum: ['blocker', 'non-blocker', 'nit'] },
          rule:      { type: 'string' },
          summary:   { type: 'string' },
          diagnosis: { type: 'string' },
          evidence:  { type: 'string' },
          correction:{ type: 'string' },
        },
        required: ['file', 'line', 'severity', 'rule', 'summary', 'diagnosis'],
      },
    },
    reviewed_files: {
      type: 'array',
      items: { type: 'string' },
    },
  },
  required: ['findings', 'reviewed_files'],
}

// ─── Angle preamble builder ───────────────────────────────────────────────────
// Each agent reads its .md prompt file from skill_dir/references/, then executes
// using the values provided here. The .md files use DIFF_PATH / CHANGED_FILES /
// PROJECT_ROOT as descriptive placeholders; agents substitute from this context.

function preamble(angleFile, extraContext) {
  const standardsLine = coding_standards_path
    ? `- CODING_STANDARDS_PATH: ${coding_standards_path}  ← read this for any standards citation`
    : `- CODING_STANDARDS_PATH: (not found — all standards violations downgrade to opinion-only nits)`
  return `CONTEXT (substitute these wherever the instructions say the placeholder name):
- DIFF_PATH:      ${diff_path}
- CHANGED_FILE_COUNT: ${changedFileCount}
- CHANGED_FILES:
${fileList}
- PROJECT_ROOT:   ${project_root || '.'}
${diffScopeLine}
${standardsLine}
${extraContext || ''}

Your detailed review instructions are in: ${refsDir}/${angleFile}

Read that file completely, then execute the review substituting the CONTEXT values above.
Return ONLY a JSON object with a "findings" array and a "reviewed_files" array of changed files you actually opened or inspected. No other text.`
}

// ─── Phase 1: Parallel review angles ────────────────────────────────────────

phase('Review')

const acContext = acceptance_criteria && acceptance_criteria.length
  ? `- ACCEPTANCE_CRITERIA:\n${acceptance_criteria.map(ac => `  [${ac.ac_id}] ${ac.text}`).join('\n')}`
  : ''

const specContext = [
  task_spec_path    ? `- TASK_SPEC_PATH: ${task_spec_path}` : '',
  feature_prd_path  ? `- FEATURE_PRD_PATH: ${feature_prd_path}` : '',
  acContext,
].filter(Boolean).join('\n')

const angleResults = await parallel([
  () => agent(preamble('angle-a-bugs.md'),                          { label: 'A: bugs + caller chains',  phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-b-behavior.md'),                      { label: 'B: removed behavior + SOLID', phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-c-sibling.md'),                       { label: 'C: cross-file + sibling',  phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-d-cleanup.md'),                       { label: 'D: reuse + complexity',    phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-e-tests.md', specContext),            { label: 'E: tests + counter-factual', phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-f-standards.md'),                     { label: 'F: standards crosswalk',   phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
])

const allFindings = angleResults
  .filter(Boolean)
  .flatMap(r => (r && r.findings) ? r.findings : [])

const reviewedFiles = [...new Set(angleResults
  .filter(Boolean)
  .flatMap(r => (r && Array.isArray(r.reviewed_files)) ? r.reviewed_files : []))]

log(`Collected ${allFindings.length} candidate findings across angles A–F`)

// ─── Phase 2: Consolidate ────────────────────────────────────────────────────

phase('Consolidate')

const consolidatorPreamble = `CONTEXT:
- DIFF_PATH:      ${diff_path}
- CHANGED_FILE_COUNT: ${changedFileCount}
- CHANGED_FILES:
${fileList}
- REVIEWED_FILES_REPORTED_BY_ANGLES:
${reviewedFiles.length ? reviewedFiles.join('\n') : '(none reported)'}
- PROJECT_ROOT:   ${project_root || '.'}
${diffScopeLine}
- ALL_FINDINGS (${allFindings.length} candidates from 6 parallel angles):
${JSON.stringify(allFindings, null, 2)}
${specContext}

Your detailed consolidation instructions are in: ${refsDir}/consolidator.md

Read that file completely, then execute the consolidation and produce the full markdown report.
Return the report as plain text (markdown). No JSON wrapper.`

const report = await agent(consolidatorPreamble, { label: 'consolidator', phase: 'Consolidate', effort: reviewEffort })

return report
