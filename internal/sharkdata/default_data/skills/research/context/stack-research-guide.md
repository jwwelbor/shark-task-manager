# Stack Research Guide

**Purpose**: Authoritative sources and research patterns per technology stack. Used by `/shark project bootstrap` for both greenfield (stack recommendation + prescriptive docs) and brownfield (coding standards reconciliation).

## Per-Stack Entries

### TypeScript / Node.js

**Official Sources**:
- TypeScript Handbook: https://www.typescriptlang.org/docs/handbook/
- Node.js Best Practices: https://nodejs.org/en/docs/guides
- ESLint Recommended Rules: https://eslint.org/docs/latest/rules/

**Research Queries** (replace `[year]` with current year):
- `"TypeScript best practices [year]"`
- `"Node.js project structure recommended [year]"`
- `"TypeScript strict mode configuration guide"`
- `"ESLint TypeScript recommended config [year]"`

**Key Standards**:
- Strict mode enabled (`strict: true` in tsconfig)
- Explicit return types on exported functions
- `interface` for object shapes, `type` for unions/intersections
- Barrel exports via `index.ts` per module
- Error handling: typed errors, never `catch(e: any)`

**Reference Architecture**:
- Feature-based directory structure (`src/features/{name}/`)
- Shared kernel (`src/shared/`, `src/lib/`)
- Entry point pattern (`src/index.ts` or `src/main.ts`)

---

### Python

**Official Sources**:
- PEP 8: https://peps.python.org/pep-0008/
- PEP 484 (Type Hints): https://peps.python.org/pep-0484/
- Google Python Style Guide: https://google.github.io/styleguide/pyguide.html

**Research Queries**:
- `"Python project structure best practices [year]"`
- `"Python type hints modern guide [year]"`
- `"pyproject.toml configuration guide [year]"`
- `"Python testing best practices pytest [year]"`

**Key Standards**:
- Type hints on all public functions
- `pyproject.toml` as single config source
- `src/` layout (`src/{package}/`)
- Docstrings: Google or NumPy style (pick one, be consistent)
- `ruff` or `black` + `isort` for formatting

**Reference Architecture**:
- `src/{package}/` with `__init__.py`
- Domain-driven modules (`src/{package}/domain/`, `src/{package}/api/`)
- `tests/` mirroring `src/` structure

---

### Go

**Official Sources**:
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://github.com/golang/go/wiki/CodeReviewComments
- Standard Project Layout: https://github.com/golang-standards/project-layout

**Research Queries**:
- `"Go project layout best practices [year]"`
- `"Go error handling patterns [year]"`
- `"Go interface design guidelines"`

**Key Standards**:
- `gofmt` / `goimports` enforced
- Accept interfaces, return structs
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Small interfaces (1-3 methods)
- Package names: short, lowercase, no underscores

**Reference Architecture**:
- `cmd/{app}/main.go` entry points
- `internal/` for private packages
- `pkg/` for public libraries (optional)
- `api/` for API definitions (proto, OpenAPI)

---

### React / Next.js

**Official Sources**:
- React Docs: https://react.dev/
- Next.js Docs: https://nextjs.org/docs
- Vercel Style Guide: https://github.com/vercel/style-guide

**Research Queries**:
- `"Next.js app router project structure [year]"`
- `"React server components best practices [year]"`
- `"Next.js TypeScript configuration guide [year]"`

**Key Standards**:
- Server Components by default, `'use client'` only when needed
- Colocation: component + styles + tests in same directory
- Named exports for components
- `use` prefix for hooks only
- Avoid prop drilling — use composition or context

**Reference Architecture**:
- `app/` directory (App Router)
- `components/` for shared UI
- `lib/` for utilities and data fetching
- `hooks/` for custom hooks
- `types/` for shared TypeScript types

---

### Java / Spring Boot

**Official Sources**:
- Spring Boot Reference: https://docs.spring.io/spring-boot/docs/current/reference/html/
- Google Java Style Guide: https://google.github.io/styleguide/javaguide.html
- Effective Java (Bloch) patterns

**Research Queries**:
- `"Spring Boot project structure best practices [year]"`
- `"Spring Boot 3 configuration guide [year]"`
- `"Java 21 modern patterns [year]"`

**Key Standards**:
- Package-by-feature, not package-by-layer
- Constructor injection (no `@Autowired` on fields)
- Records for DTOs
- `Optional` for nullable returns, never for parameters
- Structured logging with SLF4J

**Reference Architecture**:
- `src/main/java/com/company/{app}/`
- Feature packages: `{app}/order/`, `{app}/user/`
- Each feature: Controller, Service, Repository, DTOs
- `src/test/` mirroring main structure

---

### Rust

