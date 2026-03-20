# Package Dependencies

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## Internal Package Dependency Graph

```mermaid
graph TD
    %% Entry points
    CMD_SHARK["cmd/shark"] --> CLI_CMDS["cli/commands"]
    CMD_SERVER["cmd/server"] --> CLI["cli"]
    CMD_DEMO["cmd/demo"] --> CLI

    %% CLI layer
    CLI_CMDS --> CLI
    CLI_CMDS --> SERVICES["services"]
    CLI_CMDS --> MODELS["models"]
    CLI_CMDS --> REPO["repository"]
    CLI_CMDS --> FORMATTERS["formatters"]

    %% CLI framework
    CLI --> SERVICES
    CLI --> REPO
    CLI --> WORKFLOW["workflow"]
    CLI --> CONFIG["config"]
    CLI --> DB["db"]
    CLI --> PATHRESOLVER["pathresolver"]

    %% Service layer
    SERVICES --> MODELS
    SERVICES --> REPO
    SERVICES --> WORKFLOW
    SERVICES --> CONFIG
    SERVICES --> TASKCREATION["taskcreation"]
    SERVICES --> FILEOPS["fileops"]

    %% Repository layer
    REPO --> MODELS
    REPO --> DB

    %% Infrastructure
    WORKFLOW --> CONFIG
    CONFIG --> MODELS
    STATUS["status"] --> MODELS
    STATUS --> REPO
    STATUS --> CONFIG

    %% Initialization
    INIT["init"] --> CONFIG
    INIT --> DB
    INIT --> PROFILES["init/profiles"]

    %% Discovery
    DISCOVERY["discovery"] --> MODELS
    DISCOVERY --> PATTERNS["patterns"]
    DISCOVERY --> PARSER["parser"]

    %% Utilities
    TASKCREATION --> MODELS
    TASKCREATION --> KEYGEN["keygen"]
    TASKCREATION --> SLUG["slug"]
    TASKCREATION --> TEMPLATES["templates"]
    TASKFILE["taskfile"] --> MODELS
    TASKFILE --> PARSER
    VALIDATION["validation"] --> MODELS
    VALIDATION --> WORKFLOW
    REPORTING["reporting"] --> MODELS
    REPORTING --> REPO
    FORMATTERS --> MODELS
    KEYS["keys"] --> MODELS

    %% Embedded templates
    EMBEDDED["embedded.go"] --> TEMPLATES

    %% Styling
    classDef entry fill:#e1f5fe,stroke:#01579b
    classDef core fill:#fff3e0,stroke:#e65100
    classDef infra fill:#f3e5f5,stroke:#4a148c
    classDef util fill:#e8f5e9,stroke:#1b5e20

    class CMD_SHARK,CMD_SERVER,CMD_DEMO entry
    class CLI,CLI_CMDS,SERVICES,REPO,MODELS core
    class DB,WORKFLOW,CONFIG,STATUS,INIT,PROFILES infra
    class TASKCREATION,KEYGEN,SLUG,TEMPLATES,TASKFILE,PARSER,PATTERNS,FILEOPS,VALIDATION,FORMATTERS,REPORTING,KEYS,PATHRESOLVER,DISCOVERY,EMBEDDED util
```

## Layer Isolation Analysis

| Layer | Can Import | Must Not Import |
|-------|-----------|-----------------|
| **cmd/** | cli, services, repository, models | (top-level, can import anything) |
| **cli/commands** | cli, services, models, repository*, config | db directly |
| **cli (framework)** | services, repository, workflow, config, db | commands |
| **services** | models, repository interfaces, workflow, config | cli, db directly |
| **repository** | models, db | services, cli, workflow |
| **models** | stdlib only | everything else |
| **workflow** | config | services, repository, models |
| **db** | stdlib, go-sqlite3 | models, services, repository |

*Note: CLI commands importing repository directly is the legacy "fat controller" anti-pattern being refactored in E15.

## Dependency Depth

```
cmd/shark (depth 0)
  └── cli/commands (depth 1)
        ├── cli (depth 2)
        │     ├── services (depth 3)
        │     │     ├── models (depth 4 — leaf)
        │     │     ├── repository interfaces (depth 4)
        │     │     └── workflow (depth 4)
        │     │           └── config (depth 5)
        │     ├── repository (depth 3)
        │     │     ├── models (depth 4 — leaf)
        │     │     └── db (depth 4 — leaf)
        │     └── workflow (depth 3)
        ├── services (depth 2 — shared)
        └── models (depth 2 — shared)
```

Maximum dependency depth: **5** (cmd → cli → services → workflow → config)

---

See also: [Component Diagram](component-diagram.md) | [Dependencies](../architecture/dependencies.md)
