import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import Login from './Login.vue'
import { api } from '../api'
import ElementPlus from 'element-plus'

vi.mock('../api', () => ({
  api: {
    login: vi.fn()
  }
}))

// Mock useRouter
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush
  })
}))

describe('Login.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('should render login form', () => {
    const wrapper = mount(Login, {
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button').text()).toContain('登录')
  })

  it('should call api.login and store token on successful submit', async () => {
    (api.login as any).mockResolvedValue({ data: { token: 'mock_token' } })
    
    const wrapper = mount(Login, {
      global: { plugins: [ElementPlus] }
    })
    
    // Fill form
    await wrapper.find('input[type="text"]').setValue('admin')
    await wrapper.find('input[type="password"]').setValue('password')
    
    // Submit
    await wrapper.find('form').trigger('submit.prevent')
    
    // Check api call
    expect(api.login).toHaveBeenCalledWith({ username: 'admin', password: 'password' })
    
    // Await microtasks for promise resolution
    await flushPromises()
    
    // Check localStorage
    expect(localStorage.getItem('token')).toBe('mock_token')
    
    // Check router push
    expect(mockPush).toHaveBeenCalledWith('/')
  })
})
