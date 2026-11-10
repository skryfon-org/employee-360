package tenant

import (
	"context"
	"errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
)

// fakeStore is an in-memory DB; the transactor snapshots/restores it to
// emulate commit/rollback.
type fakeStore struct {
	tenants   map[uuid.UUID]*entity.Tenant
	domains   []*entity.TenantDomain // includes soft-deleted
	audits    []*entity.AuditLog
	users     []*entity.User // tenant users (may be soft-deleted)
	failAudit bool
}

func newStore() *fakeStore { return &fakeStore{tenants: map[uuid.UUID]*entity.Tenant{}} }

type snap struct {
	tenants map[uuid.UUID]entity.Tenant
	domains []entity.TenantDomain
	audits  int
}

func (s *fakeStore) snapshot() snap {
	sn := snap{tenants: map[uuid.UUID]entity.Tenant{}, audits: len(s.audits)}
	for k, v := range s.tenants {
		sn.tenants[k] = *v
	}
	for _, d := range s.domains {
		sn.domains = append(sn.domains, *d)
	}
	return sn
}

func (s *fakeStore) restore(sn snap) {
	s.tenants = map[uuid.UUID]*entity.Tenant{}
	for k, v := range sn.tenants {
		v := v
		s.tenants[k] = &v
	}
	s.domains = nil
	for _, d := range sn.domains {
		d := d
		s.domains = append(s.domains, &d)
	}
	s.audits = s.audits[:sn.audits]
}

type fakeTx struct{ s *fakeStore }

func (f fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	sn := f.s.snapshot()
	if err := fn(ctx); err != nil {
		f.s.restore(sn)
		return err
	}
	return nil
}

type tenantRepoFake struct{ s *fakeStore }

func (r tenantRepoFake) live(id uuid.UUID) (*entity.Tenant, error) {
	t, ok := r.s.tenants[id]
	if !ok || t.DeletedAt != nil {
		return nil, domainerrors.ErrTenantNotFound
	}
	return t, nil
}
func (r tenantRepoFake) GetByID(_ context.Context, id uuid.UUID) (*entity.Tenant, error) {
	t, err := r.live(id)
	if err != nil {
		return nil, err
	}
	c := *t
	return &c, nil
}
func (r tenantRepoFake) LockByID(c context.Context, id uuid.UUID) (*entity.Tenant, error) {
	return r.GetByID(c, id)
}
func (r tenantRepoFake) UpdateName(_ context.Context, id uuid.UUID, name string, actor uuid.UUID, at time.Time) error {
	t, err := r.live(id)
	if err != nil {
		return err
	}
	t.Name, t.UpdatedBy, t.UpdatedAt = name, &actor, at
	return nil
}

type domainRepoFake struct{ s *fakeStore }

