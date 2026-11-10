package persistence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormOrganizationSettingsRepository implements repository.OrganizationSettingsRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormOrganizationSettingsRepository struct {
	db *gorm.DB
}

// NewGormOrganizationSettingsRepository constructs a GORM-backed OrganizationSettingsRepository.
func NewGormOrganizationSettingsRepository(db *gorm.DB) repository.OrganizationSettingsRepository {
	return &gormOrganizationSettingsRepository{db: db}
}

var _ repository.OrganizationSettingsRepository = (*gormOrganizationSettingsRepository)(nil)
var _ repository.WeeklyOffDaysReader = (*gormOrganizationSettingsRepository)(nil)

// formatIntArray renders a Postgres array literal, e.g. {0,6}.
func formatIntArray(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(n)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// parseIntList parses the comma-separated output of array_to_string.
func parseIntList(s string) ([]int, error) {
	out := []int{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	for _, p := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("parse weekly_off_days %q: %w", s, err)
		}
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}

// loadOffDays returns the tenant's off days, or the default {0,6} when the
// tenant has no settings row.
func loadOffDays(q *gorm.DB, tenantID uuid.UUID) ([]int, error) {
	var raw []*string
	if err := q.Raw(`SELECT array_to_string(weekly_off_days, ',') FROM organization_settings WHERE tenant_id = ?`, tenantID).Scan(&raw).Error; err != nil {
		return nil, err
	}
	if len(raw) == 0 || raw[0] == nil {
		return entity.DefaultWeeklyOffDays(), nil
	}
	return parseIntList(*raw[0])
}

// WeeklyOffDays implements repository.WeeklyOffDaysReader.
func (r *gormOrganizationSettingsRepository) WeeklyOffDays(c context.Context, tenantID uuid.UUID) ([]int, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return loadOffDays(database.DBFromContext(c, r.db), tenantID)
}

func (r *gormOrganizationSettingsRepository) read(c context.Context, tenantID uuid.UUID, lock bool) (*entity.OrganizationSettings, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrOrganizationSettingsNotFound
	}
	q := database.DBFromContext(c, r.db)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var s entity.OrganizationSettings
	if err := q.Where("tenant_id = ?", tenantID).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrOrganizationSettingsNotFound
		}
		return nil, err
	}
	days, err := loadOffDays(database.DBFromContext(c, r.db), tenantID)
	if err != nil {
		return nil, err
	}
	s.WeeklyOffDays = days
	return &s, nil
}

func (r *gormOrganizationSettingsRepository) GetByTenantID(c context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
	return r.read(c, tenantID, false)
}

func (r *gormOrganizationSettingsRepository) LockByTenantID(c context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
	return r.read(c, tenantID, true)
}

func (r *gormOrganizationSettingsRepository) EnsureDefaults(c context.Context, tenantID, actorID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	now := time.Now().UTC()
	s := entity.DefaultOrganizationSettings(tenantID)
	s.ID, s.CreatedAt, s.UpdatedAt = uuid.New(), now, now
	s.CreatedBy, s.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}}, DoNothing: true}).
		Create(s).Error
}

func (r *gormOrganizationSettingsRepository) Update(c context.Context, tenantID, actorID uuid.UUID, s *entity.OrganizationSettings) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	s.UpdatedAt = time.Now().UTC()
	s.UpdatedBy = &actorID
	offDays := s.WeeklyOffDays
	if len(offDays) == 0 {
		offDays = entity.DefaultWeeklyOffDays()
	}
	res := database.DBFromContext(c, r.db).
		Model(&entity.OrganizationSettings{}).
		Where("tenant_id = ?", tenantID).
		Updates(map[string]any{
			"employee_code_prefix":       s.EmployeeCodePrefix,
			"employee_code_padding":      s.EmployeeCodePadding,
			"employee_code_start_number": s.EmployeeCodeStartNumber,
			"employee_code_next_number":  s.EmployeeCodeNextNumber,
			"welcome_message":            s.WelcomeMessage,
			"weekly_off_days":            gorm.Expr("?::smallint[]", formatIntArray(offDays)),
			"updated_at":                 s.UpdatedAt,
			"updated_by":                 actorID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrOrganizationSettingsNotFound
	}
	return nil
}

// NextEmployeeCodeNumber atomically takes and increments the tenant's employee code
// counter. It is a single INSERT ... ON CONFLICT DO UPDATE ... RETURNING statement:
// the row lock it takes serialises concurrent allocations, and a tenant without a
// settings row gets one with the defaults. Run inside the caller's transaction, a
// rollback also rolls the counter back.
func (r *gormOrganizationSettingsRepository) NextEmployeeCodeNumber(c context.Context, tenantID uuid.UUID) (entity.EmployeeCodeCounter, error) {
	if tenantID == uuid.Nil {
		return entity.EmployeeCodeCounter{}, domainerrors.ErrUnauthorized
	}
	d := entity.DefaultOrganizationSettings(tenantID)
	var row struct {
		Prefix  string
		Padding int
		Taken   int64
	}
	err := database.DBFromContext(c, r.db).Raw(`
		INSERT INTO organization_settings AS os (tenant_id, employee_code_prefix, employee_code_padding, employee_code_start_number, employee_code_next_number, employee_code_last_issued_number)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (tenant_id) DO UPDATE SET employee_code_next_number = os.employee_code_next_number + 1,
			employee_code_last_issued_number = os.employee_code_next_number
		RETURNING employee_code_prefix AS prefix, employee_code_padding AS padding, employee_code_next_number - 1 AS taken`,
		tenantID, d.EmployeeCodePrefix, d.EmployeeCodePadding, d.EmployeeCodeStartNumber, d.EmployeeCodeStartNumber+1, d.EmployeeCodeStartNumber,
	).Scan(&row).Error
	if err != nil {
		return entity.EmployeeCodeCounter{}, err
	}
	return entity.EmployeeCodeCounter{Prefix: row.Prefix, Padding: row.Padding, Number: row.Taken}, nil
}

// NewGormWeeklyOffDaysReader constructs the GORM-backed WeeklyOffDaysReader.
func NewGormWeeklyOffDaysReader(db *gorm.DB) repository.WeeklyOffDaysReader {
	return &gormOrganizationSettingsRepository{db: db}
}
