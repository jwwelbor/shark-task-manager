# Scope Boundaries

**Epic**: [Shark Templates](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in the Shark Templates epic (E20) and documents alternative approaches that were considered and rejected.

---

## Out of Scope

### Explicitly Excluded Features

**1. Template Marketplace or Registry**
- **Why It's Out of Scope**: A shared registry for downloading and publishing workflow templates adds significant infrastructure complexity (hosting, versioning, authentication) that is not justified by current user demand.
- **Future Consideration**: The externalized workflow file format established by this epic is a prerequisite for any future marketplace. Once the file format is stable and adopted, a registry can be built on top without further format changes.
- **Workaround**: Users can manually share `.sharkworkflow.json` files by copying them between projects.

**2. Per-Entity-Instance Workflow Overrides**
- **Why It's Out of Scope**: Allowing individual epics or features to override their workflow (e.g., "this specific epic uses a custom status flow") would require per-entity config resolution logic that adds substantial complexity to the config layer and workflow service.
- **Future Consideration**: Could be addressed by adding an optional `workflow_override` field to entity metadata in the database, resolved at runtime. The consistent block structure from this epic would make such overrides easier to implement.
- **Workaround**: Use the advanced workflow profile with its full status set; most custom needs are covered by the existing status flow.

**3. GUI or Web-Based Config Editor**
- **Why It's Out of Scope**: Shark is a CLI-first tool consumed primarily by AI agents and developers who edit JSON files directly. A visual editor adds frontend complexity with limited value for the current user base.
- **Future Consideration**: Low priority. If Shark gains a web dashboard in a future epic, a config editor could be added there.
- **Workaround**: Developers edit `.sharkworkflow.json` directly. The `shark config validate` command catches syntax and structural errors.

**4. YAML or TOML File Format Support**
- **Why It's Out of Scope**: Supporting multiple config file formats adds parsing complexity and format negotiation logic. JSON is already the standard for `.sharkconfig.json` and the Go standard library provides first-class JSON support.
- **Future Consideration**: Could be added if user demand materializes, but JSON covers all current needs.
- **Workaround**: Use JSON. The Go `encoding/json` package handles all serialization/deserialization needs.

**5. Automatic Removal of Inline Workflow Data from `.sharkconfig.json`**
- **Why It's Out of Scope**: Automatically deleting workflow blocks from `.sharkconfig.json` after generating `.sharkworkflow.json` risks data loss if the generation is incomplete or the user has custom modifications. The fallback behavior makes coexistence safe.
- **Future Consideration**: A future `shark admin workflow migrate --clean` flag could offer opt-in cleanup with explicit user consent.
- **Workaround**: The precedence chain (workflow file > inline) means inline data is harmless when the workflow file exists. Users can manually remove inline blocks at their convenience.

---

### Edge Cases & Scenarios Not Covered

**1. Conflicting Workflow Definitions Across Files**
- **Impact**: Low. The precedence chain (workflow file wins) resolves conflicts deterministically.
- **Rationale**: Adding merge logic for conflicting definitions would be error-prone and confusing. A simple precedence chain is predictable.
- **Mitigation**: `shark config validate` will warn when the same entity workflow is defined in both files.

**2. Workflow File on a Network or Remote Path**
- **Impact**: Very low. No current use case requires workflow files on remote filesystems.
- **Rationale**: Adding network path support (HTTP, S3, etc.) is a separate concern with its own error handling, caching, and authentication needs.
- **Mitigation**: The `workflow_config` key supports absolute local paths. Users with network-mounted filesystems can use mount points as absolute paths.

**3. Hot-Reloading of Workflow File Changes**
- **Impact**: Low. Shark CLI commands are short-lived processes that reload config on each invocation.
- **Rationale**: Hot-reloading would only matter for a long-running server process, which is not the primary use case.
- **Mitigation**: Each CLI invocation loads the latest config files. The HTTP server (if used) would need a restart to pick up config changes, which is acceptable for configuration that changes infrequently.

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Embed Workflow Config in the Template Directory**
- **Description**: Store workflow definitions as files within `shark-templates/` (e.g., `shark-templates/workflows/task.json`) rather than in a single root-level file.
- **Pros**: Keeps all template-related configuration together. Natural place for workflow data that drives template rendering.
- **Cons**: The `shark-templates/` directory is designed for Go template files (`.tmpl`, `.md`), not JSON config. Mixing concerns would complicate the template discovery code. A single file is simpler to validate, backup, and share.
- **Decision Rationale**: A single root-level file matches the existing pattern (`.sharkconfig.json` in project root) and keeps configuration discovery simple: check project root for config files.

**Alternative 2: Use Environment Variables for Workflow Selection**
- **Description**: Instead of a workflow file, use environment variables (e.g., `SHARK_WORKFLOW_PROFILE=advanced`) to select pre-built workflow configurations compiled into the binary.
- **Pros**: No file management. Simple switching between profiles.
- **Cons**: Cannot customize individual status flows or metadata. Limits users to pre-built profiles. Does not address the task workflow inconsistency.
- **Decision Rationale**: Environment variables are appropriate for deployment-time settings (like `SHARK_DB_BACKEND`), not for detailed workflow configuration that users need to inspect and modify.

**Alternative 3: Keep Everything in `.sharkconfig.json` but Restructure**
- **Description**: Add a `task_workflow` block to `.sharkconfig.json` (replacing legacy keys) without introducing a separate file.
- **Pros**: Simpler implementation. Single file to manage.
- **Cons**: Does not solve the core problem: `.sharkconfig.json` remains a 1,700+ line file mixing runtime settings with workflow definitions. Does not enable future template sharing or workflow presets.
- **Decision Rationale**: The task workflow standardization (REQ-F-005) is implemented regardless. The separate file addresses the larger architectural concern of configuration separation.

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| Workflow presets (ship multiple profiles as `.sharkworkflow.json` files) | Medium | Requires stable workflow file format from E20 |
| Template set sharing (import/export template directories with workflow config) | Low | Requires E20 + template directory improvements |
| Per-entity workflow overrides (epic-level custom status flows) | Low | Requires E20 consistent block structure |
| `shark admin workflow diff` (compare two workflow files) | Low | Requires E20 workflow file format |

---

*See also*: [Requirements](./requirements.md)
