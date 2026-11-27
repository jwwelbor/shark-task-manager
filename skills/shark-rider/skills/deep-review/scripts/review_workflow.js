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
// Optional scope metadata: changed_file_count, diff_shortstat, successor_delta
// Optional: task_spec_path, feature_prd_path, acceptance_criteria (array of {ac_id, text})

// Some hosts deliver `args` as a JSON-encoded string rather than a parsed
// object. Destructuring that string yields undefined for every field, which
// surfaces as a bogus "requires args.diff_path and args.skill_dir" error even
// though the caller passed them. Normalize before reading.
function normalizeInput(value) {
  return typeof value === 'string' ? JSON.parse(value) : value
}
const input = normalizeInput(args)

const { diff_path, base_commit, candidate_commit, candidate_tree, diff_sha256, repair_plan_sha256, changed_files, changed_file_count, diff_shortstat, project_root, skill_dir, coding_standards_path, task_spec_path, feature_prd_path, acceptance_criteria, entity_type, repair_context, effort } = input ?? {}

function requireScope() {
  if (!diff_path || !skill_dir || !candidate_commit || !candidate_tree || !diff_sha256) {
    throw new Error('review_workflow.js requires frozen diff, skill and candidate identity')
  }
}

function requireBugContext() {
  if (entity_type !== 'bug') return
  const fields = ['plan', 'plan_revision', 'research', 'coverage']
  if (!repair_context || !fields.every(k => typeof repair_context[k] === 'string' && repair_context[k].trim()) ||
      !/^sha256:[0-9a-f]{64}$/.test(repair_context.plan_revision) || repair_plan_sha256 !== repair_context.plan_revision.slice(7) ||
      !Array.isArray(repair_context.prior_findings)) {
    throw new Error('bug review requires accepted plan revision, research, coverage and prior findings context')
  }
}
requireScope()
requireBugContext()

// Effort flows from the /code-review <level> token (low|medium|high|xhigh|max).
// Higher effort = deeper per-angle reasoning and broader coverage. Undefined → inherit session effort.
function effectiveEffort() { return effort || undefined }
const reviewEffort = effectiveEffort()

function validGitPath(path) {
  // Unicode mode treats a valid surrogate pair as one scalar. An isolated
  // DC80–DCFF represents a Git filename byte under the wire contract.
  return typeof path === 'string' && !/[\0\uD800-\uDC7F\uDD00-\uDFFF]/u.test(path) &&
    path.split('/').every(part => part !== '' && part !== '.' && part !== '..')
}

function listChangedFiles() {
  if (Array.isArray(changed_files) && changed_files.every(validGitPath)) return changed_files
  if (validGitPath(changed_files) && !changed_files.includes('\n')) return [changed_files]
  throw new Error('changed_files must be the captured JSON array of exact Git filenames')
}
let changedFiles
try {
  changedFiles = listChangedFiles()
} catch (error) {
  return {report: `INCOMPLETE: ${error.message}.`, runner_mode: 'incomplete', specialists_completed: 0,
    consolidator_completed: false, angle_results: null}
}
const fileList = JSON.stringify(changedFiles)
let comparisonContext = `- BASE_COMMIT: ${base_commit || '(not provided)'}`
if (input.successor_delta) {
  comparisonContext += `\n- SUCCESSOR_DELTA: ${JSON.stringify(input.successor_delta)}
Inspect changes from predecessor_commit to CANDIDATE_COMMIT and trace their effects, including inputs restored to baseline. CHANGED_FILES is the full review manifest. Credit for prior work requires independent verification.`
}
// Parent-verified carry plan: file-local units credited from a prior receipt.
// The dispatcher, not the model, attaches them; admission re-verifies each one.
const carryUnits = Array.isArray(input.carry_units) ? input.carry_units : []
if (carryUnits.length > 0) {
  comparisonContext += `\n- CARRIED_PATHS: ${JSON.stringify(carryUnits.map(unit => unit.path))}
These paths are credited from an earlier verified review. List only paths you freshly inspected in reviewed_files, but still trace every cross-file effect of the edited paths into them.`
}
function withCarriedUnits(result) {
  if (carryUnits.length === 0 || !result || typeof result !== 'object') return result
  const carried = new Set(carryUnits.map(unit => unit.path))
  const fresh = Array.isArray(result.reviewed_files) ? result.reviewed_files.filter(file => !carried.has(file)) : result.reviewed_files
  return {...result, reviewed_files: fresh, carried_units: carryUnits, cross_file_fresh: true}
}
function scopeDescription() {
  return {count: changed_file_count || changedFiles.length, line: `- DIFF_SHORTSTAT: ${diff_shortstat || '(not provided)'}`}
}
const {count: changedFileCount, line: diffScopeLine} = scopeDescription()
const refsDir = `${skill_dir}/references`

