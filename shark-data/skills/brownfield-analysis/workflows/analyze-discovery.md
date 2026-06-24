# Discovery & Inventory

## Purpose

Get the lay of the land. Understand what this project is, what technologies it uses, and how
it is organized. Discovery produces the foundation every other analysis area builds on, so
invest in it: a complete structural picture lets later work proceed systematically.

## Workspace scan

Scan the entire workspace systematically and record everything — even things that seem minor.
A complete inventory now prevents surprises later.

- **Build systems**: `pom.xml`, `build.gradle`, `package.json`, `Cargo.toml`, `go.mod`,
  `pyproject.toml`, `Makefile`, `CMakeLists.txt`, and similar.
- **Languages**: identify every programming language from file extensions and build configs.
- **Frameworks**: detect frameworks from dependencies (e.g. Spring, Django, React, Express).
- **Configuration**: `.env`, `application.yml`, `config/`, container files, orchestration
  manifests.
- **CI/CD**: `.github/workflows/`, `Jenkinsfile`, `.gitlab-ci.yml`, `buildspec.yml`.
- **Infrastructure as code**: CDK, Terraform, CloudFormation, Pulumi.

## Project type identification

Classify the project based on the scan:

- **Monolith** — single deployable unit, possibly a multi-module build.
- **Microservices** — multiple independently deployable services.
- **Monorepo** — multiple projects/services in one repository.
- **Library / SDK** — reusable code consumed by other projects.
- **Hybrid** — a combination of the above.

## Module / package inventory

List every module, package, or significant directory with:

- Name and path
- Purpose (brief — depth comes in later analysis areas)
- Language and framework
- Approximate size (file count)
- Category: Application, Infrastructure, Shared/Library, Test, or Configuration

## Project overview document

Write the executive summary (`project-overview.md`) covering:

- Project identity — name, purpose, business domain
- Technology stack summary
- Module/package inventory table
- High-level architecture description (brief — the architecture area goes deeper)
- Key statistics — total files, lines of code if easily countable, number of modules

See `../context/output-conventions.md` for the document header and table conventions.
