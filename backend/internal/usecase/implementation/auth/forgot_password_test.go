package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// MockPasswordResetRepository implements repository.PasswordResetRepository.
type mockPasswordResetRepository struct {
	tokensByHash       map[string]*entity.PasswordResetToken
	tokensByID         map[uuid.UUID]*entity.PasswordResetToken
	invalidatedUserIDs []uuid.UUID
	usedTokenIDs       []uuid.UUID
}

func newMockPasswordResetRepository() *mockPasswordResetRepository {
	return &mockPasswordResetRepository{
		tokensByHash: make(map[string]*entity.PasswordResetToken),
		tokensByID:   make(map[uuid.UUID]*entity.PasswordResetToken),
	}
}

func (m *mockPasswordResetRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	m.tokensByHash[token.TokenHash] = token
	m.tokensByID[token.ID] = token
	return nil
}

func (m *mockPasswordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error) {
	t, ok := m.tokensByHash[tokenHash]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return t, nil
}

func (m *mockPasswordResetRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	m.usedTokenIDs = append(m.usedTokenIDs, id)
	if t, ok := m.tokensByID[id]; ok {
		if t.UsedAt != nil {
			return domainerrors.ErrTokenAlreadyUsed
		}
		now := time.Now().UTC()
		t.UsedAt = &now
	}
	return nil
}

func (m *mockPasswordResetRepository) InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error {
	m.invalidatedUserIDs = append(m.invalidatedUserIDs, userID)
	now := time.Now().UTC()
	for _, t := range m.tokensByID {
		if t.UserID == userID {
			t.UsedAt = &now
		}
	}
	return nil
}

func (m *mockPasswordResetRepository) DeleteExpiredTokens(ctx context.Context, before time.Time) error {
	return nil
}

// MockEventPublisher implements service.EventPublisher.
type mockEventPublisher struct {
	publishedEvents []event.Event
}

func (m *mockEventPublisher) Publish(ctx context.Context, events ...event.Event) error {
	m.publishedEvents = append(m.publishedEvents, events...)
	return nil
}

// mockLogger implements service.Logger, recording calls for assertions.
type mockLogger struct {
	errorCalls []struct {
		msg string
		err error
	}
}

func (m *mockLogger) Error(ctx context.Context, msg string, err error) {
	m.errorCalls = append(m.errorCalls, struct {
		msg string
		err error
	}{msg: msg, err: err})
}

// failingTransactor is a ucshared.Transactor that always fails before running fn,
// used to simulate a persistence failure inside the forgot-password flow.
type failingTransactor struct {
	err error
}

func (f *failingTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return f.err
}

const testFrontendURL = "http://frontend.test"

func TestForgotPasswordUseCase_ExistingUser(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	hashSvc := &mockHashService{}
	eventPub := &mockEventPublisher{}
	transactor := ucshared.NewNopTransactor()
	logger := &mockLogger{}

	user := &entity.User{
		ID:        userID,
		TenantID:  tenantID,
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Smith",
		IsActive:  true,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, hashSvc, eventPub, transactor, logger, testFrontendURL)

	err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{
		Email: "alice@example.com",
	})

	if err != nil {
		t.Fatalf("expected nil error (enumeration-safe), got: %v", err)
	}

	// Verify token was stored in repository with a hash
	if len(resetRepo.tokensByID) != 1 {
		t.Fatalf("expected 1 password reset token in repository, got %d", len(resetRepo.tokensByID))
	}

	var storedToken *entity.PasswordResetToken
	for _, tk := range resetRepo.tokensByID {
		storedToken = tk
	}

	if storedToken.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, storedToken.UserID)
	}
	if storedToken.TenantID != tenantID {
		t.Errorf("expected tenant ID %s, got %s", tenantID, storedToken.TenantID)
	}
	// Verify raw token is never stored directly
	if storedToken.TokenHash == "" || storedToken.TokenHash[:7] != "sha256_" {
		t.Errorf("expected token to be hashed before persistence, got: %s", storedToken.TokenHash)
	}

	// Verify event was published
	if len(eventPub.publishedEvents) != 1 {
		t.Fatalf("expected 1 domain event published, got %d", len(eventPub.publishedEvents))
	}

	evt := eventPub.publishedEvents[0]
	if evt.EventType != event.EventTypePasswordResetRequested {
		t.Errorf("expected EventTypePasswordResetRequested, got %s", evt.EventType)
	}
	if evt.TenantID != tenantID {
		t.Errorf("expected event tenant ID %s, got %s", tenantID, evt.TenantID)
	}

	payload, ok := evt.Payload.(event.PasswordResetRequestedPayload)
	if !ok {
		t.Fatalf("expected PasswordResetRequestedPayload type in event payload")
	}
	if payload.Email != "alice@example.com" {
		t.Errorf("expected payload email alice@example.com, got %s", payload.Email)
	}
	if payload.UserName != "Alice Smith" {
		t.Errorf("expected payload user name Alice Smith, got %q", payload.UserName)
	}
	if payload.PlainToken == "" {
		t.Errorf("expected non-empty plain token in event payload")
	}
	if hashSvc.HashToken(payload.PlainToken) != storedToken.TokenHash {
		t.Errorf("hash of event plain token does not match stored token hash")
	}
	if want := testFrontendURL + "/reset-password?token=" + payload.PlainToken; payload.ResetURL != want {
		t.Errorf("expected reset URL %q, got %q", want, payload.ResetURL)
	}
}