const FINDING_SCHEMA = {
  type: 'object',
  properties: {
    review_contract_version: {type: 'integer', enum: [2]},
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
          severity:  { type: 'string', enum: ['blocker', 'non-blocker'] },
          rule:      { type: 'string' },
          summary:   { type: 'string' },
          diagnosis: { type: 'string' },
          impact:    { type: 'string', minLength: 1 },
          evidence:  { type: 'string' },
          correction:{ type: 'string' },
        },
        required: ['file', 'line', 'severity', 'rule', 'summary', 'diagnosis', 'impact', 'evidence'],
      },
    },
    reviewed_files: {
      type: 'array',
      items: { type: 'string' },
    },
  },
  required: ['review_contract_version', 'candidate_identity', 'findings', 'reviewed_files'],
}

const CONSOLIDATION_SCHEMA = {
  type: 'object',
  properties: {
    report: {type: 'string'},
    decisions: {
      type: 'object',
      properties: {
        review_contract_version: {type: 'integer', enum: [2]},
        candidate_identity: FINDING_SCHEMA.properties.candidate_identity,
        reviewed_files: FINDING_SCHEMA.properties.reviewed_files,
        discoveries: {type: 'array', items: {
          type: 'object',
          properties: {angle: {type: 'string'}, file: {type: 'string'}, start_line: {type: 'integer'}, end_line: {type: 'integer'}, observation: {type: 'string'}},
          required: ['angle', 'file', 'start_line', 'end_line', 'observation'],
        }},
        findings: {type: 'array', items: {
          type: 'object',
          properties: {
            file: {type: 'string'}, line: {type: 'integer'}, severity: {type: 'string'},
            rule: {type: 'string'}, summary: {type: 'string'}, diagnosis: {type: 'string'}, impact: {type: 'string', minLength: 1}, evidence: {type: 'string', minLength: 1}, disposition: {type: 'string'}, reason: {type: 'string'}, finding_id: {type: 'string'}, origin: {type: 'string', enum: [...'abcdef', 'consolidator']},
          },
          required: ['file', 'line', 'severity', 'rule', 'summary', 'diagnosis', 'impact', 'evidence', 'disposition', 'reason'],
        }},
      },
      required: ['review_contract_version', 'candidate_identity', 'findings'],
    },
  },
  required: ['report', 'decisions'],
}

const SITE_SCHEMA = {
  type: 'object',
  properties: {
    candidate_identity: FINDING_SCHEMA.properties.candidate_identity,
    site_inspection: {
      type: 'object',
      properties: {file: {type: 'string'}, start_line: {type: 'integer'}, end_line: {type: 'integer'}, observation: {type: 'string'}},
      required: ['file', 'start_line', 'end_line', 'observation'],
    },
  },
  required: ['candidate_identity', 'site_inspection'],
}
const ANGLE_FILES = {a: 'angle-a-bugs.md', b: 'angle-b-behavior.md', c: 'angle-c-sibling.md', d: 'angle-d-cleanup.md', e: 'angle-e-tests.md', f: 'angle-f-standards.md'}

