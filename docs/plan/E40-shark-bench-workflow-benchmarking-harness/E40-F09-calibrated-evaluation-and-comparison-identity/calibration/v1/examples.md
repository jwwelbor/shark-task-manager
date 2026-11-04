# Review the calibration examples

Each example is a separate fictional scenario. Its reference sheet defines
the supplied facts; the planning entries are the work you must assess. Use
[the rubric](rubric.md) and record your grades in [the worksheet](human-scores.md).
There are no suggested grades or hidden human labels.

## CAL-001: Export a personal reading list

### Supplied references

**CAL-001-REF-01:** In this fictional desktop application, an individual reader
maintains up to 200 books locally. The existing fields are title, author,
reading state, and private notes. The fixture supports retrieving books in
insertion order. Its standard library has a CSV writer. The user request is:
"I want title, author, and reading state in a spreadsheet. My private notes
must stay out of the export." Network access is unavailable. No performance
measurements or competitor research have been supplied.

### Planning entries

**CAL-001-D01 — Problem and users:** Individual readers need a spreadsheet
copy of their local list without retyping fields. Export title, author, and
reading state; private notes, imports, and shared libraries are out of scope.

**CAL-001-D02 — Success and constraints:** Export a header and one row per
stored book in insertion order, with exactly three columns. Reopening the CSV
must preserve commas, quotes, and line breaks within fields. An empty list
produces the header only. Check that private notes never occur in output.
Preserve the local data model and operate without network access.

**CAL-001-D03 — Research:** REF-01 supplies both the requested columns and the
privacy constraint. It also identifies an available standard-library CSV
writer. No external market claim is needed. Throughput is unmeasured; the
200-book fixture defines the initial test size, not a proven timing bound.

**CAL-001-D04 — Feasibility and trade-offs:** Use the existing retrieval method
and CSV writer. Hand-building comma-separated strings is simpler initially
but risks incorrect quoting; a new export service or remote spreadsheet API
would add scope without meeting a new requirement. Keep insertion order so
export order is predictable. Surface file-write errors without modifying books.

**CAL-001-D05 — Synthesis and decomposition:** First add a pure conversion from
books to three export fields; then add CSV file output; then verify ordinary,
empty, quoted, multiline, privacy, and write-failure cases. All work stays local
and leaves the stored reading list unchanged. This completes export only.

## CAL-002: Tag several inventory items

### Supplied references

**CAL-002-REF-01:** A fictional local inventory app is used by one hobbyist.
Each item has a stable ID and a list of tags. A single-item tagging operation
exists. A user has requested tagging several selected items in one command.
Selection may include an unknown ID. The user has not decided whether unknown
IDs should abort the command or permit partial success. No user study,
competitor data, or load measurements are available.

### Planning entries

**CAL-002-D01 — Problem and users:** Tagging should be more convenient for
people with many items. Add bulk tagging for selected inventory items.

**CAL-002-D02 — Success and constraints:** The command accepts a tag and a list
of IDs. Correctness means all valid selected items acquire the tag. It should
be fast and easy. Keep the existing storage format.

**CAL-002-D03 — Research:** REF-01 says single-item tagging already exists, so
it may be reusable. Users probably expect bulk actions. There is no evidence
about common selection sizes or failure preferences.

**CAL-002-D04 — Feasibility and trade-offs:** Loop over IDs and call the existing
operation. This is easy to implement. If an ID is unknown, skip it and continue;
the command can report a count afterward. A batch transaction could be added
later if users want it.

**CAL-002-D05 — Synthesis and decomposition:** Add argument parsing, iterate
the existing operation, and print the changed-item count. Test several valid
IDs and an empty input. Ask the user whether partial success is acceptable
before releasing, although the proposed implementation already skips unknown
IDs. Duplicate tags and duplicate IDs are not yet specified.

## CAL-003: Find notes by title

### Supplied references

**CAL-003-REF-01:** In this fictional local notebook, one student has 300 notes
with stable IDs, titles, and body text. The request is: "Find titles containing
my search text, ignoring capitalization." The product must work offline. The
existing title-listing operation returns titles and IDs in creation order.
No survey, market figures, or ranking research has been supplied.

### Planning entries

**CAL-003-D01 — Problem and users:** Everyone needs smarter search. Build an
AI search feature for students, companies, and schools.

**CAL-003-D02 — Success and constraints:** Make search delightful and accurate.
It should always find the best note. There are no meaningful constraints.

**CAL-003-D03 — Research:** Most people prefer semantic search. Our system will
improve productivity by 70 percent, and all leading notebooks use cloud AI.
The fictional market is worth a billion dollars. No sources are listed.

**CAL-003-D04 — Feasibility and trade-offs:** Send every note title and body to
a hosted language model on every query. This is obviously the best approach,
so no simpler alternatives need consideration.

**CAL-003-D05 — Synthesis and decomposition:** First integrate a cloud account,
then upload notes, then add team recommendations and paid subscriptions.
Title substring matching can wait. No acceptance cases are identified.

## CAL-004: Export a pantry list

### Supplied references

