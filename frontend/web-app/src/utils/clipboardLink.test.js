import test from 'node:test'
import assert from 'node:assert/strict'
import { createClipboardLink, createClipboardLinkState, copiedEventText } from './clipboardLink.js'

function fixture(options = {}) {
  const state = createClipboardLinkState(), sent = [], written = []
  let enabled = true, local = 'initial local'
  const link = createClipboardLink({ state, active: () => enabled, readLocal: async () => local,
    writeLocal: async text => { written.push(text); local = text }, sendRemote: text => { sent.push(text); return true }, ...options })
  return { state, sent, written, link, setLocal: text => { local = text }, setActive: value => { enabled = value } }
}
const tick = () => new Promise(resolve => setTimeout(resolve, 0))

test('first local read is a baseline; later copies sync without pasting or echo loops', async () => {
  const f = fixture()
  await f.link.captureLocal(); assert.deepEqual(f.sent, [])
  f.setLocal('中文\n🙂'); await f.link.captureLocal(); await f.link.captureLocal()
  f.link.receiveRemote('中文\n🙂'); await tick()
  assert.deepEqual(f.sent, ['中文\n🙂']); assert.deepEqual(f.written, [])
})

test('phone copy writes the browser clipboard and is not sent back to the phone', async () => {
  const f = fixture()
  await f.link.captureLocal(); f.link.receiveRemote('phone value'); await tick(); await f.link.captureLocal()
  assert.deepEqual(f.written, ['phone value']); assert.deepEqual(f.sent, [])
})

test('background and denied writes retain the latest phone copy for a user gesture', async () => {
  let denied = true
  const f = fixture({ writeLocal: async () => { if (denied) throw Error('permission denied') } })
  f.setActive(false); f.link.receiveRemote('A'); f.link.receiveRemote('B')
  assert.equal(f.state.pending, 'B')
  f.setActive(true); await f.link.flushRemote(); assert.equal(f.state.pending, 'B'); assert.equal(f.state.blocked, 'write')
  await f.link.captureLocal(); assert.deepEqual(f.sent, [])
  denied = false; await f.link.flushRemote(); assert.equal(f.state.pending, null); assert.equal(f.state.local, 'B')
})

test('a phone copy invalidates a stale local read', async () => {
  let resolveRead
  const f = fixture({ readLocal: () => new Promise(resolve => { resolveRead = resolve }) })
  const reading = f.link.captureLocal({ explicit: true })
  f.link.receiveRemote('new phone value'); resolveRead('old local value'); await reading; await tick()
  assert.deepEqual(f.sent, []); assert.equal(f.state.local, 'new phone value')
})

test('writes serialize and coalesce newer phone copies', async () => {
  let finish
  const values = [], f = fixture({ writeLocal: text => { values.push(text); return new Promise(resolve => { finish = resolve }) } })
  f.link.receiveRemote('A'); f.link.receiveRemote('B'); f.link.receiveRemote('C')
  finish(); await tick(); assert.deepEqual(values, ['A', 'C'])
  finish(); await tick(); assert.equal(f.state.pending, null); assert.equal(f.state.local, 'C')
})

test('disabled views do not sync; explicit local copy supersedes pending phone text', async () => {
  const f = fixture()
  f.setActive(false); assert.equal(f.link.sendLocal('local', true), false)
  f.link.receiveRemote('phone'); f.setActive(true)
  assert.equal(f.link.sendLocal('new copy', true), true); assert.equal(f.state.pending, null)
  f.link.dispose(); f.link.receiveRemote('ignored'); await f.link.captureLocal()
  assert.deepEqual(f.sent, ['new copy'])
})

test('copy events respect actual selections and canceled/password copies', () => {
  assert.equal(copiedEventText({ target: { value: 'first second', selectionStart: 6, selectionEnd: 12 } }), 'second')
  assert.equal(copiedEventText({ target: { type: 'password', value: 'secret', selectionStart: 0, selectionEnd: 6 } }), '')
  assert.equal(copiedEventText({ defaultPrevented: true }, 'selected'), '')
  assert.equal(copiedEventText({}, 'selected text'), 'selected text')
})
