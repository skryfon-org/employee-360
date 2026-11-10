package employee

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

// world is an in-memory multi-tenant store behind all the fake repositories.
// Every fake method enforces tenant scoping so a wrong tenantID yields "not found".
type world struct {
	tenants     map[uuid.UUID]*entity.Tenant
	users       map[uuid.UUID]*entity.User
	roles       map[uuid.UUID]*entity.Role // by role id
	userRoles   []*entity.UserRole
	profiles    map[uuid.UUID]*entity.EmployeeProfile // by user id
	contacts    []*entity.EmergencyContact
	assignments []*entity.UserAssignment
	depts       map[uuid.UUID]*entity.Department
	positions   map[uuid.UUID]*entity.Position
	revoked     map[uuid.UUID]bool
	settings    map[uuid.UUID]*entity.OrganizationSettings // by tenant id
	audit       *ucsharedtest.RecordingAuditRecorder
}

func newWorld() *world {
	return &world{
		tenants: map[uuid.UUID]*entity.Tenant{}, users: map[uuid.UUID]*entity.User{},
		roles: map[uuid.UUID]*entity.Role{}, profiles: map[uuid.UUID]*entity.EmployeeProfile{},
		depts: map[uuid.UUID]*entity.Department{}, positions: map[uuid.UUID]*entity.Position{},
		revoked: map[uuid.UUID]bool{}, settings: map[uuid.UUID]*entity.OrganizationSettings{}, audit: &ucsharedtest.RecordingAuditRecorder{},
	}
}

// Constructor shortcuts: each wires the real usecase constructor to this world.

func (w *world) newListEmployees() empuc.ListEmployeesUseCase {
	return NewListEmployeesUseCase((*fDirectory)(w))
}

func (w *world) newGetEmployee() empuc.GetEmployeeUseCase {
	return NewGetEmployeeUseCase((*fDirectory)(w), (*fProfiles)(w), (*fContacts)(w))
}

func (w *world) newUpdateEmployee() empuc.UpdateEmployeeUseCase {
	return NewUpdateEmployeeUseCase((*fUsers)(w), (*fProfiles)(w), (*fContacts)(w), (*fAssignments)(w), (*fDirectory)(w), (*fOrg)(w), (*fSettings)(w), w.audit, nil)
}

func (w *world) newListAssignments() empuc.ListAssignmentsUseCase {
	return NewListAssignmentsUseCase((*fUsers)(w), (*fAssignments)(w))
}

func (w *world) newListEmergencyContacts() empuc.ListEmergencyContactsUseCase {
	return NewListEmergencyContactsUseCase((*fUsers)(w), (*fContacts)(w))
}

func (w *world) newCreateEmergencyContact() empuc.CreateEmergencyContactUseCase {
	return NewCreateEmergencyContactUseCase((*fUsers)(w), (*fProfiles)(w), (*fSettings)(w), (*fContacts)(w), w.audit, nil)
}

func (w *world) newUpdateEmergencyContact() empuc.UpdateEmergencyContactUseCase {
	return NewUpdateEmergencyContactUseCase((*fUsers)(w), (*fContacts)(w), w.audit, nil)
}

func (w *world) newDeleteEmergencyContact() empuc.DeleteEmergencyContactUseCase {
	return NewDeleteEmergencyContactUseCase((*fUsers)(w), (*fProfiles)(w), (*fContacts)(w), w.audit, nil)
}

func (w *world) newDeactivateEmployee() empuc.DeactivateEmployeeUseCase {
	return NewDeactivateEmployeeUseCase((*fUsers)(w), (*fProfiles)(w), (*fContacts)(w), (*fDirectory)(w), (*fRefresh)(w), w.audit, nil)
}

func (w *world) newActivateEmployee() empuc.ActivateEmployeeUseCase {
	return NewActivateEmployeeUseCase((*fUsers)(w), (*fProfiles)(w), (*fContacts)(w), (*fDirectory)(w), w.audit, nil)
}

