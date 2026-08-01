# Success Metrics

**Epic**: [Question and Decision Workflow Management](./epic.md)

## Release Readiness Measures

| Measure | Method | Release target |
|---|---|---|
| Lifecycle coverage | Automated and UAT evidence covers creation through resolution. | All six epic UAT scenarios pass. |
| Serial safety | Concurrency/claim tests exercise competing responders. | No test permits more than one active claimant for a Question. |
| Gate precision | Tests exercise linked blocking and unlinked work. | Every linked blocking case pauses; every unlinked case proceeds. |
| Resolution provenance | Tests inspect consequential resolutions. | Every such resolution has a valid kind and required record pointer. |
| Context safety | Validation tests submit prohibited and oversized content. | Every prohibited or over-limit value is rejected. |

## Deferred Product Measures

No adoption, resolution-time, or user-satisfaction KPI is set because the
current record contains no baseline, telemetry source, target, cohort, or
metric owner. These are explicit deferred questions in [the epic PRD](./epic.md)
front matter and must be decided before a post-release outcome claim is made.
