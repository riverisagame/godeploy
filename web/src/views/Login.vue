<template>
  <div class="login-wrapper">
    <div class="login-card">
      <div class="login-header">
        <h2 class="title">GoDeployer</h2>
        <p class="subtitle">配置驱动多项目多环境代码发布系统</p>
      </div>

      <el-form :model="loginForm" :rules="rules" ref="loginFormRef" size="large" label-position="top">
        <el-form-item label="用户名" prop="username">
          <el-input 
            v-model="loginForm.username" 
            placeholder="请输入管理员账户" 
            prefix-icon="User"
            clearable
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input 
            v-model="loginForm.password" 
            type="password" 
            placeholder="请输入密码" 
            prefix-icon="Lock" 
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-button 
          type="primary" 
          :loading="loading" 
          class="login-btn" 
          @click="handleLogin"
        >
          登录
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const router = useRouter()
const loginFormRef = ref()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const handleLogin = async () => {
  if (!loginFormRef.value) return
  
  await loginFormRef.value.validate(async (valid: boolean) => {
    if (!valid) return
    
    loading.value = true
    try {
      // 这里的 API 相对地址会通过 Vite Dev Server 的 Proxy 或者是生产环境同域代理
      const response = await axios.post('/api/login', {
        username: loginForm.username,
        password: loginForm.password
      })

      const { token, username: returnedUsername, role } = response.data
      
      localStorage.setItem('token', token)
      localStorage.setItem('username', returnedUsername)
      if (role) {
        localStorage.setItem('role', role)
      }
      
      ElMessage.success('登录成功，欢迎回来！')
      router.push('/')
    } catch (error: any) {
      const msg = error.response?.data?.error || '登录失败，请检查网络或用户名密码'
      ElMessage.error(msg)
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.login-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100vh;
  background: radial-gradient(circle at center, #1b263b 0%, #0d1b2a 100%);
}

.login-card {
  width: 420px;
  padding: 40px;
  background-color: rgba(22, 28, 45, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(12px);
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.title {
  font-size: 32px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 10px 0;
  letter-spacing: 1px;
}

.subtitle {
  font-size: 14px;
  color: #8a99ad;
  margin: 0;
}

.login-btn {
  width: 100%;
  margin-top: 15px;
  border-radius: 8px;
  font-weight: 600;
  background: linear-gradient(135deg, #0077b6 0%, #0096c7 100%);
  border: none;
}

.login-btn:hover {
  background: linear-gradient(135deg, #0096c7 0%, #48cae4 100%);
}

:deep(.el-form-item__label) {
  color: #a9b7c6;
  font-weight: 500;
  padding-bottom: 4px;
}

:deep(.el-input__wrapper) {
  background-color: #121824;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
  border-radius: 8px;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #00b4d8;
}

:deep(.el-input__inner) {
  color: #ffffff;
}
</style>
