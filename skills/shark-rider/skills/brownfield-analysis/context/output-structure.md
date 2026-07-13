# Output Structure

All brownfield-analysis documentation is written to `docs/brownfield-docs/` in the workspace root.
This file defines the canonical directory layout and the purpose of each document. Treat it
as a template: omit any directory whose subject the project does not contain.

## Directory tree

```
docs/brownfield-docs/
├── README.md                  # Master navigation hub
├── project-overview.md        # Executive summary
├── technical-debt-report.md   # Executive technical-debt summary
│
├── architecture/
│   ├── system-overview.md     # Architecture style, deployment topology, diagrams
│   ├── components.md          # Major components with responsibilities
│   ├── dependencies.md        # Internal + external dependency matrix
│   └── patterns.md            # Design patterns with evidence
│
├── reference/
│   ├── program-structure.md   # Complete file inventory by package
│   ├── interfaces.md          # All public interfaces and contracts
│   ├── data-models.md         # All data models, entities, value objects
│   └── api-reference.md       # REST/API endpoint documentation
│
├── behavior/
│   ├── business-logic.md      # Business domains and rules
│   ├── workflows.md           # End-to-end process flows
│   ├── decision-logic.md      # Decision trees and business rules
│   └── error-handling.md      # Exception hierarchy, error codes
│
├── diagrams/
│   ├── structural/
│   │   ├── component-diagram.md
│   │   ├── class-diagrams.md
│   │   └── package-dependencies.md
│   ├── behavioral/
│   │   ├── sequence-diagrams.md
│   │   └── activity-diagrams.md
│   ├── data-flow/
│   │   ├── data-sync-flow.md
│   │   └── request-flow.md
│   └── architecture/
│       ├── deployment-diagram.md
│       ├── cicd-pipeline.md
│       └── environment-topology.md
│
├── technical-debt/
│   ├── summary.md
│   ├── outdated-components.md
│   ├── security-vulnerabilities.md
│   ├── maintenance-burden.md
│   └── remediation-plan.md
│
├── analysis/
│   ├── code-metrics.md
│   ├── complexity-analysis.md
│   ├── dependency-analysis.md
│   └── security-patterns.md
│
├── migration/
│   ├── component-order.md
│   ├── test-specifications.md
│   └── validation-criteria.md
│
└── specialized/
    ├── database/
    │   └── [db-type]-schema.md
    └── infrastructure/
        ├── cicd-pipeline.md
        ├── kubernetes-deployment.md
        └── environment-configuration.md
```
