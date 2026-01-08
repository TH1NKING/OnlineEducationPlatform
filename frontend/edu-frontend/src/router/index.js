import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import CourseDetail from '../views/CourseDetail.vue' // 新增
import Profile from '../views/Profile.vue' // 新增

const routes = [
    { path: '/login', component: Login },
    { path: '/', component: Home },
    { path: '/course/:id', component: CourseDetail }, // 动态路由
    { path: '/profile', component: Profile }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('token')
    if (to.path !== '/login' && !token) {
        next('/login')
    } else {
        next()
    }
})

export default router