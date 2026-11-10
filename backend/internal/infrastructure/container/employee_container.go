package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	empimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

// EmployeeContainer wires employee management, /me and the profile gate.
type EmployeeContainer struct {
	Handler     *handlers.EmployeeHandler
	ProfileGate empuc.ProfileGateUseCase
}

// NewEmployeeContainer wires the employee repositories, usecases and handler.
func NewEmployeeContainer(db *gorm.DB, auth *AuthContainer, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*EmployeeContainer, error) {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	var (
		users       = auth.UserRepo
		profiles    = persistence.NewGormEmployeeProfileRepository(db)
		contacts    = persistence.NewGormEmergencyContactRepository(db)
		assignments = persistence.NewGormUserAssignmentRepository(db)
		directory   = persistence.NewGormEmployeeDirectoryRepository(db)
		orgNames    = persistence.NewGormOrgNameRepository(db)
		settings    = persistence.NewGormOrganizationSettingsRepository(db)
	)
	h := handlers.NewEmployeeHandler(handlers.EmployeeUseCases{
		List:          empimpl.NewListEmployeesUseCase(directory),
		Get:           empimpl.NewGetEmployeeUseCase(directory, profiles, contacts),
		Update:        empimpl.NewUpdateEmployeeUseCase(users, profiles, contacts, assignments, directory, auth.OrgReferenceRepo, settings, recorder, transactor),
		Assignments:   empimpl.NewListAssignmentsUseCase(users, assignments),
		ListContacts:  empimpl.NewListEmergencyContactsUseCase(users, contacts),
		CreateContact: empimpl.NewCreateEmergencyContactUseCase(users, profiles, settings, contacts, recorder, transactor),
		UpdateContact: empimpl.NewUpdateEmergencyContactUseCase(users, contacts, recorder, transactor),
		DeleteContact: empimpl.NewDeleteEmergencyContactUseCase(users, profiles, contacts, recorder, transactor),
		Deactivate:    empimpl.NewDeactivateEmployeeUseCase(users, profiles, contacts, directory, auth.RefreshTokenRepo, recorder, transactor),
		Activate:      empimpl.NewActivateEmployeeUseCase(users, profiles, contacts, directory, recorder, transactor),
		ChangeRole:    empimpl.NewChangeEmployeeRoleUseCase(users, auth.UserRoleRepo, auth.RoleRepo, profiles, contacts, directory, recorder, transactor),
		Me:            empimpl.NewGetMeUseCase(users, profiles, persistence.NewGormTenantRepository(db)),
		MyProfile:     empimpl.NewGetMyProfileUseCase(users, orgNames, profiles, contacts),
		UpdateMyProf:  empimpl.NewUpdateMyProfileUseCase(users, orgNames, profiles, settings, contacts, recorder, transactor),
	})
	return &EmployeeContainer{Handler: h, ProfileGate: empimpl.NewProfileGateUseCase(profiles)}, nil
}
