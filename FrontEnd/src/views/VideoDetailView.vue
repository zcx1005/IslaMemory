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

      <!-- 点赞/收藏按钮 -->
      <div class="actions">
        <button :disabled="likeLoading" @click="onToggleLike">
          {{ liked ? '已点赞（点击取消）' : '点赞' }}
        </button>
        <button :disabled="favoriteLoading" @click="onToggleFavorite">
          {{ favorited ? '已收藏（点击取消）' : '收藏' }}
        </button>
      </div>

      <!-- 评论输入 -->
      <div class="comment-editor">
        <textarea
            v-model="commentText"
            placeholder="写下你的评论..."
            rows="3"
        />
        <button :disabled="commentSubmitting" @click="onSubmitComment">
          发表评论
        </button>
      </div>

      <!-- 评论列表 -->
      <div class="comment-list">
        <h3>评论区</h3>
        <div v-if="commentsLoading">评论加载中...</div>
        <div v-else-if="comments.length === 0">暂无评论</div>

        <div v-for="c in comments" :key="c.id" class="comment-item">
          <div class="comment-header">
            <strong>{{ c.username || `用户${c.user_id}` }}</strong>
            <span class="time">{{ c.created_at }}</span>
          </div>
          <div class="comment-content">{{ c.content }}</div>

          <!-- 回复输入 -->
          <div class="reply-editor">
            <input
                v-model="replyTextMap[c.id]"
                placeholder="回复这条评论..."
            />
            <button :disabled="commentSubmitting" @click="onSubmitReply(c)">
              回复
            </button>
          </div>

          <!-- 回复列表：默认仅显示第一条 -->
          <div v-if="(c.replies?.length || 0) > 0" class="reply-list">
            <div
                v-for="r in visibleReplies(c)"
                :key="r.id"
                class="reply-item"
            >
              <strong>{{ r.username || `用户${r.user_id}` }}</strong>
              <span v-if="r.reply_to_username"> 回复 {{ r.reply_to_username }}</span>
              ：{{ r.content }}
            </div>

            <button
                v-if="(c.replies?.length || 0) > 1"
                class="expand-btn"
                @click="toggleExpand(c.id)"
            >
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
import {

  likeVideo,
  unlikeVideo,
  favoriteVideo,
  unfavoriteVideo,
  listComments,
  createComment
} from '@/api/videoInteraction'
import {
  getVideoDetail,
  postVideoPlay,
  toPlayableUrl,
} from '@/api/video'
import type { VideoDetail } from '@/types/video'

interface CommentItem {
  id: number
  user_id: number
  username?: string
  content: string
  created_at: string
  reply_to_username?: string
  replies?: CommentItem[]
}

const route = useRoute()
const loading = ref(true)
const error = ref('')
const video = ref<VideoDetail | null>(null)
const played = ref(false)

const playUrl = computed(() => toPlayableUrl(video.value?.playback_url || ''))

// 点赞/收藏状态
const liked = ref(false)
const favorited = ref(false)
const likeLoading = ref(false)
const favoriteLoading = ref(false)

// 评论状态
const comments = ref<CommentItem[]>([])
const commentsLoading = ref(false)
const commentSubmitting = ref(false)
const commentText = ref('')
const replyTextMap = reactive<Record<number, string>>({})
const expandedMap = reactive<Record<number, boolean>>({})

onMounted(async () => {
  try {
    const publicId = String(route.params.publicId || '')
    const res = await getVideoDetail(publicId)
    if (res.code !== 200) throw new Error(res.msg || '获取详情失败')
    video.value = res.data

    // 拉评论
    await fetchComments(publicId)

    // 如果后端后续返回 is_liked / is_favorited，可在这里赋值
    // liked.value = !!res.data.is_liked
    // favorited.value = !!res.data.is_favorited
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

async function onToggleLike() {
  if (!video.value || likeLoading.value) return
  likeLoading.value = true
  try {
    if (liked.value) {
      await unlikeVideo(video.value.public_id)
      liked.value = false
    } else {
      await likeVideo(video.value.public_id)
      liked.value = true
    }
  } catch {
    // 可接入 message 提示
  } finally {
    likeLoading.value = false
  }
}

async function onToggleFavorite() {
  if (!video.value || favoriteLoading.value) return
  favoriteLoading.value = true
  try {
    if (favorited.value) {
      await unfavoriteVideo(video.value.public_id)
      favorited.value = false
    } else {
      await favoriteVideo(video.value.public_id)
      favorited.value = true
    }
  } catch {
    // 可接入 message 提示
  } finally {
    favoriteLoading.value = false
  }
}

async function fetchComments(publicId: string) {
  commentsLoading.value = true
  try {
    const res = await listComments(publicId)
    if (res.code === 200) {
      comments.value = res.data?.list || []
    } else {
      comments.value = []
    }
  } finally {
    commentsLoading.value = false
  }
}

async function onSubmitComment() {
  if (!video.value) return
  const content = commentText.value.trim()
  if (!content) return

  commentSubmitting.value = true
  try {
    const res = await createComment(video.value.public_id, { content })
    if (res.code === 200) {
      commentText.value = ''
      await fetchComments(video.value.public_id)
    }
  } finally {
    commentSubmitting.value = false
  }
}

async function onSubmitReply(comment: CommentItem) {
  if (!video.value) return
  const content = (replyTextMap[comment.id] || '').trim()
  if (!content) return

  commentSubmitting.value = true
  try {
    const res = await createComment(video.value.public_id, {
      content,
      parent_id: comment.id,
      reply_to_user_id: comment.user_id
    })
    if (res.code === 200) {
      replyTextMap[comment.id] = ''
      await fetchComments(video.value.public_id)
      expandedMap[comment.id] = true // 回复后自动展开
    }
  } finally {
    commentSubmitting.value = false
  }
}

// 默认只展示第一条回复，展开后展示全部
function visibleReplies(comment: CommentItem) {
  const arr = comment.replies || []
  if (expandedMap[comment.id]) return arr
  return arr.slice(0, 1)
}

function toggleExpand(commentId: number) {
  expandedMap[commentId] = !expandedMap[commentId]
}
</script>

<style scoped>
.page { padding: 16px; }
.player { width: 720px; max-width: 100%; background: #000; }
.meta { margin-top: 12px; }

.actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}

.comment-editor {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.comment-list {
  margin-top: 20px;
}

.comment-item {
  border: 1px solid #eee;
  border-radius: 8px;
  padding: 12px;
  margin-top: 10px;
}

.comment-header {
  display: flex;
  justify-content: space-between;
}

.time {
  color: #888;
  font-size: 12px;
}

.comment-content {
  margin-top: 6px;
  line-height: 1.6;
}

.reply-editor {
  margin-top: 8px;
  display: flex;
  gap: 8px;
}

.reply-editor input {
  flex: 1;
}

.reply-list {
  margin-top: 8px;
  padding-left: 12px;
  border-left: 2px solid #f0f0f0;
}

.reply-item {
  margin-top: 6px;
}

.expand-btn {
  margin-top: 8px;
  color: #409eff;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
</style>