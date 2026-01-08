<template>
  <div class="detail-container">
    
    <div class="course-hero" v-if="course">
      <div class="hero-content">
        <el-button @click="$router.push('/')" icon="ArrowLeft" round class="back-btn">返回</el-button>
        <div class="hero-info">
          <el-tag effect="dark" size="small" class="cat-tag">{{ course.category }}</el-tag>
          <h1 class="hero-title">{{ course.title }}</h1>
          <div class="hero-meta">
            <span v-if="course.teacher">👨‍🏫 讲师：{{ course.teacher.username }}</span>
            <span>🔥 热度：{{ course.view_count }}</span>
          </div>
          
          <div class="hero-action" v-if="userRole === 'student'">
            <el-button v-if="!isEnrolled" type="primary" size="large" round @click="handleEnroll" class="enroll-btn">
               立即加入学习 ({{ course.price > 0 ? '¥'+course.price : '免费' }})
            </el-button>
            <div v-else class="progress-box">
               <span class="prog-text">学习进度</span>
               <el-progress :percentage="currentProgress" :stroke-width="10" color="#67C23A" style="width: 200px"/>
            </div>
          </div>
        </div>
      </div>
      <div class="hero-mask"></div>
      <img :src="course.cover_image || 'https://picsum.photos/1200/400'" class="hero-bg" />
    </div>

    <div class="content-wrapper" v-if="course">
      <div class="main-column">
        <div class="video-section" v-if="userRole === 'student'">
          <div v-if="isEnrolled" class="player-wrapper">
             <video :src="course.video_url" controls class="custom-video" @ended="onVideoEnded"></video>
          </div>
          <div v-else class="lock-wrapper">
            <div class="lock-content">
              <el-icon size="48"><Lock /></el-icon>
              <h3>付费/报名后观看视频</h3>
              <p>解锁完整课程内容、作业批改及导师答疑服务</p>
            </div>
          </div>
          <div v-if="isEnrolled && progressDetails.video_done" class="task-done-tip">
             <el-icon><CircleCheckFilled /></el-icon> 视频任务已完成
          </div>
        </div>

        <el-tabs v-model="activeTab" class="detail-tabs">
          <el-tab-pane label="📖 课程简介" name="intro">
            <div class="text-content">{{ course.description }}</div>
            <div class="teacher-card">
              <el-avatar :size="50" :src="course.teacher?.avatar" style="background:#409EFF">{{ course.teacher?.username?.charAt(0) }}</el-avatar>
              <div>
                <div class="t-name">{{ course.teacher?.username }}</div>
                <div class="t-bio">{{ course.teacher?.bio || '暂无介绍' }}</div>
              </div>
            </div>
          </el-tab-pane>
          
          <el-tab-pane label="📑 课程大纲" name="outline">
            <el-timeline style="padding-top: 10px">
              <el-timeline-item v-for="(ch, idx) in parsedOutline" :key="idx" :type="isChapterDone(idx) ? 'success' : 'primary'" :hollow="!isChapterDone(idx)">
                <div class="chapter-item">
                   <div>
                     <h4 :class="{done: isChapterDone(idx)}">第{{idx+1}}章：{{ ch.title }}</h4>
                     <p>{{ ch.desc }}</p>
                   </div>
                   <el-button v-if="isEnrolled && userRole==='student' && !isChapterDone(idx)" size="small" @click="markChapterDone(idx)">打卡</el-button>
                   <el-tag v-else-if="isChapterDone(idx)" type="success" size="small">已完成</el-tag>
                </div>
              </el-timeline-item>
            </el-timeline>
          </el-tab-pane>

          <el-tab-pane label="📝 课后作业" name="homework" v-if="isEnrolled && userRole === 'student'">
             <div v-if="homeworkData.exists" class="homework-result">
                <div class="result-header" :class="homeworkData.data.score > 0 ? 'scored' : 'waiting'">
                   <el-icon size="24"><component :is="homeworkData.data.score > 0 ? 'CircleCheck' : 'Timer'" /></el-icon>
                   <span>{{ homeworkData.data.score > 0 ? `得分：${homeworkData.data.score}分` : '作业已提交，等待老师批改' }}</span>
                </div>
                <div class="hw-content">
                  <p class="label">我的答案：</p>
                  <div class="answer-box">{{ homeworkData.data.content }}</div>
                  <div v-if="homeworkData.data.comment" class="teacher-comment">
                    <p class="label">👩‍🏫 老师评语：</p>
                    <div class="comment-box">{{ homeworkData.data.comment }}</div>
                  </div>
                </div>
             </div>
             <div v-else class="homework-form">
                <div class="req-box">
                  <p class="label">📌 作业要求：</p>
                  <div>{{ course.homework_req || '老师暂未发布具体要求，请自由发挥。' }}</div>
                </div>
                <el-input v-model="homeworkContent" type="textarea" rows="6" placeholder="请在此输入你的作业内容..." resize="none" />
                <el-button type="primary" size="large" class="submit-hw-btn" @click="submitHomework">提交作业</el-button>
             </div>
          </el-tab-pane>

          <el-tab-pane label="💬 课程问答" name="qa">
             <div class="qa-container">
               <div v-if="isEnrolled && userRole === 'student'" class="ask-bar">
                  <el-input v-model="newQuestion" placeholder="有疑问？向老师提问..." >
                    <template #append><el-button type="primary" @click="submitQuestion">发送</el-button></template>
                  </el-input>
               </div>
               
               <div class="chat-list">
                  <el-empty v-if="questionList.length === 0" description="还没有人提问，快来抢沙发" />
                  <div v-for="q in questionList" :key="q.ID" class="chat-group">
                    <div class="chat-bubble student">
                       <el-avatar :size="36" style="background:#409EFF">{{ q.student?.username?.charAt(0) }}</el-avatar>
                       <div class="bubble-content">
                          <div class="bubble-meta">{{ q.student?.username }} <span class="time">{{ new Date(q.CreatedAt).toLocaleDateString() }}</span></div>
                          <div class="bubble-text">{{ q.Content }}</div>
                       </div>
                    </div>
                    <div v-if="q.is_answered" class="chat-bubble teacher">
                       <div class="bubble-content">
                          <div class="bubble-meta right"><span class="badge">讲师</span> 回复</div>
                          <div class="bubble-text reply">{{ q.answer }}</div>
                       </div>
                       <el-avatar :size="36" :src="course.teacher?.avatar" style="background:#E6A23C">师</el-avatar>
                    </div>
                  </div>
               </div>
             </div>
          </el-tab-pane>
        </el-tabs>
      </div>

      <div class="side-column">
        <el-card shadow="hover" class="teacher-side-card">
           <h3>关于讲师</h3>
           <div class="t-center">
             <el-avatar :size="70" :src="course.teacher?.avatar" style="background:#409EFF">{{ course.teacher?.username?.charAt(0) }}</el-avatar>
             <h4>{{ course.teacher?.username }}</h4>
             <p>{{ course.teacher?.bio || '专注技术分享' }}</p>
             <el-button v-if="userRole==='student'" plain round size="small" @click="activeTab='qa'">向TA提问</el-button>
           </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import request from '../utils/request'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Lock, CircleCheckFilled, CircleCheck, Timer } from '@element-plus/icons-vue'

