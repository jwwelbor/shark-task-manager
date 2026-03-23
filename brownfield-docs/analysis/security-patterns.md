# Security Patterns

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 7 — Code Quality Analysis

## Authentication

| Pattern | Implementation | Location |
|---------|---------------|----------|
| Local CLI | None (single-user tool) | N/A |
| Turso Cloud | Auth token from file or env var | `config/config.go` |
| HTTP API | None (stub implementation) | `cmd/server/main.go` |

**Token Storage**: Turso auth tokens stored in separate files (`auth_token_file` config field) or environment variables (`TURSO_AUTH_TOKEN`). Not committed to repository.

## Authorization / Access Control

Not applicable — single-user CLI tool with no multi-user access control.

## Data Protection

| Aspect | Pattern | Detail |
|--------|---------|--------|
| At rest | SQLite file permissions | OS-level file permissions |
| In transit (Turso) | TLS | libSQL client uses HTTPS/WSS |
| Secrets in config | Separate file | Auth tokens stored outside .sharkconfig.json |
| Database file | .gitignore | `shark-tasks.db` excluded from git |

## Input Validation

| Layer | Pattern | Location |
|-------|---------|----------|
| Model | Structural validation | `models/validation.go` — non-empty, range checks |
| Service | Business rule validation | `services/*.go` — workflow, dependencies |
| Repository | Parameterized queries | `repository/*.go` — `?` placeholders, no SQL injection |
| Database | CHECK constraints | `db/db.go` — priority range, status values |
| CLI | Cobra arg validation | `commands/*.go` — arg count, flag types |

**SQL Injection Protection**: All SQL queries use parameterized statements (`?` placeholders). No raw SQL string concatenation observed.

## Logging & Audit Trail

| What | How | Location |
|------|-----|----------|
| Status changes | Automatic history records | `entity_history` table, DB triggers |
| Task transitions | Legacy + new history | `task_history` + `entity_history` tables |
| CLI operations | Verbose mode only | `--verbose` flag, standard `log` package |
| Error logging | stderr | `cli.Error()` function |

**Gaps**: No structured logging framework. No request-level audit trail for HTTP API (not yet implemented).

## Secrets Management

| Secret | Storage | Access |
|--------|---------|--------|
| Turso auth token | File (`~/.turso/shark-token`) | Read by config loader |
| Turso auth token (alt) | Env var (`TURSO_AUTH_TOKEN`) | Read by config loader |
| Database credentials | N/A (SQLite is local) | Filesystem access |

**Best Practice**: Documentation recommends file-based token storage with `chmod 600` permissions.

## Summary

The security posture is appropriate for a single-user CLI tool:
- No web-facing attack surface (HTTP API is a stub)
- Parameterized SQL prevents injection
- Secrets kept out of configuration files
- Cloud communication uses TLS via libSQL client

See also: [Security Vulnerabilities](../technical-debt/security-vulnerabilities.md) | [Dependency Analysis](dependency-analysis.md)
=======
> Generated: 2026-03-20
> Phase: 7 — Code Quality Analysis

## Overview

Shark Task Manager is a **local-first CLI tool** — not a network-facing application. Its attack surface is limited to local file access and optional cloud database connections.

## Authentication & Authorization

| Aspect | Implementation |
|--------|---------------|
| **User authentication** | None (local tool, trusts OS user) |
| **API authentication** | HTTP server has no auth (development/internal use) |
| **Cloud DB auth** | Turso auth token via file or env var |
| **Agent identity** | Agent type tracked in task status updates (informational, not enforced) |

### Turso Authentication
- Auth token stored in separate file (recommended): `~/.turso/shark-token`
- Alternatively via environment variable: `TURSO_AUTH_TOKEN`
- Config references token file path, not the token itself
- `.sharkconfig.json` uses env var references (`$SHARK_AUTH_TOKEN_FILE`)

Source: `.sharkconfig.json`, `internal/db/` Turso initialization

## Input Validation

| Layer | Validation Type | Notes |
|-------|----------------|-------|
| **Models** | Structural validation | Non-empty checks, range checks (priority 1-10) |
| **Services** | Business validation | Workflow transition validation, dependency checks |
| **Database** | Constraint validation | CHECK, NOT NULL, UNIQUE, FOREIGN KEY |
| **CLI** | Argument parsing | Cobra validates arg counts |

### SQL Injection Prevention
- All database queries use **parameterized statements** (`?` placeholders)
- No string concatenation in SQL queries observed
- Repository methods consistently use `db.QueryRowContext(ctx, query, args...)`

Source: `internal/repository/*.go`

## Data Protection

| Data Type | Protection |
|-----------|-----------|
| **Database file** | OS filesystem permissions only |
| **Auth tokens** | Stored in separate files with recommended `chmod 600` |
| **Configuration** | `.sharkconfig.json` — no sensitive data by default |
| **Task files** | Plain markdown in `docs/plan/` |

### Sensitive Data Handling
- No encryption at rest (SQLite file is unencrypted)
- No PII handling (task management data only)
- Auth tokens never stored directly in config (file reference or env var)
- `.gitignore` should exclude `shark-tasks.db` and token files

## Secrets Management

| Secret | Storage | Notes |
|--------|---------|-------|
| Turso auth token | File (`~/.turso/`) or env var | Never in config file |
| GitHub release tokens | GitHub Secrets (CI only) | Standard GitHub Actions pattern |
| Homebrew/Scoop tokens | Not configured (disabled) | Would use GitHub Secrets |

## Logging & Audit

- **Status change audit trail**: `task_history` table records all status transitions
- **Agent tracking**: Status updates record which agent made the change
- **Rejection notes**: Backward transitions can require reason documentation
- **Verbose mode**: `--verbose` flag enables debug logging to stderr
- **No sensitive data in logs**: Task titles and status values only

## Network Security

- **Default mode**: No network access (local SQLite)
- **Turso mode**: HTTPS/WSS connection to Turso cloud
  - TLS enforced by libsql client library
  - Auth token transmitted in connection handshake
  - No custom certificate handling observed

## Identified Security Considerations

| ID | Finding | Severity | Notes |
|----|---------|----------|-------|
| SEC-1 | HTTP API server has no authentication | Low | Documented as development/internal use |
| SEC-2 | SQLite database not encrypted at rest | Low | Local file, normal for CLI tools |
| SEC-3 | No rate limiting on CLI commands | Info | Not applicable for local CLI |
| SEC-4 | Turso client uses pre-release library | Low | API stability risk, not security risk |

## OWASP Top 10 Assessment

| Category | Applicability | Status |
|----------|--------------|--------|
| A01: Broken Access Control | Low (local tool) | N/A |
| A02: Cryptographic Failures | Low | Token file permissions recommended |
| A03: Injection | Medium | SQL injection prevented via parameterized queries |
| A04: Insecure Design | Low | Clean architecture with input validation |
| A05: Security Misconfiguration | Low | Minimal configuration surface |
| A06: Vulnerable Components | Low | Dependencies current, no known CVEs |
| A07: Auth Failures | Low | No auth needed for local CLI |
| A08: Data Integrity | Medium | Foreign keys + check constraints enforced |
| A09: Logging Failures | Low | Audit trail via task_history |
| A10: SSRF | N/A | No server-side request features |

---

See also: [Technical Debt Security](../technical-debt/security-vulnerabilities.md) | [Dependencies](../architecture/dependencies.md)
>>>>>>> Stashed changes
