<template>
  <div class="layout">
    <header class="top-header">
      <button class="logo-btn" @click="goHome">アイラMemory</button>

      <div class="search-wrap">
        <button class="search-icon-btn" @click="search" aria-label="搜索">🔍</button>
        <input
            v-model.trim="keyword"
            class="search-input"
            placeholder="搜索视频标题"
            @keyup.enter="search"
        />
      </div>

      <div class="account-wrap" @mouseenter="onAccountEnter" @mouseleave="menuOpen = false">
        <button class="account-btn" @click="onAuthButtonClick" >
          <template v-if="isLoggedIn && user">
            <img v-if="user.avatar_url" :src="user.avatar_url" class="avatar" alt="avatar" />
            <span v-else class="avatar-fallback">{{ (user.username || user.account || 'U').slice(0, 1).toUpperCase() }}</span>
          </template>
          <template v-else>登录 / 注册</template>
        </button>

        <div v-if="isLoggedIn && menuOpen" class="account-menu"  @mouseleave="menuOpen = false">
          <button @click="goProfile('info')">个人详情</button>
          <button @click="goProfile('favorites')">收藏视频页</button>
          <button @click="openUpload">上传视频</button>
          <button @click="onLogout">退出</button>
        </div>
      </div>
    </header>

    <nav class="category-nav">
      <button
          v-for="c in categories"
          :key="c.slug"
          :class="{ active: selectedCategory === c.slug }"
          @click="pickCategory(c.slug)"
      >
        {{ c.name }}
      </button>
    </nav>

    <router-view />

    <div v-if="showAuthModal" class="modal-mask" @click="closeAuthModal">
      <div class="modal-card" @click.stop>
        <div class="auth-tabs">
          <button :class="{ active: activeAuthTab === 'login' }" @click="activeAuthTab = 'login'">登录</button>
          <button :class="{ active: activeAuthTab === 'register' }" @click="activeAuthTab = 'register'">注册</button>
        </div>

        <form v-if="activeAuthTab === 'login'" class="modal-form" @submit.prevent="onLogin">
          <input v-model="loginForm.account" placeholder="账号" required />
          <input v-model="loginForm.password" type="password" placeholder="密码" required />
          <button type="submit" :disabled="authLoading">{{ authLoading ? '登录中...' : '登录' }}</button>
        </form>

        <form v-else class="modal-form" @submit.prevent="onRegister">
          <input v-model="registerForm.account" placeholder="账号" required />
          <input v-model="registerForm.username" placeholder="用户名" required />
          <input v-model="registerForm.password" type="password" placeholder="密码" required />
          <button type="submit" :disabled="authLoading">{{ authLoading ? '注册中...' : '注册' }}</button>
        </form>

        <p v-if="authMessage" class="modal-message">{{ authMessage }}</p>
      </div>
    </div>

    <div v-if="showUploadModal" class="modal-mask" @click="closeUpload">
      <div class="modal-card" @click.stop>
        <h3>上传视频</h3>
        <form class="modal-form" @submit.prevent="submitUpload">
          <input v-model="uploadForm.title" placeholder="视频标题" required />
          <textarea v-model="uploadForm.description" rows="3" placeholder="视频简介" />
          <select v-model="uploadForm.category_slug" required>
            <option disabled value="">选择视频分区</option>
            <option v-for="c in categories.filter((c) => c.slug !== '')" :key="c.slug" :value="c.slug">
              {{ c.name }}
            </option>
          </select>
          <input type="file" accept="video/*" @change="onFileChange" required />
          <button type="submit" :disabled="uploadLoading">{{ uploadLoading ? '上传中...' : '上传' }}</button>
        </form>
        <p v-if="uploadMessage" class="modal-message">{{ uploadMessage }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { uploadVideo } from '@/api/user'
import { useAuth } from '@/composables/userAuth'

const router = useRouter()
const route = useRoute()

const categories = [
  { name: '全部', slug: '' },
  { name: '动画', slug: 'animation' },
  { name: '音乐', slug: 'music' },
  { name: '游戏', slug: 'game' },
  { name: '影视', slug: 'film' },
  { name: '知识', slug: 'knowledge' },
  { name: '生活', slug: 'life' },
]

const keyword = ref('')
const selectedCategory = ref('')
const menuOpen = ref(false)
const showUploadModal = ref(false)
const uploadLoading = ref(false)
const uploadMessage = ref('')
const uploadFile = ref<File | null>(null)
const uploadForm = reactive({
  title: '',
  description: '',
  category_slug: '',
})

const loginForm = reactive({ account: '', password: '' })
const registerForm = reactive({ account: '', username: '', password: '' })
const authMessage = ref('')

const {
  user,
  isLoggedIn,
  authLoading,
  showAuthModal,
  activeAuthTab,
  register,
  login,
  logout,
  restoreSession,
  openAuthModal,
  closeAuthModal,
} = useAuth()

onMounted(async () => {
  await restoreSession()
  syncFiltersFromRoute()
})

watch(
    () => route.fullPath,
    () => {
      syncFiltersFromRoute()
    },
)

function syncFiltersFromRoute() {
  if (route.path !== '/') return
  keyword.value = String(route.query.keyword || '')
  selectedCategory.value = String(route.query.category_slug || '')
}

function goHome() {
  router.push('/')
}

function search() {
  if (route.path !== '/') {
    router.push({ path: '/', query: { keyword: keyword.value || undefined, category_slug: selectedCategory.value || undefined } })
    return
  }
  router.replace({ query: { ...route.query, keyword: keyword.value || undefined } })
}

function pickCategory(slug: string) {
  selectedCategory.value = slug
  if (route.path !== '/') {
    router.push({ path: '/', query: { keyword: keyword.value || undefined, category_slug: slug || undefined } })
    return
  }
  router.replace({ query: { ...route.query, category_slug: slug || undefined } })
}

function onAuthButtonClick() {
  if (!isLoggedIn.value) {
    openAuthModal('login')
    return
  }
  menuOpen.value = !menuOpen.value
}

function onAccountEnter() {
  if (isLoggedIn.value) {
    menuOpen.value = true
  }
}

function goProfile(tab: 'info' | 'favorites' | 'uploads') {
  menuOpen.value = false
  router.push({ path: '/profile', query: { tab } })
}

async function onLogin() {
  authMessage.value = ''
  const res = await login({ ...loginForm })
  if (res.code === 200) {
    closeAuthModal()
    return
  }
  authMessage.value = res.msg || '登录失败'
}

async function onRegister() {
  authMessage.value = ''
  const res = await register({ ...registerForm })
  if (res.code === 200) {
    activeAuthTab.value = 'login'
    loginForm.account = registerForm.account
    loginForm.password = ''
    authMessage.value = '注册成功，请登录'
    return
  }
  authMessage.value = res.msg || '注册失败'
}

function onLogout() {
  menuOpen.value = false
  logout()
}

function openUpload() {
  menuOpen.value = false
  if (!isLoggedIn.value) {
    openAuthModal('login')
    return
  }
  showUploadModal.value = true
  uploadMessage.value = ''
}

function closeUpload() {
  showUploadModal.value = false
}

function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  uploadFile.value = target.files?.[0] || null
}