func TestForgotPasswordUseCase_EnumerationSafe(t *testing.T) {
	tenantID := uuid.New()
	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	hashSvc := &mockHashService{}
	eventPub := &mockEventPublisher{}
	transactor := ucshared.NewNopTransactor()
	logger := &mockLogger{}

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, hashSvc, eventPub, transactor, logger, testFrontendURL)

	tests := []struct {
		name  string
		email string
	}{
		{"nonexistent user", "unknown@example.com"},
		{"invalid format", "not-an-email"},
		{"empty email", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventPub.publishedEvents = nil
			err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{
				Email: tc.email,
			})

			// Acceptance criterion: identical response (nil error)
			if err != nil {
				t.Errorf("expected nil error for enumeration safety, got %v", err)
			}
			if len(eventPub.publishedEvents) != 0 {
				t.Errorf("expected no events published for invalid/missing user, got %d", len(eventPub.publishedEvents))
			}
		})
	}
}

func TestForgotPasswordUseCase_InactiveUser(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	hashSvc := &mockHashService{}
	eventPub := &mockEventPublisher{}
	transactor := ucshared.NewNopTransactor()
	logger := &mockLogger{}

	user := &entity.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "inactive@example.com",
		IsActive: false,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, hashSvc, eventPub, transactor, logger, testFrontendURL)

	err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{
		Email: "inactive@example.com",
	})

	if err != nil {
		t.Fatalf("expected nil error for inactive user (enumeration-safe), got %v", err)
	}
	if len(eventPub.publishedEvents) != 0 {
		t.Errorf("expected 0 published events for inactive user, got %d", len(eventPub.publishedEvents))
	}
}

