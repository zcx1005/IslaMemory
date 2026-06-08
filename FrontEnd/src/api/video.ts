import request from './request'
import type { ApiResponse, CategoryItem, DanmakuItem, VideoDetail, VideoListData } from '@/types/video'

export function getVideoList(params: {
    page?: number
    page_size?: number
    category_slug?: string
    keyword?: string
    sort?: 'recommend' | 'latest' | 'popular'
}) {
    return request.get<any, ApiResponse<VideoListData>>('/api/v1/videos', { params })
}

export function getCategories() {
    return request.get<any, ApiResponse<CategoryItem[]>>('/api/v1/categories')
}

export function getVideoDetail(publicId: string) {
    return request.get<any, ApiResponse<VideoDetail>>(`/api/v1/videos/${publicId}`)
}

export function postVideoPlay(publicId: string, progressSeconds = 0) {
    return request.post<any, ApiResponse<{ counted: boolean }>>(`/api/v1/videos/${publicId}/play`, {
        progress_seconds: progressSeconds,
    })
}

export function listDanmaku(publicId: string) {
    return request.get<any, ApiResponse<{ list: DanmakuItem[] }>>(`/api/v1/videos/${publicId}/danmaku`)
}

export function createDanmaku(publicId: string, payload: { content: string; time_ms: number; color?: string; mode?: string }) {
    return request.post<any, ApiResponse<DanmakuItem>>(`/api/v1/videos/${publicId}/danmaku`, payload)
}

export function getDanmakuStreamUrl(publicId: string) {
    return `/api/v1/videos/${publicId}/danmaku/stream`
}

export function toPlayableUrl(url: string) {
    if (!url) return ''
    if (url.startsWith('http://') || url.startsWith('https://')) return url
    return url
}