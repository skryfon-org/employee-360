package position

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
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	posuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/position"
)

var (
	ErrPositionNameRequired = errors.New("position name is required")
	ErrPositionNameTooLong  = errors.New("position name must not exceed 100 characters")
	ErrDescriptionTooLong   = errors.New("position description must not exceed 500 characters")
)

type createPositionUseCase struct {
	repo       repository.PositionRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewCreatePositionUseCase creates a new CreatePositionUseCase.
func NewCreatePositionUseCase(
	repo repository.PositionRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) posuc.CreatePositionUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createPositionUseCase{
		repo:       repo,
		audit:      audit,
		transactor: transactor,
	}
}

func (uc *createPositionUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrPositionNameRequired
	}
	if utf8.RuneCountInString(name) > 100 {
		return nil, ErrPositionNameTooLong
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
	pos := &entity.Position{
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
			return domainerrors.ErrPositionNameTaken
		}

		if err := uc.repo.Create(txCtx, tenantID, actorID, pos); err != nil {
			return err
		}

		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionPositionCreate, domainaudit.EntityPosition, pos.ID, map[string]any{
			"name":      pos.Name,
			"is_active": pos.IsActive,
		})
	})
	if err != nil {
		return nil, err
	}

	return pos, nil
}
