# Craft Rubric

Development and review grade code against this one rubric: authors run it on
their own diff before exit, and reviewers apply the same limits and severity.

## Resolving the limits

Look for the `## Quality Limits` table in the project's local
`coding-standards.md` or a standards sub-file it links. Resolve each limit
independently. A declared limit covers the code its `applies to` column names.

| Local Quality Limits row for this limit | Value used | Source label | When new or substantially expanded code exceeds it |
|---|---|---|---|
| Present with a value | Local value | "declared in local standards (§Quality Limits)" | Blocker |
| Present but marked "proposed — confirm" | Local value | "proposed in local standards, unconfirmed" | Same as a generic default |
| Absent, section missing, or value blank or a placeholder | Generic default below | "generic default — not declared in local standards" | Admitted only with evidenced impact |

## Generic defaults

Use a default only when the local row is absent, as described above.

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
limit through a standards refresh (bootstrap fill-gaps). A missing limit never
blocks work by being missing.

## Severity

- **Locally declared limit.** New or substantially expanded code over it is a
  blocker, because it breaks a documented standard. A declared limit is a
  documented standard, not a scoring threshold.
- **Generic default or unconfirmed proposal.** The number is a signal. Admit a
  finding only with evidenced impact under the admission criteria in
  `skills/quality/context/review-rubric.md`, which also set its severity.
- **Already over a limit.** Code that was over a limit before the change, and
  that the change does not substantially expand, is a non-blocker routed to
  triage, whatever the limit's source.

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

- Check each new or changed function and file against the resolved limits for
  cyclomatic complexity, function length, file length and nesting depth.
- Use the project's declared measuring command or other tooling where available.
- Apply the severity rules above and state each limit's source label.

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
