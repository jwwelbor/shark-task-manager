# Status Viewer UI Reference

Dark-themed, IDE-style interface with a sidebar tree navigator, a content pane, and a dashboard.

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
- Instruction text
- "Pick Folder" button (accent blue)
- Note about read-only access

---

### #02 Main App — Dashboard View (default)

Three-panel layout: header, sidebar, content area.

**Header (100px fixed width):**
- Left: "Dashboard" title (clickable, returns to dashboard) + summary pills showing entity counts (e.g., "2 Epics", "3 Features", "3 Tasks")
- Right: "Refresh" button + "Pick Folder" button (change directory)

**Sidebar (320px fixed width):**
- Tree navigator (see Sidebar Tree Structure below)

**Content area — Dashboard renders four sections:**

1. **Status Breakdown** — Grid of cards, one per entity type that has items. Each card shows: type label, total count, colored status badges with counts.

2. **Feature Progress** — List of features with progress bars. For each feature: key (clickable), title, progress bar, percentage complete.

3. **Active Transitions** — Last 25 transitions across all entities, sorted newest first. Each row: key (clickable), from-badge, to-badge, note. Show "Open" button to navigate to the entity's content (only if content exists).

4. **Stale Entities** — Entities with `updated` dates older than 7 days that are NOT in a terminal status. Sorted by staleness descending. Each row: key (clickable), status badge, title, last update.

---

### #03 Main App — Entity View (clicking a tree node)

**Content pane (sidebar):** Shows all entity metadata fields as a key-value grid. File path with copy button. Linked content path with copy button. Status field rendered as a colored badge.

**Content toolbar:** Shows entity name, plus toggle buttons:
- **Info button:** switches to content view
- **Transitions button:** switches to transition history view

**Default content view:** If linked content exists, render it as markdown. Show "History" button. If no content exists, render the entity's raw data and show "History" if transitions exist.

**History drawer:** Renders the transition table in reverse chronological order. Each row: date, from-badge, to-badge, note. Show "Open" button to return to content view (only if content exists).

---

### #04 Main App — Doc View (clicking a design doc in the tree)

Plain markdown rendering. No history buttons. Content panel shows whatever metadata exists.

---

## Sidebar Tree Structure

The tree has sections, each with a sticky uppercase header.

### "Hierarchy" section

```
▼ E01: Notification System        (epic node, expanded by default)
    architecture                  (doc at epic level, italic/muted)
  ▶ E01-F01: Email Notifications  (feature node, collapsed by default)
      design                      (doc at feature level, italic/muted)
      T-E01-F01-001: Set up email...  (task node)
      T-E01-F01-002: Build templates  (task node)
```

### Indentation (tree) padding

- **Epic:** 16px left padding, bold
- **Feature:** 40px left padding
- **Doc at feature level:** 40px left padding, italic muted title
- **Task:** 56px left padding

Each entity node shows: status dot (colored circle) | key (monospace, accent blue) | title (secondary text, truncates with ellipsis).

**Expand/collapse:** Clicking the arrow toggles child visibility. Epics start expanded. Features start collapsed. Clicking the node itself (not the arrow) selects it.

**Selected state:** Black-tinted background, accent-colored left border, title becomes primary text color.

### Flat sections: "Tags", "Tech Debt", "Ideas", "Change Cards"

Flat list (no nesting). Same node format as hierarchy entities. 18px left padding. Only shown if items exist.

---

## Key Behaviors

### Navigation

Clicking any key in the dashboard (feature progress, recent activity, stale items) navigates to that entity: finds the node in the tree, expands it if collapsed, selects the node, scrolls it into view.

### Keyboard

- `Escape` closes the current view/window

---

## Visual Design

### Theme: Dark (Slate palette)

- Deep dark background base
- Accent color: blue (used for keys, buttons, selected borders)
- Secondary text: muted/gray for titles and docs
- Status values always rendered as colored badges/dots using the color table above
