---
description: Review all folders in docs/plan/* and create/update epic-index.md with summaries, features, and links
---

# Generate Epic Index

Create or update `/docs/plan/epic-index.md` with a comprehensive, navigable index of **all folders** under `docs/plan/` and their contents.

## Purpose

This command provides a bird's-eye view of all planning documentation, making it easy to:
- Discover what epics and planning documents exist
- Understand relationships and dependencies
- Navigate to detailed documentation
- Track overall product development progress

## Execution Steps

### Step 1: Discover ALL Folders

Use Bash to list all directories under `docs/plan/`:

```bash
ls -d docs/plan/*/
```

This finds **every folder** regardless of naming convention - not just `E##-*` patterns.

### Step 2: Find Main Documentation File for Each Folder

For each folder found, check for documentation files in this priority order:

1. `epic.md` - Primary epic documentation
2. `epic-prd.md` - Epic PRD format
3. `README.md` - General documentation
4. Any `.md` file in the folder root

Use Glob to check each folder:
```bash
Glob: docs/plan/{folder-name}/*.md
```

### Step 3: Read Documentation Files in Parallel

For each folder, read the main documentation file to extract:
- Title/Name (from first `# heading` or folder name as fallback)
- Summary (first paragraph or description section, condensed to ≤10 words)
- Type (Epic, Feature, Documentation, etc. - inferred from content/structure)
- Status (if available)

Example:
```bash
Read: docs/plan/E01-auth-platform/epic.md
Read: docs/plan/architecture-decisions/README.md
Read: docs/plan/some-other-folder/epic-prd.md
```

### Step 4: Discover Sub-items for Each Folder

For folders that appear to be epics (have `epic.md` or `epic-prd.md`), find nested feature directories:

```bash
Glob: docs/plan/{epic-folder}/*/README.md
Glob: docs/plan/{epic-folder}/*/prd.md
```

For other folders, check for any meaningful subdirectories with documentation.

### Step 5: Extract Sub-item Information

For each sub-item found, read the main file to extract:
- Name
- Brief description (≤10 words)
- Status (if available)
- Dependencies (if listed)

### Step 6: Generate Epic Index

Create or overwrite `/docs/plan/epic-index.md` with this structure:

```markdown
# Planning Index

**Last Updated**: {YYYY-MM-DD}

---

## Overview

This index provides a navigable overview of all planning documentation. Items are organized by type and include links to detailed documentation.

**Total Items**: {count}
**Epics**: {count}
**Other Documentation**: {count}

---

## Quick Navigation

{For each item, create a TOC link}
- [{Display Name}](#{anchor})

---

## Epics

{For folders with epic.md or epic-prd.md}

### {Epic Name}

**Key**: `{folder-name}`

**Summary**: {10-words-or-less summary}

**Status**: {Status or "Planning"}

📁 **[Full Details](./{folder-name}/{main-file})**

#### Features/Components

{If sub-items exist:}
- **[{Sub-item Name}](./{folder-name}/{sub-folder}/{file})** - {Brief description}
  - Status: {Status or "Unknown"}

{If no sub-items:}
- _No features defined yet._

---

{Repeat for each epic}

---

## Other Documentation

{For folders WITHOUT epic.md or epic-prd.md}

### {Folder Name (humanized)}

**Summary**: {10-words-or-less summary from README.md or first .md file}

📁 **[View Documentation](./{folder-name}/{main-file})**

{If has subdirectories with docs:}
#### Contents
- [{Sub-item}](./{folder-name}/{sub-folder}/{file}) - {Brief description}

---

{Repeat for each non-epic folder}

---

## Summary Table

| Item | Type | Main File | Sub-items | Status |
|------|------|-----------|-----------|--------|
| {Name} | Epic/Doc | {file} | {count} | {Status} |
| ... | ... | ... | ... | ... |

---

## Notes

- **Epics**: Folders containing `epic.md` or `epic-prd.md`
- **Documentation**: Other planning folders with `README.md` or `.md` files
- **Adding Epics**: Use `epic-prd-writer` agent to create new epics
- **Adding Features**: Use `feature-architect` agent to design features within epics

---

*This index is auto-generated. Re-run `/generate-epic-index` to refresh.*
```

### Step 7: Output Summary

Provide user with:

```markdown
## Planning Index Generated Successfully

**Location**: `/docs/plan/epic-index.md`

**Summary**:
- **Total Folders**: {count}
- **Epics Found**: {count} (folders with epic.md or epic-prd.md)
- **Other Documentation**: {count} (folders with README.md or other .md files)
- **Total Sub-items**: {count}

### Folders Indexed:
{For each folder:}
- [{Name}](./{folder-name}/{main-file}) - {type}

### Next Steps:
- Review epic-index.md for comprehensive overview
- Use `/validate-feature-design` to check feature completeness
- Use `/validate-task-readiness` to verify tasks are ready
```

## Implementation Guidelines

### File Priority for Each Folder

