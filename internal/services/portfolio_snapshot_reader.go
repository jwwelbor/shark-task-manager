package services

import (
	"context"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
)

// PortfolioSnapshotSource loads all database-backed portfolio advice evidence
// in one round trip.
type PortfolioSnapshotSource interface {
	ReadSnapshot(ctx context.Context) (portfoliorepo.Snapshot, error)
}

// PortfolioClaimFilter applies configured lease expiry without another
// database read.
type PortfolioClaimFilter interface {
	FilterActiveReadOnly(claims []*models.EntityClaim, evaluatedAt time.Time) []*models.EntityClaim
}
