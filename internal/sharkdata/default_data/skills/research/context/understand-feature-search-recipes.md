# Understand Feature Search Recipes

Use this file when `workflows/understand-feature.md` needs the fuller search guidance and example search shapes.

## Typical searches by phase

### Feature identification

- Search by feature name and synonyms
- Search for file-name patterns
- Search tests and migrations alongside implementation

### Data flow analysis

- Read controllers, services, repositories, and DTOs
- Search validation annotations and transformation helpers
- Trace entrypoints through layer boundaries

### Business logic and dependency analysis

- Search for throws, validation checks, limits, and constants
- Inspect constructor injection and service-to-service calls
- Search event emitters and listeners

### Consumer analysis

- Search imports of feature modules
- Search frontend API usage
- Search event names and public endpoints

### Extension analysis

- Search hooks, plugins, interfaces, feature flags, and configuration branches
- Compare extension options against existing seams before proposing new modules
