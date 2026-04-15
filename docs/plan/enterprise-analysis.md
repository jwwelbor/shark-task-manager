# Enterprise Shark: Centralizing AI-DLC via MCP

**Status**: Analysis / Pre-Proposal  
**Date**: 2026-04-12  
**Scope**: Architecture analysis for evolving Shark into an enterprise-grade, centralized MCP server

---

## The Vision

At its core, Shark today is a **local-first task state machine** — it tracks AI-DLC state (todo → in_progress → review → done) and provides structured workflow context to agents. Enterprise centralization flips this: instead of each repo running its own SQLite/Shark instance, there is a **central Shark MCP server** that becomes the authoritative source of truth for AI workflow state across the entire organization.

Every Claude agent, every CI gate, every human dashboard connects to this single pane of glass.

---

## Layer 1: MCP Server Design

### What Gets Exposed as MCP Primitives

**Tools** (agent-callable actions):
```
shark://tools/task.advance_status      — Move task through workflow
shark://tools/task.create              — Create work item
shark://tools/task.set_context         — Store agent working state
shark://tools/task.add_note            — Record decisions/blockers
shark://tools/task.get_resume          — Fetch full context for resuming work
shark://tools/workflow.get_next_action — "What should I do next?"
shark://tools/gate.check               — Check if entity passes a quality gate
shark://tools/gate.record_result       — Record gate pass/fail artifact
```

**Resources** (readable data):
```
shark://resources/task/{key}           — Full task with context
shark://resources/feature/{key}        — Feature + task rollup
shark://resources/epic/{key}           — Epic dashboard
shark://resources/workflow/{id}        — Workflow definition
shark://resources/project/{id}/queue   — Agent's next-up work queue
```

**Prompts** (context injection):
```
shark://prompts/resume/{task_key}      — Full resume context for agent
shark://prompts/gate/{gate_id}         — Gate criteria for agent to check itself
shark://prompts/project/{id}/standards — Project coding standards injection
```

### The Key Architectural Shift

Today Shark is reactive — agents call it. In enterprise MCP mode it becomes **partially proactive**:

- The server can push notifications when a gate blocks a transition
- It can route work to the right agent type based on workflow state
- It becomes the **orchestration backbone** for multi-agent pipelines

---

## Layer 2: Database — SQLite → PostgreSQL

### Schema Evolution for Multi-tenancy

The natural approach is **shared schema with tenant isolation** (Pool model), enforced via PostgreSQL Row Level Security:

```sql
-- Tenant hierarchy
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT UNIQUE NOT NULL,  -- "acme-corp"
    name        TEXT NOT NULL,
    plan        TEXT NOT NULL DEFAULT 'enterprise',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    UNIQUE(org_id, slug)
);

CREATE TABLE projects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    workflow_id     UUID REFERENCES workflow_definitions(id),  -- project-level override
    UNIQUE(team_id, slug)
);

-- Epics/features/tasks gain tenant columns
ALTER TABLE epics ADD COLUMN org_id UUID REFERENCES organizations(id);
ALTER TABLE epics ADD COLUMN team_id UUID REFERENCES teams(id);
ALTER TABLE epics ADD COLUMN project_id UUID REFERENCES projects(id);
```

**Row Level Security enforces tenant boundaries at the DB layer:**
```sql
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;

CREATE POLICY task_tenant_isolation ON tasks
    USING (
        org_id = current_setting('app.current_org_id')::UUID
    );
```

Every connection sets `SET app.current_org_id = '...'` at session start. No application-level tenant filtering needed — the DB enforces it.

### PostgreSQL Features That Replace SQLite Limitations

| Feature | Use Case |
|---------|----------|
| `LISTEN/NOTIFY` | Real-time push to MCP clients when task state changes |
| `JSONB` + indexes | Flexible agent context storage, fully queryable |
| Row Level Security | Tenant isolation at DB layer, not app layer |
| Logical Replication | Read replicas for analytics, regional copies |
| `pg_cron` | Scheduled workflow gates, SLA alerts |
| Full-text search | `shark search` becomes org-wide semantic search |
| Materialized Views | Pre-computed dashboards, refreshed on demand |
| Partitioning | `task_history` partitioned by month — audit trail never slows queries |

