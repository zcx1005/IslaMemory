<template>
  <div class="page">
    <h2>视频列表</h2>

    <div v-if="loading">加载中...</div>
    <div v-else-if="error">错误：{{ error }}</div>
    <div v-else-if="list.length === 0">暂无视频</div>

    <ul v-else class="video-list">
      <li v-for="v in list" :key="v.public_id" class="card">
        <h3>{{ v.title }}</h3>
        <p>分类：{{ v.category_name }}</p>
        <p>简介：{{ v.description || '-' }}</p>
        <p>播放：{{ v.play_count }}</p>
        <router-link :to="`/video/${v.public_id}`">去播放</router-link>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getVideoList } from '@/api/video'
import type { VideoListItem } from '@/types/video'

const loading = ref(true)
const error = ref('')
const list = ref<VideoListItem[]>([])

onMounted(async () => {
  try {
    const res = await getVideoList({ page: 1, page_size: 20, sort: 'latest' })
    if (res.code !== 200) throw new Error(res.msg || '获取列表失败')
    list.value = res.data.list || []
  } catch (e: any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page { padding: 16px; }
.video-list { list-style: none; padding: 0; }
.card {
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}
</style>