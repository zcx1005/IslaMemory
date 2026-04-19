export interface ApiResponse<T> {
    code: number
    msg: string
    data: T
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
}

export interface VideoListData {
    list: VideoListItem[]
    total: number
    page: number
    page_size: number
}

export interface VideoDetail {
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
    playback_type: number
    playback_url: string
    published_at: string | null
    created_at: string
}