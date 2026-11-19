---
feature_key: E36-F01-consult-bridge
epic_key: E36
title: Consult bridge
description: /shark-rider consult <agent> advisor bridge: agent/skill list descriptions (one Go change), inline and delegated consultation verbs, query.md NL recognizer, SKILL.md allowlist rows. Read-only by default.
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
2. **Skill-layer** (`skills/shark-rider/verbs/consult.md`): `/shark-rider consult <agent> [referent]` resolves the agent (exact name, else fuzzy-match by name *and* description, else show the available agents), loads the persona via `shark agent get <agent>` (frontmatter stripped), and adopts it **inline** — turn-by-turn, not a background subagent. Bare consult and consult help show a role-selection guide. A separate `/shark-rider consult-agent <agent> [referent]` verb provides one-shot advice through a bounded host subagent. The `query.md` consult-intent recognizer routes natural-language "ask/have/consult `<agent>` to/about `<task>`" phrasing to inline consultation. `SKILL.md` registers both verbs.

### Impact

- `shark agent list` / `shark skill list` become self-describing, enabling both human discovery and consult fuzzy-matching.
- Every design conversation becomes a first-class command instead of ad-hoc prompting.
- The pattern (skill verb calls CLI, adopts persona inline) is the canonical example of the skill leveraging the Go CLI as a tool.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to run `/shark-rider consult cx-designer review this plan` so that the cx-designer persona is loaded and reviews the plan in-voice without me manually copying any system prompt.

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

3. **REQ-F-003**: Consultation skill verbs
   - **Description**: `skills/shark-rider/verbs/consult.md` implements `/shark-rider consult <agent> [referent]` as an inline consultation. `skills/shark-rider/verbs/consult-agent.md` implements `/shark-rider consult-agent <agent> [referent]` as a separate one-shot delegated consultation: resolve the persona and referent, then launch one bounded host subagent only when the host can enforce read-only permissions for workspace writes, Shark mutations, and nested dispatch. The parent reviews and synthesizes the advice. If those permissions are unavailable, do not launch and direct the user to inline consult. Both modes use the shared request-resolution procedure and are registered in the `SKILL.md` allowlist.
   - **Priority**: Must-Have

4. **REQ-F-004**: Agent resolution
   - **Description**: Both modes use `context/consult-request-resolution.md` to resolve the agent, persona, and referent before mode-specific behavior. Resolution order: (1) exact name match against `agent list`; (2) fuzzy-match against names *and* descriptions (typos and role words like "ux"/"ops" work). If a fuzzy match is ambiguous, list matching agents and ask the user to clarify. If the agent list is empty, suggest `shark admin install-shark-data` and stop. Bare `/shark-rider consult` and `/shark-rider consult help` show the role-selection guide and stop.
   - **Priority**: Must-Have

5. **REQ-F-005**: Inline persona adoption
   - **Description**: Load persona via `shark agent get <agent>` (frontmatter stripped). Adopt **inline** — the current conversation continues as the persona, turn-by-turn. No background subagent is spawned.
   - **Priority**: Must-Have

6. **REQ-F-006**: Read-only by default
   - **Description**: A consult reads shark/docs for context as the persona would but does not mutate shark state (create entities, update status, write files) unless the user explicitly asks. The distinction between a consult and a dispatched `spawn_agent` worker is preserved.
   - **Priority**: Must-Have

7. **REQ-F-007**: Referent resolution
   - **Description**: A user-provided path-shaped token with a path separator or known file suffix is resolved against the active project workspace, canonicalized with symlinks, and read only if it remains inside that root; outside paths are not read. A period in ordinary prose is not enough to identify a path. "This" or "this plan/design/doc" with an unambiguous in-context artifact → use it and state which file was used. Ambiguous → ask the user or accept pasted content. Never silently guess the wrong artifact.
   - **Priority**: Must-Have

8. **REQ-F-008**: `query.md` consult-intent recognizer
   - **Description**: Add a recognizer to `query.md` (the NL catch-all router) that matches *ask / have / consult `<agent>` to/about `<task>`* phrasing. Extracts agent name + task description, hands off to `consult.md`. Fires only on clear "talk to an agent" phrasing; falls back to the explicit form if the agent name doesn't resolve.
   - **Priority**: Must-Have

