/** Decorative "i in a circle" icon; inherits color via currentColor. Size it with className (e.g. "h-4 w-4"). */
export function InfoIcon({ className = 'h-4 w-4' }: { className?: string }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 20 20" className={className} fill="currentColor">
      <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Zm-.75 4.25a.75.75 0 1 1 1.5 0 .75.75 0 0 1-1.5 0ZM9 9h1.5v5H9V9Z" />
    </svg>
  )
}