func (r domainRepoFake) Create(_ context.Context, d *entity.TenantDomain) error {
	for _, x := range r.s.domains {
		if x.Domain == d.Domain && x.DeletedAt == nil {
			return domainerrors.ErrDomainAlreadyExists
		}
	}
	c := *d
	r.s.domains = append(r.s.domains, &c)
	return nil
}
func (r domainRepoFake) ListByTenantID(_ context.Context, tid uuid.UUID) ([]*entity.TenantDomain, error) {
	var out []*entity.TenantDomain
	for _, d := range r.s.domains {
		if d.TenantID == tid && d.DeletedAt == nil {
			out = append(out, d)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r domainRepoFake) CountByTenantID(c context.Context, tid uuid.UUID) (int64, error) {
	ds, _ := r.ListByTenantID(c, tid)
	return int64(len(ds)), nil
}
func (r domainRepoFake) GetByID(_ context.Context, tid, id uuid.UUID) (*entity.TenantDomain, error) {
	for _, d := range r.s.domains {
		if d.ID == id && d.TenantID == tid && d.DeletedAt == nil {
			return d, nil
		}
	}
	return nil, domainerrors.ErrDomainNotFound
}
func (r domainRepoFake) UpdateDomain(c context.Context, tid, id uuid.UUID, domain string, actor uuid.UUID, at time.Time) error {
	d, err := r.GetByID(c, tid, id)
	if err != nil {
		return err
	}
	for _, x := range r.s.domains {
		if x.Domain == domain && x.ID != id {
			return domainerrors.ErrDomainAlreadyExists
		}
	}
	d.Domain, d.UpdatedBy, d.UpdatedAt = domain, &actor, at
	return nil
}
func (r domainRepoFake) CountUsersOnDomain(_ context.Context, tid uuid.UUID, domain string) (int64, error) {
	var n int64
	for _, u := range r.s.users {
		if u.TenantID != tid || u.DeletedAt != nil {
			continue
		}
		if i := strings.LastIndex(u.Email, "@"); i >= 0 && strings.EqualFold(u.Email[i+1:], domain) {
			n++
		}
	}
	return n, nil
}
func (r domainRepoFake) SoftDelete(c context.Context, tid, id, actor uuid.UUID, at time.Time) error {
	d, err := r.GetByID(c, tid, id)
	if err != nil {
		return err
	}
	d.DeletedAt, d.DeletedBy = &at, &actor
	return nil
}

// recorder fakes the AuditRecorder; entries land in the store so transactional
// rollback drops them, and failAudit simulates an audit-write failure.
func (s *fakeStore) recorder() *ucsharedtest.RecordingAuditRecorder {
	return &ucsharedtest.RecordingAuditRecorder{
		ErrFn: func() error {
			if s.failAudit {
				return errors.New("audit down")
			}
			return nil
		},
		OnRecord: func(l *entity.AuditLog) { s.audits = append(s.audits, l) },
	}
}

type env struct {
	s     *fakeStore
	tr    tenantRepoFake
	dr    domainRepoFake
	ar    domainservice.AuditRecorder
	tx    fakeTx
	actor uuid.UUID
	c     context.Context
}

func newEnv() *env {
	s := newStore()
	return &env{s: s, tr: tenantRepoFake{s}, dr: domainRepoFake{s}, ar: s.recorder(), tx: fakeTx{s}, actor: uuid.New(), c: context.Background()}
}

func (e *env) seedTenant(active bool, domains ...string) uuid.UUID {
	id := uuid.New()
	e.s.tenants[id] = &entity.Tenant{ID: id, Name: "T-" + id.String()[:4], IsActive: active}
	for i, d := range domains {
		e.s.domains = append(e.s.domains, &entity.TenantDomain{ID: uuid.New(), TenantID: id, Domain: d, CreatedAt: time.Unix(int64(i), 0)})
	}
	return id
}

func (e *env) lastAudit(t *testing.T) *entity.AuditLog {
	t.Helper()
	require.NotEmpty(t, e.s.audits)
	return e.s.audits[len(e.s.audits)-1]
}

func TestRename(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	uc := NewRenameTenantUseCase(e.tr, e.ar, e.tx)
	got, err := uc.Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: " New "})
	require.NoError(t, err)
	assert.Equal(t, "New", got.Name)
	assert.Equal(t, "New", e.s.tenants[id].Name)
	assert.Equal(t, e.actor, *e.s.tenants[id].UpdatedBy)
	assert.Equal(t, "tenant.rename", e.lastAudit(t).Action)

	_, err = uc.Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: " "})
	require.ErrorIs(t, err, domainerrors.ErrInvalidTenantName)
	_, err = uc.Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: strings.Repeat("a", tenanttypes.MaxNameLen+1)})
	require.ErrorIs(t, err, domainerrors.ErrInvalidTenantName)
	_, err = uc.Execute(e.c, uuid.Nil, id, tenanttypes.UpdateTenantRequest{Name: "x"})
	require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	_, err = uc.Execute(e.c, e.actor, uuid.New(), tenanttypes.UpdateTenantRequest{Name: "x"})
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	assert.Len(t, e.s.audits, 1)

	now := time.Now()
	e.s.tenants[id].DeletedAt = &now
	_, err = uc.Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: "x"})
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
}

