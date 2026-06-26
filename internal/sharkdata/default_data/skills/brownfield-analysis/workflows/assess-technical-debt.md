# Technical Debt Assessment

## Purpose

Identify everything that is outdated, risky, or expensive to maintain. This assessment often
drives the most business value, because it tells stakeholders where to invest.

## Debt summary

In `technical-debt/summary.md`, build a high-level inventory of debt items, each with:

- ID, title, severity (Critical / High / Medium / Low), category
- A one-line description
- Affected components

## Outdated components

In `technical-debt/outdated-components.md`, for each outdated dependency or framework:

- Current version (from the actual build file) vs. latest version
- End-of-life status
- Whether a migration path exists
- Relative effort estimate (small / medium / large)
- Risk of staying on the current version

## Security vulnerabilities

In `technical-debt/security-vulnerabilities.md`, document:

- Known CVEs in dependencies — check versions against known vulnerability databases
- OWASP Top 10 concerns visible in the code
- Hardcoded credentials or secrets
- Insecure configurations
- Missing security controls — authentication, authorization, input validation

Rate each: Critical (actively exploitable), High (significant risk), Medium (should fix),
Low (minor concern).

## Maintenance burden

In `technical-debt/maintenance-burden.md`, document areas that are expensive to maintain:

- Overly complex code (high cyclomatic complexity)
- Duplicated logic
- Tightly coupled components
- Missing or inadequate tests
- Poor documentation
- Manual processes that should be automated

## Remediation plan

In `technical-debt/remediation-plan.md`, write prioritized action items, each with:

- Priority rank
- What to do
- Why — the risk reduced or value gained
- Estimated effort
- Dependencies on other remediation items
- Recommended sequence

The recommended sequence here is a recommendation about the *remediation work itself* — it
describes the order in which fixes should be applied, not the order in which this analysis
area was performed.
