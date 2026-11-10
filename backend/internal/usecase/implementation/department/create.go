package department

import (
	"context"
	"errors"
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

var (
	ErrDepartmentNameRequired = errors.New("department name is required")
	ErrDepartmentNameTooLong  = errors.New("department name must not exceed 100 characters")
	ErrDescriptionTooLong     = errors.New("department description must not exceed 500 characters")
)

type createDepartmentUseCase struct {
	repo       repository.DepartmentRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewCreateDepartmentUseCase creates a new CreateDepartmentUseCase.
func NewCreateDepartmentUseCase(
	repo repository.DepartmentRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) deptuc.CreateDepartmentUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createDepartmentUseCase{
		repo:       repo,
		audit:      audit,
		transactor: transactor,
	}
}

func (uc *createDepartmentUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
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

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	now := time.Now().UTC()
	dept := &entity.Department{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		IsActive:    isActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		exists, err := uc.repo.ExistsByName(txCtx, tenantID, name)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrDepartmentNameTaken
		}

		if err := uc.repo.Create(txCtx, tenantID, actorID, dept); err != nil {
			return err
		}

		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionDepartmentCreate, domainaudit.EntityDepartment, dept.ID, map[string]any{
			"name":      dept.Name,
			"is_active": dept.IsActive,
		})
	})
	if err != nil {
		return nil, err
	}

	return dept, nil
}
