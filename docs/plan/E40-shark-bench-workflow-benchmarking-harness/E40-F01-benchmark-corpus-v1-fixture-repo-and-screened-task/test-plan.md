# Test Plan: E40-F01 - Benchmark corpus v1: fixture repo and screened tasks

**Created:** 2026-08-05
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/spec.md`
**Task Spec:** 11 task specs exist — `T-E40-F01-001` through `T-E40-F01-011` in
`tasks/` — decomposed from this plan and `spec.md`.
**Status:** APPROVED. The one item this plan originally deferred is **resolved
as of 2026-08-05**: `spec.md`'s Architecture component-changes table now names
both `bench/scripts/diff-ledgers.sh` and `bench/scripts/verify-clean-checkout.sh`
(see "PRD completeness gaps" and "Codex test-plan red-team" below for the
history of that deferral and its resolution).

## Scope and drift analysis

`spec.md` is incremental over `epic.md` (§2 G1/G4/G5/G7, §3 Phase 1 scope, §4
constraints) and `architecture.md`'s "Corpus and oracle contract", which
already pins the I-01 payload. Comparing `spec.md` against those two upstream
documents and against `feature.md`:

- Every `spec.md` REQ-F-* traces to a `feature.md` Scope item or Acceptance
  Criterion, and every `feature.md` AC bullet is covered by at least one
  `spec.md` REQ/AC pair. No scope creep and no scope narrowing found.
- The one item both `feature.md` and `architecture.md` deferred (fixture-repo
  location) is closed in `spec.md` by ADR-F01-01, with an explicit
  materiality argument for why it needs no Question. No open decision remains.
- `spec.md`'s own "Verification plan" table maps every REQ to at least one AC.
  This test plan re-derives that mapping independently below rather than
  copying it, and finds it complete — with two implementation-surface gaps
  noted under "PRD completeness gaps."

No drift found. No BA or architecture refinement is required.

### PRD completeness gaps (Step 4 coverage check)

`spec.md`'s Architecture "Component changes" table names four scripts
(`checkout-fixture.sh`, `build-ledgers.sh`, `admit.sh`, plus the fixture repo
and manifest itself). Two acceptance criteria have no script in that table
that can execute them:

1. **AC-008/AC-009 (REQ-F-011, the "new lint issues only" / regression-vs-removed
   diff method) and AC-010 (REQ-F-010, toolchain fail-closed).** REQ-F-011
   requires the method be *documented* so F02 applies it; it names no
   F01-owned executable that proves the documented arithmetic is correct.
   Without one, AC-008/AC-009 have no test surface: a Go reimplementation
   inside the one allowed contract-test file would prove the Go code agrees
   with itself, not that `bench/README.md`'s prose is correct or unambiguous —
   two independent implementations, neither checking the other. AC-010's
   toolchain guard has the same problem, plus an ownership ambiguity: nothing
   says whether `build-ledgers.sh` or a diffing tool is the one place that
   checks a live environment against a ledger's recorded toolchain before
   trusting any comparison.
   **Resolution required for this test plan to be actionable:** add
   `bench/scripts/diff-ledgers.sh` as a new deliverable with two exact
   invocation shapes:
   - `bench/scripts/diff-ledgers.sh --kind=<lint|test> --base=<base-ledger.json> --post=<post-ledger.json>`
     — implements REQ-F-011's exact semantics (multiset diff over the
     REQ-F-009 identity, floored at zero, for `--kind=lint`; pass→fail as
     regression and present-then-absent as removed, reported separately, for
     `--kind=test`). Takes two ledger-shaped JSON files, no fixture checkout
     needed, so it is testable with hand-authored synthetic data (TC-009,
     TC-010).
   - `bench/scripts/diff-ledgers.sh --toolchain-guard --base=<ledger.json>`
     — reads `<ledger.json>`'s toolchain block (Go version, `golangci-lint`
     version, `GOOS`/`GOARCH`, `.golangci.yml` hash) and the live environment,
     and exits non-zero naming the mismatched field(s) *before* the diff
     modes above run, whenever any field disagrees. **This is the single,
     named owner of the toolchain guard** (TC-011) — `build-ledgers.sh` never
     performs a toolchain *comparison* (it only records the current
     toolchain into the ledger it writes), removing the ambiguity between the
     two scripts.
   This is the same tool F02 can invoke rather than re-deriving the method
   from prose. Task decomposition must create an explicit task for this
   script; it is not optional research left to the developer's judgment.
2. **AC-001 (held-back tests never appear in a fresh checkout).** `spec.md`
   assigns verification to "`bench/scripts/`" generically ("Verified by
   `bench/scripts/`, not by TC-001"), but no listed script performs the
   grep-and-assert. **Resolution:** add
   `bench/scripts/verify-clean-checkout.sh <checkout_dir> <corpus_yaml_path>`
   — reads every held-back F2P file name and test name named across all
   admitted and negative items in `<corpus_yaml_path>`, greps `<checkout_dir>`
   for each, and fails naming the first hit; prints `CLEAN` on stdout on
   success. This is not optionally called: **`checkout-fixture.sh`'s own exit
   path invokes it unconditionally after every checkout it produces**, so
   every current and future caller (`admit.sh`, `build-ledgers.sh`, and later
   E40-F02) gets the invisibility guarantee automatically, not by convention
   or by remembering to call a second script. TC-003 drives
   `verify-clean-checkout.sh` directly and separately confirms
   `checkout-fixture.sh` invokes it.

Both gaps are resolved here with concrete script names, exact argument
shapes, an unambiguous single owner for the toolchain guard, and the test
cases that drive them, so task generation has unambiguous deliverables
rather than unowned notes. This is why the plan below reaches **APPROVED**
rather than **NEEDS_REFINEMENT**: every AC has a test, and every test has a
named, buildable entrypoint invoked the same way a developer or F02 actually
would.

## Test tiers

Verification here spans three execution tiers, because `spec.md` (ADR-F01-05)
deliberately keeps the fixture-repo submodule out of CI:

| Tier | Runs | Needs submodule? | Where |
|---|---|---|---|
| **Tier 1** | `make test` (CI + every dev machine) | No — reads only committed corpus artifacts | `tests/contracts/e40_i01_corpus_contract_test.go` |
| **Tier 1b** | Curator, manually or via a bench self-test wrapper | No — synthetic JSON only, no fixture execution | `bench/scripts/diff-ledgers.sh` invoked with hand-authored fixtures |
| **Tier 2** | Curator, at corpus build time and on every corpus/fixture change | Yes — `git submodule update --init` first | `bench/scripts/{checkout-fixture,build-ledgers,admit,verify-clean-checkout}.sh` against a real checkout |

Tier 2 is what the feature's own Success Metric ("admission gate re-run on a
clean checkout reproduces identical results") exercises. It is **not**
gated by root `make test` — CI's `actions/checkout@v4` does not initialize
submodules (ADR-F01-05) — so it must be the curator's own documented,
re-runnable command sequence, not an implied one. `bench/README.md` must name
the exact invocation for each Tier 2 script so "curator re-runs it" is a real
action, not an assumption.

## Determinism boundary (REQ-F-012)

REQ-F-012 requires "byte-identical verdicts and identity sets," not
byte-identical raw tool output. Three sources of incidental non-determinism
must be normalized away, and the reproducibility tests must prove it across
**independent fresh checkouts**, not by re-reading state left over from a
prior run in the same directory — otherwise TC-006/TC-007 could pass by
accident (e.g. a script that memoizes or re-reads its own prior output)
without the underlying checks having actually re-run:

- `go test -json` emits an `Elapsed` field per event; the test ledger's
  entry identity (REQ-F-008) is `<package>::<test>` + terminal action only —
  `build-ledgers.sh` must drop `Elapsed` before writing `tests.json`, and
  `diff-ledgers.sh`/comparisons must never read it.
- `golangci-lint`'s issue order is not guaranteed stable across runs. The
  lint ledger identity (REQ-F-009) already excludes line/column; the ledger
  writer must additionally emit entries in a fixed sort order (e.g. sorted by
  the identity tuple) so two ledgers over the same input are byte-identical
  files, not just semantically-equal multisets.
- `admit.sh`'s per-candidate verdict lines must be emitted in a fixed order
  (e.g. sorted by item id) so two runs' full stdout is byte-comparable, not
  only set-equal.

TC-006, TC-007, and TC-011 each provision **two separate temporary checkouts**
of the same `base_sha` (via two independent `checkout-fixture.sh` invocations
into two different temp directories) and assert the normalized output is
byte-identical between them — not two runs against one shared checkout.

## AC test matrix

| AC | Requirement(s) | Tier | Technique | Test case | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|---|
| AC-002, AC-013 | REQ-F-002, REQ-F-003, REQ-F-010 | 1 | Contract-surface enumeration | TC-001 | `TestTC001_I01CorpusAndOracleContract` reads, via `os.ReadFile`, the committed `bench/corpus/corpus.yaml`, all `items/*/` and `negative/*/` dirs, and `bench/corpus/ledgers/<base_sha>/*.json` | Every admitted/negative item has the full REQ-F-002 field set, well-typed; named prompt/seed/patch/F2P files exist; `p2p_set` resolves in `p2p_sets`; `schema_version` matches the validator. Cross-encoding agreement: `corpus.yaml`'s `fixture.base_sha` == the `ledgers/<base_sha>/` directory name == both ledgers' toolchain blocks == manifest toolchain pins. Negative: a manifest item with a `p2p_set` absent from `p2p_sets`, or a field of the wrong type, fails with that field named. |
| AC-011 (package visibility half) | REQ-NF-001, REQ-NF-002 | 1 | Boundary/state enumeration (submodule populated vs. not) | TC-002 | Repo root, `go list ./...` (via `os/exec`), run twice: once with the submodule uninitialized (gitlink only, matching CI), once after `git submodule update --init` | Neither run lists a fixture-module package or a `testdata/f2p` path. Negative: a hypothetical fixture `go.mod` removal (simulated by a temp copy) would make the fixture visible to `./...` — TC-002 documents this as the invariant it protects, by asserting the fixture's own `go.mod` is present at the pinned submodule path. |
| AC-001 | REQ-F-001, REQ-F-004 | 2 | Attack-class enumeration (leak surface) | TC-003 | `bench/scripts/checkout-fixture.sh <base_sha> <dest_dir>`, then `bench/scripts/verify-clean-checkout.sh <dest_dir> bench/corpus/corpus.yaml` | `verify-clean-checkout.sh` reports `CLEAN` (exit 0) when grepping the checkout for any held-back name from `corpus.yaml`. Also assert `checkout-fixture.sh` itself invokes `verify-clean-checkout.sh` (not merely that the script exists) by observing the checkout script's own exit path. Negative: a deliberately-seeded copy of one held-back file into `<dest_dir>` after checkout is detected and named on a direct re-invocation of `verify-clean-checkout.sh`. |
| AC-003 | REQ-F-006 | 2 | Decision table (5 checks x candidate class) | TC-004 | `bench/scripts/admit.sh bench/corpus/corpus.yaml` (no `--item`, meaning "all admitted candidates") | ≥10 admitted items, all five checks (F2P-red-at-base, P2P-green-at-base, patch-applies, F2P-green-post-patch, P2P-green-post-patch) pass for each, one JSON verdict line per item on stdout sorted by item id; ≥1 `bug`-type item's manifest names a repro-test set. Negative: fewer than 10 admitted items or zero bug items with repro names fails the gate's own summary assertion. |
| AC-004 | REQ-F-005, REQ-F-007 | 2 | Decision table (one candidate per rejection branch, all five branches) | TC-005 | `bench/scripts/admit.sh bench/corpus/corpus.yaml --item <id>` against: (a)/(b)/(d) the three **committed** negative candidates (F2P-already-green-at-base; P2P-already-red-at-base; patch-leaves-F2P-red); (c)/(e) two **transient, test-generated** candidates built at test time from a copy of an admitted item — (c) its `reference.patch` truncated to a malformed diff so `git apply` itself fails; (e) its `reference.patch` plus an additional one-line test-time mutation that flips one P2P test from pass to fail post-patch | Each of the five is rejected with exactly its own check named in `admit.sh`'s verdict output ((a) F2P-green-at-base, (b) P2P-red-at-base, (c) patch-apply failure surfaced from `git apply`'s exit status, (d) F2P-still-red-post-patch, (e) P2P-red-post-patch); none of the three committed negatives appears in the admitted-item list any run driver would consume. REQ-F-007 excuses (c)/(e) from needing a *committed* fixture, not from needing test *coverage* — the transient candidates satisfy that without adding permanent corpus content. Negative: a negative or transient candidate accidentally passing all five checks (misconfigured fixture) is a plan-level red flag — TC-005 fails loudly rather than silently admitting it. |
| AC-005 | REQ-F-005, REQ-F-012 | 2 | State transition (re-run stability, independent checkouts) | TC-006 | `bench/scripts/admit.sh bench/corpus/corpus.yaml` over the full candidate set (admitted + negative + the two transient TC-005 candidates), invoked twice against **two separately provisioned** `checkout-fixture.sh` temp checkouts of the same `base_sha` | Byte-identical verdict output (sorted, normalized per "Determinism boundary") both runs. Negative: an intentionally nondeterministic candidate (e.g., a flaky reference patch) would show as a plan gap, not something TC-006 papers over by loosening the assertion; a script that memoizes/re-reads a prior verdict instead of re-running the checks would be caught because the two checkouts are provisioned independently, not shared. |
| AC-006 | REQ-F-008, REQ-F-012 | 2 | Equivalence partitioning (test / subtest / skip) | TC-007 | `bench/scripts/build-ledgers.sh <checkout_dir> <output_dir>` invoked twice, once per independently provisioned checkout at `base_sha`; the fixture repo's own test suite is required to contain at least one plain test, one table-driven subtest, and one deliberately skipped test (`t.Skip(...)`) so all three terminal actions are exercised | One entry per `<package>::<test>` including subtests as distinct slash-named entries; entries cover all three terminal actions (`pass`/`fail`/`skip`); the two runs produce a byte-identical entry set under the dropped-`Elapsed`, fixed-sort normalization. Negative: a subtest collapsed into its parent's identity, or a skipped test omitted entirely, is caught because TC-007 asserts the exact entry count and the presence of a `skip`-action entry. |
| AC-007 | REQ-F-009 | 2 | Boundary value analysis (line and column position shift) | TC-008 | Fixture edited two ways, each re-linted and diffed against the base ledger via `diff-ledgers.sh --kind=lint --base=... --post=...`: (i) blank lines inserted directly above an existing issue (line shift only), (ii) leading whitespace added before an existing issue's column (column shift only, line unchanged) | Both edits report zero new issues — identity excludes both line and column, so the shifted-but-otherwise-identical issue matches its base-ledger entry in each case independently. Negative: if either line or column were included in identity (the bug this AC guards against), each case would falsely report one new issue. |
| AC-008 | REQ-F-011 | 1b | Boundary value analysis (issue count 0/1/N, duplicate identity, net-negative) | TC-009 | Synthetic base lint ledger (hand-authored JSON, shaped like real `build-ledgers.sh` output, 3 issues, one identity appearing twice to exercise multiset counting) fed to `diff-ledgers.sh --kind=lint` against five synthetic post-run ledgers: (a) identical to base — 0 new; (b) base + 1 genuinely new issue — 1 new; (c) base + 3 genuinely new issues — 3 new; (d) base with every existing issue's line shifted +2 and 1 new issue added — exactly 1 new (not 4); (e) base with 2 of its 3 issues removed and 0 added — 0 new, floored at zero rather than negative | Each of the five synthetic cases reports the exact expected new-issue count, and the duplicated-identity entry in the base ledger is preserved as depth 2 (not collapsed to 1) when computing the multiset difference. Negative: a naive line-inclusive diff would report 4 "new" issues in case (d) (3 shifted + 1 genuine); a set-based (not multiset-based) diff would silently collapse the duplicate-identity entry, understating a genuinely duplicated issue. |
| AC-009 | REQ-F-011 | 1b | State transition (every base/post terminal-action pair) | TC-010 | Synthetic base test ledger `{A: pass, B: pass, C: pass, D: fail, E: skip}` (hand-authored JSON, shaped like real ledger output) fed to `diff-ledgers.sh --kind=test` against a synthetic post-run ledger `{A: pass, B: fail, D: fail, E: fail}` (`C` absent — removed; `D` and `E` still/newly `fail` but were not `pass` at base) | `B` is reported as a regression (only `pass`→`fail` counts); `C` is reported separately as removed, never as a regression and never silently dropped; `D` (`fail`→`fail`) and `E` (`skip`→`fail`) produce **no** regression entry, because REQ-F-011 defines regression strictly as base-`pass` becoming post-`fail`; `A` produces no entry. Negative: a diff that folds "removed" into "regression," drops removal silently, or counts a base-`fail`/`skip` entry ending in `fail` as a regression fails this TC by construction — all three failure modes are distinct, asserted output classes. |
| AC-010 | REQ-F-010 | 2 | Attack-class enumeration (toolchain drift, one named owner) | TC-011 | `bench/scripts/diff-ledgers.sh --toolchain-guard --base=<ledger.json>` — the sole named toolchain-guard entrypoint (see PRD completeness gap 1) — invoked once per mismatch axis: a stub `golangci-lint` binary on `PATH` reporting a version other than the ledger's recorded `v2.9.0`; a stub `go` binary reporting a different Go version than `go.mod`'s `go 1.25.0`; an environment with `GOOS`/`GOARCH` overridden from the ledger's recorded values; a `.golangci.yml` mutated by one byte from the recorded content hash | Each of the four mismatch axes fails independently, naming the specific recorded-vs-observed mismatch; no diff mode runs afterward. Negative: `build-ledgers.sh` itself must never perform this comparison (it only records the current toolchain into the ledger it writes) — TC-011 asserts the guard exists only in `diff-ledgers.sh`, closing the ownership ambiguity between the two scripts. |
| AC-011 (full statement) | REQ-F-001, REQ-F-004, REQ-NF-001, REQ-NF-002 | 1 + 2 | Boundary/state enumeration (two submodule states) | TC-012 | Repo root, `make fmt && make lint && make test`, run once with the submodule uninitialized (as CI checks it out) and once after `git submodule update --init` | Both runs are green; `go list ./...` (cross-referenced with TC-002) lists no fixture or held-back-test package in either state; `scripts/lint-positional-selection.sh`'s `internal`/`cmd`-only grep is unaffected by `bench/`'s existence. |
| AC-012 | REQ-NF-005 | 2 | Attack-class enumeration (offline + missing pinned tool) | TC-013 | `bench/scripts/admit.sh bench/corpus/corpus.yaml` (all admitted+negative candidates) run with Go module cache warm and, for network isolation, `unshare --net` on Linux CI (true network-namespace block) or, where `unshare` is unavailable (e.g. a macOS dev machine), `GOPROXY=off GOFLAGS=-mod=readonly` plus `http_proxy=https_proxy=http://127.0.0.1:9` (a guaranteed-closed local port so any outbound attempt fails fast rather than hanging) — first with `golangci-lint v2.9.0` present at the pinned version, second with the pinned binary absent from `PATH` | First case: the full gate over the whole corpus completes with no network access. Negative: with the binary absent and network isolated, the gate fails loudly naming the missing pinned tool — it must **not** fall back to `Makefile:108`'s `curl`-based auto-install, which would silently reintroduce a network dependency and version drift against REQ-F-010; TC-013 asserts no outbound connection attempt occurs in either case (via the namespace block or the poisoned-proxy sentinel). |

## Acceptance-criteria review

Every AC is unambiguous, testable, traceable to a `spec.md` REQ, and specifies
an exact expected output (a named failing check, an exact new-issue count, a
distinct removed-vs-regressed classification) rather than "works correctly."
No AC is an open-ended robustness assertion — REQ-F-012's "byte-identical" is
the closest candidate, and it is closed by the "Determinism boundary" section
above rather than left as an assumption. Every runtime AC has at least one
explicit negative case in the matrix above.

## ISTQB technique application

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-002, AC-013 | Contract-surface enumeration | TC-001 | Manifest is an interaction surface (I-01) with another feature; every field, cross-reference, and encoding must be enumerated, not sampled. |
| AC-011 (both) | Boundary/state enumeration | TC-002, TC-012 | Submodule-populated vs. gitlink-only are the two real states CI and a dev machine can be in; both are boundaries, not one representative case. |
| AC-001 | Attack-class enumeration | TC-003 | "Never appears in the repo the agent sees" is exactly the class of defensive property Attack-class enumeration exists for — enumerate leak surfaces (grep hit), don't just assert absence in the happy path. |
| AC-003, AC-004 | Decision table | TC-004, TC-005 | Admission is five boolean checks combining into accept/reject; a decision table forces every combination, not just the all-pass and all-fail rows. TC-005 covers all five rejection branches — three via committed negative candidates (a/b/d), two via transient test-generated candidates (c/e) since REQ-F-007 excuses those from a committed fixture but not from test coverage. |
| AC-005, AC-006 | State transition (re-run stability) | TC-006, TC-007 | "Reproducible" is a claim about two executions of the same state, which is what state-transition technique is built to enumerate (same input, same output, twice) — proven across two independently provisioned checkouts, not one reused checkout, so a memoizing implementation can't pass by accident. TC-007 additionally applies Equivalence Partitioning across the test/subtest/skip classes so all three terminal actions are exercised, not only pass/fail. |
| AC-007, AC-008 | Boundary value analysis | TC-008, TC-009 | Line **and column** position, issue count (0/1/N), duplicate identity, and net-negative-floored-at-zero are all ordered/boundary domains; BVA forces each independently rather than a single arbitrary shifted-line example. |
| AC-009 | State transition (every base/post terminal-action pair) | TC-010 | A test's lifecycle across two ledger snapshots has five reachable transitions worth distinguishing (pass→pass, pass→fail, pass→absent, fail→fail, skip→fail); all five must be asserted, not just the regression case, so a base-`fail`/`skip` entry ending in `fail` is never miscounted as a regression. |
| AC-010, AC-012 | Attack-class enumeration | TC-011, TC-013 | Toolchain drift and "missing pinned tool, no network fallback" are both defensive properties against silent corruption — the class to enumerate is every way the check could be fooled into producing a false diff or a false pass. TC-011 enumerates all four toolchain axes (Go version, linter version, `GOOS`/`GOARCH`, config hash) against one named guard entrypoint; TC-013 enumerates both a true network-namespace block and a portable proxy-poison fallback. |

## Caller-Path Contracts

This feature has deterministic runtime behavior (bash scripts executing real
git/go/golangci-lint against real or synthetic files), so `content-only`
opt-outs do not apply here except where noted. For the manifest-shape test
(TC-001), the real, current entrypoint is the Go contract test itself doing
real file reads and real YAML/JSON parsing against the committed corpus —
there is no separate F02 reader to name yet, so TC-001's own parsing code
*is* the entrypoint, in the same sense a content-only test names its direct
file read (per the internal-only convention). What makes this more than a
content-only case is that TC-001 is the producer-side half of I-01's shared
contract test: `spec.md`'s Cross-feature interactions section requires
E40-F02 to reuse this same test rather than write a second reader, so the
field list and encoding it asserts today become F02's contract tomorrow. The
Go test function name follows `tests/contracts/e39_interactions_test.go`'s
`TestTC001_...` convention precisely so it is unambiguously the `#TC-001`
anchor that `spec.md`'s Cross-feature interactions section and
`E40-interaction-map.md`'s I-01 row both pin as
`tests/contracts/e40_i01_corpus_contract_test.go#TC-001` — E40-F02 must be
able to find this exact function by that pointer, not by searching the file
for "something that looks like the manifest test."

| TC | Entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `TestTC001_I01CorpusAndOracleContract` (the contract test function itself) calling `os.ReadFile` + real YAML/JSON unmarshal against `bench/corpus/corpus.yaml`, `bench/corpus/items/*/`, `bench/corpus/ledgers/<base_sha>/*.json` | Real filesystem read of the committed YAML/JSON, real parser | Do not substitute an in-memory struct for `corpus.yaml`; must parse the real committed file | A validator that reads a hand-built in-memory manifest would stay green even if the real committed YAML were malformed |
| TC-002 | `exec.Command("go", "list", "./...")` run from repo root inside the Go test | Real `go` toolchain subprocess | Do not hardcode the expected package list from memory; derive it from the live `go list` output at test time | A hardcoded expected-package list would stay green even after a future change accidentally added a fixture package to the module graph |
| TC-003 | `bench/scripts/checkout-fixture.sh <base_sha> <dest_dir>` then `bench/scripts/verify-clean-checkout.sh <dest_dir> bench/corpus/corpus.yaml` | Real filesystem grep over a real `checkout-fixture.sh` output tree, real `corpus.yaml` as the name source | Do not run against a pre-filtered file list or a hardcoded held-back-name list; must derive names from `corpus.yaml` and scan the actual checkout directory | An implementation that filters the held-back name list before scanning would hide a genuine leak the same way a bug that never populates the checkout correctly would |
| TC-004 | `bench/scripts/admit.sh bench/corpus/corpus.yaml` (no `--item`, full admitted set) | Real `git apply`, real `go test`, real `golangci-lint` in a temp checkout at `base_sha` | Do not stub any of the five checks' subprocess results; do not hardcode a boolean per item | An admission gate that hardcodes "pass" for missing test binaries would admit items that were never actually screened |
| TC-005 | `bench/scripts/admit.sh bench/corpus/corpus.yaml --item <id>`, once per one of the three committed negative ids and once per each of the two test-time-generated transient candidates (built by copying an admitted item's `reference.patch` and corrupting/mutating it as described in the AC test matrix) | Same as TC-004 | Same as TC-004; additionally do not special-case negative or transient items to force a specific rejection message | A gate that always reports the first of the five checks as the failure would pass TC-005 by coincidence unless each candidate's *specific* named check is asserted |
| TC-006 | `bench/scripts/admit.sh bench/corpus/corpus.yaml`, invoked once against each of two independently provisioned `checkout-fixture.sh` temp checkouts | Same as TC-004, run twice from a clean state each time | Do not memoize or cache verdicts between the two invocations under test — each call must independently execute the real checks against its own fresh checkout | A cached-verdict implementation would trivially "reproduce" without the checks having actually re-run, hiding real nondeterminism |
| TC-007 | `bench/scripts/build-ledgers.sh <checkout_dir> <output_dir>`, invoked once against each of two independently provisioned checkouts | Real `go test -json`, real `golangci-lint` subprocess in a temp checkout | Do not stub `go test`/`golangci-lint` output; do not read a previously-written ledger instead of regenerating | A ledger builder that reads back its own prior output instead of re-running tests would always "reproduce," defeating the reproducibility claim |
| TC-008 | `bench/scripts/build-ledgers.sh` (lint half) against a real edited fixture checkout, then `bench/scripts/diff-ledgers.sh --kind=lint --base=... --post=...` | Real `golangci-lint` subprocess | Do not stub the linter; must observe real line/column movement from a real source edit | A lint-identity implementation that silently included line or column would report a false "new issue" here, which this TC exists to catch |
| TC-009 | `bench/scripts/diff-ledgers.sh --kind=lint --base=<synthetic.json> --post=<synthetic.json>` (new script) | None — pure JSON-in, JSON-out, no subprocess needed | Do not special-case the synthetic fixture's shape in the script under test | An implementation naively diffing by list position (not by identity/multiset) would misreport shifted-but-unchanged issues as new, or collapse a genuinely duplicated issue |
| TC-010 | `bench/scripts/diff-ledgers.sh --kind=test --base=<synthetic.json> --post=<synthetic.json>` (new script) | Same as TC-009 | Do not collapse "removed" and "regression" into one output field; do not treat any base-`fail`/`skip` entry as eligible for "regression" | An implementation that only tracks any-transition-to-`fail` would misreport a pre-existing failure as a new regression and would silently drop the removed test |
| TC-011 | `bench/scripts/diff-ledgers.sh --toolchain-guard --base=<ledger.json>` invoked with a real stub `go`/`golangci-lint` executable on `PATH` reporting a wrong version, and with `GOOS`/`GOARCH`/`.golangci.yml` altered on disk | Real subprocess invocation of whatever binary `PATH` resolves to; real environment variables; real file content hash | Do not mock the version string in Go/config; the stub binary must be a real executable reporting a real, wrong version string; do not implement or accept this guard inside `build-ledgers.sh` | A version check reading a hardcoded constant instead of invoking the real binary would never catch an actually-mismatched environment; a guard split across two scripts could be silently bypassed by calling the one that lacks it |
| TC-012 | Repo root `make fmt && make lint && make test` | Real `go`, `gofmt`, `golangci-lint` subprocesses via the Makefile | Do not run against a filtered subset of packages | A green result from a hand-picked package subset would hide a fixture package accidentally entering `./...` |
| TC-013 | `bench/scripts/admit.sh bench/corpus/corpus.yaml` under `unshare --net` (Linux) or `GOPROXY=off GOFLAGS=-mod=readonly http_proxy=https_proxy=http://127.0.0.1:9` (portable fallback) | Real subprocess execution with genuine network isolation, not a code-level flag | Do not simulate "offline" by stubbing network calls in code — the environment itself must be offline; do not let a missing pinned binary fall back to `Makefile:108`'s `curl` installer | An implementation that only checks a config flag for "offline mode" instead of actually running without network access would pass a fake test while still depending on the `curl` auto-install fallback in production |

## ISO 25010 coverage matrix

`N/A` cells are justified per `uat-plan.md`'s own framing: "Not a product
concern here — the harness is offline tooling," which this feature's Non-Functional
Requirements (REQ-NF-003/004/005) and the epic's own stated non-functional
posture both confirm.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-003 | N/A | N/A | N/A | N/A | ✅ TC-003 (leak surface) | N/A | N/A |
| AC-002/013 | ✅ TC-001 | N/A | ✅ TC-001 (I-01 shape) | N/A | N/A | N/A | ✅ TC-001 (schema version gate) | N/A |
| AC-003 | ✅ TC-004 | N/A | N/A | N/A | ✅ TC-004 | N/A | N/A | N/A |
| AC-004 | ✅ TC-005 | N/A | N/A | ✅ TC-005 (named failing check) | ✅ TC-005 | N/A | N/A | N/A |
| AC-005 | ✅ TC-006 | N/A | N/A | N/A | ✅ TC-006 | N/A | N/A | N/A |
| AC-006 | ✅ TC-007 | N/A | N/A | N/A | ✅ TC-007 | N/A | N/A | N/A |
| AC-007 | ✅ TC-008 | N/A | N/A | N/A | ✅ TC-008 | N/A | ✅ TC-008 (identity stable) | N/A |
| AC-008 | ✅ TC-009 | N/A | N/A | N/A | ✅ TC-009 | N/A | N/A | N/A |
| AC-009 | ✅ TC-010 | N/A | N/A | N/A | ✅ TC-010 | N/A | N/A | N/A |
| AC-010 | ✅ TC-011 | N/A | ✅ TC-011 (toolchain pin) | ✅ TC-011 (mismatch named) | ✅ TC-011 | N/A | N/A | ✅ TC-011 (GOOS/GOARCH) |
| AC-011 | ✅ TC-002, TC-012 | N/A | ✅ TC-012 (submodule states) | N/A | N/A | N/A | ✅ TC-012 (lint clean) | N/A |
| AC-012 | ✅ TC-013 | N/A | N/A | N/A | ✅ TC-013 | ✅ TC-013 (no silent network fallback) | N/A | N/A |

No coverage gaps: every non-`N/A` cell has a citing TC; every `N/A` cell is
justified by this feature's offline-tooling, no-user-journey nature
(REQ-NF-003/004/005; `uat-plan.md` Non-functional evidence).

## Observability design

This is offline curator/CI tooling with no production request path, so there
are no metrics or trace spans to design (consistent with `uat-plan.md`'s
"Not a product concern here"). Observability here means **the scripts'
own machine-readable output**, which is what a human curator or a future F02
depends on to know what happened without re-deriving it:

| Behavior | Log / stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Clean-checkout verification | `verify-clean-checkout.sh` prints `CLEAN` on success or the first offending held-back name on failure; invoked automatically by `checkout-fixture.sh`'s own exit path | N/A — internal, no runtime request path | TC-003 asserts both the direct invocation's output and that `checkout-fixture.sh` calls it unconditionally |
| Admission verdict per candidate | `admit.sh` prints one JSON line/record per candidate, sorted by item id, naming pass/fail per check, and the exact failing check on rejection | N/A | TC-004, TC-005, TC-006 assert on this output, not just on process exit code |
| Ledger generation | `build-ledgers.sh` prints the toolchain block it recorded (Go version, `golangci-lint` version, `GOOS`/`GOARCH`, `.golangci.yml` hash) into the ledger; it performs no toolchain *comparison* itself | N/A | TC-007 asserts the printed/written block; TC-011 asserts this script never accepts a `--toolchain-guard`-equivalent flag |
| Diff result | `diff-ledgers.sh --kind=lint\|test` prints counts of new/removed/regressed entries, plus the entries themselves, in sorted order | N/A | TC-008, TC-009, TC-010 assert both the printed summary and the full entry list |
| Toolchain mismatch | `diff-ledgers.sh --toolchain-guard` — the single named owner — fails with the specific mismatched axis (Go version, linter version, `GOOS`/`GOARCH`, or config hash; recorded vs. observed) named on stderr, before any diff mode runs | N/A | TC-011 |
| Offline/missing-tool failure | Fails naming the missing pinned tool, not a generic error; asserted alongside no observed outbound connection attempt | N/A | TC-013 |

No new instrumentation beyond structured script output is required or
permitted — this is deliberately not a metrics/tracing surface.

## Cross-feature contract tests (I-01)

Carried verbatim from `spec.md`'s "Cross-feature interactions" section — this
plan does not restate or re-derive the shape, only the test that proves it:

| I-## | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-01 | E40-F01 | E40-F02 | `architecture.md#corpus-and-oracle-contract` | `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` | TC-001 |

Gate mode: `live` (per `E40-interaction-map.md`; no `contract-only`/staged row
is declared for I-01, so no `review_basis` value is fabricated here). Closure
owner: E40-F01 code-review owner. Required UAT evidence: UAT-02 (a negative
candidate is rejected with the failing check named and appears in no run
list; the gate reproduces on a clean checkout) — TC-005 and TC-006 are this
evidence. UAT-01 and UAT-07 reach this contract transitively through E40-F02
and are not this feature's evidence to produce. When E40-F02 is decomposed,
it must reuse TC-001 verbatim rather than writing a second reader — no twin
test.

## Cross-epic integration tests (X-##)

`spec.md`'s "Cross-epic integrations" section and `E40-cross-epic-map.md`
both state that F01 produces, consumes, and validates **no X-## row** (X-07
belongs to E40-F02, X-08 to E40-F04, X-09 is Phase 2 with no owning feature).
This plan does not invent an X-## row, a contract pointer, or a deferral for
F01 where none is assigned.

## Integration scenarios

| Scenario | Boundary | Epic UAT contribution | Test evidence |
|---|---|---|---|
| Corpus → future harness (I-01) | `bench/corpus/*` file shape → E40-F02's not-yet-built reader | UAT-02 (transitively UAT-01, UAT-07 via F02) | TC-001 pins the shape F02 must consume with no manual step |
| Admission gate → negative-item rejection | `admit.sh` → the three committed negative candidates | UAT-02 | TC-005, TC-006 |
| Base-SHA ledgers → future post-run diff (F02) | `build-ledgers.sh` / `diff-ledgers.sh` output → F02's post-run collector | G4, G7 (transitively, via F02 and F03's replay) | TC-007, TC-008, TC-009, TC-010, TC-011 |
| Fixture and corpus → shark's own quality gate | `bench/` tree → root `make fmt && make lint && make test` and `go list ./...` | Non-functional: repo hygiene, not a UAT scenario | TC-002, TC-012 |
| Curator re-run → Success Metric | `admit.sh` and ledger scripts re-run on a clean checkout | `feature.md` Success Metric | TC-006, TC-007, TC-013 |

Two verification-plan rows are intentionally **not** test cases, matching
`spec.md`'s own Verification Plan table:

- **REQ-NF-003** (never touches the live repo/DB/`.sharkconfig.json`, never
  invokes shark init) — verified by code review of `bench/scripts/*.sh`
  confirming no such invocation exists, not by an automated test, matching
  `spec.md`'s own "Diff review" verification method.
- **REQ-NF-004** (no credentials/PII/third-party code in the corpus) —
  verified by human review of corpus content at admission time, matching
  `spec.md`'s own "Corpus review at admission" verification method.

## Test infrastructure

**Existing patterns to reuse:**
- `tests/contracts/e38_f12_parallel_team_topology_test.go` and
  `tests/contracts/e39_interactions_test.go` establish the repository-root-relative
  artifact-reading helper style (`filepath.Abs(filepath.Join("..", ".."))`,
  then `os.ReadFile`) — TC-001/TC-002 follow this, reading real committed
  files, not embedded or mocked ones.
- Root `Makefile:86-121` (`fmt`/`vet`/`lint`/`test` targets) is the pattern the
  fixture repo's own `Makefile` mirrors (REQ-F-001); root `.golangci.yml`
  (pinned `v2.9.0`) is the pattern the fixture's `.golangci.yml` mirrors.
- `scripts/lint-positional-selection.sh`'s `internal`/`cmd`-only grep scope is
  the existing precedent that `bench/` sits outside shark's own lint reach
  (REQ-NF-002) — TC-012 asserts this stays true, it does not change the script.

**New test infrastructure needed (this feature's own deliverables):**
- `tests/contracts/e40_i01_corpus_contract_test.go` — one Go file, `package
  contracts`, containing TC-001 and TC-002. Per ADR-F01-05, this file **goes
  red until the corpus artifacts it reads exist** — task decomposition must
  sequence its creation with or after the corpus content tasks, never ahead of
  them.
- `bench/scripts/diff-ledgers.sh` (new — see PRD completeness gap 1) —
  `--kind=lint|test --base=<f> --post=<f>` and `--toolchain-guard --base=<f>`
  modes; drives TC-008, TC-009, TC-010, TC-011.
- `bench/scripts/verify-clean-checkout.sh <checkout_dir> <corpus_yaml_path>`
  (new — see PRD completeness gap 2) — drives TC-003; `checkout-fixture.sh`
  must call it unconditionally on every successful checkout.
- The fixture repo's own test suite (owned by this feature, per ADR-F01-01)
  must deliberately include one table-driven subtest and one `t.Skip(...)`
  test alongside its ordinary tests, purely so AC-006/TC-007 has a real
  `skip`-terminal-action entry to assert against — this is a fixture-content
  requirement, not only a test-design one, and belongs in the same task that
  builds the fixture's test suite.
- Synthetic ledger fixtures for TC-009/TC-010 — small, **committed**
  hand-authored JSON files shaped like real `build-ledgers.sh` output (same
  field names and types), not generated inline by the test, so the tool is
  validated against a realistic shape rather than a shape the test itself
  invented. These belong under `bench/corpus/` or a `bench/scripts/testdata/`
  sibling, kept out of the shark module's own `./...` the same way held-back
  F2P sources are.
- `bench/README.md` must name the exact Tier 2 curator command sequence (at
  minimum: `git submodule update --init`, then `admit.sh`, `build-ledgers.sh`,
  `diff-ledgers.sh --toolchain-guard`, `verify-clean-checkout.sh` — noting the
  last is also invoked automatically by `checkout-fixture.sh`) so the Success
  Metric's "re-run on a clean checkout" has one documented invocation, not an
  implied one.

## Codex test-plan red-team

**Verdict:** CONCERNS (second pass) — 5 of 8 findings closed by revision, 3
deferred to the spec owner with an explicit trigger (see below)
**Issues raised:** 8 (first pass)
**Issues addressed before development:** 5 (findings 1-5)
**Issues deferred:** 3 (findings 6-8 — spec-ownership hand-off, not a test-design gap; owner and trigger below)

The dispatch prompt names no `codex_command`. The local read-only Codex CLI
(`codex exec -m gpt-5.6-terra`, high reasoning effort) reviewed the drafted
plan together with `spec.md` and was asked directly about the
AC-008/AC-009/AC-001 verification-owner question. First-pass verdict:
**CONCERNS**, with these findings (condensed; full run against the
pre-revision draft):

1. **AC-005/AC-006 (TC-006/TC-007):** the draft compared two runs *in the
   same checkout* rather than proving reproducibility across independent
   fresh checkouts, and didn't specify canonical/sorted output.
2. **AC-003/AC-004 (TC-004/TC-005):** the five-check decision table only
   exercised branches (a)/(b)/(d) via committed negatives; branches (c)
   (patch-apply failure) and (e) (post-patch P2P failure) had no test
   coverage at all, even though REQ-F-007 only excuses them from a
   *committed* fixture, not from being tested.
3. **AC-006 (TC-007):** "test/subtest/skip" was asserted without the fixture
   actually containing a skipped test to prove the `skip` terminal action is
   captured.
4. **AC-007/AC-008 (TC-008/TC-009):** BVA was overstated — TC-008 shifted
   lines only, never columns; TC-009 tested exactly one added issue, with no
   zero-new, multiple-new, duplicate-identity/multiset, or net-negative case.
5. **AC-009 (TC-010):** only the pass→fail and pass→absent transitions were
   covered; base-`fail`/`skip` entries ending in `fail` were never asserted
   as *not* regressions.
6. **AC-010 (TC-011):** Go-version mismatch wasn't tested (only lint
   version/OS/arch/config), and the toolchain-guard owner was ambiguous
   ("`build-ledgers.sh` or `diff-ledgers.sh`").
7. **Caller-Path Contracts:** TC-001 named a not-yet-real future F02 reader
   instead of its own real, current entrypoint; TC-004–TC-008 and TC-013
   lacked exact script argument shapes ("`--all` (or equivalent)" is not a
   contract); TC-013's offline mechanism ("DNS/proxy blocked") wasn't a
   concrete, portable technique.
8. **AC-001 (TC-003)/observability:** `verify-clean-checkout.sh` lacked an
   explicit name-source argument, and "admit.sh **may** call it" left the
   production invocation unowned rather than mandatory.

Codex separately confirmed the two-new-scripts resolution itself was the
right ownership call (do not defer REQ-F-011/AC-001 to F02; do not fold
ledger diffing into `build-ledgers.sh`), conditional on the component table
and task decomposition actually adopting both scripts with named argument
shapes — which this revision now states explicitly.

**All eight findings' test-design content is addressed in the plan above**;
findings 6-8 additionally surfaced a spec-adoption gap that is a documented
deferral, not a test-design gap (see the second-pass result and the
Owner/Trigger below). The test-design fixes: independent-checkout comparisons
and canonical sort order (Determinism boundary, TC-006/TC-007); transient
test-time-generated candidates for branches (c)/(e) (TC-005); a required
fixture-content skip test (TC-007, Test infrastructure); column-shift plus the
full 0/1/N/duplicate/net-negative boundary set (TC-008/TC-009); the full
five-transition partition including fail→fail and skip→fail as
non-regressions (TC-010); a Go-version mismatch axis and a single named
toolchain-guard owner, `diff-ledgers.sh --toolchain-guard` (TC-011, PRD
completeness gap 1); exact argument shapes for every script entrypoint and
TC-001 reframed around its own real file-read/parse code (Caller-Path
Contracts); a concrete portable offline mechanism, `unshare --net` with a
proxy-poison fallback (TC-013); and a mandatory (not optional)
`verify-clean-checkout.sh` invocation from `checkout-fixture.sh`'s own exit
path, plus a corresponding observability row (TC-003, PRD completeness gap
2).

A second read-only pass against this revised plan returned **CONCERNS**,
naming findings 6, 7, and 8 as only *conditionally* closed: the pass agreed
the revised test design for all eight findings is credible, but pointed out
that `diff-ledgers.sh --toolchain-guard` and `verify-clean-checkout.sh` exist
today only as this test plan's recommendation — `spec.md`'s own Architecture
component-changes table still lists only `checkout-fixture.sh`,
`build-ledgers.sh`, and `admit.sh`, so those two scripts and their argument
shapes are not yet an authorized part of the feature's ground-truth spec.
Codex's literal suggestion was "update the spec... then this is PASS."

This test plan does **not** make that edit itself. `spec.md` line 87 already
pins the contract-test pointer and its Verification-plan table maps every
REQ to an AC; adding component rows would mean touching adjacent requirement
and verification text to stay coherent with those two — that is spec
authorship, not gap-flagging, and belongs to the same role that authored
`spec.md`, not to this test plan. Per Step 8's own rule, codex returning
**CONCERNS** (not **FAIL**) routes to "revise now or document why deferred
with explicit owner/timeframe," not to `NEEDS_REFINEMENT` — and nothing about
either script is an open design question: their argument shapes, output
format, invocation site, and owning test case are all pinned above. This is
recorded as a **named, owned deferral**, not a silent gap:

- **Owner:** the E40-F01 spec/architecture owner (the role that authored
  `spec.md`).
- **Trigger:** before task decomposition creates the `diff-ledgers.sh` or
  `verify-clean-checkout.sh` tasks — it must land ahead of, not after, those
  tasks are dispatched.
- **Exact ask:** add two rows to `spec.md`'s Architecture component-changes
  table for `bench/scripts/diff-ledgers.sh` and
  `bench/scripts/verify-clean-checkout.sh`, and note in REQ-F-011's and
  AC-001's text which script owns each (mirroring the "PRD completeness gaps"
  section above verbatim).
- **Interim status:** until that lands, this test plan's "PRD completeness
  gaps" section is the authoritative source for both scripts' contracts —
  task decomposition may proceed from it, but the spec edit should not be
  skipped indefinitely.
- **Resolved (2026-08-05):** the exact ask above has landed. `spec.md`'s
  Architecture component-changes table now names both
  `bench/scripts/diff-ledgers.sh` and `bench/scripts/verify-clean-checkout.sh`
  (see `spec.md` lines 83–86), and task decomposition proceeded from that
  authorized spec, not only from this plan's interim recommendation. This
  history is kept for context; it is no longer an open deferral.

## Recommendations

- [x] Ready for development — every AC in `spec.md` has a named test case,
  technique, ISO 25010 row, caller-path contract, and (where applicable)
  observability evidence. The two implementation-surface gaps found in Step 4
  (`diff-ledgers.sh`, `verify-clean-checkout.sh`) are resolved with concrete
  script names, argument shapes, and owning test cases — task decomposition
  must create explicit tasks for both; they are required deliverables of this
  feature, not optional polish.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
- **Hand-off resolved (2026-08-05):** the E40-F01 spec/architecture owner
  added both scripts to `spec.md`'s Architecture component-changes table
  (`spec.md` lines 83–86), so the test plan's recommendation is now the
  feature's ground truth, not a parallel source. Task decomposition (11 tasks,
  `T-E40-F01-001` through `T-E40-F01-011`) proceeded from the resolved spec;
  this bullet is kept as a record of the former hand-off, not a live blocker.
