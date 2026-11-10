package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	orgimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/organization"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

// OrganizationContainer wires the organization settings endpoints.
type OrganizationContainer struct {
	Handler *handlers.OrganizationHandler
	// WeeklyOffDays serves the employee-readable weekly off days.
	WeeklyOffDays *handlers.WeeklyOffDaysHandler
}

// NewOrganizationContainer wires the settings repository, usecases and handler.
func NewOrganizationContainer(db *gorm.DB, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*OrganizationContainer, error) {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	repo := persistence.NewGormOrganizationSettingsRepository(db)
	return &OrganizationContainer{
		WeeklyOffDays: handlers.NewWeeklyOffDaysHandler(orgimpl.NewGetWeeklyOffDaysUseCase(persistence.NewGormWeeklyOffDaysReader(db))),
		Handler: handlers.NewOrganizationHandler(
			orgimpl.NewGetSettingsUseCase(repo),
			orgimpl.NewUpdateSettingsUseCase(repo, recorder, transactor),
		)}, nil
}
