import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { addTenantDomain, getTenant, removeTenantDomain, renameTenant, updateTenantDomain } from '@employee360/api-client'

export const TENANT_KEY = ['tenant'] as const

/** The caller's own tenant, including its domains. */
export function useTenantQuery(enabled = true) {
  return useQuery({ queryKey: TENANT_KEY, queryFn: ({ signal }) => getTenant(signal), enabled })
}

/** Every tenant mutation can change the tenant or its domains, so refresh the tenant query. */
function useTenantMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({ mutationFn: fn, onSuccess: () => qc.invalidateQueries({ queryKey: TENANT_KEY }) })
}

export const useRenameTenantMutation = () => useTenantMutation((name: string) => renameTenant(name))
export const useAddDomainMutation = () => useTenantMutation((domain: string) => addTenantDomain(domain))
export const useUpdateDomainMutation = () =>
  useTenantMutation((v: { id: string; domain: string }) => updateTenantDomain(v.id, v.domain))
export const useRemoveDomainMutation = () => useTenantMutation((domainId: string) => removeTenantDomain(domainId))
