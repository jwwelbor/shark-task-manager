---
type: cross-epic-integration-map
epic: E34
last_updated: 2026-08-30
---

# E34 cross-epic execution map

E34 has one cross-epic handoff. The architect reviewed shared contracts,
state, sequencing, and API boundaries. A separate CX handoff is not applicable:
this CLI-only artifact contract has no end-user screen or journey artifact.

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|----|---------------|------------------|---------------------|-------------------------|-----------------------|----------------|--------|-----------------------|
| X-14 | E34 — Prompt and Skill Improvements | E40 — Shark Bench: workflow benchmarking harness | Compare canonical E34 policy with a reconciled project-style configuration and retain identity for tier, evidence, recurrence, and integration-closure scenarios. | [I-05 CanonicalAdoptionManifest v1](./architecture.md#i-05-canonicaladoptionmanifest-v1); E34-F09 feature.md § Requirements 8 | State that the benchmark is later validation, show the selected bundle and baseline identity, and do not present an E40 result as an E34 delivery gate. | E34-F09 Override Drift Visibility and WWGM Reconciliation | deferred | Deferred to E40's own decomposition — E40 must add the benchmark scenario and coverage pointer once its harness can execute the scenario. |

## Sequencing and closure

E34-F08 produces I-05. E34-F09 uses it to define the canonical and reconciled
configuration identities. E40 consumes those identities only after its harness
can execute the scenario. This edge is `deferred` (a contract-only staged edge, in the epic integration_review gate's vocabulary): it is fully declared
for the producer side (E34) but its activation and coverage-pointer creation
belong entirely to the consumer (E40), so it neither blocks E34 delivery nor
claims a live E40 caller today.

**Contract-only edge disposition (X-14):**
- **Counterpart identity**: E40 — Shark Bench: workflow benchmarking harness.
- **Current status (live)**: `active` (`shark get E40 --field status`).
- **Shared-contract evidence**: [I-05 CanonicalAdoptionManifest v1](./architecture.md#i-05-canonicaladoptionmanifest-v1)
  (producer-side structural coverage shipped and tested by E34-F08's
  `TestIntegrationReviewAdoptionManifestFieldListMatchesArchitecture`); E34-F09
  feature.md § Requirements 8 (consumer-facing identity requirements).
- **Activation owner**: E40. E40's own feature decomposition creates the
  benchmark scenario and the resolving coverage pointer; E34 has no further
  action once E40 exists as a live caller.
- **Closure key**: `X-14-E40-benchmark-scenario-coverage-pointer` — closes when
  E40 adds a feature whose test-plan.md resolves this key against a real,
  executing benchmark scenario that consumes I-05's adoption manifest.

The global [cross-epic integration map](../../product/cross-epic-integration-map.md)
is the source of the stable X-14 row. No additional X-## rows were identified
during design on 2026-08-25.
