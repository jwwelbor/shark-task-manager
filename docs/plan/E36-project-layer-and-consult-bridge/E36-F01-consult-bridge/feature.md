---
feature_key: E36-F01-consult-bridge
epic_key: E36
title: Consult bridge
description: /shark consult <agent> advisor bridge: agent/skill list descriptions (one Go change), verbs/consult.md, query.md NL recognizer, SKILL.md allowlist row. Read-only by default.
---

# Consult bridge

**Feature Key**: E36-F01-consult-bridge | **Size**: S

**Epic**: [Project Layer and Consult Bridge](../../epic.md) | **Design**: [plan.md §4–5](../../../../../dev-artifacts/2026-06-29-project-entity-design/plan.md)

---

## Goal

### Problem

There is no first-class way to talk to shark's agent personas as advisors. Personas exist only to be dispatched as `spawn_agent` workers. To consult a persona today you must manually copy its definition and ad-hoc prompt — fragile, undiscoverable, and not repeatable. Separately, `shark agent list` (and `shark skill list`) print `name → source` with no description, so you cannot tell agents apart without running `shark agent get` on each, making the discovery menu useless for fuzzy-matching.

### Solution

Two tightly coupled pieces ship together:

1. **One Go change** (`internal/services/content_bundle_service.go`): add `Description string` to `BundleContentEntry`, populate it from each entry's frontmatter `description:` field (respecting override > disk > embedded precedence, best-effort), and render it in `agent list` / `skill list` text and JSON output.
2. **Skill-layer** (`skills/shark/verbs/consult.md`): a `/shark consult <agent> [referent]` verb that resolves the agent (exact name, else fuzzy-match by name *and* description, else show the menu), loads the persona via `shark agent get <agent>` (frontmatter stripped), and adopts it **inline** — turn-by-turn, not a background subagent. A consult-intent recognizer added to `query.md` routes natural-language "ask/have/consult `<agent>` to/about `<task>`" phrasing automatically. One `SKILL.md` allowlist row exposes the verb.

### Impact

- `shark agent list` / `shark skill list` become self-describing, enabling both human discovery and consult fuzzy-matching.
- Every design conversation becomes a first-class command instead of ad-hoc prompting.
- The pattern (skill verb calls CLI, adopts persona inline) is the canonical example of the skill leveraging the Go CLI as a tool.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to run `/shark consult cx-designer review this plan` so that the cx-designer persona is loaded and reviews the plan in-voice without me manually copying any system prompt.

**Acceptance Criteria**:
- [ ] The verb resolves `cx-designer` via `shark agent get cx-designer`.
- [ ] The in-context plan artifact is identified and read (or the user is asked if ambiguous).
- [ ] The response is delivered in the cx-designer's persona voice.
- [ ] No shark state is mutated.

**Story 2**: As a developer, I want to type "ask the tech-director to look at this architecture" so that the router routes me to the consult verb automatically.

**Acceptance Criteria**:
- [ ] `query.md` recognizer matches the phrasing and extracts agent=`tech-director` and referent.
- [ ] Control passes to `consult.md` with those parameters.
- [ ] If the agent doesn't resolve, the recognizer falls back to the explicit form.

**Story 3**: As a developer, I want `shark agent list` to show a description for each agent so that I can choose the right persona without running `agent get` on each one.

**Acceptance Criteria**:
- [ ] Text output: each entry shows `name — description` (or similar).
- [ ] JSON output: each entry includes a `description` field.
- [ ] A frontmatter parse failure on one entry does not abort the list.

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: `BundleContentEntry` description field
   - **Description**: Add `Description string` to `BundleContentEntry` in `internal/services/content_bundle_service.go`. In `BundleContentService.List`, populate it by reusing the same frontmatter parser used by `Get`. Precedence: override > disk > embedded. If parsing fails or the field is absent, leave empty — never fail the list call.
   - **Priority**: Must-Have

2. **REQ-F-002**: Render description in list output
   - **Description**: `agent list` and `skill list` render `Description` in both text (human-readable, one line per entry) and JSON (field on each entry object) output. Applies to both `shark agent list` and `shark skill list` commands.
   - **Priority**: Must-Have

3. **REQ-F-003**: `verbs/consult.md` skill verb
   - **Description**: A `skills/shark/verbs/consult.md` file implements `/shark consult <agent> [referent]`. Registered in `SKILL.md` allowlist so it is callable via the shark skill router.
   - **Priority**: Must-Have

