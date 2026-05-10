# Success Metrics

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)

---

## Overview

This epic succeeds when Shark becomes the primary place for small entity edits and explicit Sprint actions instead of just a read-only dashboard.

---

## Metrics

### 1. Viewer-Based Mutation Adoption

- **Definition**: Percentage of routine entity edits completed in the viewer rather than through direct CLI or database edits
- **Target**: At least 70% of routine notes, dependency, and status changes performed from the viewer within one release cycle after launch

### 2. Time To Complete A Routine Edit

- **Definition**: Median time from opening an entity to saving a note/status/metadata change
- **Target**: Under 60 seconds for common single-field changes

### 3. Sprint Planning Action Completion

- **Definition**: Percentage of planned Sprint staging actions completed without leaving Sprint mode
- **Target**: 90% or higher for active planning sessions

### 4. Validation Error Rate

- **Definition**: Share of mutation attempts rejected for validation reasons
- **Target**: Expected to exist, but should trend downward after UI guidance is added; the metric is used to identify confusing flows rather than to force zero errors

### 5. History Coverage

- **Definition**: Share of successful writes that produce visible history entries
- **Target**: 100%

### 6. Viewer Exit Rate For Routine Updates

- **Definition**: Percent of routine edits that require the user to leave the viewer to finish
- **Target**: Under 20%

---

## Measurement Notes

- Measure using existing viewer and service instrumentation where available.
- Combine API usage logs with manual workflow observation for the first release.
- Use the metric trend direction as a quality signal, not as an absolute gate for every rollout.
