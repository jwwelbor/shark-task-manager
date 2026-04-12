# Rebuild Prompt: Status Tracker Viewer

Build a self-contained, single-file HTML viewer for a markdown-based project status tracking system. The viewer reads `.md` files from disk using the browser's File System Access API — no server, no build step, no generation. It is a dark-themed, IDE-style interface with a sidebar tree navigator, a content pane, and a dashboard.

---

## What It Does

The viewer is pointed at a `docs/plan` directory that contains two sections:

1. **tracker/** — a "database" of status-oriented markdown files. Each file represents one tracked entity (epic, feature, task, bug, idea, tech-debt, change-card). Files contain ONLY YAML frontmatter and a status transition table. No prose.
2. **specs/** — Markdown References — Hierarchical design documents. These are the actual content: PRDs, architecture docs, design docs, task specs. They're organized in nested folders by epic and feature.

The viewer combines these two sources: it reads entity metadata/status from `tracker/`, resolves linked spec content via a `spec:` frontmatter field, and renders both in a unified interface.

---

## Data Model

### Tracker Files (the "Database")

Located under `tracker/` in flat subdirectories: `tracker/epics/`, `tracker/features/`, `tracker/tasks/`, `tracker/bugs/`, `tracker/ideas/`, `tracker/tech-debt/`, `tracker/change-cards/`

Each tracker file has this exact format — frontmatter + a transition table, nothing else:

```markdown
---
key: E01-F01
type: feature
title: Email Notifications
status: in_progress
parent: E01
priority: 1
spec: ../E01-notifications/E01-F01-email-notifications/prd.md
created: 2026-04-06
updated: 2026-04-06
---

## Transitions

| Date       | From | To          | Note                |
|------------|------|-------------|---------------------|
| 2026-04-06 | —    | —           | Created             |
| 2026-04-06 | todo | open        | Approved            |
| 2026-04-06 | todo | in_progress | Development started |
```

### Frontmatter Fields by Entity Type

| Field        | Epic | Feature     | Task          | Bug  | Idea | Tech-Debt | Change-Card |
|--------------|------|-------------|---------------|------|------|-----------|-------------|
| `key`        | E##  | E##-F##     | E##-F##-### (or T-E##-F##-###) | B-### | — | — | CC-### |
| `type`       | epic | feature     | task          | bug  | idea | tech-debt | change-card |
| `title`      | yes  | yes         | yes           | yes  | yes  | yes       | yes         |
| `status`     | yes  | yes         | yes           | yes  | yes  | yes       | yes         |
| `parent`     | —    | epic key    | feature key   | —    | —    | —         | —           |
| `priority`   | 1-5  | 1-5         | 1-5           | 1-5  | 1-5  | 1-5       | 1-5         |
| `created`    | date | date        | date          | date | date | date      | date        |
| `updated`    | date | date        | date          | date | date | date      | date        |
| `spec`       | —    | path        | path          | —    | —    | —         | —           |
| `security`   | .archive/path | — | —            | —    | —    | —         | —           |
| `promote_to` | —    | E##-F##     | —             | —    | —    | —         | —           |

> **Hierarchy is encoded in the `parent` field, not in folder structure.** Epics have no parent. Features reference an epic key. Tasks reference a feature key.

---

### Spec Files (file content)

Located at root level in hierarchical folders:

```
E01-notifications/
  epics.md                          — epic spec
  architecture.md                   — design doc (key/type)
  E01-F01-email-notifications/
    prd.md                          — feature spec
    design.md                       — design doc
    tasks/
      T-E01-F01-001.md              — task spec
```

Spec files are full prose markdown documents with optional minimal frontmatter (just `title`). They do **NOT** contain status, key, type, or transition tables.

### Design/Architecture Doc Files

Any `.md` file outside `tracker/` that doesn't have `key` + `type` frontmatter is treated as a doc file. These are associated with their hierarchy level by extracting epic/feature folder path segments (e.g., a file in `E01-notifications/` is associated with Epic E01; a file in `E01-notifications/E01-F01-email-notifications/` is associated with Feature E01-F01).

### The `spec:` Field

Tracker files store a relative path to their spec file. Resolution requires walking `..` segments from the tracker file's directory. For example:
- Tracker file: `tracker/epics/E01-notifications.md`
- `spec: ../E01-notifications/epics.md`
- Resolved path: `E01-notifications/epics.md`

---

## Status Values and Colors

```
draft:        #8c8c8c  (gray)
todo:         #4d4dbb  (blue)
in_progress:  #e7b30e  (yellow)
open:         #e7b30e  (yellow)
done:         #4d4269  (green)
blocked:      #8c5700  (dark orange)
cancelled:    #4b4b36  (dark gray)
triaged:      #e7b500  (yellow)
verified:     #4d4269  (red)
closed:       #4b4b26  (dark gray)
wont_fix:     #4b4b26  (dark gray)
identified:   #8c8c8c  (gray)
confirmed:    #4d4dbb  (blue)
proposed:     #4d4dbb  (blue)
evaluating:   #e7b500  (blue)
accepted:     #4d4269  (green)
promoted:     #8c4dbb  (purple)
rejected:     #8c4d6c  (dark gray)
reopened:     #e7b500  (purple/blue)
approved:     #4d4269  (green)
```

**Terminal statuses** (excluded from stale detection): `done`, `cancelled`, `closed`, `wont_fix`, `verified`, `rejected`, `promoted`, `approved`.

---

## Application States

### #01 Pick Folder Screen

Show on initial load. Centered layout with:
- Folder icon (large, faded)
- "Status Tracker" heading
- Instruction text mentioning `docs/plan`
- "Pick Folder" button (accent blue)
- Note about read-only access

Clicking the button triggers `window.showDirectoryPicker({ mode: 'read' })`. On success, read all `.md` files recursively, build data, and transition to the main app.

---

### #02 Main App — Dashboard View (default after folder pick)

Three-panel layout: header, sidebar, content area.

**Header (100px fixed width):**
- Left: "Dashboard" title (clickable, returns to dashboard) + summary pills showing entity counts (e.g., "2 Epics", "3 Features", "3 Tasks")
- Right: "Refresh" button (re-reads all files from disk) + "Pick Folder" button (change directory)

**Sidebar (320px fixed width):**
- Top: "Dashboard" tab (hidden in dashboard view)
- Bottom: content panel (hidden until a node is selected)

**Content area — Dashboard renders four sections:**

1. **Status Breakdown** — Grid of cards, one per entity type that has items. Each card shows: type label, total count, colored status badges with counts.

2. **Feature Progress** — List of features with progress bars. For each feature, count tasks with `status: done` vs total tasks. Show key (clickable), title, progress bar, percentage.

3. **Active Transitions** — Last 25 transitions across ALL tracker files, sorted newest first. Show key (clickable), from-badge, to-badge, note. Displayed when clicking "History". Show "Open" button to return to spec (only if spec exists).

4. **Stale Entities** — Entities with `updated` dates older than 7 days that are NOT in terminal status. Sorted by staleness descending. Each row: key (clickable), status badge, title, last update.

---

### #03 Main App — Entity View (clicking a tree node)

When clicking a tracker entity in the tree:

**Content pane (of sidebar):** Shows all frontmatter fields from the tracker file as a key-value grid. File path with copy button. Resolved spec path with copy button. Status field rendered as a colored badge. Internal fields (`_path`, `_category`, `_specPath`, `_transitions`) are hidden.

**Content toolbar:** Shows filename, plus toggle buttons:
- **Info button:** switches to spec content view
- **Transitions button:** switches to transition history view
- **History toggle:** toggles the transition display

**Default view (of sidebar):** If a resolved `spec:` file exists, render the spec document as markdown by default. Show the "History" button. If no spec exists, render the tracker file's markdown and show "History" if transitions exist.

**History drawer:** Renders the transition table from the tracker file in reverse chronological order. Each row: date, from-badge, to-badge, note. Displayed when clicking "History". Show "Open" button to return to spec (only if spec exists).

**Mdx tagline:** When active, shows the raw markdown source of whichever view is current (spec content if viewing spec, tracker content if viewing history/tracker).

---

### #04 Main App — Doc View (clicking a design doc in the tree)

Plain markdown rendering. No spec/history buttons. Content panel shows whatever frontmatter exists. No transition handling.

---

## Sidebar Tree Structure

The tree has three sections, each with a sticky uppercase header:

### "Hierarchy" section

The tree has format — epics → tasks, with design docs interposed:

```
▼ E01: Notification System        (epic node, expanded by default)
    architecture                  (doc at epic level, italic/muted)
  ▶ E01-F01: Email Notifications  (feature node, collapsed by default)
      design                      (doc at feature level, italic/muted)
      T-E01-F01-001: Set up email...  (task node)
      T-E01-F01-002: Build templates  (task node)
  T-E01-F01-001: Push Notifications
```

### Indentation (tree) padding

- **Epic:** 16px left padding, `5m`, bold, italic muted title
- **Feature:** 40px level, `20px`, `5m`, italic muted title
- **Doc:** at feature level: 40px, `5m`, italic muted title
- **Task:** 56px

Each entity node may append across (if has children/docs): status dot (colored circle) | key (monospace, accent blue) | title (secondary text, truncates with ellipsis).

**Expand/collapse:** Clicking the arrow toggles child visibility. Epics start expanded. Features start collapsed. Clicking the node itself (not the arrow) selects it.

**Selected state:** Black-tinted background, accent-colored left border, title becomes primary text color.

### "Tags", "Tech Debt", "Ideas", "Change Cards" sections

Flat list (no nesting). Same format as entities. Split each into `Flat` indent (18px).

Only show if items exist.

---

## Key Behaviors

### File Reading

- Use `window.showDirectoryPicker()` with read mode
- Recursively read all `.md` files into a `FILES` map (relative path → content string)
- On refresh: re-read files, preserve current selection if file still exists

### Frontmatter Parsing

Simple line-by-line parser. Looks for `---` delimiters. Splits each line on first `:`. Strips surrounding quotes from values. Returns `{ #s, body }`.

### Transition Table Parsing

Scan the body for markdown table lines. Find the table with Date/From/To/Note columns. Extracts each data row into `{ date, from, to, note }`.

### Spec Path Resolution

Resolves relative `spec:` paths from tracker files:
- `resolvePath('tracker/epics/E01-notifications.md', '../E01-notifications/epics.md')`
- Walk `..` to pop directory segments, then append remaining path parts.

### Entity-to-Hierarchy Association

Hierarchy is built from `parent:` fields, not folder structure:
- **Epics:** no parent, top-level nodes
- **Features:** `parent:` matches an epic key → nested under that epic
- **Tasks:** `parent:` matches a feature key → nested under that feature
- **Orphan features** (parent doesn't match any epic) appear at top level

### Doc-to-Hierarchy Association

Non-tracker `.md` files are associated by scanning their folder path for key patterns:
- Folder name matching `E##` pattern → associated with that epic
- Folder name matching `E##-F##` pattern → associated with that feature
- Folder matching that last token in the path (currently)

### Navigation

Clicking any key in the dashboard (feature progress, recent activity, stale items) navigates to that entity: finds the tree node by matching `key` in frontmatter, expands it if collapsed, selects the node, scrolls it into view.

### Keyboard

- `Escape` closes the window

---

## Visual Design

### Theme: Dark (Slate palette)

```
bg-base:     [deepest background]
```
