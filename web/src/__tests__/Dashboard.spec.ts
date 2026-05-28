import { mount } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Dashboard from '../views/Dashboard.vue'
import axios from 'axios'
import { createRouter, createWebHistory } from 'vue-router'

// Mock axios
vi.mock('axios')
const mockedAxios = vi.mocked(axios)

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard }
  ]
})

describe('Dashboard.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('token', 'fake-jwt-token')
  })

  it('renders correctly and fetches projects', async () => {
    const mockProjects = [
      {
        id: 'test-app',
        name: 'Mock Test App',
        repo: 'git@github.com:mock/test-app.git',
        branch: 'main',
        environments: [
          { id: 'testing', name: 'Test Env' }
        ]
      }
    ]
    
    // axios.get 会被调用两次：一次是 /api/projects，一次可能因为立刻选中 default tab 而调用 /api/tasks
    mockedAxios.get.mockImplementation((url) => {
      if (url === '/api/projects') {
        return Promise.resolve({ data: mockProjects })
      }
      return Promise.resolve({ data: [] })
    })

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [router],
        stubs: ['el-container', 'el-header', 'el-main', 'el-menu', 'el-menu-item', 'el-sub-menu', 'el-tabs', 'el-tab-pane', 'el-button', 'el-icon', 'router-view', 'GitCommit', 'el-descriptions', 'el-descriptions-item', 'el-tag', 'el-empty', 'el-scrollbar', 'el-input', 'el-form-item', 'el-form', 'el-table-column', 'el-button-group', 'el-table', 'el-dialog', 'Platform', 'User', 'Monitor', 'Upload', 'Link', 'el-radio-button', 'el-radio-group', 'el-option', 'el-select', 'el-col', 'el-row', 'View', 'el-switch']
      }
    })

    // wait for promises
    await new Promise(r => setTimeout(r, 50))

    // Because interceptors in actual codebase add the Authorization header,
    // in our unit test isolated from main.ts interceptors, it's called WITHOUT the header unless we explicitly pass it.
    // So we just assert it was called with the right URL.
    expect(mockedAxios.get).toHaveBeenCalledWith('/api/projects')
    
    // Assert components exist or logic runs
    expect(wrapper.vm.projects).toEqual(mockProjects)
    expect(wrapper.vm.activeEnvTab).toBe('testing')
  })
})
