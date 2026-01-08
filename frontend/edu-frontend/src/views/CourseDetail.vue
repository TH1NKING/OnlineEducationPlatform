<template>
  <div class="detail-container">
    <el-button @click="$router.push('/')" icon="ArrowLeft">返回首页</el-button>
    
    <div v-if="course" class="content-box">
      <div class="header">
        <h2>{{ course.title }}</h2>
        <el-tag type="success" v-if="isEnrolled">已加入学习</el-tag>
        <el-button type="primary" v-else @click="handleEnroll">加入课程 (免费)</el-button>
      </div>

      <div class="video-player">
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

      <el-tabs type="border-card" style="margin-top: 20px;">
        <el-tab-pane label="课程简介">{{ course.description }}</el-tab-pane>
        
        <el-tab-pane label="课程大纲">
          <el-timeline>
            <el-timeline-item timestamp="第1章" placement="top">
              <el-card><h4>基础入门与环境搭建</h4></el-card>
            </el-timeline-item>
            <el-timeline-item timestamp="第2章" placement="top">
              <el-card><h4>{{ course.title }} - 核心实战</h4></el-card>
            </el-timeline-item>
          </el-timeline>
        </el-tab-pane>

        <el-tab-pane label="课后作业" v-if="isEnrolled">
          <div v-if="homeworkData.exists">
            <el-alert title="作业已提交" type="success" :closable="false" show-icon />
            <p><strong>我的答案：</strong> {{ homeworkData.data.content }}</p>
            <p><strong>得分：</strong> 
              <span v-if="homeworkData.data.score > 0">{{ homeworkData.data.score }}</span>
              <span v-else>等待老师批改...</span>
            </p>
          </div>
          <div v-else>
            <p>请简述本课程的核心知识点：</p>
            <el-input v-model="homeworkContent" type="textarea" rows="4" />
            <div style="margin-top: 10px;">
              <el-button type="primary" @click="submitHomework">提交作业</el-button>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import request from '../utils/request'
import { ElMessage } from 'element-plus'

const route = useRoute()
const course = ref(null)
const isEnrolled = ref(false)
const homeworkContent = ref('')
const homeworkData = ref({ exists: false })

const fetchDetail = async () => {
  const res = await request.get(`/courses/${route.params.id}`)
  course.value = res.course
  isEnrolled.value = res.is_enrolled
  if(isEnrolled.value) fetchHomework()
}

const handleEnroll = async () => {
  try {
    await request.post('/enroll', { course_id: course.value.ID })
    ElMessage.success('加入成功！')
    isEnrolled.value = true
    fetchHomework() // 刷新作业状态
  } catch(e) {}
}

const fetchHomework = async () => {
  const res = await request.get(`/homework?course_id=${course.value.ID}`)
  homeworkData.value = res
}

const submitHomework = async () => {
  try {
    await request.post('/homework', { 
      course_id: course.value.ID,
      content: homeworkContent.value 
    })
    ElMessage.success('提交成功')
    fetchHomework()
  } catch(e) {}
}

onMounted(fetchDetail)
</script>

<style scoped>
.detail-container { padding: 20px; max-width: 1000px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;}
.lock-mask { height: 300px; background: #333; color: #fff; display: flex; flex-direction: column; justify-content: center; align-items: center; }
</style>