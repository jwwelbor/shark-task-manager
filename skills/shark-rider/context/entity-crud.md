# Entity CRUD Operations

## Creating Entities

Use verb-first syntax with `shark create`:

**Post-create rule:** whenever `shark create` generates a placeholder file for
the new entity, update that file immediately with the context already available.
Do not leave placeholder content behind. Fill in the investigation, scope, or
breadcrumbs you have at creation time, but do not expand the work into a full
research or specification pass unless the current workflow explicitly calls for it.

### Registering an already-authored file (preferred for filesystem-first docs)

When the entity's doc already exists on disk, register it directly — `--file`
links the existing file instead of generating a placeholder (the CLI prints
`LINKED TO EXISTING FILE`). Epics, features, and tasks all support `--file`
(+ `--key` to pin the key):

```bash
shark create epic "Epic Title" --key=E16 --file="docs/plan/E16-slug/epic.md"
shark create feature E16 "Feature Title" --key=F01 --file="docs/plan/E16-slug/E16-F01-slug/feature.md"
shark create task E16-F01 "Task Title" --file="docs/plan/E16-slug/E16-F01-slug/tasks/T-E16-F01-001.md"
```

Verify afterward: `shark get <KEY> --field status`.

**Never reach for a filesystem sync to register a single new entity.** A sync
sweeps a whole tree and can touch entities under active leases or review;
`create --file` is additive-only and safe while other agents work against live
shark state. Sync flows are for genuine bulk drift reconciliation, run at a
quiet moment on explicit user request only.

### Epics

```bash
shark create epic "Epic Title"
shark create epic "Epic Title" --priority=5 --business-value=8
```

### Features

```bash
shark create feature E01 "Feature Title"
shark create feature E01 "Feature Title" --order=1
```

### Tasks

```bash
shark create task E01 F02 "Task Title"
shark create task E01 F02 "Task Title" --order=1 --agent=backend --priority=5
shark create task E01-F02 "Task Title"    # Combined key format also works
```

**Task create flags:**
- `--order=N` - Execution order within feature
- `--agent=TYPE` - Agent type (backend, frontend, qa, etc.)
- `--priority=N` - Priority 1-10
- `--depends-on=KEY` - Dependency on another task
- `--file=PATH` - Custom file path
- `--force` - Overwrite existing file

### Bugs

```bash
shark create bug "Login crashes on Safari"
shark create bug "Payment fails" --severity=critical --description="Card declined unexpectedly"
shark create bug "Slow query" --severity=low --linked-type=feature --linked-key=E01-F02
```

**Bug create flags:**
- `--severity=LEVEL` - critical, high, medium (default), low
- `--description=TEXT` - Bug description
- `--linked-type=TYPE` - Link to epic, feature, or task
- `--linked-key=KEY` - Linked entity key (requires --linked-type)

### Change Cards

```bash
shark create change "Migrate auth to OAuth2"
shark create change "Update dependencies" --justification="Security patches"
shark create change "Refactor DB layer" --requested-by="Product Team" --epic=E01
```

**Change create flags:**
- `--description=TEXT` - Change description
- `--justification=TEXT` - Business justification
- `--requested-by=NAME` - Requesting team or person
- `--epic=KEY` - Link to epic
- `--feature=KEY` - Link to feature

### Ideas

```bash
shark create idea "New feature idea" --description="Description here"
shark create idea "Backend optimization" --priority=8
shark idea promote I-2026-01-15-01 --epic=E01   # Promote idea to entity
```

**Idea create flags:**
- `--description=TEXT` - Idea description
- `--priority=N` - Priority 1-10
- `--notes=TEXT` - Additional notes
- `--status=STATUS` - Initial status (new, on_hold, converted, archived)

### Notes

Add a typed note to any entity. Entity type is auto-detected from key format.

```bash
shark create note E01 "Kicked off Q1 planning"
shark create note E01-F02 "Decided to use JWT" --type=decision
shark create note E01-F02-001 "Waiting on API spec" --type=blocker --created-by=alice
shark create note B001 "Reproduced on Safari 17.2" --type=comment
shark create note CC-001 "Approved by security team" --type=comment
```

**Note create flags:**
- `--type=TYPE` - Note type (default: comment). Options: comment, decision, blocker, solution, reference, implementation, testing, future, question, rejection, requirement, review
- `--created-by=NAME` - Author name (optional)

## Key Format Auto-Detection

| Pattern | Entity | Example |
|---------|--------|---------|
| `E##` | Epic | `E01`, `E01-epic-name` |
| `E##-F##` or `F##` | Feature | `E01-F02`, `F02` |
| `E##-F##-###` | Task | `E01-F02-001`, `T-E01-F02-001` |
| `B###` | Bug | `B001`, `B042` |
| `C###` or `CC-###` | Change card | `C001`, `CC-001` |
| `I-YYYY-MM-DD-##` | Idea | `I-2026-01-15-01` |

All keys are **case insensitive**.

## Reading Entities

```bash
# Auto-detect type from key format
shark get E01                                # Epic
shark get E01-F02                            # Feature
shark get E01-F02-001                        # Task
shark get B001                               # Bug
shark get CC-001                             # Change card
shark get I-2026-01-15-01                    # Idea

# JSON and field extraction
shark get E01-F02-001 --json                 # Full JSON
shark get E01-F02-001 --field status         # Single field
shark get E01-F02-001 --field title          # Single field

# View markdown file
shark view E01-F02-001                       # Display task file content
```

## Listing Entities

```bash
# Smart listing (positional args)
shark list                                   # All epics
shark list E01                               # Features in E01
shark list E01 F02                           # Tasks in E01-F02

# Task filtering (via entity-specific commands)
shark list E01 F02 --json                    # Tasks in feature (JSON)
shark task list --status=in_development      # By status (task subcommand)
shark task list --agent=backend              # By agent
shark task list --blocked                    # Blocked tasks

# Feature/Epic listing
shark list E01 --json                        # Features in epic (JSON)
shark list --json                            # All epics (JSON)
```

## Updating Entities

```bash
# Auto-detect type from key
shark update E01-F02-001 --title="New Title"
shark update E01-F02-001 --priority=8
shark update B001 --title="New bug title"
shark update CC-001 --title="New change title"

# Entity-specific updates (legacy aliases — prefer verb-first syntax above)
shark update E01-F02-001 --title="New" --priority=5 --order=2
shark update E01-F02 --title="New Feature Name"
shark update E01 --title="New Epic Name"
```

**IMPORTANT: `shark update` does NOT accept `--status`.** Use `shark status set` or `shark status advance` instead.

## Deleting Entities

```bash
shark delete E01-F02-001                     # Delete task
shark delete E01-F02                         # Delete feature
shark delete E01                             # Delete epic (cascades)
shark delete B001                            # Delete bug
shark delete CC-001                          # Delete change card
shark delete I-2026-01-15-01                 # Delete idea
```

## Entity Relationships

```bash
# Preferred: top-level link command (works across all entity types)
shark link E01-F02-001 E01-F02-002 --type=depends_on
shark link B001 E01-F02-003 --type=related_to
shark link E01-F01 E01-F02 --type=follows

# Types: depends_on, blocks, related_to, follows, spawned_from, duplicates, references, linked_to
# Question-only gate: Question -> eligible non-Question entity
shark link Q001 E01-F01 --type=question_blocks

# Legacy task-specific syntax (still works)
shark task link E01-F02-001 E01-F02-002 --type=depends_on
shark task unlink E01-F02-001 E01-F02-002
shark task deps E01-F02-001                  # Dependency tree
```
