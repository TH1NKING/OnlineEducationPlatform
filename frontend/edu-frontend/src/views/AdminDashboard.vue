<template>
  <div class="admin-container">
    <div class="header-bar">
      <h2>🖥️ 管理员监控台</h2>
      <el-button @click="$router.push('/')">返回首页</el-button>
    </div>

    <el-row :gutter="20" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <template #header><div class="card-header">👥 注册用户</div></template>
          <div class="stat-value">{{ stats.user_count }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <template #header><div class="card-header">📚 总课程数</div></template>
          <div class="stat-value">{{ stats.course_count }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <template #header><div class="card-header">🔥 平台总浏览</div></template>
          <div class="stat-value" style="color: #F56C6C">{{ stats.view_count || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <template #header><div class="card-header">⏳ 待审核</div></template>
          <div class="stat-value" style="color: #E6A23C">{{ stats.pending_count }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="audit-card">
      <template #header>
        <div class="card-header-flex">
          <span>📝 待审核课程列表</span>
          <el-button type="primary" link icon="Refresh" @click="fetchStats">刷新</el-button>
        </div>
      </template>
      
      <el-table :data="stats.pending_list" style="width: 100%" stripe border>
        <el-table-column label="提交教师" width="150">
           <template #default="scope">
             <div style="display:flex; align-items:center; gap:5px;">
               <el-avatar :size="25" style="background:#409EFF">{{ scope.row.teacher?.username?.charAt(0) }}</el-avatar>
               {{ scope.row.teacher?.username }}
             </div>
           </template>
        </el-table-column>
        <el-table-column prop="title" label="课程标题" />
        <el-table-column label="分类" width="100">
           <template #default="scope">
              <el-tag>{{ scope.row.category }}</el-tag>
           </template>
        </el-table-column>
        <el-table-column prop="created_at" label="提交时间" width="180">
           <template #default="scope">{{ new Date(scope.row.CreatedAt).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="scope">
             <el-button type="success" size="small" @click="audit(scope.row.ID, 1)">通过</el-button>
             <el-button type="danger" size="small" @click="audit(scope.row.ID, 2)">驳回</el-button>
             <el-button link type="primary" size="small" @click="$router.push(`/course/${scope.row.ID}`)">预览</el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <el-empty v-if="!stats.pending_list || stats.pending_list.length === 0" description="暂无待审核课程，太棒了！" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '../utils/request'
import { ElMessage } from 'element-plus'

const stats = ref({
  user_count: 0,
  course_count: 0,
  view_count: 0,
  pending_count: 0,
  pending_list: []
})

const fetchStats = async () => {
  try {
    const res = await request.get('/admin/stats')
    stats.value = res
  } catch (e) {
    // 错误处理在拦截器中
  }
}

const audit = async (id, status) => {
  try {
    await request.put('/admin/audit', { id, status })
    ElMessage.success(status === 1 ? '已通过该课程' : '已驳回该课程')
    fetchStats() // 刷新列表
  } catch(e) {}
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
.admin-container { max-width: 1200px; margin: 20px auto; padding: 20px; }
.header-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.stat-row { margin-bottom: 30px; }
.stat-card { text-align: center; }
.card-header { font-weight: bold; color: #606266; }
.stat-value { font-size: 32px; font-weight: bold; margin-top: 10px; color: #303133; }
.card-header-flex { display: flex; justify-content: space-between; align-items: center; }
</style>