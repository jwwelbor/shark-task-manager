# Skills Index

This index is the contributor-facing map for the embedded skill bundle. It answers "where do I edit?" without requiring a scan through every `SKILL.md`.

| Skill | Holds | Primary workflows | Consumed by |
|---|---|---|---|
| `architecture` | System design patterns, ADR templates, backend, database, frontend, security workflows | `workflows/design-backend.md`, `workflows/design-database.md`, `workflows/design-frontend.md`, `workflows/feasibility-review.md` | Architecture phases in `workflow/{epic,feature}.yaml`, architect-style agents |
| `assessment` | Complexity triage, scope validation, readiness checks, effort estimation | `workflows/complexity-triage.md`, `workflows/scope-validation.md`, `workflows/readiness-check.md`, `workflows/effort-estimation.md` | Feature assessment prompt, workflow routing via `assessment` slug |
| `breakdown-test` | Requirement decomposition and test-matrix generation | `workflows/decompose-requirement.md`, `workflows/build-test-matrix.md` | Test decomposition and requirements analysis flows |
| `clarification` | Ambiguity detection, question ladders, surfaced assumptions | `workflows/detect-ambiguity.md`, `workflows/question-ladder.md`, `workflows/surface-assumptions.md` | Clarification and refinement work before specs are locked |
| `content-validation` | Content-quality validation rubric | `workflows/validate-content-quality.md` | Content validation passes |
| `cross-artifact-analysis` | Traceability, terminology alignment, spec-drift detection | `workflows/acceptance-coverage-trace.md`, `workflows/detect-spec-drift.md`, `workflows/terminology-alignment.md` | Drift and cross-doc consistency checks |
| `debugging` | Frontend, backend, test, web, and devops debugging workflows | `workflows/debug-backend.md`, `workflows/debug-frontend.md`, `workflows/debug-tests.md`, `workflows/debug-web.md` | Debugging tasks and issue analysis |
| `demo-script` | Portable, evidence-based demo scenario maps and readiness classification | `context/demo-script-template.md` | The explicit `/shark-rider demo` procedure |
| `feature-design` | Feature wireframes and prototypes | `workflows/wireframes.md`, `workflows/prototype.md` | Feature-design phases and frontend prep |
| `frontend-design` | Aesthetic direction and refinement heuristics | `workflows/commit-to-aesthetic-direction.md`, `workflows/refine-aesthetics.md` | Frontend design polish and UI iteration |
| `implementation` | Backend, frontend, API, database, and test implementation guidance | `workflows/implement-backend.md`, `workflows/implement-frontend.md`, `workflows/implement-api.md`, `workflows/implement-tests.md` | Development phases, implementation agents |
| `overconfidence-prevention` | Confidence calibration workflow | `workflows/calibrate-confidence.md` | High-risk reasoning or validation passes |
| `product-design` | Product discovery and D01-D14 design workflow | `workflows/d01-vision.md`, `workflows/d06-user-insights.md`, `workflows/d14-validated-designs.md` | Product-design and discovery phases |
| `quality` | Design validation, task validation, test planning, code review, QA | `workflows/validate-design.md`, `workflows/validate-tasks.md`, `workflows/test-planning.md`, `workflows/review-code.md`, `workflows/qa-testing.md` | QA prompt, code-review prompt, quality gates in workflow YAML |
| `research` | Codebase analysis, filesystem mapping, dependency tracing, feature understanding | `workflows/analyze-codebase.md`, `workflows/map-filesystem.md`, `workflows/find-patterns.md`, `workflows/trace-dependencies.md`, `workflows/understand-feature.md` | Research phases in workflow YAML, research agents, brownfield prep |
| `solution-walkthrough` | Recommendation-first, decision-by-decision solution walkthroughs with durable record routing | `context/decision-record-template.md` | The explicit `/shark-rider walkthrough` procedure |
| `specification-writing` | Epic docs, feature PRDs, task-generation craft | `workflows/write-epic.md`, `workflows/write-feature-prd.md`, `workflows/write-task.md`, `workflows/decompose-epic.md` | BA, planning, and task-generation phases in workflow YAML |
| `sprint-analytics` | Sprint retrospectives | `workflows/retro-sprint.md` | Sprint closeout |
| `sprint-execution` | Solo and team sprint runbooks | `workflows/run-sprint.md`, `workflows/run-sprint-team.md` | Sprint execution orchestration |
| `sprint-planning` | Sprint planning and readiness | `workflows/plan-sprint.md` | Sprint planning flows |
| `test-driven-development` | TDD and debugging references | `references/debugging-workflow.md` | TDD-oriented implementation and quality phases |
| `uat` | User-acceptance testing and red-team rubrics | `references/redteam-rubric.md`, `references/uat-template.md` | UAT phases and final acceptance review |

Use per-skill `README.md` files for deeper structure. Edit `SKILL.md` when routing or contracts change, `workflows/` when procedure changes, and `context/` or `references/` when rubrics, templates, or examples change.
