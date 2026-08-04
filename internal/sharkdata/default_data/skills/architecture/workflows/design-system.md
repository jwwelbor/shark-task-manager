---
inputs:
  - business_requirements: structured business objectives and goals
  - functional_requirements: list of functional requirements
  - non_functional_requirements: structured {performance, scale, compliance, etc.}
  - budget_constraints: structured {monthly, per-transaction, optimization_priority}
  - team_capabilities: structured {size, expertise, devops_maturity}
  - existing_constraints: list of legacy systems / existing infrastructure constraints
  - integration_requirements: list of integrations (legacy, third-party, data migration)
  - project_stage: enum {mvp, prototype, production, enterprise}
  - integration_patterns_path: absolute path to integration-patterns.md context
  - system_design_path: absolute path where system-design.md should be written
  - design_decisions_path: absolute path where system-design-design-decisions.md should be written
  - operations_guide_path: absolute path where system-design-operations-guide.md should be written
  - implementation_roadmap_path: absolute path where system-design-implementation-roadmap.md should be written
outputs:
  - system_design_doc: structured markdown written to system_design_path
  - design_decisions_doc: structured markdown written to design_decisions_path
  - operations_guide_doc: structured markdown written to operations_guide_path
  - implementation_roadmap_doc: structured markdown written to implementation_roadmap_path
  - waf_assessment: structured assessment across {security, reliability, performance, cost, ops}
  - tech_stack: structured technology recommendations
  - cost_estimate: structured monthly cost projection with breakdown
  - risks: list of {risk, likelihood, impact, mitigation}
  - open_questions: list of unresolved decisions / assumptions / risks needing user input
---

# System Architecture Design Workflow (craft)

This workflow guides you through creating high-level system architecture for complex features or applications. It is used for holistic system design across multiple components and domains.

## CRITICAL: No Implementation Code

This workflow produces DESIGN SPECIFICATIONS, not code. Use prose, tables, and Mermaid diagrams only.

