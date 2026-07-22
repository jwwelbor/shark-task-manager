---
research_schema: 2
entity_key: E34-F02
entity_type: feature
recipe: universal
rigor: standard
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F02 Research Report: Evidence-Based Demo Script Skill

## Scope

E34-F02 creates a portable, explicit `/shark-rider demo <epic|feature>
[--draft]` Mode-3 recipe, an embedded `demo-script` skill, and its reference
template. It turns already-recorded scope, evidence, and readiness facts into a
stakeholder walkthrough; it does not add a Shark workflow status, accept work,
provision an environment, invent a command or credential, or create triage
items automatically.

This is a STANDARD feature (the recorded complexity decision is 10/27). The
shared terms are: **normal mode** (every demonstrated claim has verified,
dated/environment-scoped evidence), **draft** (uncaptured evidence is an
explicit gap), **demonstrated now**, **not demonstrated / pending integration**,
and **accepted risks and overrides**. `assessor_verdict` remains the independent
UAT result; `owner_decision` is a separately reported approval or
`override-accept`; and a `contract-only` handoff remains pending until its
activation owner closes the live production-path proof.

The parent has no feature-independent research report registered or present in
the E34 plan folder. Its applicable PRD/context is `epic.md`, `requirements.md`,
and `scope.md`. E34-F03's report and I-01 map are the upstream readiness
contract for this feature, not duplicated policy text.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md`;
  `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md#i-01-readiness-evidence-shape`.
- [x] `affected_implementation_or_contract` — Evidence: `skills/shark-rider/SKILL.md` (Mode-3 Rider actions and `shark skill get`
  boundary); `internal/sharkdata/default_data/manifest.yaml` and
  `internal/sharkdata/default_data/README.md` (embedded skill registration and
  bundle layout); `skills/shark-rider/verbs/triage.md` (explicit-action and
  triage handoff pattern).