// ─── Angle preamble builder ───────────────────────────────────────────────────
// Each agent reads its .md prompt file from skill_dir/references/, then executes
// using the values provided here. The .md files use DIFF_PATH / CHANGED_FILES /
// PROJECT_ROOT as descriptive placeholders; agents substitute from this context.

function preamble(angleFile, extraContext) {
  const standardsLine = coding_standards_path
    ? `- CODING_STANDARDS_PATH: ${coding_standards_path}  ← read this for any standards citation`
    : `- CODING_STANDARDS_PATH: (not found — do not claim standards violations; retain only independently evidenced impact)`
  return `CONTEXT (substitute these wherever the instructions say the placeholder name):
- DIFF_PATH:      ${diff_path}
${comparisonContext}
- CANDIDATE_COMMIT: ${candidate_commit}
- CANDIDATE_TREE: ${candidate_tree}
- DIFF_SHA256: ${diff_sha256}
- CHANGED_FILE_COUNT: ${changedFileCount}
- CHANGED_FILES:
${fileList}
- PROJECT_ROOT:   ${project_root || '.'}
${diffScopeLine}
${standardsLine}
REVIEW_CONTRACT_VERSION: 2. Each finding must include impact: an affected user/caller, behavior, invariant, documented standard or maintenance operation, plus its concrete consequence and evidence. Omit preference-only comments. Preserve original blocker and refuting evidence when withdrawn; an authorized stylistic downgrade is report-only with zero edits and zero tracker writes.
${extraContext || ''}

Your detailed review instructions are in: ${refsDir}/${angleFile}

Read that file completely. Also read the canonical portable craft criteria via shark skill get quality workflows/review-code.md (or the corresponding extracted bundle file), and apply them within your angle. Then execute the review substituting the CONTEXT values above.
Return ONLY a JSON object with candidate_identity {commit, tree, diff_sha256${entity_type === 'bug' ? ', repair_plan_sha256 (the 64 hex characters in the accepted plan revision)' : ''}}, a findings array and a reviewed_files array of every changed file you actually opened or inspected. No other text.`
}

// ─── Phase 1: Parallel review angles ────────────────────────────────────────

phase('Review')

function buildSpecContext() {
  const acContext = acceptance_criteria && acceptance_criteria.length
    ? `- ACCEPTANCE_CRITERIA:\n${acceptance_criteria.map(ac => `  [${ac.ac_id}] ${ac.text}`).join('\n')}`
    : ''
  return [
    task_spec_path ? `- TASK_SPEC_PATH: ${task_spec_path}` : '',
    feature_prd_path ? `- FEATURE_PRD_PATH: ${feature_prd_path}` : '',
    acContext,
    repair_context ? `- REPAIR_CONTEXT (accepted plan revision, research, coverage and prior findings):\n${JSON.stringify(repair_context, null, 2)}` : '',
  ].filter(Boolean).join('\n')
}
const specContext = buildSpecContext()

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

function validIdentity(identity) {
  return identity && identity.commit === candidate_commit && identity.tree === candidate_tree &&
    identity.diff_sha256 === diff_sha256 && (entity_type !== 'bug' || identity.repair_plan_sha256 === repair_plan_sha256)
}

function validFinding(f) {
  return f && typeof f.file === 'string' && f.file.length > 0 && Number.isInteger(f.line) && f.line > 0 &&
    ['blocker', 'non-blocker'].includes(f.severity) &&
    ['rule', 'summary', 'diagnosis', 'impact', 'evidence'].every(k => typeof f[k] === 'string' && f[k].trim().length > 0)
}

