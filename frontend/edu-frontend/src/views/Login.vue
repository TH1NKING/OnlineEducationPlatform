<template>
  <div class="login-container">
    <el-card class="box-card">
      <h2>在线教育平台 - {{ isLogin ? '登录' : '注册' }}</h2>
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        
        <el-form-item label="角色" v-if="!isLogin">
          <el-radio-group v-model="form.role">
            <el-radio label="student">学生</el-radio>
            <el-radio label="teacher">教师</el-radio>
            </el-radio-group>
        </el-form-item>

        <div class="btn-group">
          <el-button type="primary" @click="handleSubmit">{{ isLogin ? '登录' : '注册' }}</el-button>
          <el-button link @click="isLogin = !isLogin">{{ isLogin ? '去注册' : '去登录' }}</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const router = useRouter()
const isLogin = ref(true)
const form = ref({ username: '', password: '', role: 'student' })

const handleSubmit = async () => {
  if (isLogin.value) {
    // 登录逻辑
    try {
      const res = await request.post('/login', form.value)
      localStorage.setItem('token', res.token) // 存 Token
      localStorage.setItem('role', res.role)   // 存角色
      localStorage.setItem('username', res.username)
      localStorage.setItem('user_id', res.user_id)
      ElMessage.success('登录成功')
      router.push('/')
    } catch (e) {
      console.error(e)
    }
  } else {
    // 注册逻辑
    try {
      await request.post('/register', form.value)
      ElMessage.success('注册成功，请登录')
      isLogin.value = true
    } catch (e) {
        // 错误已在 request.js 处理
    }
  }
}
</script>

<style scoped>
.login-container {
  display: flex; justify-content: center; align-items: center; height: 100vh; background-color: #f0f2f5;
}
.box-card { width: 400px; }
.btn-group { display: flex; justify-content: space-between; margin-top: 20px;}
</style>