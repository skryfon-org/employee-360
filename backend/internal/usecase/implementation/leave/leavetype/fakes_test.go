package leavetype

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

type fakeTypes struct {
	rows map[uuid.UUID]*entity.LeaveType
}

func newFakeTypes() *fakeTypes { return &fakeTypes{rows: map[uuid.UUID]*entity.LeaveType{}} }

func (f *fakeTypes) add(tenant uuid.UUID, code string, active bool) *entity.LeaveType {
	lt := &entity.LeaveType{ID: uuid.New(), TenantID: tenant, Name: code, Code: code, IsActive: active}
	f.rows[lt.ID] = lt
	return lt
}
func (f *fakeTypes) Create(_ context.Context, tenantID, _ uuid.UUID, lt *entity.LeaveType) error {
	lt.TenantID = tenantID
	cp := *lt
	f.rows[lt.ID] = &cp
	return nil
}
func (f *fakeTypes) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.LeaveType, error) {
	lt, ok := f.rows[id]
	if !ok || lt.TenantID != tenantID {
		return nil, domainerrors.ErrLeaveTypeNotFound
	}
	cp := *lt
	return &cp, nil
}
func (f *fakeTypes) List(_ context.Context, tenantID uuid.UUID, activeOnly bool) ([]*entity.LeaveType, error) {
	var out []*entity.LeaveType
	for _, lt := range f.rows {
		if lt.TenantID == tenantID && (!activeOnly || lt.IsActive) {
			out = append(out, lt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (f *fakeTypes) Update(_ context.Context, _, _ uuid.UUID, lt *entity.LeaveType) error {
	cp := *lt
	f.rows[lt.ID] = &cp
	return nil
}
func (f *fakeTypes) ExistsByCode(_ context.Context, tenantID uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	for _, lt := range f.rows {
		if lt.TenantID == tenantID && lt.Code == code && lt.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

type recordedAudit struct{ actions []string }

func (r *recordedAudit) Record(_ context.Context, _, _ uuid.UUID, action, _ string, _ uuid.UUID, _ map[string]any) error {
	r.actions = append(r.actions, action)
	return nil
}

var bgctx = context.Background()

type env struct {
	types *fakeTypes
	audit *recordedAudit
	admin uuid.UUID
}

func newEnv() *env {
	e := &env{types: newFakeTypes(), audit: &recordedAudit{}, admin: uuid.New()}
	return e
}
