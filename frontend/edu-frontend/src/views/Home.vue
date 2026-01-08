<template>
  <div class="home-container">
    <el-header class="header">
      <div class="logo-area">
        <div class="logo-bg"><el-icon size="20" color="#fff"><ElementPlus /></el-icon></div>
        <span class="logo-text">EduPlatform</span>
      </div>
      
      <div class="user-area">
        <el-avatar :size="32" :src="userAvatar" style="background:#409EFF">{{ username.charAt(0) }}</el-avatar>
        <el-dropdown trigger="click">
          <span class="user-link">
            {{ username }} <el-tag size="small" effect="plain" round style="margin: 0 5px">{{ roleName }}</el-tag>
            <el-icon><CaretBottom /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item :icon="User" @click="$router.push('/profile')">个人中心</el-dropdown-item>
              <el-dropdown-item divided :icon="SwitchButton" @click="logout" style="color: #F56C6C">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <div class="main-content">
      
      <div v-if="userRole === 'student'" class="student-view">
        <div class="welcome-banner">
          <h2>👋 嗨，{{ username }}，今天想学点什么？</h2>
          <p>探索最新技术，提升你的核心竞争力</p>
        </div>
        
        <div v-if="hotCourses.length > 0" class="section-block">
          <h3 class="section-title"><el-icon color="#ff6b6b"><hot-water /></el-icon> 热门推荐</h3>
          <el-carousel :interval="5000" type="card" height="240px" class="hot-carousel">
            <el-carousel-item v-for="item in hotCourses" :key="item.ID">
              <div class="hot-card" @click="goToDetail(item.ID)">
                <img :src="item.cover_image || `https://picsum.photos/seed/${item.ID}/600/300`" class="hot-img"/>
                <div class="hot-overlay">
                  <div class="hot-info">
                    <h3>{{ item.title }}</h3>
                    <div class="hot-meta">
                      <span><el-icon><View /></el-icon> {{ item.view_count }} 人正在学</span>
                      <el-tag type="warning" effect="dark" size="small">HOT</el-tag>
                    </div>
                  </div>
                </div>
              </div>
            </el-carousel-item>
          </el-carousel>
        </div>

        <div class="toolbar-sticky">
          <el-tabs v-model="activeCategory" @tab-change="handleCategoryChange" class="custom-tabs">
            <el-tab-pane label="全部分类" name="all"></el-tab-pane>
            <el-tab-pane label="💻 前端开发" name="frontend"></el-tab-pane>
            <el-tab-pane label="⚙️ 后端架构" name="backend"></el-tab-pane>
            <el-tab-pane label="🤖 人工智能" name="ai"></el-tab-pane>
            <el-tab-pane label="🔧 运维测试" name="ops"></el-tab-pane>
          </el-tabs>
        </div>

        <div class="course-grid" v-loading="loading">
          <el-empty v-if="courseList.length === 0" description="暂无相关课程，敬请期待" image-size="200" />
          <el-card v-for="item in courseList" :key="item.ID" class="course-card" :body-style="{ padding: '0px' }" shadow="hover" @click="goToDetail(item.ID)">
            <div class="image-wrapper">
              <img :src="item.cover_image || `https://picsum.photos/seed/${item.ID}/300/180`" loading="lazy" class="course-cover"/>
              <div class="category-badge">{{ getCategoryName(item.category) }}</div>
            </div>
            <div class="card-content">
              <h4 class="course-title" :title="item.title">{{ item.title }}</h4>
              <p class="course-desc">{{ item.description }}</p>
              <div class="card-bottom">
                 <div class="price-tag" v-if="item.price > 0">¥{{ item.price }}</div>
                 <div class="price-tag free" v-else>免费</div>
                 <span class="views"><el-icon><User /></el-icon> {{ item.view_count }}</span>
              </div>
            </div>
          </el-card>
        </div>
      </div>

      <div v-else-if="userRole === 'teacher'" class="teacher-dashboard">
        <div class="dashboard-header-flex">
          <div>
            <h2>🎓 教学工作台</h2>
            <p style="color: #909399; margin: 5px 0 0;">管理您的课程与学生互动</p>
          </div>
          <el-button type="primary" size="large" icon="Plus" round @click="showCreateDialog = true" class="add-btn">
             发布新课程
          </el-button>
        </div>

        <div class="stats-row">
           <el-card class="stat-card" shadow="hover">
              <div class="stat-icon bg-blue"><el-icon><Reading /></el-icon></div>
              <div class="stat-info">
                <div class="label">我的课程</div>
                <div class="num">{{ teacherCourses.length }}</div>
              </div>
           </el-card>
           <el-card class="stat-card" shadow="hover">
              <div class="stat-icon bg-orange"><el-icon><EditPen /></el-icon></div>
              <div class="stat-info">
                <div class="label">待批改</div>
                <div class="num">{{ todoData.homeworks.length }}</div>
              </div>
           </el-card>
           <el-card class="stat-card" shadow="hover">
              <div class="stat-icon bg-green"><el-icon><ChatDotSquare /></el-icon></div>
              <div class="stat-info">
                <div class="label">新提问</div>
                <div class="num">{{ todoData.questions.length }}</div>
              </div>
           </el-card>
        </div>

        <div class="workbench-grid">
           <el-card class="work-card" shadow="hover">
              <template #header><div class="card-header"><span>📝 待批改作业</span></div></template>
              <el-table :data="todoData.homeworks" height="250" style="width: 100%" :show-header="false">
                 <el-table-column>
                    <template #default="scope">
                      <div class="todo-item">
                        <el-tag size="small">ID:{{scope.row.student_id}}</el-tag>
                        <span class="todo-content text-truncate">{{ scope.row.content }}</span>
                        <el-button size="small" type="primary" plain round @click="openGrade(scope.row)">批改</el-button>
                      </div>
                    </template>
                 </el-table-column>
              </el-table>
              <el-empty v-if="todoData.homeworks.length === 0" description="暂无作业" :image-size="60" />
           </el-card>

           <el-card class="work-card" shadow="hover">
              <template #header><div class="card-header"><span>💬 待回复提问</span></div></template>
              <el-table :data="todoData.questions" height="250" style="width: 100%" :show-header="false">
                 <el-table-column>
                    <template #default="scope">
                       <div class="todo-item">
                          <div class="user-cell">
                            <el-avatar :size="24" style="background:#67C23A">{{ scope.row.student?.username?.charAt(0) }}</el-avatar>
                            <span class="username">{{ scope.row.student?.username }}</span>
                          </div>
                          <span class="todo-content text-truncate">{{ scope.row.Content }}</span>
                          <el-button size="small" type="success" plain round @click="openReply(scope.row)">回复</el-button>
                       </div>
                    </template>
                 </el-table-column>
              </el-table>
              <el-empty v-if="todoData.questions.length === 0" description="暂无提问" :image-size="60" />
           </el-card>
        </div>

        <h3 class="sub-title">我的课程列表</h3>
        <el-table :data="teacherCourses" v-loading="loading" class="custom-table" border>
          <el-table-column prop="title" label="课程标题" min-width="200" />
          <el-table-column label="状态" width="120" align="center">
            <template #default="scope">
              <el-tag v-if="scope.row.status === 1" type="success" effect="dark">已发布</el-tag>
              <el-tag v-else-if="scope.row.status === 2" type="danger" effect="dark">已驳回</el-tag>
              <el-tag v-else type="warning" effect="dark">审核中</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="view_count" label="浏览" width="100" align="center" sortable />
          <el-table-column label="操作" width="180" align="center">
            <template #default="scope">
              <el-button link type="primary" icon="View" @click="goToDetail(scope.row.ID)">详情</el-button>
              <el-button link type="primary" icon="Edit" @click="openEditDialog(scope.row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-else-if="userRole === 'admin'" class="admin-dashboard">
        <div class="dashboard-header-flex">
           <div>
             <h2>🖥️ 平台数据监控</h2>
             <p style="color: #909399; margin: 5px 0 0;">系统核心指标实时概览</p>
           </div>
           <el-button type="primary" plain icon="Refresh" circle @click="fetchAdminStats"></el-button>
        </div>

        <el-row :gutter="20" class="stat-row">
          <el-col :span="6" v-for="(item, idx) in statCards" :key="idx">
            <el-card shadow="hover" class="admin-stat-card">
              <div class="stat-val" :style="{ color: item.color }">{{ item.value }}</div>
              <div class="stat-lbl">{{ item.label }}</div>
              <el-icon class="stat-bg-icon" :color="item.color"><component :is="item.icon" /></el-icon>
            </el-card>
          </el-col>
        </el-row>

        <el-card class="audit-card" shadow="never">
          <template #header>
            <div class="card-header-flex">
              <span class="card-title">📝 待审核课程列表</span>
              <el-tag type="warning" round>{{ adminStats.pending_count }} 待处理</el-tag>
            </div>
          </template>
          
          <el-table :data="adminStats.pending_list" style="width: 100%" stripe v-loading="loading">
            <el-table-column label="提交教师" width="180">
               <template #default="scope">
                 <div style="display:flex; align-items:center; gap:8px;">
                   <el-avatar :size="28" style="background:#409EFF">{{ scope.row.teacher?.username?.charAt(0).toUpperCase() }}</el-avatar>
                   <span style="font-weight:500">{{ scope.row.teacher?.username }}</span>
                 </div>
               </template>
            </el-table-column>
            <el-table-column prop="title" label="课程标题" />
            <el-table-column label="分类" width="120">
               <template #default="scope">
                  <el-tag effect="light">{{ getCategoryName(scope.row.category) }}</el-tag>
               </template>
            </el-table-column>
            <el-table-column prop="price" label="价格" width="100">
               <template #default="scope">¥{{ scope.row.price }}</template>
            </el-table-column>
            <el-table-column prop="created_at" label="提交时间" width="180">
               <template #default="scope">{{ new Date(scope.row.CreatedAt).toLocaleString() }}</template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="scope">
                 <el-button type="success" size="small" icon="Check" circle @click="auditCourse(scope.row.ID, 1)"></el-button>
                 <el-button type="danger" size="small" icon="Close" circle @click="auditCourse(scope.row.ID, 2)"></el-button>
                 <el-button type="primary" link size="small" @click="goToDetail(scope.row.ID)">查看</el-button>
              </template>
            </el-table-column>
          </el-table>
          
          <el-empty v-if="!adminStats.pending_list || adminStats.pending_list.length === 0" description="太棒了，所有课程都已审核完毕！" />
        </el-card>
      </div>

    </div>

    <el-dialog v-model="showCreateDialog" title="👩‍🏫 发布新课程" width="600px" align-center>
      <el-form :model="newCourse" label-width="80px">
        <el-form-item label="课程标题"><el-input v-model="newCourse.title" placeholder="例如：Vue3 高级实战" /></el-form-item>
        <el-form-item label="课程分类">
          <el-select v-model="newCourse.category" placeholder="请选择分类" style="width: 100%">
            <el-option label="前端开发" value="frontend" /><el-option label="后端架构" value="backend" />
            <el-option label="人工智能" value="ai" /><el-option label="运维/测试" value="ops" />
          </el-select>
        </el-form-item>
        <el-form-item label="课程简介"><el-input v-model="newCourse.description" type="textarea" rows="3" /></el-form-item>
        <el-form-item label="课程大纲">
          <div style="width: 100%">
            <div v-for="(item, index) in outlineList" :key="index" style="display: flex; gap: 10px; margin-bottom: 10px;">
              <el-input v-model="item.title" placeholder="章节标题" style="flex: 1" />
              <el-input v-model="item.desc" placeholder="简述" style="flex: 1" />
              <el-button type="danger" :icon="Delete" circle @click="removeChapter(index)" v-if="outlineList.length > 1"/>
            </div>
            <el-button type="primary" link size="small" @click="addChapter"><el-icon><Plus /></el-icon> 添加章节</el-button>
          </div>
        </el-form-item>
        <el-form-item label="作业要求"><el-input v-model="newCourse.homework_req" type="textarea" rows="3" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="newCourse.price" :min="0" :precision="2" controls-position="right" /></el-form-item>
        <el-form-item label="视频上传">
          <el-upload class="upload-demo" action="#" :http-request="uploadFile" :limit="1" accept=".mp4,.avi" :show-file-list="false">
            <el-button :type="newCourse.video_url ? 'success' : 'primary'" :icon="VideoPlay">
              {{ newCourse.video_url ? '视频已上传' : '点击上传视频' }}
            </el-button>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="showCreateDialog = false">取消</el-button><el-button type="primary" @click="createCourse" :loading="isSubmitting">立即发布</el-button></template>
    </el-dialog>

    <el-dialog v-model="showEditDialog" title="🛠 修改课程信息" width="600px" align-center>
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="课程标题"><el-input v-model="editForm.title" /></el-form-item>
        <el-form-item label="分类">
          <el-select v-model="editForm.category" style="width: 100%">
            <el-option label="前端开发" value="frontend" /><el-option label="后端架构" value="backend" />
            <el-option label="人工智能" value="ai" /><el-option label="运维/测试" value="ops" />
          </el-select>
        </el-form-item>
        <el-form-item label="简介"><el-input v-model="editForm.description" type="textarea" rows="2" /></el-form-item>
        <el-form-item label="课程大纲">
          <div style="width: 100%">
            <div v-for="(item, index) in outlineList" :key="index" style="display: flex; gap: 10px; margin-bottom: 10px;">
              <el-input v-model="item.title" style="flex: 1" />
              <el-input v-model="item.desc" style="flex: 1" />
              <el-button type="danger" :icon="Delete" circle @click="removeChapter(index)" v-if="outlineList.length > 1"/>
            </div>
            <el-button type="primary" link size="small" @click="addChapter"><el-icon><Plus /></el-icon> 添加章节</el-button>
          </div>
        </el-form-item>
        <el-form-item label="作业要求"><el-input v-model="editForm.homework_req" type="textarea" rows="3" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="editForm.price" :min="0" :precision="2" controls-position="right"/></el-form-item>
      </el-form>
      <template #footer><el-button @click="showEditDialog = false">取消</el-button><el-button type="primary" @click="submitEdit" :loading="isSubmitting">保存修改</el-button></template>
    </el-dialog>

    <el-dialog v-model="showGradeDialog" title="✍️ 批改作业" width="400px" align-center>
        <el-form :model="gradeForm" label-position="top">
            <el-form-item label="评分 (0-100)"><el-input-number v-model="gradeForm.score" :min="0" :max="100" style="width:100%" /></el-form-item>
            <el-form-item label="老师评语"><el-input v-model="gradeForm.comment" type="textarea" rows="3" placeholder="给学生一些鼓励或建议..." /></el-form-item>
        </el-form>
        <template #footer><el-button @click="showGradeDialog = false">取消</el-button><el-button type="primary" @click="submitGrade">确认提交</el-button></template>
    </el-dialog>

    <el-dialog v-model="showReplyDialog" title="🗣 回复学生" width="400px" align-center>
        <el-form :model="replyForm">
            <el-form-item><el-input v-model="replyForm.answer" type="textarea" rows="5" placeholder="请输入您的详细解答..." /></el-form-item>
        </el-form>
        <template #footer><el-button @click="showReplyDialog = false">取消</el-button><el-button type="primary" @click="submitReply">发送回复</el-button></template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { 
  Plus, User, SwitchButton, VideoPlay, ElementPlus, View, Edit, 
  Delete, Refresh, CaretBottom, HotWater, Reading, EditPen, ChatDotSquare, Check, Close 
} from '@element-plus/icons-vue'

