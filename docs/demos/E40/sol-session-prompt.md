# Sol session prompt: make E40 baseline and prompt-variant comparisons easy

You are implementing the operator experience for the complete E40 Shark Bench
epic in `/home/jwwel/projects/shark-task-manager`.

This is important benchmark infrastructure. Treat reproducibility, identity,
evidence provenance, spend safety, and truthful comparison as first-class
requirements. Do not stop after writing documentation or proving that the
existing unit tests pass.

## Start here

Read these files before changing anything:

1. `AGENTS.md`, `CLAUDE.md`, and the applicable `.claude/rules/` files.
2. `bench/README.md`.
3. `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`.
4. The E40 feature specifications and test plans for F01 through F10.
5. `docs/demos/E40/demo-script.md`.
6. The current implementations and usage contracts of the scripts under
   `bench/scripts/`, especially:
   - `run-batch.sh`, `run-one.sh`, `aggregate-runs.sh`,
     `report-baseline.sh`, and `replay-manifest.sh`;
   - `admit-scenario.sh`, `run-lifecycle.sh`, `run-lifecycle-batch.sh`,
     `evaluate-lifecycle.sh`, and `run-prelude.sh`;
   - `pilot-ledger.sh`, `aggregate-lifecycle.sh`, `report-lifecycle.sh`,
     `run-review-comparison.sh`, and the verification scripts;
   - `bench/scripts/tests/run-all.sh` and every registered E40 test relevant to
     the changed behavior.

Re-check live Shark state with read-only commands. Do not change E40 status,
claims, approvals, or unrelated dirty files.

## Objective

Make this workflow easy to run repeatedly:

```text
prepare and validate corpus
  -> establish immutable baseline
  -> change prompts, workflow bundle, model, or policy
  -> run the same scenarios under the variant
  -> compare baseline and variant on speed, quality, tokens, cost, LOC,
     review findings, artifact use, and lifecycle stage time
  -> report better, worse, unchanged, indeterminate, or invalid per metric
```

The primary user should not need to hand-assemble scratch Shark projects,
root entities, batch policies, candidate files, retention layouts, or report
commands for the normal local demonstration path.

## Required deliverable

Add a thin, well-tested operator layer over the existing F01–F10 scripts. Do
not duplicate their domain logic or create a second evaluator, workflow
engine, identity scheme, retention format, or comparison algorithm.

At minimum, provide:

### 1. A safe top-level command

Create a script such as:

```bash
bench/scripts/e40-benchmark.sh baseline ...
bench/scripts/e40-benchmark.sh variant --name prompt-v2 ...
bench/scripts/e40-benchmark.sh compare --baseline <id> --variant <id>
bench/scripts/e40-benchmark.sh demo ...
```

Use the repository's existing naming conventions if a better command name is
already established. The command must support:

- a no-spend `preflight` or `preview` mode;
- explicit output roots and run IDs;
- baseline creation;
- variant creation from a changed prompt/workflow/model/policy bundle;
- paired reruns against the same admitted scenarios and fixture identities;
- aggregation and comparison reports;
- focused execution of one scenario or one repetition for debugging;
- resume/retry behavior without overwriting retained evidence;
- clear exit codes and machine-readable result metadata.

Default behavior must make no provider calls. Provider-backed execution must
require an explicit acknowledgement and strictly positive cost, wall-clock,
and generated-task limits, using the existing spend-gate contract.

### 2. Reusable local setup

Add a safe setup path for the demo that creates or prepares isolated scratch
Shark projects and seeded root entities for the four admitted lifecycle
families. It must:

- use temporary or explicitly named operator roots;
- never touch the live repository database;
- never delete `shark-tasks.db` or use destructive cleanup against broad paths;
- avoid credentials, network endpoints, or provider assumptions;
- record the exact Shark binary, content bundle, workflow bundle, fixture,
  adapter, scenario, and prompt identities;
- produce a batch-policy or candidate file consumable by the existing F10
  drivers;
- make the generated inputs inspectable before any provider call.

If a scratch project cannot be safely auto-created for a real provider run,
provide a deterministic offline/demo fixture and make the missing real-runtime
inputs explicit rather than faking a successful provider run.

### 3. Baseline and variant identity

Every baseline and variant must have a durable manifest containing, at minimum:

- benchmark/run ID and creation time;
- corpus/scenario package identity and versions;
- fixture base SHA and adapter/toolchain identity;
- Shark binary identity;
- installed Shark-data/content bundle identity;
- rendered prompt or prompt-bundle digest;
- provider/model/effort identity;
- workflow-policy and enabled-gate identity;
- resource policy and repetition count;
- source-control candidate identity where code is under comparison;
- output/retention root and exact command arguments.

Do not treat a branch name, current `HEAD`, model label, or prompt filename as
the complete identity. Reuse the existing F02/F06/F08/F09/F10 identity
contracts and fail closed on missing or mixed identity.

### 4. Fair comparison report

Add a report that compares a baseline and a variant only when their comparison
boundary is valid. It must show separate dimensions, not one hidden score:

