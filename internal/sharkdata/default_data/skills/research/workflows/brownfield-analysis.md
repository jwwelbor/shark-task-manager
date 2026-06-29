---
inputs:
  - project_root: absolute path to the existing codebase
  - output_dir: absolute path to the architecture docs directory (where the four output files are written)
  - tech_stack_path: absolute path for tech-stack.md output
  - patterns_catalog_path: absolute path for patterns-catalog.md output
  - integration_map_path: absolute path for integration-map.md output
  - architecture_overview_path: absolute path for architecture-overview.md output
  - detected_manifests: list of manifest files found during brownfield detection (package.json, go.mod, pyproject.toml, etc.)
  - scan_date: ISO date string for inclusion in document headers
outputs:
  - tech_stack: structured markdown written to tech_stack_path
  - patterns_catalog: structured markdown written to patterns_catalog_path
  - integration_map: structured markdown written to integration_map_path
  - architecture_overview: structured markdown written to architecture_overview_path
  - inferred_decisions: list of architectural decisions inferred from code (for ADR seeding)
---

# Workflow: Brownfield Analysis

**Purpose**: Analyze an existing codebase to produce foundation architecture documents
**Use for**: Discovering tech stack, patterns, integrations, and architecture from existing code
**Output**: `tech-stack.md`, `patterns-catalog.md`, `integration-map.md`, `architecture-overview.md`
**Output location**: caller-supplied (typically `docs/architecture/` in project root)

## Overview

This workflow reverse-engineers an existing codebase to produce four architecture documents. It runs as Groups B+C of the brownfield track (parallel with `map-filesystem.md` which produces `file-system.md`).

All discovery is autonomous — no interactive Q&A. Read code, infer structure, document findings.

## Required Tools

- **Read** — Reading source files, configs, manifests
- **Grep** — Pattern search across codebase
- **Glob** — File discovery
- **Bash** — Directory listing, git commands
- **WebSearch** — Technology version lookups (optional)
- **Write** — Creating output documents

## Execution Strategy

Run all four parts in parallel where possible. Parts 1-3 are fully independent. Part 4 benefits from Parts 1-3 output but can start immediately and refine.

---

## Part 1: Tech Stack Discovery → `tech-stack.md`

### Step 1.1: Read Build Manifests

Read all detected manifests (from brownfield detection phase):

```
package.json → Node.js ecosystem (dependencies, devDependencies, engines)
go.mod → Go (module path, Go version, require block)
pyproject.toml → Python (dependencies, build system, tool configs)
Cargo.toml → Rust (dependencies, edition, features)
pom.xml / build.gradle → Java (dependencies, plugins, Java version)
Gemfile → Ruby (gems, Ruby version)
composer.json → PHP (packages, PHP version)
```

Extract:
- Language + version constraints
- Framework(s) + versions
- Key dependencies (ORM, HTTP, auth, testing)
- Dev tooling (linters, formatters, bundlers)
- **Quality gate commands**: from `Makefile` targets (look for `fmt`, `lint`, `test`, `check`), `package.json` `scripts` block, `pyproject.toml` `[tool.*]` sections, or CI workflow files — record the actual format/lint/unit-test/full-suite commands for the **Quality Gate** section of `tech-stack.md`

### Step 1.2: Detect Infrastructure Signals

```
Glob: Dockerfile, docker-compose*.yml, .dockerignore
Glob: *.tf, *.tfvars (Terraform)
Glob: serverless.yml, sam-template.yaml
Glob: .github/workflows/*.yml, .gitlab-ci.yml, Jenkinsfile
Glob: nginx.conf, Caddyfile, traefik.yml
Glob: k8s/, kubernetes/, helm/
```

Extract: deployment model, CI/CD pipeline, infrastructure tooling.

### Step 1.3: Detect Database / Storage

```
Grep: "prisma", "typeorm", "sequelize", "drizzle", "knex" (Node ORM)
Grep: "sqlalchemy", "django.db", "tortoise", "peewee" (Python ORM)
Grep: "gorm", "sqlx", "ent" (Go ORM)
Grep: "diesel", "sqlx", "sea-orm" (Rust ORM)
Glob: migrations/, alembic/, prisma/schema.prisma
Glob: *.sql
Grep: "redis", "mongodb", "elasticsearch", "rabbitmq", "kafka"
```

### Step 1.4: Write `tech-stack.md`