Check for main documentation file in this order:
1. `epic.md` → Type: Epic
2. `epic-prd.md` → Type: Epic
3. `README.md` → Type: Documentation
4. First `.md` file alphabetically → Type: Documentation
5. No `.md` files → Type: Empty (still list folder with warning)

### Summary Extraction (10 words or less)

1. **From `# Title` section**: Look for first paragraph after title
2. **From `## Overview` or `## Summary`**: Extract key sentence
3. **From `## Goal > Problem`**: Extract core problem statement
4. **Pattern**: "{Verb} {noun} for {benefit}"
5. **Fallback**: Use folder name humanized (e.g., `auth-platform` → "Authentication Platform")

### Handling ALL Edge Cases

**Folder with no .md files**:
```markdown
### {folder-name}

⚠️ **No documentation found**

📁 **[View Folder](./{folder-name}/)**

_Consider adding a README.md to document this folder's purpose._
```

**Folder with only non-standard .md files**:
- Use the first `.md` file found (alphabetically)
- Note the file name in the output

**Empty subdirectories**:
- Still list the subdirectory
- Note: "Empty - no documentation"

**Mixed content folders**:
- If folder has both `epic.md` AND other structure, treat as Epic
- List all sub-items found

### Folder Name Humanization

Convert folder names to readable titles:
- `E01-auth-platform` → "E01: Auth Platform" or extract title from file
- `architecture-decisions` → "Architecture Decisions"
- `api-specs` → "API Specs"

Use title from document's `# heading` if available, folder name as fallback.

## Optimization Notes

### Performance
- **List all folders first**: Single `ls` command to get all directories
- **Parallel file discovery**: Glob all folders simultaneously
- **Parallel reads**: Read all main files in parallel
- **Single write**: Generate entire index in memory, write once

### Completeness
- **Every folder gets listed**: No folder is skipped
- **Graceful degradation**: Missing files get warnings, not errors
- **Type inference**: Automatically categorize based on file presence

### Accuracy
- **Extract exact titles**: Use document headings when available
- **Preserve links**: Use relative paths for portability
- **Date stamping**: Always include current date in "Last Updated"

## Success Criteria

This command succeeds when:
1. ✅ **ALL directories** under `/docs/plan/` are discovered and listed
2. ✅ Each folder has its main documentation file identified
3. ✅ Epics (with epic.md/epic-prd.md) are categorized separately
4. ✅ Other documentation folders are listed in their own section
5. ✅ All sub-items within folders are discovered and linked
6. ✅ Folders without documentation get warnings (not skipped)
7. ✅ All links are relative and functional
8. ✅ Index is well-formatted and easy to navigate
9. ✅ File is written to `/docs/plan/epic-index.md`

## Example Output

For a docs/plan/ with mixed content:

```markdown
# Planning Index

**Last Updated**: 2025-01-15

---

## Overview

This index provides a navigable overview of all planning documentation.

**Total Items**: 5
**Epics**: 3
**Other Documentation**: 2

---

## Quick Navigation

- [E01: Authentication Platform](#e01-authentication-platform)
- [E02: User Dashboard](#e02-user-dashboard)
- [E03: Notifications](#e03-notifications)
- [Architecture Decisions](#architecture-decisions)
- [API Specifications](#api-specifications)

---

## Epics

### E01: Authentication Platform

**Key**: `E01-auth-platform`

**Summary**: Modernize authentication for improved security

**Status**: In Progress

📁 **[Full Details](./E01-auth-platform/epic.md)**

#### Features/Components

- **[OAuth Integration](./E01-auth-platform/oauth-integration/README.md)** - Third-party auth support
  - Status: In Progress
- **[MFA Implementation](./E01-auth-platform/mfa/README.md)** - Multi-factor authentication
  - Status: Planning

---

### E02: User Dashboard

**Key**: `E02-user-dashboard`

**Summary**: Create unified dashboard for user insights

**Status**: Planning

📁 **[Full Details](./E02-user-dashboard/epic-prd.md)**

#### Features/Components

- _No features defined yet._

---

## Other Documentation

### Architecture Decisions

**Summary**: Record of architectural decisions and rationale

📁 **[View Documentation](./architecture-decisions/README.md)**

#### Contents
- [ADR-001: Database Selection](./architecture-decisions/adr-001.md) - PostgreSQL chosen
- [ADR-002: API Framework](./architecture-decisions/adr-002.md) - FastAPI selected

---

### API Specifications

**Summary**: OpenAPI specs for all service endpoints

📁 **[View Documentation](./api-specs/README.md)**

---

## Summary Table

| Item | Type | Main File | Sub-items | Status |
|------|------|-----------|-----------|--------|
| E01: Auth Platform | Epic | epic.md | 2 | In Progress |
| E02: User Dashboard | Epic | epic-prd.md | 0 | Planning |
| Architecture Decisions | Doc | README.md | 2 | - |
| API Specifications | Doc | README.md | 0 | - |
```

## Notes

- Run this command periodically as content is added/updated
- Every folder gets representation - nothing is silently skipped
- Warnings highlight folders needing documentation
- For detailed information, always refer to source files
