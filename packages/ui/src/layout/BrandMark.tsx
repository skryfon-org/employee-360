export function BrandMark({ name, subtitle, hideText = false, compactOnMobile = false }: { name: string; subtitle?: string; hideText?: boolean; compactOnMobile?: boolean }) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <span
        aria-hidden="true"
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-sm bg-accent text-sm font-semibold text-white"
      >
        E
      </span>
      {!hideText && (
        <span className={`min-w-0 flex-col leading-tight ${compactOnMobile ? 'hidden sm:flex' : 'flex'}`}>
          <span className="truncate text-sm font-semibold text-ink">{name}</span>
          {subtitle && <span className="truncate text-xs text-ink-muted">{subtitle}</span>}
        </span>
      )}
    </span>
  )
}
