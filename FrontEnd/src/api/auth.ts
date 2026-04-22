import request from '@/api/request'
import type { ApiResponse } from '@/types/video'

export interface RegisterPayload {
    account: string
    username: string
    password: string
}

export interface LoginPayload {
    account: string
    password: string
}

export interface AuthUser {
    id: number
    account: string
    username: string
    avatar_url?: string
    role?: string
}

export interface LoginResponse {
    token: string
    user: AuthUser
}

export interface UserProfile extends AuthUser {
    status?: string
    can_upload?: boolean
    password_changed_at?: string
}

export function registerAccount(payload: RegisterPayload) {
    return request.post<any, ApiResponse<{ id: number; account: string; username: string }>>(
        '/api/v1/auth/register',
        payload
    )
}

export function loginAccount(payload: LoginPayload) {
    return request.post<any, ApiResponse<LoginResponse>>('/api/v1/auth/login', payload)
}

export function getMyProfile() {
    return request.get<any, ApiResponse<UserProfile>>('/api/v1/users/me')
}