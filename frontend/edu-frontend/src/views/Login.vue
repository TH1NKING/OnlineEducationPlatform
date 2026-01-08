<template>
  <div class="login-container">
    <div class="login-box">
      <div class="header">
        <el-icon :size="40" color="#409EFF"><ElementPlus /></el-icon>
        <h2>EduPlatform</h2>
        <p class="subtitle">开启你的在线学习之旅</p>
      </div>
      
      <el-card class="form-card" shadow="never">
        <h3 class="form-title">{{ isLogin ? '账号登录' : '注册新账号' }}</h3>
        
        <el-form :model="form" size="large">
          <el-form-item>
            <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" type="password" placeholder="密码" show-password :prefix-icon="Lock" />
          </el-form-item>
          
          <el-form-item v-if="!isLogin">
            <el-radio-group v-model="form.role" class="role-group">
              <el-radio-button label="student">我是学生</el-radio-button>
              <el-radio-button label="teacher">我是讲师</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-button type="primary" class="submit-btn" @click="handleSubmit" :loading="loading">
            {{ isLogin ? '立即登录' : '立即注册' }}
          </el-button>
          
          <div class="footer-links">
            <el-button link type="info" @click="toggleMode">
              {{ isLogin ? '还没有账号？去注册' : '已有账号？去登录' }}
            </el-button>
          </div>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, ElementPlus } from '@element-plus/icons-vue'

const router = useRouter()
const isLogin = ref(true)
const loading = ref(false)
const form = ref({ username: '', password: '', role: 'student' })

const toggleMode = () => {
  isLogin.value = !isLogin.value
  form.value = { username: '', password: '', role: 'student' }
}

const handleSubmit = async () => {
  if(!form.value.username || !form.value.password) return ElMessage.warning('请输入用户名和密码')
  
  loading.value = true
  try {
    if (isLogin.value) {
      const res = await request.post('/login', form.value)
      sessionStorage.setItem('token', res.token)
      sessionStorage.setItem('role', res.role)
      sessionStorage.setItem('username', res.username)
      sessionStorage.setItem('user_id', res.user_id)
      ElMessage.success('欢迎回来！')
      router.push('/')
    } else {
      await request.post('/register', form.value)
      ElMessage.success('注册成功，请登录')
      isLogin.value = true
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-box {
  width: 100%;
  max-width: 420px;
  text-align: center;
}

.header {
  margin-bottom: 30px;
  color: #fff;
}
.header h2 { margin: 10px 0 5px; font-size: 28px; letter-spacing: 1px; }
.subtitle { margin: 0; opacity: 0.8; font-size: 14px; }

.form-card {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 16px;
  border: none;
  box-shadow: 0 20px 40px rgba(0,0,0,0.2);
  backdrop-filter: blur(10px);
  padding: 10px 20px 20px;
}

.form-title {
  margin-top: 0;
  margin-bottom: 25px;
  color: #303133;
  font-weight: 600;
}

.role-group { width: 100%; display: flex; }
:deep(.el-radio-button__inner) { width: 100%; border-radius: 4px; }
:deep(.el-radio-button:first-child .el-radio-button__inner) { border-radius: 4px 0 0 4px; width: 145px;}
:deep(.el-radio-button:last-child .el-radio-button__inner) { border-radius: 0 4px 4px 0; width: 145px;}

.submit-btn { width: 100%; font-weight: bold; height: 45px; font-size: 16px; border-radius: 8px; margin-top: 10px; transition: 0.3s; }
.submit-btn:hover { transform: translateY(-2px); box-shadow: 0 5px 15px rgba(64,158,255,0.3); }

.footer-links { margin-top: 15px; }
</style>