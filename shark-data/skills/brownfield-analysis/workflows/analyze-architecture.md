# Architecture Analysis

## Purpose

Document how the system is designed — its layers, components, how they connect, what patterns
they use, and what they depend on.

## System overview

Produce `architecture/system-overview.md` covering:

- Overall architecture style (layered, microservices, event-driven, etc.)
- Deployment topology — where things run and how they connect
- A Mermaid diagram showing all major components and their relationships
- Technology choices and their rationale, where discernible from the code
- Key architectural decisions evident from the code structure

## Components

In `architecture/components.md`, for each major component/module document:

- **Purpose** — what it does
- **Responsibilities** — what it owns
- **Technology** — language, framework, runtime
- **Size** — file count, approximate lines of code
- **Key entry points** — main classes, handlers, controllers
- **Interfaces** — what it exposes to other components
- **Dependencies** — what it consumes from other components

## Dependencies

In `architecture/dependencies.md`, write two sections.

**Internal dependencies** — how modules depend on each other:

- A Mermaid dependency diagram
- For each dependency: source, target, type (compile / runtime / test), and reason

**External dependencies** — third-party libraries and services:

- For each: name, version (from the actual build file), purpose, license (if determinable),
  and health status — actively maintained, deprecated, or carrying known vulnerabilities

## Design patterns

In `architecture/patterns.md`, for each pattern identified document:

- Pattern name and category (creational, structural, behavioral, architectural)
- Where it is used — specific file paths
- How it is implemented — a brief code-level description
- Why it is used — the problem it solves in this codebase

Patterns worth looking for: DAO/Repository, Factory, Singleton, Observer, Strategy,
Template Method, Decorator, Adapter, Facade, MVC/MVP/MVVM, event-driven messaging, CQRS,
Saga, Circuit Breaker, Retry, Cache-aside.
