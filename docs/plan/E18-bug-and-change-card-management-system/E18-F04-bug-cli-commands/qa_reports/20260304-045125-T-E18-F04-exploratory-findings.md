# Exploratory Testing Findings: E18-F04 Bug CLI Commands

**Date:** 2026-03-04
**Tester:** QA Agent
**Session Duration:** Focused (code review + test run)

---

## Exploratory Charter

"Explore the Bug CLI command implementation to discover gaps between advertised functionality and actual behavior, focusing on flag forwarding and data persistence."

---

## Finding 1: Shared Package-Level Flag Variable for `bugLink`

**Risk Level:** Low-Medium

The package-level variable `bugLink` (line 190) is registered as a flag on both `bugCreateCmd` (line 224) and `bugListCmd` (line 229). In Cobra, each `*cobra.Command` maintains its own flag set, so the shared variable is overwritten by whichever command runs last. While functionally harmless for `create` (which correctly reads via `cmd.Flags().GetString("link")`), it is a maintenance smell. If a future developer assumes `bugLink` always reflects the current command's value, they will be misled.

**Recommendation:** Use `cmd.Flags().GetString("link")` at point-of-use (as `create` does) and do not register the same variable on multiple commands.

---

## Finding 2: `parseBugLinkFlag` cannot distinguish slugged feature from task

The function `parseBugLinkFlag("E07-F01-feature-name")` will classify the input as "task" (since len >= 3, parts[0] starts with "E", parts[1] starts with "F"). This is acknowledged in the test at line 123: `"E07-F01-feature-name has 4 parts ... -> 'task'"`. The test explicitly avoids asserting the type for this case.

This is an inherent ambiguity of the slug format. The test correctly documents it, but a user linking to `E07-F01-feature-name` with `--link` would have their entity misclassified as "task". Since `--link` filter is not yet functional (BUG-F04-002-A), this is low risk currently, but will become a bug once the filter is implemented.

**Recommendation:** Consider requiring entity type as a separate flag (`--link-type=feature --link-key=E07-F01`) or validate the parsed key against the DB during create/triage to catch misclassification.

---

## Finding 3: `TriageBugInput.Assign` field dead code

The `Assign *string` field in `TriageBugInput` (bug_dto.go line 33) has no downstream consumer. It is set by the CLI (runBugTriage line 422-425) but ignored by `TriageBug()` (bug_service.go lines 272-301). The `Bug` model has no assignment field.

This creates a misleading API: callers of `TriageBug(ctx, key, TriageBugInput{Assign: &agentID})` receive no error but the assignment is lost. Service tests for `TriageBug` do not test the `Assign` field behavior (there are no tests for it in the service test suite).

---

## Finding 4: `BugFilters` is incomplete relative to its documentation

The `bugListCmd` documentation (Long description, line 64-71) explicitly states `--link=E07-F01` as a supported filter. The flag is registered. But `BugFilters` has only `Status` and `Severity` fields. This creates a gap between documented capability and implementation across all three layers (CLI, service, repository).

---

## Finding 5: Delete confirmation prompt uses `fmt.Scanln` (interactive only)

`confirmBugDelete` (lines 661-666) reads from stdin with `fmt.Scanln`. This will hang in non-interactive environments (CI, automation). This is consistent with how other entity delete commands work in the codebase, so it is acceptable but worth noting. The `--force` flag mitigates this.

---

## Summary Table

| Finding | Risk | Actionable? |
|---------|------|-------------|
| Shared `bugLink` flag variable | Low | Yes - minor refactor |
| Slug ambiguity in `parseBugLinkFlag` | Low (now) | Future bug when `--link` filter is implemented |
| `TriageBugInput.Assign` dead code | High | Yes - fix now or remove |
| `BugFilters` missing `LinkedEntityKey` | Medium | Yes - documented bug BUG-F04-002-A |
| `confirmBugDelete` stdin-blocking | Low | No (consistent with codebase) |
