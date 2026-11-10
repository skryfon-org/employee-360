package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
)

const (
	defaultResetTokenExpiry = 15 * time.Minute
)

// ForgotPasswordUseCaseImpl implements authusecase.ForgotPasswordUseCase.
type ForgotPasswordUseCaseImpl struct {
	userRepo          repository.UserRepository
	tenantDomainRepo  repository.TenantDomainRepository
	passwordResetRepo repository.PasswordResetRepository
	hashService       service.HashService
	eventPublisher    service.EventPublisher
	transactor        ucshared.Transactor
	logger            service.Logger
	tokenExpiry       time.Duration
	frontendBaseURL   string
}

var _ authusecase.ForgotPasswordUseCase = (*ForgotPasswordUseCaseImpl)(nil)

// NewForgotPasswordUseCase constructs a new ForgotPasswordUseCaseImpl.
func NewForgotPasswordUseCase(
	userRepo repository.UserRepository,
	tenantDomainRepo repository.TenantDomainRepository,
	passwordResetRepo repository.PasswordResetRepository,
	hashService service.HashService,
	eventPublisher service.EventPublisher,
	transactor ucshared.Transactor,
	logger service.Logger,
	frontendBaseURL string,
) *ForgotPasswordUseCaseImpl {
	return &ForgotPasswordUseCaseImpl{
		userRepo:          userRepo,
		tenantDomainRepo:  tenantDomainRepo,
		passwordResetRepo: passwordResetRepo,
		hashService:       hashService,
		eventPublisher:    eventPublisher,
		transactor:        transactor,
		logger:            logger,
		tokenExpiry:       defaultResetTokenExpiry,
		frontendBaseURL:   strings.TrimRight(frontendBaseURL, "/"),
	}
}

// Execute initiates an enumeration-safe password reset flow.
// It always returns nil to prevent user enumeration attacks.
func (u *ForgotPasswordUseCaseImpl) Execute(ctx context.Context, input authtypes.ForgotPasswordRequest) error {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" {
		_ = u.hashService.HashToken("dummy-timing-mitigation-token")
		return nil
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		_ = u.hashService.HashToken("dummy-timing-mitigation-token")
		return nil
	}

	// Resolve the tenant server-side from the email's domain; never from client input.
	tenant, err := u.tenantDomainRepo.FindTenantByDomain(ctx, parts[1])
	if err != nil || tenant == nil {
		_ = u.hashService.HashToken("dummy-timing-mitigation-token")
		return nil
	}

	user, err := u.userRepo.GetByTenantAndEmail(ctx, tenant.ID, email)
	if err != nil || user == nil || !user.IsActive {
		_ = u.hashService.HashToken("dummy-timing-mitigation-token")
		return nil
	}

	// Generate 32 cryptographically secure random bytes
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil
	}
	plainToken := hex.EncodeToString(tokenBytes)
	tokenHash := u.hashService.HashToken(plainToken)

	now := time.Now().UTC()
	tokenEntity := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(u.tokenExpiry),
		CreatedAt: now,
		UpdatedAt: now,
	}

	resetEvent := event.Event{
		ID:            uuid.New(),
		TenantID:      user.TenantID,
		EventType:     event.EventTypePasswordResetRequested,
		AggregateType: "user",
		AggregateID:   user.ID.String(),
		Payload: event.PasswordResetRequestedPayload{
			UserID:     user.ID,
			TenantID:   user.TenantID,
			Email:      user.Email,
			UserName:   strings.TrimSpace(user.FirstName + " " + user.LastName),
			PlainToken: plainToken,
			ResetURL:   u.frontendBaseURL + "/reset-password?token=" + plainToken,
			ExpiresAt:  tokenEntity.ExpiresAt,
		},
		OccurredAt: now,
	}

	if err := u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := u.passwordResetRepo.InvalidateAllForUser(txCtx, user.ID); err != nil {
			return err
		}

		if err := u.passwordResetRepo.Create(txCtx, tokenEntity); err != nil {
			return err
		}

		if err := u.eventPublisher.Publish(txCtx, resetEvent); err != nil {
			return err
		}

		return nil
	}); err != nil {
		// Enumeration-safety requires Execute to always return nil to the
		// caller, but the failure must not be silently swallowed: record it
		// so operators can detect and investigate.
		if u.logger != nil {
			u.logger.Error(ctx, "forgot_password: failed to persist password reset request", err)
		}
		return nil
	}

	return nil
}
