export interface ApiResp<T> {
    code: number
    msg: string
    data: T
}

export interface CommentItem {
    id: number
    video_id: number
    user_id: number
    username?: string
    avatar_url?: string
    parent_id?: number | null
    root_id?: number | null
    reply_to_user_id?: number | null
    reply_to_username?: string | null
    content: string
    like_count: number
    created_at: string
    updated_at: string
    replies?: CommentItem[]
}

export interface CommentListResp {
    list: CommentItem[]
}