package invitation

import (
	"context"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// ListInvitationsUseCaseImpl implements invusecase.ListInvitationsUseCase.
type ListInvitationsUseCaseImpl struct {
	invitationRepo repository.UserInvitationRepository
}

var _ invusecase.ListInvitationsUseCase = (*ListInvitationsUseCaseImpl)(nil)

// NewListInvitationsUseCase constructs a ListInvitationsUseCaseImpl.
func NewListInvitationsUseCase(invitationRepo repository.UserInvitationRepository) *ListInvitationsUseCaseImpl {
	return &ListInvitationsUseCaseImpl{invitationRepo: invitationRepo}
}

// Execute lists invitations for the caller's tenant only. Paging is
// normalised (page >= 1, page size defaulted/capped) and the status filter
// validated here so every caller gets the same rules.
func (u *ListInvitationsUseCaseImpl) Execute(c context.Context, tenantID uuid.UUID, q invtypes.ListInvitationsQuery) (*invtypes.ListInvitationsResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	switch q.Status {
	case "", entity.InvitationStatusPending, entity.InvitationStatusAccepted,
		entity.InvitationStatusExpired, entity.InvitationStatusRevoked:
	default:
		return nil, domainerrors.ErrInvalidInvitationFilter
	}
	page, pageSize, offset := pagination.Normalize(q.Page, q.PageSize)
	items, total, err := u.invitationRepo.List(c, tenantID, repository.InvitationListFilter{
		Status: q.Status, Search: q.Search, Now: time.Now(),
		Limit: pageSize, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return &invtypes.ListInvitationsResult{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: pagination.TotalPages(total, pageSize),
	}, nil
}
