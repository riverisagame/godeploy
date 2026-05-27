/**
 * Login.spec.ts
 * Login 组件渲染测试 - 验证关键 DOM 结构与交互元素存在
 * @Ref: docs/sps/plans/20260527_m7_frontend_test_ir.md | @Date: 2026-05-27
 * 
 * 物理零污染：Mock axios，不产生任何真实 HTTP 请求
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import Login from '@/views/Login.vue'

// Mock axios 防止真实网络请求
vi.mock('axios', () => ({
  default: {
    post: vi.fn(),
    defaults: { headers: { common: {} } },
  },
}))

// Mock Element Plus，但必须保留 default export
vi.mock('element-plus', async (importOriginal) => {
  const actual = await importOriginal<any>()
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn() },
  }
})

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', component: { template: '<div>Dashboard</div>' } },
    { path: '/login', component: Login },
  ],
})

import ElementPlus from 'element-plus'

describe('Login 组件渲染测试', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应渲染 GoDeployer 标题', async () => {
    const wrapper = mount(Login, {
      global: { plugins: [router, ElementPlus] },
    })
    expect(wrapper.text()).toContain('GoDeployer')
  })

  it('应包含用户名输入框', async () => {
    const wrapper = mount(Login, {
      global: { plugins: [router, ElementPlus] },
    })
    // el-input 内部有真实的 input 元素
    const inputs = wrapper.findAll('input')
    expect(inputs.length).toBeGreaterThanOrEqual(1)
  })

  it('应包含登录按钮', async () => {
    const wrapper = mount(Login, {
      global: { plugins: [router, ElementPlus] },
    })
    const btn = wrapper.find('button')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('登录')
  })

  it('应包含系统副标题描述', async () => {
    const wrapper = mount(Login, {
      global: { plugins: [router, ElementPlus] },
    })
    expect(wrapper.text()).toContain('配置驱动多项目多环境代码发布系统')
  })
})
