package container

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	leavetypeimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave/leavetype"
	leaverequestimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave/request"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// LeaveContainer wires leave types and leave requests.
type LeaveContainer struct {
	Handler        *handlers.LeaveTypeHandler
	RequestHandler *handlers.LeaveRequestHandler
}

// NewLeaveContainer wires the leave repositories, usecases and handlers.
func NewLeaveContainer(db *gorm.DB, recorder domainservice.AuditRecorder, transactor ucshared.Transactor, publisher domainservice.EventPublisher, app config.AppConfig, log zerolog.Logger) (*LeaveContainer, error) {
	urls := leavecommon.AppURLs{Default: app.FrontendURL, Admin: app.AdminURL, Employee: app.EmployeeURL}
	if urls.Default == "" && (urls.Admin == "" || urls.Employee == "") {
		log.Warn().Msg("app.frontend_url (or both app.admin_url and app.employee_url) is not set; leave email links will be relative paths")
	}
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	types := persistence.NewGormLeaveTypeRepository(db)
	requests := persistence.NewGormLeaveRequestRepository(db)
	approvers := persistence.NewGormLeaveApproverRepository(db)
	profiles := persistence.NewGormEmployeeProfileRepository(db)
	clock := leavecommon.RealClock
	if publisher == nil {
		publisher = &nopEventPublisher{}
	}

	th := handlers.NewLeaveTypeHandler(handlers.LeaveTypeUseCases{
		List:   leavetypeimpl.NewListLeaveTypesUseCase(types),
		Create: leavetypeimpl.NewCreateLeaveTypeUseCase(types, recorder, transactor),
		Update: leavetypeimpl.NewUpdateLeaveTypeUseCase(types, recorder, transactor),
	})
	rh := handlers.NewLeaveRequestHandler(handlers.LeaveRequestUseCases{
		ListApprovers: leaverequestimpl.NewListApproversUseCase(profiles, approvers),
		Apply:         leaverequestimpl.NewApplyLeaveUseCase(requests, types, profiles, approvers, persistence.NewGormWeeklyOffDaysReader(db), persistence.NewGormHolidayRepository(db), recorder, transactor, clock, publisher, urls),
		ListMy:        leaverequestimpl.NewListMyLeaveUseCase(requests),
		Cancel:        leaverequestimpl.NewCancelLeaveUseCase(requests, recorder, transactor, clock),
		ListTeam:      leaverequestimpl.NewListTeamLeaveUseCase(requests),
		ListAll:       leaverequestimpl.NewListAllLeaveUseCase(requests),
		Approve:       leaverequestimpl.NewApproveLeaveUseCase(requests, recorder, transactor, clock, publisher, urls),
		Reject:        leaverequestimpl.NewRejectLeaveUseCase(requests, recorder, transactor, clock, publisher, urls),
	})
	return &LeaveContainer{Handler: th, RequestHandler: rh}, nil
}