func (w *world) newChangeEmployeeRole() empuc.ChangeEmployeeRoleUseCase {
	return NewChangeEmployeeRoleUseCase((*fUsers)(w), (*fUserRoles)(w), (*fRoles)(w), (*fProfiles)(w), (*fContacts)(w), (*fDirectory)(w), w.audit, nil)
}

func (w *world) newGetMe() empuc.GetMeUseCase {
	return NewGetMeUseCase((*fUsers)(w), (*fProfiles)(w), (*fTenants)(w))
}

func (w *world) newGetMyProfile() empuc.GetMyProfileUseCase {
	return NewGetMyProfileUseCase((*fUsers)(w), (*fOrgNames)(w), (*fProfiles)(w), (*fContacts)(w))
}

func (w *world) newUpdateMyProfile() empuc.UpdateMyProfileUseCase {
	return NewUpdateMyProfileUseCase((*fUsers)(w), (*fOrgNames)(w), (*fProfiles)(w), (*fSettings)(w), (*fContacts)(w), w.audit, nil)
}

func (w *world) newProfileGate() empuc.ProfileGateUseCase {
	return NewProfileGateUseCase((*fProfiles)(w))
}

func (w *world) tenant(name string) uuid.UUID {
	t := &entity.Tenant{ID: uuid.New(), Name: name, IsActive: true}
	w.tenants[t.ID] = t
	for _, r := range []string{entity.RoleSuperAdmin, entity.RoleAdmin, entity.RoleEmployee} {
		role := &entity.Role{ID: uuid.New(), TenantID: t.ID, Name: r}
		w.roles[role.ID] = role
	}
	return t.ID
}

func (w *world) role(tenant uuid.UUID, name string) *entity.Role {
	for _, r := range w.roles {
		if r.TenantID == tenant && r.Name == name {
			return r
		}
	}
	panic("role missing")
}

func (w *world) user(tenant uuid.UUID, email string, active bool, roles ...string) *entity.User {
	hash := "hash"
	u := &entity.User{ID: uuid.New(), TenantID: tenant, Email: email, FirstName: "F", LastName: "L", IsActive: active, PasswordHash: &hash}
	w.users[u.ID] = u
	for _, r := range roles {
		w.userRoles = append(w.userRoles, &entity.UserRole{ID: uuid.New(), TenantID: tenant, UserID: u.ID, RoleID: w.role(tenant, r).ID})
	}
	return u
}

func (w *world) actor(u *entity.User, roles ...string) emptypes.Actor {
	return emptypes.Actor{TenantID: u.TenantID, UserID: u.ID, Roles: roles}
}

func (w *world) dept(tenant uuid.UUID, active bool) uuid.UUID {
	d := &entity.Department{ID: uuid.New(), TenantID: tenant, Name: uuid.NewString(), IsActive: active}
	w.depts[d.ID] = d
	return d.ID
}

func (w *world) position(tenant uuid.UUID, active bool) uuid.UUID {
	p := &entity.Position{ID: uuid.New(), TenantID: tenant, Name: uuid.NewString(), IsActive: active}
	w.positions[p.ID] = p
	return p.ID
}

func (w *world) auditActions() []string {
	var a []string
	for _, l := range w.audit.Logs {
		a = append(a, l.Action)
	}
	return a
}

func (w *world) roleNamesOf(tenant, user uuid.UUID) []string {
	var out []string
	for _, ur := range w.userRoles {
		if ur.TenantID == tenant && ur.UserID == user {
			out = append(out, w.roles[ur.RoleID].Name)
		}
	}
	sort.Strings(out)
	return out
}

// ---- users ----

type fUsers world

