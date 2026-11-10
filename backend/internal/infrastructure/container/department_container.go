package container

import (
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	deptimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/department"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	deptuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/department"
)

// DepartmentContainer encapsulates dependencies, repositories, usecases, and handlers for Departments.
type DepartmentContainer struct {
	Repo repository.DepartmentRepository

	CreateUseCase deptuc.CreateDepartmentUseCase
	GetUseCase    deptuc.GetDepartmentUseCase
	ListUseCase   deptuc.ListDepartmentsUseCase
	UpdateUseCase deptuc.UpdateDepartmentUseCase
	DeleteUseCase deptuc.DeleteDepartmentUseCase

	Handler *handlers.DepartmentHandler
}

// NewDepartmentContainer initializes and wires all department-related repository, usecases, and handler.
func NewDepartmentContainer(db *gorm.DB, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*DepartmentContainer, error) {
	repo := persistence.NewGormDepartmentRepository(db)

	if recorder == nil && db != nil {
		recorder = infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db))
	}
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}

	createUC := deptimpl.NewCreateDepartmentUseCase(repo, recorder, transactor)
	getUC := deptimpl.NewGetDepartmentUseCase(repo)
	listUC := deptimpl.NewListDepartmentsUseCase(repo)
	updateUC := deptimpl.NewUpdateDepartmentUseCase(repo, recorder, transactor)
	deleteUC := deptimpl.NewDeleteDepartmentUseCase(repo, recorder, transactor)

	handler := handlers.NewDepartmentHandler(createUC, getUC, listUC, updateUC, deleteUC)

	return &DepartmentContainer{
		Repo:          repo,
		CreateUseCase: createUC,
		GetUseCase:    getUC,
		ListUseCase:   listUC,
		UpdateUseCase: updateUC,
		DeleteUseCase: deleteUC,
		Handler:       handler,
	}, nil
}