const route = useRoute()
const course = ref(null)
const isEnrolled = ref(false)
const userRole = ref(sessionStorage.getItem('role') || 'student')
const activeTab = ref('intro')
const currentProgress = ref(0)
const progressDetails = ref({ video_done: false, chapters: [] })

const homeworkContent = ref(''); const homeworkData = ref({ exists: false })
const questionList = ref([]); const newQuestion = ref('')
let qaTimer = null

const parsedOutline = computed(() => {
  if (!course.value || !course.value.outline) return []
  try { return JSON.parse(course.value.outline) } catch (e) { return [] }
})

const isChapterDone = (idx) => progressDetails.value.chapters?.includes(idx)

const fetchDetail = async () => {
  try {
    const res = await request.get(`/courses/${route.params.id}`)
    course.value = res.course
    isEnrolled.value = res.is_enrolled
    if(isEnrolled.value && userRole.value === 'student') { fetchHomework(); fetchMyEnrollmentInfo() }
    fetchQuestions()
  } catch (e) {}
}

const fetchMyEnrollmentInfo = async () => {
    try {
        const res = await request.get('/my-courses')
        const enroll = res.data.find(e => e.course_id === course.value.ID)
        if (enroll) {
            currentProgress.value = enroll.progress
            if (enroll.details) progressDetails.value = JSON.parse(enroll.details)
        }
    } catch(e) {}
}

