---
created: 2026-08-06T22:20:00-05:00
session_goal: "drive shark-rider run E40 until completed"
resume_command: "/shark-rider run E40"
branch: E40-shark-bench
epic: E40
next_dispatch: E40-F02 (approval/UAT — openai gate, BLOCKED: codex until 2026-08-07 23:30, gemini until ~2026-08-13)
---

# Resume: /shark-rider run E40 — Phase 1 of Shark Bench

Run `/shark-rider run E40`. All state lives in shark + git.

## Where things stand (verify with `shark status E40`)

- **E40-F01**: `completed`. **E40-F04**: `completed` (2026-08-06, full pipeline).
- **E40-F02**: **`approval`** — all 8 tasks completed (T-001..008 + one G7
  rework round), task_review r1 PASS, code_review r1 FAIL (G7 manifest
  blocker) → 3 sequential kickbacks (T-003→T-001→T-008) → code_review r2
  PASS. Parked at the final UAT gate: BOTH independent assessors
  quota-blocked (codex resets 2026-08-07 23:30; gemini ~2026-08-13). Blocker
  note on E40-F02. UAT gate-integrity forbids a Claude stand-in.
- **E40-F03** (baseline report, order 4): `draft` — blocked behind F02 in the
  cascade; consumes I-02 (consumer obligation note already on E40-F03).

## To resume after codex resets (2026-08-07 23:30)

1. `shark next E40 --json --prompt-out <scratchpad>/prompt.md` → E40-F02
   approval dispatch (uat-agent@openai/gpt-5.6-terra). Verify sha256, claim.
2. Run codex per the original playbook:
   `codex exec -m gpt-5.6-terra -s workspace-write -C <repo> -o <final.txt> < prompt.md`
   with a background heartbeater (240s). Pre-generate the feature diff for
   context if useful (codex can run git itself).
3. Parse verdict: APPROVED → advance pass → F02 completed → loop continues to
   F03 (assessment). REJECTED → apply task kickbacks (status set <task>
   development --reason ... --force) BEFORE advancing fail.
4. UAT context: reports at docs/review/.../E40-F02.../ (task-review r1 PASS,
   code-review r1 FAIL + r2 PASS). Round counter: this is UAT round 1 for F02.
   Reviewer must adjudicate nothing new — but NB-1 (F2P subtest filtering) and
   NB-2 (no permanent G7 negative subtest) are open non-blockers it may weigh.

## Session history worth knowing (2026-08-06 marathon)

- F04: full pipeline in one day (STANDARD 11/27, 6 tasks, task_review r2 PASS
  after 50-line trim, code_review PASS-with-triage, UAT r1 APPROVED via
  gemini substitute — owner-authorized while codex was quota-blocked).
- F02: full pipeline to approval (STANDARD 13/27, 8 tasks). Highlights:
  Q003 RESOLVED empirically (T-006 live run 2aed0d9d, claude CLI 2.1.223):
  modelUsage/num_turns/duration_api_ms confirmed; top-level `model` field
  ABSENT — no fallback shipped, modelUsage-absent = fail-loud parse error.
  First real end-to-end bench run worked (~$0.85). G7 manifest fields
  (fixture_base_sha 40-hex git SHA, corpus_schema_version, p2p_set,
  variant_bundle_sha256, shark_version, shark_binary_sha256) added across
  driver/collector/goldens after code-review adjudication.
- B054 filed: stray /tmp/.git makes FindProjectRoot escape scratch dirs.
- Known flake: never run bench/scripts/tests/run-all.sh concurrently with
  make test (tc014 g(iii) resource contention).
- agy/gemini setup (for when its quota returns ~Aug 13): allow rules live in
  ~/.gemini/antigravity-cli/settings.json (command allowlist) and
  .claude/settings.local.json (Bash(agy *)); invocation pattern + constraints
  documented in E40-F04's notes and this file's git history.

## Execution playbook (unchanged)

1. shark next → verify prompt sha256 → claim --field session_id.
2. anthropic → Agent tool (shark-worker-<effort>, model from response);
   openai → codex exec (post-reset). Named workers report via teammate
   messages; if silent, verify their diff independently and disposition as
   parent with a note.
3. Background heartbeater per claim (240s loop, hb_stop flag).
4. Parse RECOMMENDED OUTCOME → persist notes → kickbacks BEFORE advancing →
   advance --outcome --session --from-status --agent → release.
5. Gate FAIL: resume prior worker via SendMessage. Respect the task DAG when
   completing kicked-back tasks (dependencies gate completion order).
6. Commit at boundaries; docs/review/ is gitignored (disk-only reports).
7. [shark-stats] WARN lines precede shark JSON — strip to first '{'.

## Open items

- B053 (admit.sh p2p selectors), B052 (Phase 2), B054 (new), TD-074, TD-075
  (safe to close when workflow allows).
- F02 open non-blockers: NB-1 (F2P oracle subtest filtering divergence),
  NB-2 (no permanent G7-field negative subtest) — review-finding notes on
  E40-F02.
- Post-codex-reset (non-gating): optionally re-run codex red-team on F04's
  test plan (preserved prompt idea documented in test-plan.md's codex section).
- Delete this file once the resumed session is underway.
