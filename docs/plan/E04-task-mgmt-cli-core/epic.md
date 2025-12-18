# Epic: Task Management CLI - Core Functionality

**Epic Key**: E04-task-mgmt-cli-core
**Created**: 2025-12-14
**Status**: Draft
**Business Value**: High

---

## Goal

### Problem

AI agents working on multi-epic software projects in the Claude Code environment face significant inefficiencies in task discovery and state management. Currently, project status is scattered across multiple markdown files in various folders (active/, completed/, PRPs/), with no single source of truth for task availability, dependencies, or completion state. Agents must read 20+ markdown files to answer basic questions like "what should I work on next?" or "is Feature X complete?". This leads to wasted tokens (up to 50K per planning session), stale status information, manual folder management, and frequent context synchronization failures between agent sessions. The lack of structured task querying means agents cannot reliably determine task dependencies, detect blocking issues, or hand off work between sessions.

### Solution

Implement the foundational infrastructure for a SQLite-backed task management CLI (`shark` - Project Manager) that serves as the single source of truth for all project state. This epic focuses on the core capabilities: database schema, CLI framework, task lifecycle operations, epic/feature queries, file path management, task creation, and initialization/sync tools. The system combines a normalized relational database (project.db) for task status and metadata with feature-based file organization (docs/plan/{epic}/{feature}/tasks/) that maintains context locality.

This epic establishes the foundation that enables E05 (Task Management CLI - Extended Capabilities) to add advanced features like dependency management, status dashboards, and audit trails.

### Impact

- **Token Efficiency**: Reduce token usage for status lookups by 80% (from ~50K to ~10K per planning session) by eliminating the need to read multiple markdown files for state determination
- **Agent Productivity**: Decrease agent "planning overhead" from 2-3 minutes per session to <10 seconds by providing instant task availability queries
- **Status Accuracy**: Achieve 100% status consistency by maintaining a single source of truth with atomic updates
- **Developer Experience**: Enable human developers to query project state programmatically instead of manually checking 10+ folders and files
- **Foundation for Advanced Features**: Provide the infrastructure needed for E05's dashboard, dependency tracking, and audit capabilities

---

## User Personas

### Primary Persona: AI Agent (Claude Code Agents)

**Role**: Autonomous code generation and task execution agent
**Environment**: Claude Code CLI, limited context window (200K tokens), stateless between sessions

**Key Characteristics**:
- Must make decisions based on available context without human intervention
- Cannot remember previous sessions without explicit state storage
- Optimizes for token efficiency when reading project state
- Requires structured, parseable output formats (JSON preferred)
- Needs clear task boundaries and success criteria
- Cannot manually navigate folder structures or perform complex file searches efficiently

**Goals**:
- Quickly identify next available task matching agent specialty (frontend, backend, testing)
- Update task status atomically and reliably
- Report progress to human developers
- Create new tasks programmatically

**Pain Points**:
- Current system requires reading 20+ markdown files to find available tasks
- Manual file movement between folders is error-prone
- Status inconsistencies between folder location and markdown frontmatter
- No structured way to create tasks programmatically

### Secondary Persona: Product Manager / Technical Lead

**Role**: Human developer managing multi-epic projects with AI agent assistance
**Environment**: Terminal-based workflow, Git for version control, managing 2-5 concurrent epics

**Key Characteristics**:
- Oversees project planning and prioritization
- Reviews agent-generated code and task completion
- Needs quick visibility into project health and progress
- Works both directly in code and through agent delegation
- Comfortable with CLI tools and terminal interfaces

**Goals**:
- Query project state programmatically (epic/feature/task lists)
- Track progress across multiple epics and features
- Initialize new projects with proper structure
- Sync existing markdown files into the database
- Create tasks efficiently via CLI

**Pain Points**:
- No easy way to query "all tasks in Epic X"
- Manual folder creation and maintenance is tedious
- Cannot efficiently filter tasks by status, epic, or agent type
- Importing existing markdown task files is manual and error-prone

---

## Features

This epic includes the following features:

1. **F01: Database Schema & Core Data Model** - SQLite database structure with epics, features, tasks, and task_history tables
2. **F02: CLI Infrastructure & Framework** - Go CLI framework with command routing, output formatting, and configuration
3. **F03: Task Lifecycle Operations** - Core task commands (list, get, next, start, complete, approve, block, unblock, reopen)
4. **F04: Epic & Feature Queries** - Commands for querying epics and features with automatic progress calculation
5. **F05: File Path Management** - Feature-based file organization with database-tracked paths (no status-based folder movement)
6. **F06: Task Creation & Templating** - Task creation with automatic key generation and template-based file generation
7. **F07: Initialization & Synchronization** - Database initialization, folder setup, and markdown file import/sync
8. **F08: Distribution & Release Automation** - GoReleaser configuration for multi-platform binary distribution via Homebrew, Scoop, and GitHub releases

---

## Out of Scope

### Deferred to E05 (Task Management CLI - Extended Capabilities)

