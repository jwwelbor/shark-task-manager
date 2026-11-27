# Architecture decision records

All decisions are proposed. Effort estimates describe relative implementation complexity within this epic.

## ADR 1 · Use trusted descriptors and typed harness adapters

- **Alternatives.** A generic command/argument template can register simple CLIs immediately. It pushes permission semantics, streams, errors and session handling into configuration. Dedicated adapters can enforce and test each CLI contract. They require a build for deployment choices that could be data.
- **Decision.** Use typed Claude, Codex and Copilot adapters behind a registry, with a descriptor for approved executable, modes and binding options. Share process supervision. Add generic protocol support only after two integrations demonstrate the same bounded output and permission contract.
- **Rationale and compatibility.** Keep `AgentDispatcher` as the starting seam and preserve legacy result transport. Typed adapters own protocol differences and reject unsupported requirements. Operators can change model assignments and create compatible bindings through config. A genuinely new protocol requires an adapter. Complexity **high**.

## ADR 2 · Store versioned profiles in YAML

- **Alternatives.** CSV works for flat inventory and imports but requires additional files or encodings for fallbacks, review policies and binding metadata. JSON makes strict schemas and tooling straightforward but is less convenient for hand-authored profiles. Embedding everything in workflow YAML couples deployment to topology.
- **Decision.** Use strict versioned YAML with named routes, profiles and references. Point to it from project configuration. Export canonical redacted JSON for machines and benchmarks. For profile reproducibility, retain both source bytes/digest and normalized semantic identity.
- **Rationale and compatibility.** YAML matches the workflow ecosystem and expresses typed nested policy. Existing project config and E40 version 1 profile files remain readable through compatibility boundaries. Complexity **medium**.

## ADR 3 · Use discovery where authoritative and retain evidence snapshots

- **Alternatives.** Checked-in model lists are reproducible but drift with releases and account policy. Live-only discovery improves freshness where supported but varies by mode and account and can fail at launch. Monitor successful inference to establish account access at the cost of latency and spend.
- **Decision.** Use mode- and binding-specific snapshots populated through supported discovery or dated operator assertions. Reuse the Codex protocol, fixtures and normalization from B083. Require approved canary evidence when discovery cannot establish a required capability. Avoid maintaining separate Shark and benchmark lists. Keep compatibility, entitlement and availability independent.
- **Rationale and compatibility.** Pin snapshots for deterministic planning and compare their provenance to launch context. Existing default-bearing routes retain compatibility until strict opt-in. Complexity **high**.

## ADR 4 · Route by complexity and retain explicit selection

- **Alternatives.** Explicit per-step pins offer precision and simplify forensic review at the cost of portability and repetitive edits. Automatically inferred complexity reduces YAML edits but can drift with text or persona. Declared complexity with a profile makes assignments predictable and requires workflow owners to classify the work.
- **Decision.** Use low, medium and high as intent, with named route references for explicit exceptions and separately validated effort.
- **Rationale and compatibility.** The same workflow can run across qualified profiles. Keep legacy pins and omitted fields under the compatibility rules. There is no inference from an agent name or effort setting. Complexity **medium**.

## ADR 5 · Enforce review independence on candidate-bound evidence

- **Alternatives.** A distinct agent role or requested route is easy to implement but does not establish effective independence. Requiring all dimensions for every review gives stronger separation at a higher availability cost. Per-step required dimensions support the review policy and make the cost visible.
- **Decision.** Use fresh sessions, different canonical model by default and conjunctive explicit dimensions. Check author and reviewer identities, candidate and policy before admitting the existing gate. Pause when proof is missing. Retain existing authorized manual outcomes with their own provenance.
- **Rationale and compatibility.** Existing gates remain valid under their current admission policy until they opt in. Strict routing adds an admission predicate and uses existing evidence rather than a new gate system. Complexity **high**.

## ADR 6 · Use deterministic precedence and resolve complete bindings

- **Alternatives.** Field-by-field merging is convenient for small overrides but can pair an incompatible model with a new harness or provider. Whole-file replacement avoids that bug and makes global defaults cumbersome. Named route references support atomic replacement at each precedence level.
- **Decision.** Apply the [precedence contract](02-architecture.md#precedence-and-resolution) using complete route bindings. Validate standalone effort modifications against the final binding. Treat administrative permissions and strict review requirements as invariants.
- **Rationale and compatibility.** Keep legacy harness metadata resolution separate from new execution flags. New overrides require explicit selection and provenance. Complexity **medium**.

## ADR 7 · Classify failures and require proof before replay

- **Alternatives.** Stop on all failure preserves correctness but loses recoverable capacity. Retry on exit code or use opaque CLI fallback to improve completion but risk repeating writes and losing route evidence. A parent-owned policy can distinguish launch failure, safe replay and uncertainty at the cost of durable state.
- **Decision.** Use typed failure phase/effect classification, configured bounded alternates and guarded recovery. Error text alone cannot establish replay safety. Disable hidden harness fallback where policy requires a verified route.
- **Rationale and compatibility.** Starts with prelaunch fallback, preserving existing fail/blocked outcome behavior. Existing sidecars and transitions remain valid. Complexity **high**.

## ADR 8 · Keep E35 topology and share execution resolution across callers

- **Alternatives.** Encode model switching as workflow outcomes, which leaks capacity and authentication incidents into business transitions. Maintain distinct resolvers in core, Rider and E40, which keeps changes local and permits drift. Share execution resolution at the keyed action/assignment boundary.
- **Decision.** Keep E35 outcome ownership and adopt one versioned decision at dispatch. E40 uses the same resolver for profile-only runs. Rider keeps its host adapter and attestation boundary. Preserve old assignments during consumer migration.
- **Rationale and compatibility.** Extend the canonical step DTO and reuse dispatch, release and gate contracts. E40 can retire workflow rewriting as a runtime requirement once manifest/evidence parity holds. Complexity **high**.

## PRD document decisions

- Use the [epic PRD](./epic.md) as the sole source for business justification and release outcomes. Keep requirement-to-test traceability in the [catalog](./requirements.md) and boundary cases in the [scope file](./scope.md).
- Include [stakeholder responsibilities](./personas.md) because authors, operators, gate owners and benchmark maintainers make different decisions. Include [user journeys](./user-journeys.md) because selecting a route, qualifying a workflow, admitting a review and comparing runs have different failure paths.
- Omit a standalone success-metrics file. E41 measures observable release behavior against fixtures and qualification records. The PRD makes no adoption, revenue or cost-savings forecast.
- Design must produce `E41-interaction-map.md` for the four features. Reconcile the [existing architecture proposal](./02-architecture.md) with the canonical `architecture.md` sections before assigning stable interaction IDs. Record the verified map relationship through Shark during design.
- Q012 records the setup-discovery scope choice. Operator approval of versions, accounts and spend belongs at qualification. The existing implementation approval remains a separate decision.
