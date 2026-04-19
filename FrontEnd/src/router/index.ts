import { createRouter, createWebHistory } from 'vue-router'
import VideoListView from '@/views/VideoListView.vue'
import VideoDetailView from '@/views/VideoDetailView.vue'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        { path: '/', component: VideoListView },
        { path: '/video/:publicId', component: VideoDetailView }
    ]
})

export default router