function fileCoverageError(files) {
  if (!Array.isArray(files)) return 'missing reviewed_files declaration'
  for (const file of files) {
    if (!validGitPath(file)) return 'invalid reviewed path ' + JSON.stringify(file)
  }
  const inspected = new Set(files)
  const missing = changedFiles.filter(file => !inspected.has(file))
  return missing.length ? 'missing required paths ' + JSON.stringify(missing) : ''
}

const CREDIT_DIGEST = /^[0-9a-f]{64}$/

function carriedUnitError(unit, delta, manifest, changed) {
  if (!unit || unit.version !== 1 || unit.obligation !== 'file_local') return 'unsupported credit version or obligation'
  if (unit.dynamic_lookup !== false || !Array.isArray(unit.inputs)) return 'a closed input list without dynamic lookup is required'
  if (unit.predecessor_commit !== delta.predecessor_commit || !unit.predecessor_run_id || !CREDIT_DIGEST.test(String(unit.predecessor_operation_digest))) {
    return 'predecessor identity does not match the captured successor delta'
  }
  if (!manifest.has(unit.path) || changed.has(unit.path)) return 'path is outside the manifest or changed since the predecessor'
  if (unit.inputs.some(input => changed.has(input))) return 'declared input changed since the predecessor'
  return ''
}

// Without carried units the full manifest must be inspected fresh. With them,
// fresh files plus structurally valid file-local units must cover it. The
// parent verifies receipts, tree entries and history at admission.
function angleCoverageError(result) {
  const units = result.carried_units
  if (!Array.isArray(units) || units.length === 0) return fileCoverageError(result.reviewed_files)
  const delta = input.successor_delta
  if (!delta) return 'carried units require a captured successor delta'
  if (result.cross_file_fresh !== true) return 'carried units require a fresh cross-file inspection declaration'
  const manifest = new Set(changedFiles), changed = new Set(delta.changed_files || []), carried = new Set()
  for (const unit of units) {
    const error = carriedUnitError(unit, delta, manifest, changed)
    if (error) return error
    if (carried.has(unit.path)) return 'carried unit is duplicated'
    carried.add(unit.path)
  }
  if (!Array.isArray(result.reviewed_files) || result.reviewed_files.some(file => !validGitPath(file))) return 'missing reviewed_files declaration'
  if (result.reviewed_files.some(file => carried.has(file))) return 'path is both fresh and carried'
  const fresh = new Set(result.reviewed_files)
  const missing = changedFiles.filter(file => !fresh.has(file) && !carried.has(file))
  return missing.length ? 'incomplete reviewed_files coverage ' + JSON.stringify(missing) : ''
}

function angleError(result) {
  if (!result || result.review_contract_version !== 2) return 'missing or unsupported review contract version'
  if (!validIdentity(result.candidate_identity)) return 'missing or incorrect candidate identity'
  if (!Array.isArray(result.findings) || !result.findings.every(validFinding)) return 'malformed findings'
  return angleCoverageError(result)
}

