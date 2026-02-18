---
feature_key: E15-F05-epic-and-feature-service-expansion
epic_key: E15
title: Epic and Feature Service Expansion
description: Implement EpicService and FeatureService with CRUD operations, progress/health calculations, and feature rollup logic. These services receive business logic migrated from repositories (F06) and provide the foundation for CLI refactoring (F07).
---

# Epic and Feature Service Expansion

**Feature Key**: E15-F05-epic-and-feature-service-expansion

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Personas**: [User Personas](../../personas.md)

---

## Goal

### Problem

EpicService and FeatureService exist but lack critical methods for progress calculation, health analysis, and feature rollup operations. Repository layer currently performs these operations (FeatureRepository.CalculateProgress, EpicRepository.GetHealthStatus), but this violates clean architecture. Without complete services, CLI commands cannot be refactored to thin wrappers (must call repositories directly for missing functionality).

### Solution

Expand EpicService and FeatureService to include all business logic methods: CRUD operations, progress/health calculations, feature rollups, and action item tracking. Services will be ready to receive logic from repositories (F06) and support CLI refactoring (F07).

### Impact

- **Service layer complete**: All Epic/Feature business logic centralized in services
- **Architecture compliance**: Services own business logic, repositories own data access
- **CLI readiness**: Services provide all methods needed for CLI thin wrappers

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1**: As a Shark developer, I want EpicService to handle all epic business logic so that CLI commands only call service methods.

**Acceptance Criteria**:
- [ ] EpicService.GetHealth() calculates health status (healthy/warning/critical)
- [ ] EpicService.GetImpediments() analyzes blocked tasks
- [ ] EpicService.GetFeatureRollup() aggregates feature statuses
- [ ] All methods use repositories for data access only

**Story 2**: As a Shark developer, I want FeatureService to handle all feature business logic so that CLI commands only call service methods.

**Acceptance Criteria**:
- [ ] FeatureService.GetProgress() calculates weighted and completion progress
- [ ] FeatureService.GetHealth() determines health status
- [ ] FeatureService.GetActionItems() identifies ready_for_* tasks
- [ ] All methods tested with mocked repositories

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Implement EpicService.GetHealth()
   - Calculate health based on feature statuses, blocked tasks, approval age
   - Return: healthy/warning/critical

2. **REQ-F-002**: Implement EpicService.GetImpediments()
   - Analyze blocked tasks (age, reason, priority)
   - Return structured impediment data

3. **REQ-F-003**: Implement FeatureService.GetProgress()
   - Calculate weighted progress (using status metadata)
   - Calculate completion progress (raw percentage)
   - Return both metrics

4. **REQ-F-004**: Implement FeatureService.GetHealth()
   - Analyze task statuses, blockers, approval age
   - Return health status

---

## Success Metrics

- **Service completeness**: 100% of Epic/Feature business logic in services
- **Zero repository business logic**: After F06, no business logic in repositories
- **Test coverage**: All service methods tested with mocks (no database)

---

## Dependencies

- **Blocks**: E15-F06 (Repository cleanup needs these services to exist)
- **Blocks**: E15-F07 (CLI refactoring needs complete services)

---

*Last Updated*: 2026-02-17
