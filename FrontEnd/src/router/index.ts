import { createRouter, createWebHistory } from 'vue-router'
import VideoListView from '@/views/VideoListView.vue'
import VideoDetailView from '@/views/VideoDetailView.vue'
import ProfileView from '@/views/ProfileView.vue'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        { path: '/', component: VideoListView },
        { path: '/video/:publicId', component: VideoDetailView },
        { path: '/profile', component: ProfileView },
    ],
})

export default router