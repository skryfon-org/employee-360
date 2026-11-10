export interface ProfileSection {
  /** URL segment under /profile. */
  path: string
  label: string
}

/** Sub-pages of My profile; the first entry is the default (`/profile` redirects to it). */
export const PROFILE_SECTIONS: ProfileSection[] = [
  { path: 'overview', label: 'Overview' },
  { path: 'employment', label: 'Employment' },
  { path: 'personal', label: 'Personal details' },
  { path: 'address', label: 'Address' },
  { path: 'emergency-contacts', label: 'Emergency contacts' },
]
