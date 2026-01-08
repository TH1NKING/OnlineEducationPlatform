<template>
  <div class="home-container">
    <el-header class="header">
      <div class="logo-area">
        <el-icon class="logo-icon" :size="30" color="#409EFF"><ElementPlus /></el-icon>
        <span class="logo-text">EduPlatform 在线教育</span>
      </div>
      
      <div class="user-area">
        <span class="welcome-text">你好, {{ username }} ({{ roleName }})</span>
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
      
      <div v-if="userRole !== 'teacher'">
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
          <el-tabs v-model="activeCategory" @tab-change="handleCategoryChange" class="category-tabs">
            <el-tab-pane label="全部课程" name="all"></el-tab-pane>
            <el-tab-pane label="前端开发" name="frontend"></el-tab-pane>
            <el-tab-pane label="后端架构" name="backend"></el-tab-pane>
            <el-tab-pane label="人工智能" name="ai"></el-tab-pane>
            <el-tab-pane label="运维/测试" name="ops"></el-tab-pane>
          </el-tabs>
        </div>

        <div class="course-list" v-loading="loading">
          <el-empty v-if="courseList.length === 0" description="该分类下暂无课程" />
          
          <el-card 
            v-for="item in courseList" 
            :key="item.ID" 
            class="course-card" 
            shadow="hover"
          >
            <div class="image-wrapper" @click="goToDetail(item.ID)">
              <img :src="item.cover_image || `https://picsum.photos/seed/${item.ID}/300/180`" class="course-cover"/>
              <div class="category-tag">{{ getCategoryName(item.category) }}</div>
            </div>
            
            <div class="card-body">
              <h4 class="course-title" :title="item.title" @click="goToDetail(item.ID)">{{ item.title }}</h4>
              <p class="course-desc">{{ item.description }}</p>
              
              <div class="card-footer">
                <div class="meta">
                   <span class="price" v-if="item.price > 0">¥ {{ item.price }}</span>
                   <el-tag type="success" size="small" v-else>免费</el-tag>
                   <span class="views"><el-icon><View /></el-icon> {{ item.view_count }}</span>
                </div>

                <div>
                  <el-button v-if="userRole === 'admin'" type="warning" link size="small" @click.stop="openEditDialog(item)">
                    <el-icon><Edit /></el-icon> 修改
                  </el-button>
                  <el-button v-else type="primary" link @click="goToDetail(item.ID)">详情 >></el-button>
                </div>
              </div>
            </div>
          </el-card>
        </div>
      </div>

      <div v-else class="teacher-dashboard">
        <div class="dashboard-header">
          <h2>🎓 我的教学管理</h2>
          <el-button type="primary" size="large" @click="showCreateDialog = true">
            <el-icon style="margin-right: 5px"><Plus /></el-icon> 发布新课程
          </el-button>
        </div>

        <el-alert title="欢迎回来，老师！这里仅显示您发布的课程。" type="info" show-icon style="margin-bottom: 20px;" />

        <div class="teacher-workbench" style="margin-bottom: 30px; display: flex; gap: 20px;">
           <el-card style="flex: 1;" shadow="hover">
              <template #header>
                <div class="card-header"><span>📝 待批改作业 ({{ todoData.homeworks.length }})</span></div>
              </template>
              <el-table :data="todoData.homeworks" height="200" style="width: 100%">
                 <el-table-column prop="student_id" label="学生ID" width="80" />
                 <el-table-column prop="content" label="作业内容" show-overflow-tooltip />
                 <el-table-column label="操作" width="80">
                    <template #default="scope">
                       <el-button link type="primary" @click="openGrade(scope.row)">批改</el-button>
                    </template>
                 </el-table-column>
              </el-table>
           </el-card>

           <el-card style="flex: 1;" shadow="hover">
              <template #header>
                <div class="card-header"><span>💬 待回复提问 ({{ todoData.questions.length }})</span></div>
              </template>
              <el-table :data="todoData.questions" height="200" style="width: 100%">
                 <el-table-column label="学生">
                    <template #default="scope">{{ scope.row.student?.username || scope.row.student_id }}</template>
                 </el-table-column>
                 <el-table-column prop="Content" label="问题" show-overflow-tooltip />
                 <el-table-column label="操作" width="80">
                    <template #default="scope">
                       <el-button link type="success" @click="openReply(scope.row)">回复</el-button>
                    </template>
                 </el-table-column>
              </el-table>
           </el-card>
        </div>

        <el-table :data="teacherCourses" v-loading="loading" border stripe style="width: 100%">
          <el-table-column prop="ID" label="ID" width="60" />
          <el-table-column label="封面" width="120">
            <template #default="scope">
              <img :src="scope.row.cover_image || `https://picsum.photos/seed/${scope.row.ID}/100/60`" style="width: 80px; height: 50px; object-fit: cover; border-radius: 4px;" />
            </template>
          </el-table-column>
          <el-table-column prop="title" label="课程标题" />
          <el-table-column label="分类" width="100">
            <template #default="scope"><el-tag>{{ getCategoryName(scope.row.category) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="view_count" label="浏览量" width="80" sortable />
          <el-table-column label="操作" width="150">
            <template #default="scope">
              <el-button link type="primary" @click="goToDetail(scope.row.ID)">详情</el-button>
              <el-button link type="warning" @click="openEditDialog(scope.row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="showCreateDialog" title="👩‍🏫 发布新课程" width="600px">
      <el-form :model="newCourse" label-width="80px">
        <el-form-item label="课程标题"><el-input v-model="newCourse.title" placeholder="例如：Vue3 高级实战" /></el-form-item>
        <el-form-item label="课程分类">
          <el-select v-model="newCourse.category" placeholder="请选择分类" style="width: 100%">
            <el-option label="前端开发" value="frontend" />
            <el-option label="后端架构" value="backend" />
            <el-option label="人工智能" value="ai" />
            <el-option label="运维/测试" value="ops" />
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
        <el-form-item label="作业布置">
           <el-input v-model="newCourse.homework_req" type="textarea" rows="3" placeholder="请输入作业要求..." />
        </el-form-item>
        
        <el-form-item label="课程价格">
          <el-input-number v-model="newCourse.price" :min="0" :precision="2" />
        </el-form-item>
        <el-form-item label="课程视频">
          <el-upload class="upload-demo" action="#" :http-request="uploadFile" :limit="1" accept=".mp4,.avi">
            <el-button type="primary"><el-icon><VideoPlay /></el-icon> 上传视频</el-button>
          </el-upload>
          <div v-if="newCourse.video_url" class="upload-success"><el-icon color="#67C23A"><CircleCheck /></el-icon> 成功</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" @click="createCourse" :loading="isSubmitting">发布</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="showEditDialog" title="🛠 修改课程信息" width="600px">
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
        <el-form-item label="作业布置">
           <el-input v-model="editForm.homework_req" type="textarea" rows="3" />
        </el-form-item>

        <el-form-item label="价格"><el-input-number v-model="editForm.price" :min="0" :precision="2"/></el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="submitEdit" :loading="isSubmitting">保存</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="showGradeDialog" title="✍️ 批改作业" width="400px">
        <el-form :model="gradeForm">
            <el-form-item label="评分"><el-input-number v-model="gradeForm.score" :min="0" :max="100" /></el-form-item>
            <el-form-item label="评语"><el-input v-model="gradeForm.comment" type="textarea" /></el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="showGradeDialog = false">取消</el-button>
            <el-button type="primary" @click="submitGrade">确认</el-button>
        </template>
    </el-dialog>

    <el-dialog v-model="showReplyDialog" title="🗣 回复学生" width="400px">
        <el-form :model="replyForm">
            <el-form-item label="内容"><el-input v-model="replyForm.answer" type="textarea" rows="4" /></el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="showReplyDialog = false">取消</el-button>
            <el-button type="primary" @click="submitReply">发送</el-button>
        </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, User, SwitchButton, VideoPlay, CircleCheck, ElementPlus, View, Edit, Delete } from '@element-plus/icons-vue'