func TestDomains_AddAndList(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	other := e.seedTenant(true, "b.com")
	add := NewAddTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)

	d, err := add.Execute(e.c, e.actor, id, tenanttypes.AddDomainRequest{Domain: " A2.Example.ORG "})
	require.NoError(t, err)
	assert.Equal(t, "a2.example.org", d.Domain)
	assert.Equal(t, id, d.TenantID)
	assert.Equal(t, e.actor, *d.CreatedBy)
	a := e.lastAudit(t)
	assert.Equal(t, "tenant.domain.add", a.Action)
	assert.Equal(t, d.ID, a.EntityID)
	assert.Equal(t, id, a.TenantID)

	// duplicate across tenants -> conflict, same tenant -> conflict
	_, err = add.Execute(e.c, e.actor, id, tenanttypes.AddDomainRequest{Domain: "b.com"})
	require.ErrorIs(t, err, domainerrors.ErrDomainAlreadyExists)
	_, err = add.Execute(e.c, e.actor, other, tenanttypes.AddDomainRequest{Domain: "A.com"})
	require.ErrorIs(t, err, domainerrors.ErrDomainAlreadyExists)
	assert.Len(t, e.s.audits, 1)

	_, err = add.Execute(e.c, e.actor, id, tenanttypes.AddDomainRequest{Domain: "nope"})
	require.ErrorIs(t, err, domainerrors.ErrInvalidDomain)
	_, err = add.Execute(e.c, e.actor, uuid.New(), tenanttypes.AddDomainRequest{Domain: "c.com"})
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)

	ds, err := NewListTenantDomainsUseCase(e.tr, e.dr).Execute(e.c, id)
	require.NoError(t, err)
	assert.Len(t, ds, 2)
	_, err = NewListTenantDomainsUseCase(e.tr, e.dr).Execute(e.c, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
}

func TestDomains_Remove(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com", "b.com")
	other := e.seedTenant(true, "c.com")
	rm := NewRemoveTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)
	firstID, secondID, otherDomID := e.s.domains[0].ID, e.s.domains[1].ID, e.s.domains[2].ID
	byID := func(id uuid.UUID) *entity.TenantDomain {
		for _, d := range e.s.domains {
			if d.ID == id {
				return d
			}
		}
		return nil
	}

	// A domain id from another tenant is not found.
	err := rm.Execute(e.c, e.actor, id, otherDomID)
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)

	require.NoError(t, rm.Execute(e.c, e.actor, id, firstID))
	first := byID(firstID)
	require.NotNil(t, first.DeletedAt)
	assert.Equal(t, e.actor, *first.DeletedBy)
	a := e.lastAudit(t)
	assert.Equal(t, "tenant.domain.remove", a.Action)
	assert.Equal(t, first.ID, a.EntityID)

	// last remaining domain
	err = rm.Execute(e.c, e.actor, id, secondID)
	require.ErrorIs(t, err, domainerrors.ErrLastDomain)
	assert.Nil(t, byID(secondID).DeletedAt)
	err = rm.Execute(e.c, e.actor, other, otherDomID)
	require.ErrorIs(t, err, domainerrors.ErrLastDomain)

	// removed domain frees the name for re-registration
	_, err = NewAddTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx).Execute(e.c, e.actor, other, tenanttypes.AddDomainRequest{Domain: "a.com"})
	require.NoError(t, err)

	require.ErrorIs(t, rm.Execute(e.c, e.actor, uuid.New(), firstID), domainerrors.ErrTenantNotFound)
	require.ErrorIs(t, rm.Execute(e.c, e.actor, id, firstID), domainerrors.ErrDomainNotFound)
}

func TestGet(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	uc := NewGetTenantUseCase(e.tr, e.dr)
	d, err := uc.Execute(e.c, id)
	require.NoError(t, err)
	assert.Len(t, d.Domains, 1)
	_, err = uc.Execute(e.c, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
}

func TestRename_MaxLengthAccepted(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	got, err := NewRenameTenantUseCase(e.tr, e.ar, e.tx).Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: strings.Repeat("é", tenanttypes.MaxNameLen)})
	require.NoError(t, err)
	assert.Len(t, []rune(got.Name), tenanttypes.MaxNameLen)
}