**CAL-004-REF-01:** A fictional pantry tracker stores item name, quantity, and
location locally for one household. The user asks for an offline CSV export
of those three fields. The fixture has at most 500 items and a standard-library
CSV writer. Existing retrieval uses insertion order. Imports and cloud sharing
are excluded. No timing or memory measurements have been supplied.

### Planning entries

**CAL-004-D01 — Problem and users:** Household members want to print or examine
the pantry list in a spreadsheet. Add an offline export of name, quantity, and
location. Imports and online sharing remain out of scope.

**CAL-004-D02 — Success and constraints:** Write exactly the stated columns,
one row per item, preserving insertion order. Quote embedded commas and quotes
correctly. Empty lists still export a header. Use the existing data model and
operate without network access.

**CAL-004-D03 — Research:** REF-01 supports the format, storage boundary, and
available CSV writer. With at most 500 items, export is guaranteed to complete
within 5 ms and use less than 1 KB of memory; no measurements are needed.

**CAL-004-D04 — Feasibility and trade-offs:** Use the standard CSV writer.
Streaming rows avoids constructing an extra full CSV string, while returning
a string would simplify preview code. Choose streaming to a file because no
preview is requested. File size depends on field lengths; timing remains an
open measurement despite the guarantee stated in D03.

**CAL-004-D05 — Synthesis and decomposition:** Implement the three-field row
mapping, file writing, and tests for empty lists, ordering, escaping, and file
errors. Keep the offline scope. A guaranteed 5 ms completion time is retained
as an acceptance condition without defining a machine or maximum field size.

## CAL-005: Tag selected research notes

### Supplied references

**CAL-005-REF-01:** A fictional local research notebook has stable note IDs and
sets of tags. A read-only ID lookup and a transactional single-tag update are
available. A researcher requests adding one tag to a selection of up to 20
notes. Their stated preference is: "If any selected ID is missing, tell me
which ones and change nothing." Duplicate selected IDs and already-present
tags should have no extra effect. No external user study is supplied.

### Planning entries

**CAL-005-D01 — Problem and users:** Individual researchers currently repeat
the same tagging command on each note. Provide one local operation for a
bounded selection. Tag removal, saved groups, and cross-user sharing are
excluded.

**CAL-005-D02 — Success and constraints:** A valid selection receives exactly
one instance of the tag per note. Duplicate IDs and existing tags are harmless.
Any missing ID produces a list of missing IDs and leaves every note unchanged.
Reject selections above 20 before writing. Preserve the storage format.

**CAL-005-D03 — Research:** REF-01 directly supplies the selection bound and
failure policy. Reuse the lookup and transaction capabilities it identifies.
There is no claim about competitor behavior or adoption. Whether users will
need more than 20 notes remains unknown and is outside this request.

**CAL-005-D04 — Feasibility and trade-offs:** Deduplicate IDs, validate all of
them using read-only lookup, then apply updates in one transaction. Independent
writes could leave a partial selection changed if a later write fails, which
violates the requested behavior. The transaction costs some coordination but
fits the bounded selection. Report storage failure and roll back.

**CAL-005-D05 — Synthesis and decomposition:** Define input normalization and
validation first, then transactional updates, then the result summary. Verify
missing-ID rollback, write-failure rollback, duplicates, existing tags, empty
input, and the 20-item boundary. The plan implements only tag addition and
preserves the all-or-nothing rule throughout.

## CAL-006: Find saved recipes by title

### Supplied references

**CAL-006-REF-01:** A fictional recipe notebook serves one home cook and stores
up to 250 recipes locally. The request is: "Search words anywhere in a recipe
title, ignoring case, so I can find a recipe I saved." The existing query lists
recipe titles and IDs in creation order. Body search and ingredient filtering
are not requested. The desired behavior for an empty query is undecided.
The fixture includes mixed-case titles and titles containing accented letters.
No language-specific search policy or performance data has been supplied.

### Planning entries

**CAL-006-D01 — Problem and users:** The home cook needs to find a saved recipe
without scrolling through the full list. Limit the first change to title
search in the local notebook.

**CAL-006-D02 — Success and constraints:** Given a query, show matching recipe
IDs and titles without changing saved recipes. Results should ignore case and
keep creation order. Empty-query behavior and accented-letter behavior need
confirmation. No response-time target has been chosen.

**CAL-006-D03 — Research:** REF-01 provides the user need and existing listing
operation. There is no reason yet to add an index for a 250-recipe fixture.
That is an initial design assumption rather than a measured performance claim.
Matching accents remains an unresolved product decision.

**CAL-006-D04 — Feasibility and trade-offs:** Filter the title list using a
case-normalized substring comparison. It avoids an indexing dependency but
does not define accent matching or whether multiple query words must be
adjacent. A tokenizer could handle separated words, at the cost of another
matching policy. Confirm those policies before finalizing acceptance cases.

**CAL-006-D05 — Synthesis and decomposition:** Add title filtering and ordinary
case-insensitive tests after the matching decisions are recorded. Then add
ingredient filtering to make the feature useful for cooking and test it too.
Ingredient filtering was not requested in D01 or REF-01. Empty-query tests
remain pending the user's answer.
