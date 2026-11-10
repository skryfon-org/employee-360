import type { ReactNode } from 'react'
import * as RadixTabs from '@radix-ui/react-tabs'

export interface TabItem {
  value: string
  label: string
}

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

/**
 * Accessible tabs (Radix: WAI-ARIA roles, arrow/Home/End keys). Controlled, so a router can
 * drive it: pass the active value from the URL and navigate in `onValueChange`.
 * `children` is the content of the active panel.
 */
export function Tabs({
  items,
  value,
  onValueChange,
  label,
  children,
}: {
  items: TabItem[]
  value: string
  onValueChange: (value: string) => void
  /** Accessible name of the tab list. */
  label: string
  children?: ReactNode
}) {
  return (
    <RadixTabs.Root value={value} onValueChange={onValueChange} className="flex w-full min-w-0 flex-col gap-6">
      <RadixTabs.List aria-label={label} className="flex w-full min-w-0 overflow-x-auto border-b border-line">
        {items.map((item) => (
          <RadixTabs.Trigger
            key={item.value}
            value={item.value}
            className={`-mb-px inline-flex min-h-11 shrink-0 items-center whitespace-nowrap border-b-2 border-transparent px-4 text-sm font-medium text-ink-muted hover:text-ink data-[state=active]:border-slate-900 data-[state=active]:font-semibold data-[state=active]:text-ink ${FOCUS}`}
          >
            {item.label}
          </RadixTabs.Trigger>
        ))}
      </RadixTabs.List>
      <RadixTabs.Content value={value} className={`min-w-0 ${FOCUS}`}>
        {children}
      </RadixTabs.Content>
    </RadixTabs.Root>
  )
}
