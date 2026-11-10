package project

import (
	"context"
	"strings"
	"time"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

type fakeProjects struct{ rows map[uuid.UUID]*entity.Project }

func newFakeProjects() *fakeProjects { return &fakeProjects{rows: map[uuid.UUID]*entity.Project{}} }

func (f *fakeProjects) Create(_ context.Context, tenantID, _ uuid.UUID, p *entity.Project) error {
	p.TenantID = tenantID
	cp := *p
	f.rows[p.ID] = &cp
	return nil
}
func (f *fakeProjects) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.Project, error) {
	p, ok := f.rows[id]
	if !ok || p.TenantID != tenantID {
		return nil, domainerrors.ErrProjectNotFound
	}
	cp := *p
	return &cp, nil
}
func (f *fakeProjects) GetByIDForUpdate(c context.Context, t, id uuid.UUID) (*entity.Project, error) {
	return f.GetByID(c, t, id)
}
func (f *fakeProjects) List(_ context.Context, tenantID uuid.UUID, fl repository.ProjectFilter, limit, offset int) ([]*entity.Project, int64, error) {
	var out []*entity.Project
	for _, p := range f.rows {
		if p.TenantID != tenantID || (fl.Status != "" && p.Status != fl.Status) {
			continue
		}
		if fl.Search != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(fl.Search)) {
			continue
		}
		out = append(out, p)
	}
	return out, int64(len(out)), nil
}
func (f *fakeProjects) Update(_ context.Context, tenantID, _ uuid.UUID, p *entity.Project) error {
	cp := *p
	f.rows[p.ID] = &cp
	return nil
}
func (f *fakeProjects) ExistsByCode(_ context.Context, tenantID uuid.UUID, code string) (bool, error) {
	for _, p := range f.rows {
		if p.TenantID == tenantID && p.Code == code {
			return true, nil
		}
	}
	return false, nil
}

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func dp(s string) *time.Time { t := d(s); return &t }

func opt[T any](v T) emptypes.Optional[T] { return emptypes.Optional[T]{Set: true, Value: v} }

func optNull[T any]() emptypes.Optional[T] { return emptypes.Optional[T]{Set: true, Null: true} }

func optDate(s string) emptypes.Optional[emptypes.Date] {
	return opt(emptypes.Date{Time: d(s)})
}

func goodCreate() prjtypes.CreateProjectInput {
	return prjtypes.CreateProjectInput{Name: "Apollo", Code: "apo-1", StartDate: d("2026-01-01")}
}