4. **REQ-F-004**: Agent resolution
   - **Description**: Resolution order: (1) exact name match against `agent list`; (2) fuzzy-match against names *and* descriptions (typos and role words like "ux"/"ops" work); (3) if omitted entirely, show the menu (agent list with descriptions). If a fuzzy match is ambiguous, show the menu.
   - **Priority**: Must-Have

5. **REQ-F-005**: Inline persona adoption
   - **Description**: Load persona via `shark agent get <agent>` (frontmatter stripped). Adopt **inline** — the current conversation continues as the persona, turn-by-turn. No background subagent is spawned.
   - **Priority**: Must-Have

6. **REQ-F-006**: Read-only by default
   - **Description**: A consult reads shark/docs for context as the persona would but does not mutate shark state (create entities, update status, write files) unless the user explicitly asks. The distinction between a consult and a dispatched `spawn_agent` worker is preserved.
   - **Priority**: Must-Have

7. **REQ-F-007**: Referent resolution
   - **Description**: Explicit file path → read it. "This" or "this plan/design/doc" with an unambiguous in-context artifact → use it and state which file was used. Ambiguous → ask the user or accept pasted content. Never silently guess the wrong artifact.
   - **Priority**: Must-Have

8. **REQ-F-008**: `query.md` consult-intent recognizer
   - **Description**: Add a recognizer to `query.md` (the NL catch-all router) that matches *ask / have / consult `<agent>` to/about `<task>`* phrasing. Extracts agent name + task description, hands off to `consult.md`. Fires only on clear "talk to an agent" phrasing; falls back to the explicit form if the agent name doesn't resolve.
   - **Priority**: Must-Have

9. **REQ-F-009**: Graceful degradation when agent is absent
   - **Description**: If `shark agent get <agent>` fails (agent not in bundle, override, or disk), print a clear unavailability message and the `agent list` menu. Never hard-fail or throw an unhandled error.
   - **Priority**: Must-Have

### Non-Functional Requirements

1. **REQ-NF-001**: Minimal Go footprint
   - **Description**: The Go change is limited to `BundleContentEntry` and its rendering in the list commands. Quality gate (`make fmt && make lint && make test`) must pass. No new Go packages.

2. **REQ-NF-002**: No schema or dispatch changes
   - **Description**: Existing `agent get` and `spawn_agent` dispatch paths are unchanged. Agents remain canonical in `internal/sharkdata/default_data/agents/`; per-project tuning via `shark-data/overrides/agents/<name>.md` continues to work.

---

## Acceptance Criteria

**Scenario 1: Agent list shows descriptions**
- **Given** the embedded bundle contains agents with frontmatter `description:` fields
- **When** `shark agent list` is run (text output)
- **Then** each agent entry includes its description on the same line or immediately after the name
- **And** `shark agent list --json` includes a `"description"` field per entry
- **And** an agent with a missing or unparseable `description:` field still appears in the list (description empty)

**Scenario 2: Explicit consult — known agent, explicit path**
- **Given** the `cx-designer` agent is available and `docs/design.md` exists
- **When** `/shark consult cx-designer docs/design.md` is run
- **Then** the persona is loaded via `shark agent get cx-designer` (frontmatter stripped), the file is read, and the response is in the cx-designer's voice
- **And** no shark entities are created or updated

**Scenario 3: Explicit consult — "this" referent**
- **Given** a plan document was just referenced in the conversation
- **When** `/shark consult tech-director review this plan` is run
- **Then** the verb identifies the in-context artifact, states which file it used, and delivers the review
- **And** if there is no unambiguous artifact, the user is asked to paste or specify

**Scenario 4: NL routing**
- **Given** `query.md` has the consult-intent recognizer
- **When** the user types "have the devops agent review this deployment config"
- **Then** the router matches the pattern, extracts agent=`devops`, referent=`this deployment config`, and hands off to `consult.md`

**Scenario 5: Unknown agent**
- **Given** the user requests `/shark consult made-up-agent`
- **When** the verb attempts to resolve and `shark agent get made-up-agent` fails
- **Then** a clear message is printed (e.g., "Agent 'made-up-agent' not found") followed by the available agent list
- **And** execution ends gracefully without an error trace

---

## Out of Scope

- **Consult as a background subagent** — consult is inline and advisory; `spawn_agent` dispatch remains the mechanism for worker tasks.
- **State mutation during consult** — a consult is read-only by default; explicit mutation is the user's opt-in.
- **Relocating agent definitions** — agents stay in the embed (`internal/sharkdata/default_data/agents/`); the consult bridge is sufficient without moving definitions.

---

## Success Metrics

N/A — internal tooling; see epic scope.md exclusions.

---

*Last Updated*: 2026-06-29