### Connection Architecture

```
Agents/Claude clients
        ↓
   MCP Server (Go)
        ↓
   PgBouncer (connection pooling — transaction mode)
        ↓
   PostgreSQL Primary
        ↓ (logical replication)
   Read Replicas (analytics, dashboards)
```

PgBouncer is critical: hundreds of concurrent AI agents must not hold hundreds of real Postgres connections. Transaction-mode pooling means each query grabs a connection, executes, and releases — perfectly suited for stateless MCP tool calls.

---

## Layer 3: Workflow Hierarchy — The Hard Problem

This is the most architecturally interesting challenge. The goal is a four-level inheritance model:

```
Enterprise Level   → global gates, compliance rules (SOX, SOC2, etc.)
  Org Level        → company coding standards, approval chains
    Team Level     → team-specific workflows (frontend vs backend vs data)
      Project Level → project overrides (greenfield vs legacy)
```

### Workflow Inheritance Model

```json
// enterprise-baseline.json
{
  "id": "enterprise-baseline-v2",
  "scope": "enterprise",
  "gates": [
    {
      "id": "security-scan",
      "required": true,
      "trigger": "before:ready_for_review",
      "check": "snyk/scan-passed"
    },
    {
      "id": "license-check",
      "required": true,
      "trigger": "before:completed"
    }
  ],
  "statuses": { "...": "base status definitions" }
}
```

```json
// team-backend.json
{
  "id": "team-backend-workflow",
  "scope": "team",
  "extends": "enterprise-baseline-v2",
  "overrides": {
    "statuses": {
      "in_code_review": { "after": "ready_for_review", "agent": "tech_lead" }
    }
  },
  "gates": [
    {
      "id": "test-coverage",
      "required": false,
      "threshold": 80
    }
  ]
}
```

### Override Permission Rules

```
IMMUTABLE (set at enterprise, cannot be overridden below):
  - Required security gates
  - Compliance checkpoints
  - Data retention policies

CONFIGURABLE (team can set, project can narrow further):
  - Status definitions
  - Agent routing per status
  - Optional gates and thresholds
  - Notification rules

PROJECT-ONLY (leaf-level configuration):
  - Custom fields
  - Local workflow shortcuts
  - Agent prompts and context injection
```

The server computes a **resolved workflow** for each project by merging the inheritance chain at request time (and caching the result). This resolved workflow is what agents receive via MCP — they never see the raw multi-level definition.

### Workflow Versioning

Workflows must be **versioned and pinned to tasks at creation time**. A task created under `workflow-v3` continues on v3 even if the team upgrades to `workflow-v4`. Explicit migration tooling promotes in-flight tasks to a new workflow version. This prevents mid-sprint disruption and is analogous to Kubernetes API versioning.

---

## Layer 4: Authentication & Authorization

### Identity Layers

| Actor | Auth Method |
|-------|-------------|
| Human users | SSO via OIDC/SAML (Okta, Azure AD, Google Workspace) |
| AI agents | Service accounts with scoped API keys |
| CI/CD systems | Short-lived OIDC JWTs (GitHub Actions, GitLab CI, etc.) |
| MCP clients | OAuth2 device flow or API key |

### Scoped Agent API Keys

Agent keys must be **minimally scoped** to limit blast radius:

```json
{
  "key_id": "sk-shark-...",
  "scopes": [
    "tasks:read",
    "tasks:write",
    "tasks:advance_status",
    "workflow:read"
  ],
  "bound_to": {
    "org_id": "acme-corp",
    "team_id": "backend-team",
    "project_id": "payment-service"
  },
  "agent_type": "developer",
  "expires_at": null
}
```

### RBAC Model

```
enterprise_admin  → full control over all orgs
org_admin         → org, team, and project management
team_lead         → team workflow configuration
developer         → create/update tasks in assigned projects
viewer            → read-only across assigned projects
agent             → restricted write (status transitions, context, notes only)
```

