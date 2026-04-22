<template>
  <section class="profile-page">
    <h2>个人中心</h2>
    <div class="tabs">
      <button :class="{ active: tab === 'info' }" @click="setTab('info')">个人信息</button>
      <button :class="{ active: tab === 'favorites' }" @click="setTab('favorites')">收藏的视频</button>
      <button :class="{ active: tab === 'uploads' }" @click="setTab('uploads')">上传的视频</button>
    </div>

    <div v-if="tab === 'info'" class="panel">
      <label>用户名</label>
      <input v-model="profileForm.username" placeholder="用户名" />

      <label>上传头像图片</label>
      <input type="file" accept="image/*" @change="onAvatarSelect" />

      <div v-if="avatarSource" class="crop-section">
        <div class="crop-preview-wrap">
          <div
              class="crop-preview"
              :style="{
              backgroundImage: `url(${avatarSource})`,
              backgroundPosition: `${cropX}% ${cropY}%`,
              backgroundSize: `${cropScale}%`,
            }"
          />
          <p>头像预览</p>
        </div>
    </div>
        <div class="crop-controls">
          <label>缩放 {{ cropScale }}%</label>
          <input v-model.number="cropScale" type="range" min="0" max="300" step="1" />

          <label>横向位置 {{ cropX }}%</label>
          <input v-model.number="cropX" type="range" min="-100" max="100" step="1" />

          <label>纵向位置 {{ cropY }}%</label>
          <input v-model.number="cropY" type="range" min="-100" max="100" step="1" />
        </div>

      <button @click="saveProfile">保存修改</button>
      <p v-if="message">{{ message }}</p>
    </div>

    <div v-else class="panel">
      <div v-if="loading">加载中...</div>
      <div v-else-if="videos.length === 0">暂无视频</div>
      <div v-else class="video-grid">
        <article v-for="v in videos" :key="v.public_id" class="video-card">
          <img :src="v.cover_url" class="cover" @click="goDetail(v.public_id)" />
          <h4 @click="goDetail(v.public_id)">{{ v.title }}</h4>
          <p>{{ v.uploader_username || v.username || '未知用户' }} · {{ formatDate(v.created_at) }}</p>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getMyFavoriteVideos, getMyUploadedVideos, updateMyProfile } from '@/api/user'
import { useAuth } from '@/composables/userAuth'
import type { VideoListItem } from '@/types/video'

const route = useRoute()
const router = useRouter()
const { user, openAuthModal, updateUserProfile } = useAuth()

const tab = ref<'info' | 'favorites' | 'uploads'>('info')
const loading = ref(false)
const message = ref('')
const videos = ref<VideoListItem[]>([])
const profileForm = reactive({ username: '', avatar_url: '' })

const avatarSource = ref('')
const selectedAvatarFile = ref<File | null>(null)
const cropScale = ref(140)
const cropX = ref(50)
const cropY = ref(50)

const routeTab = computed(() => String(route.query.tab || 'info'))

onMounted(() => {
  if (!user.value) {
    openAuthModal('login')
    router.replace('/')
    return
  }
  profileForm.username = user.value.username || ''
  profileForm.avatar_url = user.value.avatar_url || ''
  avatarSource.value = user.value.avatar_url || ''
  syncTabAndLoad(routeTab.value)
})

watch(routeTab, (next) => {
  syncTabAndLoad(next)
})

async function syncTabAndLoad(next: string) {
  const safeTab = next === 'favorites' || next === 'uploads' ? next : 'info'
  tab.value = safeTab
  if (safeTab === 'info') return
  loading.value = true
  try {
    const res = safeTab === 'favorites' ? await getMyFavoriteVideos() : await getMyUploadedVideos()
    videos.value = res.code === 200 ? res.data?.list || [] : []
  } finally {
    loading.value = false
  }
}

function setTab(next: 'info' | 'favorites' | 'uploads') {
  router.replace({ query: { ...route.query, tab: next } })
}

