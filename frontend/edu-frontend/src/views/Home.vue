<template>
  <div class="home-container">
    <el-header class="header">
      <div class="logo-area">
        <el-icon class="logo-icon" :size="30" color="#409EFF"><ElementPlus /></el-icon>
        <span class="logo-text">EduPlatform 在线教育</span>
      </div>
      
      <div class="user-area">
        <span class="welcome-text">你好, {{ username }} ({{ userRole === 'teacher' ? '教师' : '学生' }})</span>
        <el-button type="primary" link @click="$router.push('/profile')">
          <el-icon><User /></el-icon> 个人中心
        </el-button>
        <el-divider direction="vertical" />
        <el-button type="danger" link @click="logout">
          <el-icon><SwitchButton /></el-icon> 退出
        </el-button>
      </div>
    </el-header>

    <div class="main-content">
      <div class="section-title">
        <h3>🔥 热门课程推荐</h3>
      </div>
      <el-carousel :interval="4000" type="card" height="220px" v-if="hotCourses.length > 0">
        <el-carousel-item v-for="item in hotCourses" :key="item.ID">
          <div class="hot-card" @click="goToDetail(item.ID)">
            <img :src="item.cover_image || `https://picsum.photos/seed/${item.ID}/600/300`" class="hot-img"/>
            <div class="hot-info">
              <h3>{{ item.title }}</h3>
              <p>热度: {{ item.view_count }} 🔥</p>
            </div>
          </div>
        </el-carousel-item>
      </el-carousel>

      <div class="toolbar">
        <el-tabs v-model="activeCategory" @tab-click="handleCategoryChange" class="category-tabs">
          <el-tab-pane label="全部课程" name="all"></el-tab-pane>
          <el-tab-pane label="前端开发" name="frontend"></el-tab-pane>
          <el-tab-pane label="后端架构" name="backend"></el-tab-pane>
          <el-tab-pane label="人工智能" name="ai"></el-tab-pane>
          <el-tab-pane label="运维/测试" name="ops"></el-tab-pane>
        </el-tabs>

        <el-button v-if="userRole === 'teacher'" type="primary" class="create-btn" @click="showCreateDialog = true">
          <el-icon style="margin-right: 5px"><Plus /></el-icon> 发布新课程
        </el-button>
      </div>

      <div class="course-list" v-loading="loading">
        <el-empty v-if="courseList.length === 0" description="该分类下暂无课程" />
        
        <el-card 
          v-for="item in courseList" 
          :key="item.ID" 
          class="course-card" 
          shadow="hover"
          @click="goToDetail(item.ID)"
        >
          <div class="image-wrapper">
             <img 
              :src="item.cover_image || `https://picsum.photos/seed/${item.ID}/300/180`" 
              class="course-cover"
            />
            <div class="category-tag">{{ getCategoryName(item.category) }}</div>
          </div>
          
          <div class="card-body">
            <h4 class="course-title" :title="item.title">{{ item.title }}</h4>
            <p class="course-desc">{{ item.description }}</p>
            
            <div class="card-footer">
              <div class="meta">
                 <span class="price" v-if="item.price > 0">¥ {{ item.price }}</span>
                 <el-tag type="success" size="small" v-else>免费</el-tag>
                 <span class="views"><el-icon><View /></el-icon> {{ item.view_count }}</span>
              </div>
              <el-button type="primary" link>详情 >></el-button>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <el-dialog v-model="showCreateDialog" title="👩‍🏫 发布新课程" width="600px">
      <el-form :model="newCourse" label-width="80px">
        <el-form-item label="课程标题">
          <el-input v-model="newCourse.title" placeholder="例如：Vue3 高级实战" />
        </el-form-item>
        
        <el-form-item label="课程分类">
          <el-select v-model="newCourse.category" placeholder="请选择分类" style="width: 100%">
            <el-option label="前端开发" value="frontend" />
            <el-option label="后端架构" value="backend" />
            <el-option label="人工智能" value="ai" />
            <el-option label="运维/测试" value="ops" />
          </el-select>
        </el-form-item>

        <el-form-item label="课程简介">
          <el-input v-model="newCourse.description" type="textarea" rows="3" placeholder="请输入课程内容..." />
        </el-form-item>
        
        <el-form-item label="课程价格">
          <el-input-number v-model="newCourse.price" :min="0" :precision="2" />
          <span style="margin-left: 10px; color: #999;">(0元代表免费)</span>
        </el-form-item>
        
        <el-form-item label="课程视频">
          <el-upload
            class="upload-demo"
            action="#" 
            :http-request="uploadFile"
            :limit="1"
            :show-file-list="true"
            accept=".mp4,.avi,.mov"
          >
            <el-button type="primary">
              <el-icon><VideoPlay /></el-icon> 上传视频文件
            </el-button>
            <template #tip>
              <div class="el-upload__tip">支持 mp4, avi 格式</div>
            </template>
          </el-upload>
          <div v-if="newCourse.video_url" class="upload-success">
            <el-icon color="#67C23A"><CircleCheck /></el-icon> 视频上传成功
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" @click="createCourse" :loading="isSubmitting">确认发布</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, User, SwitchButton, VideoPlay, CircleCheck, ElementPlus, View } from '@element-plus/icons-vue'

const router = useRouter()

// 状态变量
const userRole = ref(localStorage.getItem('role') || 'student')
const username = ref(localStorage.getItem('username') || '学员')
const courseList = ref([])
const hotCourses = ref([]) // 热门课程数据
const loading = ref(false)
const showCreateDialog = ref(false)
const isSubmitting = ref(false)
const activeCategory = ref('all') // 当前选中的分类

