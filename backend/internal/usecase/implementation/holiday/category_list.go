package holiday

import (
	"context"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type listCategoriesUseCase struct {
	categoryRepo repository.HolidayCategoryRepository
}

// NewListCategoriesUseCase constructs a ListCategoriesUseCase.
func NewListCategoriesUseCase(categoryRepo repository.HolidayCategoryRepository) holidayuc.ListCategoriesUseCase {
	return &listCategoriesUseCase{
		categoryRepo: categoryRepo,
	}
}

func (uc *listCategoriesUseCase) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	page, pageSize, offset := pagination.Normalize(query.Page, query.PageSize)

	items, total, err := uc.categoryRepo.List(ctx, tenantID, query.IsActive, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &holidaytypes.ListCategoriesResult{
		Categories: items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}
