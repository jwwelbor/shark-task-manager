# Shark CLI Documentation

Complete command-line interface documentation for Shark Task Manager - a CLI for AI-driven task management and workflow orchestration.

## Quick Navigation

### Core Concepts

- **[Configuration](configuration.md)** - `.sharkconfig.json` reference and settings
- **[Workflow Configuration](workflow-configuration.md)** - Status flows, transitions, and orchestrator actions
- **[Template System](template-system.md)** - File-based templating (v2) for agent instructions

### Entity Commands

- **[Epic Commands](epic-cli.md)** - Epic lifecycle and management
- **[Feature Commands](feature-cli.md)** - Feature creation and tracking
- **[Task Commands](task-cli.md)** - Task operations and status transitions

### Workflows

- **[Example Workflows](workflow-examples/index.md)** - End-to-end lifecycle examples from epic creation through implementation

## What is Shark?

Shark is a task management CLI designed for AI agent orchestration. It provides:

- **Hierarchical task structure**: Epics → Features → Tasks
- **Workflow-driven status transitions**: Configurable status flows with validation
- **Agent routing**: Automatic agent type assignment based on status
- **Progress tracking**: Weighted and completion progress calculations
- **File-based templating**: Externalized agent instructions for flexibility

## Key Features

### 1. Three-Tier Entity Hierarchy

```
Epic (E07)
├── Feature (E07-F01)
│   ├── Task (T-E07-F01-001)
│   ├── Task (T-E07-F01-002)
│   └── Task (T-E07-F01-003)
└── Feature (E07-F02)
    └── ...
```

### 2. Workflow-Driven Development

Each entity type (epic, feature, task) has its own workflow configuration:
- Status flow (valid transitions)
- Status metadata (color, phase, progress weight, responsibility)
- Orchestrator actions (agent spawning, advancement, pausing)
- Special statuses (start, complete, aggregation)

### 3. AI Agent Orchestration

Statuses define which agent types should handle work:
- **researcher**: Codebase research and analysis
- **business-analyst**: Requirements and specification
- **architect**: Technical design and architecture
- **developer**: Implementation
- **tech-lead**: Code review
- **qa**: Testing and validation
- **product-manager**: Feature decomposition and coordination
- **tech-director**: Multi-feature orchestration

### 4. File-Based Templates

Agent instructions are stored as external `.tmpl` files:
- Located in `templates/` directory
- Referenced by status metadata: `"instruction_template": "task/ready_for_development.tmpl"`
- Support template variables: `{{.task_id}}`, `{{.title}}`, `{{.file_path}}`
- Include reusable partials: `{{template "_read_section" .}}`

## Quick Start

### Initialize Project

```bash
# Initialize Shark in your project
shark init --non-interactive

# Verify configuration
cat .sharkconfig.json
```

### Create an Epic

```bash
# Create epic
shark epic create --title="User Authentication System"

# View epic
shark epic get E01
```

### Create Features

```bash
# Create feature in epic
shark feature create E01 "JWT Token Management"

# List features
shark feature list E01
```

### Create and Work on Tasks

```bash
# Create task
shark task create E01 F01 "Implement token generation" --agent=backend

# Get next task
shark task next --agent=backend

# Start working
shark task start T-E01-F01-001

# Mark complete
shark task complete T-E01-F01-001 --notes="Token generation implemented"
```

## Documentation Structure

### Configuration Files

1. **[configuration.md](configuration.md)** - Complete `.sharkconfig.json` reference
   - Database configuration (local SQLite or Turso cloud)
   - Viewer settings (color, JSON output, interactive mode)
   - Epic/feature/task workflow definitions
   - Status metadata and flow

2. **[workflow-configuration.md](workflow-configuration.md)** - Workflow system deep dive
   - Status flow patterns
   - Status metadata fields
   - Orchestrator action types
   - Special status groups
   - Validation rules

3. **[template-system.md](template-system.md)** - Template system (v2)
   - Template directory structure
   - Template variables
   - Partial templates
   - Template functions
   - Best practices

