# Verify craft-skill purity

## Goal

Verify that E32-F09 keeps its owned embedded craft skills tool-agnostic without changing workflow and orchestration skills that own execution mechanics.

## Scope

The owned craft-skill set is `specification-writing`, `uat`, `assessment`, `research`, and `quality`. The former `triage` skill is not present in the canonical bundle. Workflow and orchestration skills, including question and sprint management, are out of scope.

## Test cases

| ID | Check | Expected result |
|---|---|---|
| TC-090 | Run `go test ./internal/sharkdata -run TestEmbedded_SkillsContainNoBareSharkCLIRefs -count=1`. | The test scans the five owned prefixes and passes. |
| TC-091 | Exercise the permanent scanner with in-scope `shark related-docs`, `shark sprint`, and `/shark-rider` fixture content. | The scanner identifies each platform reference; the policy is command-family complete rather than a partial verb list. |
| TC-092 | Run `go test ./internal/cli/commands -run TestInteractionMapTemplateIsShippedWithSpecificationWritingSkill -count=1`. | The test accepts the tool-agnostic registration guidance. |
| TC-093 | Run `make fmt`, `make lint`, and `make test`. | All checks pass, including the updated rendered-prompt golden corpus. |
| TC-094 | Run `go test ./internal/sharkdata -run TestEmbedded_OutcomeReturningCraftSkillsDeclareThreeWayOutcome -count=1`. | Assessment and quality declare the host-routable `pass` / `fail` / `blocked` outcome set. |

## Caller-path contracts

All cases are internal: each test is the production regression entrypoint for its static embedded content. No mocks apply.
