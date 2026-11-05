# Ubiquitous language

## Context: Shark orchestration

| Term | Definition | Example | Do not confuse with |
| --- | --- | --- | --- |
| Shark project root | The directory that owns the Shark project's configuration and project-relative records. The parent coordinator uses it to locate its state and documents. | The scratch project that owns the benchmark database. | Worker working directory; Git repository root. |
| Worker working directory | The directory in which an agent process starts. `shark run --workdir` selects this directory for the worker. | A benchmark fixture checkout passed as `--workdir`. | Shark project root; a claim or an evidence location. |
| Git repository root | The top-level directory of the Git worktree containing a candidate. | The result of `git rev-parse --show-toplevel` in a worker checkout. | A nested Shark project root. |
| Git worktree | A separate checkout attached to a Git repository, with its own working directory and shared repository history. | A checkout created for a bug run. | The parent Shark project or an unrelated checkout. |
| Project document | A project artifact under `docs/plan` that carries durable requirements, research, plans, or decisions. | A bug research report or repair plan. | A local review artifact. |
| Local review artifact | Review evidence kept for local inspection, such as content under `docs/review`, `docs/reviews`, or `dev-artifacts`. It may be copied to the parent project for retention. | A candidate-matched review report. | A committed project document. |
| Approved plan revision | The exact repair-plan content approved by independent plan review, identified by its digest. | The plan revision consumed by development and review. | A later edit to the working plan. |

Project documents are committed and must not be Git-ignored. New local review
artifacts are not part of commit history. Existing tracked historical review
files remain tracked unless explicitly changed.

The worker working directory and Shark project root can differ. A run must
identify the project and evidence locations explicitly; changing the worker's
directory does not move its database or make a worker-local path available in
the parent project. Support for unrelated project and checkout roots is a
separate integration contract.
