package organization

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

// fakeOffReader is tenant-keyed; unknown tenants get the default {0,6}.
type fakeOffReader struct{ days map[uuid.UUID][]int }

func (f fakeOffReader) WeeklyOffDays(_ context.Context, t uuid.UUID) ([]int, error) {
	if d, ok := f.days[t]; ok {
		return d, nil
	}
	return []int{0, 6}, nil
}

func TestGetWeeklyOffDays_ScopedToTenant(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	uc := NewGetWeeklyOffDaysUseCase(fakeOffReader{days: map[uuid.UUID][]int{a: {5, 6}, b: {0}}})

	got, err := uc.Execute(context.Background(), a)
	require.NoError(t, err)
	assert.Equal(t, []int{5, 6}, got)

	got, err = uc.Execute(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, []int{0}, got)
}

func TestGetWeeklyOffDays_DefaultWhenNoRow(t *testing.T) {
	uc := NewGetWeeklyOffDaysUseCase(fakeOffReader{})
	got, err := uc.Execute(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, []int{0, 6}, got)
}

func TestGetWeeklyOffDays_NilTenantUnauthorized(t *testing.T) {
	uc := NewGetWeeklyOffDaysUseCase(fakeOffReader{})
	_, err := uc.Execute(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}
