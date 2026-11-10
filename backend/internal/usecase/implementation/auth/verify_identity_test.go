package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// viIdentityReader is a fake IdentityReader that mirrors the adapter's
// tenant+user scoping and counts calls.
type viIdentityReader struct {
	tenant    *entity.Tenant
	user      *entity.User
	roles     []string
	err       error
	calls     int
	gotTenant uuid.UUID
	gotUser   uuid.UUID
}

func (r *viIdentityReader) GetIdentityState(_ context.Context, tenantID, userID uuid.UUID) (*repository.IdentityState, error) {
	r.calls++
	r.gotTenant, r.gotUser = tenantID, userID
	if r.err != nil {
		return nil, r.err
	}
	if r.tenant == nil || r.tenant.ID != tenantID || r.tenant.DeletedAt != nil {
		return nil, domainerrors.ErrTenantNotFound
	}
	st := &repository.IdentityState{TenantActive: r.tenant.IsActive, RoleNames: []string{}}
	if r.user != nil && r.user.ID == userID && r.user.TenantID == tenantID && r.user.DeletedAt == nil {
		st.UserFound, st.UserActive = true, r.user.IsActive
		st.RoleNames = append(st.RoleNames, r.roles...)
	}
	return st, nil
}

func TestVerifyIdentityUseCase(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	now := time.Now()
	dbErr := errors.New("connection refused")

	okTenant := func() *entity.Tenant { return &entity.Tenant{ID: tenantID, IsActive: true} }
	okUser := func() *entity.User { return &entity.User{ID: userID, TenantID: tenantID, IsActive: true} }

	tests := []struct {
		name      string
		reader    *viIdentityReader
		wantErr   error // matched with errors.Is
		wantRoles []string
	}{
		{"valid, current roles loaded", &viIdentityReader{tenant: okTenant(), user: okUser(), roles: []string{"admin", "employee"}}, nil, []string{"admin", "employee"}},
		{"valid, no roles", &viIdentityReader{tenant: okTenant(), user: okUser()}, nil, []string{}},
		{"tenant missing", &viIdentityReader{user: okUser()}, domainerrors.ErrUnauthorized, nil},
		{"tenant inactive", &viIdentityReader{tenant: &entity.Tenant{ID: tenantID, IsActive: false}, user: okUser()}, domainerrors.ErrUnauthorized, nil},
		{"tenant soft-deleted", &viIdentityReader{tenant: &entity.Tenant{ID: tenantID, IsActive: true, DeletedAt: &now}, user: okUser()}, domainerrors.ErrUnauthorized, nil},
		{"user unknown", &viIdentityReader{tenant: okTenant()}, domainerrors.ErrUnauthorized, nil},
		{"user in another tenant", &viIdentityReader{tenant: okTenant(), user: &entity.User{ID: userID, TenantID: uuid.New(), IsActive: true}}, domainerrors.ErrUnauthorized, nil},
		{"user inactive", &viIdentityReader{tenant: okTenant(), user: &entity.User{ID: userID, TenantID: tenantID, IsActive: false}}, domainerrors.ErrUnauthorized, nil},
		{"user soft-deleted", &viIdentityReader{tenant: okTenant(), user: &entity.User{ID: userID, TenantID: tenantID, IsActive: true, DeletedAt: &now}}, domainerrors.ErrUnauthorized, nil},
		{"reader error is not unauthorized", &viIdentityReader{err: dbErr}, dbErr, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewVerifyIdentityUseCase(tc.reader)
			got, err := uc.Execute(context.Background(), tenantID, userID)

			if tc.reader.calls != 1 {
				t.Errorf("identity reader calls = %d, want exactly 1", tc.reader.calls)
			}
			if tc.reader.gotTenant != tenantID || tc.reader.gotUser != userID {
				t.Errorf("lookup scoped to tenant %s user %s", tc.reader.gotTenant, tc.reader.gotUser)
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr == dbErr && errors.Is(err, domainerrors.ErrUnauthorized) {
					t.Fatal("infrastructure error must not be reported as unauthorized")
				}
				if got != nil {
					t.Fatal("expected nil identity on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TenantID != tenantID || got.UserID != userID {
				t.Errorf("identity = %+v", got)
			}
			if len(got.Roles) != len(tc.wantRoles) {
				t.Fatalf("roles = %v, want %v", got.Roles, tc.wantRoles)
			}
			for i, r := range tc.wantRoles {
				if got.Roles[i] != r {
					t.Errorf("roles = %v, want %v", got.Roles, tc.wantRoles)
				}
			}
		})
	}
}

func TestVerifyIdentityUseCase_NilUUIDsRejectedWithoutLookup(t *testing.T) {
	users := &viIdentityReader{}
	uc := NewVerifyIdentityUseCase(users)
	if _, err := uc.Execute(context.Background(), uuid.Nil, uuid.New()); !errors.Is(err, domainerrors.ErrUnauthorized) {
		t.Errorf("nil tenant: err = %v", err)
	}
	if _, err := uc.Execute(context.Background(), uuid.New(), uuid.Nil); !errors.Is(err, domainerrors.ErrUnauthorized) {
		t.Errorf("nil user: err = %v", err)
	}
	if users.calls != 0 {
		t.Error("repository must not be hit for nil UUIDs")
	}
}
