package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	tenantusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/tenant"
)

// ---- Get ----

// GetTenantUseCaseImpl implements tenantusecase.GetTenantUseCase.
type GetTenantUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	domainRepo repository.TenantDomainManager
}

var _ tenantusecase.GetTenantUseCase = (*GetTenantUseCaseImpl)(nil)

// NewGetTenantUseCase constructs a GetTenantUseCaseImpl.
func NewGetTenantUseCase(tenantRepo repository.TenantRepository, domainRepo repository.TenantDomainManager) *GetTenantUseCaseImpl {
	return &GetTenantUseCaseImpl{tenantRepo: tenantRepo, domainRepo: domainRepo}
}

// Execute returns the caller's tenant (404 if missing/soft-deleted) with its live domains.
func (u *GetTenantUseCaseImpl) Execute(c context.Context, tenantID uuid.UUID) (*tenanttypes.TenantDetail, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	t, err := u.tenantRepo.GetByID(c, tenantID)
	if err != nil {
		return nil, err
	}
	ds, err := u.domainRepo.ListByTenantID(c, tenantID)
	if err != nil {
		return nil, err
	}
	return &tenanttypes.TenantDetail{Tenant: t, Domains: ds}, nil
}

// ---- Rename ----

// RenameTenantUseCaseImpl implements tenantusecase.RenameTenantUseCase.
type RenameTenantUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

var _ tenantusecase.RenameTenantUseCase = (*RenameTenantUseCaseImpl)(nil)

// NewRenameTenantUseCase constructs a RenameTenantUseCaseImpl.
func NewRenameTenantUseCase(tenantRepo repository.TenantRepository, audit service.AuditRecorder, transactor ucshared.Transactor) *RenameTenantUseCaseImpl {
	return &RenameTenantUseCaseImpl{tenantRepo: tenantRepo, audit: audit, transactor: transactor}
}

// Execute renames the tenant.
func (u *RenameTenantUseCaseImpl) Execute(c context.Context, actorID, tenantID uuid.UUID, req tenanttypes.UpdateTenantRequest) (*entity.Tenant, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	name, err := normalizeName(req.Name)
	if err != nil {
		return nil, err
	}
	var out *entity.Tenant
	err = u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		t, err := u.tenantRepo.LockByID(txCtx, tenantID)
		if err != nil {
			return err
		}
		old := t.Name
		now := time.Now().UTC()
		if err := u.tenantRepo.UpdateName(txCtx, tenantID, name, actorID, now); err != nil {
			return err
		}
		t.Name, t.UpdatedAt, t.UpdatedBy = name, now, &actorID
		out = t
		return u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionTenantRename, domainaudit.EntityTenant, tenantID,
			map[string]any{"old_name": old, "new_name": name})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ---- Domains ----

// ListTenantDomainsUseCaseImpl implements tenantusecase.ListTenantDomainsUseCase.
type ListTenantDomainsUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	domainRepo repository.TenantDomainManager
}

var _ tenantusecase.ListTenantDomainsUseCase = (*ListTenantDomainsUseCaseImpl)(nil)

// NewListTenantDomainsUseCase constructs a ListTenantDomainsUseCaseImpl.
func NewListTenantDomainsUseCase(tenantRepo repository.TenantRepository, domainRepo repository.TenantDomainManager) *ListTenantDomainsUseCaseImpl {
	return &ListTenantDomainsUseCaseImpl{tenantRepo: tenantRepo, domainRepo: domainRepo}
}

// Execute lists the live domains of an existing tenant (404 otherwise).
func (u *ListTenantDomainsUseCaseImpl) Execute(c context.Context, tenantID uuid.UUID) ([]*entity.TenantDomain, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	if _, err := u.tenantRepo.GetByID(c, tenantID); err != nil {
		return nil, err
	}
	return u.domainRepo.ListByTenantID(c, tenantID)
}

// AddTenantDomainUseCaseImpl implements tenantusecase.AddTenantDomainUseCase.
type AddTenantDomainUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	domainRepo repository.TenantDomainManager
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

var _ tenantusecase.AddTenantDomainUseCase = (*AddTenantDomainUseCaseImpl)(nil)

// NewAddTenantDomainUseCase constructs an AddTenantDomainUseCaseImpl.
func NewAddTenantDomainUseCase(tenantRepo repository.TenantRepository, domainRepo repository.TenantDomainManager, audit service.AuditRecorder, transactor ucshared.Transactor) *AddTenantDomainUseCaseImpl {
	return &AddTenantDomainUseCaseImpl{tenantRepo: tenantRepo, domainRepo: domainRepo, audit: audit, transactor: transactor}
}

