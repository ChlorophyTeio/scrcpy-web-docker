import test from 'node:test'
import assert from 'node:assert/strict'
import { cameraFrameLayout } from './cameraLayout.js'

test('complete frame fits every viewport and quarter turn without stretching', () => {
  for (const [w,h] of [[390,560],[844,230],[1024,650]]) {
    for (const rotation of [0,90,180,270]) {
      const size = cameraFrameLayout({width:1920,height:1080},{width:w,height:h},rotation)
      assert.ok(Math.abs(size.width / size.height - 16 / 9) < 1e-9)
      const rotated = rotation % 180 === 90
      const bounds = rotated ? [size.height,size.width] : [size.width,size.height]
      assert.ok(bounds[0] <= w + 1e-6 && bounds[1] <= h + 1e-6)
      assert.ok(Math.abs(bounds[0]-w)<1e-6 || Math.abs(bounds[1]-h)<1e-6)
    }
  }
})
test('fill covers both viewport dimensions after rotation', () => {
  for (const rotation of [0,90]) {
    const size = cameraFrameLayout({width:1920,height:1080},{width:390,height:560},rotation,'cover')
    const bounds = rotation ? [size.height,size.width] : [size.width,size.height]
    assert.ok(bounds[0]>=390-1e-6 && bounds[1]>=560-1e-6)
  }
})
test('initial frame and zero-sized container remain finite', () => {
  const result = cameraFrameLayout({width:0,height:0},{width:0,height:0})
  assert.ok(Number.isFinite(result.width) && result.width > 0)
  assert.ok(Number.isFinite(result.height) && result.height > 0)
})
