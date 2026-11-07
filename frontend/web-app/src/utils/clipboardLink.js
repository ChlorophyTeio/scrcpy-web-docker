import { clipboardTextError } from './clipboard.js'

export function createClipboardLinkState() {
  return { local: null, remote: null, pending: null, message: '', blocked: '' }
}

// Keep text only in the current device window. A single pending value represents
// the newest phone copy; denied/background writes must never silently drop it.
export function createClipboardLink({ state, active, readLocal, writeLocal, sendRemote }) {
  let reading = false, writing = false, disposed = false, revision = 0, localRevision = 0
  const allowed = () => !disposed && active()

  function noteLocal(text) {
    revision++
    localRevision++
    state.local = state.remote = text
    state.pending = null
    state.message = ''
    state.blocked = ''
  }

  function sendLocal(text, explicit = false) {
    if (!allowed() || clipboardTextError(text)) return false
    if (state.pending !== null && !explicit) return false
    if (text === state.local && !explicit) return true
    if (!sendRemote(text)) {
      state.message = '等待手机控制连接，文字暂未同步'
      return false
    }
    noteLocal(text)
    return true
  }

  async function flushRemote() {
    if (!allowed() || writing || state.pending === null) return false
    writing = true
    const text = state.pending, version = localRevision
    let success = false
    try {
      await writeLocal(text)
      if (disposed || localRevision !== version) return false
      state.local = state.remote = text
      if (state.pending === text) state.pending = null
      state.blocked = ''
      state.message = ''
      success = true
      return true
    } catch {
      if (!disposed && state.pending !== null) {
        state.blocked = 'write'
        state.message = '手机有新复制的文字，点击“复制到本机”完成同步'
      }
      return false
    } finally {
      writing = false
      if (success && state.pending !== null && allowed()) queueMicrotask(flushRemote)
    }
  }

  function receiveRemote(text) {
    if (disposed || typeof text !== 'string' || clipboardTextError(text)) return
    revision++ // Invalidate any older asynchronous read from the local clipboard.
    if (text === state.local && !writing) { state.remote = text; state.pending = null; return }
    state.pending = text
    state.message = '手机文字待同步到本机'
    if (allowed()) void flushRemote()
  }

  async function captureLocal({ explicit = false } = {}) {
    if (!allowed() || reading) return false
    if (state.pending !== null) return state.blocked === 'write' && !explicit ? false : flushRemote()
    reading = true
    const version = revision
    try {
      const text = await readLocal()
      if (!allowed() || version !== revision || state.pending !== null) return false
      if (state.blocked === 'read') { state.blocked = ''; state.message = '' }
      if (state.local === null && !explicit) {
        // Establish a baseline on first connection, without replacing an
        // existing phone clipboard merely because the web page was opened.
        state.local = text
        return true
      }
      return sendLocal(text, explicit)
    } catch {
      if (!disposed && state.pending === null) {
        state.blocked = 'read'
        state.message = '需允许剪贴板权限；也可直接 Ctrl+V 或长按粘贴到手机'
      }
      return false
    } finally { reading = false }
  }

  function dismissPending() {
    revision++
    localRevision++
    state.pending = null
    state.message = ''
    state.blocked = ''
  }

  function dispose() { disposed = true; revision++ }
  return { receiveRemote, captureLocal, flushRemote, sendLocal, noteLocal, dismissPending, dispose }
}

export function copiedEventText(event, selectionText = '') {
  if (event.defaultPrevented) return ''
  const target = event.target
  if (target?.type === 'password') return ''
  const supplied = event.clipboardData?.getData('text/plain')
  if (supplied) return supplied
  if (typeof target?.value === 'string' && Number.isInteger(target.selectionStart) && target.selectionStart !== target.selectionEnd) {
    return target.value.slice(target.selectionStart, target.selectionEnd)
  }
  return selectionText
}
