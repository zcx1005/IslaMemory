import { ref } from 'vue'
import {
    likeVideo,
    unlikeVideo,
    favoriteVideo,
    unfavoriteVideo,
    listComments,
    createComment
} from '@/api/videoInteraction'
import type { CommentItem } from '@/types/video-interaction'

export function useVideoInteraction(publicId: string) {
    const liked = ref(false)
    const favorited = ref(false)

    const likeLoading = ref(false)
    const favoriteLoading = ref(false)

    const comments = ref<CommentItem[]>([])
    const commentsLoading = ref(false)
    const commentSubmitting = ref(false)

    const interactionError = ref('')

    function ensurePublicId() {
        return !!publicId && publicId.trim().length > 0
    }

    // 给详情页调用：如果后端返回了 is_liked/is_favorited，可以初始化
    function setInitialState(payload: { liked?: boolean; favorited?: boolean }) {
        if (typeof payload.liked === 'boolean') liked.value = payload.liked
        if (typeof payload.favorited === 'boolean') favorited.value = payload.favorited
    }

    async function toggleLike() {
        if (!ensurePublicId() || likeLoading.value) return
        likeLoading.value = true
        interactionError.value = ''

        const prev = liked.value
        try {
            if (prev) {
                const res = await unlikeVideo(publicId)
                if (res.code !== 200) throw new Error(res.msg || '取消点赞失败')
                liked.value = false
            } else {
                const res = await likeVideo(publicId)
                if (res.code !== 200) throw new Error(res.msg || '点赞失败')
                liked.value = true
            }
        } catch (e: any) {
            liked.value = prev // 回滚
            interactionError.value = e?.message || '点赞操作失败'
            throw e
        } finally {
            likeLoading.value = false
        }
    }

    async function toggleFavorite() {
        if (!ensurePublicId() || favoriteLoading.value) return
        favoriteLoading.value = true
        interactionError.value = ''

        const prev = favorited.value
        try {
            if (prev) {
                const res = await unfavoriteVideo(publicId)
                if (res.code !== 200) throw new Error(res.msg || '取消收藏失败')
                favorited.value = false
            } else {
                const res = await favoriteVideo(publicId)
                if (res.code !== 200) throw new Error(res.msg || '收藏失败')
                favorited.value = true
            }
        } catch (e: any) {
            favorited.value = prev // 回滚
            interactionError.value = e?.message || '收藏操作失败'
            throw e
        } finally {
            favoriteLoading.value = false
        }
    }

    async function fetchComments() {
        if (!ensurePublicId()) return
        commentsLoading.value = true
        interactionError.value = ''
        try {
            const res = await listComments(publicId)
            if (res.code === 200) {
                comments.value = res.data?.list || []
            } else {
                comments.value = []
                throw new Error(res.msg || '获取评论失败')
            }
        } catch (e: any) {
            interactionError.value = e?.message || '获取评论失败'
            throw e
        } finally {
            commentsLoading.value = false
        }
    }

    async function submitComment(content: string) {
        const v = content.trim()
        if (!ensurePublicId() || !v) return
        commentSubmitting.value = true
        interactionError.value = ''
        try {
            const res = await createComment(publicId, { content: v })
            if (res.code !== 200) throw new Error(res.msg || '评论失败')
            await fetchComments()
        } catch (e: any) {
            interactionError.value = e?.message || '评论失败'
            throw e
        } finally {
            commentSubmitting.value = false
        }
    }

    async function submitReply(target: CommentItem, content: string) {
        const v = content.trim()
        if (!ensurePublicId() || !v) return
        commentSubmitting.value = true
        interactionError.value = ''
        try {
            const res = await createComment(publicId, {
                content: v,
                parent_id: target.id,
                reply_to_user_id: target.user_id
            })
            if (res.code !== 200) throw new Error(res.msg || '回复失败')
            await fetchComments()
        } catch (e: any) {
            interactionError.value = e?.message || '回复失败'
            throw e
        } finally {
            commentSubmitting.value = false
        }
    }

    return {
        liked,
        favorited,
        likeLoading,
        favoriteLoading,
        comments,
        commentsLoading,
        commentSubmitting,
        interactionError,
        setInitialState,
        toggleLike,
        toggleFavorite,
        fetchComments,
        submitComment,
        submitReply
    }
}