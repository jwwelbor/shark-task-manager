# Naming Conventions

This document defines file and directory naming standards for all specification documents.

## Directory Structure

All specification documents live under `/docs/plan/`:

```
/docs/plan/
├── E01-epic-slug/
│   ├── epic.md
│   ├── personas.md
│   ├── user-journeys.md
│   ├── requirements.md
│   ├── success-metrics.md
│   ├── scope.md
│   ├── E01-F01-feature-slug/
│   │   ├── prd.md
│   │   ├── 02-architecture.md
│   │   ├── 03-database-design.md
│   │   ├── 04-api-specification.md
│   │   ├── 05-frontend-design.md
│   │   ├── 06-security-performance.md
│   │   └── 07-implementation-phases.md
│   └── E01-F02-another-feature/
│       └── ...
└── E02-another-epic/
    └── ...
```

## Epic Keys

**Format**: `E##-{epic-slug}`

**Rules**:
- Two-digit zero-padded epic number (E01, E02, ..., E99)
- Hyphen separator
- Lowercase slug with hyphens (kebab-case)
- Epic slug should be 2-4 words describing the epic
- Epic slug should be memorable and descriptive

**Examples**:
- `E01-claude-reorg`
- `E09-identity-platform`
- `E12-notification-system`
- `E15-analytics-dashboard`

## Feature Keys

**Format**: `E##-F##-{feature-slug}`