func TestRename_AuditFailureRollsBack(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	old := e.s.tenants[id].Name
	e.s.failAudit = true
	_, err := NewRenameTenantUseCase(e.tr, e.ar, e.tx).Execute(e.c, e.actor, id, tenanttypes.UpdateTenantRequest{Name: "New"})
	require.Error(t, err)
	assert.Equal(t, old, e.s.tenants[id].Name)
}

func TestDomains_Update(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com", "b.com")
	other := e.seedTenant(true, "c.com")
	aID, otherDomID := e.s.domains[0].ID, e.s.domains[2].ID
	upd := NewUpdateTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)
	req := func(d string) tenanttypes.UpdateDomainRequest { return tenanttypes.UpdateDomainRequest{Domain: d} }

	got, err := upd.Execute(e.c, e.actor, id, aID, req("  New.Example.ORG "))
	require.NoError(t, err)
	assert.Equal(t, "new.example.org", got.Domain)
	assert.Equal(t, "new.example.org", e.s.domains[0].Domain)
	assert.Equal(t, e.actor, *e.s.domains[0].UpdatedBy)
	a := e.lastAudit(t)
	assert.Equal(t, "tenant.domain.update", a.Action)
	assert.Equal(t, id, a.TenantID)
	assert.Equal(t, aID, a.EntityID)
	assert.Equal(t, e.actor, *a.ActorUserID)
	assert.Contains(t, a.Metadata, "a.com")
	assert.Contains(t, a.Metadata, "new.example.org")
	require.Len(t, e.s.audits, 1)

	// same value (after normalisation) is a no-op without an audit entry
	got, err = upd.Execute(e.c, e.actor, id, aID, req("NEW.example.org"))
	require.NoError(t, err)
	assert.Equal(t, "new.example.org", got.Domain)
	assert.Len(t, e.s.audits, 1)

	// uniqueness across tenants and within the tenant
	_, err = upd.Execute(e.c, e.actor, id, aID, req("c.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainAlreadyExists)
	_, err = upd.Execute(e.c, e.actor, id, aID, req("b.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainAlreadyExists)
	assert.Equal(t, "new.example.org", e.s.domains[0].Domain)

	// validation
	_, err = upd.Execute(e.c, e.actor, id, aID, req("nope"))
	require.ErrorIs(t, err, domainerrors.ErrInvalidDomain)
	_, err = upd.Execute(e.c, uuid.Nil, id, aID, req("x.com"))
	require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

	// another tenant's domain id -> not found; unknown ids -> not found
	_, err = upd.Execute(e.c, e.actor, id, otherDomID, req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)
	assert.Equal(t, "c.com", e.s.domains[2].Domain)
	_, err = upd.Execute(e.c, e.actor, id, uuid.New(), req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)
	_, err = upd.Execute(e.c, e.actor, id, uuid.Nil, req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)
	_, err = upd.Execute(e.c, e.actor, other, aID, req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)

	// soft-deleted domain -> not found
	now := time.Now()
	e.s.domains[1].DeletedAt = &now
	_, err = upd.Execute(e.c, e.actor, id, e.s.domains[1].ID, req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)

	_, err = upd.Execute(e.c, e.actor, uuid.New(), aID, req("zzz.com"))
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	assert.Len(t, e.s.audits, 1)
}

func TestDomains_Update_AuditFailureRollsBack(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	e.s.failAudit = true
	_, err := NewUpdateTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx).Execute(e.c, e.actor, id, e.s.domains[0].ID, tenanttypes.UpdateDomainRequest{Domain: "z.com"})
	require.Error(t, err)
	assert.Equal(t, "a.com", e.s.domains[0].Domain)
}

func TestDomains_AddRemove_AuditFailureRollsBack(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com", "b.com")
	e.s.failAudit = true
	_, err := NewAddTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx).Execute(e.c, e.actor, id, tenanttypes.AddDomainRequest{Domain: "c.com"})
	require.Error(t, err)
	assert.Len(t, e.s.domains, 2)
	require.Error(t, NewRemoveTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx).Execute(e.c, e.actor, id, e.s.domains[0].ID))
	assert.Nil(t, e.s.domains[0].DeletedAt)
}

