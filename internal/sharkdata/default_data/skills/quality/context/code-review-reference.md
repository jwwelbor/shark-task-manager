# Code Review Reference

Use this file when `workflows/review-code.md` needs the fuller examples for idiom checks, tooling, or report layout.

## Idiomatic language review checklist

| Language | What to check |
|---|---|
| Python | Pythonic patterns, type hints, no string-formatted SQL, prefer `pathlib`, idiomatic exception handling |
| TypeScript | Strict types, no `any` without justification, prefer `unknown` for untrusted input, discriminated unions, proper Promise typing |
| Go | Error handling, goroutine lifecycle, idiomatic struct embedding, `context.Context` propagation |
| All | Avoid nested ternaries, premature abstraction, commented-out code, and debug prints |

## Complexity tooling examples

Replace `<limit>` with the cyclomatic limit resolved by `skills/quality/context/craft-rubric.md`. Prefer the measuring command declared in local standards.

```bash
# Python
uv run radon cc <files> --min=B

# TypeScript
npx eslint <files> --rule 'complexity: ["error", <limit>]'

# Go
gocyclo -over <limit> <files>
```

## Report skeleton

For a complete clean PASS, record the selected runner, frozen scope and diff, checks, applicable totals, duration when known, and `0 defects found`. Include the location, original concern and contrary observation for each fully refuted blocker.

For PASS-with-triage or FAIL, follow the detailed sections in the review workflow. Include findings and their corrections, reuse checks, standards crosswalk, tests and counterfactuals, risk hotspots, G2 independent evidence from the selected runner, production caller chains, triage dispositions and verdict. Explain any inapplicable section. A report-only withdrawal includes its original blocker, contrary evidence and host disposition. An incomplete run records its missing evidence and actual completion counts.
