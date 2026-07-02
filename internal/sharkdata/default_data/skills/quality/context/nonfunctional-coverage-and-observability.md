# Non-Functional Coverage And Observability Reference

Use this file when Step 5.6 or Step 5.7 of `workflows/test-planning.md` needs the full matrices and examples.

## ISO 25010 coverage matrix

For each AC, decide whether each quality characteristic applies and cite test evidence or justify `N/A`.

| Characteristic | What it means | Common evidence |
|---|---|---|
| **Functional Suitability** | The described behavior works | Unit or integration tests for the AC |
| **Performance Efficiency** | Time, throughput, and resource use stay within limits | Benchmark, latency assertion, load test |
| **Compatibility** | Works alongside other features or versions | Cross-version or sibling-feature integration test |
| **Usability** | Discoverable, learnable, error messages are clear | UX walkthrough, error-message review |
| **Reliability** | Recovers from failure and behaves under stress | Retry test, failure injection, idempotency test |
| **Security** | Boundaries enforced, secrets protected, inputs validated | Auth bypass test, injection test, secret scan |
| **Maintainability** | Modular, analyzable, testable, modifiable | Lint clean, complexity within bounds, test isolation |
| **Portability** | Adaptable across environments | Cross-OS test, container build test |

Example per-AC matrix:

```markdown
| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-T1 | ✅ TC-001..3 | N/A | N/A | N/A | N/A | N/A | ✅ lint | N/A |
| AC-T2 | ✅ TC-T2-01..12 | N/A | N/A | N/A | ✅ TC-T2-13 | ✅ TC-T2-14..16 | ✅ complexity | N/A |
```

Every cell must be deliberate. No blanks.

## Observability design

For each behavior, design runtime evidence showing it works in production.

| Behavior | Metric | Log | Trace span | Alert threshold |
|---|---|---|---|---|
| Reset email sent within 2 minutes | `password_reset.email_sent_count`, `password_reset.email_send_latency_seconds` | `INFO password_reset.email_dispatched ...` | `password_reset.send_email` | p95 > 90s for 5 min |
| Reset link expires after 1 hour | `password_reset.expired_link_attempts_count` | `WARN password_reset.expired_link ...` | `password_reset.validate_token` with `result=expired` | spike for 10 min |

Rules:

- Every behavior gets at least one of metric, log, or trace.
- Pure internal behaviors may be marked `internal — no observability` with justification.
- New instrumentation is part of the implementation contract, not a follow-up.
- Alerts are optional unless SLOs exist.
