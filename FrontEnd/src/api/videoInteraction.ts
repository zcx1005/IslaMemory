import request from '@/api/request'
import type { ApiResp, CommentItem, CommentListResp, InteractionStateResp } from '@/types/video-interaction'

function assertPublicId(publicId: string) {
    if (!publicId || !publicId.trim()) {
        throw new Error('publicId 不能为空')
    }
}

export function getInteractionState(publicId: string) {
    assertPublicId(publicId)
    return request.get<any, ApiResp<InteractionStateResp>>(
        `/api/v1/videos/${publicId}/interaction`
    )
}

export function likeVideo(publicId: string) {
    assertPublicId(publicId)
    return request.post<any, ApiResp<{ liked: boolean }>>(
        `/api/v1/videos/${publicId}/like`
    )
}

export function unlikeVideo(publicId: string) {
    assertPublicId(publicId)
    return request.delete<any, ApiResp<{ unliked: boolean }>>(
        `/api/v1/videos/${publicId}/like`
    )
}

export function favoriteVideo(publicId: string) {
    assertPublicId(publicId)
    return request.post<any, ApiResp<{ favorited: boolean }>>(
        `/api/v1/videos/${publicId}/favorite`
    )
}

export function unfavoriteVideo(publicId: string) {
    assertPublicId(publicId)
    return request.delete<any, ApiResp<{ unfavorited: boolean }>>(
        `/api/v1/videos/${publicId}/favorite`
    )
}

export function listComments(publicId: string) {
    assertPublicId(publicId)
    return request.get<any, ApiResp<CommentListResp>>(
        `/api/v1/videos/${publicId}/comments`
    )
}

export function createComment(
    publicId: string,
    payload: {
        content: string
        parent_id?: number
        reply_to_user_id?: number
    }
) {
    assertPublicId(publicId)
    return request.post<any, ApiResp<CommentItem>>(
        `/api/v1/videos/${publicId}/comments`,
        payload
    )
}