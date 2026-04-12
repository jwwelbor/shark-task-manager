package main

// This file re-exports ServiceContainer and WireServices from
// internal/viewer/server so that cmd/server tests can reference them as
// package-level identifiers without the external import path.
//
// The canonical implementations live in internal/viewer/server/wire.go so that
// both cmd/server/main.go and internal/cli/commands/web.go (the shark web
// command) share identical wiring logic.

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	viewerserver "github.com/jwwelbor/shark-task-manager/internal/viewer/server"
)

// ServiceContainer is an alias for viewerserver.ServiceContainer.
// Keeping this alias here preserves backward compatibility with tests and
// any existing code that references the type as ServiceContainer within
// the cmd/server package.
type ServiceContainer = viewerserver.ServiceContainer

// WireServices is a thin wrapper around viewerserver.WireServices.
// It delegates to the canonical implementation in internal/viewer/server.
func WireServices(db *repository.DB, projectRoot string) *ServiceContainer {
	return viewerserver.WireServices(db, projectRoot)
}
