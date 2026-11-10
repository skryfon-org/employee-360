package container

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	authusecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/auth"
	dashusecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/dashboard"
	invusecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/invitation"
	roleusecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/role"
	tenantusecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/tenant"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
	roleusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/role"
	"gorm.io/gorm"
)

type zerologLoggerAdapter struct {
	logger zerolog.Logger
}

func (z *zerologLoggerAdapter) Error(ctx context.Context, msg string, err error) {
	z.logger.Error().Err(err).Msg(msg)
}

type nopEventPublisher struct{}

func (n *nopEventPublisher) Publish(ctx context.Context, events ...event.Event) error {
	return nil
}

// AuthContainer encapsulates dependencies, repositories, services, usecases, and handlers for Authentication.
type AuthContainer struct {
	TokenService domainservice.TokenService
	HashService  domainservice.HashService
	Handler      *handlers.AuthHandler

	InvitationHandler *handlers.InvitationHandler
	RoleHandler       *handlers.RoleHandler
	DashboardHandler  *handlers.DashboardHandler
	TenantHandler     *handlers.TenantHandler

	ListAssignableRolesUseCase roleusecase.ListAssignableRolesUseCase

	// Usecases
	LoginUseCase        authusecase.LoginUseCase
	TokenRefreshUseCase authusecase.TokenRefreshUseCase
	LogoutUseCase       authusecase.LogoutUseCase
	// IdentityVerifier is used by the Auth middleware to confirm the token's
	// tenant/user are real and usable on every authenticated request.
	IdentityVerifier      authusecase.VerifyIdentityUseCase
	ForgotPasswordUseCase authusecase.ForgotPasswordUseCase
	ResetPasswordUseCase  authusecase.ResetPasswordUseCase

	// Invitation usecases (served by InvitationHandler)
	InviteUserUseCase         invusecase.InviteUserUseCase
	AcceptInvitationUseCase   invusecase.AcceptInvitationUseCase
	ResendInvitationUseCase   invusecase.ResendInvitationUseCase
	RevokeInvitationUseCase   invusecase.RevokeInvitationUseCase
	ListInvitationsUseCase    invusecase.ListInvitationsUseCase
	ValidateInvitationUseCase invusecase.ValidateInvitationUseCase

	// Repositories
	UserInvitationRepo repository.UserInvitationRepository
	OrgReferenceRepo   repository.OrgReferenceRepository
	RoleRepo           repository.RoleRepository
	UserRoleRepo       repository.UserRoleRepository
	AuditRecorder      domainservice.AuditRecorder
	UserRepo           repository.UserRepository
	RefreshTokenRepo   repository.RefreshTokenRepository
	PasswordResetRepo  repository.PasswordResetRepository
}