AI agents receive a dedicated `agent` role. They can advance workflow state and write context, but cannot delete entities, modify workflow definitions, or access billing and configuration.

### Audit Trail

Every agent action is recorded with full context:

```sql
CREATE TABLE audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL,
    actor_type      TEXT NOT NULL,  -- 'user', 'agent', 'system'
    actor_id        TEXT NOT NULL,  -- user_id or api_key_id
    actor_metadata  JSONB,          -- agent_type, model, session_id
    action          TEXT NOT NULL,  -- 'task.advance_status'
    resource_type   TEXT,
    resource_id     TEXT,
    before_state    JSONB,
    after_state     JSONB,
    timestamp       TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (timestamp);
```

Partitioned by month: recent data stays hot, old months archive to cold storage without impacting query performance.

---

## Layer 5: Multi-tenant Challenges

### Challenge 1: Workflow Config Bleed

**Problem**: If team A changes their workflow mid-sprint, does it affect in-flight tasks?

**Solution**: Workflow versions are pinned to tasks at creation time. Explicit migration tooling promotes in-flight tasks to a new version. Teams upgrade on their own schedule.

### Challenge 2: Cross-tenant Visibility

**Problem**: Portfolio dashboards need org-wide visibility, but raw task data must stay tenant-isolated.

**Solution**: Materialized aggregation tables pre-compute summaries (count by status, completion rate) at the org level, refreshed via triggers. Dashboards query these aggregates, never raw task rows. No cross-tenant access required.

### Challenge 3: AI Agent Context Isolation

**Problem**: Claude running in repo A must not be able to read tasks from repo B, even on the same developer machine.

**Solution**: Session-bound API keys. When a developer opens Claude Code in a project directory, the MCP server issues a short-lived token bound to that project's ID. The token expires at session end. Project ID is validated on every tool call server-side.

### Challenge 4: Rate Limiting

**Problem**: AI agents can be chatty — a single agent in a tight reasoning loop could hammer `task.get_resume` hundreds of times per minute.

**Solution**: Per-key rate limits enforced in Redis:
- `tasks:read` → 1,000/minute per key
- `tasks:write` → 100/minute per key
- `tasks:advance_status` → 20/minute per key (this should be rare and deliberate)

### Challenge 5: Regional Data Residency

**Problem**: EU organizations cannot have task data leave their region (GDPR, data sovereignty).

**Solution**: Regional Shark deployments with independent Postgres clusters. Routing by `org.region` at the load balancer. Immutable workflow definitions replicate globally; task data stays regional. The CLI and MCP clients are region-aware via config.

---

## Layer 6: Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Enterprise Shark                        │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  MCP Server  │    │  REST API    │    │  gRPC API    │  │
│  │  (Claude)    │    │  (CI/CD)     │    │  (internal)  │  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘  │
│         └───────────────────┼───────────────────┘          │
│                             │                               │
│                    ┌────────▼────────┐                      │
│                    │  Service Layer  │                      │
│                    │  (Go services)  │                      │
│                    └────────┬────────┘                      │
│                             │                               │
│              ┌──────────────┼──────────────┐               │
│              ▼              ▼              ▼               │
│         ┌─────────┐  ┌──────────┐  ┌──────────┐           │
│         │Postgres │  │  Redis   │  │   S3/    │           │
│         │(primary)│  │(cache,   │  │ Artifact │           │
│         │         │  │sessions, │  │  Store   │           │
│         │         │  │rate lim) │  │          │           │
│         └────┬────┘  └──────────┘  └──────────┘           │
│              │                                              │
│         ┌────▼────┐                                        │
│         │  Read   │                                        │
│         │Replicas │                                        │
│         └─────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

**Kubernetes deployment** with Helm charts. The MCP server, REST API, and gRPC API are separate deployments that share the same Go service layer binary, differentiated by startup flags.

---

## Before vs. After: What Changes

