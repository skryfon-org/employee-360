package container

import (
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	holidayimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

// HolidayContainer encapsulates dependencies, repositories, and usecases for Holidays.
type HolidayContainer struct {
	HolidayRepo  repository.HolidayRepository
	CategoryRepo repository.HolidayCategoryRepository
	SettingsRepo repository.HolidayCalendarSettingsRepository

	CreateUseCase     holidayuc.CreateHolidayUseCase
	GetUseCase        holidayuc.GetHolidayUseCase
	ListByYearUseCase holidayuc.ListHolidaysByYearUseCase
	UpdateUseCase     holidayuc.UpdateHolidayUseCase
	DeleteUseCase     holidayuc.DeleteHolidayUseCase

	CreateCategoryUseCase holidayuc.CreateCategoryUseCase
	GetCategoryUseCase    holidayuc.GetCategoryUseCase
	ListCategoriesUseCase holidayuc.ListCategoriesUseCase
	UpdateCategoryUseCase holidayuc.UpdateCategoryUseCase
	DeleteCategoryUseCase holidayuc.DeleteCategoryUseCase

	GetSettingsUseCase    holidayuc.GetSettingsUseCase
	UpdateSettingsUseCase holidayuc.UpdateSettingsUseCase

	Handler *handlers.HolidayHandler
}

// NewHolidayContainer initializes and wires all holiday-related repositories and usecases.
func NewHolidayContainer(db *gorm.DB, recorder domainservice.AuditRecorder, transactor ucshared.Transactor) (*HolidayContainer, error) {
	holidayRepo := persistence.NewGormHolidayRepository(db)
	categoryRepo := persistence.NewGormHolidayCategoryRepository(db)
	settingsRepo := persistence.NewGormHolidayCalendarSettingsRepository(db)

	if recorder == nil && db != nil {
		recorder = infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db))
	}
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}

	createUC := holidayimpl.NewCreateHolidayUseCase(holidayRepo, categoryRepo, recorder, transactor)
	getUC := holidayimpl.NewGetHolidayUseCase(holidayRepo)
	listUC := holidayimpl.NewListHolidaysByYearUseCase(holidayRepo)
	updateUC := holidayimpl.NewUpdateHolidayUseCase(holidayRepo, categoryRepo, recorder, transactor)
	deleteUC := holidayimpl.NewDeleteHolidayUseCase(holidayRepo, recorder, transactor)

	createCategoryUC := holidayimpl.NewCreateCategoryUseCase(categoryRepo, recorder, transactor)
	getCategoryUC := holidayimpl.NewGetCategoryUseCase(categoryRepo)
	listCategoriesUC := holidayimpl.NewListCategoriesUseCase(categoryRepo)
	updateCategoryUC := holidayimpl.NewUpdateCategoryUseCase(categoryRepo, recorder, transactor)
	deleteCategoryUC := holidayimpl.NewDeleteCategoryUseCase(categoryRepo, recorder, transactor)

	getSettingsUC := holidayimpl.NewGetSettingsUseCase(settingsRepo)
	updateSettingsUC := holidayimpl.NewUpdateSettingsUseCase(settingsRepo, recorder, transactor)

	handler := handlers.NewHolidayHandler(
		createUC,
		getUC,
		listUC,
		updateUC,
		deleteUC,
		createCategoryUC,
		getCategoryUC,
		listCategoriesUC,
		updateCategoryUC,
		deleteCategoryUC,
		getSettingsUC,
		updateSettingsUC,
	)

	return &HolidayContainer{
		HolidayRepo:           holidayRepo,
		CategoryRepo:          categoryRepo,
		SettingsRepo:          settingsRepo,
		CreateUseCase:         createUC,
		GetUseCase:            getUC,
		ListByYearUseCase:     listUC,
		UpdateUseCase:         updateUC,
		DeleteUseCase:         deleteUC,
		CreateCategoryUseCase: createCategoryUC,
		GetCategoryUseCase:    getCategoryUC,
		ListCategoriesUseCase: listCategoriesUC,
		UpdateCategoryUseCase: updateCategoryUC,
		DeleteCategoryUseCase: deleteCategoryUC,
		GetSettingsUseCase:    getSettingsUC,
		UpdateSettingsUseCase: updateSettingsUC,
		Handler:               handler,
	}, nil
}
