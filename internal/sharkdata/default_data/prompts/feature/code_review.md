All tasks for feature {{.id}} ("{{.title}}") have completed development.

Launch tech-lead with quality skill to perform the feature verification gate.

TIER CHECK (determines scope of this gate):
Read feature metadata: {{template "get_json" .}} → `complexity_tier` (also recorded as a "COMPLEXITY:" decision note from assessment).
- **SIMPLE / STANDARD** (or tier unknown): this is the ONLY same-model verification gate — perform ALL FOUR parts below in this single pass. Do not slim it: there is no separate QA pass behind you, and tasks receive no per-task review.
- **COMPLEX**: perform Part 1 (craft review) only; Parts 2–4 run as a separate deep QA gate next.

{{include: skills/quality/workflows/review-code.md}}

READ:
(1) Feature spec at {{.file_path}} for architecture decisions and acceptance criteria
(2) Feature test-plan.md for all expected test cases, coverage requirements, and Caller-Path Contracts
(3) CLAUDE.md for codebase standards and patterns
(4) All task specs: `{{template "list_json" .}}` → read each task's file_path in sequence
(5) Full feature diff: `git log --oneline -20` then `git diff $(git merge-base HEAD main)..HEAD`
(6) Implementation code for all changed files
(7) Parent interaction map and cross-epic maps if the feature spec declares
    I-## or X-## rows

PART 1 — CRAFT REVIEW (all tiers; treat as a single PR review, not per-task):
- Code quality, security, and adherence to CLAUDE.md across the full changeset
- Architecture aligns with feature spec decisions (naming, layers, patterns)
- Cross-task integration — do the task implementations fit together correctly?
- No regressions or unintended side effects in the combined changeset
- Flag any drift between feature intent (spec/test-plan) and actual implementation

PART 2 — AC + WIRING VERIFICATION (SIMPLE/STANDARD only — COMPLEX verifies these at the QA gate):
- All feature acceptance criteria are met in aggregate — cite the AC and the code that satisfies it
- Wiring coverage matrix: one row per CONTRACT-### and I-## with producer/consumer, contract test, test-exists, test-passes; one row per X-## with contract/shape source and coverage pointer or explicit deferral
- Any CONTRACT-###, I-##, or X-## row with a missing contract test, a test that does not assert the documented shape, or a failing test is an automatic FAIL

PART 3 — SCOPED TEST RUN (SIMPLE/STANDARD only):
- Run the project's format + lint (from `docs/architecture/tech-stack.md` **Quality Gate** section, or infer from the repo)
- Targeted unit tests: union changed files from the diff and task Scope sections, map to their test counterparts, run those plus any tests named in the test-plan ACs
- Targeted integration tests only if the changeset crosses an integration seam (migration, endpoint, service contract, I-##/X-## contract)
- DO NOT run the full integration suite or any codex red-team (UAT owns red-team)

PART 4 — CALLER-PATH CONTRACT COMPLIANCE (SIMPLE/STANDARD only):
- For every TC in the test plan with a Caller-Path Contract (skip `internal — function under test is the production entrypoint`): locate the committed test, confirm it calls the declared entrypoint with the production argument shape, mocks no higher than the declared seam, and mocks nothing on the forbidden list
- Any violation (or a contracted TC with no locatable test) is a BLOCKER naming the TC-ID → FAIL

PRODUCE verification report to docs/review/{epic-folder}/{feature-folder}/code_review/<timestamp>-{{.id}}-review.md
(derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders):
- Verdict: PASS or FAIL, and which parts ran (tier)
- AC verification: list each AC with the code/test that satisfies it (SIMPLE/STANDARD)
- Integration verification: I-##/X-## rows with implementation owner, contract/shape source, coverage pointer or deferral (SIMPLE/STANDARD)
- Test run summary and caller-path compliance results (SIMPLE/STANDARD)
- Notes on any non-blocking observations

REVIEW-FINDING LOG (structured, queryable — do this for EVERY finding, blocking or not, on PASS or FAIL):
- One note per finding: {{template "create_note" .}} "<one-line finding summary>" --type=review-finding --created-by="<reviewer model>" --metadata='{"gate":"code_review","round":<N>,"severity":"<critical|high|medium|low>","defect_class":"<one-line class statement>","fingerprint":"<file>:<symbol>:<class-slug>","tc_id":"<TC-ID or omit>","disposition":"open"}'
- round = how many times this gate has run for this feature (count prior reports in the code_review/ folder). The fingerprint lets the same finding resurfacing across rounds group mechanically.

ON PASS:
- Add note: {{template "create_note" .}} --type=review "Feature verification gate passed — see report"
- SIMPLE / STANDARD → advance to approval: {{template "advance" .}} --outcome pass
- COMPLEX → route to the deep QA gate: {{template "advance" .}} --outcome deep_verify

ON FAIL (blockers, spec drift, missing ACs, contract violations):
- Write the report with findings grouped by task
  - Verdict: FAIL
  - Per-task findings: task ID, specific issues, required changes, and a one-line defect-class statement per blocking finding (the general class, not the point instance)
- For each failing task, kick back: `shark status set <task-id> development --reason "<defect-class statement> — <specific findings>. Before fixing the cited instance, sweep the touched module(s) for every other instance of this defect class; fix all; list swept sites in the completion note."`
- Set feature back to active: `{{template "status_set" .}} active --reason "Verification gate failed — see report, tasks kicked back"`
  (report lives at docs/review/{epic-folder}/{feature-folder}/code_review/)
- Do NOT advance the feature.
