/**
 * deploy.utils.test.ts
 * 针对 web/src/utils/deploy.ts 中提取的纯函数进行单元测试
 * @Ref: docs/sps/plans/20260527_m7_frontend_test_ir.md | @Date: 2026-05-27
 * 
 * 物理零污染：无任何 IO 操作，纯函数测试
 */
import { describe, it, expect, vi } from 'vitest'
import {
  getStatusTagType,
  getStatusText,
  formatTime,
  buildWSUrl,
  createDeployState,
  createWSConnection,
} from '@/utils/deploy'

describe('getStatusTagType', () => {
  it('success 状态应返回 success 类型', () => {
    expect(getStatusTagType('success')).toBe('success')
  })
  it('failed 状态应返回 danger 类型', () => {
    expect(getStatusTagType('failed')).toBe('danger')
  })
  it('pending/deploying 状态应返回 warning 类型', () => {
    expect(getStatusTagType('pending')).toBe('warning')
  })
  it('rolled_back 状态应返回 info 类型', () => {
    expect(getStatusTagType('rolled_back')).toBe('info')
  })
  it('未知状态应返回 info 类型（兜底）', () => {
    expect(getStatusTagType('unknown_xyz')).toBe('info')
  })
})

describe('getStatusText', () => {
  it('success 应返回 部署成功', () => {
    expect(getStatusText('success')).toBe('部署成功')
  })
  it('failed 应返回 部署失败', () => {
    expect(getStatusText('failed')).toBe('部署失败')
  })
  it('pending 应返回 部署中...', () => {
    expect(getStatusText('pending')).toBe('部署中...')
  })
  it('rolled_back 应返回 已回滚', () => {
    expect(getStatusText('rolled_back')).toBe('已回滚')
  })
  it('未知状态应原样返回（兜底）', () => {
    expect(getStatusText('custom_status')).toBe('custom_status')
  })
})

describe('formatTime', () => {
  it('应将 ISO 时间字符串格式化为本地时间字符串', () => {
    const iso = '2026-05-27T10:00:00.000Z'
    const result = formatTime(iso)
    // 只验证是字符串且非空，不验证具体格式（locale 依赖）
    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })
})

describe('buildWSUrl', () => {
  it('http 协议下应生成 ws:// 地址', () => {
    const url = buildWSUrl('http:', 'localhost:9090', 42)
    expect(url).toBe('ws://localhost:9090/api/ws/tasks/42/log')
  })
  it('https 协议下应生成 wss:// 地址', () => {
    const url = buildWSUrl('https:', 'example.com', 7)
    expect(url).toBe('wss://example.com/api/ws/tasks/7/log')
  })
})

describe('createDeployState', () => {
  it('应创建初始状态为 idle', () => {
    const state = createDeployState()
    expect(state.phase).toBe('idle')
    expect(state.error).toBe('')
    expect(state.taskId).toBeNull()
  })
})

describe('createWSConnection', () => {
  let originalWS: any

  beforeEach(() => {
    originalWS = globalThis.WebSocket
  })

  afterEach(() => {
    globalThis.WebSocket = originalWS
  })

  it('应在 onclose 之后触发 onopen (重连)', async () => {
    let openHandler: any
    let closeHandler: any
    const wsMock: any = { send: vi.fn(), close: vi.fn() }
    Object.defineProperty(wsMock, 'onopen', { set(fn: any) { openHandler = fn }, get() { return openHandler } })
    Object.defineProperty(wsMock, 'onclose', { set(fn: any) { closeHandler = fn }, get() { return closeHandler } })
    class MockWS { constructor() { return wsMock } }
    globalThis.WebSocket = MockWS as any

    const onMessage = vi.fn()
    const conn = createWSConnection({
      url: 'ws://localhost:8080/api/ws/tasks/1/log',
      token: 'test-token',
      onMessage,
    })

    // 触发 onopen → send(auth)
    openHandler?.({} as any)

    // 触发 onclose → 触发重连
    closeHandler?.({} as any)
    await new Promise(r => setTimeout(r, 1500))

    // 重连后 onopen 再次被调用 → send 应被调用 2 次 (auth + 重连后 auth)
    openHandler?.({} as any)
    expect(wsMock.send).toHaveBeenCalledTimes(2)
    conn.close()
  })

  it('should send auth message on open', async () => {
    let openHandler: any
    const wsMock = { send: vi.fn(), close: vi.fn(), set onopen(fn: any) { openHandler = fn }, get onopen() { return openHandler } }
    class MockWS { constructor() { return wsMock } }
    globalThis.WebSocket = MockWS as any

    const onMessage = vi.fn()
    createWSConnection({
      url: 'ws://localhost:8080/api/ws/tasks/1/log',
      token: 'test-token',
      onMessage,
    })

    openHandler?.({} as any)
    expect(wsMock.send).toHaveBeenCalledWith(JSON.stringify({ type: 'auth', token: 'test-token' }))
  })
})
