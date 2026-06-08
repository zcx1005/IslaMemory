<template>
  <section class="page">
    <div class="toolbar">
      <button :class="{ active: sort === 'recommend' }" @click="setSort('recommend')">默认推荐</button>
      <button :class="{ active: sort === 'popular' }" @click="setSort('popular')">最热门</button>
      <button :class="{ active: sort === 'latest' }" @click="setSort('latest')">最新</button>
    </div>
    <div v-if="loading">加载中...</div>
    <div v-else-if="error">错误：{{ error }}</div>
    <div v-else-if="list.length === 0">暂无视频</div>

    <div v-else class="video-grid">
      <article v-for="v in list" :key="v.public_id" class="card">
        <div class="cover-wrap" @click="goVideo(v.public_id)">
          <img class="cover" :src="v.cover_url" alt="cover" />
          <span class="badge left">{{ formatPlayCount(v.play_count) }}</span>
          <span class="badge right">{{ formatDuration(v.duration_seconds) }}</span>
        </div>
        <h3 @click="goVideo(v.public_id)">{{ v.title }}</h3>
        <p>{{ v.uploader_username || v.username || '未知用户' }}</p>
        <p>{{ formatDate(v.created_at) }}</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getVideoList } from '@/api/video'
import type { VideoListItem } from '@/types/video'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const error = ref('')
const list = ref<VideoListItem[]>([])
const sort = ref<'recommend' | 'popular' | 'latest'>('recommend')

onMounted(load)
watch(
    () => route.fullPath,
    () => {
      if (route.path === '/') load()
    },
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await getVideoList({
      page: 1,
      page_size: 30,
      sort: sort.value,
      keyword: String(route.query.keyword || ''),
      category_slug: String(route.query.category_slug || ''),
    })
    if (res.code !== 200) throw new Error(res.msg || '获取列表失败')
    list.value = res.data.list || []
  } catch (e: any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
}

function setSort(next: 'recommend' | 'popular' | 'latest') {
  sort.value = next
  load()
}

function goVideo(publicId: string) {
  router.push(`/video/${publicId}`)
}

function formatDate(date: string) {
  return date ? date.slice(0, 10) : '-'
}

function formatDuration(seconds: number) {
  const mm = String(Math.floor(seconds / 60)).padStart(2, '0')
  const ss = String(seconds % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

function formatPlayCount(count: number) {
  return count >= 10000 ? `${(count / 10000).toFixed(1)}万` : String(count)
}
</script>

<style scoped>
.page { padding-top: 16px; }
.toolbar { display: flex; gap: 8px; margin: 0 0 14px; }
.toolbar button { border: 1px solid #ddd; border-radius: 999px; background: #fff; padding: 6px 14px; cursor: pointer; }
.toolbar button.active { background: #111; color: #fff; }
.video-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 14px; }
.card { text-align: left; }
.cover-wrap { position: relative; cursor: pointer; }
.cover { width: 100%; aspect-ratio: 16/9; border-radius: 8px; object-fit: cover; background: #f2f2f2; }
.badge { position: absolute; bottom: 8px; color: #fff; background: rgba(0, 0, 0, 0.55); border-radius: 10px; padding: 2px 8px; font-size: 12px; }
.badge.left { left: 8px; }
.badge.right { right: 8px; }
h3 { font-size: 15px; line-height: 1.4; margin: 8px 0 6px; cursor: pointer; }
p { font-size: 13px; color: #888; margin: 2px 0; }
</style>