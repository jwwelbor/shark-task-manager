package viewer

// This file re-exports the response DTO types from the services package as
// type aliases so that handler tests and the spec can reference them from the
// api/viewer package without needing to import services directly.
//
// The actual type definitions live in internal/services/viewer_service.go
// where the concrete ViewerService produces them. The handler only consumes
// these types; it never constructs them directly.
//
// No new types are declared here — all response shapes come from the service layer.
// The handler imports services.SummaryResponse etc. directly from the services package.
