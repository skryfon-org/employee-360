package allocation

import (
	"context"
	"strings"
	"testing"
	"time"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"

	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
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

type fakeUsers struct {
	repository.UserRepository
	rows map[uuid.UUID]*entity.User
}

func (f *fakeUsers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	u, ok := f.rows[id]
	if !ok || u.TenantID != tenantID {
		return nil, domainerrors.ErrUserNotFound
	}
	return u, nil
}

type fakeAllocs struct {
	rows     map[uuid.UUID]*entity.ProjectAllocation
	projects *fakeProjects // used to fill ProjectStatus on joined reads
	users    *fakeUsers    // used to fill user names on ListByProject
}

func newFakeAllocs() *fakeAllocs { return &fakeAllocs{rows: map[uuid.UUID]*entity.ProjectAllocation{}} }

func (f *fakeAllocs) Create(_ context.Context, tenantID, _ uuid.UUID, a *entity.ProjectAllocation) error {
	a.TenantID = tenantID
	cp := *a
	f.rows[a.ID] = &cp
	return nil
}
func (f *fakeAllocs) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.ProjectAllocation, error) {
	a, ok := f.rows[id]
	if !ok || a.TenantID != tenantID || a.DeletedAt != nil {
		return nil, domainerrors.ErrAllocationNotFound
	}
	cp := *a
	return &cp, nil
}
func (f *fakeAllocs) ListByProject(_ context.Context, tenantID, projectID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithUser, error) {
	var out []*entity.AllocationWithUser
	for _, a := range f.rows {
		if a.TenantID == tenantID && a.ProjectID == projectID && a.DeletedAt == nil && fakeInScope(a, scope, asOf) {
			v := &entity.AllocationWithUser{ProjectAllocation: *a}
			if f.users != nil {
				if u, ok := f.users.rows[a.UserID]; ok && u.TenantID == tenantID {
					v.UserFirstName, v.UserLastName = u.FirstName, u.LastName
				}
			}
			out = append(out, v)
		}
	}
	return out, nil
}
func (f *fakeAllocs) CountByUserScopes(_ context.Context, tenantID, userID uuid.UUID, asOf time.Time) (entity.AllocationScopeCounts, error) {
	var c entity.AllocationScopeCounts
	for _, a := range f.rows {
		if a.TenantID != tenantID || a.UserID != userID || a.DeletedAt != nil {
			continue
		}
		if a.ActiveOn(asOf) {
			if p, ok := f.projects.rows[a.ProjectID]; ok && entity.IsLiveProjectStatus(p.Status) {
				c.Active++
			}
		}
		if a.StartDate.After(asOf) {
			c.Upcoming++
		}
		if a.EndDate != nil && a.EndDate.Before(asOf) {
			c.Past++
		}
	}
	return c, nil
}
func (f *fakeAllocs) ListByUserScope(_ context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithProject, error) {
	var out []*entity.AllocationWithProject
	for _, a := range f.rows {
		if a.TenantID != tenantID || a.UserID != userID || a.DeletedAt != nil {
			continue
		}
		switch scope {
		case entity.AllocationScopeActive:
			if !a.ActiveOn(asOf) {
				continue
			}
		case entity.AllocationScopeUpcoming:
			if !a.StartDate.After(asOf) {
				continue
			}
		case entity.AllocationScopePast:
			if a.EndDate == nil || !a.EndDate.Before(asOf) {
				continue
			}
		}
		v := &entity.AllocationWithProject{ProjectAllocation: *a}
		if f.projects != nil {
			if p, ok := f.projects.rows[a.ProjectID]; ok {
				v.ProjectName, v.ProjectCode, v.ProjectStatus = p.Name, p.Code, p.Status
			}
		}
		out = append(out, v)
	}
	return out, nil
}
func (f *fakeAllocs) HasOverlap(_ context.Context, tenantID, projectID, userID uuid.UUID, start time.Time, end *time.Time, excludeID uuid.UUID) (bool, error) {
	for _, a := range f.rows {
		if a.TenantID != tenantID || a.ProjectID != projectID || a.UserID != userID || a.DeletedAt != nil || a.ID == excludeID {
			continue
		}
		if (a.EndDate == nil || !a.EndDate.Before(start)) && (end == nil || !a.StartDate.After(*end)) {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeAllocs) Update(_ context.Context, tenantID, _ uuid.UUID, a *entity.ProjectAllocation) error {
	cp := *a
	f.rows[a.ID] = &cp
	return nil
}
func (f *fakeAllocs) SoftDelete(_ context.Context, tenantID, id, _ uuid.UUID) error {
	a, ok := f.rows[id]
	if !ok || a.TenantID != tenantID {
		return domainerrors.ErrAllocationNotFound
	}
	n := time.Now()
	a.DeletedAt = &n
	return nil
}

type allocEnv struct {
	tenant, actor, user, inactive uuid.UUID
	p1, p2                        *entity.Project
	projects                      *fakeProjects
	users                         *fakeUsers
	allocs                        *fakeAllocs
	rec                           *ucsharedtest.RecordingAuditRecorder
}

func newAllocEnv(t *testing.T) *allocEnv {
	t.Helper()
	now = func() time.Time { return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = time.Now })
	e := &allocEnv{tenant: uuid.New(), actor: uuid.New(), user: uuid.New(), inactive: uuid.New(),
		projects: newFakeProjects(), allocs: newFakeAllocs(), rec: &ucsharedtest.RecordingAuditRecorder{}}
	e.users = &fakeUsers{rows: map[uuid.UUID]*entity.User{
		e.user:     {ID: e.user, TenantID: e.tenant, IsActive: true, FirstName: "Ada", LastName: "Lovelace"},
		e.inactive: {ID: e.inactive, TenantID: e.tenant, IsActive: false},
	}}
	mk := func(code string) *entity.Project {
		p := &entity.Project{ID: uuid.New(), TenantID: e.tenant, Name: code, Code: code, Status: entity.ProjectStatusActive, StartDate: d("2026-01-01")}
		require.NoError(t, e.projects.Create(context.Background(), e.tenant, e.actor, p))
		return p
	}
	e.allocs.projects = e.projects
	e.allocs.users = e.users
	e.p1, e.p2 = mk("P1"), mk("P2")
	return e
}

func (e *allocEnv) create() allocuc.CreateAllocationUseCase {
	return NewCreateAllocationUseCase(e.projects, e.users, e.allocs, e.rec, nil)
}

func (e *allocEnv) in(p *entity.Project, pct int, start string, end *time.Time) prjtypes.CreateAllocationInput {
	return prjtypes.CreateAllocationInput{ProjectID: p.ID, UserID: e.user, AllocationPercent: pct, StartDate: d(start), EndDate: end}
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

// setStatus changes a seeded project's status directly in the fake repo.
func (e *allocEnv) setStatus(p *entity.Project, status string) { e.projects.rows[p.ID].Status = status }

func fakeInScope(a *entity.ProjectAllocation, scope entity.AllocationScope, asOf time.Time) bool {
	switch scope {
	case entity.AllocationScopeAll:
		return true
	case entity.AllocationScopeActive:
		return a.ActiveOn(asOf)
	case entity.AllocationScopeUpcoming:
		return a.StartDate.After(asOf)
	case entity.AllocationScopePast:
		return a.EndDate != nil && a.EndDate.Before(asOf)
	}
	return false
}

func (f *fakeAllocs) ListActiveByUser(c context.Context, tenantID, userID uuid.UUID, asOf time.Time) ([]*entity.AllocationWithProject, error) {
	return f.ListByUserScope(c, tenantID, userID, entity.AllocationScopeActive, asOf)
}
