package department

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	deptuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/department"
)

type updateDepartmentUseCase struct {
	repo       repository.DepartmentRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateDepartmentUseCase creates a new UpdateDepartmentUseCase.
func NewUpdateDepartmentUseCase(
	repo repository.DepartmentRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) deptuc.UpdateDepartmentUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateDepartmentUseCase{
		repo:       repo,
		audit:      audit,
		transactor: transactor,
	}
}

func (uc *updateDepartmentUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return nil, domainerrors.ErrDepartmentNotFound
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrDepartmentNameRequired
	}
	if utf8.RuneCountInString(name) > 100 {
		return nil, ErrDepartmentNameTooLong
	}

	description := strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(description) > 500 {
		return nil, ErrDescriptionTooLong
	}

	var existing *entity.Department
	err := uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		var err error
		existing, err = uc.repo.GetByID(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		// If name has changed, check for duplicate name
		if !strings.EqualFold(existing.Name, name) {
			exists, err := uc.repo.ExistsByName(txCtx, tenantID, name)
			if err != nil {
				return err
			}
			if exists {
				return domainerrors.ErrDepartmentNameTaken
			}
		}

		wasActive := existing.IsActive
		existing.Name = name
		existing.Description = description
		if input.IsActive != nil {
			existing.IsActive = *input.IsActive
		}
		existing.UpdatedAt = time.Now().UTC()

		if err := uc.repo.Update(txCtx, tenantID, actorID, existing); err != nil {
			return err
		}

		if err := uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionDepartmentUpdate, domainaudit.EntityDepartment, existing.ID, map[string]any{
			"name":      existing.Name,
			"is_active": existing.IsActive,
		}); err != nil {
			return err
		}
		if wasActive == existing.IsActive {
			return nil
		}
		action := domainaudit.ActionDepartmentDeactivate
		if existing.IsActive {
			action = domainaudit.ActionDepartmentActivate
		}
		return uc.audit.Record(txCtx, tenantID, actorID, action, domainaudit.EntityDepartment, existing.ID, map[string]any{
			"name": existing.Name,
		})
	})
	if err != nil {
		return nil, err
	}

	return existing, nil
}
