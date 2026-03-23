# Security Vulnerabilities

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 6 — Technical Debt Assessment

## Known CVEs

No known CVEs detected in current dependency versions. All dependencies are at recent versions.

## OWASP Top 10 Assessment

| Category | Risk | Assessment |
|----------|------|-----------|
| Injection | Low | SQLite uses parameterized queries throughout repositories; no raw SQL concatenation observed |
| Broken Auth | N/A | CLI tool — no authentication (local use). Turso uses auth tokens from file/env |
| Sensitive Data | Low | Auth tokens stored in separate files, not in `.sharkconfig.json` by default. `.gitignore` excludes DB files |
| XXE/XML | N/A | No XML processing |
| Broken Access Control | N/A | Single-user CLI tool |
| Security Misconfig | Low | SQLite PRAGMAs properly configured (foreign keys on, WAL mode) |
| XSS | N/A | No web frontend |
| Deserialization | Low | JSON parsing via standard library; no custom deserialization |
| Known Vulnerabilities | Low | Dependencies current; no known CVEs |
| Logging | Low | No sensitive data logged; verbose mode shows debug info only |

## Specific Findings

### Turso Auth Token Handling

- **Finding**: Auth tokens can be stored in `.sharkconfig.json` directly (not recommended)
- **Mitigation**: Default pattern uses separate file (`auth_token_file`) or environment variable
- **Risk**: Low — documented as security best practice in guides
- **File**: `internal/config/config.go`, `docs/TURSO_QUICKSTART.md`

### Database File Permissions

- **Finding**: SQLite database file created with default filesystem permissions
- **Risk**: Low — CLI tool designed for single-user local use
- **Mitigation**: Not needed for current use case; would matter if deployed as multi-user service

### No Input Length Limits

- **Finding**: Entity titles and descriptions have no explicit length limits beyond NOT NULL
- **Risk**: Low — CLI tool, not web-facing
- **Mitigation**: Database naturally limits via SQLite page size

## Summary

The project has a small attack surface as a local CLI tool. No critical or high-severity security issues found. The Turso cloud integration is the most security-relevant component, and it follows reasonable practices (separate token files, environment variables).

See also: [Summary](summary.md) | [Outdated Components](outdated-components.md)
