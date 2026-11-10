package holiday

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestNormalizeHolidayDate(t *testing.T) {
	t.Run("zero time returns ErrHolidayDateInvalid", func(t *testing.T) {
		_, err := NormalizeHolidayDate(time.Time{})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDateInvalid)
	})

	t.Run("year before 2000 returns ErrHolidayYearInvalid", func(t *testing.T) {
		d := time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC)
		_, err := NormalizeHolidayDate(d)
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)
	})

	t.Run("year after 2100 returns ErrHolidayYearInvalid", func(t *testing.T) {
		d := time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := NormalizeHolidayDate(d)
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)
	})

	t.Run("boundary years 2000 and 2100 are valid", func(t *testing.T) {
		d2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
		norm2000, err := NormalizeHolidayDate(d2000)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), norm2000)

		d2100 := time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC)
		norm2100, err := NormalizeHolidayDate(d2100)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC), norm2100)
	})

	t.Run("truncates non-midnight time and converts non-UTC timezone to UTC midnight", func(t *testing.T) {
		loc := time.FixedZone("Asia/Kolkata", 5*3600+30*60)
		d := time.Date(2026, 8, 15, 14, 35, 42, 999999999, loc)

		norm, err := NormalizeHolidayDate(d)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), norm)
		assert.Equal(t, time.UTC, norm.Location())
	})
}
