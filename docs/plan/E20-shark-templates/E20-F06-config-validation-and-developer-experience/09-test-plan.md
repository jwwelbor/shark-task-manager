# E20-F06 Test Plan: Config Validation and Developer Experience

## AC Test Matrix

### Story 1: Config Validate (REQ-F-010)

| ID | Test Case | Input | Expected Output | AC |
|----|-----------|-------|-----------------|-----|
| V01 | Validate valid .sharkworkflow.json | Well-formed workflow file with all entities | "Configuration file is valid" + per-entity source listing | AC1 |
| V02 | Validate .sharkworkflow.json with JSON syntax error | File with missing comma | Error with file name and byte offset | AC1 |
| V03 | Validate missing .sharkworkflow.json | No workflow file present | Validation passes (only .sharkconfig.json checked) | AC1 |
| V04 | Report entity workflow sources | epic in workflow file, task in .sharkconfig.json | Sources listed per entity type | AC2 |
| V05 | Warn duplicate definitions | task_workflow in both files | Warning about duplicate definition | AC3 |
| V06 | Report missing required sub-keys | Workflow block with status_flow but no status_metadata | Warning about missing status_metadata | AC4 |
| V07 | JSON output mode | --json flag | Structured JSON with validation results | AC1-4 |

### Story 2: Config Show Source (REQ-F-011)

| ID | Test Case | Input | Expected Output | AC |
|----|-----------|-------|-----------------|-----|
| S01 | Human output shows source | Mixed sources | "epic_workflow: .sharkworkflow.json", "task_workflow: .sharkconfig.json" | AC1 |
| S02 | JSON output includes _source | --json flag | `_source` field per workflow block | AC2 |
| S03 | Default source display | No workflow file | All show ".sharkconfig.json" or "default" | AC1-2 |

### Story 3: Deprecation Warnings (REQ-F-020)

| ID | Test Case | Input | Expected Output | AC |
|----|-----------|-------|-----------------|-----|
| D01 | Warning on legacy coexistence | Legacy status_flow + task_workflow block | Deprecation warning to stderr | AC1 |
| D02 | Warning emitted once | Multiple commands | Single warning per invocation | AC1 |
| D03 | Guidance in warning | Legacy keys present | Warning includes "Migrate task workflow to task_workflow block" | AC2 |
| D04 | Suppressed in JSON mode | --json flag + legacy keys | No warning in stdout or stderr | AC3 |
| D05 | No warning when clean | Only task_workflow block, no legacy keys | No warning | AC1 |

### Story 4: Workflow Export (REQ-F-021) -- DESCOPED

> **DESCOPED**: Story 4 and REQ-F-021 were removed from scope during implementation. The test cases below are retained for reference only and are not expected to be implemented.

| ID | Test Case | Input | Expected Output | AC |
|----|-----------|-------|-----------------|-----|
| ~~X01~~ | ~~Export all entity workflows~~ | ~~Inline config with all entities~~ | ~~.sharkworkflow.json with 5 entity blocks~~ | ~~AC1-2~~ |
| ~~X02~~ | ~~Legacy task key conversion~~ | ~~Legacy top-level status_flow~~ | ~~task_workflow block in output~~ | ~~AC3~~ |
| ~~X03~~ | ~~Dry run preview~~ | ~~--dry-run flag~~ | ~~JSON preview to stdout, no file written~~ | ~~AC4~~ |
| ~~X04~~ | ~~Export with mixed sources~~ | ~~Some entities in workflow file, some inline~~ | ~~Complete .sharkworkflow.json~~ | ~~AC2~~ |
| ~~X05~~ | ~~Export when no inline data~~ | ~~Empty .sharkconfig.json~~ | ~~Error or default workflow exported~~ | ~~AC1~~ |

## Component Test Strategy

### workflow_parser.go (Source Tracking)
- Test `LoadMultiLevelWorkflow` populates `Sources` map correctly
- Test source precedence: workflow file > inline > default
- Test detection of legacy key coexistence

### config.go (Validate/Show Extensions)
- Test with temp directory containing mock config files
- Test human and JSON output formats
- Test edge cases: missing files, empty files, malformed JSON

### workflow.go (Export Command) -- DESCOPED
~~- Test export produces valid JSON~~
~~- Test legacy-to-block conversion~~
~~- Test dry-run does not write file~~
~~- Test backup creation when overwriting existing file~~

> **DESCOPED**: Export command tests are not applicable.

## Integration Scenarios

- Validate -> Fix -> Validate again (iterative validation loop)
- ~~Export -> Validate exported file (roundtrip)~~ **(DESCOPED)**
- ~~Show source before and after export (source changes from inline to file)~~ **(DESCOPED)**

## TDD Approach

For each story, write tests first following existing patterns in:
- `internal/config/workflow_file_loading_test.go` (temp dir test fixtures)
- `internal/config/workflow_test.go` (workflow config parsing)
