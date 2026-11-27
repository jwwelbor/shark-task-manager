# Craft Rubric

Development and review grade code against this one rubric: authors run it on
their own diff before exit, and reviewers apply the same limits and severity.

## Scope of the limits

The limits govern code files: the production and test source that a Quality
Limits row's `applies to` column names. Documentation, prompts, skills,
configuration, generated files and golden test data are not measured.

- An **edited code file** is one the change adds or modifies. A deleted file is
  not edited. Code moved into a file counts as edited in its destination.
- A **unit** is the file (file length) or a function in it (function length,
  cyclomatic complexity, nesting depth).

## Resolving the limits

Look for the `## Quality Limits` table in the project's local
`coding-standards.md` or a standards sub-file it links. Resolve each limit
independently.

| Local Quality Limits row for this limit | Value used | Source label |
|---|---|---|
| Present with a value | Local value | "declared in local standards (§Quality Limits)" |
| Present but marked "proposed — confirm" | Local value | "proposed in local standards, unconfirmed" |
| Absent, section missing, or value blank or a placeholder | Generic default below | "generic default — not declared in local standards" |

## Generic defaults

| Limit | Generic default |
|---|---|
| Cyclomatic complexity | 10 |
| Function length (lines) | 50 |
| File length (lines) | 500 |
| Nesting depth | 4 |

Each limit is an inclusive maximum: a measurement equal to the limit passes,
and one more is over it. A coverage target is optional and has no generic
default; with no local row, no percentage gates the change. For coverage,
"over the limit" means falling short of the target.

## Source label

State the source label on every finding or self-review line that cites a limit.
A line that relies on a generic default carries the label
"generic default — not declared in local standards" and recommends declaring the
limit: add or fill that row in the local `## Quality Limits` table, or run
bootstrap fill-gaps when the whole section is missing. A missing limit never
blocks work by being missing.

## Edited-code rule

1. Every edited code file ends within every resolved limit: its file length and
   every function in it, whatever the limit's source, including overages that
   existed before the change.
2. The default response to an overage is a behavior-preserving refactor,
   verified by existing tests that pass before and after.
3. When its safety is in doubt, follow Architect consultation below.
4. A limit exception is the only way an edited unit may stay over a limit.

## Severity table

Review and the development exit gate use these rows; the table is closed.

| Unit after the change | Declared limit | Generic default or unconfirmed proposal |
|---|---|---|
| In an edited code file, within the limit | No finding; gate item met | No finding; gate item met |
| In an edited code file, over the limit, covered by a limit exception | No finding; cite the exception; gate item met | Same as declared |
| In an edited code file, over the limit, no covering exception (new code, or an overage that predates the change) | **Blocker**: breaks a documented standard; gate unmet | **Blocker**: breaks the edited-code rule; gate unmet. The finding carries the default or proposal label and recommends declaring the limit |
| In a file the change did not edit, over the limit | Non-blocker routed to triage | Admitted only with evidenced impact under `../context/review-rubric.md`; then a non-blocker routed to triage |

A limit finding under the edited-code rule meets the admission criteria of
`../context/review-rubric.md`: it names the edited unit (the affected
maintenance operation), the unmet documented rule (the consequence) and the
measurement (the evidence). It is not a score derived from a rubric threshold.
A project that finds a generic number wrong declares its own, which then governs.

## Limit exceptions

An exception is a row in the optional `### Limit exceptions` table under the
local `## Quality Limits` section.

| Unit | Limit | Recorded value | Reason | Decision |
|---|---|---|---|---|
| File path, or file path and function name | Which limit | The unit's measurement when the decision was taken | Why a refactor is unsafe now | Architect or owner, the date, and the work item key or ADR path |

- An exception covers only a unit that existed before the change that asked for
  it, and only while the unit's measurement is at or below the recorded value.
  Growth beyond it needs a new decision. New units are never excepted.
- Only an architect or owner decision adds or changes a row. A development
  worker never adds one for its own change.
- When a tool enforces the limit, the unit also carries that tool's own
  suppression, citing the exception row.

## Architect consultation

An author who doubts that a required refactor is safe must consult the architect,
without leaving the overage in place or recording an exception.

- Return `blocked` and name the file, the unit, the limit and its source label,
  the measured value, the refactor considered and the safety concern. Where the
  result is a gate-result envelope, this text goes in
  `gate_result.no_kickback_reason`.
- The architect or owner decides: a directed refactor, recorded as a decision
  note on the work item, or a limit exception.
- The re-dispatched author reads the latest transition note and the gate report
  it cites, then acts on the decision.

## Self-review checklist

Apply every item to the diff. An author fixes each finding, or records a cited
reason why the item is not applicable.

### Reuse

- Search the codebase for an existing implementation before adding a utility,
  helper or abstraction, and call the existing one when it fits.
- Record the search, including a search that found no duplicate.

### Standards citation

- Check the changed code against the project's local standards documents.
- Cite each violation as
  `{file:line} — {rule} — {standards document §section "quoted clause"}`.
- When no clause covers a concern, report the gap and do not call it a standards
  violation. Keep it only with evidence of a concrete user, behavior, invariant,
  security or maintenance consequence.
- A demonstrated failure of a documented input or acceptance criterion is a
  correctness finding even without a clause.

### SOLID and architecture

- Single responsibility: each class or module has one clear job.
- Open/closed: stable code paths extend without modification.
- Apply Liskov substitution, interface segregation and dependency inversion
  where relevant.
- Respect the layering, dependency direction and integration points in the
  project's architecture documents.

### Idiom

- Follow the community-standard patterns of the code's language and the
  project's own conventions; replace non-idiomatic constructs.
- Leave no commented-out code, debug output or premature abstraction.

### Complexity and size

- Measure every edited code file in full, not only the changed lines, against
  the resolved limits for cyclomatic complexity, function length, file length and
  nesting depth.
- Use the project's declared measuring command or other tooling where available.
  When that command measures only new code, measure the rest of each edited file
  by tool or by hand.
- Take each verdict from the Severity table and state each limit's source label.
  Only a refactor or a covering limit exception clears an overage in an edited
  file; a recorded reason does not.

### Test design

- Each acceptance criterion has at least one test case.
- Boundaries cover minimum, maximum, empty, null, zero and negative one for each
  input.
- Negative cases assert what must not happen.
- Flaky risk: no dependence on timing, random data, network, real time or
  shared state.
- Mock discipline: a test that only asserts a mock was called does not test
  real behavior.
- Name any boundary or negative case that the criterion implies and the tests
  omit.
