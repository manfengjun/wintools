/**
 * Pure progress-state reducer for the Python installation flow.
 *
 * Architecture:
 *   initialProgress(packageNames) → state
 *   applyProgress(state, event)   → state
 *
 * State is a plain object with:
 *   step, message, percent, installing, visible, error,
 *   packages: { [name]: 'pending'|'installing'|'success'|'failed' },
 *   logs: [{ message, type, time }]
 */

export function initialProgress(packageNames = []) {
  const packages = {}
  for (const name of packageNames) {
    packages[name] = 'pending'
  }
  return {
    step: '',
    message: '',
    percent: 0,
    installing: false,
    visible: false,
    error: '',
    packages,
    logs: []
  }
}

export function applyProgress(state, event) {
  if (!event) return state

  const newState = {
    ...state,
    step: event.step || state.step,
    message: event.message || state.message,
    percent: event.percent ?? state.percent,
    packages: { ...state.packages },
    logs: [...state.logs]
  }

  // Determine state transitions from event
  if (event.error) {
    newState.error = event.error
    newState.installing = false
    newState.visible = true
    newState.logs.push({ message: '❌ ' + event.error, type: 'error', time: new Date().toLocaleTimeString() })
    return newState
  }

  if (event.done) {
    newState.installing = false
    newState.visible = true
    newState.logs.push({ message: event.message || '完成', type: 'success', time: new Date().toLocaleTimeString() })
    return newState
  }

  if (event.step === 'elevate' || event.step === 'prepare' ||
      event.step === 'download' || event.step === 'install-python' ||
      event.step === 'verify-python' || event.step === 'install-package') {
    newState.installing = true
    newState.visible = true
  }

  if (event.message) {
    newState.logs.push({ message: event.message, type: 'info', time: new Date().toLocaleTimeString() })
  }

  // Update package status if present in event
  if (event.package && event.package_status) {
    newState.packages[event.package] = event.package_status
  }

  return newState
}
