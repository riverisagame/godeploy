<template>
  <div class="login-container">
    <el-card class="login-card" shadow="always">
      <template #header>
        <div class="card-header">
          <h2>系统登录</h2>
        </div>
      </template>
      <el-form :model="loginForm" @submit.prevent="handleLogin" label-position="top">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="loginForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading" class="login-button">
            登录
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const router = useRouter()
const loading = ref(false)
const loginForm = reactive({
  username: '',
  password: ''
})

const handleLogin = async () => {
  if (!loginForm.username || !loginForm.password) {
    ElMessage.error('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    const res = await api.login({
      username: loginForm.username,
      password: loginForm.password
    })
    
    // axios interceptor might unwrap data, assuming res or res.data
    const data = (res as any).data || res
    if (data.token) {
      localStorage.setItem('token', data.token)
      ElMessage.success('登录成功')
      router.push('/')
    } else {
      ElMessage.error('登录失败: 未收到 token')
    }
  } catch (err: any) {
    console.error('Login error', err)
    ElMessage.error(err.response?.data?.message || err.message || '登录异常')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: var(--el-bg-color-page);
}

.login-card {
  width: 400px;
  background-color: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-darker);
}

.card-header h2 {
  margin: 0;
  text-align: center;
  color: var(--el-text-color-primary);
}

.login-button {
  width: 100%;
  margin-top: 20px;
}
</style>
