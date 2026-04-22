<template>
  <div class="page">
    <div v-if="loading">加载中...</div>
    <div v-else-if="error">错误：{{ error }}</div>

    <div v-else-if="video">
      <h2>{{ video.title }}</h2>
      <p class="meta-line">播放量 {{ formatPlayCount(video.play_count) }}&nbsp;&nbsp;&nbsp;{{ formatDateTime(video.created_at) }}</p>

      <video class="player" controls :src="playUrl" @play="onFirstPlay" />

      <p class="desc">{{ video.description || '暂无简介' }}</p>

      <div class="actions">
        <button :disabled="likeLoading" @click="onToggleLike">👍 {{ video.like_count }}</button>
        <button :disabled="favoriteLoading" @click="onToggleFavorite">⭐ {{ video.favorite_count }}</button>
      </div>

      <p v-if="actionError" class="action-error">{{ actionError }}</p>

      <div class="comment-editor">
        <textarea v-model="commentText" placeholder="写下你的评论..." rows="3" />
        <button :disabled="commentSubmitting" @click="onSubmitComment">发表评论</button>
      </div>

      <div class="comment-list">
        <h3>评论区（{{ video.comment_count }}）</h3>
        <div v-if="commentsLoading">评论加载中...</div>
        <div v-else-if="comments.length === 0">暂无评论</div>

        <div v-for="c in comments" :key="c.id" class="comment-item">
          <div class="comment-header">
            <strong>{{ c.username || `用户${c.user_id}` }}</strong>
            <span class="time">{{ c.created_at }}</span>
          </div>
          <div class="comment-content">{{ c.content }}</div>

          <div class="reply-editor">
            <input v-model="replyTextMap[c.id]" placeholder="回复这条评论..." />
            <button :disabled="commentSubmitting" @click="onSubmitReply(c)">回复</button>
          </div>

          <div v-if="(c.replies?.length || 0) > 0" class="reply-list">
            <div v-for="r in visibleReplies(c)" :key="r.id" class="reply-item">
              <strong>{{ r.username || `用户${r.user_id}` }}</strong>
              <span v-if="r.reply_to_username"> 回复 {{ r.reply_to_username }}</span>
              ：{{ r.content }}
            </div>

            <button v-if="(c.replies?.length || 0) > 1" class="expand-btn" @click="toggleExpand(c.id)">
              {{ expandedMap[c.id] ? '收起回复' : `展开更多回复（${(c.replies?.length || 0) - 1}）` }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { likeVideo, unlikeVideo, favoriteVideo, unfavoriteVideo, listComments, createComment } from '@/api/videoInteraction'
import { getVideoDetail, postVideoPlay, toPlayableUrl } from '@/api/video'
import { useAuth } from '@/composables/userAuth'
import type { VideoDetail } from '@/types/video'
import type { CommentItem } from '@/types/video-interaction'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const actionError = ref('')
const video = ref<VideoDetail | null>(null)
const played = ref(false)

const playUrl = computed(() => toPlayableUrl(video.value?.playback_url || ''))

const liked = ref(false)
const favorited = ref(false)
const likeLoading = ref(false)
const favoriteLoading = ref(false)

const comments = ref<CommentItem[]>([])
const commentsLoading = ref(false)
const commentSubmitting = ref(false)
const commentText = ref('')
const replyTextMap = reactive<Record<number, string>>({})
const expandedMap = reactive<Record<number, boolean>>({})

const { isLoggedIn, restoreSession, openAuthModal } = useAuth()

onMounted(async () => {
  await restoreSession()
  try {
    const publicId = String(route.params.publicId || '')
    const res = await getVideoDetail(publicId)
    if (res.code !== 200) throw new Error(res.msg || '获取详情失败')
    video.value = res.data
    await fetchComments(publicId)
  } catch (e: any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
})

function ensureLogin() {
  if (isLoggedIn.value) return true
  actionError.value = '请先登录后再操作'
  openAuthModal('login')
  return false
}

async function onFirstPlay() {
  if (played.value || !video.value) return
  played.value = true
  try {
    await postVideoPlay(video.value.public_id)
  } catch {
    // ignore
  }
}

async function onToggleLike() {
  if (!video.value || likeLoading.value || !ensureLogin()) return
  likeLoading.value = true
  actionError.value = ''
  try {
    if (liked.value) {
      await unlikeVideo(video.value.public_id)
      liked.value = false
      video.value.like_count = Math.max(0, video.value.like_count - 1)
    } else {
      await likeVideo(video.value.public_id)
      liked.value = true
      video.value.like_count += 1
    }
  } catch (e: any) {
    actionError.value = e?.message || '点赞操作失败'
  } finally {
    likeLoading.value = false
  }
}

async function onToggleFavorite() {
  if (!video.value || favoriteLoading.value || !ensureLogin()) return
  favoriteLoading.value = true
  actionError.value = ''
  try {
    if (favorited.value) {
      await unfavoriteVideo(video.value.public_id)
      favorited.value = false
      video.value.favorite_count = Math.max(0, video.value.favorite_count - 1)
    } else {
      await favoriteVideo(video.value.public_id)
      favorited.value = true
      video.value.favorite_count += 1
    }
  } catch (e: any) {
    actionError.value = e?.message || '收藏操作失败'
  } finally {
    favoriteLoading.value = false
  }
}

async function fetchComments(publicId: string) {
  commentsLoading.value = true
  try {
    const res = await listComments(publicId)
    comments.value = res.code === 200 ? res.data?.list || [] : []
  } finally {
    commentsLoading.value = false
  }
}

async function onSubmitComment() {
  if (!video.value || !ensureLogin()) return
  const content = commentText.value.trim()
  if (!content) return

  commentSubmitting.value = true
  actionError.value = ''
  try {
    const res = await createComment(video.value.public_id, { content })
    if (res.code === 200) {
      commentText.value = ''
      video.value.comment_count += 1
      await fetchComments(video.value.public_id)
    }
  } catch (e: any) {
    actionError.value = e?.message || '发表评论失败'
  } finally {
    commentSubmitting.value = false
  }
}

async function onSubmitReply(comment: CommentItem) {
  if (!video.value || !ensureLogin()) return
  const content = (replyTextMap[comment.id] || '').trim()
  if (!content) return

  commentSubmitting.value = true
  actionError.value = ''
  try {
    const res = await createComment(video.value.public_id, {
      content,
      parent_id: comment.id,
      reply_to_user_id: comment.user_id,
    })
    if (res.code === 200) {
      replyTextMap[comment.id] = ''
      video.value.comment_count += 1
      await fetchComments(video.value.public_id)
      expandedMap[comment.id] = true
    }
  } catch (e: any) {
    actionError.value = e?.message || '回复失败'
  } finally {
    commentSubmitting.value = false
  }
}

function visibleReplies(comment: CommentItem) {
  const arr = comment.replies || []
  return expandedMap[comment.id] ? arr : arr.slice(0, 1)
}

function toggleExpand(commentId: number) {
  expandedMap[commentId] = !expandedMap[commentId]
}

function formatPlayCount(count: number) {
  return count >= 10000 ? `${(count / 10000).toFixed(1)}万` : String(count)
}

function formatDateTime(date: string) {
  return date ? date.slice(0, 19).replace('T', ' ') : '-'
}
</script>

<style scoped>
.page { padding-top: 16px; text-align: left; }
.meta-line { color: #666; margin-bottom: 12px; }
.player { width: 100%; max-width: 1200px; background: #000; border-radius: 8px; }
.desc { margin-top: 12px; white-space: pre-wrap; }
.actions { margin-top: 12px; display: flex; gap: 8px; }
.actions button { border: 1px solid #ddd; background: #fff; border-radius: 8px; padding: 6px 12px; cursor: pointer; }
.action-error { color: #e74c3c; margin-top: 10px; }
.comment-editor { margin-top: 16px; display: flex; flex-direction: column; gap: 8px; }
.comment-editor textarea { border: 1px solid #ddd; border-radius: 8px; padding: 8px; }
.comment-editor button { width: 120px; height: 34px; border: 1px solid #409eff; border-radius: 8px; background: #409eff; color: #fff; }
.comment-list { margin-top: 20px; }
.comment-item { border: 1px solid #eee; border-radius: 8px; padding: 12px; margin-top: 10px; }
.comment-header { display: flex; justify-content: space-between; }
.time { color: #888; font-size: 12px; }
.comment-content { margin-top: 6px; line-height: 1.6; }
.reply-editor { margin-top: 8px; display: flex; gap: 8px; }
.reply-editor input { flex: 1; border: 1px solid #ddd; border-radius: 8px; padding: 6px 8px; }
.reply-editor button { border: 1px solid #ddd; background: #fff; border-radius: 8px; padding: 0 12px; }
.reply-list { margin-top: 8px; padding-left: 12px; border-left: 2px solid #f0f0f0; }
.reply-item { margin-top: 6px; }
.expand-btn { margin-top: 8px; color: #409eff; background: none; border: none; cursor: pointer; padding: 0; }
</style>