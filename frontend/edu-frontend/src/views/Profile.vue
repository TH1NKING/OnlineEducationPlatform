<template>
  <div class="profile-container">
    <el-button @click="$router.push('/')" icon="ArrowLeft" style="margin-bottom: 20px">返回首页</el-button>
    
    <el-card>
      <div class="user-info">
        <el-avatar :size="60" src="https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" />
        <div style="margin-left: 20px;">
          <h2>{{ username }}</h2>
          <el-tag>{{ role === 'teacher' ? '教师' : '学生' }}</el-tag>
        </div>
      </div>
    </el-card>

    <h3 style="margin-top: 30px">我的学习进度</h3>
    <el-table :data="myCourses" style="width: 100%" border stripe>
      <el-table-column prop="course.title" label="课程名称" />
      <el-table-column label="封面" width="120">
        <template #default="scope">
          <img :src="scope.row.course.cover_image || 'https://via.placeholder.com/100'" style="width: 80px; height: 50px; object-fit: cover" />
        </template>
      </el-table-column>
      <el-table-column label="学习进度">
        <template #default="scope">
          <el-progress :percentage="scope.row.progress || 10" />
        </template>
      </el-table-column>
      <el-table-column label="操作">
        <template #default="scope">
          <el-button type="primary" size="small" @click="$router.push(`/course/${scope.row.course.ID}`)">
            继续学习
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '../utils/request'

const username = localStorage.getItem('username') || '用户'
const role = localStorage.getItem('role')
const myCourses = ref([])

onMounted(async () => {
  const res = await request.get('/my-courses')
  myCourses.value = res.data
})
</script>

<style scoped>
.profile-container { padding: 20px; max-width: 1000px; margin: 0 auto; }
.user-info { display: flex; align-items: center; }
</style>