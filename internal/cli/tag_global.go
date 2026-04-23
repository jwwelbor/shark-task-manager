package cli

import (
	"context"
	"fmt"

	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// GetTagService returns a *TagService wired to the current project's DB and
// maintainer gate.
//
// The accessor creates a new service instance on every call (matching the
// "new instance per call" pattern of GetTaskService and GetMaintainerGate).
// The global DB connection is shared; the gate is a fresh *maintainer.FileGate
// per F02's "new instance per call" contract.
//
// Panics on DB failure to match the behaviour of GetTaskService (fail-fast for
// CLI entry points).
//
// Usage:
//
//	svc := cli.GetTagService()
//	tags, err := svc.ListTags(cmd.Context())
//
// Spec reference: spec.md REQ-F-012, §2.5 (accessor wiring), AC-12.
func GetTagService() *services.TagService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for tag service: %v", err))
	}

	tagRepo := tagrepo.NewTagRepository(db)
	entityTagRepo := tagrepo.NewEntityTagRepository(db)
	gate := GetMaintainerGate() // new instance per call, per F02 contract

	return services.NewTagService(tagRepo, entityTagRepo, gate)
}