```markdown
# Tech Stack

**Scan Date**: YYYY-MM-DD
**Project Root**: {path}

## Languages

| Language | Version | Role | Source |
|----------|---------|------|--------|
| TypeScript | 5.x | Primary | tsconfig.json |
| SQL | - | Database | prisma/schema.prisma |

## Frameworks

| Framework | Version | Purpose | Docs |
|-----------|---------|---------|------|
| Next.js | 14.x | Full-stack web | https://nextjs.org/docs |
| Prisma | 5.x | ORM | https://www.prisma.io/docs |

## Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| zod | 3.x | Schema validation |
| next-auth | 5.x | Authentication |

## Dev Tooling

| Tool | Purpose | Config File |
|------|---------|-------------|
| ESLint | Linting | .eslintrc.json |
| Prettier | Formatting | .prettierrc |
| Jest | Testing | jest.config.ts |

## Quality Gate

The commands an agent must run before advancing work. Record the project's ACTUAL commands (discover from Makefile targets, package.json scripts, pyproject.toml tool config, go.mod, etc.).

| Step | Command | When |
|------|---------|------|
| Format | {e.g. `make fmt` / `gofmt -w .` / `npm run format`} | before commit |
| Lint | {e.g. `make lint` / `go vet ./...` / `npm run lint`} | before commit |
| Unit tests | {e.g. `make test` / `go test ./...` / `npm test` / `uv run pytest tests/unit`} | before advancing |
| Integration tests | {if applicable} | when crossing a seam |
| Full suite | {the full gate, e.g. `make fmt && make lint && make test`} | before finishing a feature |
| Frontend visual check | {if applicable} | when UI changes |

## Infrastructure

| Component | Technology | Config |
|-----------|-----------|--------|
| Deployment | Docker + Vercel | Dockerfile, vercel.json |
| CI/CD | GitHub Actions | .github/workflows/ |
| Database | PostgreSQL | via Prisma |

## Rationale Notes

{Any inferred architectural decisions — e.g., "Prisma chosen for type-safe DB access with TypeScript", "Redis for session storage suggests stateless horizontal scaling"}
```

---

## Part 2: Pattern Cataloging → `patterns-catalog.md`

### Step 2.1: Naming Convention Discovery

```
Glob: **/*.{ts,py,go,rs,java,rb} (sample 20-30 files)
```

Analyze file names for:
- Case convention: camelCase, PascalCase, snake_case, kebab-case
- Suffixes: `.service.ts`, `.controller.ts`, `.repository.ts`, `_test.go`
- Prefixes: `use`, `get`, `create`, `I` (interfaces)

Analyze code identifiers:
- Variable naming: camelCase vs snake_case
- Class/type naming: PascalCase
- Constant naming: UPPER_SNAKE_CASE
- Function naming convention

### Step 2.2: Architectural Pattern Detection

```
Grep: "class.*Service", "class.*Controller", "class.*Repository"
Grep: "class.*Handler", "class.*Middleware", "class.*Provider"
Grep: "export.*function.*use[A-Z]" (React hooks)
Grep: "interface.*Repository", "interface.*Service"
Grep: "@Injectable", "@Controller", "@Service" (decorators)
```

Identify:
- **Layered architecture**: Controller → Service → Repository
- **Hexagonal / Ports & Adapters**: port interfaces + adapter implementations
- **Feature-based**: grouped by domain feature vs. grouped by technical layer
- **MVC / MVVM**: model-view-controller separation
- **Event-driven**: event emitters, message queues, pub/sub patterns

### Step 2.3: Code Organization Style

```
Bash: ls src/ (or equivalent top-level source dir)
```

Determine:
- **Feature-based**: `src/users/`, `src/orders/`, `src/payments/`
- **Layer-based**: `src/controllers/`, `src/services/`, `src/models/`
- **Hybrid**: `src/modules/users/user.service.ts`
- **Flat**: everything in `src/`

### Step 2.4: Testing Patterns

```
Glob: **/*test*, **/*spec*, **/tests/**, **/__tests__/**
```

Analyze:
- Test location: colocated vs. separate `tests/` directory
- Test naming: `*.test.ts`, `*_test.go`, `test_*.py`
- Test framework: jest, vitest, pytest, go test, etc.
- Fixtures/factories: presence of test helpers
- Mocking patterns: what's mocked and how

### Step 2.5: Write `patterns-catalog.md`