// TestForgotPasswordUseCase_TenantIsolation proves that two tenants with a
// user sharing the same email address are never conflated: a forgot-password
// request scoped to tenantA must only ever resolve/affect tenantA's user.
func TestForgotPasswordUseCase_TenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	const sharedEmail = "shared@example.com"

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	hashSvc := &mockHashService{}
	eventPub := &mockEventPublisher{}
	transactor := ucshared.NewNopTransactor()
	logger := &mockLogger{}

	_ = userRepo.Create(context.Background(), &entity.User{
		ID:       userA,
		TenantID: tenantA,
		Email:    sharedEmail,
		IsActive: true,
	})
	_ = userRepo.Create(context.Background(), &entity.User{
		ID:       userB,
		TenantID: tenantB,
		Email:    sharedEmail,
		IsActive: true,
	})

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantA)
	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, hashSvc, eventPub, transactor, logger, testFrontendURL)

	err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{
		Email: sharedEmail,
	})
	if err != nil {
		t.Fatalf("expected nil error (enumeration-safe), got: %v", err)
	}

	if len(eventPub.publishedEvents) != 1 {
		t.Fatalf("expected exactly 1 event published, got %d", len(eventPub.publishedEvents))
	}

	evt := eventPub.publishedEvents[0]
	if evt.TenantID != tenantA {
		t.Errorf("expected event tenant ID %s (tenantA), got %s", tenantA, evt.TenantID)
	}

	payload, ok := evt.Payload.(event.PasswordResetRequestedPayload)
	if !ok {
		t.Fatalf("expected PasswordResetRequestedPayload type in event payload")
	}
	if payload.UserID != userA {
		t.Errorf("expected reset issued for tenantA's user %s, got %s (tenantB's user is %s)", userA, payload.UserID, userB)
	}
	if payload.TenantID != tenantA {
		t.Errorf("expected payload tenant ID %s (tenantA), got %s", tenantA, payload.TenantID)
	}

	var storedToken *entity.PasswordResetToken
	for _, tk := range resetRepo.tokensByID {
		storedToken = tk
	}
	if storedToken == nil {
		t.Fatalf("expected a password reset token to be stored")
	}
	if storedToken.UserID != userA {
		t.Errorf("expected stored token to belong to tenantA's user %s, got %s", userA, storedToken.UserID)
	}
	if storedToken.TenantID != tenantA {
		t.Errorf("expected stored token tenant ID %s (tenantA), got %s", tenantA, storedToken.TenantID)
	}

	// tenantB's user must be completely untouched by tenantA's request.
	if len(resetRepo.invalidatedUserIDs) != 1 || resetRepo.invalidatedUserIDs[0] != userA {
		t.Errorf("expected only tenantA's user %s to be invalidated, got %v", userA, resetRepo.invalidatedUserIDs)
	}
	for _, id := range resetRepo.invalidatedUserIDs {
		if id == userB {
			t.Errorf("tenantB's user %s must never be affected by a tenantA-scoped request", userB)
		}
	}
}

// TestForgotPasswordUseCase_TransactionFailureIsLoggedNotSwallowed proves that
// when the transactor fails, Execute still returns nil (enumeration-safety is
// preserved) but the failure is recorded via the Logger port rather than
// being silently discarded.
func TestForgotPasswordUseCase_TransactionFailureIsLoggedNotSwallowed(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	hashSvc := &mockHashService{}
	eventPub := &mockEventPublisher{}
	txErr := domainerrors.ErrNotFound // reused only as a sentinel error for this test
	transactor := &failingTransactor{err: txErr}
	logger := &mockLogger{}

	user := &entity.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "alice@example.com",
		IsActive: true,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, hashSvc, eventPub, transactor, logger, testFrontendURL)

	err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{
		Email: "alice@example.com",
	})

	if err != nil {
		t.Fatalf("expected nil error even when the transaction fails (enumeration-safe), got: %v", err)
	}

	if len(logger.errorCalls) != 1 {
		t.Fatalf("expected the transaction failure to be logged exactly once, got %d calls", len(logger.errorCalls))
	}
	if logger.errorCalls[0].err != txErr {
		t.Errorf("expected logged error to be the transactor's error, got %v", logger.errorCalls[0].err)
	}
}

// TestForgotPasswordUseCase_UnknownDomain proves an email whose domain maps to
// no tenant returns nil, publishes nothing and stores nothing.
func TestForgotPasswordUseCase_UnknownDomain(t *testing.T) {
	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	eventPub := &mockEventPublisher{}
	tdRepo := newMockTenantDomainRepository() // no domains registered

	// Same-email user exists in some tenant, but the domain is unmapped.
	_ = userRepo.Create(context.Background(), &entity.User{
		ID: uuid.New(), TenantID: uuid.New(), Email: "alice@unmapped.example", IsActive: true,
	})

	uc := NewForgotPasswordUseCase(userRepo, tdRepo, resetRepo, &mockHashService{}, eventPub, ucshared.NewNopTransactor(), &mockLogger{}, testFrontendURL)

	if err := uc.Execute(context.Background(), authtypes.ForgotPasswordRequest{Email: "alice@unmapped.example"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(eventPub.publishedEvents) != 0 || len(resetRepo.tokensByID) != 0 {
		t.Errorf("expected no events/tokens for unknown domain")
	}
}
