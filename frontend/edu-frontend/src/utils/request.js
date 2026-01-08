import axios from 'axios'
import { ElMessage } from 'element-plus'

// 指向你的 Go 本地后端
const request = axios.create({
    baseURL: 'http://localhost:8080/api/v1', 
    timeout: 5000
})

// 请求拦截器：自动添加 Token
request.interceptors.request.use(config => {
    const token = localStorage.getItem('token')
    if (token) {
        config.headers['Authorization'] = `Bearer ${token}`
    }
    return config
}, error => {
    return Promise.reject(error)
})

// 响应拦截器：处理错误
request.interceptors.response.use(response => {
    return response.data
}, error => {
    // 如果是 401 说明 Token 过期或未登录
    if (error.response && error.response.status === 401) {
        ElMessage.error('登录已过期，请重新登录')
        localStorage.removeItem('token')
        window.location.href = '/login'
    } else {
        ElMessage.error(error.response?.data?.error || '网络错误')
    }
    return Promise.reject(error)
})

export default request