---
task_key: E40-F12-test-plan
status: draft
feature: E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v
---

# Test Plan: E40-F12 — Live I-05 evidence bundle producer for lifecycle-v2 runs

**Created:** 2026-09-08
**Feature spec:** `spec.md`
**Feature PRD:** `feature.md` (see Spec Drift Analysis — this is an unedited template stub; `spec.md` + `../epic.md` carry the real requirements)
**Epic UAT plan:** `../uat-plan.md`
**Research report:** `research-report.md`
**Status:** APPROVED

No Go file is modified by this feature (spec.md §2.1). All production entrypoints
below are `bench/scripts/*.sh` binaries and their embedded Python; test cases
are `bench/scripts/tests/tcNNN_*.sh` bash scripts in this repo's established
convention (see `tc082_retention_layout_test.sh`, `tc075_content-identity_x12_test.sh`).

---

## Product critical-path guard

Restated verbatim from `spec.md` §0 (unchanged by this test-planning pass):

- `docs/product/D01-vision-statement.md` — absent — `unresolved prerequisite: docs/product/D01-vision-statement.md missing`
- `docs/product/D02-success-criteria.md` — absent — `unresolved prerequisite: docs/product/D02-success-criteria.md missing`
- `docs/plan/product-delivery-roadmap.md` — absent — `unresolved prerequisite: docs/plan/product-delivery-roadmap.md missing`
- `docs/plan/product-critical-path.md` — absent — `unresolved prerequisite: docs/plan/product-critical-path.md missing`

All four are missing (re-verified 2026-09-08 for this test-planning pass); no
named gate, no assessable relationship, no gate-advancement evidence link is
possible. **Side-quest call: proceed** — `spec.md` §0's stated reason stands:
this closes the single hard blocker (P5, `i05_bundle_dir is not configured`)
in front of E40's first real six-family lifecycle-v2 baseline, and closes two
already-filed findings (TC-082, TD-132). This test plan does not re-litigate
the guard; it is advisory per the dispatch prompt.

---

## Spec Drift Analysis

### Drift findings

1. **`feature.md` is an unedited template stub.** Its "User Stories,"
   "Requirements," "Acceptance Criteria," and "Success Metrics" sections are
   all bracketed placeholder boilerplate (`[Requirement Title]`,
   `[Specific testable criterion 1]`, etc.) — never filled in. Only the
   Problem/Solution/Impact prose (feature.md lines 26-37) carries real
   content. Per this workflow's Step 1, `feature.md` was read first and is
   the nominal "feature PRD," but it carries none of the traceable
   requirement/AC structure the drift-detection process expects to diff
   against. **Disposition: non-blocking, recorded per fail-loud discipline.**
   `spec.md` states explicitly (line 15) "Business context and scope: see
   epic PRD," and in practice `spec.md` §1 traces every REQ-F-* to
   `E40-F08/spec.md` REQ-F-010 (the actual parent requirement), not to a
   feature.md structure that was never populated. There is nothing in
   feature.md's real prose (Problem/Solution/Impact) that spec.md
   contradicts or narrows.
2. **`spec.md` AC-005 names the wrong validator script (found via codex
   red-team, confirmed by direct inspection).** AC-005's text reads "...
   mutating one byte of a snapshot makes `verify-stage-evidence.sh` exit `1`
   naming `snapshot_mutated`." Direct inspection of both scripts shows
   `snapshot_digest` recomputation and the `snapshot_mutated` verdict are
   implemented exclusively in `bench/scripts/replay-stage-evidence.sh`
   (`recompute_snapshot_digest()`, line 184; `snapshot_mutated` verdict, line
   207) — `verify-stage-evidence.sh` contains **zero** occurrences of
   `snapshot_digest` and performs no digest recomputation at all, despite
   `bench/README.md`'s "Stage-snapshot field reference" section also
   (incorrectly) crediting it with "snapshot immutability." **Disposition:
   non-blocking for this test plan** — TC-005 below is written against the
   real script that actually implements the check
   (`replay-stage-evidence.sh`), not the one spec.md names. This discrepancy
   in spec.md/README.md itself is out of test-planning's authority to
   silently fix; it is recorded here per fail-loud discipline and should be
   corrected in spec.md (AC-005's script name) or README.md (its ownership
   claim) during or after this feature's implementation.
