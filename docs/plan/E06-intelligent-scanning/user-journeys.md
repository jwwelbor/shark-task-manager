# User Journeys

**Epic**: [Intelligent Documentation Scanning](./epic.md)

---

## Overview

This document maps the key user workflows enabled or improved by the intelligent scanning epic.

---

## Journey 1: Initial Project Import from Existing Documentation

**Persona**: Technical Lead (Existing Project Migration)

**Goal**: Import 200+ existing markdown files into shark database without manual restructuring

**Preconditions**:
- Existing project with docs/plan/ containing epics, features, and tasks
- Documentation uses mixed conventions (some E## epics, some descriptive names)
- shark CLI installed and initialized (database exists)

### Happy Path

1. **Discover Documentation Structure**
   - User action: Run `shark scan --dry-run --verbose` to preview what would be imported
   - System response: Scanner recursively walks docs/plan/, detects epic-index.md (if present), identifies folder structure, applies pattern matching
   - Expected outcome: User sees report showing 245 files matched, 20 files skipped with warnings (unknown patterns)

2. **Review Scan Report**
   - User action: Read detailed report showing epics/features/tasks discovered, warnings about unmatched files
   - System response: Report groups findings by epic, shows matched patterns, lists unmatched files with reasons
   - Expected outcome: User understands what will be imported and what needs attention

3. **Configure Special Epic Types**
   - User action: Add "tech-debt" and "bugs" to .sharkconfig.json whitelist
   - System response: Config file updated with special_epic_types: ["tech-debt", "bugs", "change-cards"]
   - Expected outcome: Scanner will recognize these folder names as valid epics

4. **Re-run Scan with Updated Config**
   - User action: Run `shark scan --dry-run` again
   - System response: Scanner picks up new config, recognizes tech-debt and bugs as epics
   - Expected outcome: Report shows 258 files matched, 7 files skipped (legitimate unknowns)

5. **Execute Import**
   - User action: Run `shark scan --execute` to perform actual database import
   - System response: Scanner creates database records for all epics/features/tasks, establishes relationships, records file paths
   - Expected outcome: Database populated with 258 items, transaction committed successfully

6. **Validate Import**
   - User action: Run `shark epic list` and `shark task list --status=todo`
   - System response: CLI queries database, returns formatted lists with counts
   - Expected outcome: User sees all epics including special types, confirms task counts match expectations

**Success Outcome**: Technical Lead has successfully imported existing project into shark with minimal manual intervention. Database is now source of truth for project state while markdown files remain for context and git history.

### Alternative Paths

**Alt Path A: epic-index.md Present**
- **Trigger**: Scanner finds epic-index.md at docs/plan/epic-index.md
- **Branch Point**: After Step 1 (Discover Documentation Structure)
- **Flow**:
  1. Scanner parses epic-index.md for explicit epic/feature links and metadata
  2. Scanner validates folder structure matches index
  3. If conflicts found, epic-index.md takes precedence (per user configuration)
  4. Scanner imports based on index structure + folder contents
- **Outcome**: Import uses index as primary source of truth, reducing ambiguity

**Alt Path B: Validation Failures Detected**
- **Trigger**: Scanner detects file path mismatches or broken references during import
- **Branch Point**: After Step 5 (Execute Import)
- **Flow**:
  1. Scanner validates all file_path entries point to actual files
  2. Scanner checks for circular references or invalid relationships
  3. If validation fails, transaction rolls back
  4. User receives detailed error report with specific files/issues
  5. User fixes issues and re-runs import
- **Outcome**: Database remains consistent; no partial imports

### Critical Decision Points

- **Decision at Step 3**: Which special epic types to whitelist? User must know their documentation conventions.
- **Decision at Step 4**: Should user manually fix unmatched files or accept them as out-of-scope? Balance between completeness and effort.
- **Decision at Step 5**: Execute import or wait? User must be confident in dry-run results.

---

## Journey 2: Incremental Sync After Development Session

**Persona**: AI Agent (Incremental Development Sync)

**Goal**: Update database to reflect 5 new task files and 2 modified files created during work session

**Preconditions**:
- shark database exists and is populated
- Agent just completed work session, created/modified files in docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/tasks/
- Agent needs to sync before ending session

### Happy Path

1. **Initiate Incremental Sync**
   - User action: Agent runs `shark sync --incremental`
   - System response: Sync engine reads last_sync_time from config, identifies files modified since that time
   - Expected outcome: Scanner identifies 7 files to check (5 new, 2 modified)

2. **Detect Changes**
   - User action: System automatically scans identified files
   - System response: Scanner parses 5 new files (creates task records), updates 2 existing files (metadata changes)
   - Expected outcome: Change detection complete in <1 second

3. **Apply Changes in Transaction**
   - User action: System processes changes
   - System response: Begins database transaction, inserts 5 new tasks, updates 2 existing tasks, records file paths
   - Expected outcome: All changes applied atomically

4. **Update Sync Timestamp**
   - User action: System updates last_sync_time
   - System response: Writes current timestamp to config, updates sync metadata table
   - Expected outcome: Future syncs will only check files modified after this time

5. **Report Results**
   - User action: System displays sync summary
   - System response: Outputs "Synced 7 files: 5 created, 2 updated in 1.2 seconds"
   - Expected outcome: Agent confirms sync complete, knows database is current

**Success Outcome**: Database reflects all changes made during agent session. Next agent session will have accurate project state without reading markdown files.

### Alternative Paths

**Alt Path A: Conflict Detected (File Modified, Database Also Changed)**
- **Trigger**: File modification timestamp is newer than last_sync_time, but database record was also updated since last sync
- **Branch Point**: After Step 2 (Detect Changes)
- **Flow**:
  1. Scanner detects conflict: file metadata doesn't match database metadata
  2. Scanner applies configured strategy (file-wins by default for incremental sync)
  3. Scanner logs conflict details with old/new values
  4. Scanner proceeds with file-wins update
- **Outcome**: File changes take precedence; conflict logged for review

**Alt Path B: Invalid File Format**
- **Trigger**: Scanner cannot parse one of the modified files (invalid frontmatter)
- **Branch Point**: After Step 2 (Detect Changes)
- **Flow**:
  1. Scanner logs error: "Cannot parse frontmatter in T-E04-F07-003.md"
  2. Scanner skips invalid file, continues with remaining files
  3. Scanner reports 6 of 7 files synced, 1 error
  4. Transaction commits successfully for valid files
- **Outcome**: Partial sync succeeds; agent receives clear error about invalid file

### Critical Decision Points

- **Decision at Step 1**: Use incremental sync vs. full scan? Incremental is faster but may miss edge cases.

---

## Journey 3: Adding Custom Epic Pattern Mid-Project

**Persona**: Product Manager (Multi-Style Documentation)

**Goal**: Add custom pattern to recognize "tech-debt" as a valid epic and import existing tech-debt tasks

**Preconditions**:
- shark database exists with standard E## epics
- docs/plan/tech-debt/ folder exists with 15 task files
- tech-debt not currently matched by default patterns

### Happy Path

1. **Attempt to Query Tech-Debt Epic**
   - User action: Run `shark epic get tech-debt`
   - System response: Error: "Epic tech-debt not found in database"
   - Expected outcome: User realizes tech-debt folder wasn't imported

2. **Check Current Configuration**
   - User action: Run `shark config show` or read .sharkconfig.json
   - System response: Displays current patterns.epic.folder regex
   - Expected outcome: User sees current pattern only matches E##-slug format

3. **Update Configuration with Custom Pattern**
   - User action: Edit .sharkconfig.json patterns.epic.folder to add alternative: `(?P<epic_id>E\\d{2})-(?P<epic_slug>[a-z0-9-]+)|(?P<epic_id>tech-debt|bugs|change-cards)`
   - System response: Config file updated with expanded regex pattern
   - Expected outcome: Scanner will now match tech-debt, bugs, change-cards in addition to E##-slug

4. **Validate Pattern (Optional)**
   - User action: Run `shark config validate-patterns`
   - System response: Validates regex syntax and required capture groups, confirms patterns are valid
   - Expected outcome: Confidence that patterns will work correctly

5. **Run Targeted Scan**
   - User action: Run `shark scan --path=docs/plan/tech-debt --execute`
   - System response: Scanner applies updated pattern, matches tech-debt folder, creates epic record, imports tasks
   - Expected outcome: tech-debt epic and tasks now in database

6. **Verify Import**
   - User action: Run `shark epic get tech-debt` and `shark task list --epic=tech-debt`
   - System response: Returns tech-debt epic details and 15 associated tasks
   - Expected outcome: Tech-debt epic fully integrated into project tracking

**Success Outcome**: Product Manager can now track tech-debt items alongside standard epics, generate reports across all work, and query tech-debt status programmatically.

### Alternative Paths

**Alt Path A: Use Pattern Preset**
- **Trigger**: User wants to add common special epics without writing regex
- **Branch Point**: After Step 2 (Check Current Configuration)
- **Flow**:
  1. User runs `shark config add-pattern --preset=special-epics`
  2. System appends preset pattern for tech-debt, bugs, change-cards to config
  3. User proceeds to Step 5 (Run Targeted Scan)
- **Outcome**: Pattern added via preset, no regex knowledge required

**Alt Path B: Bulk Special Epic Import**
- **Trigger**: User has multiple special epic folders (tech-debt, bugs, change-cards)
- **Branch Point**: After Step 3 (Update Configuration)
- **Flow**:
  1. User pattern already includes all special types (tech-debt|bugs|change-cards)
  2. User runs `shark scan --execute` (full scan) instead of targeted scan
  3. Scanner recognizes all special types, imports all at once
- **Outcome**: All special epics imported in single operation

### Critical Decision Points

- **Decision at Step 3**: Edit pattern manually vs. use preset? Manual gives full control, preset is faster.
- **Decision at Step 4**: Validate patterns? Optional but recommended for complex regex.
- **Decision at Step 5**: Targeted scan vs. full rescan? Targeted is faster for single epic addition.

---

## Journey 4: Handling Conflicting epic-index.md and Folder Structure

**Persona**: Technical Lead (Existing Project Migration)

**Goal**: Import project where epic-index.md lists different epics than folder structure shows

**Preconditions**:
- docs/plan/epic-index.md exists with 5 epics listed
- docs/plan/ folder structure shows 7 epic folders (2 not in index)
- shark configuration set to "epic-index.md takes precedence"

### Happy Path

1. **Run Scan with Conflict Detection**
   - User action: Run `shark scan --dry-run --detect-conflicts`
   - System response: Scanner parses epic-index.md (5 epics), scans folders (7 epics), identifies 2 folders not in index
   - Expected outcome: Detailed conflict report showing index vs. folder differences

2. **Review Conflict Report**
   - User action: Read report showing which epics are index-only, folder-only, or matched
   - System response: Report lists E04 and E05 found in folders but not in index, suggests actions
   - Expected outcome: User understands discrepancy

3. **Decide on Resolution Strategy**
   - User action: User determines E04 and E05 are obsolete work-in-progress folders, should be excluded
   - System response: N/A (user decision)
   - Expected outcome: User knows to keep current config (index precedence)

4. **Execute Import with Index Precedence**
   - User action: Run `shark scan --execute`
   - System response: Scanner imports 5 epics from index, ignores 2 folder-only epics, logs exclusions
   - Expected outcome: Database contains only 5 epics listed in index

5. **Verify and Document**
   - User action: Run `shark epic list`, check that E04 and E05 are not present
   - System response: Shows 5 epics matching epic-index.md
   - Expected outcome: User confirms import matches intent, may delete obsolete folders

**Success Outcome**: Database accurately reflects intended project structure based on explicit epic-index.md, with orphaned folders safely ignored.

### Alternative Paths

**Alt Path A: User Wants to Include Folder-Only Epics**
- **Trigger**: User determines E04 and E05 are valid and should be imported
- **Branch Point**: After Step 3 (Decide on Resolution Strategy)
- **Flow**:
  1. User updates epic-index.md to include E04 and E05
  2. User re-runs scan
  3. Scanner imports all 7 epics (now all in index)
- **Outcome**: Index updated to match reality, full import succeeds

**Alt Path B: Merge Strategy Instead of Precedence**
- **Trigger**: User wants to import everything and manually clean up later
- **Branch Point**: After Step 2 (Review Conflict Report)
- **Flow**:
  1. User changes config to use merge strategy
  2. User runs `shark scan --execute --strategy=merge`
  3. Scanner imports all 7 epics (5 from index + 2 from folders)
  4. User manually reviews and deletes unwanted epics later
- **Outcome**: Complete import with manual cleanup phase

### Critical Decision Points

- **Decision at Step 3**: Trust index or trust folders? Depends on which is more reliable source of truth.
- **Decision at Alt Path B**: Merge everything vs. exclude unknown? Balance between completeness and cleanliness.

---

*See also*: [Requirements](./requirements.md)
