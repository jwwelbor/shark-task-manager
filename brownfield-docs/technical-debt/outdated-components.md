# Outdated Components

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 6 — Technical Debt Assessment

## Assessment

The project's direct dependencies are **well-maintained and current**. No end-of-life or significantly outdated components were identified.

## Dependency Version Status

| Component | Current | Latest Known | Gap | Risk |
|-----------|---------|-------------|-----|------|
| Go | 1.23.4 | 1.23.x | Current | None |
| mattn/go-sqlite3 | v1.14.32 | v1.14.x | Current | None |
| spf13/cobra | v1.10.2 | v1.10.x | Current | None |
| spf13/viper | v1.21.0 | v1.21.x | Current | None |
| pterm | v0.12.82 | v0.12.x | Current | None |
| testify | v1.11.1 | v1.11.x | Current | None |
| golang.org/x/term | v0.32.0 | Current | Current | None |
| golang.org/x/text | v0.28.0 | Current | Current | None |

## Components Requiring Attention

### 1. libsql-client-go (Pre-release)

| Property | Value |
|----------|-------|
| **Current** | `v0.0.0-20251219100830-236aa1ff8acc` |
| **Status** | Pre-release (commit hash, no semver tag) |
| **Risk** | Medium — API may change without versioning guarantees |
| **Migration Path** | Pin to tagged release when available |
| **Effort** | Small — update go.mod when tagged version exists |

### 2. golangci-lint Configuration

| Property | Value |
|----------|-------|
| **Current** | v2.9.0 (installed via Makefile) |
| **CI Version** | `latest` (golangci-lint-action@v7) |
| **Risk** | Low — version mismatch between local and CI |
| **Recommendation** | Pin CI to same version as Makefile |

## No End-of-Life Components

None of the project's dependencies are deprecated, abandoned, or approaching end of life.

## Recommended Actions

1. **Monitor Turso client releases** — Pin to tagged version when available
2. **Pin golangci-lint version in CI** — Match Makefile version for consistency
3. **Regular `go get -u`** — Run quarterly to stay current

---

See also: [Summary](summary.md) | [Dependencies](../architecture/dependencies.md)
