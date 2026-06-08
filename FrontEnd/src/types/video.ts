export interface ApiResponse<T> {
    code: number
    msg: string
    data: T
}

export interface CategoryItem {
    id: number
    name: string
    slug: string
    sort_order: number
    status: number
}

export interface VideoListItem {
    public_id: string
    title: string
    description: string
    cover_url: string
    duration_seconds: number
    width: number
    height: number
    play_count: number
    like_count: number
    favorite_count: number
    comment_count: number
    category_id: number
    category_name: string
    category_slug: string
    published_at: string | null
    created_at: string
    username?: string
    uploader_username?: string
}

export interface VideoListData {
    list: VideoListItem[]
    total: number
    page: number
    page_size: number
}

export interface VideoDetail extends VideoListItem {
    playback_type: number
    playback_url: string
    transcode_status: number
    transcode_progress: number
}

export interface DanmakuItem {
    id: number
    public_id: string
    user_id: number
    content: string
    time_ms: number
    color: string
    mode: string
    created_at: string
}