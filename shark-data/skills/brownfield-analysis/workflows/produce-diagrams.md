# Visual Documentation

## Purpose

Create diagrams that make the system's structure and behavior visually comprehensible. Good
diagrams are often the most-referenced part of any project documentation.

All diagrams use Mermaid syntax (see `../context/output-conventions.md`). If a diagram already
exists from another analysis area — for example, an architecture overview — reference it
rather than duplicating it. Skip any category that does not apply: a simple library has no
deployment diagram or environment topology.

## Structural diagrams (`diagrams/structural/`)

- **component-diagram.md** — system-level component relationships
- **class-diagrams.md** — key class hierarchies for core business domains; focus on the
  important abstractions and inheritance trees, not every class
- **package-dependencies.md** — the package/module dependency graph

## Behavioral diagrams (`diagrams/behavioral/`)

- **sequence-diagrams.md** — the most important interaction flows, rendered as sequence
  diagrams
- **activity-diagrams.md** — complex multi-step business process activities

## Data-flow diagrams (`diagrams/data-flow/`)

- **data-sync-flow.md** — how data moves between stores (replication, cache sync, ETL), if
  applicable
- **request-flow.md** — how a request flows through the system from entry to response

## Architecture diagrams (`diagrams/architecture/`)

- **deployment-diagram.md** — where things run (containers, servers, cloud services)
- **cicd-pipeline.md** — the build and deployment pipeline
- **environment-topology.md** — all environments and how they relate
