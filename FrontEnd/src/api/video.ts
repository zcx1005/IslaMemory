import request from './request'
import type { ApiResponse, VideoDetail, VideoListData } from '@/types/video'

export function getVideoList(params: {
    page?: number
    page_size?: number
    category_slug?: string
    keyword?: string
    sort?: 'latest' | 'popular'
}) {
    return request.get<any, ApiResponse<VideoListData>>('/api/v1/videos', { params })
}

export function getVideoDetail(publicId: string) {
    return request.get<any, ApiResponse<VideoDetail>>(`/api/v1/videos/${publicId}`)
}

export function postVideoPlay(publicId: string) {
    return request.post<any, ApiResponse<null>>(`/api/v1/videos/${publicId}/play`)
}

// 把 /static/... 转成可访问地址（开发时走当前域名+代理）
export function toPlayableUrl(url: string) {
    if (!url) return ''
    if (url.startsWith('http://') || url.startsWith('https://')) return url
    return url // 相对路径，直接给 video 标签
}