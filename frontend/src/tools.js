/**
 * 工具清单 —— 新增工具只需在此添加一项
 *
 * labelKey 引用 locale.js 中的翻译 key。
 * 导航标签会自动根据当前语言显示。
 */
import IconLock from './components/icons/IconLock.vue'
import IconPython from './components/icons/IconPython.vue'

const tools = [
  {
    id: 'desktop-lock',
    labelKey: 'nav.desktopLock',
    icon: IconLock,
    descKey: 'desktopLock.desc',
  },
  {
    id: 'py-env',
    labelKey: 'nav.pyEnv',
    icon: IconPython,
    descKey: 'pyEnv.desc',
  },
]

export default tools
