<template>
  <div class="detail-container">
    <el-button @click="$router.push('/')" icon="ArrowLeft">返回首页</el-button>
    
    <div v-if="course" class="content-box">
      <div class="header">
        <h2>{{ course.title }}</h2>
        
        <div v-if="userRole === 'student'">
          <el-tag type="success" v-if="isEnrolled">已加入学习</el-tag>
          <el-button type="primary" v-else @click="handleEnroll">加入课程 (免费)</el-button>
        </div>
        
        <div v-else-if="userRole === 'teacher'">
          <el-tag type="info">教师预览模式</el-tag>
        </div>
      </div>

      <div class="video-player" v-if="userRole === 'student'">
        <video 
          v-if="isEnrolled" 
          :src="course.video_url" 
          controls 
          style="width: 100%; max-height: 500px; background: #000;"
        ></video>
        <div v-else class="lock-mask">
          <el-icon size="50"><Lock /></el-icon>
          <p>请先加入课程后观看视频</p>
        </div>
      </div>

      <el-tabs v-model="activeTab" type="border-card" style="margin-top: 20px;">
        <el-tab-pane label="课程简介" name="intro">{{ course.description }}</el-tab-pane>
        
        <el-tab-pane label="课程大纲" name="outline">
          <el-empty v-if="parsedOutline.length === 0" description="暂无大纲" />
          <el-timeline v-else>
            <el-timeline-item 
              v-for="(chapter, index) in parsedOutline" 
              :key="index" 
              :timestamp="`第 ${index + 1} 章`" 
              placement="top"
            >
              <el-card>
                <h4>{{ chapter.title }}</h4>
                <p style="color: #666; font-size: 13px;" v-if="chapter.desc">{{ chapter.desc }}</p>
              </el-card>
            </el-timeline-item>
          </el-timeline>
        </el-tab-pane>

        <el-tab-pane label="课后作业" name="homework" v-if="isEnrolled && userRole === 'student'">
          <div v-if="homeworkData.exists">
             <el-result
                :icon="homeworkData.data.score > 0 ? 'success' : 'info'"
                :title="homeworkData.data.score > 0 ? '已批改' : '等待批改'"
                :sub-title="homeworkData.data.score > 0 ? `得分：${homeworkData.data.score} 分` : '老师正在努力批改中...'"
              >
              <template #extra>
                 <div style="text-align: left; background: #f4f4f5; padding: 15px; border-radius: 4px; width: 100%;">
                    <p><strong>我的答案：</strong> {{ homeworkData.data.content }}</p>
                    <div v-if="homeworkData.data.comment" style="margin-top: 10px; color: #E6A23C;">
                        <strong>👩‍🏫 老师点评：</strong> {{ homeworkData.data.comment }}
                    </div>
                 </div>
              </template>
             </el-result>
          </div>
          <div v-else>
            <el-alert 
              title="作业要求" 
              type="warning" 
              :closable="false" 
              show-icon 
              style="margin-bottom: 15px;"
            >
              <template #default>
                <div style="white-space: pre-wrap; margin-top: 5px; font-weight: bold;">
                  {{ course.homework_req || '老师暂未布置具体作业要求，请简述学习心得即可。' }}
                </div>
              </template>
            </el-alert>
            
            <el-input v-model="homeworkContent" type="textarea" rows="6" placeholder="在此输入你的作业内容..." />
            <div style="margin-top: 15px; text-align: right;">
              <el-button type="primary" @click="submitHomework" size="large"><el-icon><EditPen /></el-icon> 提交作业</el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="课程问答" name="qa">
           <div v-if="isEnrolled && userRole === 'student'" style="margin-bottom: 20px; display: flex; gap: 10px;">
              <el-input v-model="newQuestion" placeholder="这就这，有什么不懂的快问老师..." />
              <el-button type="primary" @click="submitQuestion">提问</el-button>
           </div>

           <div class="qa-list">
              <el-empty v-if="questionList.length === 0" description="暂无提问" />
              <el-card v-for="q in questionList" :key="q.ID" style="margin-bottom: 15px;" shadow="hover">
                 <div style="display: flex; align-items: flex-start; gap: 10px;">
                    <el-avatar :size="30" style="background: #409EFF">{{ q.student?.username?.charAt(0).toUpperCase() }}</el-avatar>
                    <div style="flex: 1;">
                        <div style="display: flex; justify-content: space-between; align-items: center;">
                            <div style="font-weight: bold; font-size: 14px; color: #333;">
                                {{ q.student?.username || '同学' }} 
                                <span style="font-weight: normal; color: #999; font-size: 12px; margin-left: 5px;">{{ new Date(q.CreatedAt).toLocaleString() }}</span>
                            </div>
                            <el-button 
                                v-if="userRole === 'teacher' && !q.is_answered" 
                                type="primary" link size="small" 
                                @click="openReply(q)"
                            >
                                我来回复
                            </el-button>
                        </div>
                        <p style="margin: 5px 0;">{{ q.Content }}</p>
                        <div v-if="q.is_answered" style="background: #f0f9eb; padding: 10px; border-radius: 4px; margin-top: 10px; border-left: 3px solid #67C23A;">
                            <div style="font-weight: bold; color: #67C23A; font-size: 13px;"><el-icon><ChatDotRound /></el-icon> 老师回复：</div>
                            <div style="font-size: 13px; margin-top: 3px;">{{ q.answer }}</div>
                        </div>
                    </div>
                 </div>
              </el-card>
           </div>
        </el-tab-pane>
      </el-tabs>
    </div>
    
    <el-dialog v-model="showReplyDialog" title="🗣 回复学生" width="400px">
       <el-form :model="replyForm">
          <el-form-item label="回复内容">
             <el-input v-model="replyForm.answer" type="textarea" rows="4" placeholder="请输入解答..." />
          </el-form-item>
       </el-form>
       <template #footer>
          <el-button @click="showReplyDialog = false">取消</el-button>
          <el-button type="primary" @click="submitReply">发送回复</el-button>
       </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import request from '../utils/request'
