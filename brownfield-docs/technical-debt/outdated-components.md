# Outdated Components

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 6 — Technical Debt Assessment

## Dependency Freshness Assessment

| Dependency | Current Version | Status | Risk |
|-----------|----------------|--------|------|
| go-sqlite3 | v1.14.32 | Current | None |
| cobra | v1.10.2 | Current | None |
| viper | v1.21.0 | Current | None |
| pterm | v0.12.82 | Current | None |
| testify | v1.11.1 | Current | None |
| yaml.v3 | v3.0.1 | Stable | None |
| golang.org/x/term | v0.32.0 | Current | None |
| golang.org/x/text | v0.28.0 | Current | None |
| **libsql-client-go** | **v0.0.0-20251219** | **Dev commit** | **Medium** |

## Detailed Concerns

### TD-01: libsql-client-go (Medium)

- **Current**: `v0.0.0-20251219100830-236aa1ff8acc` (development commit, no stable tag)
- **Latest stable**: No stable release exists for this library
- **Risk**: API may change without notice; no semantic versioning guarantees
- **Migration path**: Monitor for stable v1.0 release; consider vendoring or pinning to specific commit
- **Effort**: Small (update go.mod when stable release available)
- **Impact**: Cloud database functionality (Turso backend) could break on API changes

### Go Version

- **Current**: Go 1.23.4
- **Latest**: Go 1.24.x (as of early 2026)
- **Risk**: Low — Go maintains excellent backward compatibility
- **Action**: Update when convenient; no urgent need

### Build Tools

| Tool | Current | Status |
|------|---------|--------|
| golangci-lint | v2.9.0 | Current |
| GoReleaser | v2.x | Current |
| Air | latest | Always current (auto-installed) |

**Overall**: Dependencies are well-maintained and current. The only concern is the unstable Turso client library.

See also: [Summary](summary.md) | [Security Vulnerabilities](security-vulnerabilities.md)
