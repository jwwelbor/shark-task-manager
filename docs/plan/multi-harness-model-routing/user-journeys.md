# E41 user journeys

See the [epic PRD](./epic.md#high-level-acceptance-criteria-uat-scenarios) for release scenarios and the [stakeholder responsibilities](./personas.md) for decision ownership.

## J1 Select and preview a route

The workflow author declares a high-complexity development step and previews it under two named profiles. Each preview explains the selected route and the source of the choice. If a pin conflicts with a binding or explicit effort is unsupported, the author uses the reported step and reason to correct the configuration before launching work. Covers UAT-01.

## J2 Qualify and migrate a workflow

At project setup, or on the first routing lookup when no inventory exists, Shark records installed harness executable paths and versions. The project operator can refresh that snapshot explicitly, then checks a binding for the intended account and execution mode and qualifies it in an approved disposable environment. If the executable is present but account access is unknown, the operator sees that distinction. After qualification the operator opts in one workflow, verifies its recorded attempt and practices rollback at a run boundary. Covers UAT-02 and UAT-07.

## J3 Review and recover a step

The gate owner selects a candidate and declares the required reviewer separation. Shark checks the candidate’s actual producers, selects an eligible fresh session and verifies the reviewer evidence before admitting the result. A missing executable can lead to an approved alternate. Rider may try once inline if external routes are exhausted and the current host meets every constraint. A possible write with uncertain effects pauses for the project operator to reconcile. A valid negative review follows its existing workflow outcome. Covers UAT-03 through UAT-05.

## J4 Compare two profiles

The benchmark maintainer freezes the same workflow and prompt inputs for two profile runs, then compares their selected and observed routes. E40 retains the evidence for both runs. If a rerun presents a different pinned input, it cannot enter the original comparison. Covers UAT-06.