- NEVER include Go, Python, TypeScript, SQL, or any language code blocks
- Describe architectures, interfaces, and data flows in prose
- The ONLY acceptable code fences are ```mermaid for diagrams
- Target 150-200 lines per document. Over 250 means you are over-specifying.

## When to Use This Workflow

Use this workflow for:
- New applications or major systems (not single features)
- Multi-component features requiring multiple services
- System-wide architectural decisions
- Cross-domain integration architecture
- Technology stack selection
- Well-Architected Framework (WAF) assessments

For single-domain features, use specialized workflows:
- design-backend.md for API/backend-only features
- design-frontend.md for UI-only features
- design-database.md for data-only features

## Step 1: Clarify Requirements

### Critical Requirements to Gather

Work with the user to understand:

**Performance & Scale**:
- Expected user load (concurrent users, requests/second)
- Data volume (initial, growth rate)
- Response time requirements (SLA targets)
- Geographic distribution
- RTO/RPO targets (recovery time/point objectives)

**Security & Compliance**:
- Regulatory requirements (GDPR, HIPAA, SOC2, etc.)
- Data residency requirements
- Authentication/authorization needs
- Encryption standards
- Audit requirements

**Budget & Cost**:
- Monthly budget constraints
- Cost per transaction targets
- Cost optimization priority level
- Reserved capacity vs. on-demand

**Operational Capabilities**:
- Team size and expertise
- DevOps maturity (manual, automated, full CI/CD)
- Monitoring and observability capabilities
- On-call support availability

**Integration Requirements**:
- Legacy systems to integrate with
- Third-party APIs
- Data migration needs
- Existing infrastructure constraints

**Project Stage**:
- MVP / Prototype
- Production-ready application
- Enterprise-scale system

Build requirements collaboratively - one section at a time to avoid overwhelming the user.

## Step 2: Research Best Practices

### Use Available Tools to Research

Never rely solely on training data. Search for:
- Service-specific best practices and current limitations
- Reference architectures for similar use cases
- Current pricing and service quotas
- Security and compliance guidance
- Performance benchmarks
- Well-Architected Framework guidance for this use case

**Cite your sources** - reference specific documentation and architecture patterns.

## Step 3: Assess Well-Architected Framework Pillars

For every architectural decision, evaluate against all five WAF pillars:

### Security
- Identity and access management approach
- Data protection (encryption at rest/transit/in use)
- Network security (VPCs, security groups, zero-trust)
- Threat detection and response
- Compliance controls
- Governance and policies

### Reliability
- Resiliency patterns (circuit breakers, retries, timeouts)
- High availability (multi-AZ, multi-region)
- Disaster recovery (backup, restore, failover)
- Fault isolation
- Monitoring and alerting
- Change management

### Performance Efficiency
- Scalability patterns (horizontal/vertical, auto-scaling)
- Capacity planning
- Caching strategies (CDN, application cache, database cache)
- Database optimization
- Compute selection (right-sizing instances)
- Network optimization

### Cost Optimization
- Resource right-sizing recommendations
- Reserved capacity vs. on-demand
- Spot instances where applicable
- Storage tiering and lifecycle policies
- Cost monitoring and governance
- Waste elimination

### Operational Excellence
- Infrastructure as Code (Terraform, CloudFormation, etc.)
- CI/CD pipelines
- Observability (structured logging, metrics, distributed tracing)
- Automation (self-healing, auto-remediation)
- Change management processes
- Incident response procedures

## Step 4: Perform Trade-Off Analysis

### Identify Primary Optimization Pillar

Which WAF pillar is the highest priority?
- Optimizing for cost? → Simpler, cheaper services
- Optimizing for reliability? → Multi-region, redundancy
- Optimizing for performance? → Premium tiers, caching
- Optimizing for security? → Additional controls, compliance
- Balancing all? → Well-rounded approach with trade-offs

### Document Trade-Offs

For each architectural decision:
- State what is being optimized
- State what is being sacrificed or de-prioritized
- Quantify when possible (e.g., "multi-region adds 2x cost but reduces RTO to seconds")
- Present alternatives with their respective trade-offs

### Example Trade-Off Documentation

**Decision**: Use multi-region active-active deployment
**Optimizes**: Reliability (RTO near zero) and Performance (global latency reduction)
**Trade-offs**:
- Cost: ~2x infrastructure cost
- Complexity: Data consistency challenges, more complex deployments
- Alternative: Single-region with standby (lower cost, longer RTO)

## Step 5: Design System Architecture

### High-Level Architecture Diagram

Create a Mermaid diagram showing:
- User-facing components (web, mobile, API)
- Application tier (services, functions, containers)
- Data tier (databases, caches, queues, storage)
- External integrations (third-party services, legacy systems)
- Infrastructure (load balancers, CDN, DNS)
- Cross-cutting concerns (monitoring, logging, security)

### Component Description

For each major component:
- **Purpose**: What it does
- **Technology Choice**: Specific service/technology and why
- **Scaling Strategy**: How it scales
- **High Availability**: Redundancy approach
- **Security**: Key security measures
- **Cost Implications**: Approximate cost contribution

### Data Flow

Create sequence diagrams for:
- Primary user flows
- Critical background processes
- Integration flows with external systems

### Integration Architecture

Document:
- Service-to-service communication (REST, gRPC, messaging)
- Event-driven patterns (pub/sub, event streams)
- API gateway patterns
- Service mesh if applicable

Apply patterns from `integration_patterns_path`.

## Step 6: Select Technology Stack

### Recommended Tech Stack

Based on requirements, project stage, and user preferences:

**Compute**:
- Service type (serverless, containers, VMs)
- Specific service recommendations
- Auto-scaling configuration

**Storage & Data**:
- Database type (relational, document, key-value, graph)
- Specific database service
- Caching strategy and service
- File/object storage
- Data warehouse/analytics if needed

**Networking**:
- Load balancing approach
- CDN strategy
- DNS management
- VPC/network segmentation

**Security**:
- Identity provider
- Secrets management
- Key management
- Security scanning tools

**Observability**:
- Logging aggregation
- Metrics and dashboards
- Distributed tracing
- Alerting

**CI/CD**:
- Source control
- Build pipeline
- Deployment automation
- Testing framework

Apply patterns from project constraints and user expertise.

## Step 7: Address Multi-Region Strategies

If multi-region is required:

### Deployment Pattern
- **Active-Active**: Both regions serve traffic (best RTO, highest cost)
- **Active-Passive**: One region standby (medium RTO, medium cost)
- **Backup Only**: Disaster recovery only (longest RTO, lowest cost)

### Data Replication
- Synchronous replication (strong consistency, latency impact)
- Asynchronous replication (eventual consistency, better performance)
- Conflict resolution strategy

### Failover Mechanisms
- DNS-based failover
- Load balancer-based failover
- Application-level failover
- RTO and RPO targets

### Global Traffic Management
- Geographic routing
- Latency-based routing
- Health-check based routing

## Step 8: Define Observability Strategy

### Logging
- Structured logging format (JSON)
- Correlation IDs for distributed tracing
- Log aggregation service
- Retention policies
- Log levels and filtering

### Metrics
- Infrastructure metrics (CPU, memory, disk, network)
- Application metrics (request rate, error rate, latency)
- Business metrics (conversions, transactions)
- Custom metrics per component
- Dashboard design

### Tracing
- Distributed tracing for microservices
- Trace sampling strategy
- Service dependency mapping

### Alerting
- Alert conditions and thresholds
- Severity levels
- Notification channels
- Escalation procedures
- Runbooks for common alerts

## Step 9: Plan Cost Optimization

### Cost Estimation

Provide projected costs with breakdown:
- Compute costs (instances, functions, containers)
- Storage costs (database, object storage, backups)
- Network costs (data transfer, load balancing)
- Third-party service costs
- Total monthly estimate

### Cost Optimization Strategies
- Right-sizing recommendations (specific instance types)
- Reserved capacity for predictable workloads
- Spot instances for fault-tolerant workloads
- Storage tiering and lifecycle policies
- Cost allocation tags for chargeback
- Waste identification (idle resources, over-provisioning)

## Step 10: Identify Risks & Mitigations

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation Strategy |
|------|------------|--------|---------------------|
| Database bottleneck | Medium | High | Read replicas, caching, query optimization |
| Third-party API downtime | High | Medium | Circuit breakers, fallback mechanisms, caching |
| Cost overrun | Medium | Medium | Cost monitoring, alerts, resource quotas |
| Data breach | Low | High | Encryption, access controls, security scanning |

### Mitigation Priorities
Focus on high-impact, high-likelihood risks first.

## Step 11: Create Implementation Roadmap

### Phased Approach

Break implementation into phases:

**Phase 1: MVP / Foundation**
- Core functionality
- Essential services
- Basic monitoring
- Single region
- Manual deployment

**Phase 2: Production Hardening**
- High availability
- Automated deployment
- Comprehensive monitoring
- Security hardening
- Performance optimization

**Phase 3: Scale & Optimize**
- Multi-region (if needed)
- Advanced caching
- Cost optimization
- Advanced monitoring
- Auto-scaling tuning

**Phase 4: Enterprise Features**
- Compliance certifications
- Advanced security
- Disaster recovery testing
- Capacity planning
- Full automation

### Milestones
Define clear milestones with success criteria for each phase.

## Step 12: Write Architecture Documents

Generate four documents:

### Document 1: System Architecture (Required if creating system design)

Write to `system_design_path`.

Sections:
- Executive Summary (business context, key decisions)
- Requirements (functional, non-functional, priorities)
- Architecture Overview (diagrams, components, interactions)
- Links to other three documents

### Document 2: Design Decisions

Write to `design_decisions_path`.

Sections:
- Key architectural decisions with rationale
- Alternatives considered and why rejected
- Trade-off analysis for each decision
- WAF pillar assessment
- Recommended technology stack

### Document 3: Operations Guide

Write to `operations_guide_path`.

Sections:
- Monitoring and alerting setup
- Incident response procedures
- Disaster recovery procedures
- Backup and restore procedures
- Cost monitoring and optimization
- Runbooks for common scenarios

### Document 4: Implementation Roadmap

Write to `implementation_roadmap_path`.

Sections:
- Cloud provider recommendations (AWS, Azure, GCP, or multi-cloud)
- Compute, storage, networking approach
- Phased implementation plan
- Milestones with success criteria
- Dependencies and blockers
- Resource requirements (team, budget, time)

## Step 13: Quality Assurance Checklist

Before finalizing, verify:

### Research & Requirements
- [ ] Searched for current best practices using tools
- [ ] Clarified all critical requirements
- [ ] Documented project stage (MVP vs. Enterprise)
- [ ] Collected tech stack preferences and expertise

### WAF Assessment
- [ ] Security pillar assessed and addressed
- [ ] Reliability pillar assessed and addressed
- [ ] Performance Efficiency pillar assessed and addressed
- [ ] Cost Optimization pillar assessed and addressed
- [ ] Operational Excellence pillar assessed and addressed

### Trade-offs & Decisions
- [ ] Primary optimization pillar stated
- [ ] All trade-offs explicitly documented
- [ ] Alternatives presented where applicable
- [ ] Quantified impacts where possible

### Actionability
- [ ] Specific service names and configurations provided
- [ ] Not generic terms like "use a database"
- [ ] Implementation guidance is concrete
- [ ] Reference architectures cited
- [ ] Documentation sources referenced

### Completeness
- [ ] All four documents created
- [ ] Architecture diagrams included
- [ ] Cost estimates provided
- [ ] Risks and mitigations documented
- [ ] Implementation roadmap defined
- [ ] Observability strategy complete

## Communication Principles

- **Be specific**: Use exact service names, not generic terms
- **Explain why**: Every recommendation has rationale
- **Present options**: When multiple valid approaches exist
- **Use diagrams**: When they improve clarity
- **Acknowledge uncertainty**: Be honest about unknowns
- **Validate understanding**: Ensure user understands implications
- **Document decisions**: Save recommendations for other agents to reference

## Record material design Questions

When a cross-domain architecture decision remains materially unresolved, use
`skills/question-management/SKILL.md` to create or reuse a linked Q###. Record
a non-material rationale in the design document instead. Do not treat the
absence of `TBD` text as decision closure.

## MANDATORY: Interactive Review of Open Questions

After generating the document(s), you MUST surface any open questions, unresolved decisions, concerns, or assumptions to the user **before proceeding to the next workflow step**. Do NOT silently move on.

### Process

1. **Scan** the completed document for:
   - Open questions or items needing stakeholder input
   - Unresolved design decisions with trade-offs
   - Assumptions that need validation
   - Risks or concerns that require discussion
   - Items marked as "TBD", "TODO", or "deferred"
   - Anything in the "Open Questions & Decisions Required" section

2. **Present** a structured summary to the user:
   - List each item clearly with its context
   - For decisions, present options with trade-offs
   - For assumptions, state what you assumed and ask for confirmation
   - For risks, explain impact and ask for guidance

3. **Walk through** each item interactively:
   - Discuss one item at a time
   - Record the resolution in the document
   - Update the document with decisions made

4. **Only proceed** when all items are resolved or explicitly deferred by the user.

If there are no open questions, confirm this explicitly: "All design decisions are resolved. No open questions remain."

---

## Output Requirements

Upon completion, you will have:
1. Four comprehensive architecture documents
2. Complete system architecture with diagrams
3. Technology stack recommendations
4. WAF pillar assessment
5. Trade-off analysis for all decisions
6. Cost estimates and optimization strategies
7. Risk assessment and mitigations
8. Phased implementation roadmap (if applicable)
9. Operational procedures (if applicable)
10. All decisions documented and justified
11. All open questions reviewed interactively and resolved