func (f *fUsers) Create(_ context.Context, u *entity.User) error { f.users[u.ID] = u; return nil }
func (f *fUsers) GetByID(_ context.Context, t, id uuid.UUID) (*entity.User, error) {
	u, ok := f.users[id]
	if !ok || u.TenantID != t || u.DeletedAt != nil {
		return nil, domainerrors.ErrUserNotFound
	}
	c := *u
	return &c, nil
}
func (f *fUsers) GetByTenantAndEmail(context.Context, uuid.UUID, string) (*entity.User, error) {
	return nil, domainerrors.ErrUserNotFound
}
func (f *fUsers) GetByIDWithRoles(ctx context.Context, t, id uuid.UUID) (*entity.User, error) {
	u, err := f.GetByID(ctx, t, id)
	if err != nil {
		return nil, err
	}
	for _, ur := range f.userRoles {
		if ur.TenantID == t && ur.UserID == id {
			u.Roles = append(u.Roles, *f.roles[ur.RoleID])
		}
	}
	return u, nil
}
func (f *fUsers) GetByTenantAndEmailWithRoles(context.Context, uuid.UUID, string) (*entity.User, error) {
	return nil, domainerrors.ErrUserNotFound
}
func (f *fUsers) Update(_ context.Context, t uuid.UUID, u *entity.User) error {
	cur, ok := f.users[u.ID]
	if !ok || cur.TenantID != t {
		return domainerrors.ErrUserNotFound
	}
	c := *u
	c.Roles = nil
	f.users[u.ID] = &c
	return nil
}
func (f *fUsers) Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error { return nil }
func (f *fUsers) List(context.Context, uuid.UUID, int, int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}

// ---- roles ----

type fUserRoles world

func (f *fUserRoles) AssignRole(_ context.Context, ur *entity.UserRole) error {
	f.userRoles = append(f.userRoles, ur)
	return nil
}
func (f *fUserRoles) RemoveRole(_ context.Context, t, u, r uuid.UUID) error {
	kept := f.userRoles[:0]
	for _, ur := range f.userRoles {
		if !(ur.TenantID == t && ur.UserID == u && ur.RoleID == r) {
			kept = append(kept, ur)
		}
	}
	f.userRoles = kept
	return nil
}
func (f *fUserRoles) GetRolesByUserID(context.Context, uuid.UUID, uuid.UUID) ([]*entity.Role, error) {
	return nil, nil
}
func (f *fUserRoles) GetUserRolesByUserID(context.Context, uuid.UUID, uuid.UUID) ([]*entity.UserRole, error) {
	return nil, nil
}
func (f *fUserRoles) DeleteByUserID(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type fRoles world

func (f *fRoles) Create(context.Context, *entity.Role) error { return nil }
func (f *fRoles) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Role, error) {
	return nil, domainerrors.ErrRoleNotFound
}
func (f *fRoles) GetByName(_ context.Context, t uuid.UUID, n string) (*entity.Role, error) {
	for _, r := range f.roles {
		if r.TenantID == t && r.Name == n {
			return r, nil
		}
	}
	return nil, domainerrors.ErrRoleNotFound
}
func (f *fRoles) List(context.Context, uuid.UUID) ([]*entity.Role, error) { return nil, nil }
func (f *fRoles) Update(context.Context, uuid.UUID, *entity.Role) error   { return nil }
func (f *fRoles) Delete(context.Context, uuid.UUID, uuid.UUID) error      { return nil }

// ---- profiles ----

type fProfiles world

func (f *fProfiles) GetByUserID(_ context.Context, t, u uuid.UUID) (*entity.EmployeeProfile, error) {
	p, ok := f.profiles[u]
	if !ok || p.TenantID != t {
		return nil, domainerrors.ErrEmployeeProfileNotFound
	}
	c := *p
	return &c, nil
}
func (f *fProfiles) Create(_ context.Context, _ uuid.UUID, p *entity.EmployeeProfile) error {
	c := *p
	f.profiles[p.UserID] = &c
	return nil
}
func (f *fProfiles) Update(_ context.Context, t, _ uuid.UUID, p *entity.EmployeeProfile) error {
	cur, ok := f.profiles[p.UserID]
	if !ok || cur.TenantID != t {
		return domainerrors.ErrEmployeeProfileNotFound
	}
	c := *p
	f.profiles[p.UserID] = &c
	return nil
}
func (f *fProfiles) EmployeeCodeExists(_ context.Context, t uuid.UUID, code string, exclude uuid.UUID) (bool, error) {
	for _, p := range f.profiles {
		if p.TenantID == t && p.UserID != exclude && p.EmployeeCode != nil && *p.EmployeeCode == code {
			return true, nil
		}
	}
	return false, nil
}

