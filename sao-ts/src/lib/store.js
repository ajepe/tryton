import { writable } from 'svelte/store'

export const rpc = writable(null)
export const session = writable(null)
export const currentAction = writable(null)
export const notifications = writable([])
export const notificationCount = writable(0)
export const toast = writable(null)
export const menuOpen = writable(false)

export function showToast(message, type = 'error') {
  toast.set({ message, type })
  setTimeout(() => toast.set(null), 5000)
}

export function resolveTitle(action) {
  if (!action) return 'Talon'
  if (action.name) return action.name
  if (action.res_model) return action.res_model
  return 'Talon'
}
