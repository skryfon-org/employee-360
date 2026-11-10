//go:build integration

package persistence

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Raw-SQL tests for the DB-level constraints from migrations 000021-000023.
// They deliberately bypass repositories/usecases so only the schema is tested.

func insertCategory(db *gorm.DB, tenantID uuid.UUID, name, color string) (uuid.UUID, error) {
	id := uuid.New()
	err := db.Exec(
		`INSERT INTO holiday_categories (id, tenant_id, name, color) VALUES (?, ?, ?, ?)`,
		id, tenantID, name, color).Error
	return id, err
}

func insertHoliday(db *gorm.DB, tenantID, categoryID uuid.UUID, name, date string) (uuid.UUID, error) {
	id := uuid.New()
	err := db.Exec(
		`INSERT INTO holidays (id, tenant_id, category_id, name, date) VALUES (?, ?, ?, ?, ?)`,
		id, tenantID, categoryID, name, date).Error
	return id, err
}

func TestHolidayCategoriesConstraints(t *testing.T) {
	db := setupTestDB(t)
	tenantA := createTestTenant(t, db, "hc-con-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "hc-con-b-"+uuid.NewString())

	t.Run("color hex check", func(t *testing.T) {
		for _, ok := range []string{"#15803D", "#abcdef", "#AbC123"} {
			_, err := insertCategory(db, tenantA.ID, "ok-"+ok, ok)
			require.NoError(t, err, ok)
		}
		for _, bad := range []string{"15803D", "#15803", "#GGGGGG", "red", "", "#12 456"} {
			_, err := insertCategory(db, tenantA.ID, "bad-"+bad, bad)
			require.Error(t, err, "color %q must be rejected", bad)
			assert.Contains(t, err.Error(), "chk_holiday_categories_color_hex")
		}
	})

	t.Run("color longer than 7 chars is rejected by VARCHAR(7)", func(t *testing.T) {
		_, err := insertCategory(db, tenantA.ID, "too-long", "#15803DD")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SQLSTATE 22001")
	})

	t.Run("unique lower(name) per tenant among live rows", func(t *testing.T) {
		_, err := insertCategory(db, tenantA.ID, "Public", "#111111")
		require.NoError(t, err)

		_, err = insertCategory(db, tenantA.ID, "PUBLIC", "#222222")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uq_holiday_categories_tenant_id_lower_name")

		// Same name in another tenant is fine.
		_, err = insertCategory(db, tenantB.ID, "Public", "#111111")
		require.NoError(t, err)
	})

	t.Run("soft-deleted row frees the name", func(t *testing.T) {
		id, err := insertCategory(db, tenantA.ID, "Optional", "#333333")
		require.NoError(t, err)

		_, err = insertCategory(db, tenantA.ID, "optional", "#333333")
		require.Error(t, err)

		require.NoError(t, db.Exec(`UPDATE holiday_categories SET deleted_at = NOW() WHERE id = ?`, id).Error)

		_, err = insertCategory(db, tenantA.ID, "OPTIONAL", "#444444")
		require.NoError(t, err, "name must be reusable after soft delete")

		// Live name is taken again.
		_, err = insertCategory(db, tenantA.ID, "Optional", "#555555")
		require.Error(t, err)

		// Multiple soft-deleted rows with the same name may coexist.
		id2, err := insertCategory(db, tenantB.ID, "Gone", "#666666")
		require.NoError(t, err)
		require.NoError(t, db.Exec(`UPDATE holiday_categories SET deleted_at = NOW() WHERE id = ?`, id2).Error)
		id3, err := insertCategory(db, tenantB.ID, "Gone", "#666666")
		require.NoError(t, err)
		require.NoError(t, db.Exec(`UPDATE holiday_categories SET deleted_at = NOW() WHERE id = ?`, id3).Error)
	})
}

func TestHolidaysConstraints(t *testing.T) {
	db := setupTestDB(t)
	tenantA := createTestTenant(t, db, "h-con-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "h-con-b-"+uuid.NewString())

	catA, err := insertCategory(db, tenantA.ID, "Public", "#15803D")
	require.NoError(t, err)
	catB, err := insertCategory(db, tenantB.ID, "Public", "#15803D")
	require.NoError(t, err)

	t.Run("unique tenant+date+lower(name) among live rows", func(t *testing.T) {
		id, err := insertHoliday(db, tenantA.ID, catA, "New Year", "2026-01-01")
		require.NoError(t, err)

		_, err = insertHoliday(db, tenantA.ID, catA, "NEW YEAR", "2026-01-01")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uq_holidays_tenant_id_date_lower_name")

		// Different date, different name, and different tenant are all allowed.
		_, err = insertHoliday(db, tenantA.ID, catA, "New Year", "2027-01-01")
		require.NoError(t, err)
		_, err = insertHoliday(db, tenantA.ID, catA, "Other", "2026-01-01")
		require.NoError(t, err)
		_, err = insertHoliday(db, tenantB.ID, catB, "New Year", "2026-01-01")
		require.NoError(t, err)

		// Soft delete frees the slot.
		require.NoError(t, db.Exec(`UPDATE holidays SET deleted_at = NOW() WHERE id = ?`, id).Error)
		_, err = insertHoliday(db, tenantA.ID, catA, "new year", "2026-01-01")
		require.NoError(t, err)
	})

	t.Run("category FK is ON DELETE RESTRICT", func(t *testing.T) {
		cat, err := insertCategory(db, tenantA.ID, "Restricted", "#777777")
		require.NoError(t, err)
		hid, err := insertHoliday(db, tenantA.ID, cat, "Held", "2026-05-05")
		require.NoError(t, err)

		err = db.Exec(`DELETE FROM holiday_categories WHERE id = ?`, cat).Error
		require.Error(t, err, "hard delete of a referenced category must fail")
		assert.Contains(t, err.Error(), "foreign key")

		// A nonexistent category is also rejected.
		_, err = insertHoliday(db, tenantA.ID, uuid.New(), "Orphan", "2026-06-06")
		require.Error(t, err)

		// Once the holiday is hard-deleted the category can go.
		require.NoError(t, db.Exec(`DELETE FROM holidays WHERE id = ?`, hid).Error)
		require.NoError(t, db.Exec(`DELETE FROM holiday_categories WHERE id = ?`, cat).Error)
	})
}

func TestHolidayCalendarSettingsConstraints(t *testing.T) {
	db := setupTestDB(t)
	tenantA := createTestTenant(t, db, "hcs-con-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "hcs-con-b-"+uuid.NewString())

	insert := func(tenantID uuid.UUID, view string, year any, wsd int) error {
		return db.Exec(
			`INSERT INTO holiday_calendar_settings (tenant_id, default_view, default_year, week_start_day)
			 VALUES (?, ?, ?, ?)`, tenantID, view, year, wsd).Error
	}

	t.Run("default_view check", func(t *testing.T) {
		err := insert(tenantA.ID, "grid", nil, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_holiday_calendar_settings_default_view")
	})

	t.Run("default_year check", func(t *testing.T) {
		for _, y := range []int{1999, 2101, 0} {
			err := insert(tenantA.ID, "calendar", y, 1)
			require.Error(t, err, "year %d", y)
			assert.Contains(t, err.Error(), "chk_holiday_calendar_settings_default_year")
		}
	})

	t.Run("week_start_day check", func(t *testing.T) {
		for _, d := range []int{-1, 7} {
			err := insert(tenantA.ID, "calendar", nil, d)
			require.Error(t, err, "week_start_day %d", d)
			assert.Contains(t, err.Error(), "chk_holiday_calendar_settings_week_start_day")
		}
	})

	t.Run("boundaries accepted and tenant_id unique", func(t *testing.T) {
		require.NoError(t, insert(tenantA.ID, "list", 2000, 0))

		err := insert(tenantA.ID, "calendar", 2100, 6)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uq_holiday_calendar_settings_tenant_id")

		// Boundary values on another tenant, and NULL year is allowed.
		require.NoError(t, insert(tenantB.ID, "calendar", 2100, 6))
		var n int64
		require.NoError(t, db.Raw(`SELECT COUNT(*) FROM holiday_calendar_settings WHERE tenant_id IN (?, ?)`,
			tenantA.ID, tenantB.ID).Scan(&n).Error)
		assert.EqualValues(t, 2, n)

		require.NoError(t, db.Exec(`UPDATE holiday_calendar_settings SET default_year = NULL WHERE tenant_id = ?`, tenantA.ID).Error)
	})
}
