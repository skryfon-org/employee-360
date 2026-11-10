/** Per-country postal code rules. Keys are lowercase country names as in the bundled dataset (plus common aliases). */
interface PostalRule {
  re: RegExp
  message: string
}

const IN: PostalRule = { re: /^[1-9][0-9]{5}$/, message: 'Enter a valid 6-digit PIN code' }
const US: PostalRule = { re: /^\d{5}(-\d{4})?$/, message: 'Enter a valid ZIP code (12345 or 12345-6789)' }
const UK: PostalRule = { re: /^[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}$/i, message: 'Enter a valid UK postcode (e.g. SW1A 1AA)' }
const CA: PostalRule = {
  re: /^[ABCEGHJ-NPRSTVXY]\d[ABCEGHJ-NPRSTV-Z][ -]?\d[ABCEGHJ-NPRSTV-Z]\d$/i,
  message: 'Enter a valid Canadian postal code (e.g. K1A 0B1)',
}
const digits = (n: number, label: string): PostalRule => ({ re: new RegExp(`^\\d{${n}}$`), message: `Enter a valid ${n}-digit ${label}` })

const RULES: Record<string, PostalRule> = {
  india: IN,
  'united states': US,
  'united states of america': US,
  usa: US,
  'united kingdom': UK,
  uk: UK,
  canada: CA,
  australia: digits(4, 'postcode'),
  germany: digits(5, 'postal code'),
  france: digits(5, 'postal code'),
  italy: digits(5, 'postal code'),
  spain: digits(5, 'postal code'),
  singapore: digits(6, 'postal code'),
  china: digits(6, 'postal code'),
  'sri lanka': digits(5, 'postal code'),
  nepal: digits(5, 'postal code'),
  pakistan: digits(5, 'postal code'),
  bangladesh: digits(4, 'postal code'),
  malaysia: digits(5, 'postal code'),
  japan: { re: /^\d{3}-?\d{4}$/, message: 'Enter a valid postal code (7 digits, e.g. 100-0001)' },
  brazil: { re: /^\d{5}-?\d{3}$/, message: 'Enter a valid CEP (e.g. 01310-100)' },
  netherlands: { re: /^\d{4} ?[A-Z]{2}$/i, message: 'Enter a valid postal code (e.g. 1012 AB)' },
}

const GENERIC: PostalRule = {
  re: /^[A-Za-z0-9 -]{3,10}$/,
  message: 'Enter a valid postal code (3-10 letters, digits, spaces or hyphens)',
}

/** Returns an error message for an invalid postal code, or null when valid or empty (the field is optional). */
export function validatePostalCode(country: string, pincode: string): string | null {
  const code = pincode.trim()
  if (code === '') return null
  const rule = RULES[country.trim().toLocaleLowerCase()] ?? GENERIC
  return rule.re.test(code) ? null : rule.message
}

/** Zod `superRefine` callback for any object with `country` and `pincode` string fields. */
export function postalCodeRefinement(
  v: { country: string; pincode: string },
  ctx: { addIssue: (issue: { code: 'custom'; path: string[]; message: string }) => void },
): void {
  const message = validatePostalCode(v.country, v.pincode)
  if (message) ctx.addIssue({ code: 'custom', path: ['pincode'], message })
}
