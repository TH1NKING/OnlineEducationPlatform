import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import CourseDetail from '../views/CourseDetail.vue'
import Profile from '../views/Profile.vue'
import AdminDashboard from '../views/AdminDashboard.vue' // 新增引入

const routes = [
    { path: '/login', component: Login },
    { path: '/', component: Home },
    { path: '/course/:id', component: CourseDetail },
    { path: '/profile', component: Profile },
    { path: '/admin', component: AdminDashboard } // 新增路由
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('token')
    const role = localStorage.getItem('role')
    
    if (to.path !== '/login' && !token) {
        next('/login')
    } else if (to.path === '/admin' && role !== 'admin') {
        // 防止非管理员访问后台
        next('/') 
    } else {
        next()
    }
})

export default router