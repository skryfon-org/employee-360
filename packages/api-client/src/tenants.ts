import { unwrapListResponse, unwrapSingleEntity } from './unwrap.ts';
import {
  deleteApiV1TenantDomainsDomainId,
  getApiV1Tenant,
  getApiV1TenantDomains,
  patchApiV1Tenant,
  patchApiV1TenantDomainsDomainId,
  postApiV1TenantDomains,
} from './generated/hooks/tenant/tenant.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantResponse as Tenant } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantDetailResponse as TenantDetail } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantDetailResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantDomainResponse as TenantDomain } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesTenantTenantDomainResponse.ts';

export type { Tenant, TenantDetail, TenantDomain };

/** `GET /api/v1/tenant` — the caller's own tenant incl. domains (super_admin only). */
export async function getTenant(signal?: AbortSignal): Promise<TenantDetail> {
  return unwrapSingleEntity(await getApiV1Tenant(signal));
}

export async function renameTenant(name: string): Promise<TenantDetail> {
  return unwrapSingleEntity(await patchApiV1Tenant({ name }));
}

export async function listTenantDomains(signal?: AbortSignal): Promise<TenantDomain[]> {
  return unwrapListResponse(await getApiV1TenantDomains(signal)).data;
}

export async function addTenantDomain(domain: string): Promise<TenantDomain> {
  return unwrapSingleEntity(await postApiV1TenantDomains({ domain }));
}

export async function updateTenantDomain(domainId: string, domain: string): Promise<TenantDomain> {
  return unwrapSingleEntity(await patchApiV1TenantDomainsDomainId(domainId, { domain }));
}

export async function removeTenantDomain(domainId: string): Promise<void> {
  await deleteApiV1TenantDomainsDomainId(domainId);
}
