# Code Reference

## Purpose

Create a complete inventory of the codebase: every file cataloged, every public interface
documented, every data model mapped. For large codebases this is the most time-consuming
area — work package by package.

## Program structure

In `reference/program-structure.md`, organize the complete file inventory by package/module.
For each package:

- Package path and purpose
- Every source file with a one-line description
- Key classes/modules within each file
- Approximate complexity level (simple utility, complex business logic, etc.)

## Interfaces

In `reference/interfaces.md`, document all public interfaces and contracts:

- API interfaces — REST controllers, GraphQL resolvers, gRPC services
- Internal interfaces — service contracts, repository interfaces, event handlers
- Remote interfaces — EJB, RMI, SOAP, for legacy systems
- Message contracts — queue/topic message formats

For each: name, methods/endpoints, parameters, return types, and purpose.

## Data models

In `reference/data-models.md`, catalog all data models, grouped by business domain:

- Entities / domain objects
- Value objects / DTOs
- Database models (ORM-mapped classes)
- API request/response models
- Event/message payloads

For each model: name, fields with types, relationships to other models, validation rules,
and which domain it belongs to.

## API reference

In `reference/api-reference.md`, document all API endpoints:

- HTTP method, path, purpose
- Request parameters, headers, body format
- Response format and status codes
- Authentication/authorization requirements
- Rate limiting or other constraints

Document internal service-to-service APIs as well, if the project has them.