// NewAuthContainer initializes and wires all auth-related repositories, services, usecases, and handlers.
func NewAuthContainer(
	cfg *config.Config,
	db *gorm.DB,
	log zerolog.Logger,
	transactor ucshared.Transactor,
	eventPublisher domainservice.EventPublisher,
) (*AuthContainer, error) {
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = config.DefaultJWTSecret
	}
	accessExpiry := cfg.JWT.AccessExpiry
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry := cfg.JWT.RefreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}

	tokenService, err := infraservice.NewJWTService(jwtSecret, accessExpiry, refreshExpiry)
	if err != nil {
		return nil, err
	}

	hashService := infraservice.NewHashService()
	loggerAdapter := &zerologLoggerAdapter{logger: log}

	userRepo := persistence.NewGormUserRepository(db)
	refreshTokenRepo := persistence.NewGormRefreshTokenRepository(db)
	tenantDomainRepo := persistence.NewGormTenantDomainRepository(db)
	passwordResetRepo := persistence.NewGormPasswordResetRepository(db)

	invitationRepo := persistence.NewGormUserInvitationRepository(db)
	orgRefRepo := persistence.NewGormOrgReferenceRepository(db)
	roleRepo := persistence.NewGormRoleRepository(db)
	userRoleRepo := persistence.NewGormUserRoleRepository(db)
	auditRecorder := infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db))
	identityReader := persistence.NewGormIdentityReader(db)

	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if eventPublisher == nil {
		eventPublisher = &nopEventPublisher{}
	}

	loginUC := authusecaseimpl.NewLoginUseCase(userRepo, tenantDomainRepo, tokenService, hashService, refreshTokenRepo, loggerAdapter)
	tokenRefreshUC := authusecaseimpl.NewTokenRefreshUseCase(userRepo, tokenService, hashService, refreshTokenRepo, loggerAdapter)
	logoutUC := authusecaseimpl.NewLogoutUseCase(tokenService, hashService, refreshTokenRepo)
	forgotPasswordUC := authusecaseimpl.NewForgotPasswordUseCase(userRepo, tenantDomainRepo, passwordResetRepo, hashService, eventPublisher, transactor, loggerAdapter, cfg.App.FrontendURL)
	verifyIdentityUC := authusecaseimpl.NewVerifyIdentityUseCase(identityReader)
	resetPasswordUC := authusecaseimpl.NewResetPasswordUseCase(userRepo, passwordResetRepo, refreshTokenRepo, hashService, transactor)

	inviteAppURLs := invusecaseimpl.AppURLs{Default: cfg.App.FrontendURL, Admin: cfg.App.AdminURL, Employee: cfg.App.EmployeeURL}
	inviteUC := invusecaseimpl.NewInviteUserUseCase(userRepo, userRoleRepo, roleRepo, invitationRepo, orgRefRepo, tenantDomainRepo, persistence.NewGormTenantRepository(db), auditRecorder, hashService, eventPublisher, transactor, inviteAppURLs)
	acceptInvUC := invusecaseimpl.NewAcceptInvitationUseCase(userRepo, invitationRepo, roleRepo, persistence.NewGormTenantRepository(db), persistence.NewGormOrganizationSettingsRepository(db), hashService, eventPublisher, transactor, inviteAppURLs)
	resendInvUC := invusecaseimpl.NewResendInvitationUseCase(roleRepo, invitationRepo, auditRecorder, hashService, eventPublisher, transactor, inviteAppURLs)
	revokeInvUC := invusecaseimpl.NewRevokeInvitationUseCase(invitationRepo, userRepo, userRoleRepo, auditRecorder, transactor)
	listInvUC := invusecaseimpl.NewListInvitationsUseCase(invitationRepo)
	validateInvUC := invusecaseimpl.NewValidateInvitationUseCase(userRepo, invitationRepo, roleRepo, hashService)

	handler := handlers.NewAuthHandler(
		loginUC,
		tokenRefreshUC,
		logoutUC,
		forgotPasswordUC,
		resetPasswordUC,
	)

	invitationHandler := handlers.NewInvitationHandler(inviteUC, acceptInvUC, resendInvUC, revokeInvUC, listInvUC, validateInvUC)

	listRolesUC := roleusecaseimpl.NewListAssignableRolesUseCase(roleRepo)
	roleHandler := handlers.NewRoleHandler(listRolesUC)
	dashRepo := persistence.NewGormDashboardRepository(db)
	adminDashUC := dashusecaseimpl.NewAdminDashboardUseCase(dashRepo, invitationRepo)
	dashboardHandler := handlers.NewDashboardHandler(
		adminDashUC,
		dashusecaseimpl.NewSuperAdminDashboardUseCase(adminDashUC, dashRepo, persistence.NewGormTenantReader(db)),
	).WithEmployee(dashusecaseimpl.NewEmployeeDashboardUseCase(dashusecaseimpl.EmployeeDeps{
		Users:       userRepo,
		Allocations: persistence.NewGormProjectAllocationRepository(db),
		StatusLogs:  persistence.NewGormStatusLogRepository(db),
		Leave:       persistence.NewGormLeaveRequestRepository(db),
		Holidays:    persistence.NewGormHolidayRepository(db),
		Categories:  persistence.NewGormHolidayCategoryRepository(db),
		OrgSettings: persistence.NewGormOrganizationSettingsRepository(db),
	}, nil))

	tenantRepo := persistence.NewGormTenantRepository(db)
	tenantDomainMgr := persistence.NewGormTenantDomainManager(db)
	tenantHandler := handlers.NewTenantHandler(handlers.TenantUseCases{
		Get:          tenantusecaseimpl.NewGetTenantUseCase(tenantRepo, tenantDomainMgr),
		Rename:       tenantusecaseimpl.NewRenameTenantUseCase(tenantRepo, auditRecorder, transactor),
		ListDomains:  tenantusecaseimpl.NewListTenantDomainsUseCase(tenantRepo, tenantDomainMgr),
		AddDomain:    tenantusecaseimpl.NewAddTenantDomainUseCase(tenantRepo, tenantDomainMgr, auditRecorder, transactor),
		UpdateDomain: tenantusecaseimpl.NewUpdateTenantDomainUseCase(tenantRepo, tenantDomainMgr, auditRecorder, transactor),
		DelDomain:    tenantusecaseimpl.NewRemoveTenantDomainUseCase(tenantRepo, tenantDomainMgr, auditRecorder, transactor),
	})

	return &AuthContainer{
		TenantHandler:              tenantHandler,
		RoleHandler:                roleHandler,
		DashboardHandler:           dashboardHandler,
		ListAssignableRolesUseCase: listRolesUC,
		InvitationHandler:          invitationHandler,
		TokenService:               tokenService,
		HashService:                hashService,
		Handler:                    handler,
		LoginUseCase:               loginUC,
		TokenRefreshUseCase:        tokenRefreshUC,
		LogoutUseCase:              logoutUC,
		IdentityVerifier:           verifyIdentityUC,
		ForgotPasswordUseCase:      forgotPasswordUC,
		ResetPasswordUseCase:       resetPasswordUC,
		InviteUserUseCase:          inviteUC,
		AcceptInvitationUseCase:    acceptInvUC,
		ResendInvitationUseCase:    resendInvUC,
		RevokeInvitationUseCase:    revokeInvUC,
		ListInvitationsUseCase:     listInvUC,
		ValidateInvitationUseCase:  validateInvUC,
		UserInvitationRepo:         invitationRepo,
		OrgReferenceRepo:           orgRefRepo,
		RoleRepo:                   roleRepo,
		UserRoleRepo:               userRoleRepo,
		AuditRecorder:              auditRecorder,
		UserRepo:                   userRepo,
		RefreshTokenRepo:           refreshTokenRepo,
		PasswordResetRepo:          passwordResetRepo,
	}, nil
}
