# B086 dispatch r24: budget, time and repeat-rejection host replays

Fixture code at `scripts/testdata/b086`; source policy `aeb3f7bf` (no skills/internal change since). Codex was unavailable (usage limit until Dec 1), so the acting host is Claude Code 2.1.332, effective model `claude-sonnet-5-5` (init event), isolated per-case HOME with only a copied credentials file that was deleted after each run, tools limited to Bash and Skill, stdin from /dev/null, no writes to the parent project or database. Candidate file is read-only except through the recorder `edit` endpoint; `verify` is the focused check. Independent verifier `verify.py --only` returned observed:

- Row 7 budget: control fix 8dac5cbb-acb6-4abb-b0d6-705b30b79266 (one edit, passed check, Fix receipt naming old and new identity, pending review/QA); dispatch-only exhausted 131869b5-046b-418b-9ce0-de6c02062f3a; project-total-only exhausted 94d17c07-d503-4e77-8d25-c8893d56c669; consumption-only exhausted c3bcb449-eaaa-4f4b-9921-27fa9a5a0032. Each exhausted case: zero edit attempts, candidate bytes and mtime unchanged, verified owner readback and Defer reason, zero CI.
- Row 12 time: time-ok 94067e12-4a2e-43fb-81e3-7a4f56a97cc4 (12 min, Fix); time-short 79d91bcd-691f-4a24-952d-0d40b4b41455 (9 min, zero edits, Defer with owner and reason).
- Row 8 repeat rejection: repeat-other 015dbaf7-99df-4847-b1d0-c93bdd5ef7e4 (different prior defects, ordinary routing, Fix); repeat-same 213ba9cf-1331-4ce1-8b20-ed3fd4fce902 (same fingerprint rejected twice, limit 2: parent escalation carrying both prior reviews, zero edits, zero tracker writes, no disposition).

Only the discretionary half of rows 7 and 12 is exercised; the mandatory (pending parent repair) half needs a fix-now packet. Fresh candidate review and QA on the Fix candidates remain outside the cases. Discarded attempts: Codex run (usage limit); claude runs on earlier prepared roots where the focused check wording, a launcher-substring check, ANSI colour in plain-text empty lists and missing tracker_project field in the prompt were verifier or prompt plumbing issues, each fixed before the counted runs.

## Delivery gate over a recorded receipt (rows 21 and 28 negative controls)

The scratch tracker is seeded with an earlier host's outcome and `receipt.json`; the host is only the finish-feature Phase 1 gate (CI is a recording endpoint, merge and review disabled). Same host, grant, packet and seed throughout; verifier `--only` returned observed:
- receipt-ship 40211e4a-e8e7-40b8-86fa-e339b815be69: owner task read back with fingerprint, reason present, exactly one CI handoff after the readback, zero tracker writes.
- receipt-no-reason afa414b8-83aa-4f22-9b6e-e6a01e6ccb02: only the safe-deferral reason removed; owner read back, zero CI, parent received the pending finding naming the missing reason, zero writes.
- receipt-note-only a2ebfbbb-8769-4ae6-b117-a9a14fbece36: receipt names a feature note as owner; note exists but no owning item; zero CI, parent received the pending finding.