// Execute registers a normalised, globally unique domain for the tenant.
func (u *AddTenantDomainUseCaseImpl) Execute(c context.Context, actorID, tenantID uuid.UUID, req tenanttypes.AddDomainRequest) (*entity.TenantDomain, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	domain, err := normalizeDomain(req.Domain)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	d := &entity.TenantDomain{ID: uuid.New(), TenantID: tenantID, Domain: domain, CreatedAt: now, UpdatedAt: now, CreatedBy: &actorID, UpdatedBy: &actorID}
	err = u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		if _, err := u.tenantRepo.LockByID(txCtx, tenantID); err != nil {
			return err
		}
		if err := u.domainRepo.Create(txCtx, d); err != nil {
			return err
		}
		return u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionTenantDomainAdd, domainaudit.EntityTenantDomain, d.ID,
			map[string]any{"domain": domain})
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateTenantDomainUseCaseImpl implements tenantusecase.UpdateTenantDomainUseCase.
type UpdateTenantDomainUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	domainRepo repository.TenantDomainManager
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

var _ tenantusecase.UpdateTenantDomainUseCase = (*UpdateTenantDomainUseCaseImpl)(nil)

// NewUpdateTenantDomainUseCase constructs an UpdateTenantDomainUseCaseImpl.
func NewUpdateTenantDomainUseCase(tenantRepo repository.TenantRepository, domainRepo repository.TenantDomainManager, audit service.AuditRecorder, transactor ucshared.Transactor) *UpdateTenantDomainUseCaseImpl {
	return &UpdateTenantDomainUseCaseImpl{tenantRepo: tenantRepo, domainRepo: domainRepo, audit: audit, transactor: transactor}
}

// Execute changes the value of one of the tenant's live domains (same
// normalisation and global uniqueness as add). A domain that does not belong to
// the tenant, or is soft-deleted, is ErrDomainNotFound. Setting the current
// value is a no-op (no write, no audit entry); an actual change is refused
// with ErrDomainInUse while users of the tenant still sign in on the old value.
func (u *UpdateTenantDomainUseCaseImpl) Execute(c context.Context, actorID, tenantID, domainID uuid.UUID, req tenanttypes.UpdateDomainRequest) (*entity.TenantDomain, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	if domainID == uuid.Nil {
		return nil, domainerrors.ErrDomainNotFound
	}
	domain, err := normalizeDomain(req.Domain)
	if err != nil {
		return nil, err
	}
	var out *entity.TenantDomain
	err = u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		if _, err := u.tenantRepo.LockByID(txCtx, tenantID); err != nil {
			return err
		}
		d, err := u.domainRepo.GetByID(txCtx, tenantID, domainID)
		if err != nil {
			return err
		}
		out = d
		old := d.Domain
		if old == domain {
			return nil
		}
		if err := ensureDomainUnused(txCtx, u.domainRepo, tenantID, old); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := u.domainRepo.UpdateDomain(txCtx, tenantID, domainID, domain, actorID, now); err != nil {
			return err
		}
		d.Domain, d.UpdatedAt, d.UpdatedBy = domain, now, &actorID
		return u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionTenantDomainUpdate, domainaudit.EntityTenantDomain, domainID,
			map[string]any{"old_domain": old, "new_domain": domain})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveTenantDomainUseCaseImpl implements tenantusecase.RemoveTenantDomainUseCase.
type RemoveTenantDomainUseCaseImpl struct {
	tenantRepo repository.TenantRepository
	domainRepo repository.TenantDomainManager
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

var _ tenantusecase.RemoveTenantDomainUseCase = (*RemoveTenantDomainUseCaseImpl)(nil)

// NewRemoveTenantDomainUseCase constructs a RemoveTenantDomainUseCaseImpl.
func NewRemoveTenantDomainUseCase(tenantRepo repository.TenantRepository, domainRepo repository.TenantDomainManager, audit service.AuditRecorder, transactor ucshared.Transactor) *RemoveTenantDomainUseCaseImpl {
	return &RemoveTenantDomainUseCaseImpl{tenantRepo: tenantRepo, domainRepo: domainRepo, audit: audit, transactor: transactor}
}

// Execute soft-deletes the domain unless users still sign in with it
// (ErrDomainInUse, checked first) or it is the tenant's last live one. The
// tenant row is locked first so two concurrent removals cannot both pass the check.
func (u *RemoveTenantDomainUseCaseImpl) Execute(c context.Context, actorID, tenantID, domainID uuid.UUID) error {
	if err := requireActor(actorID); err != nil {
		return err
	}
	if err := requireTenantID(tenantID); err != nil {
		return err
	}
	if domainID == uuid.Nil {
		return domainerrors.ErrDomainNotFound
	}
	return u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		if _, err := u.tenantRepo.LockByID(txCtx, tenantID); err != nil {
			return err
		}
		d, err := u.domainRepo.GetByID(txCtx, tenantID, domainID)
		if err != nil {
			return err
		}
		if err := ensureDomainUnused(txCtx, u.domainRepo, tenantID, d.Domain); err != nil {
			return err
		}
		n, err := u.domainRepo.CountByTenantID(txCtx, tenantID)
		if err != nil {
			return err
		}
		if n <= 1 {
			return domainerrors.ErrLastDomain
		}
		if err := u.domainRepo.SoftDelete(txCtx, tenantID, domainID, actorID, time.Now().UTC()); err != nil {
			return err
		}
		return u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionTenantDomainRemove, domainaudit.EntityTenantDomain, domainID,
			map[string]any{"domain": d.Domain})
	})
}