**Official Sources**:
- The Rust Book: https://doc.rust-lang.org/book/
- Rust API Guidelines: https://rust-lang.github.io/api-guidelines/
- Clippy Lints: https://rust-lang.github.io/rust-clippy/

**Research Queries**:
- `"Rust project structure best practices [year]"`
- `"Rust error handling thiserror anyhow [year]"`
- `"Rust workspace organization guide"`

**Key Standards**:
- `clippy` and `rustfmt` enforced
- `thiserror` for library errors, `anyhow` for application errors
- Prefer `&str` over `String` in function parameters
- Builder pattern for complex struct construction
- `#[must_use]` on functions returning values that shouldn't be ignored

**Reference Architecture**:
- `src/main.rs` or `src/lib.rs` entry point
- Module-per-file: `src/{module}.rs` or `src/{module}/mod.rs`
- Workspaces for multi-crate projects
- `tests/` for integration tests, inline `#[cfg(test)]` for unit tests

---

## Domain → Candidate Stacks

Use these recommendations when the user describes what they're building but doesn't specify a stack.

| Domain | Primary Recommendation | Alternatives | Notes |
|--------|----------------------|--------------|-------|
| **Web API** | TypeScript + Express/Fastify | Python + FastAPI, Go + Chi/Gin | TS if team knows JS; Go for high perf |
| **Full-Stack Web** | Next.js (React + API Routes) | SvelteKit, Remix, Django + HTMX | Next.js is safest default |
| **Data Pipeline** | Python + Pandas/Polars | Rust + Polars, Go | Python ecosystem is unmatched |
| **CLI Tool** | Go | Rust, Python (Click), Node (Commander) | Go compiles to single binary |
| **ML / AI** | Python + PyTorch/HuggingFace | Python + TensorFlow | Python is the only real option |
| **Mobile** | React Native / Expo | Flutter (Dart), Swift (iOS), Kotlin (Android) | RN if team knows React |
| **Systems / Low-Level** | Rust | C++, Go | Rust for safety guarantees |
| **Microservices** | Go | TypeScript, Java/Spring | Go for simplicity + performance |

## Scale Modifiers

Adjust recommendations based on project scale:

### MVP / Prototype
- **Prioritize**: Developer experience, speed of iteration, ecosystem breadth
- **Bias toward**: Dynamic/scripted languages, batteries-included frameworks
- **Example**: Next.js over separate React + Express; Django over Flask + manual setup

### Growth / Production
- **Prioritize**: Performance ceiling, type safety, testability
- **Bias toward**: Typed languages, explicit architecture, monitoring hooks
- **Example**: Migrate JS → TS; add proper service layer; containerize

### Enterprise / Scale
- **Prioritize**: Compliance, team scalability, long-term maintenance
- **Bias toward**: Strong typing, established frameworks, enterprise support
- **Example**: Spring Boot for Java shops; strict TypeScript with monorepo tooling

## Team Experience Weighting

> **Always weight familiarity heavily.** The wrong stack with a familiar team outperforms the right stack with a learning curve.

Decision matrix:

| Team Experience | Recommendation |
|----------------|----------------|
| Strong in language X | Use X unless fundamentally unsuitable for the domain |
| Mixed experience | Choose the stack with broadest team overlap |
| No preference / new team | Use domain default from table above |
| Learning goal stated | Acknowledge trade-off, use preferred stack, note risk |

When recommending, always state: "This recommendation assumes [experience level]. If your team is more comfortable with [alternative], that's a valid choice too."

## Coding Standards Augmentation Pattern

Used during brownfield bootstrap to reconcile discovered patterns with official guidance.

### Process

1. **Discover** what the code actually does (via `brownfield-analysis.md`)
   - Naming conventions in use
   - File organization patterns
   - Error handling approaches
   - Testing patterns

2. **Research** what official sources recommend (using per-stack entries above)
   - Official style guide rules
   - Framework-specific conventions
   - Community consensus

3. **Reconcile** discovered vs. official:
   - **Match**: Document as confirmed standard with official reference
   - **Divergence (minor)**: Document both, flag in `coding-standards-gaps.md`, recommend aligning gradually
   - **Divergence (major)**: Document current practice as standard, note official alternative in gaps file, flag for team discussion
   - **Missing (code doesn't address)**: Add official recommendation as new standard

4. **Output**:
   - `coding-standards.md` — What to follow (mix of discovered + official)
   - `coding-standards-gaps.md` — Where the code diverges from official recommendations

### Reconciliation Priority

When discovered and official conflict:
1. If the codebase is consistent in its approach → keep codebase convention, document divergence
2. If the codebase is inconsistent → adopt official recommendation
3. If neither applies → default to official recommendation
