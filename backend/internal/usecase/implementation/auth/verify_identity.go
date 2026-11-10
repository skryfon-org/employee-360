package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
)

// VerifyIdentityUseCaseImpl implements authusecase.VerifyIdentityUseCase.
type VerifyIdentityUseCaseImpl struct {
	identityReader repository.IdentityReader
}

var _ authusecase.VerifyIdentityUseCase = (*VerifyIdentityUseCaseImpl)(nil)

// NewVerifyIdentityUseCase constructs a new VerifyIdentityUseCaseImpl.
func NewVerifyIdentityUseCase(identityReader repository.IdentityReader) *VerifyIdentityUseCaseImpl {
	return &VerifyIdentityUseCaseImpl{identityReader: identityReader}
}

// Execute verifies the tenant and user and loads the user's current roles with
// a single repository call. Any unusable identity is a uniform ErrUnauthorized;
// repository errors propagate so callers can fail closed.
func (u *VerifyIdentityUseCaseImpl) Execute(ctx context.Context, tenantID, userID uuid.UUID) (*authtypes.VerifiedIdentity, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	state, err := u.identityReader.GetIdentityState(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrTenantNotFound) || errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, domainerrors.ErrUnauthorized
		}
		return nil, err
	}
	if state == nil || !state.TenantActive || !state.UserFound || !state.UserActive {
		return nil, domainerrors.ErrUnauthorized
	}

	roles := make([]string, len(state.RoleNames))
	copy(roles, state.RoleNames)
	return &authtypes.VerifiedIdentity{TenantID: tenantID, UserID: userID, Roles: roles}, nil
}
