import { FIELD_LABELS } from '../schemas/profileSchemas'

/** Lists the mandatory items the backend still reports as missing. */
export function MissingFieldsAlert({ fields }: { fields?: string[] }) {
  if (!fields || fields.length === 0) return null
  return (
    <div role="status" className="rounded-sm border border-slate-300 bg-slate-100 p-3 text-sm text-slate-900">
      <p className="font-medium">Still needed to finish your profile:</p>
      <ul className="mt-1 list-disc pl-5 text-slate-600">
        {fields.map((f) => (
          <li key={f}>{FIELD_LABELS[f] ?? f}</li>
        ))}
      </ul>
    </div>
  )
}
