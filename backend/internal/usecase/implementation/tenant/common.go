// Package tenant implements the super_admin current-tenant settings usecases
// (view/rename the caller's own tenant, manage its domains). The product is
// self-hosted, so there is no tenant creation or cross-tenant management.
package tenant

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
)

func requireActor(actorID uuid.UUID) error {
	if actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	return nil
}

func requireTenantID(id uuid.UUID) error {
	if id == uuid.Nil {
		return domainerrors.ErrTenantNotFound
	}
	return nil
}

// domainLabel / domainTLD describe a hostname-like value: two or more
// dot-separated LDH labels, ending in an alphabetic (or punycode) TLD.
var (
	domainLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	domainTLD   = regexp.MustCompile(`^([a-z]{2,63}|xn--[a-z0-9-]{1,59})$`)
)

// normalizeDomain lowercases/trims and validates a domain.
func normalizeDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	if d == "" || len(d) > 253 {
		return "", domainerrors.ErrInvalidDomain
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return "", domainerrors.ErrInvalidDomain
	}
	for _, l := range labels {
		if !domainLabel.MatchString(l) {
			return "", domainerrors.ErrInvalidDomain
		}
	}
	if !domainTLD.MatchString(labels[len(labels)-1]) {
		return "", domainerrors.ErrInvalidDomain
	}
	return d, nil
}

func normalizeName(raw string) (string, error) {
	n := strings.TrimSpace(raw)
	if n == "" || utf8.RuneCountInString(n) > tenanttypes.MaxNameLen {
		return "", domainerrors.ErrInvalidTenantName
	}
	return n, nil
}

// ensureDomainUnused refuses (ErrDomainInUse) while any non-soft-deleted user
// of the tenant has an email on domain: login resolves the tenant from the
// email's domain, so removing/changing it would lock those users out.
func ensureDomainUnused(c context.Context, repo repository.TenantDomainManager, tenantID uuid.UUID, domain string) error {
	n, err := repo.CountUsersOnDomain(c, tenantID, domain)
	if err != nil {
		return err
	}
	if n > 0 {
		return domainerrors.ErrDomainInUse
	}
	return nil
}