9. **REQ-F-009**: Graceful degradation when agent is absent
   - **Description**: If `shark agent get <agent>` fails for either `consult` or `consult-agent` (agent not in bundle, override, or disk), print a clear unavailability message, show `shark agent list --all`, and stop before adopting or dispatching the persona. Never hard-fail or throw an unhandled error.
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
- **When** `/shark-rider consult cx-designer docs/design.md` is run
- **Then** the persona is loaded via `shark agent get cx-designer` (frontmatter stripped), the file is read, and the response is in the cx-designer's voice
- **And** no shark entities are created or updated

**Scenario 3: Explicit consult — "this" referent**
- **Given** a plan document was just referenced in the conversation
- **When** `/shark-rider consult tech-director review this plan` is run
- **Then** the verb identifies the in-context artifact, states which file it used, and delivers the review
- **And** if there is no unambiguous artifact, the user is asked to paste or specify

**Scenario 4: NL routing**
- **Given** `query.md` has the consult-intent recognizer
- **When** the user types "have the devops agent review this deployment config"
- **Then** the router matches the pattern, extracts agent=`devops`, referent=`this deployment config`, and hands off to `consult.md`

**Scenario 5: Unknown agent**
- **Given** the user requests `/shark-rider consult made-up-agent`
- **When** `shark agent list --json` contains no exact or fuzzy match
- **Then** a clear no-match message is printed followed by `shark agent list --all`
- **And** execution stops without calling `shark agent get` or printing an error trace

**Scenario 6: Listed agent content cannot be loaded**
- **Given** the requested agent appears in `shark agent list --json`
- **When** `shark agent get <agent>` fails or returns no persona definition
- **Then** a clear content-unavailable message is printed followed by `shark agent list --all`
- **And** execution stops before persona adoption or subagent dispatch without an error trace

**Scenario 7: Successful delegated consultation**
- **Given** the `architect` agent is available, `docs/design.md` is the unambiguous design artifact in the conversation, and the host can enforce read-only permissions for workspace writes, Shark mutations, and nested dispatch
- **When** `/shark-rider consult-agent architect review this design` is run
- **Then** the persona is loaded, `docs/design.md` is identified and read, and both are included in one bounded host-subagent request with the enforced read-only profile
- **And** the parent reviews the advice and synthesizes a response in its own voice
- **And** no Shark state, workspace files, or additional agents are changed or launched

**Scenario 8: No agents are installed**
- **Given** `shark agent list --json` returns an empty list
- **When** the user requests either `/shark-rider consult <agent>` or `/shark-rider consult-agent <agent>`
- **Then** the user is told that no Shark agents are installed and is advised to run `shark admin install-shark-data`
- **And** execution stops before matching, persona loading, or subagent dispatch

**Scenario 9: Host cannot enforce delegated read-only permissions**
- **Given** no available host launcher can prevent workspace writes, Shark mutations, and nested dispatch
- **When** `/shark-rider consult-agent architect review this design` is requested
- **Then** no subagent is launched, the host limitation is explained, and the user is directed to inline `/shark-rider consult architect review this design`

**Scenario 10: User path referent escapes the workspace**
- **Given** a user-provided path-shaped referent resolves outside the active project workspace, directly or through a symlink
- **When** either consultation mode resolves that referent
- **Then** the target is not read, the user is asked to paste the content or provide an in-workspace copy, and execution stops before persona adoption or dispatch; no target content is sent to a subagent, while documented host-local skill/context reads remain available

---

## Out of Scope

- **Replacing inline consult with subagent dispatch** — `/shark-rider consult` remains inline and advisory. `/shark-rider consult-agent` is a separate, explicit one-shot advice mode.
- **State mutation during consult** — a consult is read-only by default; explicit mutation is the user's opt-in.
- **Relocating agent definitions** — agents stay in the embed (`internal/sharkdata/default_data/agents/`); the consult bridge is sufficient without moving definitions.

---

## Success Metrics

N/A — internal tooling; see epic scope.md exclusions.

---

*Last Updated*: 2026-06-29

## Post-completion clarification (2026-11-19)

The completed inline consultation contract remains in force. Bare `consult` and
`consult help` show the role-selection guide; named `consult` requests continue
in the persona's voice. The separate `consult-agent` command delegates one
read-only advice request only when the host can enforce the required permission
profile; otherwise it does not launch and directs the user to inline `consult`.
Natural-language consult requests continue to use the inline mode.