const updateProgress = async (type, index) => {
    try {
        const res = await request.post('/progress/update', { course_id: course.value.ID, type, index })
        currentProgress.value = res.progress
        progressDetails.value = res.details
    } catch (e) {}
}

const onVideoEnded = () => { if (!progressDetails.value.video_done) { updateProgress('video', 0); ElMessage.success('视频学习完成') } }
const markChapterDone = (idx) => { updateProgress('chapter', idx); ElMessage.success('章节打卡成功') }
const handleEnroll = async () => { await request.post('/enroll', { course_id: course.value.ID }); ElMessage.success('加入成功'); isEnrolled.value = true; fetchMyEnrollmentInfo() }

const fetchHomework = async () => { homeworkData.value = await request.get(`/homework?course_id=${course.value.ID}`) }
const submitHomework = async () => { if (!homeworkContent.value.trim()) return ElMessage.warning('内容不能为空'); await request.post('/homework', { course_id: course.value.ID, content: homeworkContent.value }); ElMessage.success('提交成功'); fetchHomework() }

const fetchQuestions = async () => { questionList.value = (await request.get(`/questions?course_id=${course.value.ID}`)).data }
const submitQuestion = async () => { if(!newQuestion.value.trim()) return; await request.post('/questions', { course_id: course.value.ID, content: newQuestion.value }); ElMessage.success('提问成功'); newQuestion.value = ''; fetchQuestions() }

onMounted(async () => { await fetchDetail(); qaTimer = setInterval(() => { if (activeTab.value === 'qa') fetchQuestions() }, 3000) })
onUnmounted(() => { if (qaTimer) clearInterval(qaTimer) })
</script>