const router = useRouter()

// 状态
const userRole = ref(sessionStorage.getItem('role') || 'student')
const username = ref(sessionStorage.getItem('username') || '学员')
const userId = ref(parseInt(sessionStorage.getItem('user_id') || 0))
const userAvatar = ref('') // 可扩展
const courseList = ref([])
const hotCourses = ref([]) 
const loading = ref(false)
const activeCategory = ref('all') 
const isSubmitting = ref(false)
let dashboardTimer = null 

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showGradeDialog = ref(false)
const showReplyDialog = ref(false)

const todoData = ref({ homeworks: [], questions: [] })
const gradeForm = ref({ id: 0, score: 90, comment: '做得不错，继续加油！' })
const replyForm = ref({ id: 0, answer: '' })

const adminStats = ref({ user_count: 0, course_count: 0, view_count: 0, pending_count: 0, pending_list: [] })

const outlineList = ref([{ title: '第一章', desc: '' }])
const newCourse = ref({ title: '', description: '', price: 0, video_url: '', category: '', teacher_id: 0, homework_req: '', outline: '' })
const editForm = ref({ ID: 0, title: '', description: '', category: '', price: 0, homework_req: '', outline: '' })

// Computed
const roleName = computed(() => {
  if (userRole.value === 'teacher') return '金牌讲师'
  if (userRole.value === 'admin') return '超级管理员'
  return 'VIP学员'
})

