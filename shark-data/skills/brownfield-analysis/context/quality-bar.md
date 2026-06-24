# Quality Bar — Worked Example

The documentation you produce should read like a senior architect's project handoff. This
file makes that standard concrete with a worked example, so you can calibrate depth and
specificity.

## The standard

Someone reading the documentation with no prior context should be able to:

1. Understand what the system does and why it exists.
2. Navigate the codebase confidently.
3. Identify the riskiest areas — technical debt, security, complexity.
4. Plan a modernization or migration effort.
5. Onboard new team members effectively.

## Worked example: a 1,564-file Java EE enterprise system

A gold-standard analysis of a financial data repository — running on JBoss/WildFly with Oracle
and MongoDB, a Swing desktop client, and Kubernetes deployment — produced 42 documents. It
included:

- **Project overview** with a complete file inventory (1,564 Java, 285 XML, 42 SQL files), a
  module dependency graph, and a full technology stack table with pinned versions drawn from
  the actual `pom.xml`.
- **Architecture docs** with diagrams of the multi-tier topology (client → app server → data
  tier), the deployment architecture (pods, autoscaling, config maps), and a communication
  patterns matrix (EJB remoting, JMS, JDBC, REST, cloud SDK calls).
- **Code reference** covering the structural hierarchy of every package, all public service
  interfaces and REST endpoints, data models with ORM annotations, and API contracts.
- **Behavior analysis** documenting business logic (statement management, calculation engines,
  template hierarchies), workflow process flows, decision trees with branching rules, and
  error handling with exception hierarchies.
- **Visual documentation** with 10+ Mermaid diagrams: component diagrams, class hierarchies,
  package dependency graphs, sequence diagrams for key flows, activity diagrams, data sync
  flows, request flows, deployment topology, and CI/CD pipeline diagrams.
- **Technical debt assessment** that identified 20 issues across three severity levels —
  including CVE-bearing libraries, end-of-life SDKs, hardcoded credentials, insecure artifact
  downloads, and framework version inconsistencies — each with file paths, current-vs-latest
  versions, and impact.
- **Code quality analysis** with complexity metrics, dependency health scores, and a security
  pattern review.
- **Migration readiness** with a dependency-ordered component migration sequence, test
  specifications per component, and validation/acceptance criteria.
- **Specialized docs** for database schemas (stored procedures, document collections with ORM
  mappings) and infrastructure (CI/CD workflows, orchestration manifests, a multi-environment
  configuration matrix).
- **An executive technical-debt report** leading with the single highest-impact transformation
  recommendation, followed by critical/high/medium issue summaries.

## What the bar means in practice

- Every claim is backed by a file path.
- Every dependency is pinned to its actual version from the build file.
- Every diagram reflects the real topology, not an idealized version.
- Severity ratings are grounded: "Critical" means something exploitable or broken now, not
  merely unpleasant.
- The reader can act on the output without needing to re-examine the codebase themselves.

## Signs of insufficient depth

- Dependencies listed without versions.
- Patterns mentioned without file paths ("uses Repository pattern" — where?).
- Diagrams that show what should be there rather than what is.
- Security findings without CVE numbers or affected version ranges.
- A remediation plan whose items cannot be independently executed (too vague).
- An executive summary that requires reading the supporting docs to understand the key risks.
