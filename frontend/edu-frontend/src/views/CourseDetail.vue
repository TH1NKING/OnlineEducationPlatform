<template>
  <div class="detail-container">
    <el-button @click="$router.push('/')" icon="ArrowLeft">返回首页</el-button>
    
    <div v-if="course" class="content-box">
      <div class="header">
        <div style="flex: 1">
           <h2>{{ course.title }}</h2>
           <el-progress 
              v-if="isEnrolled" 
              :percentage="currentProgress" 
              :format="formatProgress"
              :status="currentProgress >= 100 ? 'success' : ''"
              style="width: 300px; margin-top: 10px;"
            />
        </div>
        
        <div v-if="userRole === 'student'">
          <el-tag type="success" v-if="isEnrolled">已加入学习</el-tag>
          <el-button type="primary" v-else @click="handleEnroll">加入课程 (免费)</el-button>
        </div>
      </div>

      <div class="video-player" v-if="userRole === 'student'">
        <div v-if="isEnrolled">
           <video 
             :src="course.video_url" 
             controls 
             style="width: 100%; max-height: 500px; background: #000;"
             @ended="onVideoEnded"
           ></video>
           <div v-if="progressDetails.video_done" style="color: #67C23A; margin-top: 5px;">
              <el-icon><CircleCheck /></el-icon> 视频任务已完成 (获得50%进度)
           </div>
        </div>
        <div v-else class="lock-mask">
          <el-icon size="50"><Lock /></el-icon>
          <p>请先加入课程后观看视频</p>
        </div>
      </div>

      <el-tabs v-model="activeTab" type="border-card" style="margin-top: 20px;">
        <el-tab-pane label="课程简介" name="intro">{{ course.description }}</el-tab-pane>
        
        <el-tab-pane label="课程大纲 (进度打卡)" name="outline">
          <el-empty v-if="parsedOutline.length === 0" description="暂无大纲" />
          <el-timeline v-else>
            <el-timeline-item 
              v-for="(chapter, index) in parsedOutline" 
              :key="index" 
              :timestamp="`第 ${index + 1} 章`" 
              placement="top"
              :type="isChapterDone(index) ? 'success' : 'primary'"
            >
              <el-card class="chapter-card">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <div>
                        <h4>{{ chapter.title }}</h4>
                        <p style="color: #666; font-size: 13px;">{{ chapter.desc }}</p>
                    </div>
                    
                    <div v-if="isEnrolled && userRole === 'student'">
                        <el-button 
                          v-if="!isChapterDone(index)" 
                          size="small" 
                          @click="markChapterDone(index)"
                        >
                          标记已学
                        </el-button>
                        <el-tag type="success" v-else><el-icon><Select /></el-icon> 已学完</el-tag>
                    </div>
                </div>
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
            <el-alert title="作业要求" type="warning" :closable="false" show-icon style="margin-bottom: 15px;">
              <template #default>
                <div style="white-space: pre-wrap; margin-top: 5px; font-weight: bold;">{{ course.homework_req || '暂无具体要求' }}</div>
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
                        <div style="font-weight: bold; font-size: 14px; color: #333;">
                            {{ q.student?.username || '同学' }} 
                            <span style="font-weight: normal; color: #999; font-size: 12px; margin-left: 5px;">{{ new Date(q.CreatedAt).toLocaleString() }}</span>
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import request from '../utils/request'
import { ElMessage } from 'element-plus'
import { Lock, ArrowLeft, EditPen, ChatDotRound, Select, CircleCheck } from '@element-plus/icons-vue'

const route = useRoute()
const course = ref(null)
const isEnrolled = ref(false)
const userRole = ref(localStorage.getItem('role') || 'student')
const activeTab = ref('intro')

// 进度相关
const currentProgress = ref(0)
const progressDetails = ref({ video_done: false, chapters: [] })

// 作业与问答相关
const homeworkContent = ref('')
const homeworkData = ref({ exists: false })
const questionList = ref([])
const newQuestion = ref('')

