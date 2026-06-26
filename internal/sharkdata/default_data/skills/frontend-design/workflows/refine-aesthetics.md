# Refine the Five Aesthetic Dimensions

## Purpose

Take a committed aesthetic direction and execute it with precision across the
five dimensions that make an interface distinctive and cohesive. This is the
craft work: translating one bold idea into typography, color, motion, layout,
and atmosphere that all point the same way.

Use this after a direction is committed (`commit-to-aesthetic-direction.md`),
and whenever an interface feels flat, incoherent, or generic.

## Inputs

- The committed aesthetic-direction statement and its four lens answers.
- The detailed per-dimension guidance in `context/aesthetic-lenses.md`.

## The Refinement Loop

For each of the five dimensions, make a strong, intentional choice that serves
the committed direction, then check it against the others for coherence. The
dimensions reinforce each other — a refined typographic choice should make the
color and spacing choices more obvious, not fight them.

1. **Typography** — Choose beautiful, characterful fonts. Pair a distinctive
   display font with a refined body font. Reject generic system defaults.

2. **Color & Theme** — Commit to a cohesive palette expressed through CSS
   variables. Favor dominant colors with sharp accents over timid,
   evenly-distributed palettes.

3. **Motion** — Concentrate effort on one or two high-impact moments (e.g. a
   single orchestrated page-load with staggered reveals) rather than scattering
   micro-interactions. Add hover and scroll surprises that fit the direction.

4. **Spatial Composition** — Use unexpected layouts: asymmetry, overlap,
   diagonal flow, grid-breaking elements, and either generous negative space or
   deliberately controlled density.

5. **Backgrounds & Visual Details** — Build atmosphere and depth instead of
   defaulting to flat solids: gradient meshes, noise/grain textures, geometric
   patterns, layered transparencies, dramatic shadows, decorative borders,
   custom cursors.

Consult `context/aesthetic-lenses.md` for the full guidance on each dimension.

## Guard Against Generic Output

Throughout refinement, check every choice against `context/anti-slop-checklist.md`.
If a choice matches a blacklisted pattern (generic font, cliché palette,
predictable layout), replace it. Also confirm the design does not converge on
choices used in previous generations — variety across designs is mandatory.

## Match Effort to Ambition

Apply the standard in `context/complexity-matching.md`: maximalist directions
demand elaborate code with extensive animation and effects; minimalist or
refined directions demand restraint, precision, and careful attention to
spacing, typography, and subtle detail. The implementation effort should match
what the committed direction actually needs.

## Output

A cohesive interface in which all five dimensions visibly serve one committed
direction, every choice survives the anti-slop checklist, and implementation
effort matches the aesthetic ambition.
