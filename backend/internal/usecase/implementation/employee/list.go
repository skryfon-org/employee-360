package employee

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type listEmployeesUseCase struct {
	directory repository.EmployeeDirectoryRepository
}

// NewListEmployeesUseCase creates a new ListEmployeesUseCase.
func NewListEmployeesUseCase(directory repository.EmployeeDirectoryRepository) empuc.ListEmployeesUseCase {
	return &listEmployeesUseCase{directory: directory}
}

func (uc *listEmployeesUseCase) Execute(ctx context.Context, tenantID uuid.UUID, q emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if s := q.Filter.EmploymentStatus; s != "" {
		if err := validateEmploymentStatus(s); err != nil {
			return nil, err
		}
	}
	if q.Filter.IncludeAllocation {
		n := time.Now().UTC()
		q.Filter.AllocationAsOf = time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	}
	page, size, offset := pagination.Normalize(q.Page, q.PageSize)
	items, total, err := uc.directory.List(ctx, tenantID, q.Filter, size, offset)
	if err != nil {
		return nil, err
	}
	return &emptypes.ListEmployeesResult{Employees: items, Total: total, Page: page, PageSize: size}, nil
}
