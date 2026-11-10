package container

import (
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	posimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/position"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	posuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/position"
)

// PositionContainer encapsulates dependencies, repositories, usecases, and handlers for Positions.
type PositionContainer struct {
	Repo repository.PositionRepository

	CreateUseCase posuc.CreatePositionUseCase
	GetUseCase    posuc.GetPositionUseCase
	ListUseCase   posuc.ListPositionsUseCase
	UpdateUseCase posuc.UpdatePositionUseCase
	DeleteUseCase posuc.DeletePositionUseCase

	Handler *handlers.PositionHandler
}

// NewPositionContainer initializes and wires all position-related repository, usecases, and handler.
func NewPositionContainer(db *gorm.DB, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*PositionContainer, error) {
	repo := persistence.NewGormPositionRepository(db)

	if recorder == nil && db != nil {
		recorder = infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db))
	}
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}

	createUC := posimpl.NewCreatePositionUseCase(repo, recorder, transactor)
	getUC := posimpl.NewGetPositionUseCase(repo)
	listUC := posimpl.NewListPositionsUseCase(repo)
	updateUC := posimpl.NewUpdatePositionUseCase(repo, recorder, transactor)
	deleteUC := posimpl.NewDeletePositionUseCase(repo, recorder, transactor)

	handler := handlers.NewPositionHandler(createUC, getUC, listUC, updateUC, deleteUC)

	return &PositionContainer{
		Repo:          repo,
		CreateUseCase: createUC,
		GetUseCase:    getUC,
		ListUseCase:   listUC,
		UpdateUseCase: updateUC,
		DeleteUseCase: deleteUC,
		Handler:       handler,
	}, nil
}
