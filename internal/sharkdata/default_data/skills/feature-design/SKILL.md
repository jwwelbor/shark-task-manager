---
name: feature-design
description: Feature-level UX design workflows producing wireframes and prototypes. Use during SDLC feature refinement after product-design evidence reaches its exit point. Consumes feature PRDs, D08 personas, D09 journey maps, and optionally D11 friction points.
version: 1.0.0
---

# Feature Design Skill

Feature design turns a feature PRD into wireframes and, when needed, a
prototype specification that the build team can implement from.

This skill runs during the SDLC after product-level design evidence exists. It
consumes product-level artifacts such as D08 personas, D09 journey maps, and
D11 friction points, then produces feature-scoped artifacts inside the feature
directory.

If D08 and D09 are unavailable, stop and tell the host that product-design
artifacts must be completed through at least D09 before feature-level design can
be trusted.

## Required Inputs

The host workflow should provide:

- `feature_key`
- `feature_title`
- `feature_prd_path`
- `feature_directory`
- `personas_path` for D08 personas
- `journey_maps_path` for D09 journey maps
- `friction_points_path` for D11 friction points, when relevant

Use the active feature PRD path and workflow-provided feature metadata. If the
feature key, PRD path, or feature directory is unavailable, ask the host for the
feature key and PRD path before continuing. Do not infer feature scope from a
directory listing alone.

## Available Workflows

### Wireframes (`workflows/wireframes.md`)

Low-fidelity wireframes for all key screens and states of a feature.

- **Use when:** Layout, information architecture, screen inventory, and user
  flow need to be defined before tasks or implementation begin.
- **Input:** Feature PRD, D08 personas, D09 journey maps, and any relevant D11
  friction points.
- **Output:** `docs/plan/<epic-id>/<feature-id>/wireframes.md`

### Prototype (`workflows/prototype.md`)

Interaction specification for the feature's primary flows.

- **Use when:** The feature has a multi-step flow, a novel interaction, or a
  stakeholder/user-validation need that static wireframes cannot satisfy.
- **Input:** `wireframes.md`
- **Output:** `docs/plan/<epic-id>/<feature-id>/prototype.md`
- **Skip when:** Wireframes are sufficient to write tasks and start build.

## Typical Sequence

For most features:

`feature PRD -> wireframes.md -> tasks -> build`

Add `prototype.md` when interaction complexity warrants validating the flow
before task writing or implementation.

## Relationship to Product Design

| Skill | Scope | Artifacts | Runs |
|---|---|---|---|
| `product-design` | Whole initiative | D01-D14 in `docs/product/` | Once for the initiative |
| `feature-design` | One feature | `wireframes.md`, `prototype.md` in `docs/plan/<epic-id>/<feature-id>/` | Per feature |

Feature design consumes product-design artifacts. It does not produce or
replace them.

## Tools Required

- **Read** - Verify and read the PRD, D08, D09, and any relevant D11 artifacts.
- **Write** - Create the wireframe and prototype files.
- **Question mechanism** - Ask the host or user for missing required inputs and
  prototype-format decisions.
- **frontend-design skill** - Use when creating or reviewing an interactive
  HTML prototype that accompanies `prototype.md`.

## Output Standards

All feature design artifacts must:

1. Live in `docs/plan/<epic-id>/<feature-id>/` with canonical filenames:
   `wireframes.md` and `prototype.md`.
2. Reference D08 persona IDs verbatim, such as `P1` or `P2`.
3. Reference D09 journey stage names verbatim.
4. Include rationale for non-obvious design decisions.
5. Cover all relevant screen states: default, populated, empty, loading, error,
   and success.
