package employee

import (
	"testing"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEmployee_CrossTenant404(t *testing.T) {
	f := newFixture()
	other := f.w.tenant("Other")
	_, err := f.w.newGetEmployee().Execute(bg, other, f.employee.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	d, err := f.w.newGetEmployee().Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Nil(t, d.Profile)
}
