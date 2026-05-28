import { mount } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import UserManagement from '../views/UserManagement.vue'
import axios from 'axios'

// Mock axios
vi.mock('axios')
const mockedAxios = vi.mocked(axios)

describe('UserManagement.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders correctly and fetches users', async () => {
    const mockUsers = [
      {
        id: 1,
        username: 'admin',
        role: 'admin',
        created_at: '2026-05-28T00:00:00Z',
        bound_git_authors: '',
        restrict_git_authors: false,
        permitted_projects: '*'
      }
    ]

    mockedAxios.get.mockResolvedValueOnce({ data: mockUsers })

    const wrapper = mount(UserManagement, {
      global: {
        stubs: ['el-table', 'el-table-column', 'el-button', 'el-tag', 'el-dialog', 'el-form', 'el-form-item', 'el-input', 'el-select', 'el-option', 'el-switch']
      }
    })

    // wait for promises
    await new Promise(r => setTimeout(r, 50))

    expect(mockedAxios.get).toHaveBeenCalledWith('/api/users')
    
    // Check if the add user button exists
    const addButton = wrapper.findComponent({ name: 'el-button' })
    expect(addButton.exists()).toBe(true)
  })
})
