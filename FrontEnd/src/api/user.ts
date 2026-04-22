import request from '@/api/request'
import type { ApiResponse, VideoListItem } from '@/types/video'
import type { UserProfile } from '@/api/auth'

export interface UpdateProfilePayload {
    username?: string
    avatar_url?: string
    avatar_file?: File
}

export interface UploadVideoPayload {
    title: string
    description?: string
    category_slug: string
    file: File
}

export function updateMyProfile(payload: UpdateProfilePayload) {
    const formData = new FormData()
    if (payload.username !== undefined) {
        formData.append('username', payload.username)
    }
    if (payload.avatar_file) {
        formData.append('avatar', payload.avatar_file)
    } else if (payload.avatar_url !== undefined) {
        formData.append('avatar_url', payload.avatar_url)
    }

    return request.put<any, ApiResponse<UserProfile>>('/api/v1/users/me', formData, {
        headers: {
            'Content-Type': 'multipart/form-data',
        },
    })
}

export function getMyFavoriteVideos(params?: { page?: number; page_size?: number }) {
    return request.get<any, ApiResponse<{ list: VideoListItem[]; total: number }>>('/api/v1/users/me/favorites', {
        params,
    })
}

export function getMyUploadedVideos(params?: { page?: number; page_size?: number }) {
    return request.get<any, ApiResponse<{ list: VideoListItem[]; total: number }>>('/api/v1/users/me/uploads', {
        params,
    })
}

export function uploadVideo(payload: UploadVideoPayload) {
    const formData = new FormData()
    formData.append('title', payload.title.trim())
    formData.append('description', payload.description || '')
    formData.append('category_slug', String(payload.category_slug))
    formData.append('file', payload.file)

    return request.post<any, ApiResponse<{ public_id: string }>>('/api/v1/videos/upload', formData, {
        headers: {
            'Content-Type': 'multipart/form-data',
        },
    })
}