// ---- contacts ----

type fContacts world

func (f *fContacts) live(t, u uuid.UUID) []*entity.EmergencyContact {
	var out []*entity.EmergencyContact
	for _, c := range f.contacts {
		if c.TenantID == t && c.UserID == u && c.DeletedAt == nil {
			out = append(out, c)
		}
	}
	return out
}
func (f *fContacts) ListByUser(_ context.Context, t, u uuid.UUID) ([]*entity.EmergencyContact, error) {
	var out []*entity.EmergencyContact
	for _, c := range f.live(t, u) {
		cc := *c
		out = append(out, &cc)
	}
	return out, nil
}
func (f *fContacts) GetByID(_ context.Context, t, u, id uuid.UUID) (*entity.EmergencyContact, error) {
	for _, c := range f.live(t, u) {
		if c.ID == id {
			cc := *c
			return &cc, nil
		}
	}
	return nil, domainerrors.ErrEmergencyContactNotFound
}
func (f *fContacts) CountByUser(_ context.Context, t, u uuid.UUID) (int64, error) {
	return int64(len(f.live(t, u))), nil
}
func (f *fContacts) Create(_ context.Context, _ uuid.UUID, c *entity.EmergencyContact) error {
	cc := *c
	f.contacts = append(f.contacts, &cc)
	return nil
}
func (f *fContacts) Update(_ context.Context, t, _ uuid.UUID, c *entity.EmergencyContact) error {
	for i, cur := range f.contacts {
		if cur.ID == c.ID && cur.TenantID == t && cur.DeletedAt == nil {
			cc := *c
			f.contacts[i] = &cc
			return nil
		}
	}
	return domainerrors.ErrEmergencyContactNotFound
}
func (f *fContacts) Delete(_ context.Context, t, u, id, _ uuid.UUID) error {
	for _, c := range f.live(t, u) {
		if c.ID == id {
			now := time.Now()
			c.DeletedAt = &now
			c.IsPrimary = false
			return nil
		}
	}
	return domainerrors.ErrEmergencyContactNotFound
}
func (f *fContacts) ClearPrimary(_ context.Context, t, u, except, _ uuid.UUID) error {
	for _, c := range f.live(t, u) {
		if c.ID != except {
			c.IsPrimary = false
		}
	}
	return nil
}

// ---- assignments ----

type fAssignments world

func (f *fAssignments) GetOpen(_ context.Context, t, u uuid.UUID) (*entity.UserAssignment, error) {
	for _, a := range f.assignments {
		if a.TenantID == t && a.UserID == u && a.EffectiveTo == nil {
			c := *a
			return &c, nil
		}
	}
	return nil, nil
}
func (f *fAssignments) CloseOpen(_ context.Context, t, u, _ uuid.UUID, to time.Time) error {
	for _, a := range f.assignments {
		if a.TenantID == t && a.UserID == u && a.EffectiveTo == nil {
			x := to
			a.EffectiveTo = &x
		}
	}
	return nil
}
func (f *fAssignments) Create(_ context.Context, _ uuid.UUID, a *entity.UserAssignment) error {
	c := *a
	f.assignments = append(f.assignments, &c)
	return nil
}
func (f *fAssignments) ListByUser(_ context.Context, t, u uuid.UUID) ([]*entity.UserAssignment, error) {
	var out []*entity.UserAssignment
	for i := len(f.assignments) - 1; i >= 0; i-- {
		if a := f.assignments[i]; a.TenantID == t && a.UserID == u {
			out = append(out, a)
		}
	}
	return out, nil
}

