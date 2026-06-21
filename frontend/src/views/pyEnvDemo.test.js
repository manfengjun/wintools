import test from 'node:test'
import assert from 'node:assert/strict'
import { playPyEnvDemo } from './pyEnvDemo.js'

test('plays a safe Python installation demo from not installed to ready', async () => {
  const states = []
  const logs = []

  await playPyEnvDemo({
    setState: state => states.push(state),
    addLog: (message, type) => logs.push({ message, type }),
    wait: async () => {},
  })

  assert.equal(states[0].installed, false)
  assert.equal(states.at(-1).installed, true)
  assert.equal(states.at(-1).version, 'Python 3.12.0')
  assert.deepEqual(logs.map(item => item.type), ['info', 'info', 'info', 'success'])
  assert.match(logs.at(-1).message, /安装完成/)
})
