<template>
  <div class="page">
    <div v-if="loading">加载中...</div>
    <div v-else-if="error">错误：{{ error }}</div>

    <div v-else-if="video">
      <h2>{{ video.title }}</h2>
      <p>{{ video.description || '暂无简介' }}</p>

      <video
          class="player"
          controls
          :src="playUrl"
          @play="onFirstPlay"
      />

      <div class="meta">
        <p>分类：{{ video.category_name }}</p>
        <p>分辨率：{{ video.width }} x {{ video.height }}</p>
        <p>时长：{{ video.duration_seconds }}s</p>
        <p>播放量：{{ video.play_count }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getVideoDetail, postVideoPlay, toPlayableUrl } from '@/api/video'
import type { VideoDetail } from '@/types/video'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const video = ref<VideoDetail | null>(null)
const played = ref(false)

const playUrl = computed(() => toPlayableUrl(video.value?.playback_url || ''))

onMounted(async () => {
  try {
    const publicId = String(route.params.publicId || '')
    const res = await getVideoDetail(publicId)
    if (res.code !== 200) throw new Error(res.msg || '获取详情失败')
    video.value = res.data
  } catch (e: any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
})

async function onFirstPlay() {
  if (played.value || !video.value) return
  played.value = true
  try {
    await postVideoPlay(video.value.public_id)
  } catch {
    // 上报失败不影响播放体验
  }
}
</script>

<style scoped>
.page { padding: 16px; }
.player { width: 720px; max-width: 100%; background: #000; }
.meta { margin-top: 12px; }
</style>