func (w *world) openCount(t, u uuid.UUID) int {
	n := 0
	for _, a := range w.assignments {
		if a.TenantID == t && a.UserID == u && a.EffectiveTo == nil {
			n++
		}
	}
	return n
}

// ---- directory ----

type fDirectory world

func (f *fDirectory) summary(u *entity.User) *entity.EmployeeSummary {
	s := &entity.EmployeeSummary{UserID: u.ID, Email: u.Email, FirstName: u.FirstName, LastName: u.LastName,
		IsActive: u.IsActive, DepartmentID: u.DepartmentID, PositionID: u.PositionID, Roles: []string{}}
	s.Roles = append(s.Roles, (*world)(f).roleNamesOf(u.TenantID, u.ID)...)
	if p, ok := f.profiles[u.ID]; ok && p.TenantID == u.TenantID {
		s.EmployeeCode, s.EmploymentType = p.EmployeeCode, p.EmploymentType
		st := p.EmploymentStatus
		s.EmploymentStatus = &st
		s.ProfileCompleted = p.ProfileCompletedAt != nil
	}
	return s
}
func (f *fDirectory) List(_ context.Context, t uuid.UUID, flt entity.EmployeeFilter, limit, offset int) ([]*entity.EmployeeSummary, int64, error) {
	var all []*entity.EmployeeSummary
	for _, u := range f.users {
		if u.TenantID == t && u.DeletedAt == nil && (flt.IsActive == nil || *flt.IsActive == u.IsActive) {
			all = append(all, f.summary(u))
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Email < all[j].Email })
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}
func (f *fDirectory) GetSummary(_ context.Context, t, id uuid.UUID) (*entity.EmployeeSummary, error) {
	u, ok := f.users[id]
	if !ok || u.TenantID != t || u.DeletedAt != nil {
		return nil, domainerrors.ErrUserNotFound
	}
	return f.summary(u), nil
}
func (f *fDirectory) LockActiveUserIDsWithRole(_ context.Context, t uuid.UUID, role string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, u := range f.users {
		if u.TenantID != t || !u.IsActive || u.DeletedAt != nil {
			continue
		}
		for _, r := range (*world)(f).roleNamesOf(t, u.ID) {
			if r == role {
				ids = append(ids, u.ID)
			}
		}
	}
	return ids, nil
}

// ---- org refs, tokens, tenants ----

type fOrg world

func (f *fOrg) DepartmentExists(_ context.Context, t, id uuid.UUID) (bool, error) {
	d, ok := f.depts[id]
	return ok && d.TenantID == t, nil
}
func (f *fOrg) LockDepartmentShared(_ context.Context, t, id uuid.UUID) (bool, bool, error) {
	d, ok := f.depts[id]
	if !ok || d.TenantID != t {
		return false, false, nil
	}
	return true, d.IsActive, nil
}
func (f *fOrg) LockPositionShared(_ context.Context, t, id uuid.UUID) (bool, bool, error) {
	p, ok := f.positions[id]
	if !ok || p.TenantID != t {
		return false, false, nil
	}
	return true, p.IsActive, nil
}

type fOrgNames world

func (f *fOrgNames) DepartmentName(_ context.Context, t, id uuid.UUID) (string, bool, error) {
	d, ok := f.depts[id]
	if !ok || d.TenantID != t {
		return "", false, nil
	}
	return d.Name, true, nil
}
func (f *fOrgNames) PositionName(_ context.Context, t, id uuid.UUID) (string, bool, error) {
	p, ok := f.positions[id]
	if !ok || p.TenantID != t {
		return "", false, nil
	}
	return p.Name, true, nil
}

type fRefresh world

func (f *fRefresh) Create(context.Context, *entity.RefreshToken) error { return nil }
func (f *fRefresh) GetByTokenHash(context.Context, string) (*entity.RefreshToken, error) {
	return nil, domainerrors.ErrNotFound
}
func (f *fRefresh) Revoke(context.Context, uuid.UUID) error       { return nil }
func (f *fRefresh) RevokeFamily(context.Context, uuid.UUID) error { return nil }
func (f *fRefresh) RevokeAllForUser(_ context.Context, u uuid.UUID) error {
	f.revoked[u] = true
	return nil
}
func (f *fRefresh) DeleteExpiredTokens(context.Context, time.Time) error { return nil }