- final correctness / held-back execution-oracle result;
- scenario pass rate and invalid/incomplete counts;
- lifecycle wall time and stage-category time;
- provider-active, tool/test, wait, retry, and unclassified time;
- input/output tokens and provider cost;
- generated LOC and changed paths where applicable;
- gate time, emitted/unique/duplicate/recurrent/confirmed/unconfirmed findings;
- artifact production, downstream consumption, reuse, and orphan counts;
- replayed interaction proxies, clearly not human minutes;
- per-scenario and aggregate deltas;
- published noise-band result: better, worse, no detectable effect, or
  indeterminate;
- exact reason for every invalid or excluded comparison.

Quality must dominate interpretation: a faster or cheaper variant that fails
the held-back oracle is not better. Do not collapse quality, time, and cost
into a composite score. When no seeded truth set exists, report confirmed
yield/overlap/recurrence/downstream escape without inventing precision or
recall.

### 5. Prompt-change workflow

Make prompt changes explicit and reproducible. Support either a copied
versioned bundle or a controlled override path, but record the resulting
content digest and ensure the variant cannot silently use the baseline prompt.

The normal workflow should look like:

```bash
# No-spend validation
bench/scripts/e40-benchmark.sh preflight --config <demo-config.yaml>

# Establish baseline
bench/scripts/e40-benchmark.sh baseline \
  --config <demo-config.yaml> \
  --run-id baseline-<name> \
  --reps <n> \
  --acknowledge-provider-spend \
  --max-cost-usd <limit> \
  --max-wall-clock-seconds <limit> \
  --max-generated-tasks <limit>

# Change a prompt/workflow bundle, then verify its digest
bench/scripts/e40-benchmark.sh validate-variant \
  --config <demo-config.yaml> \
  --variant-name prompt-v2 \
  --prompt-root <changed-prompt-root>

# Run the same matrix with the variant
bench/scripts/e40-benchmark.sh variant \
  --config <demo-config.yaml> \
  --variant-name prompt-v2 \
  --run-id prompt-v2-<name> \
  --reps <n> \
  --acknowledge-provider-spend \
  --max-cost-usd <limit> \
  --max-wall-clock-seconds <limit> \
  --max-generated-tasks <limit>

# Compare and render the report
bench/scripts/e40-benchmark.sh compare \
  --baseline <baseline-retention-root-or-id> \
  --variant <variant-retention-root-or-id> \
  --out <comparison-report-root>
```

The final command must point to retained raw evidence and produce both a
machine-readable comparison and a concise human-readable report.

## Testing requirements

Add contract and integration tests for:

1. preflight makes zero provider calls and does not mutate the live database;
2. baseline and variant manifests differ when a prompt or policy changes;
3. identical baseline/variant inputs produce a valid zero-delta comparison;
4. changed prompt, model, workflow bundle, fixture, candidate, or policy
   identity prevents an invalid comparison from being published;
5. the same scenario matrix is used for paired runs;
6. incomplete, timed-out, failed-oracle, mixed-identity, and insufficient-rep
   runs remain visible and cannot silently enter the baseline;
7. quality regressions are reported even when speed or cost improves;
8. missing truth-set conditions suppress precision/recall claims;
9. rerunning a completed pair does not overwrite it;
10. concurrent or repeated invocations do not collide in scratch or retention
    roots;
11. reports reconcile stage intervals to lifecycle wall time;
12. existing F01–F10 registered tests remain green.

Run the repository quality gate after Go changes:

```bash
make fmt && make lint && make test
bench/scripts/tests/run-all.sh
```

For shell-only changes, run the focused tests and the complete registered
wrapper. Do not claim the full epic is verified if any required suite is
skipped or if evidence is only serialized/summary-only when concurrency or
production-path behavior is required.

## Documentation and demo requirements

Update `bench/README.md` and `docs/demos/E40/demo-script.md` with the final
commands and a short end-to-end walkthrough. Include:

- the no-spend preflight;
- baseline creation;
- prompt change and digest inspection;
- variant execution;
- comparison report interpretation;
- recovery/resume behavior;
- explicit provider-spend and resource-limit gates;
- the distinction between test-backed evidence, UAT, production-path proof,
  and publication eligibility.

## Safety and collaboration constraints

- Preserve unrelated dirty files, branches, worktrees, and ignored artifacts.
- Stage only task-owned paths.
- Do not delete or reset the Shark database.
- Do not invent credentials, endpoints, provider results, or evidence.
- Do not use completed Shark status as proof that a baseline is publishable.
- Keep parent/orchestrator ownership of claims, transitions, and releases if
  the implementation invokes Shark workflow dispatch.
- Inspect `--help` and current script contracts before assuming syntax.
- Stop and report a blocker if a real provider/runtime input cannot be
  discovered safely; provide an offline path and name the missing authority.

## Definition of done

The work is complete only when a fresh operator can follow one documented
sequence to:

1. validate the E40 corpus and scenario packages;
2. preview the exact matrix with zero provider calls;
3. establish and retain a named baseline;
4. change a prompt or workflow bundle and see its identity change;
5. rerun the same matrix as a named variant;
6. generate a report that says where the variant is better, worse,
   unchanged, indeterminate, or invalid across quality, speed, tokens, cost,
   and review/artifact metrics;
7. reproduce the comparison from retained artifacts;
8. run the complete focused and registered test suites;
9. understand every evidence limitation without relying on agent memory.

Report exact changed files, commands run, test results, retained artifact
paths, and any remaining external/runtime blocker. Do not report success based
only on a plan or a green subset of tests.
