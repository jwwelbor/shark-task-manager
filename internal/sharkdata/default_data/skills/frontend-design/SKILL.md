---
name: frontend-design
description: Create distinctive, production-grade frontend interfaces with high design quality. Use this skill when the user asks to build web components, pages, or applications. Generates creative, polished code that avoids generic AI aesthetics.
when_to_use: when building any web component, page, or application interface where visual quality, distinctiveness, and a coherent aesthetic point-of-view matter
version: 1.0.0
license: Complete terms in LICENSE.txt
---

# Frontend Design Skill

## Overview

This skill is the reusable methodology for creating distinctive, production-grade
frontend interfaces that avoid generic "AI slop" aesthetics. It answers one
question: **how is frontend design done well?**

It does not own any flow, ordering, or tooling. It carries design judgment —
the frameworks, lenses, standards, and anti-patterns that make an interface
feel genuinely designed rather than defaulted.

The user provides frontend requirements: a component, page, application, or
interface to build, often with context about purpose, audience, or technical
constraints. This skill supplies the aesthetic discipline applied while
building it.

## The Core Stance

Implement real, working code with exceptional attention to aesthetic detail
and creative choices. Choose a clear conceptual direction and execute it with
precision.

**Bold maximalism and refined minimalism both succeed — the key is
intentionality, not intensity.** Elegance comes from executing a vision well,
not from how loud the vision is.

Every interface should be:

- **Production-grade and functional** — real working code, not a mockup
- **Visually striking and memorable** — there is one thing someone remembers
- **Cohesive** — a single, clear aesthetic point-of-view runs through it
- **Meticulously refined** — every detail is intentional

## Design Thinking Framework

Before writing any code, understand the context through four lenses and then
commit to a single bold aesthetic direction:

- **Purpose** — What problem does this interface solve? Who uses it?
- **Tone** — Which extreme aesthetic does this call for? (brutally minimal,
  maximalist chaos, retro-futuristic, organic/natural, luxury/refined,
  playful/toy-like, editorial/magazine, brutalist/raw, art deco/geometric,
  soft/pastel, industrial/utilitarian, and many more)
- **Constraints** — Technical requirements (framework, performance,
  accessibility).
- **Differentiation** — What makes this UNFORGETTABLE? What is the one thing
  someone will remember?

Use the tone list as inspiration, not a menu — design a direction that is
genuinely true to the context rather than picking the nearest label.

The procedure for turning these four lenses into one committed direction lives
in `workflows/commit-to-aesthetic-direction.md`.

## The Five Aesthetic Dimensions

Distinctive interfaces are built by making strong, intentional choices across
five dimensions. Each is detailed in `context/aesthetic-lenses.md`:

- **Typography** — distinctive, characterful fonts; a display/body pairing
- **Color & Theme** — a cohesive palette driven by CSS variables; dominant
  colors with sharp accents
- **Motion** — high-impact animation moments over scattered micro-interactions
- **Spatial Composition** — unexpected layouts, asymmetry, deliberate negative
  space or controlled density
- **Backgrounds & Visual Details** — atmosphere and depth instead of flat
  solids

The procedure for iterating these dimensions toward a polished result lives in
`workflows/refine-aesthetics.md`.

## Reference Material

- `context/aesthetic-lenses.md` — detailed guidance for each of the five
  dimensions
- `context/anti-slop-checklist.md` — the generic patterns to never use, and the
  variation mandate that keeps successive designs distinct
- `context/complexity-matching.md` — the standard for matching implementation
  effort to the aesthetic ambition

## Domain Procedures

- `workflows/commit-to-aesthetic-direction.md` — turn loose requirements into a
  single, defensible aesthetic direction
- `workflows/refine-aesthetics.md` — iterate the five dimensions until the
  interface is cohesive and memorable

## A Final Note

Claude is capable of extraordinary creative work. Don't hold back. Show what
can truly be created when thinking outside the box and committing fully to a
distinctive vision.

---

**Remember:** No two designs should be the same. Vary themes, fonts, and
aesthetics between generations, and never converge on a default.