```markdown
# Patterns Catalog

**Scan Date**: YYYY-MM-DD
**Project Root**: {path}

## Naming Conventions

| Element | Convention | Example | File Reference |
|---------|-----------|---------|----------------|
| Files | kebab-case | `user-service.ts` | src/services/ |
| Classes | PascalCase | `UserService` | src/services/user-service.ts:5 |
| Functions | camelCase | `getUserById` | src/services/user-service.ts:12 |
| Constants | UPPER_SNAKE | `MAX_RETRIES` | src/config/constants.ts:3 |
| Test files | *.test.ts | `user-service.test.ts` | tests/services/ |

## Architectural Patterns

### Primary: {e.g., Layered Architecture}

{Description of the dominant pattern}

**Evidence**:
- {file:line reference showing pattern}
- {file:line reference showing pattern}

**Flow**:
```
Route Handler → Service → Repository → Database
```

### Secondary: {e.g., Repository Pattern}

{Description}

**Evidence**:
- {file:line references}

## Code Organization

**Style**: {Feature-based | Layer-based | Hybrid | Flat}

**Structure**:
```
src/
├── {directory}/ — {purpose}
├── {directory}/ — {purpose}
└── {directory}/ — {purpose}
```

## Testing Patterns

| Aspect | Pattern | Example |
|--------|---------|---------|
| Location | {colocated / separate} | tests/unit/ |
| Naming | {convention} | *.test.ts |
| Framework | {name} | Jest |
| Mocking | {approach} | jest.mock() for external deps |
| Fixtures | {present / absent} | tests/fixtures/ |

## Error Handling

| Pattern | Usage | Example |
|---------|-------|---------|
| {e.g., Custom error classes} | {where} | {file:line} |
| {e.g., Result type} | {where} | {file:line} |

## State Management (if frontend)

| Pattern | Library | Usage |
|---------|---------|-------|
| {e.g., Server state} | React Query | API data fetching |
| {e.g., Client state} | Zustand | UI state |
```

---

## Part 3: Integration Mapping → `integration-map.md`

### Step 3.1: HTTP Clients / SDKs

```
Grep: "fetch(", "axios", "got", "node-fetch", "ky"
Grep: "http.Get", "http.Post", "http.NewRequest" (Go)
Grep: "requests.get", "requests.post", "httpx" (Python)
Grep: "HttpClient", "RestTemplate", "WebClient" (Java)
```

For each found, trace to discover:
- Base URL / endpoint patterns
- Authentication method
- What data is fetched/sent

### Step 3.2: Database Connections

```
Grep: "DATABASE_URL", "DB_HOST", "MONGO_URI", "REDIS_URL"
Grep: "createConnection", "createPool", "PrismaClient"
Grep: "mongoose.connect", "MongoClient"
```

Map:
- Database type and connection method
- Schema / migration location
- Read vs. write patterns (if detectable)

### Step 3.3: Outbound API Surfaces

```
Grep: "stripe", "twilio", "sendgrid", "aws-sdk", "googleapis"
Grep: "OPENAI_API_KEY", "ANTHROPIC_API_KEY"
Grep: "S3Client", "SQSClient", "SNSClient"
```

### Step 3.4: Inbound API Surface

```
Grep: "app.get(", "app.post(", "router.get(", "router.post(" (Express)
Grep: "@Get(", "@Post(", "@Put(", "@Delete(" (NestJS/decorators)
Grep: "@app.route", "@router.get" (Python Flask/FastAPI)
Grep: "func.*Handler.*http" (Go)
Grep: "tRPC", "createRouter", "publicProcedure"
Grep: "typeDefs", "resolvers", "GraphQL"
```

### Step 3.5: Write `integration-map.md`

```markdown
# Integration Map

**Scan Date**: YYYY-MM-DD
**Project Root**: {path}

## Inbound API Surface

| Protocol | Framework | Route Pattern | Auth | Reference |
|----------|-----------|--------------|------|-----------|
| REST | Express | /api/v1/* | JWT | src/routes/ |
| GraphQL | Apollo | /graphql | Session | src/graphql/ |
| tRPC | tRPC v10 | type-safe RPC | Session | src/trpc/ |

### Key Endpoints

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | /api/users | userController.list | List users |
| POST | /api/auth/login | authController.login | Authenticate |

## Outbound Services

| Service | SDK/Client | Purpose | Config Reference |
|---------|-----------|---------|------------------|
| PostgreSQL | Prisma | Primary data store | prisma/schema.prisma |
| Redis | ioredis | Session cache | src/lib/redis.ts |
| Stripe | stripe-node | Payment processing | src/services/billing.ts |
| S3 | @aws-sdk/client-s3 | File storage | src/services/storage.ts |

## Data Flow Diagram

```
[Client/Browser]
    ↓ HTTP/WebSocket
[API Layer (Express/Next.js)]
    ↓ Service calls
[Business Logic]
    ↓ ORM queries          ↓ SDK calls
[PostgreSQL]          [External APIs]
    ↓                     (Stripe, S3, etc.)
[Redis Cache]
```

## Authentication Flow

{Describe the auth mechanism discovered: JWT, sessions, OAuth, API keys}

## Environment Dependencies

| Variable | Service | Required |
|----------|---------|----------|
| DATABASE_URL | PostgreSQL | Yes |
| REDIS_URL | Redis | Yes |
| STRIPE_SECRET_KEY | Stripe | For payments |
| AWS_ACCESS_KEY_ID | AWS S3 | For file uploads |
```

