import { createRouter, createWebHashHistory } from 'vue-router'
import tools from '../tools.js'

/**
 * 路由由工具清单自动生成。
 * 新增工具只需在 tools.js 添加一项，路由和导航自动更新。
 * 视图文件遵循 `../views/{id}.vue` 命名约定。
 */
const routes = tools.map(t => ({
  path: '/' + t.id,
  name: t.id,
  component: () => import(`../views/${kebabToPascal(t.id)}.vue`),
}))

// 首页重定向到第一个工具
routes.unshift({ path: '/', redirect: routes[0]?.path || '/' })

function kebabToPascal(str) {
  return str.split('-').map(s => s.charAt(0).toUpperCase() + s.slice(1)).join('')
}

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
