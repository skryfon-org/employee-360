package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	alimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/auditlog"
	aluc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auditlog"
)

// AuditLogContainer wires the read-only audit-log viewer.
type AuditLogContainer struct {
	Repo repository.AuditLogQueryRepository

	ListUseCase aluc.ListAuditLogsUseCase
	GetUseCase  aluc.GetAuditLogUseCase

	Handler *handlers.AuditLogHandler
}

// NewAuditLogContainer initializes the audit-log repository, usecases and handler.
func NewAuditLogContainer(db *gorm.DB) (*AuditLogContainer, error) {
	repo := persistence.NewGormAuditLogQueryRepository(db)
	listUC := alimpl.NewListAuditLogsUseCase(repo)
	getUC := alimpl.NewGetAuditLogUseCase(repo)
	return &AuditLogContainer{
		Repo:        repo,
		ListUseCase: listUC,
		GetUseCase:  getUC,
		Handler:     handlers.NewAuditLogHandler(listUC, getUC),
	}, nil
}
