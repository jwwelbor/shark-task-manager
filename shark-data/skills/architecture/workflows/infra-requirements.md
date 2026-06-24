---
inputs:
  - dev_ready_package_paths: list of paths to developer-ready package documents
  - security_design_path: absolute path to 06-security-design.md (optional, if available)
  - backend_design_path: absolute path to 04-backend-design.md
  - infra_requirements_path: absolute path where infrastructure requirements summary should be written
  - compute_needs_path: absolute path where compute requirements should be written
  - storage_needs_path: absolute path where storage requirements should be written
  - arch_approval_path: absolute path where infrastructure architecture approval should be written
  - infra_review_path: absolute path where infrastructure implementation review should be written
  - mode: enum {requirements_analysis, architecture_review, implementation_review} — which review variant to run
outputs:
  - infra_requirements_doc: structured markdown written to infra_requirements_path (mode=requirements_analysis)
  - compute_needs_doc: structured markdown written to compute_needs_path (mode=requirements_analysis)
  - storage_needs_doc: structured markdown written to storage_needs_path (mode=requirements_analysis)
  - arch_approval_doc: structured markdown written to arch_approval_path (mode=architecture_review)
  - infra_review_doc: structured markdown written to infra_review_path (mode=implementation_review)
  - approval_verdict: enum {approved, approved_with_changes, rejected}
  - required_changes: list of changes required (if not approved)
---

# Infrastructure Requirements & Architecture Review Workflow (craft)

**When**: Feature needs infrastructure planning or infrastructure implementation needs review.

## Process: Requirements Analysis

### 1. Determine Compute Needs

- Expected traffic/load (requests/sec, concurrent users)
- Processing requirements (CPU-bound, memory-bound, IO-bound)
- Scaling strategy (horizontal, vertical, auto-scaling triggers)

### 2. Determine Storage Needs

- Database requirements (type, estimated size, IOPS)
- File/object storage needs (uploads, generated content)
- Caching requirements (session, query cache, CDN)
- Backup and retention policy

### 3. Determine Networking Needs

- Public vs. private resources
- VPC/network segmentation
- Load balancing (ALB, NLB, reverse proxy)
- CDN requirements for static assets

### 4. Document with Rationale

- Each requirement should include why it's needed and how it was sized

## Process: Architecture Review

### 1. Verify Alignment

- Infrastructure design aligns with system architecture
- Security requirements are met (encryption, network isolation, access control)
- Technology choices are appropriate for the workload

### 2. Validate Patterns

- Standards and patterns followed (IaC, naming conventions, tagging)
- Scalability and performance characteristics acceptable
- Error handling and resilience (health checks, failover, monitoring)

### 3. Document Approval or Changes

- Approve design, or list required changes with rationale

## Output

When `mode == requirements_analysis`:

- `infra_requirements_path` — Infrastructure requirements summary
- `compute_needs_path` — Compute requirements (load, processing, scaling)
- `storage_needs_path` — Storage and database requirements (type, size, IOPS, backup, retention)

When `mode == architecture_review`:

- `arch_approval_path` — Infrastructure architecture approval (verdict + required changes)

When `mode == implementation_review`:

- `infra_review_path` — Infrastructure implementation review (verdict + required changes)
