import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProjectList from './ProjectList.vue'
import { api } from '../api'

vi.mock('../api', () => ({
  api: {
    getProjects: vi.fn(),
  }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() })
}))

// Mock Element components
const ElementMock = {
  template: '<div><slot/></div>'
}

describe('ProjectList.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(api.getProjects as any).mockResolvedValue({
      data: [
        { id: 1, name: 'proj-1', repo_url: 'git@x', keep_releases: 5, webhook_secret: 'secret-123' },
        { id: 2, name: 'proj-2', repo_url: 'git@y', keep_releases: 3, webhook_secret: '' }
      ]
    })
  })

  it('should display webhook payload URL and secret in the card body', async () => {
    const wrapper = mount(ProjectList, {
      global: {
        stubs: {
          'el-card': ElementMock,
          'el-button': ElementMock,
          'el-icon': ElementMock,
          'el-row': ElementMock,
          'el-col': ElementMock,
          'el-tag': ElementMock,
          'el-dialog': ElementMock,
          'el-form': ElementMock,
          'el-form-item': ElementMock,
          'el-input': ElementMock,
          'el-popconfirm': ElementMock,
          'el-empty': ElementMock,
          'Link': ElementMock,
          'CopyDocument': ElementMock,
          'ArrowRight': ElementMock
        }
      }
    })
    
    await flushPromises()
    expect(wrapper.text()).toContain('api/webhook/github/1')
    expect(wrapper.text()).toContain('secret-123')
    
    // For project 2, without secret
    expect(wrapper.text()).toContain('api/webhook/github/2')
    expect(wrapper.text()).toContain('未配置')
  })
})