3. **`missing_provider` (i05-schema.yaml's own documented `error_kind`) is
   not enforced by any shipped validator.** Grepped `verify-stage-evidence.sh`,
   `replay-stage-evidence.sh`, and the Go contract test
   `e40_i05_stage_evidence_contract_test.go`: the string `missing_provider`
   appears only inside `i05-schema.yaml`'s own vocabulary list and header
   comment, never in executable validation logic. **Disposition:
   non-blocking** — REQ-F-003/i05-schema.yaml's header text requires the
   snapshot to *carry* a `provider` field, which this feature's producer
   still must satisfy (AC-004); TC-004 below asserts that directly rather
   than routing through a validator check that does not exist today. Whether
   `missing_provider` should be wired into `verify-stage-evidence.sh` is
   E40-F06 scope, not E40-F12's (REQ-F-016 forbids modifying that file here).
4. **feature.md's Impact claim is already self-corrected in spec.md.**
   feature.md line 37 states this feature "Unblocks a real, spend-eligible
   `pilot`/`baseline` run across all 6 lifecycle-v2 families." `spec.md`
   §1.4 item 3 explicitly narrows this: "feature.md's Impact ... is NOT met
   by this feature alone" — a correct-pair (non-`failed`) evaluated run also
   requires closing Q008 (§3.2), which is out of this feature's scope. This
   is the one real drift between feature.md and spec.md, and spec.md already
   resolves it by stating the narrower, honest claim and pinning it to
   AC-023. No further test-plan action needed beyond preserving AC-023's
   content-only scope (see AC-023 below) — a test case must not claim "a
   non-failed evaluated pair" as an achieved postcondition anywhere in this
   plan.

### Traceability matrix

| Parent requirement | Feature requirement | Covered? | Notes |
|---|---|---|---|
| `E40-F08/spec.md` REQ-F-010 ("the runner MUST write or reference the I-05 stage snapshot, time ledger, candidate snapshot, artifact producer/consumer events, provider usage, and evaluator-access ordering") | REQ-F-002/003/005/006/007 | Yes | This feature is REQ-F-010's missing production half; every field REQ-F-010 lists maps onto REQ-F-002 (bundle.json/stage index), REQ-F-003 (per-stage snapshot), REQ-F-005 (time ledger), REQ-F-006 (access.jsonl), REQ-F-007 (transcripts/) |
| `E40-F10` code-review finding TC-082 (`retain_pair()` never copies real I-05 content) | REQ-F-007, REQ-F-013 | Yes | AC-011, AC-018 |
| `TD-132` (`content_root` never populated) | REQ-F-012 | Yes | AC-016, AC-017 |
| `bench/README.md` "Bundle layout (I-05)" / "Stage-snapshot field reference" (E40-F06's shipped schema) | REQ-F-002, REQ-F-003, REQ-F-004 | Yes, byte-for-byte, no new vocabulary | AC-002, AC-004, AC-007, AC-021, AC-022 |
| I-05 staged edge (E40-interaction-map.md) activation obligation | Not closed by this feature (Q009) | Explicitly deferred | §3.2 / Q009; not modeled as an open finding here — see "Recommendations" |

### Ambiguity findings

One, found by the codex red-team pass and confirmed by direct file
inspection (see Spec Drift Analysis item 2): **AC-005 names
`verify-stage-evidence.sh` for a check that script does not implement** —
the real script is `replay-stage-evidence.sh`. Non-blocking; TC-005 targets
the correct script. Every other AC in `spec.md` §1.3 is stated with concrete,
checkable postconditions (exact field names, exact exit codes, named error
kinds) that were verified to exist in the real, current shipped scripts
during this test-planning pass (not merely trusted from the spec's prose).

### Missing coverage

None found. Every REQ-F-001..017 has at least one AC (AC-001..023), and every
AC maps to at least one test case below.

---

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Decision table (mode × option-presence) | TC-001 | §2.3's mode table is a 4×2 closed grid (live/contract/dry-run/resolve-route × present/absent) |
| AC-002 | Equivalence partitioning + boundary (exact field census) | TC-002 | Required-vs-additive-vs-forbidden field sets are three disjoint partitions |
| AC-003 | Equivalence partitioning (index integrity) | TC-003 | Ordinal set equality against I-07 is a set-membership check |
| AC-004 | Equivalence partitioning (field inventory) | TC-004 | Field-reference completeness across all snapshot fields |
| AC-005 | Boundary value analysis (digest recompute) + mutation testing | TC-005 | Digest equality is exact; a single-byte mutation is the boundary case |
| AC-006 | Equivalence partitioning (stage_category → candidate presence, 3 partitions: code, review, non-candidate) | TC-006 | `candidate` presence is a closed partition of `stage_category` |
| AC-007 | **State-transition / decision table** (closed lifecycle table, `skills/quality/workflows/state-space-coverage.md` "Technique selection from state shape") | TC-007 | §2.4 is a closed table of 14 dispatchable phases + 3 non-dispatchable phases + 1 catch-all → 8-category mapping (corrected count per codex red-team, see TC-007); every value, an invalid-transition case (unlisted phase), a non-dispatchable-value case, and a state-addition regression case are required |
| AC-008 | **State-transition / decision table** (six `interval_category` values) + BVA at window edges | TC-008 | §2.5 is a closed 6-category table with a reconciliation invariant |
| AC-009 | Equivalence partitioning (envelope-reports-none vs envelope-reports-some) | TC-009 | Two-partition split on `provider_active` sourcing |
| AC-010 | Equivalence partitioning (zero-access vs non-zero-access run) | TC-010 | |
| AC-011 | Equivalence partitioning + real-consumer integration | TC-011 | Directory-shape presence/absence is binary; consumer (`retain_pair`) integration proves the shape is load-bearing, not just documented |
| AC-012 | Equivalence partitioning (root triad validity) | TC-012 | |
| AC-013 | **State-transition** (closed 11-row stop-outcome table, §2.6) | TC-013, TC-014 | Every value; split preserved exactly as `spec.md` pre-negotiated: 5 end-to-end induced, 5 direct-unit induced, 1 clean-terminal |
| AC-014 | Equivalence partitioning (interrupted-after-N vs completed) | TC-015 | |
| AC-015 | Equivalence partitioning (owned-vs-foreign entries) + attack-class enumeration (symlink refusal) | TC-016 | Reset must not touch a 5th, operator-owned entry; symlink is an adversarial input class |
| AC-016 | Equivalence partitioning (scheme marker present/absent) + real end-to-end digest equality | TC-017 | |
| AC-017 | Mutation testing (single-byte corruption under `content_root`) | TC-018 | Counterfactual: the crosscheck must fire, closing TD-132 |
| AC-018 | Equivalence partitioning (both driver call sites) | TC-019, TC-020 | Two independent callers, same contract |
| AC-019 | Equivalence partitioning (join-field presence at named paths) + negative (`dispatch_id`/`dispatch_ordinal` reasons ARE expected at this AC's boundary, per Q008 — asserted as required presence, not treated as a defect) | TC-021 | |
| AC-020 | Attack-class enumeration (write-failure injection) | TC-022 | Mid-run I/O failure is an adversarial/environmental fault class |
| AC-021 | Real end-to-end integration (unmodified validator over a real live run) | TC-023 | The acceptance bar is the shipped validator itself |
| AC-022 | Content-only (`git diff --exit-code`) | TC-024 | No behavior to exercise; a byte-identity check |
| AC-023 | Content-only (direct spec section + `shark get` read) | TC-025 | Documentation/disclosure requirement, not runtime behavior |
| REQ-NF-002 (no AC; landed in ISO matrix) | Boundary value analysis (timing threshold) | TC-026 | Bundled into TC-001..023's real runs, no separate functional AC |
| REQ-NF-003 (no AC; landed in ISO matrix) | Negative / attack-class (grep for prompt bytes) | TC-027 | |
| REQ-NF-004 (no AC; landed in ISO matrix) | Equivalence partitioning (two identical-input runs) | TC-028 | |

ACs without a technique annotation: none.

---

## ISO 25010 Coverage Matrix

| AC / NFR | Functional Suitability | Performance Efficiency | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-001 | N/A (CLI arg parse, no measured perf) | N/A (single-process CLI, no interop surface) | ✅ TC-001 (named-error message on missing option) | N/A (no persisted state at risk before first dispatch) | N/A (no untrusted input parsed here beyond the operator's own CLI args) | N/A (no maintainability-bearing structure introduced) | N/A (POSIX shell + Python stdlib only, REQ-NF-001) |
| AC-002 | ✅ TC-002 | N/A (see REQ-NF-002 row) | ✅ TC-002 (byte-shape matches shipped `i05-schema.yaml`, TC-042) | N/A (no direct human interaction with this artifact) | ✅ TC-002 (rewritten after every dispatch — durability of partial state) | N/A (no adversarial-input surface distinct from AC-015's) | N/A (no structure change beyond field census) | N/A (see AC-001) |
| AC-003 | ✅ TC-003 | N/A (index bookkeeping, no measured perf target) | ✅ TC-003 (ordinal parity with I-07, the I-07 consumer contract) | N/A (internal index, no direct human interaction) | ✅ TC-003 | N/A (no adversarial-input surface here; digest tamper is AC-005's Security row) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-004 | ✅ TC-004 | N/A (field presence check, no timing budget of its own) | ✅ TC-004 (X-09 usage-mapping field shape, `provider`) | N/A (internal field, no direct human interaction) | N/A (covered by AC-002's durability property, not a distinct reliability claim here) | N/A (`missing_provider` enforcement gap recorded as out-of-scope, Spec Drift item 3 — no security property this AC itself proves) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-005 | ✅ TC-005 | N/A (digest recompute is O(file size), not a measured-budget concern) | N/A (single-repo digest scheme, no cross-system interop) | N/A (no direct human interaction) | ✅ TC-005 (immutability under mutation) | ✅ TC-005 (a tampered snapshot is detected, not silently trusted) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-006 | ✅ TC-006 | N/A (partitioning check, no timing budget) | N/A (internal schema shape, not a cross-system contract point) | N/A (no direct human interaction) | N/A (no reliability property distinct from AC-002's durability) | N/A (no adversarial-input surface — a structural partition of already-trusted internal state) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-007 | ✅ TC-007 | N/A (categorization is O(1) per dispatch, folded into REQ-NF-002's budget) | N/A (bench-owned table, not an external contract surface) | ✅ TC-007 (named error naming status+phase, never a silent guess) | ✅ TC-007 (fail-loud on an unmapped phase rather than mis-categorizing) | N/A (no adversarial-input surface — the phase comes from Shark's own trusted CLI, not untrusted operator input) | ✅ TC-007 (state-addition regression case: a new workflow phase cannot silently mis-map) | N/A (see AC-001) |
| AC-008 | ✅ TC-008 | N/A (ledger construction is inside the already-measured dispatch, see REQ-NF-002) | N/A (single-repo ledger format, not an external contract point beyond I-05 itself) | N/A (no direct human interaction with the ledger) | ✅ TC-008 (reconciliation invariant enforced) | N/A (no adversarial-input surface distinct from AC-015's symlink/reset class) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-009 | ✅ TC-009 | N/A (partitioning check, folded into REQ-NF-002's budget) | N/A (internal sourcing rule, not a cross-system contract point) | N/A (no direct human interaction) | ✅ TC-009 (never inflates `provider_active` — UAT-16's named property) | N/A (no adversarial-input surface — envelope content comes from the already-isolated adapter seam) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-010 | ✅ TC-010 | N/A (append is O(1), no measured budget of its own) | ✅ TC-010 (append contract `--grant-access` depends on) | N/A (no direct human interaction) | ✅ TC-010 (append-only durability) | N/A (access-log integrity is `access.jsonl`'s own append-only property, already covered under Reliability; no distinct confidentiality/tamper claim beyond it) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-011 | ✅ TC-011 | N/A (directory creation is O(1), no measured budget) | ✅ TC-011 (real `retain_pair` consumer integration) | N/A (no direct human interaction) | N/A (durability covered by AC-002/AC-014, not distinct here) | ✅ TC-027 (transcript content confidentiality — REQ-NF-003's no-prompt-bytes grep runs against this exact directory; cited here per codex red-team, not left bare) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-012 | ✅ TC-012 | N/A (root declaration is O(1), no measured budget) | ✅ TC-012 (evaluator's `run_oracle()` consumption) | N/A (no direct human interaction) | N/A (durability covered elsewhere; this AC's own claim is structural correctness, in Functional Suitability) | ✅ TC-012 (three-root `worker_access` boundary correctness — the negative case's `agent_fixture_checkout` absence counterfactual is itself an isolation-boundary check; cited here per codex red-team, not left bare) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-013 | ✅ TC-013, TC-014 | N/A (triad write is O(1), folded into REQ-NF-002's budget) | N/A (internal eligibility vocabulary, not a cross-system contract point beyond I-07's own `outcome.terminal` parity, already in Functional Suitability) | ✅ TC-013/TC-014 (named `ineligibility_reasons` per outcome) | ✅ TC-013/TC-014 (eligibility triad never silently wrong under any of 11 terminal states) | N/A (no adversarial-input surface — outcomes are driven by the trusted stub/harness, not untrusted operator input) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-014 | ✅ TC-015 | N/A (no timing budget for a crash-survival property) | N/A (internal durability property, not a cross-system contract) | N/A (no direct human interaction) | ✅ TC-015 (crash/interrupt survivability — the core reliability property of this AC) | N/A (interruption here is an operational fault, not an adversarial-input class — that is AC-015's symlink case) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-015 | ✅ TC-016 | N/A (reset is O(1) over 4 fixed entries, no timing budget) | N/A (internal reset scope, not a cross-system contract) | N/A (no direct human interaction) | ✅ TC-016 (repetition isolation) | ✅ TC-016 (symlink-refusal — an adversarial/misconfigured-operator input) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-016 | ✅ TC-017 | N/A (digest walk cost is bounded by `content_root` size, not a per-dispatch budget) | ✅ TC-017 (byte-for-byte scheme match with `evaluate-lifecycle.sh`'s own function) | N/A (no direct human interaction) | N/A (durability covered by AC-002; this AC's own claim is a value-correctness property, in Functional Suitability) | N/A (tamper-detection is AC-017's own Security row, not duplicated here) | ✅ TC-017 (scheme marker prevents silent cross-scheme comparison — a maintainability/versioning property) | N/A (see AC-001) |
| AC-017 | ✅ TC-018 | N/A (crosscheck timing is bounded by AC-016's own walk cost) | N/A (internal to the identity crosscheck, not a distinct interop surface) | N/A (no direct human interaction) | N/A (durability not this AC's claim; its claim is tamper-detection, in Security) | ✅ TC-018 (tamper-evidence: the crosscheck fires) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-018 | ✅ TC-019, TC-020 | N/A (pass-through is O(1) argument plumbing) | ✅ TC-019/TC-020 (two independent driver call sites, one contract) | N/A (no direct human interaction) | N/A (durability not this AC's claim; correctness is in Functional Suitability) | N/A (no adversarial-input surface — internal argument plumbing between two trusted drivers) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-019 | ✅ TC-021 | N/A (join-field presence check, no timing budget) | ✅ TC-021 (I-07 evaluator join contract) | N/A (no direct human interaction) | N/A (durability not this AC's claim; correctness is in Functional Suitability) | N/A (no adversarial-input surface — join fields come from the same trusted in-process state as AC-002) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-020 | ✅ TC-022 | N/A (fault-injection timing is not this AC's budget; REQ-NF-002 covers the happy-path budget) | N/A (internal fault-handling contract, not a cross-system interop point) | ✅ TC-022 (named write-failure message) | ✅ TC-022 (fail-loud, no false `publication_eligible: true`) | N/A (an I/O environmental fault, not an adversarial-input class — AC-015 covers the adversarial symlink class) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-021 | ✅ TC-023 | N/A (validator timing is out of scope; TC-023 asserts correctness, not speed) | ✅ TC-023 (shipped validator acceptance, the actual acceptance bar per REQ-F-016) | N/A (no direct human interaction beyond the validator's own documented stdout summary) | ✅ TC-023 | N/A (no adversarial-input surface — the bundle under test is this feature's own real, trusted output, per TC-023's negative case) | N/A (no maintainability-bearing structure introduced beyond correctness already covered in Functional Suitability) | N/A (see AC-001) |
| AC-022 | N/A (content-only — no functional behavior to exercise) | N/A (content-only) | ✅ TC-024 (no drift into the shipped, single-owner validator) | N/A (content-only) | N/A (content-only) | N/A (content-only; REQ-F-016's non-modification is a maintainability/compatibility property, not a security one) | ✅ TC-024 (guards against silent scope creep into a file this feature must not touch) | N/A (see AC-001) |
| AC-023 | N/A (content-only — disclosure, not behavior) | N/A (content-only) | N/A (documentation artifact, not a cross-system contract) | ✅ TC-025 (disclosure is legible to a human reader and to `shark get`) | N/A (content-only) | N/A (content-only) | ✅ TC-025 (durable Question record prevents the gap being silently forgotten) | N/A (see AC-001) |
| REQ-NF-002 | N/A (this row's own claim IS performance, in the Performance column) | ✅ TC-026 (<1s/dispatch write-path timing) | N/A (not a cross-system contract) | N/A (no direct human interaction) | N/A (not this NFR's claim) | N/A (not this NFR's claim) | N/A (not this NFR's claim) | N/A (see AC-001) |
| REQ-NF-003 | N/A (this row's own claim IS security, in the Security column) | N/A (not this NFR's claim) | N/A (not a cross-system contract) | N/A (no direct human interaction) | N/A (not this NFR's claim) | ✅ TC-027 (no prompt bytes/secrets leak into the bundle) | N/A (not this NFR's claim) | N/A (see AC-001) |
| REQ-NF-004 | N/A (this row's own claim IS reliability/maintainability, in those columns) | N/A (not this NFR's claim) | N/A (not a cross-system contract) | N/A (no direct human interaction) | ✅ TC-028 (determinism across identical-input runs) | N/A (not this NFR's claim) | ✅ TC-028 (canonical serialization discipline) | N/A (see AC-001) |
| REQ-NF-005 | N/A (this row's own claim IS maintainability, in that column) | N/A (not this NFR's claim) | N/A (not a cross-system contract) | N/A (no direct human interaction) | N/A (not this NFR's claim) | N/A (not this NFR's claim) | ✅ `make fmt && make lint && make test` and `bench/scripts/tests/run-all.sh` stay green (CI gate, not a bench TC) | N/A (see AC-001) |

### Coverage gaps

None. Portability is `N/A` across the board with one shared justification,
stated once rather than repeated per row: this feature ships no compiled
artifact and touches no OS-specific path handling beyond what
`bench/scripts/**` already carries (POSIX shell + Python stdlib, REQ-NF-001);
no new portability surface is introduced.

---

## Observability Design (per behavior)

`bench/scripts/**` is an offline batch/CLI tool, not a served process — there
is no metrics/tracing backend in this codebase for it to emit into (confirmed:
`bench/scripts/lib/e40_benchmark.py` and every `run-*.sh` driver communicate
exclusively via exit code, stdout JSON, and stderr diagnostics). Runtime
evidence here is therefore **the artifact record itself plus named stderr
diagnostics and validator stdout summaries** — the same "evidence over
telemetry" posture the epic's own I-05/I-07 design already takes. This section
maps each behavior to its evidence channel rather than forcing a
metric/log/trace vocabulary this codebase does not have.

| Behavior | Evidence channel | Test assertion |
|---|---|---|
| Bundle write per dispatch (REQ-F-002/003) | File artifacts (`bundle.json`, `stages/*.json`) rewritten/added each iteration | TC-002, TC-004, TC-015 (partial-write durability) |
| Unmapped workflow phase (REQ-F-004) | `errors[]` entry, `{kind: "unknown_stage_category"}`, naming status+phase | TC-007 |
| Producer write failure (REQ-F-015) | stderr diagnostic naming the failing artifact and reason; I-07 `outcome.terminal: "error"`, `reason` = writer's own message | TC-022 |
| Validator acceptance/rejection (REQ-F-016, unmodified `verify-stage-evidence.sh`) | Fixed-order JSON summary on stdout (exit 0) or named rejection verdict on stderr (exit 1) | TC-023, TC-005 (mutation → `snapshot_mutated`), TC-016 (symlink refusal) |
| Content-digest crosscheck (TD-132/REQ-F-012) | `evaluate-lifecycle.sh`'s `ineligibility_reasons`/`invalidity_reasons` entry `identity_mismatch` at `/identity/shark_content_digest` | TC-018 |
| Reset scope (REQ-F-011) | Directory listing before/after — repetition-1 entries absent, operator-placed file present | TC-016 |
| Per-dispatch overhead (REQ-NF-002) | Internal — no observability beyond the test harness's own wall-clock instrumentation around the write path; this codebase has no runtime metrics emitter to hook into (justification, per Step 5.7's internal-only allowance) | TC-026 |

**Implementation hook:** REQ-F-004's `errors[]`/`unknown_stage_category` and
REQ-F-015's named write-failure message are hard requirements on the
implementation (already stated as such in `spec.md` REQ-F-004/REQ-F-015); this
test plan verifies both are actually emitted, not merely documented.

---

## Caller-Path Contracts (per test case)

All caller-path entrypoints are the real `bench/scripts/*.sh` binaries,
following this codebase's own established convention (`tc082`, `tc075`): drive
the shipped CLI surface, never a hand-built fixture standing in for it. The one
legitimate stub seam this codebase already uses for offline/no-egress testing
is `SHARK_BIN` / `RUN_LIFECYCLE_BIN` / `EVALUATE_LIFECYCLE_BIN` /
`LIFECYCLE_ADAPTER` sibling-path overrides (documented in each driver's own
header) — never a bare `PATH` substitution, never mocking the identity/digest
functions under test themselves.

| TC | Production entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `run-lifecycle.sh --scenario <pkg> --run-id <id> --root <key> --scratch-root <dir> --mode <mode> [--i05-bundle-dir <dir>]` (the full CLI) | `SHARK_BIN` (stub `shark` binary; the option-validation code path under test runs entirely before any `shark` call) | `parse_args()` / the mode-validation branch itself | A buggy impl silently defaults the bundle dir or accepts `resolve-route --i05-bundle-dir` without exiting 2 |
| TC-002..TC-016, TC-019..TC-022 | `run-lifecycle.sh` full CLI, `--mode live`, real dispatch loop | `SHARK_BIN` (stub `shark` returning scripted `next`/`claim`/`heartbeat`/`release` JSON) and `LIFECYCLE_ADAPTER` (stub worker envelope) — the same two external-system seams `run-lifecycle.sh`'s own header already documents as substitutable | `candidate_identity()`, `refresh_candidate()`, `canonical_digest()`, `stage_record()`, the `I05BundleWriter` class itself, and any internal identity/digest function — these must run for real (ADR-F12-01's whole rationale is reusing already-computed loop state; mocking them hides exactly the drift class this feature exists to close, research-report Finding 3) | A buggy impl derives a snapshot field from a second, ad-hoc computation that silently disagrees with the I-07 record it shares state with |
| TC-011 | `bench/scripts/lib/retain_pair` (real binary, positional-arg shape identical to `tc082`'s `build_golden()`) driven against the producer's real output directory | None — `retain_pair` is a pure filesystem consumer over already-materialized files | Any reimplementation of `retain_pair`'s copy/digest logic | A missing `transcripts/` directory makes `retain_pair` refuse the whole pair (research-report Finding 6) |
| TC-012, TC-017, TC-018, TC-021 | `evaluate-lifecycle.sh --i05 <dir> --i07 <path> --scenario <pkg> --output <path>` (real binary) | `SHARK_BIN`/adapter seams used only in the upstream `run-lifecycle.sh` invocation that produced the input files; the evaluator itself is invoked with zero mocks | `content_digest()`, `declared_content_root`, `run_oracle()`, `validate_identity_join()` — must run for real | A hand-computed "expected" digest fed straight to a comparator (this codebase's own documented defect class, `tc075` header) never exercises the evaluator's real mismatch-detection code |
| TC-013 (5 end-to-end rows) | `run-lifecycle.sh` full CLI with an environment/fixture perturbation per outcome (a 0.01 USD ceiling for `resource_limit`; a stub worker returning no `recommended_outcome` for `missing_outcome`; a stub `shark` exiting non-zero for `error`; an adapter exiting non-zero for `worker_failure`; a stub worker returning `kind: "question"` for `pause`) | Same as TC-002 row | Same as TC-002 row | A buggy triad writer reports `publication_eligible: true` on a run that did not reach a clean terminal state |
| TC-014 (5 direct-unit rows + clean case) | The producer's real triad-writer function, source-extracted from `run-lifecycle.sh` and `exec()`'d in isolation — the same technique `tc075` already uses for `content_digest()` (see TC-014 body for why a bare `import` is not possible against this script's embedded-heredoc structure); **internal-only justification**: `spec.md` AC-013 itself pre-negotiates this split — these five are environmental/racy conditions (lease expiry, external cancellation signal) not cheaply reproducible from a scripted CLI driver | n/a — extraction, not a mock | Reimplementing the triad-writer's logic by hand instead of extracting the real source | A buggy triad writer omits `ineligibility_reasons` for one of the five hard-to-induce outcomes specifically because it was never exercised |
| TC-019 | `run-lifecycle-batch.sh` (real binary) `dispatch_pair()`, invoked via the same `RUN_LIFECYCLE_BIN`/`EVALUATE_LIFECYCLE_BIN` sibling-path stub convention `tc082`'s driver-path section (a2) already uses | `RUN_LIFECYCLE_BIN`, `EVALUATE_LIFECYCLE_BIN` | The pass-through line itself (`"$RUN_LIFECYCLE_BIN" ... --i05-bundle-dir "$i05_bundle_dir"`) and the guard-ordering change | A regression silently drops `--i05-bundle-dir` from the invocation and the test would not notice unless it asserts the stub actually received it |
| TC-020 | `run-review-comparison.sh` (real binary) `dispatch_gate()`, same stub convention | `RUN_LIFECYCLE_BIN`, `EVALUATE_LIFECYCLE_BIN` | Same as TC-019 | Same as TC-019, for the comparison driver |
| TC-023 | `bench/scripts/verify-stage-evidence.sh <i05_bundle_dir>` (real, unmodified binary) over a bundle from a real `run-lifecycle.sh --mode live` run of one of the six lifecycle-v2 families (per AC-021's own "at least one" scope) | None | The validator itself | A validator-satisfying bundle that was hand-built rather than produced by a real live dispatch would not prove the producer wired the acceptance bar for real |
| TC-024 | `git diff --exit-code -- bench/scripts/verify-stage-evidence.sh` | n/a — content-only | n/a | A relaxed or parallel validator variant slipping into this file would pass every functional TC above while breaching REQ-F-016 |
| TC-025 | Direct file read of `spec.md` §3.2 plus `shark get Q008 --json` (real CLI read) | n/a — content-only / direct-file | n/a | A spec that fixes the identity gap silently, with no durable Question, would leave downstream consumers unaware Q008 exists to resolve |
| TC-026 | Wall-clock instrumentation wrapped around the real per-dispatch write path inside a real `run-lifecycle.sh --mode live` run (same entrypoint as TC-002) | Same as TC-002 | Same as TC-002 | A write path that serializes the whole bundle on every dispatch (rather than incrementally) silently regresses past the 1s budget on a larger fixture |
| TC-027 | `grep -R` over a real produced bundle directory, driven from the same real run as TC-002 | n/a — assertion over real output, no mocks | n/a | A transcript or snapshot field that embeds raw prompt text or a provider token leaks credentials into retained evidence |
| TC-028 | Two independent real `run-lifecycle.sh --mode live` invocations with identical scenario/scratch-root inputs | Same as TC-002 | Same as TC-002 | Non-deterministic key ordering or float formatting makes two otherwise-identical runs byte-diverge, breaking replay/diff tooling downstream |

---

## Test Infrastructure

**Existing patterns to follow:**

- `bench/scripts/tests/tc082_retention_layout_test.sh` — the reference
  convention for driving real production binaries end-to-end via
  `RUN_LIFECYCLE_BIN`/`EVALUATE_LIFECYCLE_BIN`/`ENTITY_HISTORY_EXPORT_BIN`
  sibling-path stub overrides, `mktemp -d` + `trap ... EXIT` workdirs, a local
  `fail()` helper, and lettered sub-cases `(a)`, `(b)`, ... iterating an array
  rather than one script per value (used here for TC-007/TC-008/TC-013's
  closed-table loops).
- `bench/scripts/tests/tc075_content-identity_x12_test.sh` — the reference
  convention for extracting and reusing the *real* `content_digest()` source
  from `evaluate-lifecycle.sh` (never a hand-retyped reimplementation) while
  always making the pass/fail determination via the real subprocess. TC-017/
  TC-018 (tc116) reuse this discipline and extend it to a live-produced
  `content_root`, which tc075 could not reach.
- `bench/scripts/tests/tc044_time_ledger_reconciliation_test.sh` and
  `tc047_usage_mapping_canary_test.sh` — existing ledger/usage-mapping guards
  that stay green unmodified; TC-008/TC-009/TC-004 must not duplicate or
  contradict them, only add the "these values came from a real live run,"
  producer-side half.
- `bench/scripts/tests/run-all.sh` — registration point for the two new
  suites.

**New test artifacts (per `spec.md` §2.1's Component changes table):**

| File | Purpose |
|---|---|
| `bench/scripts/tests/tc115_i05_producer_test.sh` | New. Guard suite for AC-001..AC-015, AC-018..AC-021 (TC-001..TC-016, TC-019..TC-023, TC-026..TC-028) |
| `bench/scripts/tests/tc116_content_identity_crosscheck_test.sh` | New. Guard suite for AC-016/AC-017 (TC-017, TC-018), complementing `tc075` |
| `bench/evidence/stage-category-map.yaml` | New, referenced by TC-007's state-addition regression case — read by the guard, never re-derived by it |
| `bench/scripts/tests/run-all.sh` | Modified — register the two new suites |

No new test helper library is needed: every TC below reuses existing shared
helpers (`lib/retain_pair`, `lib/path-safety.sh`'s dereference/symlink
predicates where applicable, `canonical_digest()` extraction pattern from
tc075) rather than introducing a new one (CLAUDE.md Rule 2 / ADR-F12-01's own
"no abstraction for single-use code" posture, mirrored here for tests).

---

## Cross-feature contract tests (I-##)

### I-05 (produced) — `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042`

Verified present: `TestTC061_I07LifecycleRunContract`-equivalent header
confirms `tests/contracts/e40_i05_stage_evidence_contract_test.go` line 1
declares "TC-042 verifies the I-05 stage evidence and evaluator isolation
contract" — the pointer resolves.

- **Producer of record:** E40-F06 (unchanged by this feature — E40-F12 is
  not written in as producer; per the interaction map's I-05 staged edge and
  its E40-F12 provisional record).
- **Shape source:** `../architecture.md#stage-evidence-and-isolation-contract`.
- **Staged fields (preserved verbatim, none altered by this feature):**
  `gate_mode` = `contract-only` until each consumer proves live
  production-path use; `activation_owner` = E40-F08 / E40-F09 / E40-F10, each
  closing its own consumption independently; `closure_key` = E40-F08 /
  E40-F09 / E40-F10 respectively, at each feature's own UAT; `counterpart_status`
  = read live from Shark at review/UAT time (2026-09-08: E40-F06/F08/F09/F10
  all `completed`); `review_basis` = E40-F06's completed specification and the
  interaction-map row, present together at F06 task_review;
  `demonstrability_disposition` = `pending-integration` until each consumer's
  live wiring closes. **Ownership of the activation obligation is Q009,
  open** — this test plan does not close Q009; TC-023 supplies the live
  bundle Q009's resolution will need, nothing more.
- **TC-042 role in this plan:** TC-042 is the schema-shape contract test
  (unmodified, Go, already shipped). This feature's TC-023 is what, for the
  first time, gives TC-042 a *real* bundle to validate the shape of — prior
  to E40-F12, TC-042 could only run against hand-built fixtures. TC-023 and
  TC-042 are not the same test and are not merged: TC-042 stays a Go contract
  test over the schema; TC-023 is a bench-side end-to-end guard proving a
  real run produces a bundle `verify-stage-evidence.sh` (the schema's own
  validator, which TC-042 also exercises) accepts.

### I-04 (consumed) — `tests/contracts/e40_i04_scenario_contract_test.go#TC-030`

Verified present: file exists, header declares "TC-030 verifies the I-04
lifecycle scenario package contract E40-F05."

- **Producer:** E40-F05. **Shape source:**
  `../architecture.md#lifecycle-scenario-package-contract`.
- **Staged fields (preserved verbatim):** `gate_mode` = `contract-only` until
  each consumer proves live production-path use; `activation_owner`/
  `closure_key` = E40-F06 / E40-F07 / E40-F08, each closing its own slice at
  its own UAT; `counterpart_status` = read live (2026-09-08: E40-F05/F06/F07/F08
  all `completed`); `review_basis` = E40-F05's completed `spec.md` and the map
  row at F05 task_review; `demonstrability_disposition` = `pending-integration`.
- **This feature's slice:** E40-F12 reads only the read-only slice already
  inside E40-F08's declared consumption (`stage_matrix.lifecycle`, `fixture`,
  `adapter`, `resource_policy`, `scenario_id`, `scenario_version`,
  `entity_family`) to populate `bundle.json`'s `scenario` block and
  `stage_matrix_source`. TC-002 asserts these fields are copied verbatim from
  the real I-04 package (not re-derived, not narrowed).

### I-07 (consumed) — `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`

Verified present: `TestTC061_I07LifecycleRunContract` (line 27) — the pointer
resolves to a real Go test function.

- **Producer:** E40-F08. **Shape source:**
  `../architecture.md#lifecycle-run-record-contract`.
- **Staged fields (preserved verbatim):** `gate_mode` = `contract-only` until
  each consumer proves live production-path use; `activation_owner` =
  E40-F09; E40-F10; `closure_key` = E40-F09 / E40-F10 at each feature's own
  UAT; `counterpart_status` = read live (2026-09-08: E40-F08 `completed`);
  `review_basis` = E40-F08's completed specification and the map row at F08
  task_review; `demonstrability_disposition` = `pending-integration`.
- **This feature's slice:** E40-F12 reads `identity`, `dispatches[]`,
  `stages[]`, `outcome` and writes two additive `identity` fields
  (`content_root`, `content_digest_scheme`) plus a changed
  `shark_content_digest` value scheme. TC-017/TC-018 cover the additive
  fields and the scheme change; TC-021 covers the join-field completeness
  the I-07 side of `validate_identity_join()` requires from I-05.
  **AC-019's explicit negative scope**: TC-021 must assert `/join/dispatch_id`
  and `/join/dispatch_ordinal` reasons remain expected (Q008's subject, not a
  defect here) — a test that treats their presence as a producer bug would
  contradict spec.md §3.2/§1.4 item 4.

---

## Cross-epic integration tests (X-##)

### X-12 (validates) — E40 UAT-14

Producer E32 (E32-F04); consumer E40 (E40-F09); owning feature E40-F09.
**Contract/shape source:** `E32-F04 Shark-data canonical-content contract;
E40 architecture "Lifecycle evaluation record contract"`. Matches
`docs/product/cross-epic-integration-map.md` line 26 and
`E40-cross-epic-map.md` line 22 verbatim (status `assigned`).

**Test coverage:** TC-017/TC-018 (new, `tc116_content_identity_crosscheck_test.sh`)
plus the existing `bench/scripts/tests/tc075_content-identity_x12_test.sh`
(unmodified, stays green). TC-017/TC-018 are additive: tc075 can only reach
its mismatch-detection branches with a **hand-injected** `content_root`;
TC-017/TC-018 are the first tests to drive a **live-produced** `content_root`
(a real `run-lifecycle.sh --mode live` run against the installed
`internal/sharkdata/default_data` tree copied into a scratch Shark project),
proving the affordance the X-12 map row's own note contemplates ("It does not
require E32 to add a benchmark-specific digest API if E40 can deterministically
hash the installed canonical tree") is real, not aspirational.

### X-09 (consumed) — E40 UAT-09 and UAT-14

Producer E27 (E27-F15); consumer E40 (E40-F06 contract owner; E40-F08 runtime
writer); owning feature E40-F06. Matches `E40-cross-epic-map.md` line 19
verbatim.

**Test coverage:** TC-004 (the required top-level `provider` field and the
fail-closed `usage_slot_unavailable` posture, populated from a real run rather
than a fixture) plus the existing `bench/scripts/canary-usagemapping.sh` and
`bench/scripts/tests/tc047_usage_mapping_canary_test.sh`, which stay green
unmodified — this feature adds no new usage-mapping field and invents no
missing slot value (spec.md §4, X-09 row).

### X-11 (consumed) — E40 UAT-11 and UAT-12

Producer E38 (E38-F07; E38-F09); consumer E40 (E40-F08); owning feature
E40-F08. Matches `E40-cross-epic-map.md` line 21 verbatim.

**Test coverage:** TC-008/TC-009 (the ledger intervals derived from the
already-observed claim/adapter/release windows — `queue_or_claim_wait`,
`provider_active` sourcing) plus E40's own UAT-11 and UAT-12 scenarios, which
remain X-11's coverage of record and are unaffected by this feature (E40-F12
adds no dispatch, claim, or routing behavior and makes no Go change — spec.md
§4, X-11 row).

---

## Acceptance Test Cases

Location for every runtime TC below (unless stated otherwise): new file
`bench/scripts/tests/tc115_i05_producer_test.sh`, one lettered sub-case per
row of a closed table where the AC's own technique requires exhaustive
coverage (following `tc082`'s `for artifact in "${ARTIFACTS[@]}"` convention),
registered in `bench/scripts/tests/run-all.sh`. TC-017/TC-018 live in
`bench/scripts/tests/tc116_content_identity_crosscheck_test.sh`.

### TC-001: `--i05-bundle-dir` mode/option decision table

**Feature Requirement:** REQ-F-001, §2.3's mode table.
**Task Acceptance Criterion:** AC-001.
**Technique Applied:** Decision table (mode × option-presence).
**ISO 25010 Characteristic(s):** Functional Suitability, Usability.
**Caller-Path Contract:** see table above (row TC-001).

**Preconditions:** A valid admitted I-04 scenario package; a real scratch
project.

**Input / Expected Output (one row per cell of the closed grid):**
- `--mode live`, no `--i05-bundle-dir` → exit non-zero **before** the first
  `shark next` call (assert via a `SHARK_BIN` stub that records whether it
  was ever invoked — it must not be); stderr names the missing option.
- `--mode live`, `--i05-bundle-dir <dir>` → exit 0 path proceeds to dispatch
  (covered jointly with TC-002).
- `--mode resolve-route`, `--i05-bundle-dir <dir>` → exit non-zero (2) naming
  the rejected option; `<dir>` is created empty beforehand and MUST be
  byte-for-byte untouched afterward (`find <dir>` before/after diff empty).
- `--mode resolve-route`, no `--i05-bundle-dir` → unaffected (existing
  behavior, regression guard only).
- `--mode contract` / `--mode dry-run`, option absent → unaffected, no-op
  (regression guard: existing tc060/tc064-style offline suites stay green).
- `--mode contract` / `--mode dry-run`, option present → full bundle written
  (equivalence with the `live` present case, one representative check).

**Negative Cases:** A missing-option `live` run must not partially create
`<i05_bundle_dir>` before failing (REQ-F-001 "never run and silently produce
no evidence").

---

### TC-002: `bundle.json` required/additive field census

**Feature Requirement:** REQ-F-002, REQ-F-014; `bench/README.md` "Bundle
layout (I-05)".
**Task Acceptance Criterion:** AC-002 (also feeds AC-019's field-presence
half).
**Technique Applied:** Equivalence partitioning (required / additive /
forbidden field sets).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility,
Reliability.
**Caller-Path Contract:** see table (TC-002 row).

**Preconditions:** Real `run-lifecycle.sh --mode live` run, ≥1 dispatch,
stubbed `SHARK_BIN`/`LIFECYCLE_ADAPTER` per the established convention.
`SHARK_BIN`'s stub responds to `admin workflow list <level> --json` with the
**real, once-captured** output of a live `shark admin workflow list` call
(frozen as a committed fixture) for every one of this test's rows, so
"real installed workflow phase data" and "stubbed shark binary" are the same
declared seam throughout this suite, never two different, inconsistently
applied claims.

**Input:** The produced `bundle.json` after run termination.

**Expected Output:**
- All ten documented fields present and non-null where required:
  `schema_version`, `scenario`, `run_id`, `roots`, `stage_matrix_source`,
  `stages`, `terminal_status`, `stop_outcome` (absent on clean run),
  `publication_eligible`, `ineligibility_reasons`.
- `schema_version` equals `bench/evidence/i05-schema.yaml`'s `schema_version`
  read at test time, never a hard-coded literal in the test itself (so the
  test does not silently pass a schema bump the producer failed to track).
- `scenario.{scenario_id,scenario_version,entity_family}` equal the real I-04
  package's own values (I-04 contract, TC-030's slice).
- The only fields present beyond the ten are exactly `scenario_id`,
  `scenario_version`, `dispatches` (REQ-F-002's additive-only allowance) —
  assert the full top-level key set is a subset of these 13 names, never more.

**Edge Cases:** Zero-dispatch run (mode `dry-run`, no dispatches reachable) —
`bundle.json` still written before the (absent) first dispatch per REQ-F-002
"before the first dispatch."

**Negative Cases:** A 14th, undeclared top-level key (simulated by monkeying
with the writer in a throwaway branch during development, not shipped) must
fail this test — guards against "no new vocabulary invented" (spec.md REQ-F-002).

---

### TC-003: `stages[]` index integrity vs I-07 dispatch ordinals

**Feature Requirement:** REQ-F-002.
**Task Acceptance Criterion:** AC-003.
**Technique Applied:** Equivalence partitioning (set equality).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility, Reliability.
**Caller-Path Contract:** TC-002 row.

**Preconditions:** Same real live run as TC-002, ≥2 dispatches (multi-entity
fixture so ordinal uniqueness is a genuine check, not a single-element
trivial pass).

**Expected Output:** `bundle.json.stages[]` has one entry per dispatch;
`{s.dispatch_ordinal for s in stages}` is a set of unique integers equal to
`{d.ordinal for d in i07.dispatches}`; for every stage entry, recomputing
`sha256` over the file at `snapshot_path` (canonical form, minus
`snapshot_digest`) equals the recorded `snapshot_digest`.

**Negative Cases:** A stage index entry whose `snapshot_digest` does not match
its file's real content is a defect this test would catch (covered jointly
with TC-005's mutation case via the same fixture).

---

### TC-004: Stage-snapshot field completeness + required `provider`

**Feature Requirement:** REQ-F-003; X-09.
**Task Acceptance Criterion:** AC-004.
**Technique Applied:** Equivalence partitioning (field reference completeness).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** TC-002 row.

**Expected Output:** Every `stages/<ordinal>-<stage_key>.json` file parses as
JSON and carries every field `bench/README.md`'s stage-snapshot field
reference lists (`dispatch_ordinal`, `entity`, `stage_key`, `stage_category`,
`prompt_digest`, `input_lineage`, `artifacts`, `usage`, `time_ledger`,
`errors`, `rework_count`, `evaluator_access`, `snapshot_digest`, plus
`candidate`/`replay_lineage` where applicable), plus a non-empty top-level
`provider` string (i05-schema.yaml header requirement, REQ-F-003).

**Negative Cases:** A snapshot missing `provider` entirely is a producer
defect this test catches directly (parse the real produced JSON and assert
the key is present and non-empty). **Correction from codex red-team,
confirmed by inspection:** `missing_provider` is declared in
`i05-schema.yaml`'s vocabulary but is **not** currently checked by
`verify-stage-evidence.sh`, `replay-stage-evidence.sh`, or the Go contract
test (`e40_i05_stage_evidence_contract_test.go`) — grepped, zero hits outside
the schema file itself. This test does not claim the unmodified validator
enforces it (REQ-F-016 forbids modifying that validator here); it asserts
the producer's own output directly. See Spec Drift Analysis item 3.

---

### TC-005: `snapshot_digest` immutability and mutation detection

**Feature Requirement:** REQ-F-003.
**Task Acceptance Criterion:** AC-005.
**Technique Applied:** Boundary value analysis (exact digest equality) +
mutation testing.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability, Security.
**Caller-Path Contract:** TC-002 row (for canonical fixture, and for the
recompute-equality check, done as direct arithmetic over real produced
files); `bench/scripts/replay-stage-evidence.sh <dir>` (real, unmodified) for
the mutation-detection negative case. **Correction from codex red-team,
confirmed by inspection (Spec Drift Analysis item 2):** `spec.md` AC-005's
own text names `verify-stage-evidence.sh` for this check, but
`recompute_snapshot_digest()`/the `snapshot_mutated` verdict live exclusively
in `replay-stage-evidence.sh` (`verify-stage-evidence.sh` contains zero
`snapshot_digest` references). This TC targets the real script; the
discrepancy in `spec.md`'s AC-005 wording is recorded, not silently
corrected.

**Expected Output:** For every snapshot, recomputing `sha256` over the
canonical serialization excluding `snapshot_digest` reproduces the recorded
value exactly.

**Negative Cases:** Flipping one byte inside one snapshot file (outside the
`snapshot_digest` field itself) makes `replay-stage-evidence.sh` exit
non-zero, naming `snapshot_mutated` for that stage — proves recomputation is
real, not a cached/short-circuited comparison.

---

### TC-006: `candidate` block partitioned by `stage_category`

**Feature Requirement:** REQ-F-003.
**Task Acceptance Criterion:** AC-006.
**Technique Applied:** Equivalence partitioning (3 partitions: `code`,
`review`, non-candidate).
**ISO 25010 Characteristic(s):** Functional Suitability.
**Caller-Path Contract:** TC-002 row.

**Preconditions:** A fixture exercising at least one `code`-category, one
`review`-category, and one `discovery`-category dispatch (multi-entity
lifecycle-v2 fixture, per epic UAT-11's own "hierarchy fork" scenario).

**Expected Output:** `code`/`review` snapshots carry all six REQ-F-006
identity fields (`base_commit`, `tree_digest`, `binary_diff_digest`,
`changed_path_digest`, `dirty_untracked_manifest`, `test_suite_digest`) plus
`test_suite_ids` and `test_suite_dir`; the `discovery` snapshot carries no
`candidate` key at all (not `null`, not `{}` — key absent) and is still
accepted by `verify-stage-evidence.sh`.

---

### TC-007: `stage_category` closed-table exhaustive coverage (state-transition)

**Feature Requirement:** REQ-F-004, §2.4's 18-phase closed table.
**Task Acceptance Criterion:** AC-007.
**Technique Applied:** **State-transition / decision table**, per
`skills/quality/workflows/state-space-coverage.md` "Technique selection from
state shape" (mandatory for a declared closed lifecycle table).
**ISO 25010 Characteristic(s):** Functional Suitability, Usability,
Reliability, Maintainability.
**Caller-Path Contract:** TC-002 row. **Correction from codex red-team:**
§2.4's table has 14 real workflow phases marked `dispatchable: yes`
(`research`, `triage`, `refinement`, `specification`, `design`,
`decomposition`, `planning`, `test_planning`, `development`, `code_review`,
`review`, `qa`, `approval`, `execution`) + 3 marked `dispatchable: no`
(`blocked`, `paused`, `done`) + 1 catch-all invalid/unmapped row — **17 real
phase values plus 1 catch-all, not "18 phases."** The phase source
(`shark admin workflow list <level> --json`) is read through the **same**
`SHARK_BIN` stub declared for the whole suite (TC-002's precondition note),
configured to return the real captured phase data for all 17 positive-partition
rows and one fabricated, never-shipped phase name for the catch-all row —
one consistently declared seam for both the positive and invalid partitions,
not two different claims.

**Input:** For every one of the 17 real `(workflow phase, dispatchable status)`
pairs in §2.4's table (looped from a data table in the guard script sourced
from the frozen `SHARK_BIN` fixture above, not sampled), a dispatch at that
status.

**Expected Output — every value (not a sampled subset):** for the 14
dispatchable phases, the resulting snapshot's `stage_category` equals the
tabulated value. For the 3 non-dispatchable phases (`blocked`, `paused`,
`done`), no snapshot is written at all — `shark next` never returns
`spawn_agent` for them (assert absence, not a `stage_category` value).

**Invalid-transition case:** A dispatch at a status whose phase (from the
same `SHARK_BIN` stub, this one row configured with a fabricated phase name
absent from §2.4's table) yields a snapshot with `errors[]` containing exactly
one `{kind: "unknown_stage_category"}` entry naming the status and phase, and
no `stage_category` value is substituted (assert the field is either absent
or an explicit sentinel — never one of the eight real category values).

**Recovery case:** The unmapped-phase run does not abort the whole lifecycle
run; subsequent dispatches in the same run still get correctly categorized
(the error is per-snapshot, not fatal to the process) — unless the run's own
resource/stop-outcome logic terminates it for an unrelated reason.

**State-addition regression case:** Read `bench/evidence/stage-category-map.yaml`
(the new machine-readable table) and assert its declared phase set is exactly
the union of `phase` values `shark admin workflow list <level> --json` returns
across all six levels (task/feature/epic/bug/change/tech_debt), read live. This
test fails the day a new phase is added to a shipped workflow without a
matching row added to `stage-category-map.yaml` — the regression case the
technique requires.

**Negative Cases:** A status with **no** phase at all (empty string) falls
into the same "unmapped" branch as an unlisted phase — must not be treated
differently or silently defaulted.

---

### TC-008: Time-ledger interval-category closed-table coverage + reconciliation

**Feature Requirement:** REQ-F-005, §2.5's six-category closed table.
**Task Acceptance Criterion:** AC-008.
**Technique Applied:** **State-transition / decision table** (six
`interval_category` values) + boundary value analysis (window edges,
`reconciliation_epsilon_ns` boundary).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.
**Caller-Path Contract:** TC-002 row.

**Input:** A real live run exercising, across its dispatches, at least one
interval in each of the six categories per §2.5's "Produced" column: a
provider-active interval (via a stubbed adapter envelope reporting one), tool
work (`candidate_identity()`/`refresh_candidate()` run for real — no stub),
claim/release wait (real `shark claim`/`shark release` stub-timed calls), a
question handoff (`replay_or_human_gate_wait`), a heartbeat retry
(`retry_or_backoff`), and residual/unattributable time (`unclassified`).

**Expected Output:** `verify-stage-evidence.sh` (real, unmodified) accepts
every produced `time_ledger`: no `ledger_overlap`, no `ledger_window_escape`,
no `ledger_non_reconciling` rejection reasons appear.

**Boundary cases:** An interval whose `end` lands exactly at `stage_end`
(closed/open boundary of the half-open `[start, end)` contract) is accepted; a
residual gap of exactly `reconciliation_epsilon_ns` (1,000,000 ns) is accepted
into `unclassified`; a residual of `reconciliation_epsilon_ns + 1` is rejected
naming its magnitude (constructed via a mutated fixture, since the producer
should never emit this in practice — proves the validator's own boundary
still catches a hypothetical producer regression).

**Invalid-transition case:** A ledger interval tagged with a 7th, invented
`interval_category` name is rejected by the schema (closed vocabulary,
unmodified `i05-schema.yaml`) — constructed via a mutated fixture; proves the
producer's writer is constrained to the closed set, not merely conventionally
compliant.

**Recovery case:** After `verify-stage-evidence.sh` rejects one stage's
`time_ledger` (e.g. the invalid-transition fixture above), every *other*
stage's snapshot in the same bundle is still independently accepted or
rejected purely on its own merits — one bad ledger does not cascade into a
false rejection (or false acceptance) of an unrelated stage's ledger.

**State-addition regression case (corrected from codex red-team):** the
producer's writer must only ever emit one of the six category names
`provider_active`, `tool_and_test`, `queue_or_claim_wait`,
`replay_or_human_gate_wait`, `retry_or_backoff`, `unclassified` — asserted
against this **fixed, literal six-name list pinned in the test itself**, not
read dynamically from `i05-schema.yaml` at test time. Reading the "expected"
set from the same file a real 7th-category addition would also edit lets such
an addition pass this check silently (self-consistently) — the whole point of
a state-addition regression case is that it must NOT move when the thing it
guards moves without an explicit, reviewed test change.

---

### TC-009: `provider_active` sourcing partitions (empty vs. populated envelope)

**Feature Requirement:** REQ-F-005 (ADR-F12-05).
**Task Acceptance Criterion:** AC-009.
**Technique Applied:** Equivalence partitioning (2 partitions).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.
**Caller-Path Contract:** TC-002 row.

**Partition 1 — envelope reports none:** stub `LIFECYCLE_ADAPTER` to return an
envelope with no `time_ledger.provider_active` intervals. Expected: the whole
adapter window appears under `unclassified`; `provider_active` totals zero
(absent or `[]`, never a coerced zero-length-but-present placeholder that
looks measured).

**Partition 2 — envelope reports intervals:** stub the adapter to report one
or more explicit `provider_active` intervals within the adapter window.
Expected: those exact `[start, end)` pairs appear under `provider_active`;
the remainder of the adapter window appears under `unclassified`; no part of
the reported `provider_active` interval leaks into any other category.

**Negative Cases:** An interval the envelope reports **outside** the adapter's
own window (malformed stub, adversarial-input class) must not be accepted as
`provider_active` verbatim — it is clipped to the stage window per REQ-F-005,
or rejected as a window-escape by the validator.

---

### TC-010: `access.jsonl` existence and append-only contract

**Feature Requirement:** REQ-F-006.
**Task Acceptance Criterion:** AC-010.
**Technique Applied:** Equivalence partitioning (zero-access vs. non-zero-access run).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility, Reliability.
**Caller-Path Contract:** TC-002 row; `verify-stage-evidence.sh <dir> --grant-access inject-tests --accessor <name> --adapter <adapter.sh> --checkout <dir> --files <path>...` (real, unmodified) for the append check.

**Partition 1 — clean run, zero evaluator access:** `access.jsonl` exists and
is empty (0 bytes or valid-but-empty JSONL) after a run with no
`evaluator_access` events.

**Partition 2 — `--grant-access` invoked post-run:** calling the real
`verify-stage-evidence.sh ... --grant-access inject-tests ...` against the
produced bundle appends an event to the existing `access.jsonl` file (assert
via inode/mtime-independent line-count before/after) without recreating the
file (assert the file is not deleted+recreated — e.g. via a pre-existing
sentinel byte offset check or hard-link identity where the filesystem
supports it).

**Negative Cases:** Any `evaluator_access` entry a snapshot carries also
appears, verbatim, in `access.jsonl` — a snapshot recording an access event
that `access.jsonl` omits is a defect.

---

### TC-011: `transcripts/` materialization + real `retain_pair` consumption

**Feature Requirement:** REQ-F-007.
**Task Acceptance Criterion:** AC-011.
**Technique Applied:** Equivalence partitioning + real-consumer integration.
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** TC-002 row for the producer half; `lib/retain_pair`
row (see Caller-Path table) for the consumer half.

**Expected Output:** `<i05_bundle_dir>/transcripts/` exists as a real
directory (`os.path.isdir()` true, `os.path.islink()` false) after every run,
with one bounded transcript artifact per dispatch. A direct
`bench/scripts/lib/retain_pair` invocation (the same positional-arg shape
`tc082`'s `build_golden()` uses) against the produced bundle succeeds (exit 0)
and writes a `manifest.json` whose `artifacts.evidence` and
`artifacts.transcripts` entries both carry a real, non-placeholder `sha256`.

**Negative Cases (regression guard against Finding 6's exact defect class):**
A bundle directory with `bundle.json`/`stages/`/`access.jsonl` but **no**
`transcripts/` subdirectory makes `retain_pair` refuse the **whole** pair
(no `manifest.json` written at all, matching `refuse_missing_source()`'s
documented fail-closed behavior) — constructed by deleting `transcripts/`
from an otherwise-real produced bundle, proving the requirement is load-bearing
and not merely documented.

---

### TC-012: Three-root triad with a real `agent_fixture_checkout`

**Feature Requirement:** REQ-F-008.
**Task Acceptance Criterion:** AC-012.
**Technique Applied:** Equivalence partitioning (valid triad vs. each
single-root defect).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** TC-002 row for the producer; `evaluate-lifecycle.sh`
row for the consumer check.

**Expected Output:** `bundle.json.roots` declares all three roots
(`agent_fixture_checkout`, `scratch_shark_project`, `evaluator_only`) with
their `i05-schema.yaml` `worker_access` values, as pairwise non-nested
absolute paths; `os.path.isdir()` accepts `agent_fixture_checkout`.
`evaluate-lifecycle.sh --i05 <bundle> ...` (real), run against this bundle
whose `agent_fixture_checkout` is a **real, present** directory, does not
append a `missing_oracle` invalidity reason (corrected wording — codex
red-team caught this row contradicting its own Negative Cases entry below).

**Negative Cases:** An `agent_fixture_checkout` value that does not exist as a
directory reproduces the pre-fix defect (`run_oracle()` silently skipping the
held-back oracle) — constructed as a counterfactual against a mutated bundle,
proving TC-012 would have caught the original gap.

---

### TC-013: Stop-outcome eligibility triad — 5 end-to-end induced outcomes (+ clean-run)

**Feature Requirement:** REQ-F-009, §2.6's closed 11-row table.
**Task Acceptance Criterion:** AC-013 (first half, preserved verbatim per
`spec.md`'s own pre-negotiated split — do not merge with TC-014).
**Technique Applied:** **State-transition** (closed table), end-to-end induction subset.
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Reliability.
**Caller-Path Contract:** TC-002 row, with a per-row environmental
perturbation (see Caller-Path table, TC-013 row).

**Rows covered here (exactly these five, plus the clean-terminal row —
mirrors `spec.md` AC-013 verbatim, no more, no fewer):**

| Induced condition | `stop_outcome` | `publication_eligible` | `ineligibility_reasons[]` |
|---|---|---|---|
| Clean terminal run | absent | `true` | `[]` |
| Ceiling of 0.01 USD | `resource_limit` | `false` | names `limits.first_exceeded` |
| Stub worker returns no `recommended_outcome` | `missing_outcome` | `false` | names the entity with no final outcome |
| Stub `shark` exits non-zero | `error` | `false` | carries `outcome.reason` |
| Adapter exits non-zero | `worker_failure` | `false` | carries `outcome.reason` |
| Stub worker returns `kind: "question"` | `pause` | `false` | names the routed Question key |

**Expected Output (each row):** the run yields the tabulated `stop_outcome`,
`publication_eligible`, a non-empty `ineligibility_reasons[]` matching the
tabulated content, and `verify-stage-evidence.sh` (real) accepts the bundle
(exit 0 — a `stop_outcome`-carrying bundle is still a *structurally valid*
bundle, just not publication-eligible).

**Negative Cases:** The bundle's `stop_outcome` equals the same run's I-07
`outcome.terminal` value exactly, for every row (REQ-F-009's cross-record
consistency requirement) — a divergence between the two records for the same
run is a defect.

**Invalid-transition case (added per codex red-team; state-transition
technique requires it for this closed table):** a bundle whose `stop_outcome`
is mutated to a value outside the eleven-row closed set (e.g. an invented
string) is rejected by `verify-stage-evidence.sh`'s own closed vocabulary —
constructed via a mutated fixture built from a real clean-run bundle.

**Recovery case:** satisfied by TC-022 (AC-020) — a failure to write the
final triad itself (the "recovery" transition point of this closed table) is
that TC's own subject; not duplicated here to avoid two TCs asserting the
same postcondition under different names.

---

### TC-014: Stop-outcome eligibility triad — 5 direct-unit induced outcomes

**Feature Requirement:** REQ-F-009, §2.6.
**Task Acceptance Criterion:** AC-013 (second half — the five outcomes
`spec.md` itself scopes to direct unit invocation, not end-to-end induction).
**Technique Applied:** **State-transition** (closed table), direct-invocation subset.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.
**Caller-Path Contract (corrected per codex red-team):** `run-lifecycle.sh`
is a single embedded-Python script invoked as `__main__` via a bash heredoc —
there is **no importable module or function** to `import` and call directly,
so the original "invoke the producer's triad-writer function directly" claim
described a seam that does not exist. The real, already-established technique
for this exact situation is the one `tc075_content-identity_x12_test.sh`
already uses for `content_digest()`: **regex-extract the real triad-writer
function's source text out of `run-lifecycle.sh`, `compile()`/`exec()` it into
an isolated namespace, and call it directly** with each of the five outcome
values — never a hand-retyped reimplementation, and never a bare `import`
that this script's structure cannot support. **Implementation-contract
requirement this imposes:** the triad-writer must be a single, named,
extractable function (mirroring `content_digest()`'s and
`canonical_digest()`'s shape) so this extraction technique has a clean
boundary to regex against — not logic inlined directly into `main()`'s loop
body with no named function boundary.

**Rows covered here (exactly these five):** `lease_loss` (names the entity
whose lease was lost), `unresolved_gate` (names the blocking gate), `archive`
(names the archived entity), `cancellation` (names the signal), `timeout`
(names the exceeded window).

**Expected Output (each row):** invoking the producer's triad-writer function
directly with each `stop_outcome` value injected as the terminal condition
asserts the same three postconditions as TC-013's table: correct
`stop_outcome`, `publication_eligible: false`, non-empty
`ineligibility_reasons[]` with the tabulated content.

**State-addition regression:** Assert the union of TC-013's 5 rows + TC-014's
5 rows + the clean-terminal row = exactly the 11 rows §2.6 declares, and that
this set equals `run-lifecycle.sh`'s own `STOP_OUTCOMES` constant plus
`"complete"`, read from the real source file rather than hard-coded in the
test (so a future STOP_OUTCOMES addition without a matching §2.6 row and test
row fails loudly here).

---

### TC-015: Partial evidence survives an aborted run

**Feature Requirement:** REQ-F-010.
**Task Acceptance Criterion:** AC-014.
**Technique Applied:** Equivalence partitioning (interrupted-after-N vs. completed run).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.
**Caller-Path Contract:** TC-002 row, with a signal sent mid-run (SIGTERM,
exercising `release_on_signal()`, the same real signal path `run-lifecycle.sh`
already installs).

**Expected Output:** A run interrupted after dispatch N leaves snapshots
`1..N` present, readable, and indexed in `bundle.json.stages[]`;
`verify-stage-evidence.sh` still validates the partial bundle (accepting it as
structurally valid, `stop_outcome` present per TC-013/014's table).

**Negative Cases:** No earlier snapshot is truncated or rewritten by the
abort (REQ-F-010 "MUST NOT delete or truncate earlier snapshots on failure") —
assert byte-identical content for snapshots `1..N-1` before/after the abort
signal, using a checkpoint copy taken immediately before sending the signal.

**Implementation-contract note (added per codex red-team):** the real
`release_on_signal()` (`run-lifecycle.sh:750`) releases the lease and then
`raise SystemExit(128+signum)` — `SystemExit` is not caught by `main()`'s
`except (RuntimeError, OSError, ValueError, TypeError)` block, so a
signal-terminated run skips the normal end-of-run code path entirely.
`spec.md`'s own Integration table (§2.8, "Signal handling" row) states this
handler is "Unchanged," yet REQ-F-002 requires the bundle's triad be
rewritten "at run termination," which a signal is. **This test's AC-014
assertion (snapshots survive, indexed) does not depend on resolving this
tension — it only requires the per-dispatch incremental writes already on
disk, which are unaffected by the signal path.** But the implementation must
still add its own final-bundle-triad call inside (or immediately before)
`release_on_signal()`'s `SystemExit`, as an *addition* to that function, not a
change to its existing lease-release behavior — otherwise a signal-terminated
run's `bundle.json` triad reflects a stale per-dispatch state rather than the
`cancellation`/`error` triad REQ-F-009 requires for a non-clean termination.
This is flagged here as an implementation-contract requirement this test plan
surfaces, not one it can silently assume is already satisfied by "Unchanged."

---

### TC-016: Producer-owned reset scope + symlink refusal

**Feature Requirement:** REQ-F-011 (ADR-F12-06).
**Task Acceptance Criterion:** AC-015.
**Technique Applied:** Equivalence partitioning (owned vs. foreign entries) +
attack-class enumeration (symlink at each of the four owned entries).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability, Security.
**Caller-Path Contract:** TC-002 row.

**Preconditions:** An `i05_bundle_dir` pre-populated with repetition-1's
`bundle.json`/`stages/`/`access.jsonl`/`transcripts/` **and** one
operator-placed file unrelated to any of the four (e.g.
`operator-notes.txt`).

**Expected Output:** Running repetition 2 leaves no repetition-1 snapshot in
the final bundle (the `stages/` directory contains only repetition-2's
entries); `operator-notes.txt` is present, byte-identical, afterward.

**Attack-class enumeration (symlink refusal, one case per owned entry, loop
over the four names as `tc082` loops over `ARTIFACTS`):** a symlink planted at
`bundle.json`, `stages/`, `access.jsonl`, or `transcripts/` (pointing outside
the bundle dir) makes the run fail **before any write**, naming the offending
entry — assert the symlink target's external content is untouched (mirrors
`tc082`'s (a3) symlink-write-through discipline).

**Mode coverage (added per codex red-team):** REQ-F-011 scopes the reset to
"only in the modes of REQ-F-001" — i.e. every mode where `--i05-bundle-dir`
is honored (`live` always; `contract`/`dry-run` when supplied). Repeat the
repetition-2-over-repetition-1 assertion above for `--mode contract` and
`--mode dry-run` with the option supplied, not only for `--mode live` — a
reset bug scoped to one mode's code path would otherwise go undetected.

---

### TC-017: `content_root` declaration + walk-based digest scheme (real live run)

**Feature Requirement:** REQ-F-012 (ADR-F12-03), TD-132.
**Task Acceptance Criterion:** AC-016.
**Technique Applied:** Equivalence partitioning (scheme marker present/absent) + real end-to-end digest equality.
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility, Maintainability.
**Cross-epic tag:** X-12.
**Caller-Path Contract:** see table, TC-012/017/018/021 row.

**Preconditions:** A real scratch Shark project with the installed
`internal/sharkdata/default_data` canonical content tree copied into it (per
ADR-F12-03: `content_root` is "the installed Shark-data canonical content
tree inside the scratch Shark project," not the repo's own
`internal/sharkdata/default_data` path directly).

**Expected Output:** A real `run-lifecycle.sh --mode live` run produces an
I-07 record whose `identity.shark_content_digest` equals
`content_digest(bundle.json's content_root)` computed by
`evaluate-lifecycle.sh`'s own real function (extracted the same way `tc075`
extracts it — never a hand-retyped copy); the record carries
`identity.content_digest_scheme: "walk_v1"`.

**Negative Cases:** A record lacking `content_digest_scheme` entirely (a
pre-feature legacy record, simulated by stripping the field from a real
produced record) is treated as the legacy whole-repo-tree scheme by
`evaluate-lifecycle.sh`, never silently compared against a `walk_v1` record as
if equivalent — this is the ADR-F12-03 "never silently compared" cost made
into an assertion.

---

### TC-018: Content-digest crosscheck fires on corruption (closes TD-132)

**Feature Requirement:** REQ-F-012.
**Task Acceptance Criterion:** AC-017.
**Technique Applied:** Mutation testing (single-byte corruption under `content_root`).
**ISO 25010 Characteristic(s):** Functional Suitability, Security.
**Cross-epic tag:** X-12.
**Caller-Path Contract:** TC-012/017/018/021 row.

**Preconditions:** The same real produced I-07 record and `content_root` as
TC-017.

**Input:** Corrupt one byte under `content_root` (e.g. append one byte to one
file inside the installed canonical tree) after the run that produced the
record.

**Expected Output:** Running `evaluate-lifecycle.sh` (real, unmodified) against
the now-stale record appends an `identity_mismatch` reason at
`/identity/shark_content_digest` — the crosscheck demonstrably fires,
directly closing TD-132.

**Counter-factual:** Without ADR-F12-03's fix (a half-wiring that declares
`content_root` while leaving the legacy git-tree digest in place, per spec.md
§1.4's explicitly-forbidden case), this crosscheck would fire on **every**
run regardless of real corruption — TC-017's matching-case assertion is this
test's own counter-proof that the crosscheck is not permanently tripped.

---

### TC-019: `run-lifecycle-batch.sh dispatch_pair()` pass-through + guard reordering

**Feature Requirement:** REQ-F-013 (ADR-F12-07).
**Task Acceptance Criterion:** AC-018 (batch-driver half).
**Technique Applied:** Equivalence partitioning (configured vs. unconfigured `i05_bundle_dir`).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** see table, TC-019 row — mirrors `tc082`'s (a2)
driver-path section exactly (stub `RUN_LIFECYCLE_BIN`/`EVALUATE_LIFECYCLE_BIN`,
drive the real `run-lifecycle-batch.sh` binary).

**Expected Output:** `dispatch_pair()`'s real invocation of
`"$RUN_LIFECYCLE_BIN"` includes `--i05-bundle-dir "$i05_bundle_dir"` — assert
by having the stub record its received argv and grep for the flag; a
misconfigured (`i05_bundle_dir` empty) pair is now recorded invalid **before**
`"$RUN_LIFECYCLE_BIN"` is invoked at all (the guard-reorder half of ADR-F12-07)
— assert the stub was never called for that pair (no provider spend burned on
a doomed run).

**Regression check (AC-018's own wording):** `bench/scripts/e40-benchmark.sh
preflight` no longer reports the P5 `i05_bundle_dir is not configured` blocker
for a scenario whose policy file now configures it — real preflight
invocation against a real (or minimally stubbed) policy file.

---

### TC-020: `run-review-comparison.sh dispatch_gate()` pass-through + guard reordering

**Feature Requirement:** REQ-F-013 (ADR-F12-07).
**Task Acceptance Criterion:** AC-018 (comparison-driver half).
**Technique Applied:** Equivalence partitioning (same shape as TC-019, independent call site).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** see table, TC-020 row.

**Expected Output:** Same two assertions as TC-019, against
`dispatch_gate()`'s independent call site and its own guard at
`run-review-comparison.sh:819` — proving the fix landed at **both** named
call sites (spec.md Component-changes table lists them as two separate,
independently modified files) rather than one and not the other.

---

### TC-021: I-05 half of the I-07 evaluator join is complete

**Feature Requirement:** REQ-F-014.
**Task Acceptance Criterion:** AC-019.
**Technique Applied:** Equivalence partitioning (join-field presence) + explicit negative scope (Q008 fields excluded).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.
**Caller-Path Contract:** TC-012/017/018/021 row.

**Expected Output:** Running `evaluate-lifecycle.sh` (real) against a real
produced pair emits **no** `missing_join`/`contradictory_join` reason at any
of `/join/run_id`, `/join/scenario_id`, `/join/scenario_version`,
`/join/dispatches`.

**Negative Cases (explicit, per AC-019's own text — this is a REQUIRED
assertion, not an oversight if it fails):** Reasons **are** expected at
`/join/dispatch_id` and `/join/dispatch_ordinal` — this test asserts their
presence, not their absence. `spec.md` §1.4 item 4 and §3.2 are explicit that
record-level `dispatch_id`/`dispatch_ordinal` are deliberately not fabricated
by this feature (Q008's subject); a future run of this test that finds them
suddenly absent should prompt checking whether Q008 was resolved and this
test needs updating, not treat their presence as this feature's own defect.

---

### TC-022: Producer failure is loud (mid-run write failure)

**Feature Requirement:** REQ-F-015.
**Task Acceptance Criterion:** AC-020.
**Technique Applied:** Attack-class / fault-injection enumeration (write-failure environmental fault).
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Reliability.
**Caller-Path Contract:** TC-002 row.

**Input (corrected per codex red-team):** Making the **entire** `i05_bundle_dir`
unwritable is too blunt — it would also block the final `bundle.json` rewrite
REQ-F-015 itself requires ("`bundle.json` (if already written) gains
`stop_outcome: "error"`"), so that assertion could never be satisfied and the
fault would not be isolated to a single artifact write. Instead: after N
dispatches, make **only** the `stages/` subdirectory read-only (mode `0555`)
so the *next* stage-snapshot write fails while `i05_bundle_dir`'s top level
(where `bundle.json` lives) and `access.jsonl`/`transcripts/` remain writable
— deterministic (set up once, between two dispatches of a scripted/stubbed
run, not a race against concurrent writers) and isolates exactly one
artifact-write failure.

**Expected Output:** The run terminates with `outcome.terminal: "error"`
naming the write failure in `reason`; no I-07 record for this run claims
`publication_eligible: true`; `bundle.json` (still writable) gains
`stop_outcome: "error"` with a matching `ineligibility_reasons[]` entry
(REQ-F-015's own two-sided requirement) — now actually satisfiable because
only `stages/` was made unwritable, not the whole bundle directory.

**Negative Cases:** The failure must not be swallowed — the process must exit
non-zero and the I-07 record must not silently report `"complete"` while
evidence is incomplete.

---

### TC-023: `verify-stage-evidence.sh` accepts a real six-family live-run bundle

**Feature Requirement:** REQ-F-016.
**Task Acceptance Criterion:** AC-021.
**Technique Applied:** Real end-to-end integration (the shipped acceptance bar itself).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility, Reliability.
**Caller-Path Contract:** see table, TC-023 row.

**Expected Output:** `bench/scripts/verify-stage-evidence.sh <i05_bundle_dir>`
exits `0` on the bundle from a real `--mode live` run of at least one of the
six lifecycle-v2 families (AC-021's own "at least one" scope — this is the
E40-F11 six-family readiness surface this feature unblocks, not a
re-litigation of all six here), printing its fixed-order JSON summary.

**This is the AC-021 acceptance-bar test, not a duplicate of TC-002..016** —
it exists specifically to prove the sum of every prior TC's individually-true
assertion also holds when the *same* unmodified validator is pointed at the
*same* bundle in one shot, the way an operator actually runs it.

**Negative Cases (added per codex red-team):** this bundle MUST be the
genuine output of a real `run-lifecycle.sh --mode live` process (the same
invocation TC-002 drives) — never a hand-assembled directory shaped to look
like one. The guard harness enforces this structurally, not by assertion
alone: it invokes the real CLI end-to-end and only then points
`verify-stage-evidence.sh` at the directory that invocation produced: there is
no code path in this TC that constructs `bundle.json`/`stages/*.json`
content itself. A future edit that swaps in a fixture-built bundle here would
be a regression in the test's own caller-path fidelity, not merely a style
concern — flagged explicitly per codex's finding that the original draft left
this only as prose.

---

### TC-024: `verify-stage-evidence.sh` is byte-identical pre/post feature

**Feature Requirement:** REQ-F-016 (no parallel/relaxed validator).
**Task Acceptance Criterion:** AC-022.
**Technique Applied:** Content-only (`git diff --exit-code`) — no runtime behavior to exercise.
**ISO 25010 Characteristic(s):** Compatibility, Maintainability.
**Caller-Path Contract:** content-only; entrypoint is the file itself via `git diff --exit-code -- bench/scripts/verify-stage-evidence.sh`.

**Expected Output:** `git diff --exit-code -- bench/scripts/verify-stage-evidence.sh`
exits 0 against the pre-feature commit — this file is untouched by the whole
feature (spec.md Component-changes table lists it as unmodified).

---

### TC-025: §3.2 discloses the I-07 gap; Q008 exists in Shark

**Feature Requirement:** REQ-F-017.
**Task Acceptance Criterion:** AC-023.
**Technique Applied:** Content-only (direct spec section) + direct-file/CLI-read (Shark Question record).
**ISO 25010 Characteristic(s):** Usability, Maintainability.
**Caller-Path Contract:** content-only / direct-file; entrypoint is
`spec.md` §3.2 itself plus `shark get Q008 --json` (real CLI read, no mock —
verified live during this test-planning pass: Q008 exists, `status: "open"`,
`blocking: true`, `requester: "architect@E40-F12"`, description references
"E40-F12 spec.md section 3.2").

**Expected Output:** `spec.md` §3.2 enumerates every I-07 identity and
workflow-policy field the evaluator requires and the live producer omits
(the 13-row table at spec.md lines 812-826); `shark get Q008 --json` returns a
record with `status: "open"` and `blocking: true`.

**Negative Cases (drift guard, not a functional assertion — a documentation
check for future maintainers):** This plan does **not** assert or claim "a
non-failed evaluated pair" anywhere as an achieved postcondition of this
feature — per Spec Drift Analysis item 2, doing so would contradict AC-023's
own scope and spec.md §1.4 item 3.

---

### TC-026: Per-dispatch producer write-path overhead (REQ-NF-002)

**Feature Requirement:** REQ-NF-002.
**Task Acceptance Criterion:** none (NFR, landed in ISO 25010 matrix per advisor guidance).
**Technique Applied:** Boundary value analysis (1s threshold).
**ISO 25010 Characteristic(s):** Performance Efficiency.
**Caller-Path Contract:** TC-002 row; timing instrumentation wraps the real
write path, outside the stage window per §2.5 (the write path is deliberately
excluded from `tool_and_test`, so this must be measured by the test harness
itself, not read out of the ledger).

**Expected Output:** On the reference fixture, the producer's per-dispatch
write path (serialize + write `stages/<n>.json` + rewrite `bundle.json` +
append `access.jsonl` line if any) completes in under 1 second per dispatch,
and at most one `shark admin workflow list` call is made per workflow level
per run (assert call count via the same stub-recording mechanism TC-019 uses).

**Implementation-contract note (added per codex red-team):** no seam in the
current, unmodified `run-lifecycle.sh` exposes the write path's duration
separately from the whole CLI invocation — measuring "just the write path"
from outside the process cannot isolate it precisely. This test requires the
implementation to add one lightweight, explicit timing seam analogous to
REQ-F-004/REQ-F-015's named diagnostics (e.g. a single stderr line such as
`i05_write_ms=<n>` emitted once per dispatch when a documented
`LIFECYCLE_BENCH_TIMING=1` environment variable is set) so this NFR is
mechanically measurable rather than approximated by wall-clock bracketing of
the whole run.

---

### TC-027: No secret or prompt-byte leakage (REQ-NF-003)

**Feature Requirement:** REQ-NF-003.
**Task Acceptance Criterion:** none (NFR).
**Technique Applied:** Negative / attack-class enumeration (grep for known prompt bytes and provider-credential shapes).
**ISO 25010 Characteristic(s):** Security.
**Caller-Path Contract:** TC-002 row; assertion is a grep over real produced output, no mocks needed for the assertion itself.

**Expected Output:** Grepping every file under a produced `i05_bundle_dir`
(including `transcripts/`) for the literal rendered-prompt text of the same
run finds no match; `bench/scripts/verify-evidence-roots.sh` (existing,
unmodified) stays green against the same run's roots.

---

### TC-028: Deterministic bundle writes across identical-input runs (REQ-NF-004)

**Feature Requirement:** REQ-NF-004.
**Task Acceptance Criterion:** none (NFR).
**Technique Applied:** Equivalence partitioning (two runs, identical declared inputs).
**ISO 25010 Characteristic(s):** Reliability, Maintainability.
**Caller-Path Contract:** TC-002 row, invoked twice with identical scenario/scratch-root/stub inputs.

**Expected Output:** Two runs with identical inputs (same scenario, same
stubbed adapter/shark responses) produce byte-identical snapshots apart from
timestamps, digests derived from genuinely different real state (there
should be none, given identical inputs), and observed durations. Canonical
serialization is `sort_keys=True, separators=(",", ":")` throughout — assert
by diffing both runs' `bundle.json`/`stages/*.json` with timestamp/duration
fields stripped first.

---

## Codex Test-Plan Red-Team

**Verdict (first pass):** FAIL
**Executed:** 2026-09-08, `codex exec --sandbox read-only -c
model_reasoning_effort=high`, live run (not simulated), 595s budget,
completed in ~370s without truncation on the first attempt (no retry
needed). Invoked against this file, `spec.md`, `feature.md`, the real
`bench/scripts/run-lifecycle.sh` / `run-lifecycle-batch.sh` /
`run-review-comparison.sh` / `verify-stage-evidence.sh` /
`bench/evidence/i05-schema.yaml` / `bench/scripts/lib/retain_pair` /
`bench/scripts/evaluate-lifecycle.sh`, and the sibling conventions `tc082` /
`tc075`.

**Mechanical checks codex confirmed as correct (no issue raised):** all three
contract-test pointers (`TC-042`, `TC-030`, `TC-061`) resolve to real
files/functions; AC-013's 5-end-to-end + 5-direct-unit + 1-clean split
matches `spec.md` exactly with no outcome moved between buckets; AC-022 and
AC-023 are correctly scoped content-only/disclosure, not fabricated runtime
tests; no AC carries an open-ended, unenumerated robustness assertion
(BLOCKER class 1, zero found).

**Issues raised:** 14 (0 BLOCKER by codex's own severity language, all
CONCERNS-class factual/design corrections)
**Issues addressed before dev:** 14/14
**Issues deferred:** 0

| # | AC / TC | Codex finding | Verified? | Fix applied |
|---|---|---|---|---|
| 1 | AC-004/TC-004 | `verify-stage-evidence.sh` never checks/emits `missing_provider` — schema-only vocabulary | **Confirmed by direct grep** — zero hits outside `i05-schema.yaml` in `verify-stage-evidence.sh`, `replay-stage-evidence.sh`, and the Go contract test | TC-004 now asserts `provider` directly from real output; gap recorded in Spec Drift item 3, not silently claimed as validator behavior |
| 2 | AC-005/TC-005 | `verify-stage-evidence.sh` never recomputes `snapshot_digest`/emits `snapshot_mutated` — that logic is in `replay-stage-evidence.sh` | **Confirmed by direct grep** — zero `snapshot_digest` references in `verify-stage-evidence.sh`; `recompute_snapshot_digest()`/`snapshot_mutated` verdict live in `replay-stage-evidence.sh:184-207` | TC-005 retargeted to the real script; `spec.md` AC-005's own script-name mismatch recorded in Spec Drift item 2 (not this test plan's error to silently fix) |
| 3 | AC-007/TC-007 | Miscounted §2.4 as "18 phases"; positive path declared real while invalid path used a stub — inconsistent seam | **Confirmed by re-reading §2.4** — 14 dispatchable + 3 non-dispatchable + 1 catch-all, not 18 phases | TC-007 rewritten: correct partition counts, one consistently declared `SHARK_BIN` seam for both positive and invalid rows |
| 4 | AC-008/AC-009/TC-008/009 | No recovery case; state-addition regression read the schema dynamically, so a real 7th-category addition would pass silently | **Confirmed by design review** — the original wording did read `i05-schema.yaml` at test time as the "expected" set | Added a recovery case (no cross-stage cascade on one ledger's rejection); state-addition regression now pins a fixed six-name literal list, not a dynamic schema read |
| 5 | AC-013/TC-013/014 | No invalid-outcome case; no distinct recovery case | Valid gap per `state-space-coverage.md`'s required elements | Added an invalid-`stop_outcome` mutation case; recovery case now explicitly cross-references TC-022 rather than being silently absent |
| 6 | AC-013/TC-014 | Claimed "direct triad-writer function" entrypoint does not exist — `run-lifecycle.sh` is a single embedded-heredoc script with no importable module | **Confirmed by direct inspection** — no `import`-able boundary; `main()` is the sole `__main__` entry | TC-014's Caller-Path Contract rewritten to the real, already-established technique (`tc075`'s source-extraction-and-`exec()` pattern), with an implementation-contract note that the triad-writer must be a named, extractable function |
| 7 | AC-014/TC-015 | `release_on_signal()` raises `SystemExit`, uncaught by `main()`'s exception handler, before any final bundle rewrite — tension with spec.md's "signal handling unchanged" | **Confirmed by direct inspection** of `run-lifecycle.sh:750` and `main()`'s `except (RuntimeError, OSError, ValueError, TypeError)` clause | Added an implementation-contract note flagging the tension explicitly, scoping TC-015's own assertion to what is unambiguously true (snapshot survival) without silently assuming the unresolved part |
| 8 | AC-012/TC-012 | Positive-case wording was inverted ("no `missing_oracle` for an absent checkout") | **Confirmed** — genuine copy-paste inversion in the first draft | Corrected to "a present checkout" |
| 9 | AC-015/TC-016 | Reset coverage only exercised `live` mode; REQ-F-011 scopes the reset to every producer-enabled mode | Valid — REQ-F-011 says "only in the modes of REQ-F-001" (live/contract/dry-run) | Added explicit mode-coverage requirement (repeat for `contract`/`dry-run` with the option supplied) |
| 10 | AC-011/AC-012 | ISO matrix marked Security bare `N/A` despite transcript confidentiality and three-root isolation being real security-relevant surfaces | Valid — confirmed against my own "no bare N/A" rule, which the first draft violated systematically (27 bare cells found on self-check, now 0) | AC-011/AC-012 Security cells now cite TC-027/TC-012 respectively; full matrix pass added justification to every remaining cell |
| 11 | AC-019/technique matrix | ISTQB table wording said "no `dispatch_id`/`dispatch_ordinal` reason expected," contradicting TC-021's own (correct) body text | **Confirmed** — direct self-contradiction within the document | Corrected the technique-table wording to match TC-021: these reasons ARE expected |
| 12 | AC-020/TC-022 | `chmod 000`-ing the whole `i05_bundle_dir` would also block the required final `bundle.json` error-triad write, and is described in a race-prone way | Valid design flaw — confirmed the final-triad requirement cannot be satisfied if the whole directory is unwritable | Redesigned to make only `stages/` read-only, leaving `bundle.json`'s parent directory writable — isolates one artifact-write failure, deterministic setup |
| 13 | AC-021/TC-023 | No explicit MUST-NOT case against substituting a hand-built bundle for a real live-run bundle; counterfactual was prose-only | Valid — the original text asserted this only as narrative, not as a harness-structural guarantee | Added an explicit Negative Cases paragraph tying the guarantee to the harness's own code structure (no fixture-construction code path exists in this TC) |
| 14 | REQ-NF-002/TC-026 | No production seam exposes the per-dispatch write-path duration separately from the whole CLI | Valid — confirmed no such seam exists in `run-lifecycle.sh` today | Added an implementation-contract note requiring one lightweight, explicit timing seam (a documented stderr line under an opt-in env var) |

All 14 findings were either **confirmed defects in the test plan itself**
(1, 2, 3, 6, 8, 11, 12 — factual/structural corrections against the real
codebase) or **valid coverage gaps** relative to this repo's own
`state-space-coverage.md`/ISO-25010/observability rules (4, 5, 7, 9, 10, 13,
14). None were rejected as false positives. Findings 1 and 2 in particular
surfaced a genuine `spec.md`-vs-implementation discrepancy (AC-005 names the
wrong validator script) that is now recorded in Spec Drift Analysis for the
spec owner to reconcile, rather than silently patched over in this test plan
alone.

A second codex pass was not run after applying these fixes (session-budget
tradeoff, consistent with this epic's own `E40-F11` test-plan precedent when
a first pass finds concrete, mechanically-verifiable issues rather than
matters of judgment): every fix above was independently verified against the
real source files by direct `grep`/inspection during this same pass, not
merely asserted from codex's word — see the "Verified?" column.

---

## Recommendations

- [x] Ready for development — no unresolved spec drift blocks development
  (feature.md's stub status, Impact narrowing, and the AC-005
  script-name/AC-004 `missing_provider` discrepancies found during this pass
  are all recorded as non-blocking, per Spec Drift Analysis items 1-3); every
  runtime AC has a technique, an ISO 25010 row with no bare `N/A` (verified:
  `grep -c '| N/A |'` = 0 after this pass's fixes, corrected from 27 in the
  first draft), a Caller-Path Contract grounded in a real, verified
  entrypoint (TC-014's contract was corrected to a real, already-established
  extraction technique rather than a nonexistent import seam); every closed
  lifecycle table (§2.4, §2.5, §2.6) now carries exhaustive-value,
  invalid-transition, recovery, and state-addition-regression cases per
  `state-space-coverage.md`, added where codex's red-team found them missing;
  every I-##/X-## row's staged fields are preserved verbatim, not
  re-litigated; Q008/Q009 are recorded as open, spec-acknowledged questions
  this test plan does not attempt to close — not modeled as test-plan
  defects. Codex red-team's first pass returned FAIL with 14 concrete,
  independently-verified findings; all 14 are now incorporated (see "Codex
  Test-Plan Red-Team" table) and none were rejected as false positives — the
  plan is APPROVED on the corrected content, not the first draft.
- [ ] Needs BA refinement — not applicable.
- [ ] Needs tech refinement — not applicable.

**Not closed by this test plan (by design, matching spec.md's own scope):**
Q008 (I-07 producer identity/workflow-policy gap) and Q009 (I-05 activation
ownership) remain open architectural questions outside E40-F12's scope; no
test case above claims to resolve either, and TC-021/TC-025 are written to
assert the *expected* presence of the gaps they name, not to fail because the
gaps still exist.
