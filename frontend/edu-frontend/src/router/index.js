import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import CourseDetail from '../views/CourseDetail.vue'
import Profile from '../views/Profile.vue'
import AdminDashboard from '../views/AdminDashboard.vue'

const routes = [
    { path: '/login', component: Login },
    { path: '/', component: Home },
    { path: '/course/:id', component: CourseDetail },
    { path: '/profile', component: Profile },
    { path: '/admin', component: AdminDashboard }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

router.beforeEach((to, from, next) => {
    // 【修改】改为 sessionStorage
    const token = sessionStorage.getItem('token')
    const role = sessionStorage.getItem('role')
    
    if (to.path !== '/login' && !token) {
        next('/login')
    } else if (to.path === '/admin' && role !== 'admin') {
        next('/') 
    } else {
        next()
    }
})

export default router