<style scoped>
.detail-container { min-height: 100vh; background: #f5f7fa; padding-bottom: 40px; }
.course-hero { position: relative; height: 280px; display: flex; align-items: center; justify-content: center; color: #fff; overflow: hidden; }
.hero-bg { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; filter: blur(4px); z-index: 0; }
.hero-mask { position: absolute; inset: 0; background: rgba(0,0,0,0.6); z-index: 1; }
.hero-content { position: relative; z-index: 2; max-width: 1100px; width: 100%; padding: 0 20px; display: flex; align-items: flex-start; gap: 20px; flex-direction: column;}
.back-btn { background: rgba(255,255,255,0.2); border: none; color: #fff; }
.back-btn:hover { background: rgba(255,255,255,0.3); }
.hero-title { font-size: 32px; margin: 10px 0; text-shadow: 0 2px 4px rgba(0,0,0,0.5); }
.hero-meta { display: flex; gap: 20px; opacity: 0.9; margin-bottom: 20px; font-size: 14px; }
.cat-tag { background: #409EFF; border: none; }
.hero-action { display: flex; align-items: center; gap: 20px; }
.enroll-btn { box-shadow: 0 4px 15px rgba(64,158,255,0.5); font-weight: bold; width: 200px;}
.progress-box { display: flex; align-items: center; gap: 10px; background: rgba(0,0,0,0.4); padding: 5px 15px; border-radius: 20px; }
.prog-text { font-size: 12px; }

.content-wrapper { max-width: 1100px; margin: -40px auto 0; padding: 0 20px; display: flex; gap: 30px; position: relative; z-index: 10; }
.main-column { flex: 1; }
.side-column { width: 300px; }

.video-section { background: #000; border-radius: 12px; overflow: hidden; margin-bottom: 20px; box-shadow: 0 10px 30px rgba(0,0,0,0.2); }
.custom-video { width: 100%; max-height: 500px; display: block; }
.lock-wrapper { height: 350px; display: flex; align-items: center; justify-content: center; background: #1f1f1f; color: #fff; text-align: center; }
.lock-content h3 { margin: 10px 0 5px; }
.lock-content p { color: #888; font-size: 14px; }
.task-done-tip { background: #f0f9eb; color: #67C23A; padding: 10px; text-align: center; font-size: 14px; font-weight: bold; display: flex; align-items: center; justify-content: center; gap: 5px; }

.detail-tabs { background: #fff; border-radius: 12px; padding: 20px; min-height: 400px; box-shadow: 0 4px 12px rgba(0,0,0,0.05); }
.text-content { line-height: 1.8; color: #444; font-size: 15px; white-space: pre-wrap; margin-bottom: 30px; }
.teacher-card { display: flex; align-items: center; gap: 15px; background: #f8f9fa; padding: 15px; border-radius: 8px; }
.t-name { font-weight: bold; margin-bottom: 2px; }
.t-bio { font-size: 13px; color: #888; }

.chapter-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; }
.chapter-item h4 { margin: 0 0 4px; font-size: 16px; color: #303133; }
.chapter-item h4.done { color: #67C23A; text-decoration: line-through; }
.chapter-item p { margin: 0; font-size: 13px; color: #909399; }

.homework-result { border: 1px solid #ebeef5; border-radius: 8px; overflow: hidden; }
.result-header { padding: 15px; display: flex; align-items: center; gap: 10px; font-weight: bold; }
.result-header.scored { background: #f0f9eb; color: #67C23A; }
.result-header.waiting { background: #fdf6ec; color: #E6A23C; }
.hw-content { padding: 20px; }
.label { font-weight: bold; color: #303133; margin-bottom: 8px; }
.answer-box { background: #f8f9fa; padding: 15px; border-radius: 6px; color: #555; margin-bottom: 20px; }
.teacher-comment { border-top: 1px dashed #eee; padding-top: 15px; }
.comment-box { color: #E6A23C; }
.req-box { background: #fff8e6; border-left: 4px solid #faad14; padding: 15px; margin-bottom: 20px; }
.submit-hw-btn { margin-top: 15px; width: 100%; }

.chat-list { padding-top: 20px; }
.chat-group { margin-bottom: 25px; }
.chat-bubble { display: flex; gap: 15px; margin-bottom: 15px; }
.chat-bubble.student { flex-direction: row; }
.chat-bubble.teacher { flex-direction: row; justify-content: flex-end; }
.bubble-content { max-width: 70%; }
.bubble-meta { font-size: 12px; color: #999; margin-bottom: 4px; }
.bubble-meta.right { text-align: right; }
.badge { background: #E6A23C; color: #fff; padding: 1px 4px; border-radius: 2px; }
.bubble-text { background: #f4f4f5; padding: 10px 15px; border-radius: 0 12px 12px 12px; color: #333; line-height: 1.5; }
.chat-bubble.teacher .bubble-text { background: #ecf5ff; border-radius: 12px 0 12px 12px; color: #409EFF; text-align: right;}

.teacher-side-card { text-align: center; border-radius: 12px; border: none; }
.t-center { display: flex; flex-direction: column; align-items: center; gap: 10px; }
</style>