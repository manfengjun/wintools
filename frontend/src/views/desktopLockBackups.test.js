import test from 'node:test'
import assert from 'node:assert/strict'

import { normalizeBackupItems } from './desktopLockBackups.js'

test('normalizeBackupItems initializes icon_base64 for backup rows', () => {
  const result = normalizeBackupItems([
    { name: 'Alpha.lnk', mod_time: '2026-06-20 09:00:00' },
  ])

  assert.deepEqual(result, [
    { name: 'Alpha.lnk', mod_time: '2026-06-20 09:00:00', icon_base64: '' },
  ])
})

test('normalizeBackupItems preserves icons returned by ListBackups', () => {
  const result = normalizeBackupItems([
    { name: 'Beta.url', mod_time: '2026-06-20 09:00:01', icon_base64: 'data:image/png;base64,BBB' },
  ])

  assert.deepEqual(result, [
    { name: 'Beta.url', mod_time: '2026-06-20 09:00:01', icon_base64: 'data:image/png;base64,BBB' },
  ])
})
