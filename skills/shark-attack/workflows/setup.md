# Set up a Shark Attack council

## Goal

Prepare a project for a chair-led Shark Attack council without creating a
second workflow, scheduler, claim store, or provider runtime.

## Prerequisites

- Use a project that has Shark installed.
- Use the embedded Shark-data bundle or a project-local replacement for this
  skill.
- Identify the project root before you create council files.

## Steps

1. Materialize or refresh the bundled content through the host's data-install
   procedure. Use `admin install-shark-data` and `admin validate-data` only in
   that owning procedure; do not copy its CLI calls into this skill.

2. Keep council memory under `docs/council/`. Create and retain these paths:

   - `docs/council/decisions/`
   - `docs/council/handoffs/`
   - `docs/council/escalations/`
   - `docs/council/inbox/<member-id>/`

3. Copy and adapt `context/roster-schema.yaml` only when the project needs a
   roster. Keep member IDs stable, map built-in IDs to existing personas, and
   treat `model_tier` as a preference rather than routing or claim authority.

4. Validate the installed data before the council begins work. Correct
   field-specific validation errors instead of bypassing the validator.

5. Decide which council material the project can commit. Keep durable,
   shareable decisions and resolutions available to refreshed workers. If a
   project needs private deliberation, add only that local content to
   `.gitignore`; do not ignore the layout or the bounded pointers needed for
   continuity.

6. Use the matching `shark-attack/` subtree under `overrides/skills/` for a
   project override. An override replaces the matching Shark Attack file only;
   it does not replace unrelated bundled skills or personas.

## Handle missing prerequisites

If product gates are missing, recommend bootstrap or an escalation.
Do not guess product decisions. If a required team capability is unavailable,
state the reason and use an explicit sequential fallback only when it is safe;
else stop with an actionable capability gap. Preserve ordinary
`/shark-rider run` routing in every case.

## Result

The project has a validated, durable council layout. Existing Shark workflow
metadata and the owning claim service remain the only sources of routing and
lease authority.