func (e *env) seedUser(tid uuid.UUID, email string, active bool) *entity.User {
	u := &entity.User{ID: uuid.New(), TenantID: tid, Email: email, IsActive: active}
	e.s.users = append(e.s.users, u)
	return u
}

func TestDomains_Remove_BlockedWhileUsersOnDomain(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com", "b.com")
	other := e.seedTenant(true, "c.com")
	aID := e.s.domains[0].ID
	rm := NewRemoveTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)

	// A user of ANOTHER tenant on the same domain value does not block.
	e.seedUser(other, "x@a.com", true)
	// A lookalike domain does not block either.
	e.seedUser(id, "x@evila.com", true)
	e.seedUser(id, "x@a.com.evil.io", true)

	// Inactive/invited users of this tenant do block (case-insensitive).
	u := e.seedUser(id, "Bob@A.COM", false)
	require.ErrorIs(t, rm.Execute(e.c, e.actor, id, aID), domainerrors.ErrDomainInUse)
	assert.Nil(t, e.s.domains[0].DeletedAt)
	assert.Empty(t, e.s.audits)

	// Once the user is soft-deleted the removal goes through.
	now := time.Now()
	u.DeletedAt = &now
	require.NoError(t, rm.Execute(e.c, e.actor, id, aID))
	assert.NotNil(t, e.s.domains[0].DeletedAt)
}

func TestDomains_Remove_InUseCheckedBeforeLastDomain(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	e.seedUser(id, "x@a.com", true)
	rm := NewRemoveTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)
	require.ErrorIs(t, rm.Execute(e.c, e.actor, id, e.s.domains[0].ID), domainerrors.ErrDomainInUse)
	// not found still wins over in-use
	require.ErrorIs(t, rm.Execute(e.c, e.actor, id, uuid.New()), domainerrors.ErrDomainNotFound)
	e.s.users[0].DeletedAt = new(time.Time)
	require.ErrorIs(t, rm.Execute(e.c, e.actor, id, e.s.domains[0].ID), domainerrors.ErrLastDomain)
}

func TestDomains_Update_BlockedWhileUsersOnOldDomain(t *testing.T) {
	e := newEnv()
	id := e.seedTenant(true, "a.com")
	other := e.seedTenant(true, "c.com")
	aID := e.s.domains[0].ID
	upd := NewUpdateTenantDomainUseCase(e.tr, e.dr, e.ar, e.tx)
	req := func(d string) tenanttypes.UpdateDomainRequest { return tenanttypes.UpdateDomainRequest{Domain: d} }

	e.seedUser(other, "x@a.com", true) // other tenant: no effect
	got, err := upd.Execute(e.c, e.actor, id, aID, req("a.com"))
	require.NoError(t, err) // same value: no-op
	assert.Equal(t, "a.com", got.Domain)

	u := e.seedUser(id, "bob@a.com", false)
	// same value stays a no-op even with users on the domain (no check)
	_, err = upd.Execute(e.c, e.actor, id, aID, req("A.com"))
	require.NoError(t, err)
	// a real change is blocked
	_, err = upd.Execute(e.c, e.actor, id, aID, req("new.com"))
	require.ErrorIs(t, err, domainerrors.ErrDomainInUse)
	assert.Equal(t, "a.com", e.s.domains[0].Domain)
	assert.Empty(t, e.s.audits)

	now := time.Now()
	u.DeletedAt = &now
	got, err = upd.Execute(e.c, e.actor, id, aID, req("new.com"))
	require.NoError(t, err)
	assert.Equal(t, "new.com", got.Domain)
}