// 解析大纲
const parsedOutline = computed(() => {
  if (!course.value || !course.value.outline) return []
  try { return JSON.parse(course.value.outline) } catch (e) { return [] }
})

// 进度格式化显示
const formatProgress = (percentage) => percentage === 100 ? '已完成' : `${percentage.toFixed(1)}%`

// 判断某章节是否已学
const isChapterDone = (index) => {
    return progressDetails.value.chapters && progressDetails.value.chapters.includes(index)
}

// 核心：获取详情与进度
const fetchDetail = async () => {
  try {
    const res = await request.get(`/courses/${route.params.id}`)
    course.value = res.course
    isEnrolled.value = res.is_enrolled
    
    if(isEnrolled.value && userRole.value === 'student') {
        fetchHomework()
        // 获取当前详细进度 (这里我们复用 MyCourses 接口或者通过 enroll 接口获取，
        // 为了简便，这里我们调用后端获取进度的逻辑，但因为后端 detail 接口还没加 detail 返回
        // 建议：我们在 get my-courses 时获取，或者在 enroll 信息里加。
        // 为方便起见，我们直接发起一次空更新或重新获取
        fetchMyEnrollmentInfo() 
    }
    fetchQuestions()
  } catch (e) { console.error(e) }
}

// 获取选课的具体信息（用于初始化进度条）
const fetchMyEnrollmentInfo = async () => {
    try {
        const res = await request.get('/my-courses')
        // 找到当前课程的记录
        const enroll = res.data.find(e => e.course_id === course.value.ID)
        if (enroll) {
            currentProgress.value = enroll.progress
            if (enroll.details) {
                progressDetails.value = JSON.parse(enroll.details)
            }
        }
    } catch(e) {}
}

// 视频看完事件
const onVideoEnded = async () => {
    if (progressDetails.value.video_done) return // 已完成就不重复提交
    await updateProgress('video', 0)
    ElMessage.success('视频观看完成，进度已更新！')
}

// 标记章节完成
const markChapterDone = async (index) => {
    await updateProgress('chapter', index)
    ElMessage.success(`第 ${index+1} 章已标记为完成`)
}

// 统一更新接口
const updateProgress = async (type, index) => {
    try {
        const res = await request.post('/progress/update', {
            course_id: course.value.ID,
            type: type,
            index: index
        })
        // 更新前端状态
        currentProgress.value = res.progress
        progressDetails.value = res.details
    } catch (e) {}
}

const handleEnroll = async () => {
  try {
    await request.post('/enroll', { course_id: course.value.ID })
    ElMessage.success('加入成功！')
    isEnrolled.value = true
    fetchMyEnrollmentInfo()
  } catch(e) {}
}

// ... (fetchHomework, submitHomework, fetchQuestions, submitQuestion 保持原有逻辑) ...
const fetchHomework = async () => {
  const res = await request.get(`/homework?course_id=${course.value.ID}`)
  homeworkData.value = res
}
const submitHomework = async () => {
  if (!homeworkContent.value.trim()) return ElMessage.warning('请填写作业内容')
  await request.post('/homework', { course_id: course.value.ID, content: homeworkContent.value })
  ElMessage.success('提交成功')
  fetchHomework()
}
const fetchQuestions = async () => {
  const res = await request.get(`/questions?course_id=${course.value.ID}`)
  questionList.value = res.data
}
const submitQuestion = async () => {
  if(!newQuestion.value.trim()) return ElMessage.warning('请输入问题内容')
  await request.post('/questions', { course_id: course.value.ID, content: newQuestion.value })
  ElMessage.success('提问成功')
  newQuestion.value = ''
  fetchQuestions()
}

onMounted(fetchDetail)
</script>

<style scoped>
.detail-container { padding: 20px; max-width: 1000px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px;}
.lock-mask { height: 300px; background: #333; color: #fff; display: flex; flex-direction: column; justify-content: center; align-items: center; }
.chapter-card { margin-bottom: 5px; }
</style>