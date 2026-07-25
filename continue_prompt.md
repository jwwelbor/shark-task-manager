---
created: 2026-07-24 16:46:51 CDT
branch: feat/shark-plan-command
plan_file: /home/jwwel/.claude/plans/i-think-it-s-fine-shimmering-hamster.md
shark_task: local task #14 "Discuss remaining work on feat/shark-plan-command with user"
status: planning — design complete, ONE open decision pending user answer
---

# Resume: cascade-to-fork for `shark next`

## What we're building

Teach `shark next <key> --json` to **cascade through single-option tiers as today, but
STOP and return a fork (the candidate tier) when a tier has 2+ dispatchable children** —
instead of silently picking the first child. The rider then decides fan-out-subset vs
follow-one; following one = `shark next <child-id>`. This is what unlocks parallel dispatch:
today the rider is never shown a set, so it can never fan out.

Read the plan file first — it has the full context, confirmed findings, and decisions:
`/home/jwwel/.claude/plans/i-think-it-s-fine-shimmering-hamster.md`

## Decisions already locked (with user)

1. **Default fan-out.** Escape hatches to force legacy single-track: a `.sharkconfig.json`
   bool `sequential_dispatch` (default false) AND a `--sequential` flag on `shark next`.
   Flag > config > default.
2. **Full edges** on each fork candidate: `depends_on` + `blocks` + `links`.
3. Reuse the existing `HierarchyPlanSelectionResponse` envelope for the fork (no new shape).

## THE ONE OPEN DECISION (ask user first, before any code)