### Entity CLI Commands

4. **[epic-cli.md](epic-cli.md)** - Epic commands
   - `epic create` - Create new epic
   - `epic list` - List all epics
   - `epic get` - Get epic details
   - `epic update` - Update epic fields
   - `epic next-status` - Advance epic status
   - Progress calculation and aggregation

5. **[feature-cli.md](feature-cli.md)** - Feature commands
   - `feature create` - Create feature in epic
   - `feature list` - List features (filtered by epic)
   - `feature get` - Get feature details
   - `feature update` - Update feature fields
   - `feature next-status` - Advance feature status
   - Complexity triage and routing

6. **[task-cli.md](task-cli.md)** - Task commands
   - `task create` - Create task in feature
   - `task list` - List tasks (filtered by epic/feature/status)
   - `task get` - Get task details
   - `task next` - Get next available task
   - `task start` - Start working on task
   - `task complete` - Mark task complete
   - `task approve` - Approve completed task
   - `task block/unblock` - Manage blockers
   - Status transitions and cascading

### Workflow Examples

7. **[workflow-examples/index.md](workflow-examples/index.md)** - Real-world workflows
   - Epic creation through feature delivery
   - Simple feature workflow (SIMPLE tier)
   - Standard feature workflow (STANDARD tier)
   - Complex feature workflow (COMPLEX tier)
   - Agent coordination patterns
   - Status transition examples

## Command Conventions

### Key Formats

Shark supports flexible key formats (all case-insensitive):

**Epics:**
- Numeric: `E07`
- Slugged: `E07-user-authentication`

**Features:**
- Numeric: `F01` or `E07-F01`
- Slugged: `F01-jwt-tokens` or `E07-F01-jwt-tokens`

**Tasks:**
- Short: `E07-F01-001`
- Traditional: `T-E07-F01-001`
- Slugged: `E07-F01-001-implement-jwt` or `T-E07-F01-001-implement-jwt`

### Common Flags

All commands support:
- `--json` - Machine-readable JSON output
- `--verbose` / `-v` - Debug logging
- `--config` - Custom config file path

Entity commands support:
- `--file` - Custom file path (relative to project root)
- `--force` - Force reassignment of claimed files

### Output Modes

**Human-readable** (default):
- Colored terminal output
- Table formatting
- Progress bars

**JSON** (`--json`):
- Machine-parseable
- Complete field exposure
- Suitable for AI agent processing

## Project Structure

```
.
├── .sharkconfig.json              # Main configuration
├── shark-tasks.db                 # SQLite database
├── templates/                     # External template files
│   ├── epic/                      # Epic status templates
│   ├── feature/                   # Feature status templates
│   ├── task/                      # Task status templates
│   └── partials/                  # Reusable template partials
└── docs/
    └── plan/                      # Generated documentation
        └── E##-epic-name/         # Epic directory
            ├── epic.md            # Epic PRD
            └── E##-F##-feature-name/  # Feature directory
                ├── feature.md     # Feature description
                ├── prd.md         # Feature PRD (STANDARD/COMPLEX)
                ├── 0X-*.md        # Architecture docs
                └── tasks/         # Task files
                    └── T-E##-F##-###.md
```

## Related Documentation

- **[CLAUDE.md](../CLAUDE.md)** - Development guidelines for AI agents
- **[README.md](../README.md)** - Project overview and quick start
- **Legacy Docs**:
  - [CLI_REFERENCE.md](../CLI_REFERENCE.md) - Original CLI reference
  - [WORKFLOW_GUIDE.md](../WORKFLOW_GUIDE.md) - Original workflow guide
  - [cli-reference/](../cli-reference/) - Detailed command reference

## Version History

- **v2.0** - File-based templating system, complexity-adaptive workflows
- **v1.1** - Feature triage and routing, agent type assignment
- **v1.0** - Core task management, status flows, database schema

## Support

- **Issues**: https://github.com/jwwelbor/shark-task-manager/issues
- **Documentation**: This directory (`docs/cli/`)