---

## Part 4: Architecture Extraction → `architecture-overview.md`

### Step 4.1: Entry Point Analysis

Find and read the main entry point(s):

```
Glob: src/index.{ts,js}, src/main.{ts,py,go,rs}, src/app.{ts,js,py}
Glob: cmd/*/main.go
Glob: manage.py, wsgi.py, asgi.py
```

Trace initialization:
- What gets bootstrapped
- Dependency injection setup
- Middleware registration
- Route registration

### Step 4.2: Module Boundary Mapping

From the file system and imports, identify:
- What constitutes a "module" in this project
- Which modules depend on which
- Where are the boundaries (what's public vs. private)

```
Grep: "^import ", "^from .* import", "require(" (sample 30-50 files)
```

Build adjacency: Module A imports from Module B → A depends on B.

### Step 4.3: Data Flow Tracing

Trace 2-3 representative flows through the system:
1. The most common read path (e.g., GET endpoint → service → DB)
2. The most common write path (e.g., POST endpoint → validation → service → DB)
3. A background/async path if present (e.g., queue consumer → processor → DB)

### Step 4.4: Write `architecture-overview.md`

```markdown
# Architecture Overview

**Scan Date**: YYYY-MM-DD
**Project Root**: {path}

## System Summary

{2-3 sentence description of what this system does and its primary architectural style}

## Architecture Style

**Primary**: {e.g., Layered Monolith, Microservices, Modular Monolith, Serverless}
**Secondary**: {e.g., Event-driven for async processing}

## Component Diagram

```
┌─────────────────────────────────────────┐
│              Presentation               │
│  (Routes, Controllers, API Handlers)    │
├─────────────────────────────────────────┤
│            Business Logic               │
│  (Services, Use Cases, Domain)          │
├─────────────────────────────────────────┤
│            Data Access                  │
│  (Repositories, ORM, Clients)           │
├─────────────────────────────────────────┤
│           Infrastructure                │
│  (Database, Cache, External APIs)       │
└─────────────────────────────────────────┘
```

## Module Map

| Module | Responsibility | Dependencies | Key Files |
|--------|---------------|--------------|-----------|
| {name} | {purpose} | {other modules} | {entry files} |

## Key Boundaries

### {Boundary 1: e.g., API ↔ Business Logic}

- **Interface**: {How they communicate — function calls, DTOs, events}
- **Enforced by**: {e.g., dependency injection, module exports}
- **Reference**: {file:line}

## Data Flow: {Representative Flow Name}

```
1. [Client] sends POST /api/orders
2. [Route Handler] validates request body (src/routes/orders.ts:15)
3. [OrderService] applies business rules (src/services/order.service.ts:42)
4. [OrderRepository] persists to database (src/repositories/order.repo.ts:28)
5. [EventEmitter] publishes OrderCreated event (src/events/order.events.ts:10)
6. [NotificationService] sends confirmation (async, src/services/notification.ts:33)
```

## Cross-Cutting Concerns

| Concern | Implementation | Reference |
|---------|---------------|-----------|
| Logging | {library/approach} | {file} |
| Error handling | {pattern} | {file} |
| Authentication | {mechanism} | {file} |
| Configuration | {approach} | {file} |
| Monitoring | {tools} | {file} |

## Deployment Architecture

{Describe how the system is deployed based on infrastructure signals}

## Known Architectural Decisions

{List any inferred ADRs — decisions visible from the code structure}

1. **{Decision}**: {What was chosen and evidence for why}
2. **{Decision}**: {What was chosen and evidence for why}
```

---

## Success Criteria

All four output documents:
- [ ] Written to `docs/architecture/`
- [ ] Contain real file:line references (not placeholders)
- [ ] Are internally consistent (tech-stack matches patterns matches integrations)
- [ ] Accurately reflect the codebase (verifiable by reading referenced files)
- [ ] Include scan date for staleness tracking

## Related Workflows

- `map-filesystem.md` — Runs in parallel (Group A), produces `file-system.md`
- `../context/stack-research-guide.md` — Used by coding standards step (Group D, after this workflow)
- `../context/brownfield-detection.md` — Runs before this workflow to confirm brownfield track

## Scope note

This workflow is the **lightweight bootstrap**: it reverse-engineers the four
`docs/architecture/` foundation documents (`tech-stack.md`, `patterns-catalog.md`,
`integration-map.md`, `architecture-overview.md`) as Groups B+C of the bootstrap flow.

For a comprehensive enterprise handoff — full technical debt audit, security assessment, migration
readiness, behavior documentation, and ~10–20 output documents — use the standalone
**`brownfield-analysis` sub-skill** (`/shark brownfield-analysis` or `/brownfield-analysis`).
