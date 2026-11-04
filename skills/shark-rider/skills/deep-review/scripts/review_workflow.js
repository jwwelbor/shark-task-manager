export const meta = {
  name: 'code-review',
  description: 'Multi-angle parallel code review: 6 specialist subagents read their angle prompts from the skill references directory, run in parallel, then a consolidator verifies and ranks findings into a compact PASS report or a detailed PASS-with-triage/FAIL report.',
  phases: [
    { title: 'Review', detail: 'Angles A–F run in parallel from .md prompt files' },
    { title: 'Consolidate', detail: 'Deduplicate, verify, triage, emit compact PASS or detailed PASS-with-triage/FAIL report' },
  ],
}

// ─── Args ────────────────────────────────────────────────────────────────────
// Required: diff_path, changed_files (array), project_root, skill_dir
// Optional scope metadata: changed_file_count, diff_shortstat
// Optional: task_spec_path, feature_prd_path, acceptance_criteria (array of {ac_id, text})

// Some hosts deliver `args` as a JSON-encoded string rather than a parsed
// object. Destructuring that string yields undefined for every field, which
// surfaces as a bogus "requires args.diff_path and args.skill_dir" error even
// though the caller passed them. Normalize before reading.
const input = typeof args === 'string' ? JSON.parse(args) : args

const { diff_path, base_commit, candidate_commit, candidate_tree, diff_sha256, repair_plan_sha256, changed_files, changed_file_count, diff_shortstat, project_root, skill_dir, coding_standards_path, task_spec_path, feature_prd_path, acceptance_criteria, entity_type, repair_context, effort } = input ?? {}

if (!diff_path || !skill_dir || !candidate_commit || !candidate_tree || !diff_sha256) {
  throw new Error('review_workflow.js requires frozen diff, skill and candidate identity')
}

if (entity_type === 'bug' && (!repair_context || !['plan', 'plan_revision', 'research', 'coverage'].every(k => typeof repair_context[k] === 'string' && repair_context[k].trim()) || !/^sha256:[0-9a-f]{64}$/.test(repair_context.plan_revision) || repair_plan_sha256 !== repair_context.plan_revision.slice(7) || !Array.isArray(repair_context.prior_findings))) {
  throw new Error('bug review requires accepted plan revision, research, coverage and prior findings context')
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
    candidate_identity: {
      type: 'object',
      properties: {commit: {type: 'string'}, tree: {type: 'string'}, diff_sha256: {type: 'string'}, repair_plan_sha256: {type: 'string'}},
      required: ['commit', 'tree', 'diff_sha256'],
    },
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          file:      { type: 'string' },
          line:      { type: 'integer' },
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
  required: ['candidate_identity', 'findings', 'reviewed_files'],
}

