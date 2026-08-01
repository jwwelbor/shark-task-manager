---
feature_key: E39-F03-scoped-question-blocking-gate
epic_key: E39
title: Scoped Question Blocking Gate
description: Add the direct question_blocks predicate and dispatch/advance gate that stops only explicitly linked work for an open blocking Question, without mutating linked work or changing generic blocks behavior. Includes the additive SQLite relationship-vocabulary migration required to persist question_blocks. Consumes: I-01, I-02. Produces: I-03. Depends on E39-F01 and E39-F02; E39-F04 consumes its compact safe handoff.
---

# Scoped Question Blocking Gate

**Feature Key**: E39-F03-scoped-question-blocking-gate

## Decomposition brief

Add the Question-only `question_blocks` relationship predicate and apply it at supported keyed-dispatch and advancement boundaries so an open, explicitly blocking Question stops only the directly linked candidate with a compact safe handoff. This feature consumes I-01 and I-02, produces I-03 for E39-F04, and preserves unrelated advancement, generic `blocks`, the Question's own transitions, and linked-work ownership.

## Approved scope amendment

On 2026-07-31, the owner approved the additive SQLite migration required to
persist the new `question_blocks` relationship type. The migration must widen
only the existing relationship vocabulary, preserve rows, indexes, dependent
views, and the Question cleanup trigger, and include upgrade regression
coverage. It does not authorize a new table, a general schema redesign, or any
other migration work.
