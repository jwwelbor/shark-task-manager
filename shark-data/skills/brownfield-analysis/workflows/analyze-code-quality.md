# Code Quality Analysis

## Purpose

A quantitative assessment of code health — metrics, complexity, dependency health, and
security patterns.

## Code metrics

In `analysis/code-metrics.md`, capture:

- Total lines of code, by language
- File counts, by type
- Module/package counts
- Average file size
- Test-to-production code ratio

## Complexity analysis

In `analysis/complexity-analysis.md`, identify:

- The most complex files/classes — by size, nesting depth, number of dependencies
- Hotspots — files that are both complex and frequently referenced
- God classes / god modules — files that do too much
- Deeply nested logic

## Dependency analysis

In `analysis/dependency-analysis.md`, capture:

- Dependency count — internal and external
- Dependency freshness — how up-to-date versions are
- Circular dependencies
- Unused dependencies, if detectable
- Heavy dependencies — large transitive dependency trees

## Security patterns

In `analysis/security-patterns.md`, document:

- Authentication implementation patterns
- Authorization / access-control patterns
- Data encryption at rest and in transit
- Input validation patterns
- Logging and audit-trail patterns
- Secrets management approach
