package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	allocimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project/allocation"
	prjimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// ProjectContainer wires projects and project allocations.
type ProjectContainer struct {
	Handler           *handlers.ProjectHandler
	AllocationHandler *handlers.ProjectAllocationHandler
}

// NewProjectContainer wires the project repositories, usecases and handler.
func NewProjectContainer(db *gorm.DB, users repository.UserRepository, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*ProjectContainer, error) {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	projects := persistence.NewGormProjectRepository(db)
	allocations := persistence.NewGormProjectAllocationRepository(db)
	ph := handlers.NewProjectHandler(handlers.ProjectUseCases{
		Create:      prjimpl.NewCreateProjectUseCase(projects, recorder, transactor),
		Get:         prjimpl.NewGetProjectUseCase(projects),
		List:        prjimpl.NewListProjectsUseCase(projects),
		Update:      prjimpl.NewUpdateProjectUseCase(projects, recorder, transactor),
		SuggestCode: prjimpl.NewSuggestProjectCodeUseCase(projects),
	})
	ah := handlers.NewProjectAllocationHandler(handlers.ProjectAllocationUseCases{
		ListByProject: allocimpl.NewListProjectAllocationsUseCase(projects, allocations),
		Create:        allocimpl.NewCreateAllocationUseCase(projects, users, allocations, recorder, transactor),
		Update:        allocimpl.NewUpdateAllocationUseCase(projects, allocations, recorder, transactor),
		Delete:        allocimpl.NewDeleteAllocationUseCase(allocations, recorder, transactor),
		ByUser:        allocimpl.NewListUserAllocationsUseCase(users, allocations),
		MyProjects:    allocimpl.NewListMyProjectsUseCase(users, allocations),
	})
	return &ProjectContainer{Handler: ph, AllocationHandler: ah}, nil
}