- [x] `related_work` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/epic.md`, `requirements.md`,
  and `scope.md`; sibling
  `E34-F01-harness-aware-prompt-rendering/{feature.md,decisions.md}`; sibling
  `E34-F03-deliverable-feature-decomposition-and-staged-integ/{feature.md,research-report.md,spec.md,test-plan.md}`;
  and `E34-interaction-map.md`.
- [x] `pattern_contract` — Evidence: `skills/shark-rider/SKILL.md` and
  `skills/shark-rider/verbs/{triage.md,revalidate.md,update-docs.md}` establish
  local Mode-3 verb procedures; `internal/sharkdata/default_data/skills/uat/SKILL.md`
  and `references/redteam-rubric.md` establish evidence collection,
  independent-verdict, explicit-approval, and non-automatic-triage boundaries.

## Capability map

| Capability | Evidence | Decision | E34-F02 responsibility |
|---|---|---|---|
| Rider local Mode-3 action routing | `skills/shark-rider/SKILL.md`; existing verb procedures | EXTEND | Add a `demo` verb using the established explicit local-recipe shape; do not add a `shark demo` CLI command or workflow transition. |
| Embedded reusable craft-skill delivery | `internal/sharkdata/default_data/manifest.yaml`; `README.md` | EXTEND | Add `demo-script` as a canonical embedded skill and register its normalized identity in the manifest; keep the Rider verb as host-side coordination that retrieves the portable skill. |
| Related-document discovery and reference-note linkage | `skills/shark-rider/SKILL.md` Mode-1 data-plane contract; F02 REQ-F-003 | REUSE | Write `docs/demos/<entity-key>/demo-script.md`, retain evidence below `evidence/`, and attach the result through existing related-document/reference-note mechanisms. |
| Evidence-led, independent UAT and explicit human decision | `internal/sharkdata/default_data/skills/uat/SKILL.md`; `references/redteam-rubric.md` | REUSE | Consume assessor evidence and owner facts; never let generated demo output approve, rewrite, or replace UAT. |
| I-01 staged-readiness contract | `E34-F03.../research-report.md`; `E34-F03.../spec.md`; `E34-interaction-map.md` | EXTEND (consumer) | Consume the exact nine readiness fields read-only and classify open activation as pending integration. F02 is the activation owner and closure key; it must not redefine the producer contract or create a second I-##/test. |
| Harness-aware prompt rendering | `E34-F01.../feature.md`; `E34-F01.../decisions.md` | REUSE | Keep demo recipe prompt assembly compatible with Shark-owned prompt assembly and client-owned execution; F02 does not alter model metadata or harness routing. |
| A stack-specific UI/GitHub/Playwright recipe, automatic provisioning, or demo-as-UAT | F02 `feature.md` Out of Scope and REQ-NF-001/002; E34 `scope.md` | CONTRADICTS | Exclude it. Evidence type follows the documented project surface, unknown setup is reported as a gap, and UAT/workflow authority remains separate. |

## Findings

1. Shark Rider already separates host-local Mode-3 procedures from Shark CLI
   data-plane commands. A demo action fits this model: the verb validates an
   epic/feature key, gathers state and linked documentation through existing
   commands, retrieves the embedded skill, and writes only the planned
   artifact. It should not become a new Go command, entity type, or default
   status gate.

2. The embedded bundle has an explicit skill manifest. A new portable
   `demo-script` skill therefore requires both its `skills/demo-script/` content
   and a matching canonical manifest entry; otherwise bundle validation will
   treat the skill layout as drift. The reference template belongs with the
   skill so projects do not need a web-stack-specific external dependency.

3. Existing Rider verbs are concise procedures that state their safety boundary
   and then use the documented CLI/skill surface. The new verb should likewise
   reject non-epic/non-feature keys, make normal-versus-draft behavior explicit,
   and route discovered discrepancies only as `/shark-rider triage` candidates
   after the usual duplicate search and user confirmation.

4. F03 has already established the authoritative readiness handoff. I-01 is
   currently a predeclared `contract-only` producer-to-consumer handoff, with
   E34-F02 as the activation owner. Until this feature proves its live caller
   chain and closes the obligation, completion markers, a demo artifact, or an
   override cannot put that claim in `Demonstrated now`. See the upstream F03
   report, specification, and interaction map rather than a copied policy
   table.

5. The UAT skill supplies the right truthfulness pattern: collect concrete
   evidence, distinguish an independent assessment from the owner's decision,
   and never silently turn a missing proof into success. The demo skill needs
   a lighter presentational output, but must preserve that separation and show
   blockers/conditions in their dedicated sections.

6. There is no existing demo verb or embedded demo-script skill in the checked
   Rider verb catalog or embedded manifest. The capability is therefore NEW at
   the product level, while its routing, bundle-registration, evidence, and
   readiness components are extensions/reuses of established contracts.

## Decisions

1. **Create a portable recipe, not a runtime feature.** Implement a host-local
   `/shark-rider demo` Mode-3 verb plus an embedded `demo-script` skill and
   template. The verb obtains portable instructions through `shark skill get`;
   no schema, Go command, database table, or workflow status is warranted.

2. **Use evidence types selected by surface.** The template must support UI
   captures, CLI transcripts, API request/response plus state, SDK examples,
   pipeline artifacts/data, infrastructure health/metrics, and background
   trigger/log/result evidence. It may execute only project-documented steps;
   missing guidance becomes a normal-mode evidence gap or an explicitly marked
   draft gap.

3. **Make scenario traceability mechanical and visible.** Each scenario needs
   stakeholder value, source criterion, prerequisites/data, presenter action,
   expected observable, evidence type/path with environment/date,
   acceptance/readiness classification, reset/recovery, and limitations.
   Normal mode must verify the source and evidence exist before claiming the
   scenario is demonstrated.

4. **Consume I-01 without acceptance authority.** Read the F03 readiness
   shape at demo time; preserve `assessor_verdict`, `owner_decision`, and open
   conditions separately. A contract-only, unclosed, or override-accepted
   claim belongs under pending integration or accepted risks, never under
   `Demonstrated now`.

5. **Persist the generated script as a discoverable artifact.** Use
   `docs/demos/<entity-key>/demo-script.md` with evidence in its `evidence/`
   directory, then use existing related-document and reference-note contracts.
   Report discrepancy candidates rather than creating backlog entities.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md,E34-interaction-map.md}`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/{feature.md,decisions.md}`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/{feature.md,research-report.md,spec.md,test-plan.md}`
- `skills/shark-rider/SKILL.md` and `skills/shark-rider/verbs/{triage.md,revalidate.md,update-docs.md}`
- `internal/sharkdata/default_data/{manifest.yaml,README.md}`
- `internal/sharkdata/default_data/skills/uat/{SKILL.md,references/redteam-rubric.md}`

RECOMMENDED OUTCOME: standard