// 新课程表单
const newCourse = ref({
  title: '',
  description: '',
  price: 0,
  video_url: '', 
  category: '', // 新增分类字段
  teacher_id: 0 
})

// 辅助函数：分类名映射
const getCategoryName = (key) => {
  const map = {
    'frontend': '前端',
    'backend': '后端',
    'ai': 'AI/算法',
    'ops': '运维'
  }
  return map[key] || '综合'
}

// 1. 获取课程列表 (支持分类筛选)
const fetchCourses = async () => {
  loading.value = true
  try {
    // 构建查询参数
    const params = {}
    if (activeCategory.value !== 'all') {
      params.category = activeCategory.value
    }
    const res = await request.get('/courses', { params })
    courseList.value = res.data
  } catch (e) {
    console.error("获取列表失败", e)
  } finally {
    loading.value = false
  }
}

// 2. 获取热门推荐
const fetchHotCourses = async () => {
  try {
    const res = await request.get('/courses', { params: { sort: 'hot' } })
    hotCourses.value = res.data
  } catch (e) {}
}

// 3. 切换分类
const handleCategoryChange = () => {
  fetchCourses()
}

// 4. 跳转详情
const goToDetail = (courseId) => {
  router.push(`/course/${courseId}`)
}

// 5. 上传视频
const uploadFile = async (param) => {
  const formData = new FormData()
  formData.append('file', param.file)
  try {
    const res = await request.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    newCourse.value.video_url = res.url
    ElMessage.success('视频上传成功！')
  } catch (e) {
    ElMessage.error('上传失败')
  }
}

// 6. 发布课程
const createCourse = async () => {
  if (!newCourse.value.title || !newCourse.value.video_url || !newCourse.value.category) {
    ElMessage.warning('请填写完整信息（含分类）并上传视频')
    return
  }
  isSubmitting.value = true
  try {
    newCourse.value.teacher_id = parseInt(localStorage.getItem('user_id') || 0)
    await request.post('/courses', newCourse.value)
    ElMessage.success('课程发布成功！')
    showCreateDialog.value = false
    // 重置表单
    newCourse.value = { title: '', description: '', price: 0, video_url: '', category: '' }
    fetchCourses() // 刷新列表
  } catch (e) {
    ElMessage.error('发布失败')
  } finally {
    isSubmitting.value = false
  }
}

const logout = () => {
  localStorage.clear()
  router.push('/login')
}

// 初始化
onMounted(() => {
  fetchCourses()     // 获取主列表
  fetchHotCourses()  // 获取热门推荐
})
</script>

<style scoped>
.home-container { min-height: 100vh; background-color: #f5f7fa; }

/* Header */
.header {
  background-color: #fff; box-shadow: 0 2px 8px rgba(0,0,0,0.05);
  display: flex; justify-content: space-between; align-items: center; padding: 0 40px; height: 60px;
}
.logo-area { display: flex; align-items: center; gap: 10px; font-weight: bold; color: #303133; font-size: 20px;}
.user-area { display: flex; align-items: center; gap: 10px; }
.welcome-text { color: #606266; font-size: 14px; margin-right: 10px; }

/* Content */
.main-content { max-width: 1200px; margin: 20px auto; padding: 0 20px; }

/* Hot Section */
.section-title h3 { margin: 20px 0 15px; border-left: 5px solid #ff6b6b; padding-left: 15px; color: #303133;}
.hot-card { position: relative; height: 100%; cursor: pointer; border-radius: 8px; overflow: hidden;}
.hot-img { width: 100%; height: 100%; object-fit: cover; filter: brightness(0.8); transition: 0.3s;}
.hot-card:hover .hot-img { filter: brightness(1); transform: scale(1.05);}
.hot-info {
  position: absolute; bottom: 0; left: 0; right: 0; padding: 20px;
  background: linear-gradient(to top, rgba(0,0,0,0.8), transparent); color: white;
}
.hot-info h3 { margin: 0 0 5px 0; font-size: 20px; }

/* Toolbar & Tabs */
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-top: 30px; margin-bottom: 20px; }
.category-tabs { flex: 1; margin-right: 20px; :deep(.el-tabs__header) { margin: 0; } }

/* Course List */
.course-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 25px; }
.course-card { border-radius: 8px; border: none; transition: 0.3s; cursor: pointer; }
.course-card:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.1); }

.image-wrapper { position: relative; height: 160px; overflow: hidden; }
.course-cover { width: 100%; height: 100%; object-fit: cover; }
.category-tag {
  position: absolute; top: 10px; right: 10px;
  background: rgba(0,0,0,0.6); color: #fff; padding: 2px 8px;
  border-radius: 4px; font-size: 12px;
}

.card-body { padding: 15px; }
.course-title { font-size: 16px; margin: 0 0 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #303133;}
.course-desc { font-size: 13px; color: #909399; margin-bottom: 15px; height: 40px; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }

.card-footer { display: flex; justify-content: space-between; align-items: center; border-top: 1px solid #ebeef5; padding-top: 10px; }
.meta { display: flex; gap: 10px; align-items: center; }
.price { color: #F56C6C; font-weight: bold; }
.views { font-size: 12px; color: #999; display: flex; align-items: center; gap: 2px; }

.upload-success { margin-top: 10px; font-size: 12px; color: #67C23A; display: flex; align-items: center; gap: 5px; }
</style>