async function submitUpload() {
  if (!uploadFile.value) {
    uploadMessage.value = '请选择视频文件'
    return
  }
  uploadLoading.value = true
  uploadMessage.value = ''
  try {
    const res = await uploadVideo({ ...uploadForm, file: uploadFile.value })
    if (res.code === 200) {
      uploadMessage.value = '上传成功'
      showUploadModal.value = false
      uploadForm.title = ''
      uploadForm.description = ''
      uploadForm.category_slug = ''
      uploadFile.value = null
    } else {
      uploadMessage.value = res.msg || '上传失败'
    }
  } catch (e: any) {
    uploadMessage.value = e?.message || '上传失败'
  } finally {
    uploadLoading.value = false
  }
}
</script>

<style scoped>
.layout {
  padding: 0 24px 24px;
  color: #222;
}

.top-header {
  display: grid;
  grid-template-columns: 220px 1fr 170px;
  align-items: center;
  gap: 16px;
  padding: 16px 0;
  height: 80px;
}

.logo-btn {
  height: 44px;
  width: 400px;
  border: none;
  background: transparent;
  color: #111;
  cursor: pointer;
  font-size: 30px;
  font-weight: 800;
  text-align: left;
  letter-spacing: 1px;
  text-shadow: 2px 2px 0 #d9d9d9;
}

.search-wrap {
  max-width: 640px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  align-items: center;
  border: 1px solid #d7d7d7;
  border-radius: 22px;
  background: #fff;
  overflow: hidden;
  position: relative;
}

.search-icon-btn {
  width: 46px;
  height: 40px;
  border: none;
  background: #f0f0f0;
  color: #222;
  cursor: pointer;
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
}

.search-icon-btn:hover {
  background: #e3e3e3;
}

.search-input {
  width: 100%;
  height: 40px;
  border: none;
  color: #111;
  background: #fff;
  padding: 0 12px;
}

.search-input:focus {
  outline: none;
}

.account-wrap {
  position: relative;
}

.account-btn {
  width: 100%;
  height: 40px;
  border: 1px solid #d0d0d0;
  background: #f3f3f3;
  color: #111;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.account-btn:hover {
  background: #e6e6e6;
}

.avatar,
.avatar-fallback {
  width: 28px;
  height: 28px;
  border-radius: 50%;
}

.avatar {
  object-fit: cover;
}

.avatar-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  background: #d6d6d6;
  color: #111;
}

.account-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 6px);
  width: 160px;
  background: #f3f3f3;
  border: 1px solid #ddd;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  z-index: 10;
  overflow: hidden;
  margin-top: -6px;

}

.account-menu button {
  border: none;
  background: #f3f3f3;
  color: #111;
  padding: 10px;
  text-align: left;
  cursor: pointer;
}

.account-menu button:hover {
  background: #e1e1e1;
}

.category-nav {
  display: flex;
  gap: 30px;
  flex-wrap: wrap;
  padding-bottom: 16px;
  border-bottom: 1px solid #ececec;
  margin-top: 30px;
}

.category-nav button {
  border: 1px solid #d6d6d6;
  background: #f0f0f0;
  color: #111;
  border-radius: 14px;
  padding: 4px 12px;
  cursor: pointer;
}

.category-nav button:hover {
  background: #e3e3e3;
}

.category-nav button.active {
  background: #dbdbdb;
  color: #111;
  border-color: #cfcfcf;
}

.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 30;
  backdrop-filter: blur(4px);
}

.modal-card {
  width: 380px;
  max-width: calc(100vw - 24px);
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  color: #111;
}

.auth-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.auth-tabs button {
  flex: 1;
  height: 34px;
  border: 1px solid #ddd;
  background: #f3f3f3;
  color: #111;
  border-radius: 8px;
}

.auth-tabs button.active {
  color: #111;
  background: #dcdcdc;
  border-color: #cfcfcf;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modal-form input,
.modal-form textarea,
.modal-form select {
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 8px;
  font: inherit;
  color: #111;
  background: #fff;
}

.modal-form button {
  height: 36px;
  border: 1px solid #d0d0d0;
  border-radius: 8px;
  background: #f0f0f0;
  color: #111;
  cursor: pointer;
}

.modal-form button:hover {
  background: #e2e2e2;
}

.modal-message {
  margin-top: 10px;
  color: #444;
}
</style>