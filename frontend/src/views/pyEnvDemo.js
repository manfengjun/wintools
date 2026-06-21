const delay = ms => new Promise(resolve => setTimeout(resolve, ms))

export async function playPyEnvDemo({ setState, addLog, wait = delay }) {
  setState({ installed: false, installing: true, version: '', pythonExe: '', pipInstalled: false })
  addLog('正在准备 Python 3.12 安装环境…', 'info')
  await wait(3000)
  addLog('正在下载 Python 官方安装包（国内加速）…', 'info')
  await wait(4000)
  addLog('正在配置 pip、pygame、numpy 等教学常用库…', 'info')
  await wait(4000)
  addLog('✅ Python 环境安装完成，可以开始上课了。', 'success')
  setState({
    installed: true,
    installing: false,
    version: 'Python 3.12.0',
    pythonExe: 'C:\\Python\\3.12\\python.exe',
    pipInstalled: true,
  })
}
