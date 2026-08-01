# Stakeholder Roles

**Epic**: [Question and Decision Workflow Management](./epic.md)

E39 serves existing Shark users and agents in four distinct roles. These are
role descriptions for this workflow, not new global personas.

## Requester

Creates a Question after discovering an uncertainty that needs an accountable
answer. The requester needs to see whether the Question is open, who owns the
next response, which work it affects, and where the final answer landed.

## Requested Responder

Provides one bounded perspective when serially routed to the Question. The
responder needs a scoped prompt and evidence destination, not the burden of
recovering every historical report or deciding linked-work status.

## Resolution Owner

Determines whether the answer is a local clarification, product decision,
feature contract, architectural policy, or follow-up work. This role is
responsible for the resolution classification and authoritative record pointer.

## Linked-Work Owner

Owns a feature, task, bug, change card, tech-debt item, or epic affected by a
Question. This role needs a precise gate: pause only when a linked blocking
Question remains open, and continue otherwise.

*See also*: [User Journeys](./user-journeys.md) and [Requirements](./requirements.md).

