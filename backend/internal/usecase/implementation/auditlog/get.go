package auditlog

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
	aluc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auditlog"
)

type getAuditLogUseCase struct {
	repo repository.AuditLogQueryRepository
}

// NewGetAuditLogUseCase creates a new GetAuditLogUseCase.
func NewGetAuditLogUseCase(repo repository.AuditLogQueryRepository) aluc.GetAuditLogUseCase {
	return &getAuditLogUseCase{repo: repo}
}

func (uc *getAuditLogUseCase) Execute(c context.Context, tenantID uuid.UUID, input altypes.GetAuditLogQuery) (*entity.AuditLogEntry, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return nil, domainerrors.ErrAuditLogNotFound
	}
	e, err := uc.repo.GetByID(c, tenantID, input.ID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, domainerrors.ErrAuditLogNotFound
	}
	e.Metadata = sanitizeMetadata(e.Metadata)
	return e, nil
}
