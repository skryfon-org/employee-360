package department

import (
	"context"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
	deptuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/department"
)

type listDepartmentsUseCase struct {
	repo repository.DepartmentRepository
}

// NewListDepartmentsUseCase creates a new ListDepartmentsUseCase.
func NewListDepartmentsUseCase(repo repository.DepartmentRepository) deptuc.ListDepartmentsUseCase {
	return &listDepartmentsUseCase{repo: repo}
}

func (uc *listDepartmentsUseCase) Execute(c context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	page, pageSize, offset := pagination.Normalize(input.Page, input.PageSize)

	items, total, err := uc.repo.List(c, tenantID, input.IsActive, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &depttypes.ListDepartmentsResult{
		Departments: items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}
