---
feature_key: E39-F02-serial-question-workflow-and-resolution-provenance
epic_key: E39
title: Serial Question Workflow and Resolution Provenance
description: Deliver the validated question_state, first-pending responder routing, bounded response recording, and resolution provenance rules on the registered Question entity. Consumes: I-01. Produces: I-02. Depends on E39-F01 and supplies the state contract consumed by E39-F03 and E39-F04.
---

# Serial Question Workflow and Resolution Provenance

**Feature Key**: E39-F02-serial-question-workflow-and-resolution-provenance

## Decomposition brief

Use the registered Question entity to provide a validated, bounded `question_state`, a route-based workflow that dispatches only the first pending responder under the existing one-active-claim rule, and explicit response, resolution, withdrawal, and supersession provenance. This feature consumes I-01 and produces I-02 for E39-F03 and E39-F04; it does not introduce a parallel responder queue, mutate linked work, or implement the blocking predicate or focused read surfaces.
