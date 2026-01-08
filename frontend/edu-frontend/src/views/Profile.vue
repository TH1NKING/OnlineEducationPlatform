<template>
  <div class="profile-container">
    <div class="profile-main">
      <div class="sidebar">
        <el-card shadow="hover" class="user-card">
           <div class="avatar-wrapper">
             <el-avatar :size="100" :src="userInfo.avatar || 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'" />
             <div class="edit-mask" @click="openEditDialog"><el-icon><Edit /></el-icon></div>
           </div>
           <h2 class="username">{{ userInfo.username }}</h2>
           <el-tag :type="roleTagType" effect="dark" round class="role-badge">{{ roleName }}</el-tag>
           <p class="bio">{{ userInfo.bio || '暂无个人介绍' }}</p>
           
           <div class="divider"></div>
           <div class="nav-item active"><el-icon><User /></el-icon> 个人资料</div>
           <div class="nav-item" @click="$router.push('/')"><el-icon><HomeFilled /></el-icon> 返回首页</div>
        </el-card>
      </div>

      <div class="content">
         <el-card shadow="never" class="content-card">
            <template #header>
               <div class="card-header">
                 <span>📚 我的学习进度</span>
                 <el-tag type="info">共 {{ myCourses.length }} 门课程</el-tag>
               </div>
            </template>
            
            <div v-if="userInfo.role === 'student'">
               <el-table :data="myCourses" style="width: 100%" :show-header="true">
                 <el-table-column label="课程信息" width="300">
                    <template #default="scope">
                       <div class="course-info">
                          <img :src="scope.row.course.cover_image || 'https://via.placeholder.com/80'" class="thumb" />
                          <span class="title">{{ scope.row.course.title }}</span>
                       </div>
                    </template>
                 </el-table-column>
                 <el-table-column label="进度" align="center">
                    <template #default="scope">
                       <el-progress :percentage="scope.row.progress || 0" :status="scope.row.progress >= 100 ? 'success' : ''"/>
                    </template>
                 </el-table-column>
                 <el-table-column label="操作" width="120" align="right">
                    <template #default="scope">
                       <el-button type="primary" round size="small" @click="$router.push(`/course/${scope.row.course.ID}`)">去学习</el-button>
                    </template>
                 </el-table-column>
               </el-table>
               <el-empty v-if="myCourses.length === 0" description="还没加入任何课程，去首页逛逛吧" />
            </div>
            <div v-else class="teacher-empty">
               <el-empty description="讲师/管理员暂无学习记录" />
            </div>
         </el-card>
      </div>
    </div>

    <el-dialog v-model="showEditDialog" title="编辑资料" width="450px" align-center>
      <el-form :model="editForm" label-position="top">
        <el-form-item label="头像上传">
           <el-upload class="avatar-uploader" action="#" :http-request="uploadAvatar" :show-file-list="false">
             <img v-if="editForm.avatar" :src="editForm.avatar" class="avatar" />
             <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
           </el-upload>
        </el-form-item>
        <el-form-item label="用户名"><el-input v-model="editForm.username" /></el-form-item>
        <el-form-item label="个人简介"><el-input v-model="editForm.bio" type="textarea" :rows="3" maxlength="100" show-word-limit /></el-form-item>
        <el-form-item label="修改密码 (留空则不改)"><el-input v-model="editForm.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showEditDialog=false">取消</el-button><el-button type="primary" @click="submitEdit">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import request from '../utils/request'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Edit, Plus, User, HomeFilled } from '@element-plus/icons-vue'

const router = useRouter()
const userId = sessionStorage.getItem('user_id')
const userInfo = ref({ username: '', role: '', avatar: '', bio: '' })
const myCourses = ref([])
const showEditDialog = ref(false)
const editForm = ref({ username: '', password: '', avatar: '', bio: '' })

const roleName = computed(() => { return { admin: '管理员', teacher: '讲师', student: '学生' }[userInfo.value.role] || '用户' })
const roleTagType = computed(() => { return { admin: 'danger', teacher: 'warning', student: 'success' }[userInfo.value.role] || 'info' })

const initData = async () => {
  try {
    userInfo.value = await request.get('/user/profile')
    if (userInfo.value.role === 'student') myCourses.value = (await request.get('/my-courses')).data
  } catch (e) {}
}

const openEditDialog = () => {
  editForm.value = { username: userInfo.value.username, password: '', avatar: userInfo.value.avatar, bio: userInfo.value.bio }
  showEditDialog.value = true
}

const uploadAvatar = async (param) => {
  const formData = new FormData(); formData.append('file', param.file)
  try { editForm.value.avatar = (await request.post('/upload', formData)).url; ElMessage.success('上传成功') } catch (e) { ElMessage.error('失败') }
}

const submitEdit = async () => {
  try {
    await request.put('/user/profile', editForm.value)
    ElMessage.success('保存成功，请重新登录')
    sessionStorage.clear(); router.push('/login')
  } catch (e) {}
}

onMounted(initData)
</script>

<style scoped>
.profile-container { min-height: 100vh; background: #f6f8fc; padding: 40px 20px; display: flex; justify-content: center; }
.profile-main { display: flex; gap: 30px; max-width: 1000px; width: 100%; align-items: flex-start; }
.sidebar { width: 300px; flex-shrink: 0; }
.content { flex: 1; }

.user-card { text-align: center; padding: 20px; border-radius: 12px; border: none; }
.avatar-wrapper { position: relative; display: inline-block; margin-bottom: 15px; }
.edit-mask { position: absolute; inset: 0; background: rgba(0,0,0,0.5); border-radius: 50%; display: flex; align-items: center; justify-content: center; color: #fff; opacity: 0; transition: 0.3s; cursor: pointer; }
.avatar-wrapper:hover .edit-mask { opacity: 1; }
.username { margin: 0 0 5px; font-size: 22px; color: #333; }
.bio { color: #909399; font-size: 14px; margin: 15px 0 20px; line-height: 1.5; }
.divider { height: 1px; background: #eee; margin: 20px 0; }
.nav-item { padding: 12px; border-radius: 8px; cursor: pointer; display: flex; align-items: center; gap: 10px; color: #606266; margin-bottom: 5px; }
.nav-item:hover { background: #f5f7fa; color: #409EFF; }
.nav-item.active { background: #ecf5ff; color: #409EFF; font-weight: bold; }

.content-card { border-radius: 12px; border: none; min-height: 500px; }
.card-header { display: flex; justify-content: space-between; align-items: center; font-size: 18px; font-weight: bold; }

.course-info { display: flex; align-items: center; gap: 12px; }
.thumb { width: 60px; height: 40px; border-radius: 4px; object-fit: cover; }
.title { font-weight: 500; color: #303133; }

.avatar-uploader { width: 100px; height: 100px; border: 1px dashed #d9d9d9; border-radius: 6px; display: flex; align-items: center; justify-content: center; cursor: pointer; overflow: hidden; }
.avatar { width: 100%; height: 100%; object-fit: cover; }
.avatar-uploader-icon { font-size: 28px; color: #8c939d; }
</style>