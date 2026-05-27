/**
 * deploy.ts - 部署相关纯函数工具库
 * 从 Dashboard.vue 提取，方便单元测试与复用
 * @Ref: docs/sps/plans/20260527_m7_frontend_test_ir.md | @Date: 2026-05-27
 */

/** 根据部署状态返回 Element Plus Tag 类型 */
export function getStatusTagType(status: string): string {
  switch (status) {
    case 'success':    return 'success'
    case 'failed':     return 'danger'
    case 'pending':    return 'warning'
    case 'deploying':  return 'warning'
    case 'rolled_back': return 'info'
    default:           return 'info'
  }
}

/** 根据部署状态返回中文文本 */
export function getStatusText(status: string): string {
  switch (status) {
    case 'success':    return '部署成功'
    case 'failed':     return '部署失败'
    case 'pending':    return '部署中...'
    case 'deploying':  return '部署中...'
    case 'rolled_back': return '已回滚'
    default:           return status
  }
}

/** 将 ISO 时间字符串格式化为本地时间 */
export function formatTime(timeStr: string): string {
  return new Date(timeStr).toLocaleString()
}

/**
 * 构建 WebSocket 连接 URL
 * @param protocol - window.location.protocol（'http:' 或 'https:'）
 * @param host - window.location.host
 * @param taskId - 任务 ID
 */
export function buildWSUrl(protocol: string, host: string, taskId: number): string {
  const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:'
  return `${wsProtocol}//${host}/api/ws/tasks/${taskId}/log`
}
