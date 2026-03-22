# Technical Debt Summary

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 6 — Technical Debt Assessment

## Debt Inventory

| ID | Title | Severity | Category | Affected Components |
|----|-------|----------|----------|-------------------|
| TD-01 | Fat controller anti-pattern (direct repo calls from CLI) | High | Architecture | cli/commands, repository |
| TD-02 | Schema god file (db.go at 2,335 LOC) | Medium | Complexity | internal/db/ |
| TD-03 | Entity-type code duplication in CLI helpers | Medium | Duplication | cli/commands/*_helpers.go |
| TD-04 | Pre-release Turso client library | Medium | Dependency | internal/db/, go.mod |
| TD-05 | CGO cross-compilation complexity | Medium | Build | Makefile, .goreleaser.yml |
| TD-06 | Missing HTTP API authentication | Low | Security | cmd/server/ |
| TD-07 | Deprecated depends_on field in tasks table | Low | Schema | internal/db/db.go |
| TD-08 | Migration file growth (migrate.go at 1,560 LOC) | Low | Complexity | internal/db/migrate.go |
| TD-09 | Feature/Epic helper formatting duplication | Medium | Duplication | cli/commands/ |
| TD-10 | Disabled Homebrew/Scoop distribution | Low | Distribution | .goreleaser.yml |
| TD-11 | Incomplete formatter implementations (Markdown, YAML, CSV) | Low | Completeness | internal/formatters/ |
| TD-12 | Deprecated status phase derivation uses hardcoded values | Low | Legacy | internal/status/derivation.go |
| TD-13 | Search command still uses direct repository pattern | Low | Architecture | internal/cli/commands/search.go |

## Severity Distribution

| Severity | Count | Items |
|----------|-------|-------|
| **Critical** | 0 | — |
| **High** | 1 | TD-01 |
| **Medium** | 5 | TD-02, TD-03, TD-04, TD-05, TD-09 |
| **Low** | 7 | TD-06, TD-07, TD-08, TD-10, TD-11, TD-12, TD-13 |

## Key Observations

1. **No critical debt** — The codebase is in good health overall
2. **Architecture debt is being actively addressed** — E15 (service layer) and E21 (entity polymorphism) epics target the two biggest items
3. **Test coverage is strong** — 1.87:1 test-to-production ratio mitigates risk
4. **Dependencies are current** — No known CVEs in direct dependencies
5. **Most debt is structural, not functional** — Code works correctly but could be better organized

---

See also: [Outdated Components](outdated-components.md) | [Maintenance Burden](maintenance-burden.md) | [Remediation Plan](remediation-plan.md)
