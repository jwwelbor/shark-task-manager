# Drift Taxonomy

| Dimension | Drift Pattern | Severity Cue |
|---|---|---|
| Scope | Child omits a parent requirement or adds unauthorized behavior | High when user-visible or contract-changing |
| Terminology | Same concept has different names or one name has different meanings | Medium to high when implementation could diverge |
| Acceptance criteria | Parent requirement has no validating child criterion | High when no downstream test would catch it |
| Constraints | Child relaxes performance, security, compatibility, or compliance limits | High when risk posture changes |
| Data shapes | Fields, types, identifiers, or relationships disagree | High when APIs or persistence are affected |
| Dependencies | Child assumes ordering, ownership, or prerequisites not established by parent | Medium to high based on execution risk |

Treat added detail as healthy refinement unless it contradicts or silently expands the parent contract.