1. **Status Dashboard & Reporting** - The `pm status` command with progress bars, active tasks, and recent activity (E05-F01)
2. **Dependency Management** - Dependency tree visualization, circular dependency detection, and automatic blocking (E05-F02)
3. **History & Audit Trail** - Detailed activity logs, history commands, and export capabilities (E05-F03)
4. **Advanced Features** - Estimation, agent metrics, batch operations, custom views (E05-F04+)

### Explicitly Excluded from E04

1. **Web-based Dashboard** - All interfaces are CLI-only
2. **Multi-User Collaboration** - Single-developer focus with AI agent assistance
3. **Cloud Synchronization** - Strictly local database and files
4. **External Tool Integration** - No Jira, Linear, or GitHub Issues integration
5. **Time Tracking & Timers** - No Pomodoro or elapsed time measurement
6. **MCP Server Implementation** - While designed to be MCP-compatible, server wrapper is out of scope
7. **Deep Subtask Hierarchy** - Only Epic → Feature → Task (three levels)

---

## Success Metrics

### 1. Token Efficiency (Leading Indicator)
**What**: Average tokens consumed per agent planning session for task queries
**Baseline**: 50,000 tokens per planning session (reading 20+ markdown files)
**Target**: 10,000 tokens per planning session (80% reduction)
**Timeline**: Measure over 30-day period post-launch with 50+ agent sessions

### 2. Agent Task Discovery Time (Leading Indicator)
**What**: Time elapsed from "find next task" request to task start
**Baseline**: 120 seconds average
**Target**: <10 seconds average (90% reduction)
**Timeline**: Measure first 100 agent task starts post-launch

### 3. File Path Consistency Rate (Leading Indicator)
**What**: Percentage of tasks where database file_path matches actual file location
**Baseline**: ~70% consistency (current manual tracking)
**Target**: 100% consistency
**Timeline**: Weekly validation checks over 90-day period

### 4. CLI Command Success Rate (Leading Indicator)
**What**: Percentage of CLI commands that complete successfully without errors
**Target**: >98% success rate for valid operations
**Timeline**: Track all command executions during first 60 days

### 5. Database Query Performance (Leading Indicator)
**What**: Average query response time for task listing and filtering
**Target**: <100ms for datasets up to 1,000 tasks
**Timeline**: Performance benchmarks during testing and first 30 days of use

---

## Non-Functional Requirements

### Performance
- Database queries must return in <100ms for datasets up to 10,000 tasks
- Task status updates must complete atomically in <20ms (database-only, no file movement)
- CLI startup time must be <50ms (single compiled binary)
- Full project sync must process 100 existing files in <5 seconds

### Security & Data Integrity
- Database transactions must be ACID-compliant
- Input validation must prevent SQL injection (parameterized queries only)
- Status transitions must be validated against allowed state transitions
- Referential integrity enforced at database level
- File paths must be validated to prevent directory traversal attacks

### Accessibility & Usability
- Comprehensive CLI help text for all commands
- Actionable error messages
- Examples in help text for complex commands
- Colorized output with --no-color flag for agent compatibility
- Valid JSON output with documented schema

### Reliability
- Database corruption detection on startup
- Graceful degradation if database is locked
- Atomic database transactions with proper rollback
- Validation command to detect and repair file path inconsistencies

### Compatibility
- Go 1.21+ for development
- Cross-platform binaries: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- SQLite 3.35+ (embedded via go-sqlite3 or modernc.org/sqlite)
- Single binary distribution with no external dependencies
- Git-friendly: markdown files are authoritative

### Maintainability
- >80% unit test coverage (using Go's testing package)
- Integration tests for all CLI commands
- Go's strong typing throughout
- Godoc comments for all exported functions and types
- Structured logging for state-changing operations (using slog or zerolog)
- Database migration support (using golang-migrate or similar)

---

## Business Value

**Rating**: High

**Justification**:

This epic delivers the foundational infrastructure that unlocks AI-assisted multi-epic project management. By implementing the core database, CLI framework, and task operations, it removes the primary bottleneck preventing agents from efficiently coordinating across sessions: the lack of structured, queryable project state.

**Direct Impact**:
- Reduces agent planning overhead by 80-90%, saving ~100 seconds per session
- Eliminates manual folder management and status tracking for developers
- Provides programmatic access to project state via CLI and JSON output

**Strategic Value**:
- Foundational infrastructure for E05 advanced capabilities (dashboards, dependencies, audit)
- Enables future autonomous agent workflows (multi-agent collaboration, automated planning)
- Differentiates Claude Code from competitors (GitHub Copilot, Cursor) with structured project management

**Risk Mitigation**:
- Replaces error-prone manual status tracking with database-backed single source of truth
- Prevents data loss and inconsistencies as projects scale
- Provides atomic operations and audit trails for reliability

The combination of immediate productivity gains, developer experience improvements, and strategic positioning makes this a high-value investment with clear ROI and strong foundation for future capabilities.