| Current Shark | Enterprise Shark MCP |
|---|---|
| SQLite, local file | PostgreSQL, centralized with RLS |
| JSON workflow files in repo | Versioned workflow definitions in DB |
| CLI as primary interface | MCP primary; CLI becomes a thin remote client |
| Single user, no auth | Multi-tenant RBAC, SSO + scoped API keys |
| No audit trail | Partitioned audit log, every action recorded |
| `make build` | Kubernetes + Helm, regional deployments |
| Config in `.sharkconfig.json` | Config in DB; repo contains only a project pointer |
| Workflow per-repo | Four-level hierarchy: enterprise → org → team → project |
| Local SQLite = microseconds | Network MCP calls = 10–100ms; caching required |

### What Stays the Same

The Go service layer (`internal/services/`) maps almost directly to the enterprise version. The architectural investment already made in clean service boundaries, repository interfaces, and dependency injection means the enterprise transition is primarily about:

1. Adding MCP and REST transport handlers alongside the existing Cobra CLI
2. Swapping the SQLite repository implementations for PostgreSQL equivalents
3. Adding tenant context propagation through the service layer
4. Layering auth middleware at the transport layer

The core domain logic (workflow transitions, status management, context tracking) does not change.

---

## Key Risks & Hard Problems

### 1. Workflow Migration Complexity
Changing a workflow definition that 50 projects depend on is a breaking change. Enterprise Shark needs semantic versioning for workflows, migration tooling for in-flight tasks, and a compatibility layer for tasks that span a version boundary.

### 2. Latency
Local SQLite is microseconds. Centralized network calls are 10–100ms. AI agents making multiple sequential MCP calls will feel this. Mitigation: aggressive read caching of task and workflow data, cache invalidation via PostgreSQL `LISTEN/NOTIFY`, and designing agent patterns to batch reads where possible.

### 3. Bootstrap Chicken-and-Egg
The MCP server must exist before teams adopt it. Early adoption requires the server to be as easy to start with as running local Shark. A hosted SaaS path (like a managed Shark Cloud) solves this for initial adoption before self-hosted enterprise deployment.

### 4. Workflow Governance Without Bureaucracy
Enterprise gates are essential for compliance and terrible for velocity if misapplied. The architecture should support a **shadow mode** for gates: they run, record results, and emit metrics, but do not block. Teams evaluate a gate's real-world impact before making it required. Required gates should be rare and justified.

### 5. AI Agent Accountability
When an agent advances a task status incorrectly or marks work complete prematurely, the audit log provides post-hoc traceability. Real-time guardrails are also needed: maximum status advances per session, human-in-the-loop approval gates for critical transitions (e.g., `before:completed` on high-priority tasks), and confidence metadata attached to agent state changes.

### 6. The CLI in an MCP World
The existing Shark CLI does not disappear — it becomes a **thin remote client** that speaks to the central MCP/REST API rather than a local database. Developers can still use `shark get E07-F01-001` from their terminal; it just makes an authenticated API call instead of a local SQLite query. The `--db` flag becomes `--server` or is configured via `.sharkconfig.json` with an API endpoint and key.

---

## Potential Epic Structure

If this moves to implementation, the natural epic breakdown:

| Epic | Scope |
|------|-------|
| E?? — PostgreSQL Migration | Schema migration, repository layer swap, connection pooling |
| E?? — Multi-tenancy & RLS | Org/team/project model, row-level security, tenant context |
| E?? — Authentication | SSO/OIDC integration, API key management, RBAC |
| E?? — MCP Server | MCP protocol implementation, tool/resource/prompt definitions |
| E?? — Workflow Hierarchy | Four-level inheritance, versioning, migration tooling |
| E?? — Enterprise Gates | Gate definitions, shadow mode, required gate enforcement |
| E?? — Audit & Observability | Partitioned audit log, metrics, OpenTelemetry (see E23) |
| E?? — CLI as Remote Client | CLI refactor to call API instead of local DB |
| E?? — Regional Deployment | Multi-region Kubernetes, data residency, routing |
