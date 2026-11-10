package organization

import (
	"context"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	orguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/organization"
)

type getWeeklyOffDaysUseCase struct {
	reader repository.WeeklyOffDaysReader
}

// NewGetWeeklyOffDaysUseCase constructs a GetWeeklyOffDaysUseCase.
func NewGetWeeklyOffDaysUseCase(reader repository.WeeklyOffDaysReader) orguc.GetWeeklyOffDaysUseCase {
	return &getWeeklyOffDaysUseCase{reader: reader}
}

func (uc *getWeeklyOffDaysUseCase) Execute(ctx context.Context, tenantID uuid.UUID) ([]int, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return uc.reader.WeeklyOffDays(ctx, tenantID)
}
