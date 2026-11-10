/**
 * Builds a PATCH body from the dirty fields of a form. Strings are trimmed; an empty value for
 * a `nullable` key is sent as an explicit `null` (the backend's "clear this field"), otherwise
 * as an empty string (which the backend also treats as clear for text and enum fields).
 */
export function buildPatch<T extends Record<string, string | boolean>>(
  values: T,
  dirty: Partial<Record<keyof T, unknown>>,
  nullable: readonly (keyof T)[] = [],
): Record<string, string | boolean | null> {
  const out: Record<string, string | boolean | null> = {}
  for (const key of Object.keys(dirty) as (keyof T)[]) {
    if (!dirty[key]) continue
    const raw = values[key]
    const v = typeof raw === 'string' ? raw.trim() : raw
    out[key as string] = v === '' && nullable.includes(key) ? null : v
  }
  return out
}
