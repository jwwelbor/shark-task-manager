# Workflow Examples

End-to-end workflow examples showing how to use Shark for complete development lifecycles.

## Overview

This section provides real-world examples of using Shark from epic creation through task implementation. Examples demonstrate:

- Epic-to-feature-to-task decomposition
- Complexity-adaptive workflows (SIMPLE, STANDARD, COMPLEX)
- Agent coordination patterns
- Status transitions and workflow gates
- Progress tracking and monitoring

## Workflow Index

### Foundation Workflows

1. **[Epic Creation Through Feature Delivery](epic-to-feature.md)** - Complete flow from vision to delivered features
2. **[Simple Feature Workflow](simple-feature.md)** - SIMPLE tier (0-3 complexity score)
3. **[Standard Feature Workflow](standard-feature.md)** - STANDARD tier (4-7 complexity score)
4. **[Complex Feature Workflow](complex-feature.md)** - COMPLEX tier (8+ complexity score)

### Advanced Workflows

5. **[Multi-Agent Coordination](multi-agent.md)** - Coordinating business-analyst, architect, developer, qa agents
6. **[Parallel Feature Development](parallel-features.md)** - Managing multiple features simultaneously
7. **[Dependency Management](dependencies.md)** - Handling task dependencies and blocking
8. **[Approval and UAT Workflows](approval-uat.md)** - Human approval gates and UAT

### Troubleshooting Workflows

9. **[Handling Blockers](handling-blockers.md)** - Dealing with blocked tasks and dependencies
10. **[Rework and Rejection](rework-rejection.md)** - Handling review failures and rework cycles
11. **[Status Cascade Debugging](status-cascade.md)** - Understanding status propagation

## Coming Soon

> 🚧 **Note**: This section is under development. We'll be collaboratively building out comprehensive workflow examples with:
>
> - Step-by-step command sequences
> - Expected outputs and agent behaviors
> - Common pitfalls and solutions
> - Integration with external tools (GitHub, CI/CD, etc.)
>
> **Priority examples to develop:**
> 1. Complete epic-to-delivery workflow with screenshots
> 2. Complexity tier routing examples
> 3. Multi-agent handoff patterns
> 4. Real-world debugging scenarios

## Example Structure

Each workflow example will follow this structure:

### Scenario
Description of the use case and goals.

### Prerequisites
Required setup and starting state.

### Step-by-Step Guide
Numbered steps with commands and expected outputs.

### Key Concepts Demonstrated
What this workflow teaches.

### Common Issues
Typical problems and solutions.

### Related Workflows
Links to related examples.

## Quick Reference

For quick command lookup without full workflows:
- **[Epic Commands](../epic-cli.md)** - Epic CLI reference
- **[Feature Commands](../feature-cli.md)** - Feature CLI reference
- **[Task Commands](../task-cli.md)** - Task CLI reference

## Contributing

These workflow examples are living documentation. As you use Shark and discover useful patterns, please contribute:

1. Document your workflow
2. Include actual command outputs
3. Highlight lessons learned
4. Add troubleshooting tips

## Feedback

Found these examples helpful? Have suggestions for new workflows? Open an issue:
- GitHub Issues: https://github.com/jwwelbor/shark-task-manager/issues

---

## Next Steps

Start with the foundation workflows:
1. Read **[Epic to Feature Delivery](epic-to-feature.md)** for the complete picture
2. Choose complexity tier based on your feature:
   - **[Simple Feature](simple-feature.md)** - Quick implementations, 1-3 files
   - **[Standard Feature](standard-feature.md)** - Standard features, 4-10 files
   - **[Complex Feature](complex-feature.md)** - Complex features, 10+ files, architectural changes
3. Learn **[Multi-Agent Coordination](multi-agent.md)** for team workflows

## Related Documentation

- **[Configuration](../configuration.md)** - Understanding .sharkconfig.json
- **[Workflow Configuration](../workflow-configuration.md)** - Status flows and transitions
- **[Template System](../template-system.md)** - Agent instruction templates
