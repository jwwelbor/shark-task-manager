package repository

import (
	advanceguardrepo "github.com/jwwelbor/shark-task-manager/internal/repository/advanceguard"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

func NewAdvanceGuardRepository(db *dbconn.DB) *advanceguardrepo.Repository {
	return advanceguardrepo.NewRepository(db)
}
