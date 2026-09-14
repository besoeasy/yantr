import { ref } from 'vue'

// Module-level reactive state – works both inside and outside Vue components
export const notificationState = ref(null)

// Every notification auto-dismisses after this long (ms).
export const AUTO_DISMISS_MS = 10000

let autoCloseTimer = null

export function useNotification() {
  const show = (type, message) => {
    if (autoCloseTimer) {
      clearTimeout(autoCloseTimer)
      autoCloseTimer = null
    }
    notificationState.value = { type, message, id: Date.now() + Math.random() }

    // Auto-dismiss every notification after 10 s
    autoCloseTimer = setTimeout(() => {
      notificationState.value = null
      autoCloseTimer = null
    }, AUTO_DISMISS_MS)
  }

  const dismiss = () => {
    if (autoCloseTimer) {
      clearTimeout(autoCloseTimer)
      autoCloseTimer = null
    }
    notificationState.value = null
  }

  return {
    success: (msg) => show('success', msg),
    error:   (msg) => show('error',   msg),
    warning: (msg) => show('warning', msg),
    info:    (msg) => show('info',    msg),
    dismiss,
  }
}