const teacherCourses = computed(() => {
  if (userRole.value !== 'teacher') return []
  return courseList.value.filter(c => c.teacher_id === userId.value)
})

const statCards = computed(() => [
  { label: '注册用户', value: adminStats.value.user_count, icon: 'User', color: '#409EFF' },
  { label: '总课程数', value: adminStats.value.course_count, icon: 'Reading', color: '#67C23A' },
  { label: '平台流量', value: adminStats.value.view_count || 0, icon: 'HotWater', color: '#F56C6C' },
  { label: '待审核', value: adminStats.value.pending_count, icon: 'Timer', color: '#E6A23C' },
])

const getCategoryName = (key) => {
  const map = { 'frontend': '前端开发', 'backend': '后端架构', 'ai': '人工智能', 'ops': '运维测试' }
  return map[key] || '综合课程'
}

// Logic
const addChapter = () => outlineList.value.push({ title: '', desc: '' })
const removeChapter = (index) => outlineList.value.splice(index, 1)

const fetchCourses = async () => {
  if (userRole.value === 'admin') return
  loading.value = true
  try {
    const params = {}
    if (userRole.value !== 'teacher' && activeCategory.value !== 'all') params.category = activeCategory.value
    const res = await request.get('/courses', { params })
    courseList.value = res.data
  } catch (e) {} finally { loading.value = false }
}

