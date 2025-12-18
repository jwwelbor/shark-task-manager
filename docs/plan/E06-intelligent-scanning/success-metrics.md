# Success Metrics

**Epic**: [Intelligent Documentation Scanning](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of the intelligent scanning epic.

**Measurement Timeline**: Initial metrics within 2 weeks of release, full evaluation at 90 days post-launch

---

## Primary Success Metrics

### Metric 1: Import Success Rate

**Type**: Leading

**What We're Measuring**:
Percentage of existing markdown files successfully imported into shark database without manual intervention or file restructuring.

**How We'll Measure**:
- **Data Source**: Scan reports generated during initial project imports
- **Calculation Method**: (files_imported / total_files_scanned) × 100
- **Measurement Frequency**: Per-import basis, aggregated weekly

**Success Criteria**:
- **Baseline**: ~40% success rate with current rigid scanning (E04-F07)
- **Target**: 90% success rate with intelligent scanning
- **Timeline**: Achieve target within first 10 project imports
- **Minimum Viable**: 80% success rate (still 2x improvement over baseline)

**Relates To**:
- **Requirement(s)**: REQ-F-001 through REQ-F-008 (all discovery requirements)
- **User Journey**: Journey 1 (Initial Project Import)
- **Business Value**: Directly measures adoption barrier reduction

---

### Metric 2: Incremental Sync Performance

**Type**: Leading

**What We're Measuring**:
Time elapsed from `shark sync --incremental` command execution to completion, measuring different file change volumes.

**How We'll Measure**:
- **Data Source**: CLI timing instrumentation (log start/end time)
- **Calculation Method**: end_time - start_time in seconds
- **Measurement Frequency**: Every sync operation, aggregated by change volume buckets

**Success Criteria**:
- **Baseline**: 15-30 seconds for full project scan (E04-F07 current implementation)
- **Targets**:
  - 1-10 files changed: <2 seconds (typical AI agent session)
  - 11-50 files changed: <5 seconds
  - 51-100 files changed: <5 seconds
  - 100+ files changed: <30 seconds (still faster than baseline via incremental)
- **Timeline**: Achieve targets consistently within first 30 days of use
- **Minimum Viable**: 5 seconds for 100 files (still 3-6x faster than baseline)

**Relates To**:
- **Requirement(s)**: REQ-F-009, REQ-F-010, REQ-NF-001
- **User Journey**: Journey 2 (Incremental Sync After Development Session)
- **Business Value**: Reduces agent session overhead, enables frequent syncing

---

### Metric 3: Conflict Resolution Accuracy

**Type**: Leading

**What We're Measuring**:
Percentage of automatically resolved conflicts (file vs. database, index vs. folder) that align with user intent (no manual reversal required).

**How We'll Measure**:
- **Data Source**: Manual review of conflict logs + user feedback surveys
- **Calculation Method**: (correct_resolutions / total_conflicts_auto_resolved) × 100
- **Measurement Frequency**: Weekly review of conflict logs for first 60 days

**Success Criteria**:
- **Baseline**: N/A (no automatic conflict resolution in E04-F07)
- **Target**: 95% accuracy (only 5% of auto-resolutions require manual correction)
- **Timeline**: Achieve target by 30 days post-launch through strategy refinement
- **Minimum Viable**: 85% accuracy (reasonable trade-off for automation benefit)

**Relates To**:
- **Requirement(s)**: REQ-F-004 (Conflict Resolution Strategy), REQ-F-010
- **User Journey**: Journey 4 (Handling Conflicting epic-index.md and Folder Structure)
- **Business Value**: Measures trust in automated resolution, reduces manual intervention needs

---

### Metric 4: Special Epic Type Adoption

**Type**: Lagging

**What We're Measuring**:
Number of projects using special epic types (tech-debt, bugs, change-cards) as percentage of total shark projects.

**How We'll Measure**:
- **Data Source**: Telemetry (if enabled) or manual survey of early adopters
- **Calculation Method**: (projects_with_special_epics / total_projects_using_shark) × 100
- **Measurement Frequency**: Monthly survey or telemetry aggregation

**Success Criteria**:
- **Baseline**: 0% (feature doesn't exist yet)
- **Target**: 40% of projects use at least one special epic type
- **Timeline**: Measure at 90 days post-launch
- **Minimum Viable**: 25% adoption (validates feature solves real need)

**Relates To**:
- **Requirement(s)**: REQ-F-003 (Special Epic Type Whitelist)
- **User Journey**: Journey 3 (Adding Special Epic Type Mid-Project)
- **Business Value**: Validates demand for flexible epic organization

---

### Metric 5: Parse Error Rate

**Type**: Leading (quality indicator)

**What We're Measuring**:
Percentage of scanned files that fail to parse due to invalid frontmatter or markdown structure.

**How We'll Measure**:
- **Data Source**: Scan report warnings/errors aggregated across all scans
- **Calculation Method**: (files_with_parse_errors / total_files_scanned) × 100
- **Measurement Frequency**: Per-scan, aggregated weekly

**Success Criteria**:
- **Baseline**: Unknown (not currently tracked)
- **Target**: <5% parse error rate (95% of files parse successfully)
- **Timeline**: Maintain target consistently after first 30 days
- **Minimum Viable**: <10% parse error rate

**Relates To**:
- **Requirement(s)**: REQ-NF-005 (Error Recovery)
- **User Journey**: Journey 2, Alt Path B (Invalid File Format)
- **Business Value**: Measures scanner robustness against real-world file variations

---

## Secondary Success Metrics

### Metric 6: Dry-Run Usage Rate

**Type**: Leading (indicates user caution/trust)

**What We're Measuring**:
Percentage of first-time scans that use --dry-run flag before actual import.

**Target**: 60% of first-time users run dry-run before execute (indicates users value preview)

**Justification**: High usage validates need for preview capability; declining usage over time indicates growing trust in scanner.

---

### Metric 7: Validation Command Usage

**Type**: Leading (indicates data integrity concerns)

**What We're Measuring**:
Frequency of `shark validate` command usage relative to sync operations.

**Target**: <10% validation-to-sync ratio (most syncs trusted without explicit validation)

**Justification**: Low ratio indicates high confidence in scan accuracy; spikes indicate potential bugs needing investigation.

---

### Metric 8: Manual Correction Rate

**Type**: Lagging (measures import quality)

**What We're Measuring**:
Number of manual database edits or file renames required post-import to correct scan mistakes.

**Target**: <5 manual corrections per 100 imported files

**Justification**: Measures end-to-end import quality including edge cases not caught in primary metrics.

---

## Success Criteria Summary

The intelligent scanning epic is considered **successful** if:

1. **Import success rate ≥90%**: Vast majority of existing markdown files import without manual intervention
2. **Incremental sync <5 seconds for 100 files**: Agent session overhead minimized
3. **Conflict resolution accuracy ≥95%**: Automated strategies align with user intent
4. **Parse error rate <5%**: Scanner handles real-world file variations robustly
5. **No critical regressions**: Existing E04-F07 functionality maintains ≥98% reliability

---

## Measurement Plan

### Phase 1: Internal Dogfooding (Weeks 1-2)
- Use intelligent scanner on shark project itself
- Measure import success rate, sync performance, parse errors
- Identify and fix critical bugs before external release

### Phase 2: Early Adopter Beta (Weeks 3-6)
- Recruit 5-10 early adopter projects with varied documentation styles
- Collect detailed scan reports and user feedback
- Measure all primary metrics weekly
- Iterate on pattern matching and conflict resolution

### Phase 3: General Availability (Weeks 7+)
- Public release with telemetry (opt-in)
- Monthly metric reviews and trend analysis
- Quarterly feature refinement based on adoption patterns

### Instrumentation Requirements

Implement the following logging/telemetry:
- Scan report generation with detailed metrics (files scanned, matched, skipped, reasons)
- CLI timing instrumentation for sync performance tracking
- Conflict log with resolution strategy and outcomes
- Parse error tracking with file paths and error types
- Configuration usage tracking (special epic types, validation levels, resolution strategies)

All telemetry must be opt-in and privacy-preserving (no file content, only metadata and counts).

---

*See also*: [Requirements](./requirements.md)