type fTenants world

func (f *fTenants) GetByID(_ context.Context, id uuid.UUID) (*entity.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return nil, domainerrors.ErrTenantNotFound
	}
	return t, nil
}

// ---- shared test helpers ----

var bg = context.Background()

func parse(t *testing.T, s string) emptypes.UpdateEmployeeInput {
	t.Helper()
	var in emptypes.UpdateEmployeeInput
	require.NoError(t, json.Unmarshal([]byte(s), &in))
	return in
}

type fixture struct {
	w        *world
	tenant   uuid.UUID
	admin    *entity.User
	employee *entity.User
	super    *entity.User
}

func newFixture() *fixture {
	w := newWorld()
	t := w.tenant("Acme")
	return &fixture{
		w: w, tenant: t,
		admin:    w.user(t, "admin@acme.test", true, entity.RoleAdmin),
		employee: w.user(t, "emp@acme.test", true, entity.RoleEmployee),
		super:    w.user(t, "root@acme.test", true, entity.RoleSuperAdmin),
	}
}

func (f *fixture) adminActor() emptypes.Actor { return f.w.actor(f.admin, entity.RoleAdmin) }
func (f *fixture) superActor() emptypes.Actor { return f.w.actor(f.super, entity.RoleSuperAdmin) }

func contact(name string, primary bool) emptypes.EmergencyContactInput {
	return emptypes.EmergencyContactInput{Name: name, Relationship: "spouse", Phone: "+91 98765 43210", IsPrimary: primary}
}

func fillAll(t *testing.T, uc interface {
	Execute(context.Context, uuid.UUID, uuid.UUID, emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error)
}, tenant, user uuid.UUID) *emptypes.MyProfile {
	t.Helper()
	var in emptypes.UpdateMyProfileInput
	require.NoError(t, json.Unmarshal([]byte(`{"phone":"+91 98765 43210","date_of_birth":"1990-04-02","gender":"female","address_line1":"1 MG Road","city":"Kochi","country":"India"}`), &in))
	m, err := uc.Execute(bg, tenant, user, in)
	require.NoError(t, err)
	return m
}

// fSettings fakes the read side of OrganizationSettingsRepository used by the update usecase.
type fSettings world

func (f *fSettings) GetByTenantID(_ context.Context, t uuid.UUID) (*entity.OrganizationSettings, error) {
	if s, ok := f.settings[t]; ok {
		c := *s
		return &c, nil
	}
	return nil, domainerrors.ErrOrganizationSettingsNotFound
}

func (f *fSettings) EnsureDefaults(_ context.Context, t, _ uuid.UUID) error {
	if _, ok := f.settings[t]; !ok {
		f.settings[t] = entity.DefaultOrganizationSettings(t)
	}
	return nil
}

func (f *fSettings) LockByTenantID(c context.Context, t uuid.UUID) (*entity.OrganizationSettings, error) {
	return f.GetByTenantID(c, t)
}

func (f *fSettings) Update(_ context.Context, t, _ uuid.UUID, s *entity.OrganizationSettings) error {
	c := *s
	f.settings[t] = &c
	return nil
}

func (f *fSettings) NextEmployeeCodeNumber(_ context.Context, t uuid.UUID) (entity.EmployeeCodeCounter, error) {
	s, ok := f.settings[t]
	if !ok {
		s = entity.DefaultOrganizationSettings(t)
		f.settings[t] = s
	}
	taken := s.EmployeeCodeNextNumber
	s.EmployeeCodeLastIssuedNumber = taken
	s.EmployeeCodeNextNumber++
	return entity.EmployeeCodeCounter{Prefix: s.EmployeeCodePrefix, Padding: s.EmployeeCodePadding, Number: taken}, nil
}
