import test from 'node:test'
import assert from 'node:assert/strict'
import { CLIPBOARD_MAX_BYTES, clipboardTextError, shouldRouteRemotePaste } from './clipboard.js'

test('clipboard validation measures UTF-8 bytes without silently truncating text', () => {
  assert.equal(clipboardTextError('中文\n🙂\t '), '')
  assert.equal(clipboardTextError('a'.repeat(CLIPBOARD_MAX_BYTES)), '')
  assert(clipboardTextError('中'.repeat(Math.floor(CLIPBOARD_MAX_BYTES / 3) + 1)))
  assert(clipboardTextError(''))
})

test('paste targets only the focused remote surface and its hidden input', () => {
  const event = (local = false, hidden = false) => ({ target: { classList: { contains: () => hidden }, closest: () => local }, clipboardData: { getData: () => 'hello' } })
  assert.equal(shouldRouteRemotePaste(event(), { focused: true }), true)
  assert.equal(shouldRouteRemotePaste(event(true), { focused: true }), false)
  assert.equal(shouldRouteRemotePaste(event(true, true), { focused: true }), true)
  for (const policy of [{ focused: false }, { focused: true, modalOpen: true }, { focused: true, editing: true }, { focused: true, camera: true }]) {
    assert.equal(shouldRouteRemotePaste(event(), policy), false)
  }
  assert.equal(shouldRouteRemotePaste({ ...event(), defaultPrevented: true }, { focused: true }), false)
})