import { ElMessage } from 'element-plus'
import { Lock, ArrowLeft, EditPen, ChatDotRound } from '@element-plus/icons-vue'

const route = useRoute()
const course = ref(null)
const isEnrolled = ref(false)
const homeworkContent = ref('')
const homeworkData = ref({ exists: false })
const userRole = ref(localStorage.getItem('role') || 'student')
const activeTab = ref('intro')

// Q&A states
const questionList = ref([])
const newQuestion = ref('')
const showReplyDialog = ref(false)
const replyForm = ref({ id: 0, answer: '' })

// 解析大纲
const parsedOutline = computed(() => {
  if (!course.value || !course.value.outline) {
    return []
  }
  try {
    return JSON.parse(course.value.outline)
  } catch (e) {
    return []
  }
})

const fetchDetail = async () => {
  try {
    const res = await request.get(`/courses/${route.params.id}`)
    course.value = res.course
    isEnrolled.value = res.is_enrolled
    if(isEnrolled.value && userRole.value === 'student') fetchHomework()
    // 加载问答
    fetchQuestions()
  } catch (e) {
    console.error(e)
  }
}

const handleEnroll = async () => {
  try {
    await request.post('/enroll', { course_id: course.value.ID })
    ElMessage.success('加入成功！')
    isEnrolled.value = true
    fetchHomework() 
  } catch(e) {}
}

const fetchHomework = async () => {
  const res = await request.get(`/homework?course_id=${course.value.ID}`)
  homeworkData.value = res
}

const submitHomework = async () => {
  if (!homeworkContent.value.trim()) return ElMessage.warning('请填写作业内容')
  try {
    await request.post('/homework', { 
      course_id: course.value.ID,
      content: homeworkContent.value 
    })
    ElMessage.success('提交成功')
    fetchHomework()
  } catch(e) {}
}

// --- Q&A Logic ---
const fetchQuestions = async () => {
  const res = await request.get(`/questions?course_id=${course.value.ID}`)
  questionList.value = res.data
}

const submitQuestion = async () => {
  if(!newQuestion.value.trim()) return ElMessage.warning('请输入问题内容')
  try {
    await request.post('/questions', {
      course_id: course.value.ID,
      content: newQuestion.value
    })
    ElMessage.success('提问成功')
    newQuestion.value = ''
    fetchQuestions()
  } catch(e) {}
}

const openReply = (q) => {
  replyForm.value = { id: q.ID, answer: '' }
  showReplyDialog.value = true
}

const submitReply = async () => {
  try {
    await request.put('/questions/reply', replyForm.value)
    ElMessage.success('回复成功')
    showReplyDialog.value = false
    fetchQuestions()
  } catch (e) {}
}

onMounted(fetchDetail)
</script>

<style scoped>
.detail-container { padding: 20px; max-width: 1000px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;}
.lock-mask { height: 300px; background: #333; color: #fff; display: flex; flex-direction: column; justify-content: center; align-items: center; }
</style>