const fetchHotCourses = async () => {
  if (userRole.value !== 'student') return
  try {
    const res = await request.get('/courses', { params: { sort: 'hot' } })
    hotCourses.value = res.data
  } catch (e) {}
}

const fetchTeacherDashboard = async () => {
  if (userRole.value !== 'teacher') return
  try {
    const res = await request.get('/teacher/dashboard')
    todoData.value = res 
  } catch(e) {}
}

const fetchAdminStats = async () => {
  if (userRole.value !== 'admin') return
  loading.value = true
  try {
    const res = await request.get('/admin/stats')
    adminStats.value = res
  } catch (e) {} finally { loading.value = false }
}

const auditCourse = async (id, status) => {
  try {
    await request.put('/admin/audit', { id, status })
    ElMessage.success(status === 1 ? '已通过' : '已驳回')
    fetchAdminStats()
  } catch(e) {}
}

const handleCategoryChange = () => fetchCourses()
const goToDetail = (id) => router.push(`/course/${id}`)
const logout = () => { sessionStorage.clear(); router.push('/login') }

const uploadFile = async (param) => {
  const formData = new FormData()
  formData.append('file', param.file)
  try {
    const res = await request.post('/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    newCourse.value.video_url = res.url
    ElMessage.success('上传成功')
  } catch (e) { ElMessage.error('上传失败') }
}

const createCourse = async () => {
  if (!newCourse.value.title || !newCourse.value.video_url || !newCourse.value.category) return ElMessage.warning('信息不完整')
  isSubmitting.value = true
  try {
    newCourse.value.teacher_id = userId.value
    newCourse.value.outline = JSON.stringify(outlineList.value)
    await request.post('/courses', newCourse.value)
    ElMessage.success('发布成功，请等待审核')
    showCreateDialog.value = false
    newCourse.value = { title: '', description: '', price: 0, video_url: '', category: '', homework_req: '', outline: '' }
    outlineList.value = [{ title: '第一章', desc: '' }]
    fetchCourses() 
  } catch (e) {} finally { isSubmitting.value = false }
}

const openEditDialog = (item) => {
  editForm.value = { ...item }
  try { outlineList.value = item.outline ? JSON.parse(item.outline) : [{ title: '第一章', desc: '' }] } 
  catch(e) { outlineList.value = [{ title: '第一章', desc: '' }] }
  showEditDialog.value = true
}

const submitEdit = async () => {
  isSubmitting.value = true
  try {
    await request.put(`/courses/${editForm.value.ID}`, { ...editForm.value, outline: JSON.stringify(outlineList.value) })
    ElMessage.success('修改成功')
    showEditDialog.value = false
    fetchCourses() 
  } catch (e) {} finally { isSubmitting.value = false }
}

const openGrade = (hw) => { gradeForm.value = { id: hw.ID, score: 90, comment: '' }; showGradeDialog.value = true }
const submitGrade = async () => {
  await request.put('/homework/grade', gradeForm.value)
  ElMessage.success('批改完成'); showGradeDialog.value = false; fetchTeacherDashboard() 
}
const openReply = (q) => { replyForm.value = { id: q.ID, answer: '' }; showReplyDialog.value = true }
const submitReply = async () => {
  await request.put('/questions/reply', replyForm.value)
  ElMessage.success('回复成功'); showReplyDialog.value = false; fetchTeacherDashboard() 
}

onMounted(() => {
  if (userRole.value === 'student') { fetchCourses(); fetchHotCourses() } 
  else if (userRole.value === 'teacher') { fetchCourses(); fetchTeacherDashboard(); dashboardTimer = setInterval(fetchTeacherDashboard, 5000) } 
  else if (userRole.value === 'admin') { fetchAdminStats() }
})
onUnmounted(() => { if (dashboardTimer) clearInterval(dashboardTimer) })
</script>

<style scoped>
/* 全局布局 */
.home-container { min-height: 100vh; background-color: #f6f8fc; }
.header { 
  background: #fff; box-shadow: 0 4px 12px rgba(0,0,0,0.03); 
  display: flex; justify-content: space-between; align-items: center; 
  padding: 0 40px; height: 64px; position: sticky; top: 0; z-index: 100;
}
.logo-area { display: flex; align-items: center; gap: 12px; cursor: pointer; }
.logo-bg { width: 36px; height: 36px; background: linear-gradient(135deg, #409EFF, #337ecc); border-radius: 8px; display: flex; align-items: center; justify-content: center; }
.logo-text { font-size: 22px; font-weight: 800; color: #2c3e50; letter-spacing: -0.5px; }
.user-area { display: flex; align-items: center; gap: 12px; }
.user-link { cursor: pointer; display: flex; align-items: center; font-size: 14px; color: #606266; transition: 0.2s; }
.user-link:hover { color: #409EFF; }
.main-content { max-width: 1280px; margin: 30px auto; padding: 0 20px; }

/* 学生端 */
.welcome-banner { margin-bottom: 30px; }
.welcome-banner h2 { margin: 0; font-size: 28px; color: #2c3e50; }
.welcome-banner p { color: #909399; margin: 8px 0 0; }
.section-title { font-size: 20px; display: flex; align-items: center; gap: 8px; margin-bottom: 20px; }
.hot-carousel { margin-bottom: 40px; }
.hot-card { position: relative; height: 100%; border-radius: 12px; overflow: hidden; cursor: pointer; }
.hot-img { width: 100%; height: 100%; object-fit: cover; transition: 0.5s; }
.hot-overlay { position: absolute; inset: 0; background: linear-gradient(to top, rgba(0,0,0,0.8) 0%, transparent 60%); display: flex; align-items: flex-end; padding: 20px; }
.hot-info h3 { color: #fff; margin: 0 0 8px; font-size: 22px; text-shadow: 0 2px 4px rgba(0,0,0,0.3); }
.hot-meta { display: flex; justify-content: space-between; align-items: center; color: rgba(255,255,255,0.8); width: 100%; }
.hot-card:hover .hot-img { transform: scale(1.08); }

.toolbar-sticky { position: sticky; top: 70px; z-index: 10; background: #f6f8fc; padding: 10px 0; margin-bottom: 20px; }
.custom-tabs :deep(.el-tabs__nav-wrap::after) { height: 0; }
.course-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 30px; }
.course-card { border-radius: 16px; border: none; transition: all 0.3s ease; }
.course-card:hover { transform: translateY(-8px); box-shadow: 0 12px 24px rgba(0,0,0,0.08); }
.image-wrapper { position: relative; height: 170px; overflow: hidden; }
.course-cover { width: 100%; height: 100%; object-fit: cover; }
.category-badge { position: absolute; top: 12px; right: 12px; background: rgba(0,0,0,0.65); backdrop-filter: blur(4px); color: #fff; padding: 4px 10px; border-radius: 6px; font-size: 12px; }
.card-content { padding: 16px; }
.course-title { margin: 0 0 8px; font-size: 16px; color: #2c3e50; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.course-desc { color: #909399; font-size: 13px; line-height: 1.5; margin-bottom: 16px; height: 40px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.card-bottom { display: flex; justify-content: space-between; align-items: center; }
.price-tag { color: #ff6b6b; font-weight: bold; font-size: 18px; }
.price-tag.free { color: #67C23A; }
.views { font-size: 12px; color: #c0c4cc; display: flex; align-items: center; gap: 4px; }

/* 教师端 & 管理员端 */
.dashboard-header-flex { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.stats-row { display: flex; gap: 20px; margin-bottom: 30px; }
.stat-card { flex: 1; border: none; border-radius: 12px; }
.stat-card :deep(.el-card__body) { display: flex; align-items: center; padding: 25px; gap: 20px; }
.stat-icon { width: 56px; height: 56px; border-radius: 12px; display: flex; align-items: center; justify-content: center; font-size: 26px; color: #fff; }
.bg-blue { background: linear-gradient(135deg, #409EFF, #ecf5ff); color: #409EFF; }
.bg-orange { background: linear-gradient(135deg, #E6A23C, #fdf6ec); color: #E6A23C; }
.bg-green { background: linear-gradient(135deg, #67C23A, #f0f9eb); color: #67C23A; }
.stat-info .label { color: #909399; font-size: 14px; }
.stat-info .num { font-size: 24px; font-weight: bold; color: #303133; margin-top: 5px; }

.workbench-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; }
.work-card { border-radius: 12px; border: none; }
.card-header { font-weight: bold; font-size: 16px; border-left: 4px solid #409EFF; padding-left: 10px; }
.todo-item { display: flex; align-items: center; justify-content: space-between; padding: 12px 0; border-bottom: 1px dashed #ebeef5; }
.todo-content { flex: 1; margin: 0 15px; color: #606266; font-size: 14px; }
.user-cell { display: flex; align-items: center; gap: 6px; width: 100px; }
.username { font-size: 13px; font-weight: 500; }
.text-truncate { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* 管理员特有 */
.admin-stat-card { text-align: center; position: relative; border-radius: 12px; overflow: hidden; height: 120px; display: flex; flex-direction: column; justify-content: center; }
.stat-val { font-size: 32px; font-weight: 800; line-height: 1; margin-bottom: 8px; }
.stat-lbl { color: #909399; font-size: 13px; }
.stat-bg-icon { position: absolute; right: -10px; bottom: -10px; opacity: 0.1; font-size: 80px; transform: rotate(-15deg); }
.card-header-flex { display: flex; justify-content: space-between; align-items: center; }
.add-btn { box-shadow: 0 4px 12px rgba(64,158,255,0.4); }
</style>