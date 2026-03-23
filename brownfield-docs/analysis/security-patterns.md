# Security Patterns

> Part of the Shark Task Manager Brownfield Analysis
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
