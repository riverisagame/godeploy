import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('api instance', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('attaches Authorization header when token exists', async () => {
    localStorage.setItem('token', 'test-jwt-token')
    const { default: api } = await import('../utils/api')
    const interceptor = api.interceptors.request.handlers[0]
    const config = await interceptor.fulfilled({ headers: {} } as any)
    expect(config.headers.Authorization).toBe('Bearer test-jwt-token')
  })

  it('does not attach Authorization header when no token', async () => {
    const { default: api } = await import('../utils/api')
    const interceptor = api.interceptors.request.handlers[0]
    const config = await interceptor.fulfilled({ headers: {} } as any)
    expect(config.headers.Authorization).toBeUndefined()
  })

  it('baseURL is /api', async () => {
    const { default: api } = await import('../utils/api')
    expect(api.defaults.baseURL).toBe('/api')
  })
})