In default fan-out mode, how should **single-option tiers** resolve?
- **(A) Plan engine throughout** (Plan agent's rec): fan-out drills via `PlanHierarchyService`
  the whole way. Simpler, one engine. Single-option is *shape-preserving but NOT byte-identical*
  to legacy — diverges only in the rare unclaimed-in-progress-out-of-execution-order case
  (legacy resumes the in-progress task; fan-out drills the order-1 todo). `--sequential` gives
  exact legacy behavior.
- **(B) Legacy cascade for single-option**: reuse exact `CascadeService` resolution for
  single-option tiers, diverge only at a real fork. Most faithful to user's "cascade as long
  as one option," byte-identical single-option even by default, but interleaves two engines
  (more complex, higher regression risk).

The user interrupted right as this question was posed. **Get this answer, then finalize the
plan's implementation section around it.** (A) keeps the design as written below; (B) requires
`tryCascadeFanout` to call the legacy single-drill path for the `len(selected)==1` branch.

## Implementation design (from Plan agent — assumes decision A; adjust if B)

Architecture: fan-out cascade = a drill-through wrapper around plan's one-level selection.
Per tier: `PlanHierarchyService.DescribeChildren` → `selectPlanChildTier` →
`len(selected)`: 0 → auto-advance/pause (unchanged semantics); 1 → drill (recurse
`resolveNext` into the single child); ≥2 → fork (`HierarchyPlanSelectionResponse`).
Tie-tiering means `execution_order 1,2,3` does NOT fork (only top tie-tier is a candidate set).
`tryCascade` + `CascadeService` (sequential path) left byte-for-byte untouched.

**Phase 1 — config + flag plumbing**
- `internal/config/config.go`: add `SequentialDispatch bool` (json `sequential_dispatch,omitempty`)
  beside `MaxParallelItems`; add nil-safe getter `GetSequentialDispatch()`.
- `internal/config/manager.go`: parse `sequential_dispatch` bool in `Load` (~line 101). No
  config-validation allowlist exists, so no allowlist edit needed.
- `internal/cli/commands/next.go`: register `--sequential` bool flag on `nextCmd`; resolve
  mode in `runNext` (flag>config>default); add `nextGetSequentialDispatch` indirection hook.
- `nextAdapterCache`: add ONLY `fanout bool` (NOT maxParallelItems — would shadow the embedded
  `planAdapterCache.maxParallelItems`). Set `adapters.fanout = !sequential`.

**Phase 2 — fork-detection core (`next.go`)**
- `resolveNext` cascade branch (~line 406): `if cache.fanout { return tryCascadeFanout(...) }
  return tryCascade(...)`.
- New `tryCascadeFanout` mirrors plan.go `tryPlanHierarchy`, differs in single-child branch:
  drill via `resolveNext(depth+1)`, prepend parent to trail (handle both `childResp.selection`
  and `childResp.ResolvedVia`). Fork branch: load edges, `buildHierarchyPlanSelection(...)`,
  enrich candidates, set `selection.ResolvedVia`, `resp.selection = &selection`.
  `E02→F03→3 tasks` ⇒ fork at task tier, `root_key=F03`, `resolved_via=[E02,F03]`.
- New `autoAdvanceCascadeParentFanout` = copy of `autoAdvancePlanCascadeParent` but final
  recursion targets `resolveNext` (fan-out) not `resolvePlanEntity`.

**Phase 3 — candidate edges (`plan.go` + service)**
- `HierarchyPlanCandidate`: add `DependsOn/Blocks/Links []CandidateEdge` (all omitempty).
  New wire type `CandidateEdge{Key, Status, Type}`. Leave `buildHierarchyPlanSelection`
  signature unchanged (plan output stays edge-less via omitempty).
- New service method (layering-correct, CLI never touches repo): e.g.
  `PlanHierarchyService.DescribeChildEdges(ctx, entityType, keys) (map[string]PlanHierarchyEdges, error)`.
  REUSE existing semantics — do NOT invent SQL: `EntityRelationshipService.GetTaskBlockedBy/
  GetTaskBlocks/GetOutgoing/GetIncoming/GetTaskRelationships` and
  `task/dependency.go GetTaskDependents/GetTaskDependencies`. Must be entity-type-polymorphic
  (feature-tier forks need feature edges; `readEpicFeatures` currently hardcodes Dependencies=[]).
- Wire via a `cli.Get…Service()` accessor + `package commands` indirection hook (mirror
  `planDescribeDispatchableChildren`) for test injection.

**Phase 4 — output wiring (`next.go`)**
- `runNext` (~line 295): `if resp.selection != nil { return outputHierarchyPlanSelectionJSON(*resp.selection) }`
  BEFORE the normal `outputNextJSON`. Set span attrs like plan's `outputPlanResult`. Add
  `ResolvedVia []string json:"resolved_via,omitempty"` to `HierarchyPlanSelectionResponse`
  (plan never sets it → plan golden unaffected). Marker values: `Mode="hierarchy_selection"`,
  `Action="parallel_candidates"`, `SelectionReason="parallel_tie"`.

**Phase 5 — rider skill (`skills/shark-rider/verbs/run.md` only)**
- Add fork branch: detect `mode=hierarchy_selection`/`action=parallel_candidates`/`entities[]`/
  no prompt. REUSE `verbs/plan.md`'s existing map-validation / evidence-gap /
  `parallel_execution=available` logic (X-## `docs/product/cross-epic-integration-map.md`,
  I-## epic-local interaction map, task-dependency evidence) to pick the safe subset. Dispatch
  each chosen child with bare `shark next <child-id> --json`; follow-one = same on one child.
- NO embedded rider copy under `internal/sharkdata/default_data/` (memory about embedded
  canonical does NOT apply to rider skill). Contract test forbids `/shark` slash-syntax only,
  not bare `shark` CLI strings.

**Deferred (do NOT bundle here):** `CascadeService.dependenciesSatisfied` hardcodes terminal
set (`"completed"/"archived"`) — violates no-hardcoded-statuses rule. Fixing it changes the
sequential path (must stay byte-identical to 0e3f0103). Separate follow-up; fan-out path
already avoids the bug via config-driven `PlanHierarchyService`.

## Tests (see plan file for full list)
Sequential byte-identical (golden); default single-drill vs --sequential identical dispatch
JSON; 2+ → fork; multi-level drill-then-fork resolved_via; execution_order 1,2,3 does NOT
fork; edges populated (task + feature tier); config/flag precedence matrix; auto-advance
preserved; claim/dep-filtered children excluded; plan golden unchanged; `DescribeChildEdges`
real-DB test. Gate: `make fmt && make lint && make test`.

## Critical files
- internal/cli/commands/next.go — cascade fork branch, flag, output wiring
- internal/cli/commands/plan.go — HierarchyPlanCandidate edges, CandidateEdge, reused helpers
- internal/services/plan_hierarchy_service.go — DescribeChildEdges + entity-polymorphic edges
- internal/config/config.go (+ internal/config/manager.go) — sequential_dispatch
- skills/shark-rider/verbs/run.md — fork branch reusing plan-verb validation

## How to optimize the execution path (keep main thread lean)

1. **Answer the open decision with the user FIRST** (main thread, no agents).
2. **Do NOT re-run the three Explore agents or the Plan agent** — their findings are captured
   in the plan file and above. Re-read the plan file instead of re-exploring.
3. Implement in dependency order with focused subagents so the main thread stays out of large
   file bodies:
   - Agent 1 (Go, sequential): Phase 1 config+flag plumbing (config.go, manager.go, next.go
     flag/cache) — small, self-contained. Verify with a `go build`.
   - Agent 2 (Go): Phase 3 edges — `CandidateEdge`, `HierarchyPlanCandidate` fields, and the
     `DescribeChildEdges` service method reusing existing relationship methods. Independent of
     Phase 2 core; can run in PARALLEL with Agent 1.
   - Then main thread wires Phase 2 (`tryCascadeFanout`) + Phase 4 output — this is the delicate
     part touching the restored contract; keep it on the main thread or a single careful agent,
     NOT parallel, and diff against `--sequential` golden.
   - Agent 3 (docs/skill): Phase 5 rider `run.md` fork branch — independent, run in parallel
     once the wire shape (marker values) is fixed.
   - Agent 4 (tests): author the test list once Phases 1–4 compile.
4. Run `make fmt && make lint && make test` on the main thread as the final gate; fix failures
   before declaring done. Use writable Go caches if the sandbox needs it.
5. Commit logically on `feat/shark-plan-command` (already the branch). Do NOT touch
   shark-tasks.db. Wait for gemini-code-assist auto-review before merging any PR.

## Guardrails (from project memory / rules)
- Any new command capability must support ALL entity types (edge loader is entity-polymorphic).
- No hardcoded status names (use workflow terminal-status APIs).
- CLI commands are thin wrappers → business logic in services, no direct repo calls from commands.
- Edit embedded `internal/sharkdata/default_data/` for bundle content — but rider skill has NO
  embedded copy, so Phase 5 edits `skills/shark-rider/verbs/run.md` directly.