const router = useRouter()

// --- 状态变量 ---
const userRole = ref(localStorage.getItem('role') || 'student')
const username = ref(localStorage.getItem('username') || '学员')
const userId = ref(parseInt(localStorage.getItem('user_id') || 0))
const courseList = ref([])
const hotCourses = ref([]) 
const loading = ref(false)
const activeCategory = ref('all') 
const isSubmitting = ref(false)

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showGradeDialog = ref(false)
const showReplyDialog = ref(false)

// Dashboard data
const todoData = ref({ homeworks: [], questions: [] })
const gradeForm = ref({ id: 0, score: 0, comment: '' })
const replyForm = ref({ id: 0, answer: '' })

// Course Forms
const outlineList = ref([{ title: '第一章', desc: '' }])
const newCourse = ref({
  title: '', description: '', price: 0, video_url: '', category: '', teacher_id: 0,
  homework_req: '', outline: ''
})
const editForm = ref({
  ID: 0, title: '', description: '', category: '', price: 0,
  homework_req: '', outline: ''
})

// --- Computed ---
const roleName = computed(() => {
  if (userRole.value === 'teacher') return '教师'
  if (userRole.value === 'admin') return '管理员'
  return '学生'
})

const teacherCourses = computed(() => {
  if (userRole.value !== 'teacher') return []
  return courseList.value.filter(c => c.teacher_id === userId.value)
})

const getCategoryName = (key) => {
  const map = { 'frontend': '前端', 'backend': '后端', 'ai': 'AI/算法', 'ops': '运维' }
  return map[key] || '综合'
}

// --- Logic ---
const addChapter = () => outlineList.value.push({ title: '', desc: '' })
const removeChapter = (index) => outlineList.value.splice(index, 1)

const fetchCourses = async () => {
  loading.value = true
  try {
    const params = {}
    if (userRole.value !== 'teacher' && activeCategory.value !== 'all') params.category = activeCategory.value
    const res = await request.get('/courses', { params })
    courseList.value = res.data
  } catch (e) {
  } finally { loading.value = false }
}

