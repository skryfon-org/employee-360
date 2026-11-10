/** True when keyboard focus is somewhere text is typed or a choice is made, so global shortcuts must stay out of the way. */
export function isTypingTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false
  return t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName)
}

/** True while any modal dialog is mounted. */
export const isModalOpen = () => document.querySelector('[role="dialog"]') !== null

/** Whether a keydown is a plain (unmodified) global shortcut candidate. */
export function isPlainShortcut(e: KeyboardEvent): boolean {
  return !(e.ctrlKey || e.metaKey || e.altKey || e.shiftKey || e.isComposing || e.defaultPrevented) && !isTypingTarget(e.target) && !isModalOpen()
}
