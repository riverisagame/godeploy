import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Layout from './Layout.vue'
import * as auth from '../utils/auth'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ path: '/' }),
  RouterView: { template: '<div></div>' }
}))

// Mock Element Plus components
vi.mock('element-plus', () => ({
  ElContainer: { template: '<div><slot/></div>' },
  ElHeader: { template: '<div><slot/></div>' },
  ElMain: { template: '<div><slot/></div>' },
  ElAside: { template: '<div><slot/></div>' },
  ElMenu: { template: '<div><slot/></div>' },
  ElMenuItem: { template: '<div><slot/></div>', props: ['index'] },
  ElIcon: { template: '<div><slot/></div>' },
  ElButton: { template: '<button><slot/></button>' },
  ElDropdown: { template: '<div><slot/></div>' },
  ElDropdownMenu: { template: '<div><slot/></div>' },
  ElDropdownItem: { template: '<div><slot/></div>' }
}))

// Mock icons
vi.mock('@element-plus/icons-vue', () => ({
  Menu: { template: '<span></span>' },
  Platform: { template: '<span></span>' },
  Setting: { template: '<span></span>' },
  User: { template: '<span></span>' },
  ArrowDown: { template: '<span></span>' }
}))

describe('Layout.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should not render Server Management for non-admins', () => {
    vi.spyOn(auth, 'isAdmin').mockReturnValue(false)
    const wrapper = mount(Layout)
    expect(wrapper.html()).not.toContain('服务器管理')
    expect(wrapper.html()).not.toContain('index="/servers"')
  })

  it('should not render User Management for non-admins', () => {
    vi.spyOn(auth, 'isAdmin').mockReturnValue(false)
    const wrapper = mount(Layout)
    expect(wrapper.html()).not.toContain('用户管理')
    expect(wrapper.html()).not.toContain('index="/users"')
  })

  it('should render Server and User Management for admins', () => {
    vi.spyOn(auth, 'isAdmin').mockReturnValue(true)
    const wrapper = mount(Layout)
    expect(wrapper.html()).toContain('index="/servers"')
    expect(wrapper.html()).toContain('index="/users"')
  })
})
