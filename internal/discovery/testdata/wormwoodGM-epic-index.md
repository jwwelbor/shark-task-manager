---
status: draft
feature: /docs/plan
---

# Planning Index

**Last Updated**: 2025-12-09


## Overview

This index provides a navigable overview of all planning documentation. Items are organized by type and include links to detailed documentation.

**Total Items**: 15
**Epics**: 11
**Other Documentation**: 4

---

## Quick Navigation

### Epics
- [E00: Launchpad](#e00-launchpad)
- [E01: Content Ingestion](#e01-content-ingestion)
- [E02: CLI Content Submission](#e02-cli-content-submission)
- [E03: Voice Integration](#e03-voice-integration)
- [E04: shadcn-vue Migration](#e04-shadcn-vue-migration)
- [E05: Integration Foundation](#e05-integration-foundation)
- [E05: Memory System](#e05-memory-system)
- [E06: Content Ingestion Pipeline](#e06-content-ingestion-pipeline)
- [E07: Extended Content Types](#e07-extended-content-types)
- [E08: Remove Pin Feature](#e08-remove-pin-feature)
- [E09: UX Enhancement](#e09-ux-enhancement)
- [E10: OpenTelemetry Observability](#e10-opentelemetry-observability)

### Other Documentation
- [Bug Tracker](#bug-tracker)
- [Change Cards](#change-cards)
- [Tech Debt](#tech-debt)
- [E09: Campaign Screen Redux](#e09-campaign-screen-redux)

---

## Epics

### E00: Launchpad

**Key**: `E00-launchpad`

**Summary**: Core platform infrastructure and user onboarding features

**Status**: In Progress

**[Full Details](./E00-launchpad/epic.md)**

#### Features/Components

- **[F01: User Authentication](./E00-launchpad/F01-User_Authentication/README.md)** - Supabase auth with JWT tokens
- **[F02: Campaign Creation Wizard](./E00-launchpad/F02-Campaign_Creation_Wizard/README.md)** - Multi-step campaign setup workflow
- **[F03: Invitation System](./E00-launchpad/F03-Invitation_System/README.md)** - Player invitation and sharing
- **[F04: Player Onboarding](./E00-launchpad/F04-Player_Onboarding/README.md)** - New player experience flow
- **[F05: Private Content Management](./E00-launchpad/F05-Private_Content_Management/README.md)** - User content CRUD operations
- **[F06: Session Launch Lobby](./E00-launchpad/F06-session-launch-lobby/README.md)** - Real-time game session management
- **[F07: Dashboard Navigation](./E00-launchpad/F07-Dashboard_Navigation/README.md)** - Main navigation and layout

---

### E01: Content Ingestion

**Key**: `E01-content-ingestion`

**Summary**: PDF and document ingestion with AI extraction

**Status**: Planning

**[Full Details](./E01-content-ingestion/epic.md)**

#### Features/Components

- _Feature PRDs documented in epic folder (F01-F19)_
- Focus on content upload, processing, and storage pipeline

---

### E02: CLI Content Submission

**Key**: `E02-cli-content-submission`

**Summary**: REST APIs for TTRPG content CRUD operations

**Status**: Implemented

**[Full Details](./E02-cli-content-submission/epic.md)**

#### Features/Components

- **[F01: Ruleset Management API](./E02-cli-content-submission/F01-ruleset-management-api-implementation/README.md)** - Game system definitions
- **[F02: Campaign Management API](./E02-cli-content-submission/F02-campaign-management-api-implementation/README.md)** - Campaign CRUD with hierarchy
- **[F03: Scenario Management API](./E02-cli-content-submission/F03-scenario-management-api-implementation/README.md)** - Adventure/module management
- **[F04: Room/Location Management API](./E02-cli-content-submission/F04-room-location-management-api-implementation/README.md)** - Location content types
- **[F05: NPC/Character Management API](./E02-cli-content-submission/F05-npc-character-management-api-implementation/README.md)** - Character data management
- **[F06: Supporting Content Types API](./E02-cli-content-submission/F06-supporting-content-types-api-implementation/README.md)** - Factions, rumors, events
- **[F07: Relationship/Dependency Management](./E02-cli-content-submission/F07-relationship-dependency-management-implementation/README.md)** - Content linking
- **[F11: Authentication/Authorization](./E02-cli-content-submission/F11-authentication-authorization-implementation/README.md)** - API security layer

---

### E03: Voice Integration

**Key**: `E03-voice-integration`

**Summary**: Real-time voice streaming and AI-powered transcription

**Status**: In Progress

**[Full Details](./E03-voice-integration/epic.md)**

#### Features/Components

- **[F01: WebSocket Streaming](./E03-voice-integration/F01-websocket-streaming/README.md)** - Real-time audio transport
- **[F02: Speech-to-Text](./E03-voice-integration/F02-speech-to-text/README.md)** - Deepgram transcription integration
- **[F04: Text-to-Speech](./E03-voice-integration/F04-text-to-speech/README.md)** - AI GM voice synthesis
- **[F07: Session Transcription](./E03-voice-integration/F07-session-transcription/README.md)** - Full session capture
- **[F08: Voice Profile Management](./E03-voice-integration/F08-voice-profile-management/README.md)** - User voice settings
- **[F09: Multi-User Sessions](./E03-voice-integration/F09-multi-user-sessions/README.md)** - Multiplayer voice handling

---

### E04: shadcn-vue Migration

**Key**: `E04-shadcn-migration`

**Summary**: Migrate from React shadcn/ui to Vue-native components

**Status**: Planning

**Complexity**: M (5 story points)

**[Full Details](./E04-shadcn-migration/epic.md)**

#### Features/Components

- F01: shadcn-vue Installation & Setup
- F02: Authentication Flow Migration
- F03: Component Cleanup
- F04: Documentation & Testing

---

### E05: Integration Foundation

**Key**: `E05-integration-foundation`

**Summary**: Security fixes and agent context service foundation

**Status**: In Progress

**Priority**: P0 - Critical Path to MVP

**[Full Details](./E05-integration-foundation/epic.md)**

#### Features/Components

- **[PRPs](./E05-integration-foundation/prps/README.md)** - Implementation prompts
- F01: Security Authorization Fixes
- F02: AgentContextService
- F03: Campaign Hierarchy Verification
- F04: Shared Schema Foundation

---

### E05: Memory System

**Key**: `E05-memory-system`

**Summary**: AI GM memory continuity for campaign context recall

**Status**: Planning

**[Full Details](./E05-memory-system/epic.md)**

#### Features/Components

- F01: Session State Persistence
- F02: Hierarchical Summarization
- F03: Knowledge Graph
- F04: Context Retrieval API
- F05: Memory Query Interface
- F06: Narrative Callbacks
- F07: NPC Dossiers
- F08: Scenario Progress
- F09: Privacy Controls
- F10: Voice Transcription

---

### E06: Content Ingestion Pipeline

**Key**: `E06-content-ingestion-pipeline`

**Summary**: Connect uploads to extraction and storage pipeline

**Status**: Planning

**Priority**: P0 - MVP Core

**[Full Details](./E06-content-ingestion-pipeline/epic.md)**

#### Features/Components

- **[F01: Content Ingestion Orchestrator](./E06-content-ingestion-pipeline/F01-content-ingestion-orchestrator/README.md)** - Central pipeline coordinator
- **[F02: Content Source Tracking](./E06-content-ingestion-pipeline/F02-content-source-tracking/README.md)** - Bridge table for source linking
- **[F03: Processing Queue Integration](./E06-content-ingestion-pipeline/F03-processing-queue-integration/README.md)** - Async processing via queue
- **[F04: Basic Rules Extraction](./E06-content-ingestion-pipeline/F04-basic-rules-extraction/README.md)** - Minimal viable extractor
- **[F05: E01 PRD Updates](./E06-content-ingestion-pipeline/F05-e01-prd-updates/README.md)** - Documentation alignment

---

### E07: Extended Content Types

**Key**: `E07-extended-content-types`

**Summary**: APIs for character options, equipment, and lore content

**Status**: Planning

**Priority**: P1 - MVP Completeness

**[Full Details](./E07-extended-content-types/epic.md)**

#### Features/Components

- **[F01: Character Options API](./E07-extended-content-types/F01-character-options-api/README.md)** - Races, classes, feats, backgrounds
- **[F02: Equipment Catalog API](./E07-extended-content-types/F02-equipment-catalog-api/README.md)** - Weapons, armor, items
- **[F04: Lore Content API](./E07-extended-content-types/F04-lore-content-api/README.md)** - Deities, pantheons, lore entries
- F03: Random Tables API Completion
- F05: AgentContextService Extension

---

### E08: Remove Pin Feature

**Key**: `E08-remove-pin`

**Summary**: Remove unused campaign pin functionality and cleanup

**Status**: Draft

**Complexity**: L (Large)

**[Full Details](./E08-remove-pin/epic.md)**

#### Features/Components

- F01: Remove Campaign Pinning - Backend
- F02: Remove Campaign Pinning - Frontend
- F03: Remove Secondary Pin Features

---

### E09: UX Enhancement

**Key**: `E09-ux-enhancement`

**Summary**: Fix navigation failures and improve campaign management UX

**Status**: In Progress

**Complexity**: XL (Split into 2 Phases)

**[Full Details](./E09-ux-enhancement/epic.md)**

#### Features/Components

- **[F01: Navigation Route Fixes](./E09-ux-enhancement/F01-navigation-route-fixes/prps/README.md)** - Fix broken /characters and /library routes
- **[F02: Campaign Tabbed Interface](./E09-ux-enhancement/F02-campaign-tabbed-interface/prps/README.md)** - Add tabbed navigation to campaigns
- **[F03: Player Invitation Enhancement](./E09-ux-enhancement/F03-player-invitation-enhancement/prps/README.md)** - Improve invite discoverability
- **[P02-F01: Character Management](./E09-ux-enhancement/E09-P02-F01-character-management/README.md)** - Complete character list and editing

---

### E10: OpenTelemetry Observability

**Key**: `E10-opentelemetry-observability`

**Summary**: Distributed tracing and observability infrastructure

**Status**: Draft

**[Full Details](./E10-opentelemetry-observability/epic.md)**

#### Features/Components

- F01: Core Infrastructure Setup
- F02: Request Tracing Enhancement
- F03: External Service Instrumentation
- F04: Celery Worker Tracing
- F05: Structured Logging Integration
- F06: Metrics Integration
- F07: Infrastructure Setup (Tempo, Loki)
- F08: Dashboards and Alerting
- **[PRPs](./E10-opentelemetry-observability/prps/)** - 21 implementation prompts

---

## Other Documentation

### Bug Tracker

**Summary**: Active and resolved bug reports for the platform

**[View Folder](./bug-tracker/)**

#### Contents
- **[Open Bugs](./bug-tracker/open/)** - 3 open issues
  - B-20251206-001: Campaign wizard scenario filter (Major)
  - B-20251206-002: Ruleset not found in campaign wizard (Major)
  - B-20251206-003: Character save wizard (Critical)
- **[Resolved Bugs](./bug-tracker/resolved/)** - 1 resolved
  - B-20251206-004: Character wizard 409 version conflict

---

### Change Cards

**Summary**: Lightweight change cards for small tweaks and fixes

**[View Folder](./change-cards/)**

#### Contents
- **[CC-20251206-001](./change-cards/CC-20251206-001-add-characters-tab-to-campaign-view.md)** - Add characters tab to campaign view
- **[CC-20251202-001](./change-cards/CC-20251202-001-character-ready-to-active-transition.md)** - Character ready-to-active transition
- **[CC-20251202-003](./change-cards/CC-20251202-003-campaign-selection-wizard-bug.md)** - Campaign selection wizard bug
- **[CC-20251202-004](./change-cards/CC-20251202-004-wizard-localstorage-isolation.md)** - Wizard localStorage isolation
- **[CC-20251201-001](./change-cards/CC-20251201-001-character-status-alignment.md)** - Character status alignment
- **[CC-20251201-002](./change-cards/CC-20251201-002-rename-standalone-to-characters.md)** - Rename standalone to characters

---

### Tech Debt

**Summary**: Technical debt items tracked for future resolution

**[View Folder](./tech-debt/)**

#### Contents
- **[TD-001](./tech-debt/TD-001-consolidate-http-clients.md)** - Consolidate HTTP clients
- **[TD-003](./tech-debt/TD-003-missing-game-systems-list-endpoint.md)** - Missing game systems list endpoint
- **[TD-20251129-004](./tech-debt/TD-20251129-004-consolidate-utility-modules.md)** - Consolidate utility modules
- **[TD-20251130-005](./tech-debt/TD-20251130-005-consolidate-gamesystemresponse.md)** - Consolidate GameSystemResponse
- **[E03-F03 Code Review](./tech-debt/E03-F03-code-review-tech-debt.md)** - Voice integration tech debt
- **[Frontend Standards Gaps](./tech-debt/coding-standards-gaps-frontend.md)** - Frontend coding standards gaps

---

### E09: Campaign Screen Redux

**Key**: `E09-campaign-screen-redux`

**Summary**: Campaign management screen redesign notes

**[View Folder](./E09-campaign-screen-redux/)**

#### Contents
- Napkin sketch for campaign screen redesign

---

### FUTURE_SCOPE

**Key**: `FUTURE_SCOPE`

**[View Folder](./FUTURE_SCOPE/)**

_Empty - No documentation found. Consider adding a README.md to document future scope items._

---

## Summary Table

| Item | Type | Main File | Sub-items | Status |
|------|------|-----------|-----------|--------|
| E00: Launchpad | Epic | epic.md | 7 | In Progress |
| E01: Content Ingestion | Epic | epic.md | 19 PRDs | Planning |
| E02: CLI Content Submission | Epic | epic.md | 8 | Implemented |
| E03: Voice Integration | Epic | epic.md | 6 | In Progress |
| E04: shadcn-vue Migration | Epic | epic.md | 4 | Planning |
| E05: Integration Foundation | Epic | epic.md | 4 | In Progress |
| E05: Memory System | Epic | epic.md | 10 | Planning |
| E06: Content Ingestion Pipeline | Epic | epic.md | 5 | Planning |
| E07: Extended Content Types | Epic | epic.md | 5 | Planning |
| E08: Remove Pin Feature | Epic | epic.md | 3 | Draft |
| E09: UX Enhancement | Epic | epic.md | 4 | In Progress |
| E10: OpenTelemetry Observability | Epic | epic.md | 8 | Draft |
| Bug Tracker | Doc | - | 4 | - |
| Change Cards | Doc | - | 6 | - |
| Tech Debt | Doc | - | 6 | - |

---

## Notes

- **Epics**: Folders containing `epic.md` or `epic-prd.md`
- **Documentation**: Other planning folders with `README.md` or `.md` files
- **Adding Epics**: Use `epic-prd-writer` agent to create new epics
- **Adding Features**: Use `feature-architect` agent to design features within epics
- **E05 Numbering Note**: Both Integration Foundation and Memory System use E05 prefix - consider renumbering Memory System

---

*This index is auto-generated. Re-run `/generate-epic-index` to refresh.*
