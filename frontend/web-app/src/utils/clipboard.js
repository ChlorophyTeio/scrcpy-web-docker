// scrcpy's control packet is limited to 256 KiB, including a 14-byte header.
export const CLIPBOARD_MAX_BYTES = (1 << 18) - 14

export function clipboardTextError(text) {
  if (typeof text !== 'string' || !text.length) return '没有可发送的文字，请先粘贴或输入。'
  if (new TextEncoder().encode(text).length > CLIPBOARD_MAX_BYTES) return '文字超过 256 KB，请分段发送，避免内容被截断。'
  return ''
}

export function shouldRouteRemotePaste(event, { focused, editing, modalOpen, camera } = {}) {
  if (!focused || editing || modalOpen || camera || event.defaultPrevented) return false
  const target = event.target
  if (target?.isContentEditable) return false
  // The hidden input belongs to remote typing; other form fields stay local.
  if (!target?.classList?.contains('hidden-keyboard-input') && target?.closest?.('input, textarea, select, [contenteditable], [role="textbox"]')) return false
  return Boolean(event.clipboardData?.getData('text/plain') || event.clipboardData?.getData('text'))
}

export async function readBrowserClipboard() {
  if (!navigator.clipboard?.readText) throw new Error('浏览器不支持读取剪贴板，请在文本框内粘贴或输入。')
  try { return await navigator.clipboard.readText() } catch {
    throw new Error('浏览器未允许读取剪贴板。请在文本框内使用 Ctrl+V，或长按粘贴。')
  }
}

export async function copyBrowserText(text, textarea) {
  try {
    if (navigator.clipboard?.writeText) { await navigator.clipboard.writeText(text); return }
  } catch { /* A visible, selected textarea provides a user-gesture fallback. */ }
  if (textarea) {
    textarea.focus(); textarea.select()
    try { if (document.execCommand('copy')) return } catch {}
  }
  throw new Error('自动复制被浏览器阻止，文字已选中，请长按或按 Ctrl+C 复制。')
}
