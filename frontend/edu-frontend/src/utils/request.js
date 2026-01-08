import axios from 'axios'
import { ElMessage } from 'element-plus'

const request = axios.create({
    baseURL: 'http://localhost:8080/api/v1', 
    timeout: 5000
})

// 请求拦截器：自动添加 Token
request.interceptors.request.use(config => {
    // 【修改】改为 sessionStorage
    const token = sessionStorage.getItem('token')
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
    // 401 说明 Token 过期或被顶号
    if (error.response && error.response.status === 401) {
        ElMessage.error(error.response.data.error || '登录已过期，请重新登录')
        // 【修改】清除 sessionStorage 并跳转
        sessionStorage.clear()
        window.location.href = '/login'
    } else {
        ElMessage.error(error.response?.data?.error || '网络错误')
    }
    return Promise.reject(error)
})

export default request