function onAvatarSelect(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  selectedAvatarFile.value = file
  const reader = new FileReader()
  reader.onload = () => {
    avatarSource.value = String(reader.result || '')
    cropScale.value = 140
    cropX.value = 50
    cropY.value = 50
  }
  reader.readAsDataURL(file)
}

async function buildCroppedAvatarFile(): Promise<File | null> {
  if (!avatarSource.value) return selectedAvatarFile.value

  const canvas = document.createElement('canvas')
  const size = 256
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext('2d')
  if (!ctx) return selectedAvatarFile.value

  const img = new Image()
  const loaded = new Promise<void>((resolve, reject) => {
    img.onload = () => resolve()
    img.onerror = () => reject(new Error('头像读取失败'))
  })
  img.src = avatarSource.value
  await loaded

  const scale = cropScale.value / 100
  const w = img.width * scale
  const h = img.height * scale
  const x = ((100 - cropX.value) / 100) * (w - size)
  const y = ((100 - cropY.value) / 100) * (h - size)

  ctx.clearRect(0, 0, size, size)
  ctx.drawImage(img, -x, -y, w, h)

  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob((result) => resolve(result), 'image/png', 0.92)
  })
  if (!blob) return selectedAvatarFile.value

  return new File([blob], `avatar-${Date.now()}.png`, { type: 'image/png' })
}

async function saveProfile() {
  message.value = ''
  const payload: { username?: string; avatar_file?: File } = {}
  const name = profileForm.username.trim()
  if (name) payload.username = name
  if (selectedAvatarFile.value || avatarSource.value) {
    const cropped = await buildCroppedAvatarFile()
    if (cropped) payload.avatar_file = cropped
  }

  const res = await updateMyProfile(payload)
  if (res.code === 200) {
    profileForm.avatar_url = res.data?.avatar_url || profileForm.avatar_url
    selectedAvatarFile.value = null
    avatarSource.value = profileForm.avatar_url
    updateUserProfile({ username: profileForm.username.trim(), avatar_url: profileForm.avatar_url })
    message.value = '保存成功'
    return
  }
  message.value = res.msg || '保存失败'
}

function goDetail(publicId: string) {
  router.push(`/video/${publicId}`)
}

function formatDate(date: string) {
  if (!date) return '-'
  return date.slice(0, 10)
}
</script>

<style scoped>
.profile-page { padding: 16px 0; }
.tabs { display: flex; gap: 8px; margin-bottom: 16px; }
.tabs button { border: 1px solid #ddd; background: #fff; border-radius: 8px; padding: 6px 12px; cursor: pointer; color: #111; }
.tabs button.active { background: #e3e3e3; border-color: #d0d0d0; color: #111; }
.panel { border: 1px solid #eee; border-radius: 12px; padding: 16px; display: flex; flex-direction: column; gap: 8px; text-align: left; color: #111; }
.panel input { min-height: 34px; border: 1px solid #ddd; border-radius: 8px; padding: 0 10px; color: #111; }
.panel button { width: 120px; height: 34px; border: 1px solid #d0d0d0; border-radius: 8px; background: #efefef; color: #111; }
.crop-section { display: flex; gap: 16px; align-items: flex-start; margin: 8px 0; }
.crop-preview-wrap { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.crop-preview { width: 140px; height: 140px; border-radius: 50%; border: 1px solid #ddd; background-repeat: no-repeat; background-color: #f5f5f5; }
.crop-controls { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.crop-controls input { width: 30%; }
.video-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.video-card { border: 1px solid #eee; border-radius: 8px; padding: 10px; }
.cover { width: 100%; aspect-ratio: 16/9; object-fit: cover; border-radius: 8px; cursor: pointer; }
.video-card h4 { margin: 8px 0 6px; cursor: pointer; color: #111; }
.video-card p { color: #666; font-size: 13px; }
</style>