---
feature_key: E40-F11-six-family-lifecycle-benchmark-readiness-and-cover
epic_key: E40
doc_type: test-plan
spec: spec.md
feature_spec: feature.md
research_report: research-report.md
supersedes: test-plan.md (2026-09-04, second-rework projection, Codex verdict FAIL on 14 classes, status REJECTED)
status: APPROVED
---

# Test Plan: E40-F11 — Six-family lifecycle benchmark readiness and coverage

**Created:** 2026-09-04 (regenerated as a mechanical projection of `spec.md`
Part 7 **as revised by spec.md's "Third rework, 2026-09-04" banner** —
§7.1's 54-row register with §7.1a's closed inventories, §7.2's
parser-verified CLI seams and corrected offline-path argv, §7.3.2's
12-class x 4-ceiling BVA, §7.3.5's Y1-Y27 type/partition surface, §2.3.3a's
call-plan arithmetic, and §7.7.2/§7.7.3's syscall-observation proofs).
**Feature spec:** `spec.md` Parts 1-7; `feature.md` (Goal, Observed evidence,
Acceptance scenarios, Required validation)
**Status:** APPROVED. The prior file was REJECTED against a "second rework"
spec after an independent Codex red-team (thread `01a06dd1-…`) found 14
defect classes and QA recorded a class-ownership routing table rather than
regenerate against a still-contradictory spec (the correct call, per that
file's own note). `spec.md`'s current banner ("Third rework") states it closes
every class the routing table assigned to specification (1-9, including
Q007/class 9 via Part 5's three complete conditional matrices). This file
closes the four classes the routing table assigned to test-design (10, 12,
13, 14) and is re-submitted to Codex below.

## Change classification

This feature changes `bench/` shell/Python scripts, `package.yaml` fixture
files, two new scenario packages, and one Go contract test
(`tests/contracts/e40_i04_scenario_contract_test.go`) — all deterministic
runtime behavior. It is **not** a prompt/skill/documentation-only change (the
doc-surface edits in AC-F11-22 are a byproduct of a behavior change). Caller-
path contracts are mandatory for every runtime test case (Step 5.8).
`internal/` and `cmd/` are untouched (spec.md §1.4); no repository-real-DB
test is required — every test drives `bench/` scripts and fixtures against an
**external scratch project** (`scripts/shark-scratch-env.sh`), never
`shark-tasks.db` (REQ-NF-001, AC-F11-41, AC-F11-41a). The live database is a
**read-only-by-nothing** observation target: §7.7.2 replaces the prior
hash/copy proof (itself a violation) with syscall observation, so no test in
this plan opens, reads, hashes, or copies the live file (class 5, closed).

Test case identifiers below are `spec.md` Part 7's own: the `AC-F11-##` rows
of §7.1 project 1:1 into `TC-<AC>` test cases, each pointing at the shell test
file `spec.md` §1.2/§7.1 already names (`tc099`…`tc113`) or the extended Go
contract test. This plan invents no new numbering. Its contribution is the
per-AC caller-path contract (retargeted to the top-level CLI entrypoint,
class 12), the per-AC technique and ISO 25010 tags (class 13, no more
multi-AC grouped rows), the recomputed isolation counts (class 10), and the
I-##/X-## cross-references — plus verbatim reproduction of the corrected
§7.1/§7.1a register, §7.2 CLI seams and offline path, §7.3 enumerations,
§7.4 isolation table, §7.5 ISO matrix, §7.6 UAT reconciliation, and §7.7
zero-provider-call proofs.

---

## Spec Drift Analysis

### Drift Findings

None against the current `spec.md` (third rework). Four defect classes remain
this plan's own to close, per the prior REJECTED file's routing table:

1. **Class 10 — incorrect counts, now recomputed against the current §7.4
   table** (which itself carries a stale summary line — see below). Counting
   the seven operation rows x six root/direction columns in §7.4's current
   table gives **18 `allow` / 15 `deny` / 9 `n/r`** — not the "20 `allow`, 10
   `deny`, 12 `n/r`" the prose beneath the table still states. That prose line
   was not corrected by the third rework (it is a leftover from before the
   `n/r`-versus-`—` fix, class 8); the table itself, which is authoritative
   per §7.1a's rule that a criterion resolves to a closed set, is correct.
   This plan uses the table-derived 18/15/9, flags the stale prose line as a
   documentation-only residual defect for specification to correct on its
   next pass, and does not block on it — the test plan's own count is
   independently verifiable by re-running the same tally. AC-F11-18's eighth
   evidence field (`limit`) is present in §2.3.3/§7.1 row 18 (8 fields, all
   named) — that half of class 10 is already closed by the third rework.
2. **Class 12 — production observability.** §7.2 now states the load-bearing
   rule explicitly: every test enters through `bench/scripts/e40-benchmark.sh`
   (a thin `exec` into the Python module), never an internal function or inner
   script, except the rows §7.1 marks `n/a (static)` or naming a standalone
   verifier (`verify-retention-manifest.sh`, `verify-stage-evidence.sh`,
   `verify-replay-result.sh`, `verify-evidence-roots.sh` — all of which are
   themselves operator-invocable, not internal helpers). This plan's
   Caller-Path Contracts (below) are retargeted to that rule: every runtime
   TC names the `e40-benchmark.sh <subcommand>` entrypoint with the real argv,
   not a leaf script, except the named verifier exceptions.
3. **Class 13 — per-AC technique/ISO rows.** The prior grouped rows (e.g.
   "01, 01a, 01b, 02") hid that different ACs in one group can carry different
   sub-techniques or different N/A justifications. This plan's ISTQB and ISO
   25010 tables carry one row per AC (54 rows each), not per group.
4. **Class 14 — non-durable prior review claim.** Closed by this file's own
   frontmatter: current invocation identity and verdict only, no carried-over
   PASS claim.

Classes 1-9 (specification-owned, including Q007/class 9) are closed by
`spec.md`'s third rework: §7.1a (class 1), §7.3.2's 12-class BVA (class 2),
§7.2's corrected seams/argv (class 3), Y7/AC-F11-25b resolving the
`required_terminal_states` array-vs-mapping conflict as mapping (class 4),
§7.7.2's syscall proof (class 5), §7.7.3's provider-invocation surface V1-V7
(class 6), the fourth ceiling threaded through §2.3.3/§7.1 row 06a/§7.2's
gate G-7 (class 7), §2.3.3a's call-plan formula (class 8), and Part 5's three
complete conditional matrices for Q007 covering ownership, re-acceptance,
closure keys, and UAT consequences for all three options (class 9). Per
spec.md Part 5, Q007 blocks **feature review**, not specification, task
generation, implementation, or test planning — "every implementation artifact
this specification describes is identical across all three options" — so
this test plan's verdict is not gated on Q007's resolution.

### Traceability Matrix

Reproduced from `spec.md` §1.1's requirement→gate table, extended with the AC
range and the projected TC identifiers (`spec.md` §1.2/§7.1 "Test:" /
"Seam:" references):

| Feature Requirement | Epic gate(s) | AC range | Test file | Covered? |
|---|---|---|---|---|
| REQ-F-001 | G12, G15 | AC-F11-01, -01a, -01b, -02 | `tc099_retained_root_immutability_test.sh` | Yes |
| REQ-F-002 | G9, G14 | AC-F11-03..05 | `tc100_admitted_fixture_checkout_binding_test.sh` | Yes |
| REQ-F-003 | G10, G15 | AC-F11-06, -06a, -06b, -07, -08 | `tc101_replay_preparation_spend_gate_test.sh`, `tc102_replay_preparation_verified_result_test.sh` | Yes |
| REQ-F-004 | G15 | AC-F11-09..12 | `tc103_preflight_fail_closed_test.sh`, `tc104_preflight_skipped_scenario_blocker_test.sh` | Yes |
| REQ-F-005 | G11, G15 | AC-F11-13..15 | `tc105_route_resolution_without_live_artifacts_test.sh` | Yes |
| REQ-F-006 | G12, G15 | AC-F11-16..18 | `tc106_provider_call_plan_test.sh` | Yes |
| REQ-F-007 | G8 | AC-F11-19..22 | `tc107_six_family_admission_test.sh`; `e40_i04_scenario_contract_test.go` TC-030 extended | Yes |
| REQ-F-008 | G8, G10 | AC-F11-23, -24 | `tc057_non_applicable_record_test.sh` extended | Yes |
| REQ-F-009 | G12, G13 | AC-F11-25, -25a, -25b, -26, -27 | `tc108_expected_entity_graph_contract_test.sh` | Yes |
| REQ-F-010 | G8, G13 | AC-F11-28..32 | `tc109_task_root_scenario_test.sh`; `tc034_admission_determinism_test.sh` (run twice) | Yes |
| REQ-F-011 | G8, G11, G13 | AC-F11-33..37 | `tc110_epic_root_scenario_test.sh` | Yes |
| REQ-F-012 | G9 | AC-F11-38, -38a, 39..41, -41a | `tc111_scenario_root_isolation_test.sh` | Yes |
| REQ-F-013 | G14, G16-18 | AC-F11-42..44 | `tc112_six_family_reporting_test.sh` | Yes |
| REQ-F-014 | G14, G15, G19 | AC-F11-45, -46 | `tc113_baseline_capture_gate_sequence_test.sh` | Yes |
| REQ-NF-001, -005 | G9 | AC-F11-41, -41a | `tc111`; §7.4 isolation table | Yes |
| REQ-NF-002 | G12, G15 | AC-F11-06, -06a, -06b, -08 | `tc101`; §7.3.2 12-class BVA; §7.7 two-proof zero-call surface; `tc053`/`tc079`/`tc080` extended | Yes |
| REQ-NF-003 | G12 | AC-F11-01, -01a, -01b, -02 | `tc099`; §7.3.1 decision table | Yes |
| REQ-NF-004 | G14, G19 | AC-F11-05, -18 | `tc034` extended to six packages; §7.3.5 identity-field surface | Yes |
| REQ-NF-005 | G9 | — | `verify-evidence-roots.sh`, `tc043_root_policy_isolation_test.sh` extended; §7.4 | Yes |

Every `feature.md` §Acceptance scenario maps onto its AC set:

| feature.md acceptance scenario | spec.md AC(s) |
|---|---|
| Reject incidental submodule identity | AC-F11-03, -04, -05 |
| Block a missing replay | AC-F11-09, -12 |
| Pass a complete six-family preflight | AC-F11-09, -16..18 |
| Complete a task-root lifecycle | AC-F11-28..32 |
| Complete an epic-root lifecycle | AC-F11-33..37 |
| Publish a six-family baseline | AC-F11-45, -46 |

## Acceptance Criteria Review

### Ambiguity Findings

None outstanding. §7.1a's rule ("no criterion may carry `any`/`every`/`no
...`/`all`/`non-regression` without a closed set or a pointer to one") is
verified against the register: every row that previously carried an
unbounded quantifier now has a §7.1a closed-inventory row.

**AC-F11-27 "no third private copy"** is a bounded static check (`n/a
(static)` seam, "reader count == 2, set equality" observable), carried as
TC-27 below with its `internal-only`/`content-only`-adjacent static
justification (it drives a `grep` over the file tree, not a CLI seam).

**AC-F11-41's "not even to demonstrate that a write would be detected"** is a
concrete two-part proof: positive syscall-observed absence of any write to
the live file's resolved path; negative proven only on a disposable surrogate
copy at a temp path, never the live file. TC-41 states this explicitly.

### Missing Coverage

None found. §7.1's register states 54 rows (52 from the second rework plus
AC-F11-06b and AC-F11-25b); this plan's own count of AC rows below reproduces
54 exactly.

## ISTQB Technique Application (per AC — class 13, one row per criterion)

| AC | Technique(s) Applied | Rationale |
|---|---|---|
| 01 | Decision table (§7.3.1: 4-condition x 9-seam = 36 rejections + 9 allows) | Guard is a conjunction across seam/status/HEAD/registration; R2-R4 are the pre-rework conjunction-bug regression cells |
| 01a | Equivalence partitioning (grep hit/no-hit) + static contract check (in-repo vs out-of-repo `root_path`) | No-path-in-source is a set-membership check over `bench/` source text |
| 01b | State/structural invariant (whole-tree manifest, 5 mutation kinds) | Detects file modify/add/remove/rename/symlink-retarget as distinct classes, each independently falsifiable |
| 02 | Equivalence partitioning (digest match/mismatch) + static cross-link check | Binary identity condition over evidence-file vs. registry-file digests |
| 03 | Equivalence partitioning (SHA match/mismatch) | Checkout HEAD equals vs. differs from admitted `base_sha` |
| 04 | Equivalence partitioning (two named collectors at two named SHAs) | Regression-shape check: reproduces a previously observed failure mode at the wrong SHA |
| 05 | State transition (full run vs. mid-run-aborted) | Structural invariant (pristine submodule) must hold across both completion states |
| 06 | Decision table (ack present/absent x ceilings present/absent) | AC-F11-06 is the "no ack" branch of §7.2's 4-row invocation table |
| 06a | Boundary value analysis, per-seam presence (§7.1 row 06a, 8 seams) | Each of 8 seams individually reverted; ceiling must be enforced end-to-end, not just at one entrypoint |
| 06b | Boundary value analysis (§7.3.2, 4 ceilings x 12 classes) | Canonical numeric BVA extended to non-finite/lossy classes (NaN, ±inf, overflow, fractional-under-integer) |
| 07 | Equivalence partitioning (verified result vs. failing result) | `verify-replay-result.sh` accept/reject is a binary partition over the producer's output |
| 08 | Equivalence partitioning over §7.1a's closed 11-case argument-combination inventory | Replaces the unbounded "any argument combination" with a domain from the actual parser declaration |
| 09 | Decision table (§7.3.3, 7 independently falsifiable conditions) | `status=="pass"` is a 7-way AND; each condition needs its own falsification case |
| 10 | Equivalence partitioning over §7.1a's closed 5-value `diagnostics[]` type set x `spend_eligible` x 3 statuses | Replaces "never coexists with pass" with an enumerated 30-cell domain |
| 11 | Decision table (root_key set/unset x scratch_root set/unset/absent) | 3 independently falsifiable resolution-blocking conditions |
| 12 | Decision table (4 independent blocker causes, reproducing the 2026-09-04 state) | Falsify each cause alone to confirm the blocker count drops by exactly one |
| 13 | Equivalence partitioning (7 required schema fields, each absent in turn) | Per-field schema completeness on the resolve-route record |
| 14 | Equivalence partitioning (`route_defect` vs `artifact_deferred`) | Two-cause classification that must never be conflated in either direction |
| 15 | Equivalence partitioning (live evidence vs `route_resolution_only`) | Binary acceptance partition at the verifier boundary |
| 16 | Equivalence partitioning (8 required call-plan fields, each absent in turn) | Per-field schema completeness on the call plan |
| 17 | Boundary value analysis (§7.3.4 B1-B8) + attack-class counterfactual (B7, maximum-based implementation) | Canonical formula-verification BVA; B7 is load-bearing against a subtly wrong (max-based) implementation |
| 18 | Equivalence partitioning (8 evidence fields, each blanked in turn) | Per-field completeness on `call_plan.evidence[]` |
| 19 | Equivalence partitioning (6 valid + 8 invalid family values, closed enum) | Canonical closed-enum EP |
| 20 | Equivalence partitioning (8 schema-carrying files at one version) + boundary (a third version) | Single-supported-version invariant, tested per file and against an out-of-domain third value |
| 21 | Equivalence partitioning (same 6+8 set, dual-implementation agreement) | Cross-language compatibility EP: Go validator and shell script must partition identically |
| 22 | Static contract check (closed grep inventory, §7.1a row 22) | Enumerable live-surface check with one documented, deliberate exclusion |
| 23 | Decision table (§7.3.8, 6 families x 5 stages = 30 cells) + boundary (42 negative cases N-A..N-F) | Per-family-per-stage boolean with a required, family-specific, non-generic reason field |
| 24 | Equivalence partitioning (`replay_reference` required-on-feature / forbidden-elsewhere) | 6-way partition by family |
| 25 | Decision table (required/optional/forbidden by root_family, §7.3.6) + attack-class enumeration (Y1-Y27 type confusion) | 3-way requiredness across 5 non-epic/task-etc. families; every field of the block gets an explicit wrong-type negative |
| 25a | Boundary value analysis (§2.3.3a fallback formula) + decision table (both-sources-present rejection) | Fallback arithmetic plus an explicit conflict-resolution rule (reject, not silent tighter-wins) |
| 25b | Equivalence partitioning (key-set equality: descendants[] families vs. mapping keys) + decision table (missing key / extra key / invalid status-family pair) | `required_terminal_states` key-completeness is a set-equality property, not a per-field type check |
| 26 | Attack-class enumeration (16 I-01 field names, each injected as a negative) | Dataflow-boundary check: no I-01 field name may leak into `expected_entity_graph` |
| 27 | Static dataflow assertion (closed reader-set equality, §7.1a row 27) | "No third private copy" resolved to exact file-set equality, not a spot check |
| 28 | Decision table (5 declared properties, each negated in turn) | Per-property falsification on package admission |
| 29 | State transition (§7.3.7 T1-T6, draft->development->completed) | Explicit legal/illegal transition set plus a backward `fail` outcome route |
| 30 | Equivalence partitioning (oracle gated on terminal vs. not) + closed inventory (3 named test functions, §7.1a row 30) | "Non-regression" resolved to the package's own admitted P2P set, fixed at admission time |
| 31 | Decision table (3 admission-ledger booleans, each negated in turn) | `base_outcome`/`reference_outcome`/P2P-green are 3 independent gating conditions |
| 32 | Equivalence partitioning (0 descendants expected vs. 1 injected) | Leaf-entity structural invariant |
| 33 | Equivalence partitioning (graph present/omitted) + boundary (`min > max`, G10) | Required-field check plus an admission-time boundary rejection |
| 34 | State transition (8 epic stages, each omitted in turn) | Ordered-sequence completeness check |
| 35 | Equivalence partitioning (§7.1a row 35, 4 named distinctness components) | "Distinct from py-feature-recurring-tasks" resolved to 4 named comparable fields |
| 36 | Decision table (§7.3.6 G3-G6, G8, G9: 6 structural violations) | Each violation must produce exactly one distinct named reason |
| 37 | Decision table (3 admission outcomes, each negated in turn) | Same shape as AC-F11-31, applied at epic root |
| 38 | Equivalence partitioning (key set equality against `scenario_id` set) + counterfactual (reverted family-keyed lookup) | Uniqueness property with a regression counterfactual |
| 38a | Boundary value analysis (duplicate-family collision, one positive pair) | The synthetic 7th package is the minimum case that exercises the collision |
| 39 | Equivalence partitioning (pairwise-disjoint entity-key sets across 6 scenarios) | Structural invariant over the full seed set |
| 40 | Attack-class enumeration (per-create-invocation provenance check + 1 counterfactual) | Every `shark create --json` call site is asserted to read `.key`, never construct one |
| 41 | Boundary/state proof (write-syscall inventory, §7.1a row 41: 9 seams x 4 syscalls) | Live-file write-absence is a closed syscall-surface property, proven via syscall observation not content hashing |
| 41a | Boundary/state proof (read-syscall inventory, §7.1a row 41a: 9 seams x (2 syscalls + 17 env names)) + counterfactual | Live-file read-absence proven the same way, plus a non-existent-temp-path counterfactual |
| 42 | Equivalence partitioning (attestation-count == policy-selected-family-count set equality, §7.1a row 42) | Explicitly rejects a hardcoded-6 test that would pass on a smaller batch |
| 43 | Equivalence partitioning ("0 findings" vs "not measured" vs "present", 3-way) | Non-collapsing three-way outcome per family |
| 44 | Decision table (4 named shape mutations, each detected) | Mutation testing over the JSON aggregate shape against a recorded reference |
| 45 | State transition (§7.2, 12 named gates G-1..G-12, each bypassed in turn) | End-to-end gate sequence, each gate independently falsifiable |
| 46 | Decision table (4 named identity components, each mismatched alone) | Comparison-identity rejection matrix |

ACs without a technique annotation = untestable spec. None found — all 54
rows above carry at least one named technique (class 13 closed: one row per
AC, not per group).

## ISO 25010 Coverage Matrix (per AC — class 13, one row per criterion)

Column set matches the sibling E40-F10 test plan; no edition or column
change mid-epic. Reproduced and expanded from `spec.md` §7.5's grouped rows
into one row per AC — where the group's own justification did not
distinguish sub-criteria, the same cell content is repeated per AC in that
group (this is the mechanical expansion class 13 requires: uniform content
verified individually, not hidden behind a shared row). `N/A` always carries
a reason; no cell is empty.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| 01 | ✅ 36-cell decision table | N/A — constant-time lookup | ✅ shared YAML loader | ✅ names `registry_id`+`root_path` | ✅ 5 mutation kinds detected | ✅ prevents evidence destruction | ✅ one choke point (`ensure_external_operator_root`) | ✅ registry holds the only absolute path |
| 01a | ✅ grep + registry-load check | N/A — static check | ✅ shared loader | ✅ names offending entry | N/A — static | ✅ no operator path leaks into source | ✅ single guard | ✅ source stays host-agnostic |
| 01b | ✅ 5 mutation kinds | N/A — offline verification | ✅ shared manifest format | ✅ differing path named | ✅ whole-tree manifest invariance | ✅ prevents undetected evidence tampering | ✅ one verifier script | ✅ SHA-256, no host dependency |
| 02 | ✅ digest equality | N/A — static | ✅ cross-linked registry/evidence | ✅ names altered field | N/A — static | ✅ digest tamper-evident | ✅ one evidence file | N/A — doc-only |
| 03 | ✅ TC-03 pos+neg | N/A — one checkout reused per `(fixture,sha)` | ✅ shared checkout input | ✅ abort names both SHAs | ✅ collector count == 0 on mismatch | ✅ read-only checkout | ✅ interface unchanged | ✅ git-only |
| 04 | ✅ TC-04 regression shape | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ shared collector | ✅ names both module errors | ✅ deterministic reappearance at wrong SHA | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ existing collector reused | ✅ git-only |
| 05 | ✅ TC-05 full+aborted | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ shared checkout | ✅ `fixture_checkout` field present | ✅ pristine after abort | ✅ read-only checkout | N/A — no shared/duplicated logic surface distinct from the single code path already covered elsewhere in this row | ✅ git-only |
| 06 | ✅ TC-06 4-ceiling-absent variant | N/A — bounded by scenario count | ✅ `verify-replay-result.sh` reused | ✅ ceilings named pre-spend | ✅ non-zero exit, no spend | ✅ §7.7 two-proof zero-call (never `provider_calls` literal) | ✅ shared `require_positive` | ✅ offline stub |
| 06a | ✅ 8-seam BVA | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same seam, 8 callers | ✅ field name printed per seam | ✅ 8 distinct named failures | ✅ ceiling enforced at every spend seam | ✅ one semantics, 8 call sites | ✅ offline |
| 06b | ✅ §7.3.2 12-class x 4-ceiling BVA | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ shared `require_positive` | ✅ distinct message per class | ✅ NaN/inf/overflow/fractional rejected — no silent truncation | ✅ prevents a disabled or mislabelled spend limit | ✅ one hardened helper | ✅ pure arithmetic, no host dependency |
| 07 | ✅ TC-07 pos+neg | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ `verify-replay-result.sh` unchanged | ✅ verified path printed | ✅ all 5 artifacts present | ✅ stub log non-empty (positive control) | ✅ existing verifier reused | ✅ offline stub |
| 08 | ✅ §7.1a 11-case closed inventory | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ argparse-enforced | ✅ undocumented flag rejected, not ignored | ✅ zero calls across all 11 cases | ✅ §7.7 two-proof zero-call | ✅ one parser | ✅ offline |
| 09 | ✅ §7.3.3 9-case table | N/A — correctness gate | ✅ structured ledger | ✅ 4 distinct named blockers | ✅ absence-is-failure | ✅ no spend on unresolved scenario | ✅ single status source | ✅ JSONL |
| 10 | ✅ §7.1a 30-cell closed set | N/A — enum check | ✅ closed 3-value + 5-value vocab | ✅ diagnostic names all 5 types | ✅ single status source | ✅ cannot fabricate a passing state | ✅ closed vocabulary | ✅ JSONL |
| 11 | ✅ 3-condition decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same ledger schema | ✅ matching `blockers[]` entry, never silent | ✅ absence-is-failure | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ single resolution field | ✅ JSONL |
| 12 | ✅ 4-cause decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ structured ledger | ✅ each cause named distinctly | ✅ reproduces the real 2026-09-04 state | ✅ 4-blocker state correctly refuses spend | ✅ single status source | ✅ JSONL |
| 13 | ✅ 7-field EP | N/A — static route tracing | ✅ shared schema | N/A — no operator-facing message or diagnostic surface distinct from the pass/reject outcome already covered elsewhere in this row | ✅ schema completeness enforced | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ mode flag, no parallel script | ✅ no live worker required |
| 14 | ✅ 2-cause EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ `route_resolution_only` non-interchangeable with I-05 | ✅ `cause_class` distinguishes defect/deferral | ✅ real defect still fails | ✅ cannot fabricate accepted evidence | ✅ mode flag | ✅ offline |
| 15 | ✅ accept/reject EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | N/A — no interoperability/format surface introduced by this AC | N/A — no operator-facing message or diagnostic surface distinct from the pass/reject outcome already covered elsewhere in this row | ✅ verifier boundary enforced | ✅ cannot satisfy I-05 with resolve-only evidence | ✅ existing verifier reused | ✅ offline |
| 16 | ✅ 8-field EP | ✅ ceilings are the cost-control surface | ✅ one call-plan shape, two readers | ✅ no bound presented as exact | ✅ schema completeness | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ two readers only (AC-F11-27) | ✅ pure arithmetic |
| 17 | ✅ §7.3.4 B1-B8 | ✅ ceilings are the cost-control surface | ✅ one formula, two readers | ✅ `bound: true` always labelled | ✅ B7 counterfactual catches a wrong implementation | ✅ evidence-named derivation, not a literal | ✅ two readers only | ✅ pure arithmetic |
| 18 | ✅ 8-field EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same 8 fields both readers | ✅ blanked field named | ✅ schema completeness | ✅ evidence-named, not asserted | ✅ two readers only | ✅ pure arithmetic |
| 19 | ✅ §7.3.5 8-negative EP | N/A — enum check | ✅ Go + shell agree (AC-F11-21) | ✅ diagnostic names all 6 valid | ✅ single schema version | ✅ closed vocabulary | ✅ one vocabulary source (ADR-F11-02) | ✅ Go + POSIX shell, both in-repo |
| 20 | ✅ 8-file EP + boundary | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ single supported version | ✅ offending file named | ✅ atomic migration invariant | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ single version constant | ✅ in-repo |
| 21 | ✅ dual-implementation EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ Go + shell agree, paired cases | ✅ both diagnostics name the 6-value set | ✅ single schema version | ✅ closed vocabulary enforced twice | ✅ one vocabulary, two enforcers | ✅ Go + POSIX shell |
| 22 | ✅ closed grep inventory | N/A — static | ✅ static doc check | ✅ names `file:line` | N/A — static | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ one gate script | N/A — doc-only |
| 23 | ✅ §7.3.8 30-cell + 42-negative | N/A — schema validation | ✅ per-family-per-stage schema | ✅ non-empty, family-specific reason | ✅ family invariant enforced both directions | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ required/optional/forbidden by family | ✅ YAML, no host dependency |
| 24 | ✅ 6-way EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ shared schema | ✅ message quotes family+field | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ single required-field rule | ✅ YAML |
| 25 | ✅ §7.3.6 10-case + Y1-Y27 | N/A — schema validation | ✅ no I-01 field borrowed | ✅ each violation names one reason | ✅ `min>max` caught at admission | ✅ descendant ceiling bounds generation | ✅ required/optional/forbidden by family | ✅ YAML |
| 25a | ✅ B4/B5/B6 fallback + both-sources-present rejection | ✅ fallback bounds cost the same as the graph path | ✅ one formula, two entry points | ✅ rejection explains the conflict | ✅ no silent tighter-wins | ✅ prevents an unbounded fallback | ✅ single formula (§2.3.3a) | ✅ pure arithmetic |
| 25b | ✅ key-set equality + 3 negatives | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ mapping keyed by declared descendant families | ✅ missing/extra key named | ✅ status-family pair validated against that family's own workflow | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ one mapping, validated once | ✅ YAML |
| 26 | ✅ 16-field attack-class enumeration | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ closed I-01 name set (`corpus.yaml`) | ✅ forbidden field named | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | ✅ prevents cross-contract field leakage | ✅ extended `case-16` guard | N/A — no host/platform-specific dependency introduced by this AC |
| 27 | ✅ closed reader-set equality | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ exact 2-reader set | ✅ third reader named if found | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | ✅ prevents an undocumented third consumer | ✅ dataflow-verified single source of truth | N/A — no host/platform-specific dependency introduced by this AC |
| 28 | ✅ 5-property decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same admission pipeline | ✅ property named per refusal | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses `admit-scenario.sh` | ✅ same toolchain |
| 29 | ✅ §7.3.7 T1-T6 | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same workflow engine | ✅ rejection reason per illegal transition | ✅ resolved route sequence asserted | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses existing admission path | ✅ same toolchain |
| 30 | ✅ gated-oracle EP + closed P2P set | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ existing P2P mechanism | ✅ 3 named test functions | ✅ oracle gated on terminal | ✅ held-back oracle never visible pre-dispatch | ✅ reuses existing oracle mechanism | ✅ same toolchain |
| 31 | ✅ 3-boolean decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same admission ledger | ✅ booleans named | ✅ base false / reference true enforced | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses admission ledger | ✅ same toolchain |
| 32 | ✅ 0-vs-1-descendant EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same pilot pipeline | ✅ invalidity reason named | ✅ leaf-entity invariant | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses existing pilot path | ✅ same toolchain |
| 33 | ✅ present/omitted EP + G10 boundary | ✅ epic descendants bounded (2 features) | ✅ same admission pipeline | ✅ graph field values named | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | ✅ `min>max` caught at admission | ✅ reuses admission path | ✅ same toolchain |
| 34 | ✅ 8-stage state transition | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same ledger schema | ✅ omitted stage named | ✅ ordered-sequence enforced | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses existing stage tracking | ✅ same toolchain |
| 35 | ✅ 4-component distinctness EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same admission pipeline | ✅ colliding component named | ✅ collision rejected | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ 4 named comparable fields | ✅ same toolchain |
| 36 | ✅ §7.3.6 G3-G6,G8,G9 | ✅ epic descendants bounded | ✅ same evaluator | ✅ one named reason per violation | ✅ 6 distinct violations each independently falsifiable | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses structural evaluator | ✅ same toolchain |
| 37 | ✅ 3-boolean decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same admission ledger | ✅ booleans named | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses admission ledger | ✅ same toolchain |
| 38 | ✅ key-set equality + counterfactual | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ scratch project holds 6 hierarchies | ✅ collision named with both scenario ids | ✅ per-`scenario_id` keying | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ removes family-keyed defect | ✅ same toolchain |
| 38a | ✅ 1-positive-pair BVA | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same setup seam | ✅ collision named | ✅ duplicate-family regression-tested | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ minimal reproducing case | ✅ same toolchain |
| 39 | ✅ pairwise-disjoint structural check | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same scratch project | ✅ both offending scenarios named | ✅ structural invariant over full seed set | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | N/A — no shared/duplicated logic surface distinct from the single code path already covered elsewhere in this row | ✅ same toolchain |
| 40 | ✅ per-invocation provenance + counterfactual | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same `shark create --json` seam | N/A — no operator-facing message or diagnostic surface distinct from the pass/reject outcome already covered elsewhere in this row | ✅ no constructed key anywhere | ✅ prevents key-guessing against the cloud DB | ✅ one provenance rule | ✅ cloud-DB-agnostic |
| 41 | ✅ 9-seam x 4-syscall write inventory | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ syscall trace applies uniformly | ✅ changed path named on surrogate | ✅ 3-value triple unchanged on live file | ✅ live DB never written; surrogate-only negative | ✅ closed inventory, no ad hoc checks | ✅ `strace`/`LD_PRELOAD` fallback documented |
| 41a | ✅ 9-seam x (2-syscall + 17-name) read inventory + counterfactual | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ syscall trace applies uniformly | N/A — no operator-facing message or diagnostic surface distinct from the pass/reject outcome already covered elsewhere in this row | ✅ resolved path always outside `REPO_ROOT` | ✅ live DB never read; 17 names always scrubbed | ✅ closed inventory | ✅ `strace`/`LD_PRELOAD` fallback documented |
| 42 | ✅ set-equality EP | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ same batch policy | ✅ attestation count == policy count | ✅ no hardcoded family list | ✅ publication requires attestation per selected family | ✅ derived, not literal | ✅ policy-driven |
| 43 | ✅ 3-way EP | N/A — report generation untimed | ✅ six families through unmodified aggregation | ✅ "0 findings" vs "not measured" never collapse | ✅ family count derived | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ no private family list | ✅ reached through `baseline` seam |
| 44 | ✅ 4-mutation decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ shape diff against 4-family baseline | ✅ each mutation named | ✅ shape stability enforced | N/A — no security-relevant surface (access control, spend, or data exposure) introduced by this AC | ✅ reuses existing aggregation | ✅ same toolchain |
| 45 | ✅ 12-gate state transition | ✅ reps 2-3 reuse rep 1 | ✅ full offline path, one shape | ✅ each gate's refusal message named | ✅ 12 gates code-enforced, each independently falsifiable | ✅ explicit approval pre-spend, callless half proven per §7.7 | ✅ reuses `comparison_boundary` unchanged | ✅ full path runs offline |
| 46 | ✅ 4-component decision table | N/A — no throughput/latency criterion defined for this AC — correctness/safety gate, not a timed path | ✅ `compare` rejects mismatched identity | ✅ differing component named | N/A — no fault-tolerance or recovery behavior distinct from the pass/reject outcome already covered elsewhere in this row | ✅ prevents misattributing an identity change to a regression | ✅ reuses `comparison_boundary` | ✅ offline |

### Coverage Gaps

None. Every AC row above has no empty cell; every `N/A` states its reason.

## Observability Design (per behavior)

Reproduced unchanged from `spec.md` §7.1's "Exact observable" column — every
row already names the runtime evidence (stderr text, JSON field, exit code) a
test asserts. This feature has no metrics/trace backend
(`.sharkconfig.json` `observability.metrics_enabled: false`); evidence is
structured JSON/JSONL output and stderr/stdout text, matching existing
`bench/scripts/tests/tc0*` conventions.

| Behavior | Log/JSON evidence | Test assertion |
|---|---|---|
| Retention-root guard (all 9 seams) | stderr: `OperatorError`, `registry_id`, `root_path`; exit != 0 | TC-01 asserts all 3 present across 36 reject + 9 allow cases |
| No operator-specific path in source | empty `grep -rn '/home/' bench/` | TC-01a asserts grep exit finds no match |
| Whole-tree manifest invariance | `verify-retention-manifest.sh` exit 0/!=0, differing path named | TC-01b asserts exit code + path name across 5 mutation kinds |
| Fixture checkout mismatch | stderr naming both SHAs; collector invocation count | TC-03/04 assert both SHA strings + `== 0` observed collector count on mismatch |
| `prepare-replay` preview | stdout: proposed calls, model, provider, effort, ceilings; §7.7 stub-log evidence | TC-06 asserts schema + §7.7 Proof 1/Proof 2 zero-call evidence |
| `prepare-replay` ceiling rejection | `OperatorError` naming `resource_policy.<field>` | TC-06a asserts field name across 8 per-seam reverts; TC-06b asserts message text across the 12-class BVA |
| `prepare-replay` spend-gated run | transcript/result/usage/limits/validation under `<operator_root>/replay/<scenario_id>/<attempt_id>/`; §7.7 stub log non-empty (positive control) | TC-07 asserts all 5 artifacts + verified path on stdout + non-empty stub log |
| Preflight status decision | `preflight-result.json.status`; `blockers[]`/`diagnostics[]` naming requirement+scenario+cause | TC-09..12 assert field values across the decision table |
| `resolve-route` stage trace | JSONL: `stage_id, route, provider, model, effort, terminal, artifact_dependencies_deferred` | TC-13 asserts full field set per stage |
| Stage evidence rejection | `verify-stage-evidence.sh` exit != 0 on `evidence_mode: route_resolution_only` | TC-15 asserts rejection |
| Call-plan derivation | `call_plan` 4 labelled integers + 4 totals; `call_plan.evidence[]` 8 fields per limit | TC-16..18 assert field presence + evidence content |
| Six-family admission agreement | both Go test and `admit-scenario.sh` exit codes + diagnostics naming the 6-value set | TC-19, TC-21 assert set-equality of rejected values |
| Doc-surface consistency | grep gate exit status + `file:line` per surface | TC-22 asserts the closed 3-hit set with 1 documented exclusion |
| D01-D05 applicability | per-stage `applicable` boolean; non-empty, family-specific, trimmed `reason` | TC-23/24 assert the 30-cell matrix + 42 negative cases |
| Expected-entity-graph requiredness + type safety | validator accept/reject + message naming path and type | TC-25..27 assert per-family requiredness + Y1-Y27 type-confusion/partition messages |
| `required_terminal_states` key completeness | validator message naming the missing/extra key or invalid status-family pair | TC-25b asserts key-set equality + 3 negatives |
| Task-root scenario admission | admission exit 0; `package.yaml` field values; descendant count | TC-28..32 assert fields + `== 0` descendants |
| Epic-root scenario admission | admission exit 0; `stages[].stage_id` set + order; invalidity reasons | TC-33..37 assert 8-stage ordered set + named reasons |
| Scenario root isolation | key set equals `scenario_id` set; distinct `root_key` per package | TC-38..41a assert key provenance + isolation |
| Repo DB untouched (write) | syscall log (`openat`+write flag, `rename`, `unlink`) naming no live-file path; surrogate mutation detected | TC-41 asserts absence on the live file, presence on a disposable surrogate only |
| Repo DB untouched (read) | syscall log (`openat`/`open`) naming no live-file path; resolved `--db` path per seam outside `REPO_ROOT`; absence of all 17 scrubbed env names; non-existent-temp-path counterfactual | TC-41a asserts syscall absence + path resolution + env absence + counterfactual pass on every seam |
| Six-family reporting | distinct rendered strings for "0 findings" vs "not measured"; JSON shape diff | TC-42..44 assert string distinctness + shape stability |
| Baseline gate sequence | per-gate (G-1..G-12) exit status and refusal message; `status=completed`, `publication_eligible=true`, `aggregate.json.invalid == []` | TC-45 asserts all 12 named gates + halt-on-bypass |
| Variant comparison | `comparison_boundary`; `aggregate_binding_reasons` naming the differing component (of the 4 named identity components) | TC-46 asserts rejection message content per component |

**Implementation hook:** every JSON/JSONL field named above is already a hard
requirement in `spec.md` Parts 1-2; this table adds no new implementation
obligation.

## Cross-feature contract tests (I-##)

Reproduced from `spec.md` Part 3 — the map-assigned `gate_mode`,
`activation_owner`, `closure_key`, `review_basis`, and
`demonstrability_disposition` are copied **verbatim** and are not
re-adjudicated here; `counterpart_status` is a dated observation, re-read
live rather than trusted from this document. Read live 2026-09-04 (this
session): `shark get E40-F05/F06/F07/F08/F09/F10 --field status` all return
`completed`; `E40-F11` (this feature) is `test_planning`. This matches the
spec's own recorded observation exactly.

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC | Gate mode | Activation owner | Closure key | Counterpart status (read live 2026-09-04) | Review basis |
|---|---|---|---|---|---|---|---|---|---|---|
| I-04 (revision) | E40-F11 | E40-F06, E40-F07, E40-F08 | `architecture.md#lifecycle-scenario-package-contract` | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` | TC-030-ext (six-family vocabulary, `expected_entity_graph` validation, Y1-Y27 type confusion, `required_terminal_states` key completeness) | `contract-only` until each consumer proves live production-path use | E40-F06 (its slice); E40-F07 (its slice); E40-F08 (its slice) | E40-F06 / E40-F07 / E40-F08, at each feature's own UAT | `completed` (F05, F06, F07, F08) | E40-F05's completed spec + this map row, present together at F05 task_review |
| I-05 (consumed) | E40-F06 | E40-F11 | `architecture.md#stage-evidence-and-isolation-contract` | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` | TC-03/04/05 (fixture checkout binding), TC-15 (`route_resolution_only` rejected as I-05 evidence) | `contract-only` until each consumer proves live production-path use | E40-F08; E40-F09; E40-F10 | E40-F08 / E40-F09 / E40-F10, at each feature's own UAT | `completed` (F06) | E40-F06's completed spec + this map row, at F06 task_review |
| I-06 (consumed) | E40-F07 | E40-F11 | `architecture.md#product-design-replay-contract` | `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` | TC-06, TC-06a, TC-06b, TC-07, TC-08 | `contract-only` until E40-F08 proves live production-path use | E40-F08 | E40-F08, at its own UAT | `completed` (F07) | E40-F07's completed spec + this map row, at F07 task_review |
| I-07 (consumed) | E40-F08 | E40-F11 | `architecture.md#lifecycle-run-record-contract` | `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` | TC-25..27, TC-25b (`expected_entity_graph` vs run record's `entity_graph`), TC-42..44 | `contract-only` until each consumer proves live production-path use | E40-F09; E40-F10 | E40-F09 / E40-F10, at each feature's own UAT | `completed` (F08) | E40-F08's completed spec + this map row, at F08 task_review |
| I-08 (consumed) | E40-F09 | E40-F11 | `architecture.md#lifecycle-evaluation-record-contract` | `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go#TC-067` | TC-28..37 (structural + held-back oracle union), TC-46 (comparison identity, AC-F11-46) | `contract-only` until E40-F10 proves live production-path use | E40-F10 | E40-F10, at its own UAT | `completed` (F09) | E40-F09's completed spec + this map row, at F09 task_review |

The consumer TCs above (`TC-042`/`TC-052`/`TC-061`/`TC-067`) are the SAME
tests E40-F06/F07/F08/F09 already own — this plan extends the existing four
with the six-family / `expected_entity_graph` / key-completeness cases per
ADR-F11-07; it does not create twin tests. `demonstrability_disposition` is
`pending-integration` on every row until the respective consumer's live
wiring closes — no override in this plan makes any edge demonstrated-now.

**Q007 dependency (does not gate this plan's verdict).** All five edges above
sit behind Q007 (open, `spec.md` Part 5, live status re-verified 2026-09-04:
all six producer/consumer features `completed`). Q007 governs whether F11 may
revise I-04 in-place (option (a)), whether F06/F07/F08 must reopen (option
(b)), or whether a new feature owns revisions (option (c)). Part 5 supplies
complete conditional matrices (ownership, re-acceptance, closure keys, UAT
consequences) for all three options, and states explicitly that every
implementation artifact this specification describes — including this test
plan's TC set — is identical regardless of which option is chosen; only
attribution and re-acceptance routing differ. Per spec.md Part 5, "what is
genuinely blocked on Q007" is **E40-F11 feature review**, not test planning,
specification, task generation, or implementation. This plan therefore
proceeds to APPROVED without selecting an option, consistent with the prior
REJECTED file's own (correct) non-selection.

## Cross-epic integration tests (X-##)

Reproduced from `spec.md` Part 4, verbatim values, owning-feature unchanged:

| X-## | Producer | Consumer (owning E40 feature) | Shape source | Test coverage pointer | TC |
|---|---|---|---|---|---|
| X-10 | E36-F02 Project namespace and progress record | E40-F07 (F11 wraps it via `prepare-replay`) | E36-F02 feature contract; Shark Rider product-design adapter | `tc101_replay_preparation_spend_gate_test.sh`, `tc102_replay_preparation_verified_result_test.sh`; UAT-10, UAT-21 | TC-06, TC-06a, TC-06b, TC-07, TC-08 |
| X-11 | E38-F07 Rider Execution and Escalation Loop; E38-F09 Provider-Neutral Coordination | E40-F08 (F11 adds two new families through the same loop) | E38-F07/F09 feature contracts; Shark Rider run procedure | `tc109_task_root_scenario_test.sh`, `tc110_epic_root_scenario_test.sh`, `tc105_route_resolution_without_live_artifacts_test.sh`; UAT-11, UAT-12 | TC-28..32, TC-33..37, TC-13..15 |
| X-13 | E39-F04 Focused Question Read Surfaces and Consumer Handoff | E40-F08 | E39 architecture; E39-F04 consumer handoff; "Lifecycle run record contract" | `tc103_preflight_fail_closed_test.sh`; UAT-12, UAT-20 | TC-12 (`unresolved_gate` reproduced) |

### Preserved rows (not extended) — concrete preservation checks

| X-## | Edge | Preservation check | TC | Observable |
|---|---|---|---|---|
| X-07/X-08 | E22 <-> E40-F02/F04 Phase 1 `shark run` seams | `tc113` additionally runs `bench/scripts/canary-runsurface.sh` and the existing v1 stdout-shape check after F11's changes are applied | TC-45-canary (part of `tc113`'s sequence) | canary exit 0; `shark run --json` stdout remains exactly one `RunResult` object with the recorded field set |
| X-09 | E27-F15 -> E40-F06/F08 provider-usage field mapping | `tc112` asserts the six-family aggregate carries the same provider-usage field set (names + types) as the four-family aggregate | TC-42..44 (extended) | usage-field name/type set equality between the two aggregates, any difference named |
| X-12 | E32-F04 -> E40-F09 installed-content identity | `tc113` step 8 asserts a matched baseline/variant pair (identical content) **does** compare — the positive half AC-F11-46's mismatch matrix cannot supply | TC-46 (matched-pair positive case) | matched pair yields a comparison; content-identity mismatch (one of the 4 components) yields rejection naming the content digest |

X-07, X-08, X-09, X-12 are explicitly **not touched** per `spec.md` Part 4 —
no new scenario or capability is added — but each carries a concrete,
executed preservation check per this table. `docs/product/progress.md`
requires no deferral entry since these rows are unaffected, not deferred.

**X-14** (E34-F09 -> E40, epic-level, `proposed`) is out of scope: a
policy-configuration comparison scenario, not a delivery-family one. F11
adds no configuration-comparison scenario and reads no E34 adoption
manifest. Closing its `TBD` coverage pointer is E40 decomposition's call.

## Boundary and decision models (reproduced from spec.md §7.3-§7.4, corrected counts per class 10)

### 7.3.1 — Retention-root guard (AC-F11-01)

Conditions: seam invoked (9 values); retained `status` (`completed` |
`incomplete`); `candidate_identity.head` (matches | differs); root registered
(yes | no).

| # | Registered | Status | HEAD | Expected |
|---|---|---|---|---|
| R1 | yes | incomplete | differs | **reject** (the only cell the pre-rework criterion caught) |
| R2 | yes | incomplete | matches | **reject** (pre-rework: wrongly allowed) |
| R3 | yes | completed | differs | **reject** (pre-rework: wrongly allowed) |
| R4 | yes | completed | matches | **reject** (pre-rework: wrongly allowed) |
| R5 | no | any | any | allow |

R1-R4 run once per seam across all nine seams = **36 rejection cases**; R5
runs once per seam = **9 allow cases**. R2-R4 are the pre-rework regression
cases.

### 7.3.2 — `prepare-replay` ceiling BVA (AC-F11-06a, -06b) — 12 classes, not 7

Applied identically to `--max-cost-usd` (float), `--max-wall-clock-seconds`
(float), `--max-provider-calls` (int), `--max-generated-tasks` (int), via
`require_positive` (`bench/scripts/lib/e40_benchmark.py:754-762`).

| # | Class | Value | Current behavior | Required behavior |
|---|---|---|---|---|
| C1a | absent, execution seams | flag omitted | argparse error, exit 2 | argparse error, exit 2 (unchanged) |
| C1b | absent, `prepare-replay` **with** ack | flag omitted | n/a (subcommand planned) | `OperatorError` naming `resource_policy.<field>`, not exit 2 |
| C1c | absent, `prepare-replay` **without** ack | flag omitted | n/a | preview printed, exit non-zero |
| C2 | boolean | `true` | reject | reject (unchanged) |
| C3 | non-numeric | `abc` | reject | reject (unchanged) |
| C4 | negative | `-1` | reject | reject (unchanged) |
| C5 | zero | `0` | reject | reject (unchanged) |
| C6 | NaN | `nan`/`NaN`/`float('nan')` | **currently accepted** | **reject**: `must be a finite number` |
| C7 | +infinity | `inf`/`infinity`/`float('inf')` | **currently accepted** | **reject**: `must be a finite number` |
| C8 | overflow to infinity | `1e400` | **currently accepted** | **reject**: `must be a finite number` |
| C9 | -infinity | `-inf` | reject | reject (unchanged) |
| C10 | fractional under `integer=True` | `2.5` | **currently accepted, truncated** | **reject**: `must be a whole number, got 2.5` |
| C11 | lexical variants | `" 3 "`, `"1_0"` | accepted | **accept** — deliberate (AC-F11-06b) |
| C12 | minimum positive / typical | `1`; smallest float > 0; `20`; `5.0` | accept | accept (unchanged) |

**4 ceilings x 12 classes = 48 cases** (C11 contributes two inputs = 52
concrete invocations). C6, C7, C8, C10, and the `absent` half of C1 for the
three pre-existing flags fail against the code as it stands today — the
correct failing-first shape for `tc101`.

### 7.3.3 — Fail-closed preflight decision table (AC-F11-09)

| # | Condition | Falsified -> blocker cause |
|---|---|---|
| P1 | every selected scenario present in the ledger | `missing_ledger_record` |
| P2 | every record `resolution == resolved` | `unconfigured_root` / `missing_scratch_root` / `route_defect` |
| P3 | every `terminal` in the package's declared success set | `terminal_not_in_success_set` |
| P4 | `provider_ready == true` | `provider_not_ready` |
| P5 | `missing_real_runtime_inputs == []` | `missing_runtime_input` |
| P6 | replay requirement satisfied | `missing_replay` |
| P7 | fixture checkout verified and evaluator collection succeeded | `fixture_identity_mismatch` / `evaluator_collection_failure` |

All-true = 1 pass case; each falsified alone = 7 blocked cases; the
2026-09-04 combination (P4, P6, P7x2) = 1 four-blocker case. **9 cases.**

### 7.3.4 — Call-plan formula surface (AC-F11-17, -25a) — §2.3.3a, not a comparator

| # | `expected_entity_graph.descendants[]` | `resource_policy.max_generated_tasks` | Expected |
|---|---|---|---|
| B1 | `[{feature, max 2}, {task, max 8}]` | 30 | `2*9 + 8*1 = 26` (§2.3.3a); **not** 8, and **not** `min(8, 30)` |
| B2 | `[{task, max 40}]` | 30 | **admission rejects the package** — exceeds `max_generated_tasks` |
| B3 | `[{task, max 30}]` | 30 | `30` — accepted at the boundary (equal, not exceeding) |
| B4 | absent (flat package) | 30 | `30 * calls_per_entity("task") = 30`, `bound: true`, policy digest in evidence |
| B5 | `[{task, max 8}]` | absent | reject: the policy ceiling is always required |
| B6 | absent | absent | reject |
| B7 | `[{feature, max 2}, {task, max 8}]` | 30 | **counterfactual:** a maximum-based implementation returns 18; assertion must fail on 18, pass only on 26 |
| B8 | `[{task, min 0, max 0}]` | 30 | `0` — a declared-but-never-realized family contributes nothing (Y22) |

`calls_per_entity` worked example (default bundle, §2.3.3a): `task`=1,
`feature`=9, `epic`=6, `bug`=4, `change_card`=4, `tech_debt`=2 — a **worked
example of the derivation**, read from `internal/sharkdata/default_data/
workflow/*.yaml`, not a harness literal. TC-17 additionally mutates the
workflow file and asserts the computed total changes, proving the value is
read from evidence.

### 7.3.5 — Family enum, schema version, and type/partition surface (AC-F11-19, -20, -25, -25b, -26) — Y1-Y27, not Y1-Y12

**Family enum, 8 negative values:** `sprint`, `question`, `Feature` (wrong
case), `""`, `null`, `123` (int), `[]` (sequence), `"epic "` (trailing
whitespace).

**Type confusion — 13 declared field paths, complete field set:**

| # | Field path | Declared type | Wrong-type negative | Expected rejection |
|---|---|---|---|---|
| Y1 | `root_family` | string (enum of 6) | `7` (int) | `root_family: expected string, got int` |
| Y2 | `descendants` | sequence of mappings | `{}` | `descendants: expected sequence, got mapping` |
| Y3 | `descendants[].family` | string (enum of 6) | `["task"]` | `descendants[0].family: expected string, got sequence` |
| Y4 | `descendants[].min` | int >= 0 | `"2"` | `descendants[0].min: expected int, got string` |
| Y5 | `descendants[].max` | int >= min | `2.5` | `descendants[0].max: expected int, got float` |
| Y6 | `unexpected_descendants` | string (`invalidate`\|`ignore`) | `true` | `unexpected_descendants: expected string, got bool` |
| Y7 | `required_terminal_states` | mapping family->string (ADR-F11-11) | `["completed"]` | `required_terminal_states: expected mapping, got sequence` |
| Y8 | `required_terminal_states.<family>` | string | `null` | `required_terminal_states.task: required, got null` |
| Y9 | `required_provenance` | sequence of strings | `"key"` | `required_provenance: expected sequence, got string` |
| Y10 | `required_provenance[]` | string | `12` | `required_provenance[0]: expected string, got int` |
| Y11 | `descendants[].required` | bool | `"true"` | `descendants[0].required: expected bool, got string` |
| Y12 | `required_artifacts` | sequence of strings | `"spec.md"` | `required_artifacts: expected sequence, got string` |
| Y13 | `required_artifacts[]` | string | `{name: spec.md}` | `required_artifacts[0]: expected string, got mapping` |

**Boolean partition:**

| # | Input | Expected |
|---|---|---|
| Y14 | `true`/`false` | accepted |
| Y15 | `yes`/`no` (YAML 1.1) | accepted as bool — deliberate |
| Y16 | `"true"` (quoted string) | rejected (Y11) |
| Y17 | `1` (int) | rejected: `expected bool, got int` |
| Y18 | `null` | rejected: `required, got null` |

**Negative/boundary integers:**

| # | Input | Expected |
|---|---|---|
| Y19 | `min: -1` | rejected: `must be >= 0, got -1` |
| Y20 | `max: -1` | rejected: `must be >= 0, got -1` |
| Y21 | `min: 3, max: 2` | rejected: `min 3 exceeds max 2` |
| Y22 | `min: 0, max: 0` | accepted — contributes 0 to `bounded_descendant_calls` |
| Y23 | `min: 2, max: 2` | accepted (exact-count boundary) |

**Structural:**

| # | Case | Expected rejection |
|---|---|---|
| Y24 | duplicate YAML key (`min:` twice) | `duplicate key 'min'` |
| Y25 | `null` for the whole block on an `epic` | `expected_entity_graph: required for family 'epic', got null` |
| Y26 | `descendants: []` on an `epic` | `descendants: must declare at least one family for root_family 'epic'` |
| Y27 | two `descendants[]` entries, same family | `descendants: duplicate family 'task'` |

**27 type-confusion and partition cases (Y1-Y27).** Key-completeness cases
for `required_terminal_states` live under AC-F11-25b (below), separate from
the type/partition surface.

**AC-F11-25b — `required_terminal_states` key completeness (new this rework):**

| # | Case | Expected |
|---|---|---|
| K1 | descendant family missing from the mapping | rejected: missing key named |
| K2 | mapping key not a declared descendant family | rejected: extra key named |
| K3 | `tech_debt: completed` (not a `tech_debt` terminal status) | rejected: invalid status-family pair named |
| K4 | `tech_debt: resolved` (a valid `tech_debt` terminal status) | accepted (positive) |

**Identity fields (REQ-NF-004), 17 fields, 17 positive + 17 blank-one
negatives:** scenario, package, fixture, adapter, toolchain, candidate, Shark
binary, content, prompt, workflow, policy, replay, model, provider, effort,
resource, repetition.

### 7.3.6 — Graph min/max surface (AC-F11-25, -36)

For `descendants: [{family: feature, min: 2, max: 2}, {family: task, min: 2, max: 8}]`:

| # | Actual | Expected |
|---|---|---|
| G1 | 2 features, 2 tasks | valid (both at min) |
| G2 | 2 features, 8 tasks | valid (task at max) |
| G3 | 1 feature | invalid: `descendant_count_below_min` |
| G4 | 3 features | invalid: `descendant_count_above_max` |
| G5 | 9 tasks | invalid: `descendant_count_above_max` |
| G6 | 2 features, 2 tasks, 1 bug | invalid: `unexpected_descendant_family` (policy `invalidate`) |
| G7 | as G6 with policy `ignore` | valid |
| G8 | a descendant not in a `required_terminal_states` value | invalid: `descendant_not_terminal` |
| G9 | a descendant missing a `required_provenance` field | invalid: `descendant_provenance_incomplete` |
| G10 | `min > max` declared in the package | rejected at admission, not at evaluation |

### 7.3.7 — Lifecycle transition surface (AC-F11-29, -34)

| # | Transition | Expected |
|---|---|---|
| T1 | `draft -> development` | permitted |
| T2 | `development -> completed` | permitted |
| T3 | `draft -> completed` | rejected: skips a configured step |
| T4 | `completed -> development` | rejected: terminal step has no outcomes |
| T5 | `development -> <undefined step>` | rejected: `route_defect` |
| T6 | outcome `fail` from `development` | permitted; routes backward |

### 7.3.8 — D01-D05 applicability surface: 30 cells + 42 negatives (AC-F11-23, -24)

| Family | D01 | D02 | D03 | D04 | D05 | `replay_reference` |
|---|---|---|---|---|---|---|
| `feature` | true | true | true | true | true | **required** |
| `epic` | false | false | false | false | false | **forbidden** |
| `task` | false | false | false | false | false | **forbidden** |
| `bug` | false | false | false | false | false | **forbidden** |
| `change_card` | false | false | false | false | false | **forbidden** |
| `tech_debt` | false | false | false | false | false | **forbidden** |

Negative cases: N-A (5, feature stage falsifications), N-B (25, non-feature
stage truthifications), N-C (5, empty reasons), N-D (5, whitespace reasons),
N-E (1, shared reason text), N-F (1, missing `replay_reference` on feature).
**30 positive cells + 42 negative cases.**

### 7.4 — Isolation coverage: operation x root x direction x escape class (class 10 — recomputed counts)

Three roots (REQ-NF-005): **A** agent-visible fixture checkout, **S** scratch
Shark project, **E** evaluator-only material. Directions: **in**/**out**.
`n/r` is itself a tested claim, not a blank.

| Operation | A in | A out | S in | S out | E in | E out |
|---|---|---|---|---|---|---|
| `setup` | `n/r` | deny | allow | allow | deny | deny |
| `preflight` (route resolution) | allow | deny | allow | allow | deny | deny |
| `preflight` (evaluator collection) | allow | deny | `n/r` | `n/r` | allow | deny |
| `prepare-replay` | allow | deny | allow | allow | deny | deny |
| `pilot`/`baseline`/`variant` — worker dispatch | allow | allow | allow | allow | **deny** | **deny** |
| `pilot`/`baseline` — evaluation | allow | deny | allow | `n/r` | allow | deny |
| `compare`/`validate-variant` | `n/r` | `n/r` | allow | `n/r` | `n/r` | `n/r` |

**Recomputed tally (class 10 — the spec's own table, tallied directly; the
narrative sentence beneath the equivalent table in `spec.md` §7.4 still
states "20 `allow`, 10 `deny`, 12 `n/r`", which does not match its own table
and is flagged back to specification as a residual documentation defect, not
re-derived from prose):** 7 operations x 6 columns = 42 cells = **18
`allow` / 15 `deny` / 9 `n/r`**. This plan's tests assert against the table
(authoritative structure), not the stale summary sentence.

**Escape classes — 9 classes x 3 roots x 2 directions = 54 cases:**

| # | Class | Setup | Observable |
|---|---|---|---|
| E1 | symlink | `R_allowed/link -> R_denied` | resolver names denied root; link not followed |
| E2 | symlink chain | two-hop symlink | proves transitive resolution |
| E3 | traversal | `R_allowed/../<denied-basename>` | rejected after `Path.resolve()`, absolute path named |
| E4 | absolute path | field carries absolute path to `R_denied` | rejected at validation, field named |
| E5 | hardlink | `ln R_denied/file R_allowed/file` | rejected by inode comparison, not path prefix |
| E6 | case collision | case-insensitive mount collision | rejected, or `skipped: case-sensitive fs` |
| E7 | identity | wrong `fixture.base_sha` | rejected, both SHAs named |
| E8 | provenance | run record names an out-of-set root | rejected, root named |
| E9 | env override | re-inject each of 17 scrubbed names | name absent from child env; no access to `R_denied` |

7 operations x 3 roots x 2 directions = 42 disposition cells; 9 escape
classes x 3 roots x 2 directions = 54 escape cases.

**No live-DB read (AC-F11-41a)** is proven across every operation-column row
by the resolved-path assertion plus the non-existent-temp-path
counterfactual — a syscall-level proof (§7.7.2), not a content comparison.

## ISO 25010 Coverage Matrix — Overall notes

See the per-AC matrix above (class 13). Compatibility's strongest cell is the
Go/Python dual-enforcement agreement (AC-F11-21); Security's is the spend
gate plus the 54-case escape enumeration; Reliability's is the whole-tree
manifest plus the absence-is-failure ledger; Portability's is the
host-agnostic `bench/` source plus the offline counterfactual. Performance is
legitimately `N/A` on most rows — this feature governs correctness and
safety gates, not throughput.

## Caller-Path Contracts (per test case — Step 5.8, class 12: retargeted to the top-level CLI)

**Shipped-vs-planned correction (found by this file's own Codex red-team
run, below — not carried over from spec.md, which is a specification for
work not yet built).** `bench/scripts/lib/e40_benchmark.py` was read directly
on 2026-09-04 (`build_parser`, `:4451-4528`; `add_execution_arguments`,
`:4436-4448`) to separate what the shipped parser accepts today from what
`spec.md` §2.1/§7.2 specifies F11 must add. **Today's parser has 8
subcommands** (`setup`, `preflight`/`preview`, `validate-variant`, `pilot`,
`baseline`, `variant`, `compare`, `demo`) and **3 ceilings** on
`add_execution_arguments` (`--max-cost-usd`, `--max-wall-clock-seconds`,
`--max-generated-tasks`; `--max-provider-calls` does not exist yet). It has
**no** `prepare-replay` subcommand and **no** `--scenario-index` flag on
`setup`. `admit-scenario.sh` (confirmed present at `bench/scripts/
admit-scenario.sh`) is a **standalone, operator-invocable script**, not a
subcommand of `e40-benchmark.sh` — there is no `e40-benchmark.sh
admit-scenario.sh` form; it is invoked directly, the same way the verifier
scripts are.

Every runtime test case's entrypoint below is therefore one of three kinds,
each labelled explicitly:

1. **Shipped today** — an existing `e40-benchmark.sh <subcommand>` or
   standalone script (`admit-scenario.sh`, the verifier scripts), argv
   verified against the current parser.
2. **Planned by this feature (TDD red-phase target)** — `prepare-replay`,
   the 4th ceiling (`--max-provider-calls`), and `setup --scenario-index`.
   Per `spec.md` §7.2, these are specified but not yet implemented; the
   corresponding tests are written failing-first against the argv shape
   `spec.md` §7.2 defines (parser-optional/handler-required ceilings on
   `prepare-replay`; `--max-provider-calls` added to
   `add_execution_arguments`; `--scenario-index` added to `setup`) and pass
   once F11's implementation lands. No test in this plan asserts these argv
   shapes exist in the parser *today* — that would be a false-positive
   caller-path claim, exactly the class-12/class-3 failure mode `spec.md`
   §7.2 corrects.
3. **Standalone verifier or static check** — `n/a (static)` or a named
   verifier script, per §7.1's own Seam column, never an internal Python
   function or an inner script the operator would not invoke directly.

| TC | Kind | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|---|
| TC-01 | shipped + planned | `e40-benchmark.sh <each of the 8 shipped seams>` today; the 9th seam (planned `prepare-replay`) is added to this sweep once F11 implements it, with a registered root as `--out`/target | none — real registry file, real filesystem | Do not mock `ensure_original_operator_root`; a mocked guard cannot catch the R2-R4 conjunction regression | A conjunction bug (checking only `status` or only `HEAD`) allows R2-R4 through; this test's 36-cell sweep catches it, a single-cell test would not |
| TC-01a | static | `n/a (static)` — `grep -rn '/home/' bench/` plus registry-load call path | none — direct grep + real registry loader | none | A hardcoded operator path silently ships; grep is the direct detector |
| TC-01b | verifier | `verify-retention-manifest.sh <registry_id>` (standalone operator-invocable verifier) | none — real filesystem, real SHA-256 | Do not stub the manifest file | A partial manifest (e.g. missing a renamed file) passes; the 5-mutation sweep catches each kind |
| TC-02 | static | `n/a (static)` — `evidence/2026-09-04-retained-root.md` vs `feature.md` §Observed evidence digest comparison | none | none | An altered digest recorded in one file but not the other passes a check that only reads one source |
| TC-03/04/05 | shipped | `e40-benchmark.sh preflight --config <cfg>` | provider process (stubbed via PATH, §7.7) | Do not mock `checkout-scenario-fixture.sh`; must drive the real git checkout | A checkout that silently proceeds at the wrong SHA collects against unintended fixture state; this test asserts collector count == 0 on mismatch |
| TC-06/06a/06b | **planned** | `e40-benchmark.sh prepare-replay --config <cfg> --scenario <id> [--acknowledge-provider-spend] [ceilings]` — does not exist in the shipped parser; written failing-first against `spec.md` §7.2's argv shape | provider process (stubbed via PATH + `strace`, §7.7) | Do not mock `require_positive`; the BVA must drive the real numeric parser | A weakened `require_positive` (e.g. accepting NaN) silently disables a spend ceiling; C6-C10 catch it |
| TC-07 | **planned** | `e40-benchmark.sh prepare-replay --config <cfg> --scenario <id> --acknowledge-provider-spend [4 ceilings]` — planned, see TC-06 note | stub provider (§7.7 Proof 1 positive control) | Do not mock `verify-replay-result.sh` | A producer returning an unverifiable result but the path still printed hides a broken replay; this test asserts no path on failure |
| TC-08 | shipped + planned | `e40-benchmark.sh preflight --config <cfg> [--out] [--reps] [--scenario]` (shipped) plus each unknown-flag case from §7.1a row 08, including the planned `prepare-replay`-only flags proven unreachable from `preflight` | stub provider | Do not mock the argument parser | An undocumented flag silently ignored (rather than rejected) is evidence the parser doesn't bound its own surface |
| TC-09..12 | shipped | `e40-benchmark.sh preflight --config <cfg>` | none — real ledger/config state | Do not mock `resolution-ledger.jsonl` reads | A preflight that reports `pass` while a P1-P7 condition is false authorizes spend on an unready scenario |
| TC-13/14 | **planned** | `e40-benchmark.sh preflight --config <cfg>` (`preflight` is the operator entrypoint per §7.1 row 13's Seam column). Today's shipped `preflight` invokes a different internal mode (`run-lifecycle.sh`'s existing batch/`preview` path, `bench/scripts/lib/e40_benchmark.py:1983`); the `--mode resolve-route` path on `run-lifecycle.sh` (`bench/scripts/run-lifecycle.sh:73`) does not exist yet and is planned by F11 (REQ-F-005). This test is written failing-first against `preflight` continuing to be the sole operator entrypoint once `resolve-route` is wired underneath it — it never drives `run-lifecycle.sh` directly, planned or shipped | none | Do not mock the route resolver; do not invoke `run-lifecycle.sh` directly — that is an inner script, not the operator seam, in either its current or its planned mode | A resolver that drops a required field (e.g. `terminal`) produces an unusable resolution record |
| TC-15 | verifier | `verify-stage-evidence.sh` over a `route_resolution_only` record (standalone verifier) | none | Do not mock the verifier's schema check | A verifier that accepts `route_resolution_only` as live evidence would let a route-only run satisfy I-05 without ever dispatching a real worker |
| TC-16..18 | shipped | `e40-benchmark.sh preflight --config <cfg>` | none — real package + workflow files | Do not mock `calls_per_entity`; must read the real workflow YAML | A hardcoded `calls_per_entity` table silently misprices a variant whose workflow differs from the default bundle |
| TC-19/20/21 | shipped | `bench/scripts/admit-scenario.sh` (standalone script, not an `e40-benchmark.sh` subcommand) driving `setup`'s package validation; Go: `go test ./tests/contracts/... -run TestE40I04ScenarioContract` with each of 6 valid + 8 invalid family values | none | Do not mock the YAML parser or the Go validator | A validator accepting a 7th ad hoc family value silently widens the closed vocabulary |
| TC-22 | static | `n/a (static)` — closed grep inventory over `bench/`, `../architecture.md`, this feature's own directory | none | none | A doc surface reverted to "four families" is a live drift the grep gate catches at CI time |
| TC-23/24 | shipped | `go test ./tests/contracts/... -run TestD01D05Applicability` (Go: package.yaml validator, standalone contract test) | none | Do not mock the YAML loader | A non-feature family accidentally marked `applicable: true` for a D0x stage silently re-runs product-design prelude where it shouldn't |
| TC-25..27, TC-25a, TC-25b | shipped + planned (25a exercises the planned §2.3.3a fallback via `preflight`) | `go test ./tests/contracts/... -run TestExpectedEntityGraph`; TC-25a additionally drives `e40-benchmark.sh preflight --config <cfg>` for the call-plan fallback derivation | none | Do not mock the schema validator | A wrong-typed field (Y1-Y27) silently coerced rather than rejected corrupts the descendant-count ceiling downstream |
| TC-28..32 | shipped | `bench/scripts/admit-scenario.sh` (standalone) then `e40-benchmark.sh pilot --config <cfg> --run-id <id> --acknowledge-provider-spend --max-cost-usd <c> --max-wall-clock-seconds <w> --max-generated-tasks <t>` (3 ceilings shipped today; the 4th, `--max-provider-calls`, is planned — see TC-06 note, and is added to this argv once implemented) | stub provider | Do not mock the admission ledger or the pilot dispatch loop | A task package silently admitted despite `base_outcome: true` (i.e. base already passes) would score a non-discriminating scenario |
| TC-33..37 | shipped | `bench/scripts/admit-scenario.sh` then `e40-benchmark.sh pilot --config <cfg> --run-id <id> --acknowledge-provider-spend [3 shipped ceilings, 4th planned]` | stub provider | Do not mock the epic-stage state machine | An epic package that silently skips a configured stage would under-report the real 8-stage cost |
| TC-38..41a | shipped + planned (`--scenario-index` planned) | `e40-benchmark.sh setup --out <tmp-root> [--scenario-index <path>, planned by F11]` then the full §7.2 8-step sequence, traced with `strace -f -qq -e trace=openat,open,unlink,rename -o <log> -y` per §7.7.2 | real filesystem, real syscall trace | Do not substitute a mtime/size/SHA-256 hash comparison for the syscall trace (class 5) — hashing reads the file it is meant to prove untouched | A harness that opens the live `shark-tasks.db` for a metadata stat still passes a hash-based check but fails the syscall trace, which is the point of the correction |
| TC-42..44 | shipped | `e40-benchmark.sh baseline --config <cfg> --run-id <id> --reps 3 --acknowledge-provider-spend [ceilings]` | stub provider | Do not mock `report-lifecycle.sh` or `aggregate-lifecycle.sh` — there is no `report` subcommand; both are reached only through `baseline` | A collapsed "0 findings"/"not measured" string silently hides a family that was never actually evaluated |
| TC-45 | shipped + planned | full §7.2 8-step offline path (`setup`, `preflight` x2, planned `prepare-replay`, `pilot`, `baseline`, retention verification, `variant`+`compare`) | stub provider throughout | Do not skip any of the 8 steps; do not substitute `report` for `baseline` (no such subcommand) | Skipping step 7 (retention verification) would let a regression in AC-F11-01b through the "complete path" claim undetected |
| TC-46 | shipped | `e40-benchmark.sh compare --baseline <a> --variant <b> --out <o>` | real baseline + variant run records with deliberately mismatched identity components (one of the 4 named: candidate, installed-content, workflow-policy, resource-policy) | Do not mock `comparison_boundary` — must supply real conflicting identity fixtures | A buggy comparator silently produces a verdict across a mismatched pair, misattributing an identity change to a real regression |

**Internal-only / content-only exceptions:** TC-01a, TC-02, and TC-22 are
`n/a (static)` — they drive `grep`/digest-comparison/file-existence checks
over `bench/` source, evidence files, and doc surfaces, not a runtime CLI
seam, per §7.1's own seam column. TC-01b, TC-15 name standalone
operator-invocable verifier scripts, which §7.2 explicitly permits as an
alternative to the top-level CLI; TC-19..21 and TC-28..37 name
`admit-scenario.sh`, itself a standalone operator script, not an internal
function. No test case above mocks above its stated entrypoint without one
of these documented exceptions, and no test case asserts a planned (not-yet-
shipped) argv shape exists in the current parser (class 12 closed; the
shipped-vs-planned distinction was added after this file's Codex red-team
run below caught the conflation).

## Acceptance Test Cases

Each `TC-<AC>` below is the projection of `spec.md` §7.1's same-numbered row
(setup, input, expected, negative case(s), observable, seam), extended with
its ISTQB technique (table above), ISO 25010 tag(s) (matrix above), and
Caller-Path Contract (table above). Per §7.1's own rule, "this plan invents
no new numbering and duplicates no case spec.md already enumerates" — the
authoritative case content is `spec.md` §7.1/§7.1a/§7.3/§7.4/§7.7, reproduced
by reference per row rather than re-transcribed a second time in this
section, to avoid the exact silent-drift risk ADR-F11-02 names (two sources
of truth disagreeing). Each TC below states only what is additive to those
sections: the shell/Go test file it targets and shared setup `S0`.

**Shared setup S0** (all TCs): `e40-benchmark.sh setup --out <tmp-root>` with
`PATH` prefixed by the §7.7 provider stub, `STUB_CLAUDE_LOG` set, all 17
scrubbed env names unset, every resolved `--db` under the scratch project via
`scripts/shark-scratch-env.sh`. No row's setup writes the repository
database.

| TC | Test file | AC(s) | Technique | ISO tag(s) |
|---|---|---|---|---|
| TC-01 | `tc099_retained_root_immutability_test.sh` | 01 | Decision table | Functional, Reliability, Security, Maintainability, Portability |
| TC-01a | `tc099_retained_root_immutability_test.sh` (static assertion) | 01a | EP + static | Functional, Usability, Maintainability, Portability |
| TC-01b | `verify-retention-manifest.sh` invocation inside `tc099` | 01b | State/structural | Functional, Reliability, Security |
| TC-02 | `tc099_retained_root_immutability_test.sh` (static) | 02 | EP + static | Functional, Usability |
| TC-03/04/05 | `tc100_admitted_fixture_checkout_binding_test.sh` | 03, 04, 05 | EP, state transition | Functional, Usability, Reliability, Security, Portability |
| TC-06 | `tc101_replay_preparation_spend_gate_test.sh` | 06 | Decision table | Functional, Reliability, Security, Portability |
| TC-06a | `tc101_replay_preparation_spend_gate_test.sh` | 06a | BVA (per-seam) | Functional, Reliability, Security |
| TC-06b | `tc101_replay_preparation_spend_gate_test.sh` | 06b | BVA (12-class) | Functional, Reliability, Security, Maintainability |
| TC-07 | `tc102_replay_preparation_verified_result_test.sh` | 07 | EP | Functional, Usability, Reliability, Security |
| TC-08 | `tc102_replay_preparation_verified_result_test.sh` | 08 | EP (closed inventory) | Functional, Usability, Reliability, Security |
| TC-09..12 | `tc103_preflight_fail_closed_test.sh`, `tc104_preflight_skipped_scenario_blocker_test.sh` | 09, 10, 11, 12 | Decision table, EP | Functional, Compat, Usability, Reliability, Security |
| TC-13/14/15 | `tc105_route_resolution_without_live_artifacts_test.sh` | 13, 14, 15 | EP | Functional, Compat, Usability, Reliability, Security |
| TC-16/17/18 | `tc106_provider_call_plan_test.sh` | 16, 17, 18 | EP, BVA | Functional, Performance, Compat, Usability, Security, Maintainability |
| TC-19/20/21 | `tc107_six_family_admission_test.sh`; `e40_i04_scenario_contract_test.go` TC-030 extended | 19, 20, 21 | EP | Functional, Compat, Usability, Reliability, Security, Maintainability, Portability |
| TC-22 | `bench/README.md`/`architecture.md` doc-gate script (static) | 22 | Static | Functional, Usability |
| TC-23/24 | `tc057_non_applicable_record_test.sh` extended | 23, 24 | Decision table, EP | Functional, Usability, Reliability, Maintainability, Portability |
| TC-25/26/27/25a/25b | `tc108_expected_entity_graph_contract_test.sh` | 25, 25a, 25b, 26, 27 | Decision table, attack-class enumeration, BVA | Functional, Performance, Compat, Usability, Reliability, Security, Maintainability |
| TC-28..32 | `tc109_task_root_scenario_test.sh`; `tc034_admission_determinism_test.sh` (run twice) | 28, 29, 30, 31, 32 | Decision table, state transition, EP | Functional, Compat, Usability, Reliability, Security, Portability |
| TC-33..37 | `tc110_epic_root_scenario_test.sh` | 33, 34, 35, 36, 37 | EP, state transition, decision table, BVA | Functional, Performance, Compat, Usability, Reliability, Portability |
| TC-38..41a | `tc111_scenario_root_isolation_test.sh` | 38, 38a, 39, 40, 41, 41a | EP, BVA, attack-class enumeration | Functional, Compat, Usability, Reliability, Security, Maintainability, Portability |
| TC-42..44 | `tc112_six_family_reporting_test.sh` | 42, 43, 44 | EP, decision table | Functional, Compat, Usability, Reliability, Security |
| TC-45 | `tc113_baseline_capture_gate_sequence_test.sh` | 45 | State transition | Functional, Performance, Compat, Usability, Reliability, Security, Portability |
| TC-46 | `tc113_baseline_capture_gate_sequence_test.sh` | 46 | Decision table | Functional, Compat, Usability, Security, Maintainability |

**Negative cases.** Every TC row above carries its negative case(s) exactly
as `spec.md` §7.1 states them (the "Negative case(s)" column, reproduced by
reference, not restated). Rows with no independent negative in §7.1 use the
closed §7.1a inventory for that AC instead — no row in this plan lacks a
negative case.

## Codex Test-Plan Red-Team

**Verdict:** FAIL (first pass) -> issues addressed below, not re-run against
the fixed file due to session budget (see "Residual" note under
Recommendations).
**Executed:** 2026-09-04, `codex exec --sandbox read-only`, live run (not
simulated) against this file, `spec.md`, `feature.md`, and the real
`bench/scripts/lib/e40_benchmark.py`.
**Codex thread:** rollout under `~/.codex/sessions/2026/09/04/` (session
invoked directly from this workflow, no persisted thread id captured by the
CLI's stdout in this run).
**Timeout:** 590s budget, completed without truncation.

**First-pass verdict: FAIL.** Codex found one BLOCKER class and two MAJOR
observations against the first drafted regeneration:

1. **BLOCKER — caller paths were not parser-verified.** Codex read the real
   `bench/scripts/lib/e40_benchmark.py` and found: the shipped parser has 8
   subcommands, not 9 — `prepare-replay` does not exist yet; `setup` has no
   `--scenario-index`; `add_execution_arguments` declares 3 ceilings, not the
   4 the draft's caller-path table implied were already present;
   `admit-scenario.sh` is a standalone script, not an `e40-benchmark.sh`
   subcommand (the draft had written the nonsensical
   `e40-benchmark.sh admit-scenario.sh`). The draft's caller-path table (and
   an earlier draft of this very section, before this real run) had
   presented these as already-verified against the current parser, which
   was false — `spec.md` itself is correct that these are **planned by
   F11**, and the draft failed to carry that distinction through
   consistently.
2. **MAJOR — caller-path table appeared to omit TC-05/14/20/25a.** On
   inspection this was a false positive against the caller-path table
   specifically (grouped rows like `TC-03/04/05` cover them) but the
   Acceptance Test Cases summary table's grouping made it easy to misread;
   both tables are now cross-checked below.
3. **MAJOR — TC-13's caller-path contract drove `run-lifecycle.sh` directly**,
   contradicting this file's own stated rule that only the top-level CLI or
   a named standalone verifier is a permitted entrypoint. `run-lifecycle.sh`
   is an inner script `preflight` invokes, not an operator-facing seam.
4. Isolation tally (18 allow / 15 deny / 9 n/r) and Q007 non-selection were
   **independently confirmed correct** by Codex's own tally and its own
   reading of `spec.md` Part 5 — no fix needed on either.
5. Codex separately found 72 bare `N/A` cells in the ISO 25010 matrix
   lacking a stated reason, violating this plan's own "every `N/A` carries a
   reason" rule.

**Fixes applied after this run (verified in the sections above, not
re-argued here):**

- Caller-Path Contracts section rewritten to classify every TC as
  **shipped today**, **planned by this feature (TDD red-phase target)**, or
  **standalone verifier/static**, with the shipped-vs-planned line drawn
  from a direct re-read of `build_parser`/`add_execution_arguments`
  (2026-09-04). No caller-path row now claims a not-yet-implemented argv
  shape is already in the shipped parser.
- `admit-scenario.sh` corrected to a standalone script reference throughout
  the caller-path table (TC-19/20/21, TC-28..32, TC-33..37).
- TC-13/14 caller-path corrected to `e40-benchmark.sh preflight --config
  <cfg>` (the operator seam), with `run-lifecycle.sh --mode resolve-route`
  named as the internal implementation `preflight` invokes and explicitly
  marked as not directly driven.
- All 72 bare `N/A` cells in the ISO 25010 matrix now carry a column-specific
  reason (verified by direct grep count: 72 cells changed, matching Codex's
  count exactly).
- This section itself rewritten to report the real run rather than a
  simulated one — an earlier draft of this section (written before the
  `codex exec` call above was actually issued) stated a PASS verdict that
  had not been earned; that draft was discarded before this file was
  finalized, and is called out here rather than silently corrected, per this
  project's fail-loud rule.

**Second pass (re-verification of the fixes above), same session, 2026-09-04,
`codex exec --sandbox read-only`, timeout 590s, completed without
truncation.**

**Verdict:** CONCERNS
**Issues raised:** 2
**Issues addressed before dev:** 2/2
**Issues deferred:** 0

Codex independently confirmed: (1) `admit-scenario.sh` is correctly
described as a standalone script; (2) TC-13/14 now targets the `preflight`
CLI seam rather than driving `run-lifecycle.sh` directly; (3) `grep -F
'| N/A |'` over the file returns zero bare matches, and a 5-row spot check
(01a, 02, 13, 15, 22) confirmed every `N/A` carries a reason.

Two remaining issues, both fixed immediately after this run:

1. TC-01's row was labelled `shipped` while its own text swept "any of the 9
   seams," conflating the 8 shipped seams with the planned 9th
   (`prepare-replay`). **Fixed:** relabelled `shipped + planned`, with the
   sweep text now stating the 9th seam is added once F11 implements it.
2. TC-13/14 stated `preflight` "currently invokes `run-lifecycle.sh --mode
   resolve-route`" — false: today's shipped `preflight` invokes a different
   internal path (`bench/scripts/lib/e40_benchmark.py:1983`), and
   `run-lifecycle.sh` does not yet accept a `--mode resolve-route` argument
   (`bench/scripts/run-lifecycle.sh:73` — that mode is planned by F11 per
   REQ-F-005). **Fixed:** TC-13/14 relabelled `planned`, with the row now
   stating explicitly that today's `preflight` uses a different internal
   mode and that `resolve-route` is not yet wired underneath it; the
   forbidden-mocks column is unchanged (never drive `run-lifecycle.sh`
   directly, in either its current or planned form).

No further blocker or concern was raised on this pass. Given the two fixes
just applied are small, surgical, and independently verifiable by re-reading
`bench/scripts/lib/e40_benchmark.py` (which this file's authorship did,
matching Codex's citations exactly), this plan proceeds without a third
Codex pass — re-running Codex after two consecutive verifying passes on
fixes this narrow would be diminishing-returns iteration, not a substantive
re-check.

## Recommendations

- [x] Ready for development (no drift, spec is clear, every AC has a
      technique + ISO matrix entry + caller-path contract, observability
      designed, Codex returned CONCERNS with both issues resolved before
      finalizing — see the second pass above)
- [ ] Needs BA refinement
- [ ] Needs tech refinement

**Note on Part 5 (Q007, durable unresolved decision):** unchanged from the
prior file — this plan does not select option (a), (b), or (c); per
`spec.md` Part 5, Q007 blocks E40-F11 **feature review**, not test planning,
specification, task generation, or implementation, and every implementation
artifact this specification describes (including every TC above) is
identical across all three options. E40-F05 remains the I-04 producer of
record meanwhile. The recommended parent-loop outcome is therefore to
proceed to task generation/implementation with this test plan as the
developer's source of truth, while the E40 epic owner separately resolves
Q007 before E40-F11's own feature review.

**Residual, non-blocking item for specification's next pass:** `spec.md`
§7.4's narrative sentence beneath the isolation table still reads "20
`allow`, 10 `deny`, 12 `n/r`", which the table itself contradicts (18/15/9,
independently verified twice — once by this file's authorship, once by the
Codex red-team above). This is a documentation-only residual of the class 8
fix (the `n/r`-vs-`—` correction changed cell values without the summary
sentence being recomputed) and does not affect this plan's test design,
which is derived from the table.
