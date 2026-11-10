package invitation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A nil tenant or actor is rejected by every management usecase before any work happens.
func TestManagementUseCases_RequireIdentity(t *testing.T) {
	f := newFixture()
	inv, _ := f.doInvite(t, "x@acme.com")
	bg := context.Background()
	list := NewListInvitationsUseCase(fakeInvRepo{f.s})

	for name, c := range map[string]struct{ tenant, actor uuid.UUID }{
		"no tenant": {uuid.Nil, f.adminID},
		"no actor":  {f.tenantA, uuid.Nil},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.resend().Execute(bg, adminActor(c.tenant, c.actor), inv.ID)
			assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
			assert.ErrorIs(t, f.revoke().Execute(bg, c.tenant, c.actor, inv.ID), domainerrors.ErrUnauthorized)
			if c.tenant == uuid.Nil {
				_, err = list.Execute(bg, c.tenant, invtypes.ListInvitationsQuery{})
				assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
			}
		})
	}
	assert.True(t, f.s.invitations[inv.ID].IsPending(), "rejected calls must not change the invitation")
}

func TestAccept_EmptyTokenIsInvalid(t *testing.T) {
	f := newFixture()
	err := f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: "  ", Password: "password123"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
}

// filterRecorder wraps the fake repo to record the filter the usecase passes down.
type filterRecorder struct {
	fakeInvRepo
	got repository.InvitationListFilter
}

func (r *filterRecorder) List(c context.Context, t uuid.UUID, f repository.InvitationListFilter) ([]*entity.InvitationListItem, int64, error) {
	r.got = f
	return r.fakeInvRepo.List(c, t, f)
}

func TestList_PaginationBounds(t *testing.T) {
	tests := []struct {
		name                  string
		page, size            int
		wantLimit, wantOffset int
	}{
		{"passes through", 3, 50, 50, 100},
		{"max allowed", 1, 100, 100, 0},
		{"over max is capped", 1, 101, 100, 0},
		{"zero size uses default", 1, 0, 20, 0},
		{"zero page clamps to 1", 0, 10, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			rec := &filterRecorder{fakeInvRepo: fakeInvRepo{f.s}}
			_, err := NewListInvitationsUseCase(rec).Execute(context.Background(), f.tenantA, invtypes.ListInvitationsQuery{Page: tt.page, PageSize: tt.size})
			require.NoError(t, err)
			assert.Equal(t, tt.wantLimit, rec.got.Limit)
			assert.Equal(t, tt.wantOffset, rec.got.Offset)
		})
	}
}

func TestList_StatusFilterValidationAndPassThrough(t *testing.T) {
	f := newFixture()
	rec := &filterRecorder{fakeInvRepo: fakeInvRepo{f.s}}
	uc := NewListInvitationsUseCase(rec)
	_, err := uc.Execute(context.Background(), f.tenantA, invtypes.ListInvitationsQuery{Status: "bogus"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInvitationFilter)

	_, err = uc.Execute(context.Background(), f.tenantA, invtypes.ListInvitationsQuery{Status: entity.InvitationStatusExpired, Search: "bob"})
	require.NoError(t, err)
	assert.Equal(t, entity.InvitationStatusExpired, rec.got.Status)
	assert.Equal(t, "bob", rec.got.Search)
	assert.False(t, rec.got.Now.IsZero())
}
