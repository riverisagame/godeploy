import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import UserList from './UserList.vue'
import { api } from '../api'
import ElementPlus from 'element-plus'

vi.mock('../api', () => ({
  api: {
    getUsers: vi.fn(),
    createUser: vi.fn()
  }
}))

describe('UserList.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(api.getUsers as any).mockResolvedValue({
      data: [
        { id: 1, username: 'admin', role: 'admin', created_at: '2023-01-01T00:00:00Z' },
        { id: 2, username: 'test_dev', role: 'developer', created_at: '2023-01-02T00:00:00Z' }
      ]
    })
  })

  it('should fetch and display users on mount', async () => {
    const wrapper = mount(UserList, {
      global: {
        plugins: [ElementPlus]
      }
    })
    
    await flushPromises()
    expect(api.getUsers).toHaveBeenCalled()
    
    // We expect the text to contain the usernames
    expect(wrapper.text()).toContain('admin')
    expect(wrapper.text()).toContain('test_dev')
  })
})
