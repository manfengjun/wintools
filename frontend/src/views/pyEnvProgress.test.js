import test from 'node:test'
import assert from 'node:assert/strict'

import { initialProgress, applyProgress } from './pyEnvProgress.js'

test('initialProgress returns default state with no packages', () => {
  const state = initialProgress()
  assert.equal(state.installing, false)
  assert.equal(state.visible, false)
  assert.equal(state.percent, 0)
  assert.deepEqual(state.packages, {})
  assert.deepEqual(state.logs, [])
})

test('initialProgress builds package map from name list', () => {
  const state = initialProgress(['numpy', 'pygame'])
  assert.deepEqual(state.packages, { numpy: 'pending', pygame: 'pending' })
})

test('keeps completed logs visible after installation ends', () => {
  const state = applyProgress(initialProgress(), { step: 'done', message: '完成', done: true, percent: 100 })
  assert.equal(state.installing, false)
  assert.equal(state.visible, true)
  assert.equal(state.logs.length, 1)
  assert.equal(state.logs[0].message, '完成')
})

test('updates one package without discarding other package states', () => {
  const state = applyProgress(initialProgress(['numpy', 'pygame']), {
    step: 'install-package', package: 'numpy', package_status: 'success'
  })
  assert.equal(state.packages.numpy, 'success')
  assert.equal(state.packages.pygame, 'pending')
})

test('sets error state and keeps it visible', () => {
  const state = applyProgress(initialProgress(), {
    step: 'error', error: '下载失败'
  })
  assert.equal(state.installing, false)
  assert.equal(state.visible, true)
  assert.equal(state.error, '下载失败')
  assert.equal(state.logs.length, 1)
  assert.equal(state.logs[0].type, 'error')
})

test('marks installing true during active steps', () => {
  const state = applyProgress(initialProgress(), {
    step: 'download', message: '正在下载...'
  })
  assert.equal(state.installing, true)
  assert.equal(state.visible, true)
})

test('logs multiple events in order', () => {
  let state = initialProgress()
  state = applyProgress(state, { step: 'prepare', message: '准备中' })
  state = applyProgress(state, { step: 'download', message: '下载中' })
  state = applyProgress(state, { step: 'done', message: '完成', done: true })
  assert.equal(state.logs.length, 3)
  assert.equal(state.logs[0].message, '准备中')
  assert.equal(state.logs[1].message, '下载中')
  assert.equal(state.logs[2].message, '完成')
})
