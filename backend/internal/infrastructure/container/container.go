// Package container manages application dependency injection and wiring.
package container

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/config"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/eventing"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// AppContainer holds wired configuration, infrastructure, and domain sub-containers.
type AppContainer struct {
	Config *config.Config
	Log    zerolog.Logger
	DB     *gorm.DB
	// Cache is the Redis-backed cache, or a no-op cache when Redis is disabled.
	// No consumers yet; it is wired for follow-up tickets.
	Cache domainservice.Cache

	Health       *HealthContainer
	Auth         *AuthContainer
	Department   *DepartmentContainer
	Position     *PositionContainer
	AuditLog     *AuditLogContainer
	Holiday      *HolidayContainer
	Employee     *EmployeeContainer
	Organization *OrganizationContainer
	Project      *ProjectContainer
	StatusLog    *StatusLogContainer
	Leave        *LeaveContainer
}

// NewAppContainer wires the full dependency graph for the application.
//
// cache may be nil when Redis is disabled; the container then exposes a no-op
// cache and health reports redis as "disabled".
func NewAppContainer(cfg *config.Config, db *gorm.DB, cache domainservice.Cache, log zerolog.Logger) (*AppContainer, error) {
	healthContainer := NewHealthContainer(db, cache)
	if cache == nil {
		cache = infraservice.NewNoopCache()
	}

	// Transactional outbox: the GORM transactor and River-backed publisher share
	// the in-flight transaction via context. A nil db (construction-only tests)
	// falls back to the no-op transactor/publisher inside NewAuthContainer.
	var (
		transactor ucshared.Transactor
		publisher  domainservice.EventPublisher
	)
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("container: resolve sql.DB: %w", err)
		}
		riverClient, err := eventing.NewInsertOnlyClient(sqlDB)
		if err != nil {
			return nil, fmt.Errorf("container: build river insert client: %w", err)
		}
		transactor = database.NewGormTransactor(db)
		publisher = eventing.NewRiverPublisher(riverClient, eventing.NewDispatcher())
	}

	authContainer, err := NewAuthContainer(cfg, db, log, transactor, publisher)
	if err != nil {
		return nil, err
	}

	departmentContainer, err := NewDepartmentContainer(db, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	positionContainer, err := NewPositionContainer(db, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	auditLogContainer, err := NewAuditLogContainer(db)
	if err != nil {
		return nil, err
	}

	holidayContainer, err := NewHolidayContainer(db, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	employeeContainer, err := NewEmployeeContainer(db, authContainer, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	organizationContainer, err := NewOrganizationContainer(db, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	projectContainer, err := NewProjectContainer(db, authContainer.UserRepo, authContainer.AuditRecorder, transactor)
	if err != nil {
		return nil, err
	}

	allocationsRepo := persistence.NewGormProjectAllocationRepository(db)
	statusLogContainer, err := NewStatusLogContainer(db, authContainer.UserRepo, allocationsRepo, transactor)
	if err != nil {
		return nil, err
	}

	leaveContainer, err := NewLeaveContainer(db, authContainer.AuditRecorder, transactor, publisher, cfg.App, log)
	if err != nil {
		return nil, err
	}

	return &AppContainer{
		Config:       cfg,
		Log:          log,
		DB:           db,
		Cache:        cache,
		Health:       healthContainer,
		Auth:         authContainer,
		Department:   departmentContainer,
		Position:     positionContainer,
		AuditLog:     auditLogContainer,
		Holiday:      holidayContainer,
		Employee:     employeeContainer,
		Organization: organizationContainer,
		Project:      projectContainer,
		StatusLog:    statusLogContainer,
		Leave:        leaveContainer,
	}, nil
}

// Container is an alias for AppContainer.
type Container = AppContainer

// New is an alias for NewAppContainer.
func New(cfg *config.Config, db *gorm.DB, cache domainservice.Cache, log zerolog.Logger) (*AppContainer, error) {
	return NewAppContainer(cfg, db, cache, log)
}

// WorkerContainer holds the dependencies of the cmd/worker process.
type WorkerContainer struct {
	RiverClient *river.Client[*sql.Tx]
}

// NewWorkerContainer wires the River worker client with the SMTP EmailService.
func NewWorkerContainer(cfg *config.Config, db *gorm.DB) (*WorkerContainer, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("container: resolve sql.DB: %w", err)
	}
	emailSvc := infraservice.NewSMTPMailService(cfg.SMTP)
	slogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client, err := eventing.NewWorkerClient(sqlDB, emailSvc, slogger, 0)
	if err != nil {
		return nil, fmt.Errorf("container: build river worker client: %w", err)
	}
	return &WorkerContainer{RiverClient: client}, nil
}
