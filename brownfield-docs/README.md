# Shark Task Manager — Brownfield Analysis

> Comprehensive documentation of the Shark Task Manager codebase
> Generated: 2026-03-22
> Analysis scope: Full codebase (~1,197 Go files, ~392K LOC)

## Quick Reference

| Attribute | Value |
|-----------|-------|
| Language | Go 1.23.4 |
| Database | SQLite (local) / Turso (cloud) |
| Architecture | Clean Layered (CLI → Service → Repository → DB) |
| Build | Make + GoReleaser |
| CI/CD | GitHub Actions (3 workflows) |
| Test Ratio | 1.9:1 (test-to-code) |
| Schema Version | 10 (16 tables) |
| Dependencies | 9 direct, ~100+ transitive |
| Tech Debt | 0 critical, 0 high, 5 medium, 7 low |

## Documents by Category

### Overview
- [Project Overview](project-overview.md) — Executive summary, tech stack, statistics
- [Technical Debt Report](technical-debt-report.md) — Executive tech debt summary

### Architecture (Phase 2)
- [System Overview](architecture/system-overview.md) — Architecture style, deployment, technology choices
- [Components](architecture/components.md) — All major components with responsibilities
- [Dependencies](architecture/dependencies.md) — Internal + external dependency matrix
- [Design Patterns](architecture/patterns.md) — 16 patterns with evidence and file paths

### Code Reference (Phase 3)
- [Program Structure](reference/program-structure.md) — Complete file inventory by package
- [Interfaces](reference/interfaces.md) — All public interfaces and contracts
- [Data Models](reference/data-models.md) — All entities, DTOs, relationships
- [API Reference](reference/api-reference.md) — CLI commands, HTTP endpoints, JSON formats

### Behavior Analysis (Phase 4)
- [Business Logic](behavior/business-logic.md) — Business domains and rules
- [Workflows](behavior/workflows.md) — End-to-end process flows with sequence diagrams
- [Decision Logic](behavior/decision-logic.md) — Decision trees and flowcharts
- [Error Handling](behavior/error-handling.md) — Error hierarchy, exit codes, recovery

### Visual Documentation (Phase 5)
- **Structural**
  - [Component Diagram](diagrams/structural/component-diagram.md)
  - [Package Dependencies](diagrams/structural/package-dependencies.md)
- **Behavioral**
  - [Sequence Diagrams](diagrams/behavioral/sequence-diagrams.md)
  - [Activity Diagrams](diagrams/behavioral/activity-diagrams.md)
- **Data Flow**
  - [Request Flow](diagrams/data-flow/request-flow.md)
- **Architecture**
  - [Deployment Diagram](diagrams/architecture/deployment-diagram.md)
  - [CI/CD Pipeline](diagrams/architecture/cicd-pipeline.md)

### Technical Debt (Phase 6)
- [Summary](technical-debt/summary.md) — Debt inventory with severity ratings
- [Outdated Components](technical-debt/outdated-components.md) — Dependency freshness
- [Security Vulnerabilities](technical-debt/security-vulnerabilities.md) — OWASP assessment
- [Maintenance Burden](technical-debt/maintenance-burden.md) — High-cost areas
- [Remediation Plan](technical-debt/remediation-plan.md) — Prioritized action items

### Code Quality (Phase 7)
- [Code Metrics](analysis/code-metrics.md) — LOC, file counts, coverage
- [Complexity Analysis](analysis/complexity-analysis.md) — Hotspots, large packages
- [Dependency Analysis](analysis/dependency-analysis.md) — Freshness, circularity
- [Security Patterns](analysis/security-patterns.md) — Auth, validation, secrets

### Migration Readiness (Phase 8)
- [Component Order](migration/component-order.md) — Dependency-ordered migration sequence
- [Test Specifications](migration/test-specifications.md) — Required test coverage
- [Validation Criteria](migration/validation-criteria.md) — Acceptance criteria, rollback

### Specialized Documentation (Phase 9)
- [SQLite Schema](specialized/database/sqlite-schema.md) — 16 tables, indexes, triggers, migrations
- [CI/CD Infrastructure](specialized/infrastructure/cicd-pipeline.md) — Workflows, GoReleaser, environments

### Progress
- [PROGRESS.md](PROGRESS.md) — Analysis progress tracker
