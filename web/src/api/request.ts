import axios from 'axios'
import type { AxiosInstance, AxiosResponse, AxiosError } from 'axios'
import { ElMessage } from 'element-plus'

// Define the API client
const apiClient: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request Interceptor
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response Interceptor
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '#/login'
      ElMessage.warning('会话已过期，请重新登录')
      return Promise.reject(error)
    }

    let errorMessage = '请求失败'
    if (error.response) {
      if (typeof error.response.data === 'string') {
        errorMessage = error.response.data
      } else if (error.response.data && (error.response.data as any).message) {
        errorMessage = (error.response.data as any).message
      } else {
        errorMessage = `请求失败: ${error.response.status}`
      }
    } else if (error.request) {
      errorMessage = '网络错误，无法连接到服务器'
    } else {
      errorMessage = error.message
    }
    
    // Only show error message if it's not explicitly disabled
    if (!error.config || (error.config as any).showError !== false) {
      ElMessage.error(errorMessage)
    }
    
    return Promise.reject(error)
  }
)

export default apiClient
