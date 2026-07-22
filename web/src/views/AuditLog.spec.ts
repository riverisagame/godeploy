import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import AuditLog from './AuditLog.vue'
import * as apiModule from '../api'

vi.mock('../api', () => ({
  api: {
    getAuditLogs: vi.fn()
  }
}))

describe('AuditLog.vue', () => {
  it('renders audit logs table', async () => {
    // @ts-ignore
    apiModule.api.getAuditLogs.mockResolvedValue({
      data: {
        data: [
          { id: 1, username: 'admin', method: 'POST', path: '/api/test', details: '', created_at: '2026-07-22T00:00:00Z' }
        ],
        total: 1
      }
    })

    const wrapper = mount(AuditLog as any, {
      global: {
        stubs: ['el-table', 'el-table-column', 'el-pagination']
      }
    })

    await new Promise(r => setTimeout(r, 10)) // wait for onMounted
    
    expect(wrapper.exists()).toBe(true)
    expect(apiModule.api.getAuditLogs).toHaveBeenCalledWith(1, 10)
  })
})