if (!Array.isArray(angleResults)) {
  return {report: 'INCOMPLETE: specialist dispatch returned no result vector.', runner_mode: 'incomplete',
    specialists_completed: 'unknown', consolidator_completed: false, angle_results: null}
}
angleResults = angleResults.map(withCarriedUnits)
const retainedAngles = {}
const errors = []
let completed = 0
for (const [index, label] of Array.from('abcdef').entries()) {
  const result = angleResults[index]
  if (result != null) retainedAngles[label] = result
  const error = angleError(result)
  if (error) errors.push(`angle ${label}: ${error}`)
  else completed++
}
if (completed !== 6 || angleResults.length !== 6) {
  return {report: `INCOMPLETE: ${completed} of 6 specialist results were valid; consolidation was not run. ${errors.join(' ')}`,
    runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}

const allFindings = Object.entries(retainedAngles).flatMap(([origin, packet]) =>
  packet.findings.map(finding => ({...finding, origin})))

log(`Collected ${allFindings.length} candidate findings across angles A–F`)

// ─── Phase 2: Consolidate ────────────────────────────────────────────────────

phase('Consolidate')

function makeConsolidatorPreamble() {
  return `CONTEXT:
- EVIDENCE_MODE: canonical-six-angle
- SPECIALISTS_COMPLETED: 6
- CONSOLIDATOR_COMPLETED: true
${comparisonContext}
- CANDIDATE_COMMIT: ${candidate_commit}
- CANDIDATE_TREE: ${candidate_tree}
- DIFF_SHA256: ${diff_sha256}
- DIFF_PATH:      ${diff_path}
- CHANGED_FILE_COUNT: ${changedFileCount}
- CHANGED_FILES:
${fileList}
- ANGLE_RESULTS:
${JSON.stringify(retainedAngles, null, 2)}
- PROJECT_ROOT:   ${project_root || '.'}
${diffScopeLine}
${specContext}

Your detailed consolidation instructions are in: ${refsDir}/consolidator.md

Read that file completely, then execute the consolidation and produce the compact-or-detailed markdown report it specifies.
Return one JSON object with "report" (the Markdown report) and "decisions" (an object containing candidate_identity and findings). Each decision must include file, positive integer line, severity, rule, summary, diagnosis, impact, supporting evidence, disposition, and a nonempty reason. Give each specialist decision a unique finding_id, its lowercase originating angle in origin, and the original evidence. Mark new consolidator findings with origin "consolidator". Every raw finding must have its own matching decision; use disposition "refuted" only when evidence disproves it. If the report uses a FINDING_DETAIL footer, its F-number must match the retained decision's finding_id and its severity and disposition must match. A clean PASS cannot retain an unrefuted finding. Use an empty findings array only when no findings exist. The candidate_identity must include commit, tree, diff_sha256${entity_type === 'bug' ? ', and repair_plan_sha256' : ''}.`
}
const consolidatorPreamble = makeConsolidatorPreamble()

let consolidation
try {
  consolidation = await agent(consolidatorPreamble, { label: 'consolidator', phase: 'Consolidate', schema: CONSOLIDATION_SCHEMA, effort: reviewEffort })
} catch (error) {
  return {report: `INCOMPLETE: consolidation failed: ${String(error)}`, runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles}
}
function consolidationFields(value) {
  return {report: value && value.report, decisions: value && value.decisions}
}
const {report, decisions} = consolidationFields(consolidation)

function validDecision(f) {
  return validFinding(f) && ['disposition', 'reason'].every(k => typeof f[k] === 'string' && f[k].trim().length > 0) &&
    (f.severity !== 'nit' || f.disposition === 'report-only') &&
    !(f.severity === 'blocker' && f.disposition === 'refuted' && /\bnot\s+refuted\b/i.test(f.reason))
}

function findingKey(f) {
  return JSON.stringify([f.file, f.line, f.rule, f.summary, f.diagnosis])
}

function validDecisionSet(value) {
  if (!value || value.review_contract_version !== 2 || !validIdentity(value.candidate_identity) || !Array.isArray(value.findings)) return false
  if (value.reviewed_files !== undefined && fileCoverageError(value.reviewed_files)) return false
  return value.findings.every(validDecision)
}

function validReport(body) {
  if (typeof body !== 'string' || Array.from(body.trim()).length < 80) return false
  if (/\b(?:current|this|the[ \t]+current)[ \t]+(?:review|report|assessment|inspection)[ \t]+(?:(?:is|was|did|has|remains)[ \t]+)*(?:(?:still[ \t]+)?unfinished|not[ \t]+(?:yet[ \t]+)?(?:finish|finished|complete|completed)|failed[ \t]+to[ \t]+finish|only[ \t]+partially[ \t]+complete|did_not_finish)\b/i.test(body)) return false
  const lines = body.replace(/\r\n/g, '\n').split('\n')
  const historical = /^HISTORICAL_REVIEW_STATUS:[ \t]*DID_NOT_FINISH[ \t]*$/i
  const finding = /^FINDING_DETAIL:[ \t]*INCOMPLETE (VALIDATION|PRODUCT BEHAVIOR)[ \t]*\|[ \t]*(F-[1-9][0-9]*)[ \t]*\|[ \t]*(blocker|non-blocker|nit)[ \t]*\|[ \t]*([a-z0-9_-]+)[ \t]*$/i
  const currentFields = lines.filter(line => /\bCURRENT_REVIEW_STATUS\b/i.test(line))
  if (currentFields.length && (currentFields.length !== 1 || !/^CURRENT_REVIEW_STATUS:[ \t]*COMPLETE[ \t]*$/i.test(currentFields[0]))) return false
  const special = lines.flatMap((line, i) => historical.test(line) || finding.test(line) ? [i] : [])
  if (special.length && (currentFields.length !== 1 || lines.slice(special[0]).some(line => line.trim() && !historical.test(line) && !finding.test(line)))) return false
  if (lines.filter(line => historical.test(line)).length > 1) return false
  if (lines.some(line => /\bINCOMPLETE\b/i.test(line) && !finding.test(line))) return false
  return /\b(scope|coverage)\b/i.test(body) && /\b(findings|defects)\b/i.test(body)
}

function reportVerdict(body) {
  const matches = [...body.matchAll(/\bVERDICT[\t\n\v\f\r ]*:?[\t\n\v\f\r ]*\*{0,2}[\t\n\v\f\r ]*:?[\t\n\v\f\r ]*(PASS(?:-with-triage)?|FAIL)\b/gi)]
  if (!matches.length || matches.some(match => match[1].toUpperCase() !== matches[0][1].toUpperCase())) return null
  return matches[0][1].toUpperCase()
}

const specialistOrigins = [...'abcdef']

function decisionKey(site, origin, evidence) {
  return JSON.stringify([site, origin, specialistOrigins.includes(origin) ? evidence : ''])
}

function indexDecision(decision, remaining, ids) {
  const {origin = '', finding_id: id} = decision
  const location = JSON.stringify(decision.file) + ':' + decision.line
  if (!['', 'consolidator', ...specialistOrigins].includes(origin)) return 'invalid decision origin at ' + location
  if (specialistOrigins.includes(origin) && !(typeof id === 'string' && id.trim())) return 'angle ' + origin + ' requires a finding_id at ' + location
  if (id !== undefined) {
    if (typeof id !== 'string' || !id.trim() || ids.has(id)) return 'invalid or duplicate finding_id ' + id
    ids.add(id)
  }
  const key = decisionKey(findingKey(decision), origin, decision.evidence)
  if (remaining.has(key)) return 'duplicate decision for angle ' + origin + ' at ' + location
  remaining.set(key, decision)
  return ''
}

function matchObservation(finding, remaining, uniqueSite, verdict) {
  const site = findingKey(finding)
  let key = decisionKey(site, finding.origin, finding.evidence)
  if (!remaining.has(key) && uniqueSite) key = decisionKey(site, '', '')
  const decision = remaining.get(key)
  if (!decision || decision.evidence !== finding.evidence) {
    return 'angle ' + finding.origin + ' has no matching decision for its observation at ' + JSON.stringify(finding.file) + ':' + finding.line
  }
  remaining.delete(key)
  if (verdict.startsWith('PASS') && finding.severity === 'blocker') {
    const resolved = decision.severity === 'blocker' ? decision.disposition === 'refuted' :
      verdict === 'PASS-WITH-TRIAGE' && decision.disposition !== 'refuted'
    if (!resolved) return 'unresolved blocker for angle ' + finding.origin
  }
  return ''
}

function sourceDecisionError(values, raw, verdict) {
  const counts = new Map()
  for (const finding of raw) {
    const site = findingKey(finding)
    counts.set(site, (counts.get(site) || 0) + 1)
  }
  const remaining = new Map()
  const ids = new Set()
  for (const decision of values) {
    const error = indexDecision(decision, remaining, ids)
    if (error) return error
  }
  for (const finding of raw) {
    const error = matchObservation(finding, remaining, counts.get(findingKey(finding)) === 1, verdict)
    if (error) return error
  }
  for (const decision of remaining.values()) {
    if (!decision.origin && counts.has(findingKey(decision))) return 'ambiguous extra decision at ' + JSON.stringify(decision.file) + ':' + decision.line
    if (specialistOrigins.includes(decision.origin)) return 'angle ' + decision.origin + ' has no observation for ' + decision.finding_id
  }
  return ''
}

function passingDecisions(verdict, values) {
  if (!verdict.startsWith('PASS')) return true
  if (values.some(f => f.severity === 'blocker' && f.disposition !== 'refuted')) return false
  if (verdict === 'PASS-WITH-TRIAGE') return values.some(f => !['excluded', 'refuted'].includes(f.disposition))
  return !values.some(f => f.disposition !== 'refuted')
}

function contradictoryReport(body, verdict) {
  if (!verdict.startsWith('PASS')) return false
  const normalized = body.toLowerCase().replaceAll('no blockers remain', '').replaceAll('no unresolved blockers', '')
  return /\bnot\s+refuted\b|\bblockers?\s+remain\b|\bunresolved\s+blockers?\b/.test(normalized)
}

function validConsolidation(body, value) {
  if (!validReport(body) || !validDecisionSet(value)) return false
  const verdict = reportVerdict(body)
  const markers = [...body.matchAll(/^FINDING_DETAIL:[ \t]*INCOMPLETE (VALIDATION|PRODUCT BEHAVIOR)[ \t]*\|[ \t]*(F-[1-9][0-9]*)[ \t]*\|[ \t]*(blocker|non-blocker|nit)[ \t]*\|[ \t]*([a-z0-9_-]+)[ \t]*$/gim)]
  if (new Set(markers.map(marker => marker[2])).size !== markers.length) return false
  for (const marker of markers) {
    const related = value.findings.filter(f => f.finding_id === marker[2])
    if (verdict === 'PASS' || related.length !== 1 || related[0].severity !== marker[3].toLowerCase() || related[0].disposition !== marker[4].toLowerCase() || related[0].disposition === 'refuted') return false
  }
  return verdict && !contradictoryReport(body, verdict) && passingDecisions(verdict, value.findings)
}

// Returns why a consolidation cannot be accepted, or an empty string.
function consolidationProblem(body, value, raw) {
  if (!validConsolidation(body, value)) {
    const coverage = value && Object.hasOwn(value, 'reviewed_files') && fileCoverageError(value.reviewed_files)
    return coverage || 'missing or malformed report.'
  }
  return sourceDecisionError(value.findings, raw, reportVerdict(body))
}

function incomplete(reason, extra) {
  return {report: `INCOMPLETE: ${reason}`, runner_mode: 'incomplete', specialists_completed: completed, consolidator_completed: false, angle_results: retainedAngles, ...extra}
}

function validSite(site) {
  return site && validGitPath(site.file) && Number.isInteger(site.start_line) && site.start_line > 0 &&
    Number.isInteger(site.end_line) && site.end_line >= site.start_line && typeof site.observation === 'string' && site.observation.trim().length > 0
}

// A causal site first found at consolidation is inspected afresh by the angle
// responsible for it before the consolidator may rely on it. The runner records
// which angle it dispatched and which tree the reply was bound to; a label in
// the reply cannot establish either.
function discoveryList(value) {
  const list = value && value.discoveries
  if (list === undefined) return []
  if (!Array.isArray(list) || !list.every(item => validSite(item) && Object.hasOwn(ANGLE_FILES, item.angle))) return null
  return list
}

async function inspectDiscovery(discovery) {
  const assignment = `FOLLOW_UP_ASSIGNMENT (inspect this site afresh on the frozen tree before the consolidator relies on it):
- ASSIGNED_ANGLE: ${discovery.angle}
- CANDIDATE_TREE: ${candidate_tree}
- PATH: ${discovery.file}
- RANGE: lines ${discovery.start_line}-${discovery.end_line}
- CONSOLIDATOR_DISCOVERY: ${discovery.observation}
Return ONLY a JSON object with candidate_identity and site_inspection {file, start_line, end_line, observation} for exactly this path and range. The observation states what you found when you read it yourself.`
  const reply = await agent(preamble(ANGLE_FILES[discovery.angle], `${specContext}\n${assignment}`), {label: `${discovery.angle.toUpperCase()}: site follow-up ${discovery.file}:${discovery.start_line}-${discovery.end_line}`, phase: 'Review', schema: SITE_SCHEMA, effort: reviewEffort})
  const site = reply && reply.site_inspection
  if (!reply || !validIdentity(reply.candidate_identity) || !validSite(site) || site.file !== discovery.file || site.start_line !== discovery.start_line || site.end_line !== discovery.end_line) {
    throw new Error(`angle ${discovery.angle} did not inspect ${discovery.file}:${discovery.start_line}-${discovery.end_line}`)
  }
  return {file: site.file, start_line: site.start_line, end_line: site.end_line, observation: site.observation, tree: candidate_tree, assigned_angle: discovery.angle, responder_angle: discovery.angle}
}

function discoveriesCarried(value, inspections) {
  const reconsolidated = discoveryList(value)
  return reconsolidated !== null && reconsolidated.length === inspections.length && inspections.every(inspection =>
    reconsolidated.some(item => item.angle === inspection.assigned_angle && item.file === inspection.file && item.start_line === inspection.start_line &&
      item.end_line === inspection.end_line && item.observation === inspection.observation))
}

const discovered = discoveryList(decisions)
if (discovered === null) return incomplete('consolidator returned malformed discoveries.')
if (discovered.length === 0) {
  const problem = consolidationProblem(report, decisions, allFindings)
  if (problem) return incomplete('consolidator returned ' + problem)
  return {report, runner_mode: 'canonical-six-angle', specialists_completed: completed, consolidator_completed: true, decisions, angle_results: retainedAngles}
}

phase('Review')
const inspections = []
try {
  for (const discovery of discovered) inspections.push(await inspectDiscovery(discovery))
} catch (error) {
  return incomplete(`responsible-angle follow-up failed: ${String(error)}`, {discoveries: discovered})
}
for (const inspection of inspections) {
  const packet = retainedAngles[inspection.assigned_angle]
  retainedAngles[inspection.assigned_angle] = {...packet, site_inspections: [...(packet.site_inspections || []), inspection]}
}

phase('Consolidate')
let reconsolidation
try {
  reconsolidation = await agent(`${consolidatorPreamble}

FOLLOW_UP_OBSERVATIONS (fresh inspections returned by each responsible angle for the sites you discovered; reconcile each finding against them):
${JSON.stringify(inspections)}
Return the same JSON shape. Its decisions must list every discovery again with the follow-up observation copied exactly.`, { label: 'consolidator (reconsolidation)', phase: 'Consolidate', schema: CONSOLIDATION_SCHEMA, effort: reviewEffort })
} catch (error) {
  return incomplete(`reconsolidation failed: ${String(error)}`, {discoveries: discovered})
}
const second = consolidationFields(reconsolidation)
if (consolidationProblem(second.report, second.decisions, allFindings) || !discoveriesCarried(second.decisions, inspections)) {
  return incomplete('reconsolidation did not account for the responsible-angle follow-up.', {discoveries: discovered})
}
return {report: second.report, runner_mode: 'canonical-six-angle', specialists_completed: completed, consolidator_completed: true, decisions: second.decisions, angle_results: retainedAngles}