const fetchHotCourses = async () => {
  if (userRole.value === 'teacher') return
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

const handleCategoryChange = (tabName) => {
  activeCategory.value = tabName 
  fetchCourses()
}

const goToDetail = (id) => router.push(`/course/${id}`)

const logout = () => {
  localStorage.clear()
  router.push('/login')
}

// Upload
const uploadFile = async (param) => {
  const formData = new FormData()
  formData.append('file', param.file)
  try {
    const res = await request.post('/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    newCourse.value.video_url = res.url
    ElMessage.success('视频上传成功')
  } catch (e) { ElMessage.error('上传失败') }
}

// Create
const createCourse = async () => {
  if (!newCourse.value.title || !newCourse.value.video_url || !newCourse.value.category) return ElMessage.warning('信息不完整')
  isSubmitting.value = true
  try {
    newCourse.value.teacher_id = userId.value
    newCourse.value.outline = JSON.stringify(outlineList.value)
    await request.post('/courses', newCourse.value)
    ElMessage.success('发布成功')
    showCreateDialog.value = false
    newCourse.value = { title: '', description: '', price: 0, video_url: '', category: '', homework_req: '', outline: '' }
    outlineList.value = [{ title: '第一章', desc: '' }]
    fetchCourses() 
  } catch (e) { ElMessage.error('发布失败') } finally { isSubmitting.value = false }
}

// Edit
const openEditDialog = (item) => {
  editForm.value = { ...item }
  try {
    outlineList.value = item.outline ? JSON.parse(item.outline) : [{ title: '第一章', desc: '' }]
  } catch(e) { outlineList.value = [{ title: '第一章', desc: '' }] }
  showEditDialog.value = true
}

const submitEdit = async () => {
  isSubmitting.value = true
  try {
    await request.put(`/courses/${editForm.value.ID}`, {
      ...editForm.value,
      outline: JSON.stringify(outlineList.value)
    })
    ElMessage.success('修改成功')
    showEditDialog.value = false
    fetchCourses() 
  } catch (e) { ElMessage.error('修改失败') } finally { isSubmitting.value = false }
}

// Grade & Reply
const openGrade = (hw) => {
  gradeForm.value = { id: hw.ID, score: 80, comment: '做得不错！' }
  showGradeDialog.value = true
}
const submitGrade = async () => {
  await request.put('/homework/grade', gradeForm.value)
  ElMessage.success('批改完成')
  showGradeDialog.value = false
  fetchTeacherDashboard() 
}
const openReply = (q) => {
  replyForm.value = { id: q.ID, answer: '' }
  showReplyDialog.value = true
}
const submitReply = async () => {
  await request.put('/questions/reply', replyForm.value)
  ElMessage.success('回复成功')
  showReplyDialog.value = false
  fetchTeacherDashboard() 
}

onMounted(() => {
  fetchCourses()
  if (userRole.value === 'teacher') fetchTeacherDashboard()
  else fetchHotCourses()
})
</script>

<style scoped>
.home-container { min-height: 100vh; background-color: #f5f7fa; }
.header { background: #fff; box-shadow: 0 2px 8px rgba(0,0,0,0.05); display: flex; justify-content: space-between; align-items: center; padding: 0 40px; height: 60px; }
.logo-area { display: flex; align-items: center; gap: 10px; font-weight: bold; color: #303133; font-size: 20px;}
.user-area { display: flex; align-items: center; gap: 10px; }
.main-content { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
.section-title h3 { margin: 20px 0 15px; border-left: 5px solid #ff6b6b; padding-left: 15px; color: #303133;}
.hot-card { position: relative; height: 100%; cursor: pointer; border-radius: 8px; overflow: hidden;}
.hot-img { width: 100%; height: 100%; object-fit: cover; filter: brightness(0.8); transition: 0.3s;}
.hot-card:hover .hot-img { filter: brightness(1); transform: scale(1.05);}
.hot-info { position: absolute; bottom: 0; left: 0; right: 0; padding: 20px; background: linear-gradient(to top, rgba(0,0,0,0.8), transparent); color: white; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-top: 30px; margin-bottom: 20px; }
.course-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 25px; }
.course-card { border-radius: 8px; border: none; transition: 0.3s; cursor: pointer; }
.course-card:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.1); }
.image-wrapper { position: relative; height: 160px; overflow: hidden; }
.course-cover { width: 100%; height: 100%; object-fit: cover; }
.category-tag { position: absolute; top: 10px; right: 10px; background: rgba(0,0,0,0.6); color: #fff; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.card-body { padding: 15px; }
.course-title { font-size: 16px; margin: 0 0 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #303133;}
.course-desc { font-size: 13px; color: #909399; margin-bottom: 15px; height: 40px; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
.card-footer { display: flex; justify-content: space-between; align-items: center; border-top: 1px solid #ebeef5; padding-top: 10px; }
.meta { display: flex; gap: 10px; align-items: center; }
.price { color: #F56C6C; font-weight: bold; }
.views { font-size: 12px; color: #999; display: flex; align-items: center; gap: 2px; }
.teacher-dashboard { padding-top: 20px; }
.dashboard-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.upload-success { margin-top: 10px; font-size: 12px; color: #67C23A; display: flex; align-items: center; gap: 5px; }
</style>