**Rules**:
- Includes parent epic number (E##)
- Two-digit zero-padded feature number (F01, F02, ..., F99)
- Hyphen separator
- Lowercase slug with hyphens (kebab-case)
- Feature slug should be 2-5 words describing the feature
- Feature numbers are scoped to the epic (each epic starts at F01)

**Examples**:
- `E01-F01-skill-extraction`
- `E09-F01-oauth-integration`
- `E09-F02-user-management`
- `E12-F01-email-notifications`

## Phased Epics

If an epic requires phased implementation, break into separate epics:

**Format**: `E##a-{epic-slug}`, `E##b-{epic-slug}`

**Examples**:
- `E09a-identity-platform` (Phase A: Core authentication)
- `E09b-identity-platform` (Phase B: Social login)
- `E09c-identity-platform` (Phase C: MFA)

**Alternative**: Use separate epic numbers if phases are independent:
- `E09-core-authentication`
- `E10-social-login`
- `E11-multi-factor-auth`

## Task Keys

**Format**: `T-E##-F##-###.md`

**Rules**:
- Starts with `T-` prefix
- Includes epic number (E##) and feature number (F##)
- Three-digit zero-padded task sequence (001, 002, ..., 999)
- Hyphen separators
- Created in `/docs/plan/{epic-key}/{feature-key}/tasks/` directory
- Task files remain in feature directory throughout their lifecycle
- Status is tracked in database via `shark` CLI, not by folder location

**Examples**:
- `T-E01-F01-001.md`
- `T-E01-F01-002.md`
- `T-E04-F01-001.md`
- `T-E09-F01-001.md`

**File Location**:
All tasks for a feature live in:
```
/docs/plan/{epic-key}/{feature-key}/tasks/
├── T-E##-F##-001.md
├── T-E##-F##-002.md
└── T-E##-F##-003.md
```

**Status Management**:
Task status is managed via `shark` CLI:
- `shark task list --status=todo` - List todo tasks
- `shark task start T-E01-F01-001` - Start task (status: in_progress)
- `shark task complete T-E01-F01-001` - Complete task (status: ready_for_review)
- `shark task approve T-E01-F01-001` - Approve task (status: completed)

## Epic Files

Epic files use simple, descriptive names without numbering:

- `epic.md` - Main index and summary
- `personas.md` - User personas
- `user-journeys.md` - User workflows
- `requirements.md` - Requirements catalog
- `success-metrics.md` - KPIs and metrics
- `scope.md` - Boundaries and exclusions

## Feature Files

Feature files use numbered prefixes to indicate reading order:

- `prd.md` - Product Requirements Document (always first)
- `02-architecture.md` - System architecture
- `03-database-design.md` - Database schema
- `04-api-specification.md` - API contracts
- `05-frontend-design.md` - UI components
- `06-security-performance.md` - Non-functional requirements
- `07-implementation-phases.md` - Phasing and timeline

**Note**: `prd.md` doesn't have a number prefix because it's the entry point. Design documents start at `02-` to indicate they come after the PRD.

## Task Directory Structure

All implementation tasks live with their feature documentation:

```
/docs/plan/{epic-key}/{feature-key}/
├── prd.md
├── 02-architecture.md
├── 03-data-design.md
└── tasks/
    ├── README.md          # Task index and workflow
    ├── T-E##-F##-001.md
    ├── T-E##-F##-002.md
    └── T-E##-F##-003.md
```

Task status is tracked in database, managed via `shark` CLI.

## Slug Guidelines

### Epic Slugs
- Use 2-4 words
- Focus on the domain or capability
- Examples: `identity-platform`, `analytics-dashboard`, `notification-system`

### Feature Slugs
- Use 2-5 words
- Focus on the specific feature or component
- Examples: `oauth-integration`, `user-management`, `email-notifications`

### Task Slugs
- Use 2-4 words
- Focus on the component or implementation phase
- Examples: `contract-validation`, `database-setup`, `api-implementation`, `frontend-development`

### General Slug Rules
- Use only lowercase letters, numbers, and hyphens
- No spaces, underscores, or special characters
- Start and end with a letter or number (not a hyphen)
- Be descriptive but concise
- Avoid redundancy (don't repeat epic/feature info in slug)

## Bad vs Good Examples

### Bad:
- ❌ `epic-1` (not descriptive)
- ❌ `E1-identity` (missing zero-padding)
- ❌ `E01_identity_platform` (underscores instead of hyphens)
- ❌ `E01-Identity-Platform` (uppercase)
- ❌ `E01-F1-oauth` (missing zero-padding)
- ❌ `T1-database.md` (missing epic/feature context, wrong number format)
- ❌ `01-Task-database-setup.md` (wrong format, uppercase)
- ❌ `E01-F01-T01-database-setup.md` (old format with slugs)

### Good:
- ✅ `E01-identity-platform`
- ✅ `E01-F01-oauth-integration`
- ✅ `T-E01-F01-001.md`
- ✅ `T-E12-F03-002.md`
- ✅ `E12-notification-system`
- ✅ `E12-F03-sms-integration`

## Special Cases

### Index Files
- Epic index: `epic.md` (at epic root)
- Task index: `README.md` (in feature's tasks/ directory)

### Persona Files
If creating new personas, save to `/docs/personas/`:
- Format: `{persona-name-desc}.md`
- Example: `marketing-manager-saas.md`
- Example: `enterprise-it-admin.md`

### Design Document Templates
If creating templates, use descriptive names:
- `architecture-template.md`
- `api-spec-template.md`
- `database-design-template.md`

## Consistency Rules

1. **Always use hyphens**, never underscores or spaces
2. **Always use lowercase**, never uppercase or mixed case
3. **Always zero-pad numbers** (01, not 1)
4. **Always include context** (E##-F##-P## for PRPs)
5. **Be descriptive** - slugs should be self-explanatory
6. **Be concise** - but don't sacrifice clarity

## Validation Checklist

Before finalizing any specification document, verify:

- [ ] Directory name follows `E##-{epic-slug}` or `E##-F##-{feature-slug}` format
- [ ] Epic number is zero-padded (E01, not E1)
- [ ] Feature number is zero-padded (F01, not F1)
- [ ] Task sequence is zero-padded (001, not 1)
- [ ] Task files follow format: `T-E##-F##-###.md`
- [ ] Task files are created in `/docs/plan/{epic-key}/{feature-key}/tasks/` directory
- [ ] No uppercase letters in any file or directory names
- [ ] No underscores or spaces in any file or directory names
- [ ] Slugs are descriptive and meaningful
