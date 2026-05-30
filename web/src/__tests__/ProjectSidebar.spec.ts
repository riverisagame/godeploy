import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import ElementPlus from 'element-plus'
import ProjectSidebar from '../components/ProjectSidebar.vue'

function mountSidebar(props: any) {
  return mount(ProjectSidebar, {
    props,
    global: { plugins: [ElementPlus] }
  })
}

describe('ProjectSidebar', () => {
  const mockProjects = [
    { id: 'proj-a', name: 'Project A', environments: [{ id: 'dev' }] },
    { id: 'proj-b', name: 'Project B', environments: [] },
  ]

  it('renders project list', () => {
    const wrapper = mountSidebar({ projects: mockProjects, selectedId: '' })
    expect(wrapper.text()).toContain('Project A')
    expect(wrapper.text()).toContain('Project B')
  })

  it('highlights selected project', () => {
    const wrapper = mountSidebar({ projects: mockProjects, selectedId: 'proj-a' })
    const items = wrapper.findAll('.project-item')
    expect(items[0].classes()).toContain('active')
    expect(items[1].classes()).not.toContain('active')
  })

  it('emits select-project on click', async () => {
    const wrapper = mountSidebar({ projects: mockProjects, selectedId: '' })
    await wrapper.findAll('.project-item')[0].trigger('click')
    expect(wrapper.emitted('select-project')?.[0]).toEqual([mockProjects[0]])
  })

  it('shows empty state when no projects', () => {
    const wrapper = mountSidebar({ projects: [], selectedId: '' })
    expect(wrapper.text()).toContain('未加载到项目配置')
  })
})
