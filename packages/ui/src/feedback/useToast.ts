import { toast as sonner } from 'sonner'

type Options = { duration?: number }

export const DEFAULT_TOAST_DURATION = 5000
export const ERROR_TOAST_DURATION = 8000

/** Thin, stable wrapper over sonner for raising toasts from any component. */
const api = {
  success: (message: string, o?: Options) =>
    sonner.success(message, { duration: o?.duration ?? DEFAULT_TOAST_DURATION }),
  error: (message: string, o?: Options) =>
    sonner.error(message, { duration: o?.duration ?? ERROR_TOAST_DURATION }),
  info: (message: string, o?: Options) =>
    sonner.info(message, { duration: o?.duration ?? DEFAULT_TOAST_DURATION }),
  warning: (message: string, o?: Options) =>
    sonner.warning(message, { duration: o?.duration ?? DEFAULT_TOAST_DURATION }),
  dismiss: (id?: string | number) => sonner.dismiss(id),
}

export function useToast() {
  return api
}