const CONSOLIDATION_SCHEMA = {
  type: 'object',
  properties: {
    report: {type: 'string'},
    decisions: {
      type: 'object',
      properties: {
        candidate_identity: FINDING_SCHEMA.properties.candidate_identity,
        findings: {type: 'array', items: {
          type: 'object',
          properties: {
            file: {type: 'string'}, line: {type: 'integer'}, severity: {type: 'string'},
            rule: {type: 'string'}, summary: {type: 'string'}, diagnosis: {type: 'string'}, disposition: {type: 'string'}, reason: {type: 'string'},
          },
          required: ['file', 'line', 'severity', 'rule', 'summary', 'diagnosis', 'disposition', 'reason'],
        }},
      },
      required: ['candidate_identity', 'findings'],
    },
  },
  required: ['report', 'decisions'],
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
- CANDIDATE_COMMIT: ${candidate_commit}
- CANDIDATE_TREE: ${candidate_tree}
- DIFF_SHA256: ${diff_sha256}
- CHANGED_FILE_COUNT: ${changedFileCount}
- CHANGED_FILES:
${fileList}
- PROJECT_ROOT:   ${project_root || '.'}
${diffScopeLine}
${standardsLine}
${extraContext || ''}

Your detailed review instructions are in: ${refsDir}/${angleFile}

Read that file completely. Also read the canonical portable craft criteria via shark skill get quality workflows/review-code.md (or the corresponding extracted bundle file), and apply them within your angle. Then execute the review substituting the CONTEXT values above.
Return ONLY a JSON object with candidate_identity {commit, tree, diff_sha256${entity_type === 'bug' ? ', repair_plan_sha256 (the 64 hex characters in the accepted plan revision)' : ''}}, a findings array and a reviewed_files array of every changed file you actually opened or inspected. No other text.`
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
  repair_context ? `- REPAIR_CONTEXT (accepted plan revision, research, coverage and prior findings):\n${JSON.stringify(repair_context, null, 2)}` : '',
].filter(Boolean).join('\n')

let angleResults
try {
  angleResults = await parallel([
  () => agent(preamble('angle-a-bugs.md', specContext),                          { label: 'A: bugs + caller chains',  phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-b-behavior.md', specContext),                      { label: 'B: removed behavior + SOLID', phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-c-sibling.md', specContext),                       { label: 'C: cross-file + sibling',  phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-d-cleanup.md', specContext),                       { label: 'D: reuse + complexity',    phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-e-tests.md', specContext),            { label: 'E: tests + counter-factual', phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  () => agent(preamble('angle-f-standards.md', specContext),                     { label: 'F: standards crosswalk',   phase: 'Review', schema: FINDING_SCHEMA, effort: reviewEffort }),
  ])
} catch (error) {
  return {report: `INCOMPLETE: specialist dispatch failed: ${String(error)}`, runner_mode: "incomplete", specialists_completed: "unknown", consolidator_completed: false, angle_results: null}
}

const validAngle = r => r && r.candidate_identity && r.candidate_identity.commit === candidate_commit && r.candidate_identity.tree === candidate_tree && r.candidate_identity.diff_sha256 === diff_sha256 && (entity_type !== 'bug' || r.candidate_identity.repair_plan_sha256 === repair_context.plan_revision.slice(7)) && Array.isArray(r.findings) && Array.isArray(r.reviewed_files) && changedFiles.every(f => r.reviewed_files.includes(f)) &&
  r.reviewed_files.every(f => typeof f === 'string') && r.findings.every(f =>
    f && typeof f.file === 'string' && f.file.length > 0 && Number.isInteger(f.line) && f.line > 0 &&
    ['blocker', 'non-blocker', 'nit'].includes(f.severity) &&
    ['rule', 'summary', 'diagnosis'].every(k => typeof f[k] === 'string' && f[k].length > 0))
const completed = Array.isArray(angleResults) ? angleResults.filter(validAngle).length : 0
const retainedAngles = Array.isArray(angleResults) ? Object.fromEntries('abcdef'.split('').flatMap((name, i) => angleResults[i] == null ? [] : [[name, angleResults[i]]])) : null
if (completed !== 6 || angleResults.length !== 6) {
  return {report: `INCOMPLETE: ${completed} of 6 specialist results were valid; consolidation was not run.`, runner_mode: "incomplete", specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}

const allFindings = angleResults
  .filter(Boolean)
  .flatMap(r => (r && r.findings) ? r.findings : [])

const reviewedFiles = [...new Set(angleResults
  .filter(Boolean)
  .flatMap(r => (r && Array.isArray(r.reviewed_files)) ? r.reviewed_files : []))]

const uncovered = changedFiles.filter(file => !reviewedFiles.includes(file))
if (uncovered.length) {
  return {report: `INCOMPLETE: changed files lack review coverage: ${uncovered.join(', ')}`, runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}
log(`Collected ${allFindings.length} candidate findings across angles A–F`)

// ─── Phase 2: Consolidate ────────────────────────────────────────────────────

phase('Consolidate')

const consolidatorPreamble = `CONTEXT:
- EVIDENCE_MODE: canonical-six-angle
- SPECIALISTS_COMPLETED: 6
- CONSOLIDATOR_COMPLETED: true
- BASE_COMMIT: ${base_commit || '(not provided)'}
- CANDIDATE_COMMIT: ${candidate_commit}
- CANDIDATE_TREE: ${candidate_tree}
- DIFF_SHA256: ${diff_sha256}
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

Read that file completely, then execute the consolidation and produce the compact-or-detailed markdown report it specifies.
Return one JSON object with "report" (the Markdown report) and "decisions" (an object containing candidate_identity and findings). Each decision must include file, positive integer line, severity, rule, summary, diagnosis, disposition, and a nonempty reason. Every raw finding must have its own matching decision; use disposition "refuted" only when evidence disproves it. A clean PASS cannot retain an unrefuted finding. Use an empty findings array only when no findings exist. The candidate_identity must include commit, tree, diff_sha256${entity_type === 'bug' ? ', and repair_plan_sha256' : ''}.`

let consolidation
try {
  consolidation = await agent(consolidatorPreamble, { label: 'consolidator', phase: 'Consolidate', schema: CONSOLIDATION_SCHEMA, effort: reviewEffort })
} catch (error) {
  return {report: `INCOMPLETE: consolidation failed: ${String(error)}`, runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}
const report = consolidation && consolidation.report
const decisions = consolidation && consolidation.decisions

const verdictMatch = typeof report === 'string' ? report.match(/\bVERDICT\s*:?\s*\*{0,2}\s*:?\s*(PASS(?:-with-triage)?|FAIL)\b/i) : null
const identity = decisions && decisions.candidate_identity
const validDecisions = identity && identity.commit === candidate_commit && identity.tree === candidate_tree && identity.diff_sha256 === diff_sha256 &&
  (entity_type !== 'bug' || identity.repair_plan_sha256 === repair_plan_sha256) && Array.isArray(decisions.findings) &&
  decisions.findings.every(f => f && typeof f.file === 'string' && f.file.length > 0 && Number.isInteger(f.line) && f.line > 0 &&
    ['blocker', 'non-blocker', 'nit'].includes(f.severity) && ['rule', 'summary', 'diagnosis', 'disposition', 'reason'].every(k => typeof f[k] === 'string' && f[k].length > 0) &&
    !(f.severity === 'blocker' && f.disposition === 'refuted' && /\bnot\s+refuted\b/i.test(f.reason))) &&
  new Set(decisions.findings.map(f => JSON.stringify([f.file, f.line, f.rule, f.summary, f.diagnosis]))).size === decisions.findings.length
const passing = verdictMatch && /^PASS/i.test(verdictMatch[1])
const cleanPass = verdictMatch && /^PASS$/i.test(verdictMatch[1])
const normalizedReport = typeof report === 'string' ? report.toLowerCase().replaceAll('no blockers remain', '').replaceAll('no unresolved blockers', '') : ''
const contradictory = passing && /\bnot\s+refuted\b|\bblockers?\s+remain\b|\bunresolved\s+blockers?\b/.test(normalizedReport)
const unresolvedBlocker = passing && (!validDecisions || decisions.findings.some(f => f.severity === 'blocker' && f.disposition !== 'refuted') ||
  allFindings.some(f => !decisions.findings.some(d => d.file === f.file && d.line === f.line && d.rule === f.rule && d.summary === f.summary && d.diagnosis === f.diagnosis && (f.severity !== 'blocker' || (d.severity === 'blocker' && d.disposition === 'refuted')))) ||
  (cleanPass && decisions.findings.some(f => f.disposition !== 'refuted')))
if (typeof report !== 'string' || report.trim().length < 80 || /\bINCOMPLETE\b/i.test(report) || !/\b(scope|coverage)\b/i.test(report) || !/\b(findings|defects)\b/i.test(report) || !verdictMatch || !validDecisions || unresolvedBlocker || contradictory) {
  return {report: 'INCOMPLETE: consolidator returned missing or malformed report.', runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}
return {report, runner_mode: 'canonical-six-angle', specialists_completed: completed, consolidator_completed: true, decisions, angle_results: retainedAngles}
