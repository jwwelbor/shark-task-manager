# Package Dependencies

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## Internal Package Dependency Graph

```mermaid
graph TD
    CMD_SHARK["cmd/shark"] --> CLI_CMDS["cli/commands"]
    CMD_SERVER["cmd/server"] --> SERVICES
    CMD_SHARK --> CLI

    CLI_CMDS --> CLI["cli"]
    CLI_CMDS --> SERVICES["services"]
    CLI_CMDS --> MODELS["models"]
    CLI_CMDS --> FORMATTERS["formatters"]

    CLI --> REPOSITORY["repository"]
    CLI --> WORKFLOW["workflow"]
    CLI --> CONFIG["config"]
    CLI --> PATHRESOLVER["pathresolver"]

    SERVICES --> REPOSITORY
    SERVICES --> WORKFLOW
    SERVICES --> MODELS
    SERVICES --> TASKCREATION["taskcreation"]
    SERVICES --> STATUS["status"]
    SERVICES --> PROGRESS["progress"]

    REPOSITORY --> DB["db"]
    REPOSITORY --> MODELS
    REPOSITORY --> SLUG["slug"]

    DB --> CONFIG

    TASKCREATION --> TEMPLATES["templates"]
    TASKCREATION --> FILEOPS["fileops"]
    TASKCREATION --> KEYGEN["keygen"]
    TASKCREATION --> SLUG

    WORKFLOW --> CONFIG
    STATUS --> CONFIG
    STATUS --> REPOSITORY

    DISCOVERY["discovery"] --> PARSER["parser"]
    DISCOVERY --> PATTERNS["patterns"]
    DISCOVERY --> MODELS

    SYNC["sync"] --> DISCOVERY
    SYNC --> REPOSITORY

    INIT["init"] --> CONFIG
    INIT --> DB
    INIT --> TEMPLATES

    VALIDATION["validation"] --> MODELS
    VALIDATION --> PATTERNS

    KEYS["keys"] --> PATTERNS

    REPORTING["reporting"] --> SERVICES
    REPORTING --> FORMATTERS

    RUNNER["runner"] --> SERVICES
    RUNNER --> WORKFLOW
```

## External Dependency Graph

```mermaid
graph TD
    SHARK["shark-task-manager"] --> COBRA["spf13/cobra v1.10.2"]
    SHARK --> VIPER["spf13/viper v1.21.0"]
    SHARK --> SQLITE["mattn/go-sqlite3 v1.14.32"]
    SHARK --> TURSO["tursodatabase/libsql-client-go"]
    SHARK --> PTERM["pterm v0.12.82"]
    SHARK --> TESTIFY["stretchr/testify v1.11.1"]
    SHARK --> YAML["gopkg.in/yaml.v3"]
    SHARK --> XTERM["golang.org/x/term"]
    SHARK --> XTEXT["golang.org/x/text"]

    COBRA --> PFLAG["spf13/pflag"]
    COBRA --> MOUSETRAP["mousetrap"]
    VIPER --> AFERO["spf13/afero"]
    VIPER --> CAST["spf13/cast"]
    VIPER --> FSNOTIFY["fsnotify"]
    VIPER --> MAPSTRUCTURE["mapstructure/v2"]
    VIPER --> TOML["go-toml/v2"]
    VIPER --> GOTENV["gotenv"]
    TURSO --> ANTLR["antlr4-go"]
    TURSO --> WEBSOCKET["coder/websocket"]
    PTERM --> CURSOR["atomicgo.dev/cursor"]
    PTERM --> KEYBOARD["atomicgo.dev/keyboard"]
    PTERM --> COLOR["gookit/color"]
    PTERM --> RUNEWIDTH["go-runewidth"]
    TESTIFY --> SPEW["go-spew"]
    TESTIFY --> DIFFLIB["go-difflib"]
```

See also: [Component Diagram](component-diagram.md) | [Dependencies](../architecture/dependencies.md)
