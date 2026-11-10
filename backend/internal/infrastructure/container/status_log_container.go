package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	statuslogimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/statuslog"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// StatusLogContainer wires status logs dependencies.
type StatusLogContainer struct {
	Handler *handlers.StatusLogHandler
}

// NewStatusLogContainer wires repository, usecases and handler for status logs.
func NewStatusLogContainer(
	db *gorm.DB,
	users repository.UserRepository,
	allocations repository.ProjectAllocationRepository,
	transactor ucshared.Transactor,
) (*StatusLogContainer, error) {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	repo := persistence.NewGormStatusLogRepository(db)
	clock := statuslogimpl.RealClock
	offDays := persistence.NewGormWeeklyOffDaysReader(db)
	leaves := persistence.NewGormLeaveRequestRepository(db)
	holidays := persistence.NewGormHolidayRepository(db)

	h := handlers.NewStatusLogHandler(handlers.StatusLogUseCases{
		Create:       statuslogimpl.NewCreateStatusLogUseCase(repo, allocations, transactor, clock),
		Update:       statuslogimpl.NewUpdateStatusLogUseCase(repo, allocations, transactor, clock),
		Delete:       statuslogimpl.NewDeleteStatusLogUseCase(repo, transactor, clock),
		ListMy:       statuslogimpl.NewListMyStatusLogsUseCase(repo, clock),
		MyDayStatus:  statuslogimpl.NewListMyDayStatusesUseCase(repo, offDays, leaves, holidays, clock),
		GetDaily:     statuslogimpl.NewGetDailyStatusLogsUseCase(repo, offDays, leaves, holidays, clock),
		GetMonthly:   statuslogimpl.NewGetMonthlyStatusLogsUseCase(repo, offDays, leaves, holidays, clock),
		ListEmployee: statuslogimpl.NewListEmployeeStatusLogsUseCase(repo, users, clock),
	})

	return &StatusLogContainer{Handler: h}, nil
}
