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
    onProgress?: (progress: number) => void
}

const CHUNK_SIZE = 4 * 1024 * 1024

export function updateMyProfile(payload: UpdateProfilePayload) {
    const formData = new FormData()
    if (payload.username !== undefined) formData.append('username', payload.username)
    if (payload.avatar_file) formData.append('avatar', payload.avatar_file)
    else if (payload.avatar_url !== undefined) formData.append('avatar_url', payload.avatar_url)

    return request.put<any, ApiResponse<UserProfile>>('/api/v1/users/me', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    })
}

export function getMyFavoriteVideos(params?: { page?: number; page_size?: number }) {
    return request.get<any, ApiResponse<{ list: VideoListItem[]; total: number }>>('/api/v1/users/me/favorites', { params })
}

export function getMyUploadedVideos(params?: { page?: number; page_size?: number }) {
    return request.get<any, ApiResponse<{ list: VideoListItem[]; total: number }>>('/api/v1/users/me/uploads', { params })
}

export function getMyWatchHistory(params?: { page?: number; page_size?: number }) {
    return request.get<any, ApiResponse<{ list: VideoListItem[]; total: number }>>('/api/v1/users/me/history', { params })
}

export async function uploadVideo(payload: UploadVideoPayload) {
    const file = payload.file
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
    if (totalChunks <= 1) {
        const formData = new FormData()
        formData.append('title', payload.title.trim())
        formData.append('description', payload.description || '')
        formData.append('category_slug', payload.category_slug)
        formData.append('file', file)
        const res = await request.post<any, ApiResponse<{ public_id: string }>>('/api/v1/videos/upload', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
            onUploadProgress: (event) => {
                if (event.total) payload.onProgress?.(Math.round((event.loaded / event.total) * 100))
            },
        })
        payload.onProgress?.(100)
        return res
    }

    const init = await request.post<any, ApiResponse<{ upload_id: string; uploaded_chunks: number[]; chunk_size: number; total_chunks: number }>>('/api/v1/uploads/videos/init', {
        title: payload.title.trim(),
        description: payload.description || '',
        category_slug: payload.category_slug,
        filename: file.name,
        total_size: file.size,
        chunk_size: CHUNK_SIZE,
        total_chunks: totalChunks,
    })
    const uploadId = init.data.upload_id
    const uploaded = new Set(init.data.uploaded_chunks || [])

    for (let index = 0; index < totalChunks; index += 1) {
        if (uploaded.has(index)) continue
        const start = index * CHUNK_SIZE
        const chunk = file.slice(start, Math.min(file.size, start + CHUNK_SIZE))
        const formData = new FormData()
        formData.append('chunk', chunk, `${index}.part`)
        await request.post(`/api/v1/uploads/videos/${uploadId}/chunks/${index}`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
        })
        payload.onProgress?.(Math.round(((index + 1) / totalChunks) * 95))
    }

    const complete = await request.post<any, ApiResponse<{ public_id: string }>>(`/api/v1/uploads/videos/${uploadId}/complete`)
    payload.onProgress?.(100)
    return complete
}