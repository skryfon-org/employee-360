package invitation

import (
	"context"
	"time"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// ValidateInvitationUseCaseImpl implements invusecase.ValidateInvitationUseCase.
type ValidateInvitationUseCaseImpl struct {
	userRepo       repository.UserRepository
	invitationRepo repository.UserInvitationRepository
	roleRepo       repository.RoleRepository
	hashService    service.HashService
}

var _ invusecase.ValidateInvitationUseCase = (*ValidateInvitationUseCaseImpl)(nil)

// NewValidateInvitationUseCase constructs a ValidateInvitationUseCaseImpl.
func NewValidateInvitationUseCase(
	userRepo repository.UserRepository,
	invitationRepo repository.UserInvitationRepository,
	roleRepo repository.RoleRepository,
	hashService service.HashService,
) *ValidateInvitationUseCaseImpl {
	return &ValidateInvitationUseCaseImpl{userRepo: userRepo, invitationRepo: invitationRepo, roleRepo: roleRepo, hashService: hashService}
}

// Execute checks the invitation token is usable and returns the invitee's email
// and role. Unknown tokens yield ErrInvalidToken; known-but-unusable ones yield
// ErrInvitationExpired / ErrInvitationRevoked / ErrInvitationAccepted.
func (u *ValidateInvitationUseCaseImpl) Execute(c context.Context, token string) (*invtypes.ValidateInvitationResponse, error) {
	inv, _, err := lookupUsableInvitation(c, u.invitationRepo, u.userRepo, u.hashService, token, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	// The endpoint is unauthenticated, so the tenant for the (tenant-scoped)
	// role lookup is taken from the invitation row, as accept does.
	role, err := u.roleRepo.GetByID(c, inv.TenantID, inv.RoleID)
	if err != nil || role == nil {
		return nil, domainerrors.ErrRoleNotFound
	}
	return &invtypes.ValidateInvitationResponse{Email: inv.Email, Role: role.Name}, nil
}
