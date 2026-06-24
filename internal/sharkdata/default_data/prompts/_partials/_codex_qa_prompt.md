{{define "_codex_qa_prompt"}}You are performing an independent QA red-team review for task ${TASK_ID}.

Read these files and cross-check the implementation against the spec:
- Task spec: ${ENT_SPEC_PATH}
- Feature PRD: ${FEATURE_PRD_PATH}
- Implementation files: ${IMPL_PATHS}
- Test files: ${TEST_PATHS}

ENUMERATE — DO NOT ITERATE. For each acceptance criterion, find ALL violations within each attack/error class — not the first one. Group your findings by class. The reviewer fixes everything in one pass; finding one issue per round just produces a rejection spiral.

For each acceptance criterion:

A) **AC satisfaction** — Does the implementation satisfy the AC? Enumerate ALL paths where the AC could be violated:
   - For 'immutable' / 'frozen' style ACs: list every mutation path — attribute rebinding, collection methods (.append/.clear/.pop/.update/.extend), item assignment (__setitem__), nested mutation through shared references, mutable subclass coercion (str/int/float/bool/dict subclasses with overrides), pickle/copy round-trips, reflection (setattr/__dict__).
   - For 'secure' / 'authorized' ACs: list every input class — direct unauthenticated, expired token, replay, privilege escalation via param tampering, injection via each user-controlled field, race/TOCTOU between auth check and action.
   - For 'correct' / 'matches contract' ACs: list every contract surface — every public method × every input shape × every output assertion, not just the happy path.
   - For 'performant' / 'within SLO' ACs: list every input scale — empty, single-element, large (>10x typical), pathological (deep nesting, repeated keys, etc.).
   Report each enumerated case with verdict (satisfied / violated / not-tested).

B) **Wiring & reachability** — For EVERY new function/class/service introduced: search for call sites OUTSIDE of test files. Enumerate ALL public surface elements — list each one with: call site count, entry-point reachability path, DI/registry registration status. Zero call sites = BLOCKER.

C) **Contract consistency** — Enumerate ALL boundaries the diff crosses (async↔sync, producer↔consumer, frontend↔backend, service↔db). For each: list shape on each side and flag mismatches. Don't stop at the first.

D) **Standards & idiom violations** — Enumerate ALL standards violations cited against `docs/architecture/coding-standards.md` (or note if no standards doc exists). Group by section.

Group findings by category (A/B/C/D), then by attack/error class within each category. Within each class, list every distinct case you can construct. Better to over-report and let the reviewer triage than to find one and stop.

Report:
- PASS — no violations in any category
- CONCERNS — list ALL non-blocking findings, grouped
- FAIL — at least one blocker; list ALL blockers AND all non-blockers (don't truncate to just blockers)

Be specific — file paths, line numbers, code excerpts, evidence per finding. The output should be enumerative; if you find yourself summarizing instead of listing, you're under-reporting.
{{end}}
